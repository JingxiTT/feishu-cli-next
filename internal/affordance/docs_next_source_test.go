// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/meta"
)

func TestDocsNextGuidancePreservesNativeEditingDecisions(t *testing.T) {
	raw, err := os.ReadFile("../../affordance/docs.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, decision := range []string{"document.reference_map", "matching scope", "exact unchanged", "Markdown equality", "concurrent edits", "returned receipts", "pending work", "plain replacement can remove existing formatting"} {
		if !strings.Contains(content, decision) {
			t.Errorf("Docs guidance missing %q", decision)
		}
	}
	for _, procedure := range []string{"Before authoring `--content`, read", "unless already read"} {
		if strings.Contains(content, procedure) {
			t.Errorf("Docs guidance mandates unnecessary repeated discovery: %q", procedure)
		}
	}
}

func TestDocsTargetedEditGuidanceIsParsedIntoCommandTips(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	raw, ok := For("docs", "+update")
	if !ok {
		t.Fatal("missing docs +update overlay")
	}
	parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
	if !ok {
		t.Fatal("docs +update overlay did not parse")
	}
	tips := strings.Join(parsed.Tips, "\n")
	for _, decision := range []string{
		"smallest `--pattern` established unique across the document",
		"leave unchanged styled text outside the match",
		"native block edit addressed by ID",
		"preserving unrelated block IDs and resources",
		"actual XML marks and link `href`",
		"verify text, styles and link destinations",
	} {
		if !strings.Contains(tips, decision) {
			t.Errorf("parsed command tips missing %q", decision)
		}
	}
}

func TestDocsFactReadGuidanceIsParsedIntoCommandTips(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	raw, ok := For("docs", "+fetch")
	if !ok {
		t.Fatal("missing docs +fetch overlay")
	}
	parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
	if !ok {
		t.Fatal("docs +fetch overlay did not parse")
	}
	tips := strings.Join(parsed.Tips, "\n")
	for _, decision := range []string{
		"For facts and native block addresses, choose XML `with-ids`",
		"use `full` when styles and edit/reference metadata are needed",
		"Read small fact-lookup results directly from JSON stdout",
		"requested exports or large responses",
		"`document.reference_map` alongside `document.content`",
		"Bounds require a matching scope",
	} {
		if !strings.Contains(tips, decision) {
			t.Errorf("parsed command tips missing %q", decision)
		}
	}
	if !containsExact(parsed.Skills, "lark-doc/references/lark-doc-fetch.md") {
		t.Fatal("fetch guide is no longer reachable from command guidance")
	}
}
