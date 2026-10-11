// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package lint

import (
	"strings"
	"testing"
)

func TestRunRejectsUnparseableHTML(t *testing.T) {
	body := strings.Repeat("<div>", 600) + `<script>alert(1)</script>text` + strings.Repeat("</div>", 600)
	rep := Run(body, Options{})
	if rep.ParseError == nil {
		t.Fatal("expected excessive nesting to fail parsing")
	}
	if rep.CleanedHTML != "" {
		t.Fatal("unsafe original HTML must not be returned as cleaned HTML")
	}
	if rep.Applied == nil || rep.Blocked == nil {
		t.Fatal("finding arrays must remain non-nil")
	}
}
