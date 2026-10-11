// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
)

func TestRunWritePathLintRejectsUnparseableHTML(t *testing.T) {
	body := strings.Repeat("<div>", 600) + `<script>alert(1)</script>text` + strings.Repeat("</div>", 600)
	cleaned, rep, err := runWritePathLint(body)
	var typed *errs.ValidationError
	if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument {
		t.Fatalf("expected typed mail validation error, got %v", err)
	}
	if rep.ParseError == nil || !errors.Is(err, rep.ParseError) {
		t.Fatalf("parse error cause was lost: %v", err)
	}
	if cleaned != "" || rep.CleanedHTML != "" {
		t.Fatal("uninspected HTML must not become writable output")
	}
}

func TestComposeRejectsUnparseableHTMLBeforeMutation(t *testing.T) {
	for _, shortcut := range []common.Shortcut{MailDraftCreate, MailSend} {
		t.Run(shortcut.Command, func(t *testing.T) {
			f, stdout, _, reg := mailShortcutTestFactory(t)
			chdirTemp(t)
			registerMailboxProfileMock(reg)
			mutation := &httpmock.Stub{Method: "POST", URL: "/user_mailboxes/me/", Optional: true, Body: map[string]interface{}{"code": 0}}
			reg.Register(mutation)
			body := strings.Repeat("<div>", 600) + `<script>alert(1)</script>text` + strings.Repeat("</div>", 600)
			err := runMountedMailShortcut(t, shortcut, []string{shortcut.Command, "--to", "alice@example.com", "--subject", "Test", "--body", body}, f, stdout)
			var typed *errs.ValidationError
			if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || errors.Unwrap(typed) == nil {
				t.Fatalf("expected typed validation preserving parser error, got %v", err)
			}
			if len(mutation.CapturedBodies) != 0 {
				t.Fatal("parse failure must prevent mutation")
			}
			if stdout.Len() != 0 {
				t.Fatal("parse failure must not emit success")
			}
		})
	}
}

func TestDraftEditRejectsUnparseableHTMLBeforeAttachmentUpload(t *testing.T) {
	f, stdout, _, reg := mailShortcutTestFactory(t)
	chdirTemp(t)
	originalEML := "MIME-Version: 1.0\r\nFrom: me@example.com\r\nTo: alice@example.com\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<p>original body</p>"
	reg.Register(&httpmock.Stub{
		Method: "GET", URL: "/user_mailboxes/me/drafts/d_edit",
		Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"draft_id": "d_edit", "raw": base64URLEncode(originalEML)}},
	})
	upload := &httpmock.Stub{
		Method: "POST", URL: "/drive/v1/medias/upload_all", Optional: true,
		Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"file_token": "file_EXAMPLE_TOKEN"}},
	}
	reg.Register(upload)
	update := &httpmock.Stub{Method: "PUT", URL: "/user_mailboxes/me/drafts/d_edit", Optional: true, Body: map[string]interface{}{"code": 0}}
	reg.Register(update)
	if err := os.WriteFile("large.zip", make([]byte, 19<<20), 0600); err != nil {
		t.Fatal(err)
	}
	body := strings.Repeat("<div>", 600) + "text" + strings.Repeat("</div>", 600)
	patch, err := json.Marshal(map[string]interface{}{"ops": []map[string]string{
		{"op": "add_attachment", "path": "large.zip"},
		{"op": "set_body", "value": body},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("patch.json", patch, 0600); err != nil {
		t.Fatal(err)
	}
	err = runMountedMailShortcut(t, MailDraftEdit, []string{"+draft-edit", "--draft-id", "d_edit", "--patch-file", "patch.json"}, f, stdout)
	var typed *errs.ValidationError
	if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || errors.Unwrap(typed) == nil {
		t.Fatalf("expected typed validation preserving parser cause, got %v", err)
	}
	if len(upload.CapturedBodies) != 0 || len(update.CapturedBodies) != 0 {
		t.Fatal("unparseable body must prevent attachment upload and draft update")
	}
	if stdout.Len() != 0 {
		t.Fatal("parse failure must not emit success")
	}
}

func TestDraftEditRejectsUnparseableAttachmentCardBeforeUpdate(t *testing.T) {
	for _, withUpload := range []bool{false, true} {
		name := "remove_only"
		if withUpload {
			name = "remove_and_upload"
		}
		t.Run(name, func(t *testing.T) {
			f, stdout, _, reg := mailShortcutTestFactory(t)
			chdirTemp(t)
			body := strings.Repeat("<div>", 600) + `<div id="large-file-area-1"><div id="large-file-item"><a data-mail-token="tokA">file.pdf</a></div></div>` + strings.Repeat("</div>", 600)
			header := base64.StdEncoding.EncodeToString([]byte(`[{"id":"tokA"}]`))
			original := "MIME-Version: 1.0\r\nFrom: me@example.com\r\nTo: alice@example.com\r\nX-Lms-Large-Attachment-Ids: " + header + "\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" + body
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/user_mailboxes/me/drafts/d_edit", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"draft_id": "d_edit", "raw": base64URLEncode(original)}}})
			update := &httpmock.Stub{Method: "PUT", URL: "/user_mailboxes/me/drafts/d_edit", Optional: true, Body: map[string]interface{}{"code": 0}}
			reg.Register(update)
			upload := &httpmock.Stub{Method: "POST", URL: "/drive/v1/medias/upload_all", Optional: true, Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"file_token": "file_EXAMPLE_TOKEN"}}}
			reg.Register(upload)
			ops := []map[string]interface{}{{"op": "remove_attachment", "target": map[string]string{"token": "tokA"}}}
			if withUpload {
				if err := os.WriteFile("large.zip", make([]byte, 19<<20), 0600); err != nil {
					t.Fatal(err)
				}
				ops = append(ops, map[string]interface{}{"op": "add_attachment", "path": "large.zip"})
			}
			patch, err := json.Marshal(map[string]interface{}{"ops": ops})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("patch.json", patch, 0600); err != nil {
				t.Fatal(err)
			}
			err = runMountedMailShortcut(t, MailDraftEdit, []string{"+draft-edit", "--draft-id", "d_edit", "--patch-file", "patch.json"}, f, stdout)
			var typed *errs.ValidationError
			if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || errors.Unwrap(typed) == nil {
				t.Fatalf("expected typed validation preserving attachment parser cause, got %v", err)
			}
			if len(update.CapturedBodies) != 0 || len(upload.CapturedBodies) != 0 {
				t.Fatal("unparseable attachment card must prevent uploads and draft update")
			}
			if stdout.Len() != 0 {
				t.Fatal("failed removal must not emit success")
			}
		})
	}
}
