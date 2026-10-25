// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"encoding/json"
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/larksuite/cli/internal/affordance"
	"github.com/larksuite/cli/internal/apicatalog"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

type countedValue struct{ descriptions *int }

func (v countedValue) String() string   { return "" }
func (v countedValue) Set(string) error { return nil }
func (v countedValue) Type() string {
	*v.descriptions++
	return "string"
}

func TestDiscoveryDescribesOnlyReturnedPage(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	counts := make([]int, 3)
	for i, name := range []string{"alpha", "beta", "gamma"} {
		cmd := &cobra.Command{Use: name, Short: "Read item"}
		cmd.Flags().Var(countedValue{&counts[i]}, "item-id", "native identifier")
		root.AddCommand(cmd)
	}
	root.AddCommand(NewCmd(f, root))
	root.SetArgs([]string{"discover", "item", "--flags", "--offset", "1", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if counts[0] != 1 || counts[1] != 0 || counts[2] != 0 {
		t.Fatalf("rendered flags outside returned page: %v", counts)
	}
	var result struct {
		Data []entry `json:"data"`
		Meta struct {
			Total int `json:"total"`
			Next  int `json:"next_offset"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Command != "alpha" || len(result.Data[0].Flags) != 1 || result.Meta.Total != 3 || result.Meta.Next != 2 {
		t.Fatalf("pagination or flag contract changed: %s", out)
	}
}

func TestDiscoveryPathsMatchCobraInNestedTree(t *testing.T) {
	outer := &cobra.Command{Use: "wrapper"}
	root := &cobra.Command{Use: "test-cli"}
	outer.AddCommand(root)
	group := &cobra.Command{Use: "group"}
	nested := &cobra.Command{Use: "nested"}
	leaf := &cobra.Command{Use: "read [id]"}
	root.AddCommand(group)
	group.AddCommand(nested)
	nested.AddCommand(leaf)
	for _, e := range collect(root, "*") {
		want := strings.TrimSpace(strings.TrimPrefix(e.command.CommandPath(), root.Name()))
		if e.Command != want {
			t.Fatalf("command path changed: %q != %q", e.Command, want)
		}
	}
}

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
	for _, e := range collect(root, "*") {
		if e.Command == "private" || e.Command == "private read" {
			t.Fatal("hidden subtree leaked")
		}
	}
	groups := collect(root, "")
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
		entries := collect(root, query)
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
		if found := collect(root, query); len(found) != 0 {
			t.Fatalf("query %q leaked hidden/default text or ignored a term: %+v", query, found)
		}
	}
	entries := collect(root, "contact user-ids")
	shared := describePage(entries, func(*entry) {})
	for _, flag := range append(entries[0].Flags, shared...) {
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
	if strings.Contains(out.String(), "common_flags") {
		t.Fatalf("a single entry must keep its inherited flags inline: %s", out)
	}
	flags := result.Data[0].Flags
	if len(flags) != 2 || flags[0].Name != "identity-type" || flags[0].Description != "open_id, user_id or union_id" || flags[0].Default != "open_id" {
		t.Fatalf("missing inherited flag guidance or sorted output: %+v", flags)
	}
	if flags[1].Name != "selector" || flags[1].Description != usage || flags[1].Type != "string" || !flags[1].Required {
		t.Fatalf("local flag guidance was lost or replaced: %+v", flags[1])
	}
}

func TestDiscoveryListsSharedFlagsOncePerPage(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	root.PersistentFlags().String("identity-type", "open_id", "open_id, user_id or union_id")
	for _, name := range []string{"alpha", "beta"} {
		root.AddCommand(&cobra.Command{Use: name, Short: "Read item"})
	}
	root.AddCommand(NewCmd(f, root))
	root.SetArgs([]string{"discover", "*", "--flags", "--limit", "10"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
		Meta struct {
			CommonFlags []flag `json:"common_flags"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) < 3 || len(result.Meta.CommonFlags) != 1 || result.Meta.CommonFlags[0].Name != "identity-type" {
		t.Fatalf("shared flag not listed once in meta: %s", out)
	}
	for _, e := range result.Data {
		for _, fl := range e.Flags {
			if fl.Name == "identity-type" {
				t.Fatalf("shared flag repeated in %s: %s", e.Command, out)
			}
		}
	}
}

func TestDiscoverySharedFlagsRecoverEveryEntryFlag(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	root.PersistentFlags().String("identity-type", "open_id", "open_id, user_id or union_id")
	root.PersistentFlags().String("format", "json", "output format")
	root.PersistentFlags().Bool("dry-run", false, "preview only")
	contact := &cobra.Command{Use: "contact", Short: "Contacts"}
	contact.PersistentFlags().String("identity-type", "user_id", "contact identity namespace")
	lookup := &cobra.Command{Use: "lookup", Short: "Read a person"}
	lookup.Flags().String("format", "", "lookup format")
	lookup.Flags().String("selector", "", "local selector")
	contact.AddCommand(lookup)
	docs := &cobra.Command{Use: "docs", Short: "Documents"}
	docs.AddCommand(&cobra.Command{Use: "read", Short: "Read document"})
	root.AddCommand(contact, docs, NewCmd(f, root))
	root.SetArgs([]string{"discover", "*", "--flags", "--limit", "1000"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	expected := map[string]map[string]flag{}
	for _, e := range collect(root, "*") {
		local, inherited := splitPublicFlags(e.command)
		byName := map[string]flag{}
		for _, fl := range describeFlags(append(local, inherited...)) {
			byName[fl.Name] = fl
		}
		expected[e.Command] = byName
	}
	var result struct {
		Data []entry `json:"data"`
		Meta struct {
			CommonFlags []flag `json:"common_flags"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != len(expected) {
		t.Fatalf("page does not cover every entry: %d != %d", len(result.Data), len(expected))
	}
	shared := map[string]bool{}
	for _, fl := range result.Meta.CommonFlags {
		shared[fl.Name] = true
	}
	if !shared["dry-run"] || shared["identity-type"] || shared["format"] {
		t.Fatalf("wrong shared flags: %+v", result.Meta.CommonFlags)
	}
	lookupFlags := map[string]flag{}
	for _, e := range result.Data {
		got := map[string]flag{}
		for _, fl := range append(slices.Clone(e.Flags), result.Meta.CommonFlags...) {
			if _, dup := got[fl.Name]; dup {
				t.Fatalf("%s lists %s twice: %s", e.Command, fl.Name, out)
			}
			got[fl.Name] = fl
		}
		if !maps.Equal(got, expected[e.Command]) {
			t.Fatalf("%s recovered %+v, want %+v", e.Command, got, expected[e.Command])
		}
		if e.Command == "contact lookup" {
			for _, fl := range e.Flags {
				lookupFlags[fl.Name] = fl
			}
		}
	}
	if lookupFlags["identity-type"].Description != "contact identity namespace" || lookupFlags["format"].Description != "lookup format" || lookupFlags["dry-run"].Name != "" {
		t.Fatalf("subgroup override or local shadow lost: %+v", lookupFlags)
	}
}

func TestDiscoveryFlagsIncludeCompactRoutingExample(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	task := &cobra.Command{Use: "+complete", Short: "Complete a task", Example: "lark-cli task +complete --task-id <task-id>", RunE: func(*cobra.Command, []string) error { return nil }}
	task.Flags().String("task-id", "", "task GUID")
	_ = task.MarkFlagRequired("task-id")
	task.Flags().String("format", "json", "output format")
	root.AddCommand(task, NewCmd(f, root))
	root.SetArgs([]string{"discover", "complete", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Example != "test-cli task +complete --task-id <task-id>" {
		t.Fatalf("missing compact route: %s", out)
	}
}

func TestDiscoveryFlagsUseRequiredFlagSkeletonWithoutExample(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	get := &cobra.Command{Use: "get", Short: "Get one item", RunE: func(*cobra.Command, []string) error { return nil }}
	get.Flags().String("item-id", "", "item id")
	_ = get.MarkFlagRequired("item-id")
	get.Flags().Bool("all", false, "include all")
	_ = get.MarkFlagRequired("all")
	root.AddCommand(get, NewCmd(f, root))
	root.SetArgs([]string{"discover", "get", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Example != "test-cli get --all --item-id <item-id>" {
		t.Fatalf("missing required route: %s", out)
	}
}

func TestDiscoveryFlagsUseAffordanceExample(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	f.Affordance = affordance.NewResolver(fstest.MapFS{
		"task.md": &fstest.MapFile{Data: []byte("# task\n\n## +complete\nComplete a task.\n\n### Examples\n\n**Complete**\n```bash\nlark-cli task +complete --task-id task-a\n```\n")},
	}, apicatalog.Catalog{})
	root := &cobra.Command{Use: "test-cli"}
	task := &cobra.Command{Use: "+complete", Short: "Complete a task", RunE: func(*cobra.Command, []string) error { return nil }}
	task.Flags().String("task-id", "", "task GUID")
	_ = task.MarkFlagRequired("task-id")
	cmdmeta.SetAffordanceRef(task, "task", "+complete")
	root.AddCommand(task, NewCmd(f, root))
	root.SetArgs([]string{"discover", "complete", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Example != "test-cli task +complete --task-id task-a" {
		t.Fatalf("missing affordance route: %s", out)
	}
}

func TestDiscoveryRoutingOmitsNonExecutableGroups(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	task := &cobra.Command{Use: "task", Short: "Task group"}
	root.AddCommand(task, NewCmd(f, root))
	root.SetArgs([]string{"discover", "task", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Example != "" {
		t.Fatalf("group leaked a route: %s", out)
	}
}

func TestDiscoveryRoutingOmitsSkeletonWithoutRequiredFlags(t *testing.T) {
	f, out, _, _ := cmdutil.TestFactory(t, nil)
	root := &cobra.Command{Use: "test-cli"}
	list := &cobra.Command{Use: "list", Short: "List items", RunE: func(*cobra.Command, []string) error { return nil }}
	list.Flags().String("query", "", "optional query")
	root.AddCommand(list, NewCmd(f, root))
	root.SetArgs([]string{"discover", "list", "--flags", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []entry `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Example != "" {
		t.Fatalf("optional route was invented: %s", out)
	}
}

func TestDiscoveryFlaglessProjectionOmitsSharedFlags(t *testing.T) {
	for _, tc := range []struct {
		fields string
		want   bool
	}{{"command,example", false}, {"command, flags", true}} {
		f, out, _, _ := cmdutil.TestFactory(t, nil)
		root := &cobra.Command{Use: "test-cli"}
		root.PersistentFlags().String("result-fields", "", "select JSON result fields")
		root.PersistentFlags().String("identity-type", "open_id", "open_id, user_id or union_id")
		for _, name := range []string{"alpha", "beta"} {
			root.AddCommand(&cobra.Command{Use: name, Short: "Read item"})
		}
		root.AddCommand(NewCmd(f, root))
		root.SetArgs([]string{"discover", "item", "--flags", "--result-fields", tc.fields})
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := strings.Contains(out.String(), "common_flags"); got != tc.want {
			t.Errorf("--result-fields %q: common_flags present=%v, want %v: %s", tc.fields, got, tc.want, out)
		}
	}
}
