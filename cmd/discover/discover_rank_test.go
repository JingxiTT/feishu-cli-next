// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func rankingTree() *cobra.Command {
	root := &cobra.Command{Use: "test-cli"}
	root.PersistentFlags().Bool("save-result", false, "save a complete read result as a versioned artifact")
	add := func(parent *cobra.Command, use, short string) *cobra.Command {
		c := &cobra.Command{Use: use, Short: short}
		parent.AddCommand(c)
		return c
	}
	sheets := add(root, "sheets", "Spreadsheet operations")
	add(sheets, "+cells-get", "Read cell ranges with values")
	add(sheets, "+sheet-create", "Create a sub-sheet")
	add(sheets, "+dim-hide", "Hide rows or columns")
	docs := add(root, "docs", "Documents")
	add(docs, "+search", "Search docs and spreadsheet files")
	add(docs, "+read", "Read a document")
	auth := add(root, "auth", "Authentication")
	login := add(auth, "login", "Device Flow authorization login")
	login.Flags().String("sheet-scope", "", "spreadsheet access scope")
	apps := add(root, "apps", "App operations")
	add(apps, "+list", "List apps")
	add(apps, "+analytics-list", "List analytics reports")
	add(apps, "+automation-list", "List automations")
	contact := add(root, "contact", "Contacts")
	add(contact, "+search-user", "Search users")
	getUser := add(contact, "+get-user", "Get a user")
	getUser.Flags().String("user-id-type", "", "user_id, open_id or union_id")
	approval := add(root, "approval", "Approvals")
	search := add(approval, "approvals", "Search approvals")
	search.Flags().String("user-id", "", "approver user")
	return root
}

func rankedNames(entries []entry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Command)
	}
	return names
}

func assertTop(t *testing.T, query string, entries []entry, want ...string) {
	t.Helper()
	got := rankedNames(entries)
	if len(got) > len(want) {
		got = got[:len(want)]
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("query %q: top results %v, want %v", query, got, want)
	}
}

func TestRankPutsCommandMatchingEveryTermFirst(t *testing.T) {
	root := rankingTree()
	assertTop(t, "read sheet", collect(root, "read sheet"), "sheets +cells-get", "docs +search", "sheets")
}

func TestRankPrefersExactSegmentOverLongerSegmentContainingTerm(t *testing.T) {
	root := rankingTree()
	assertTop(t, "apps list", collect(root, "apps list"), "apps +list", "apps +analytics-list", "apps +automation-list")
}

func TestRankPrefersPathMatchOverFlagOnlyMatch(t *testing.T) {
	root := rankingTree()
	assertTop(t, "search user", collect(root, "search user"), "contact +search-user", "approval approvals")
}

func TestRankDemotesGlobalFlagOnlyMatchesBelowPathMatches(t *testing.T) {
	root := rankingTree()
	entries := collect(root, "read")
	position := map[string]int{}
	for i, e := range entries {
		position[e.Command] = i
	}
	for _, inherited := range []string{"auth login", "sheets +sheet-create"} {
		if position[inherited] <= position["docs +read"] || position[inherited] <= position["sheets +cells-get"] {
			t.Fatalf("inherited-flag match %q ranked above a path match: %v", inherited, rankedNames(entries))
		}
	}
	assertTop(t, "read", entries, "docs +read", "sheets +cells-get")
}

func TestRankKeepsUnionIDFindingGetUser(t *testing.T) {
	root := rankingTree()
	assertTop(t, "contact union_id", collect(root, "contact union_id"), "contact +get-user")
}

func TestDiscoveryHintsOnlyWhenNothingMatches(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := rankingTree()
	root.AddCommand(NewCmd(f, root))
	root.SetArgs([]string{"discover", "no-such-thing-here"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var empty struct {
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &empty); err != nil {
		t.Fatal(err)
	}
	if hint, _ := empty.Meta["hint"].(string); hint == "" {
		t.Fatalf("missing hint for empty result: %s", out)
	}

	f, out, _, _ = cmdutil.TestFactory(t, nil)
	root = rankingTree()
	root.AddCommand(NewCmd(f, root))
	root.SetArgs([]string{"discover", "apps", "list"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var matched struct {
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &matched); err != nil {
		t.Fatal(err)
	}
	if _, ok := matched.Meta["hint"]; ok {
		t.Fatalf("hint added to a matching query: %s", out)
	}
}

func TestDiscoveryListingsWithoutTermsStayAlphabetical(t *testing.T) {
	root := &cobra.Command{Use: "cli"}
	for _, name := range []string{"calendar", "vc", "docs"} {
		group := &cobra.Command{Use: name, Short: name}
		group.AddCommand(&cobra.Command{Use: "+list", Short: "list", Run: func(*cobra.Command, []string) {}})
		root.AddCommand(group)
	}
	for _, query := range []string{"", "*"} {
		var got []string
		for _, e := range collect(root, query) {
			got = append(got, e.Command)
		}
		if !sort.StringsAreSorted(got) {
			t.Errorf("query %q: listing not alphabetical: %v", query, got)
		}
	}
}
