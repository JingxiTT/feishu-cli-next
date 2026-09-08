// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TestDiscoveryReflectsTreeAndRetainsContinuation(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	docs := &cobra.Command{Use: "docs", Short: "Documents"}
	read := &cobra.Command{Use: "read", Short: "Read Unicode 文档"}
	read.Flags().String("document-id", "", "native identity")
	_ = read.MarkFlagRequired("document-id")
	docs.AddCommand(read, &cobra.Command{Use: "write", Short: "Write document"})
	hidden := &cobra.Command{Use: "private", Hidden: true}
	hidden.AddCommand(&cobra.Command{Use: "read", Short: "Read private document"})
	root.AddCommand(docs, hidden, NewCmd(f, root))
	root.SetArgs([]string{"discover", "docs", "--limit", "1", "--offset", "1", "--flags"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
		Meta struct {
			Query    string `json:"query"`
			Complete bool   `json:"complete"`
			Next     int    `json:"next_offset"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Command != "docs read" || result.Meta.Complete || result.Meta.Next != 2 || result.Meta.Query != "docs" {
		t.Fatalf("unexpected result: %s", out)
	}
	if len(result.Data[0].Flags) != 1 || !result.Data[0].Flags[0].Required {
		t.Fatalf("missing required flag: %+v", result.Data[0])
	}
	for _, e := range collect(root, "*", false) {
		if e.Command == "private" || e.Command == "private read" {
			t.Fatal("hidden subtree leaked")
		}
	}
	groups := collect(root, "", false)
	foundDocs := false
	for _, group := range groups {
		if strings.Contains(group.Command, " ") {
			t.Fatalf("nested command in group listing: %+v", group)
		}
		if group.Command == "docs" {
			foundDocs = true
		}
	}
	if !foundDocs {
		t.Fatalf("missing docs group: %+v", groups)
	}
}

func TestDiscoveryValidatesBoundsAndEmptyResults(t *testing.T) {
	for _, args := range [][]string{{"--limit", "0"}, {"--limit", "1001"}, {"--offset", "-1"}} {
		f, _, _, _ := cmdutil.TestFactory(t, nil)
		root := &cobra.Command{Use: "test-cli"}
		c := NewCmd(f, root)
		root.AddCommand(c)
		root.SetArgs(append([]string{"discover"}, args...))
		if err := root.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	root.AddCommand(NewCmd(f, root))
	root.SetArgs([]string{"discover", "does-not-exist"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if data, ok := result["data"].([]any); !ok || len(data) != 0 {
		t.Fatalf("want empty array: %s", out)
	}
}

func TestDiscoveryMatchesPublicFlagsWithoutChangingOutputShape(t *testing.T) {
	root := &cobra.Command{Use: "test-cli"}
	root.PersistentFlags().String("identity-type", "", "user_id or open_id identity namespace")
	contact := &cobra.Command{Use: "contact", Short: "Contacts"}
	lookup := &cobra.Command{Use: "lookup", Short: "Read a person"}
	lookup.Flags().String("user-ids", "", "CSV identifiers including union_id")
	lookup.Flags().String("private-selector", "", "hidden-canary")
	_ = lookup.Flags().MarkHidden("private-selector")
	lookup.Flags().String("known-default", "default-canary", "ordinary option")
	contact.AddCommand(lookup)
	root.AddCommand(contact)
	for _, query := range []string{"contact USER-IDS", "contact UNION_ID", "contact lookup OPEN_ID"} {
		entries := collect(root, query, false)
		if len(entries) != 1 || entries[0].Command != "contact lookup" {
			t.Fatalf("query %q: %+v", query, entries)
		}
		if len(entries[0].Flags) != 0 {
			t.Fatalf("matching added unsolicited flag output: %+v", entries[0])
		}
		encoded, err := json.Marshal(entries[0])
		if err != nil {
			t.Fatal(err)
		}
		var shape map[string]any
		if err := json.Unmarshal(encoded, &shape); err != nil {
			t.Fatal(err)
		}
		if len(shape) != 2 || shape["command"] == nil || shape["description"] == nil {
			t.Fatalf("output shape changed: %s", encoded)
		}
	}
	for _, query := range []string{"hidden-canary", "private-selector", "default-canary", "contact union_id missingword"} {
		if found := collect(root, query, true); len(found) != 0 {
			t.Fatalf("query %q leaked hidden/default text or ignored a term: %+v", query, found)
		}
	}
	entries := collect(root, "contact user-ids", true)
	for _, flag := range entries[0].Flags {
		if flag.Name == "private-selector" {
			t.Fatal("hidden flag leaked into output")
		}
	}
}

func TestDiscoveryFlagDescriptionsPreserveRegisteredUsageAndVisibility(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	root.PersistentFlags().String("identity-type", "open_id", "open_id, user_id or union_id")
	root.PersistentFlags().String("selector", "", "inherited selector must not leak")
	lookup := &cobra.Command{Use: "lookup", Short: "Read a person"}
	usage := "Native 用户 identity.\nPreserve leading zeroes."
	lookup.Flags().String("selector", "", usage)
	_ = lookup.MarkFlagRequired("selector")
	lookup.Flags().Bool("private", false, "hidden description must not leak")
	_ = lookup.Flags().MarkHidden("private")
	root.AddCommand(lookup, NewCmd(f, root))
	root.SetArgs([]string{"discover", "lookup", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("unexpected result: %s", out)
	}
	flags := result.Data[0].Flags
	if len(flags) != 2 || flags[0].Name != "identity-type" || flags[0].Description != "open_id, user_id or union_id" || flags[0].Default != "open_id" {
		t.Fatalf("missing inherited flag guidance or sorted output: %+v", flags)
	}
	if flags[1].Name != "selector" || flags[1].Description != usage || flags[1].Type != "string" || !flags[1].Required {
		t.Fatalf("local flag guidance was lost or replaced: %+v", flags[1])
	}
}
