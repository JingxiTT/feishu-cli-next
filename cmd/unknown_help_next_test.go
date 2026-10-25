// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

func newUnknownHelpTree() *cobra.Command {
	nop := func(*cobra.Command, []string) error { return nil }
	root := &cobra.Command{Use: "lark-cli"}
	sheets := &cobra.Command{Use: "sheets", Short: "Spreadsheet operations"}
	sheets.AddCommand(&cobra.Command{Use: "+cells-get", Short: "Read one or more cell ranges", RunE: nop})
	docs := &cobra.Command{Use: "docs", Short: "Document operations"}
	docs.AddCommand(
		&cobra.Command{Use: "+pull", Short: "Pull a document", RunE: nop},
		&cobra.Command{Use: "+push", Short: "Push a document", RunE: nop},
	)
	root.AddCommand(sheets, docs)
	installUnknownSubcommandGuard(root)
	return root
}

func runUnknownHelpTree(args ...string) (*bytes.Buffer, error) {
	root := newUnknownHelpTree()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	return &out, executeRejectingUnknownHelp(root, args)
}

func TestExecuteRejectingUnknownHelp_UnknownShortcutIsStructuredError(t *testing.T) {
	out, err := runUnknownHelpTree("sheets", "+read", "--help")

	var verr *errs.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *errs.ValidationError, got %T (%v)", err, err)
	}
	if got := output.ExitCodeOf(err); got != output.ExitValidation {
		t.Errorf("exit code = %d, want %d", got, output.ExitValidation)
	}
	if out.Len() != 0 {
		t.Errorf("group help must not print for an unknown subcommand, got:\n%s", out.String())
	}
	if len(verr.Params) != 1 || !slices.Contains(verr.Params[0].Suggestions, "+cells-get") {
		t.Errorf("suggestions should include +cells-get, got %+v", verr.Params)
	}
}

func TestExecuteRejectingUnknownHelp_GroupHelpStillPrints(t *testing.T) {
	out, err := runUnknownHelpTree("sheets", "--help")
	if err != nil {
		t.Fatalf("group help returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Spreadsheet operations") {
		t.Errorf("expected group help, got:\n%s", out.String())
	}
}

func TestExecuteRejectingUnknownHelp_TypoSuggestsEditDistance(t *testing.T) {
	_, err := runUnknownHelpTree("docs", "+pul", "--help")

	var verr *errs.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *errs.ValidationError, got %T (%v)", err, err)
	}
	if len(verr.Params) != 1 || !slices.Contains(verr.Params[0].Suggestions, "+pull") {
		t.Errorf("suggestions should include +pull, got %+v", verr.Params)
	}
}

func TestExecuteRejectingUnknownHelp_ValidCommandHelpUnchanged(t *testing.T) {
	out, err := runUnknownHelpTree("docs", "+pull", "--help")
	if err != nil {
		t.Fatalf("valid command help returned error: %v", err)
	}
	if !strings.Contains(out.String(), "Pull a document") {
		t.Errorf("expected +pull help, got:\n%s", out.String())
	}
}

func TestUnknownSubcommandSuggestions_KeywordHitsLeadEditDistance(t *testing.T) {
	root := newUnknownHelpTree()
	sheets, _, _ := root.Find([]string{"sheets"})
	names := []string{"+cells-get"}

	got := unknownSubcommandSuggestions(sheets, "+read", names)
	if len(got) == 0 || got[0] != "+cells-get" {
		t.Errorf("keyword hit +cells-get should lead: %v", got)
	}
	if len(got) != 1 {
		t.Errorf("duplicate or extra suggestions: %v", got)
	}
}

func TestUnknownSubcommandSuggestions_WildcardAddsNoKeywordHits(t *testing.T) {
	root := newUnknownHelpTree()
	sheets, _, _ := root.Find([]string{"sheets"})

	if got := unknownSubcommandSuggestions(sheets, "*", []string{"+cells-get"}); len(got) != 0 {
		t.Errorf("wildcard should not suggest every child, got %v", got)
	}
}
