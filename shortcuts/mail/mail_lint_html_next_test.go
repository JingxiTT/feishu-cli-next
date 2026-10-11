// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail

import (
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestMailLintHTMLRejectsUnparseableHTML(t *testing.T) {
	f, stdout, _, _ := mailShortcutTestFactory(t)
	body := strings.Repeat("<div>", 600) + `<script>alert(1)</script>` + strings.Repeat("</div>", 600)
	err := runMountedMailShortcut(t, MailLintHTML, []string{"+lint-html", "--body", body}, f, stdout)
	var typed *errs.ValidationError
	if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || errors.Unwrap(typed) == nil {
		t.Fatalf("expected typed validation failure preserving parser cause, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("parse failure must not emit success: %s", stdout.String())
	}
}
