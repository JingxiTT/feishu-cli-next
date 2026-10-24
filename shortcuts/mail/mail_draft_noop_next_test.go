// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail

import (
	"encoding/base64"
	"testing"

	"github.com/larksuite/cli/internal/httpmock"
	draftpkg "github.com/larksuite/cli/shortcuts/mail/draft"
)

func TestDraftReplacementNoOpDoesNotWrite(t *testing.T) {
	for _, args := range [][]string{{"--set-subject", "Original"}, {"--body", "hello\n"}, {"--set-priority", "normal"}} {
		t.Run(args[0], func(t *testing.T) {
			f, stdout, _, reg := mailShortcutTestFactory(t)
			raw := base64.RawURLEncoding.EncodeToString([]byte("From: fixture@example.invalid\r\nTo: fixture@example.invalid\r\nSubject: Original\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nhello\r\n"))
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/user_mailboxes/me/drafts/d1", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"draft_id": "d1", "raw": raw}}})
			err := runMountedMailShortcut(t, MailDraftEdit, append([]string{"+draft-edit", "--draft-id", "d1"}, args...), f, stdout)
			if err != nil {
				t.Fatal(err)
			}
			data := decodeShortcutEnvelopeData(t, stdout)
			if data["no_op"] != true || data["draft_id"] != "d1" {
				t.Fatalf("unexpected result: %#v", data)
			}
			// Only GET is registered; any PUT fails the mock transport and command.
		})
	}
}

func TestDraftNoOpDoesNotSuppressExplicitEffects(t *testing.T) {
	for _, op := range []string{"append_body", "add_attachment", "insert_signature", "set_calendar", "remove_attachment"} {
		if draftReplacementOnly(draftpkg.Patch{Ops: []draftpkg.PatchOp{{Op: op}}}) {
			t.Fatalf("classified %s as replacement", op)
		}
	}
}

func TestTemplateReplacementNoOpDoesNotWrite(t *testing.T) {
	f, stdout, _, reg := mailShortcutTestFactory(t)
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/user_mailboxes/me/templates/77", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"template": map[string]interface{}{"template_id": "77", "name": "Original", "subject": "Original", "template_content": "<p>正文</p>"}}}})
	if err := runMountedMailShortcut(t, MailTemplateUpdate, []string{"+template-update", "--template-id", "77", "--set-subject", "Original"}, f, stdout); err != nil {
		t.Fatal(err)
	}
	if got := decodeShortcutEnvelopeData(t, stdout); got["no_op"] != true {
		t.Fatalf("%#v", got)
	}
}
