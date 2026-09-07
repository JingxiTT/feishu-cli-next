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
