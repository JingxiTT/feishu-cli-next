// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"
)

func TestDocsNextGuidancePreservesNativeEditingDecisions(t *testing.T) {
	raw, err := os.ReadFile("../../affordance/docs.md")
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	for _, decision := range []string{"document.reference_map", "matching scope", "exact unchanged", "Markdown equality", "concurrent edits", "returned receipts", "pending work"} {
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
