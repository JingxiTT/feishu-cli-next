// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package markdown

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/tidwall/gjson"
)

const privateMarkdownLink = "https://example.com/private-markdown"

// Source files and native rendering are separate capabilities. Both phases
// require the same independently established private parent and fresh identity.
// No version upload, Wiki creation, bot grant or recipient is involved.
func TestMarkdownNextPrivateLiveLifecycle(t *testing.T) {
	if os.Getenv("NEXT_MARKDOWN_LIVE") != "1" {
		t.Skip("requires coordinator-verified private parent, account binding and explicit live selection")
	}
	fixturePath, proofPath, evidence := os.Getenv("NEXT_MARKDOWN_LIVE_FIXTURE"), os.Getenv("NEXT_MARKDOWN_LIVE_BINDING"), os.Getenv("NEXT_MARKDOWN_LIVE_EVIDENCE_DIR")
	if !filepath.IsAbs(fixturePath) || !filepath.IsAbs(proofPath) || !filepath.IsAbs(evidence) {
		t.Fatal("absolute private fixture, binding and evidence paths required")
	}
	readPrivate := func(path string) string {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatal("owner-readable fixture evidence required")
		}
		b, err := os.ReadFile(path)
		if err != nil || !gjson.ValidBytes(b) {
			t.Fatal("invalid fixture evidence")
		}
		return string(b)
	}
	fixture, proof := readPrivate(fixturePath), readPrivate(proofPath)
	parent := gjson.Get(fixture, "folder_token").String()
	if parent == "" || !gjson.Get(fixture, "created_by_this_run").Bool() || gjson.Get(fixture, "invited_collaborators").Int() != 0 || !gjson.Get(fixture, "child_writes_require_cli_account_and_acl_preflight").Bool() {
		t.Fatal("private parent evidence incomplete")
	}
	if err := os.MkdirAll(evidence, 0700); err != nil {
		t.Fatal("cannot retain private evidence")
	}
	if err := os.Chmod(evidence, 0700); err != nil {
		t.Fatal("cannot restrict evidence directory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	sequence := 0
	call := func(t *testing.T, ctx context.Context, label string, request clie2e.Request) *clie2e.Result {
		t.Helper()
		sequence++
		// User files stay inside the invocation's real FileIO working root.
		request.WorkDir = evidence
		result, err := clie2e.RunCmdWithRetry(ctx, request, clie2e.RetryOptions{Attempts: 1})
		record := map[string]any{"label": label, "attempts": 1}
		if result != nil {
			record["stdout"] = result.Stdout
			record["stderr"] = result.Stderr
			record["exit_code"] = result.ExitCode
		}
		if err != nil {
			record["process_error"] = err.Error()
		}
		raw, _ := json.Marshal(record)
		// Retain failure output before assertions, including unknown create results.
		if writeErr := os.WriteFile(filepath.Join(evidence, fmt.Sprintf("%02d-%s.json", sequence, label)), raw, 0600); writeErr != nil {
			t.Error("cannot retain private command evidence")
		}
		if err != nil || result == nil {
			t.Fatal("CLI process failed; no replay performed")
		}
		return result
	}
	ok := func(t *testing.T, label string, result *clie2e.Result) string {
		t.Helper()
		if result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
			t.Fatalf("%s failed (exit %d); private diagnostics retained; no replay", label, result.ExitCode)
		}
		return result.Stdout
	}
	user := func(t *testing.T, label string, args ...string) string {
		t.Helper()
		return ok(t, label, call(t, ctx, label, clie2e.Request{Args: args, DefaultAs: "user"}))
	}
	auth := call(t, ctx, "auth-binding", clie2e.Request{Args: []string{"auth", "status", "--verify", "--json"}})
	if auth.ExitCode != 0 || !gjson.Get(auth.Stdout, "verified").Bool() || !gjson.Get(auth.Stdout, "identities.user.verified").Bool() {
		t.Fatal("verified current-user identity unavailable")
	}
	self := user(t, "self-binding", "contact", "+get-user")
	account, tenant := gjson.Get(self, "data.user.open_id").String(), gjson.Get(self, "data.user.tenant_key").String()
	digest := sha256.Sum256([]byte("contact-current-account-v1\x00" + account + "\x00" + tenant))
	if account == "" || tenant == "" || gjson.Get(auth.Stdout, "identities.user.openId").String() != account || gjson.Get(proof, "algorithm").String() != "sha256" || hex.EncodeToString(digest[:]) != gjson.Get(proof, "digest").String() {
		t.Fatal("independent account/tenant binding mismatch")
	}
	acl := func(t *testing.T, label, token, kind string) {
		t.Helper()
		settings := user(t, label+"-settings", "drive", "+permission-get-setting", "--token", token, "--type", kind)
		if gjson.Get(settings, "data.permission_public.link_share_entity").String() != "closed" {
			t.Fatal("resource link sharing is not closed")
		}
		members := user(t, label+"-members", "drive", "+member-list", "--token", token, "--type", kind)
		items := gjson.Get(members, "data.items")
		if !items.IsArray() || len(items.Array()) != 1 {
			t.Fatal("resource does not have exactly one collaborator")
		}
		m := items.Array()[0]
		if m.Get("member_type").String() != "openid" || m.Get("member_id").String() != account || m.Get("perm").String() != "full_access" || (kind == "folder" && m.Get("perm_type").String() != "container") {
			t.Fatal("resource collaborator is not the current owner")
		}
	}
	metadata := func(t *testing.T, ctx context.Context, label, token, kind string) *clie2e.Result {
		return call(t, ctx, label, clie2e.Request{Args: []string{"drive", "metas", "batch_query"}, DefaultAs: "user", Data: map[string]any{"request_docs": []any{map[string]string{"doc_token": token, "doc_type": kind}}, "with_url": false}})
	}
	owned := func(raw, token, kind string) bool {
		items := gjson.Get(raw, "data.metas")
		return items.IsArray() && len(items.Array()) == 1 && items.Array()[0].Get("doc_token").String() == token && items.Array()[0].Get("doc_type").String() == kind && items.Array()[0].Get("owner_id").String() == account
	}
	acl(t, "parent", parent, "folder")
	if !owned(ok(t, "parent-owner", metadata(t, ctx, "parent-owner", parent, "folder")), parent, "folder") {
		t.Fatal("private parent metadata owner mismatch")
	}
	suffix := clie2e.GenerateSuffix()
	name := "next-private-markdown-" + suffix
	source := "# 私有 Markdown 你好 😀\n\nKeep **bold** and [link](" + privateMarkdownLink + ").\n\n| Key | Value |\n| --- | --- |\n| 001 | résumé 😀 |\n\n- [ ] pending\n\n```text\n001 & <literal>\n```\n"
	sourcePath := filepath.Join(evidence, "source.md")
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal("cannot retain Markdown input")
	}
	// Register exact returned IDs before asserting success. The cleanup itself
	// rechecks metadata ownership and sends one delete only, even on unknown exit.
	register := func(t *testing.T, label, token, kind string) {
		t.Helper()
		if token == "" || token == parent {
			t.Fatal("create did not return an unambiguous run-owned ID; no guessed cleanup or replay")
		}
		manifest, _ := json.Marshal(map[string]string{"token": token, "type": kind, "parent": parent, "name": name})
		if err := os.WriteFile(filepath.Join(evidence, label+"-run-owned.json"), manifest, 0600); err != nil {
			t.Error("cannot retain run-owned cleanup binding")
		}
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cleanupCancel()
			meta := metadata(t, cleanupCtx, label+"-cleanup-owner", token, kind)
			if meta.ExitCode != 0 || !gjson.Get(meta.Stdout, "ok").Bool() || !owned(meta.Stdout, token, kind) {
				t.Error("cleanup ownership not established; no delete performed")
				return
			}
			deletion := call(t, cleanupCtx, label+"-delete", clie2e.Request{Args: []string{"drive", "+delete", "--file-token", token, "--type", kind}, DefaultAs: "user", Yes: true})
			if deletion.ExitCode != 0 || !gjson.Get(deletion.Stdout, "data.deleted").Bool() {
				t.Error("delete outcome unconfirmed; no retry performed")
				return
			}
			path := "/open-apis/docx/v1/documents/" + token
			params := map[string]string(nil)
			if kind == "file" {
				path = "/open-apis/drive/v1/permissions/" + token + "/members/auth"
				params = map[string]string{"type": "file", "action": "export"}
			}
			request := clie2e.Request{Args: []string{"api", "GET", path}, DefaultAs: "user"}
			if params != nil {
				request.Params = params
			}
			absent := call(t, cleanupCtx, label+"-deleted-proof", request)
			if !privateMarkdownDeleted(absent, kind) {
				t.Error("independent native deletion proof missing; no repeated delete")
				return
			}
			t.Log("run-owned child deletion independently confirmed; parent untouched")
		})
	}
	t.Run("source_file", func(t *testing.T) {
		// Parent ACL is re-read immediately before each independent create phase.
		acl(t, "source-parent", parent, "folder")
		created := call(t, ctx, "source-create", clie2e.Request{Args: []string{"markdown", "+create", "--folder-token", parent, "--name", name + ".md", "--file", sourcePath}, DefaultAs: "user"})
		token := gjson.Get(created.Stdout, "data.file_token").String()
		register(t, "source", token, "file")
		ok(t, "source-create", created)
		acl(t, "source-child", token, "file")
		fetched := user(t, "source-fetch", "markdown", "+fetch", "--file-token", token)
		if gjson.Get(fetched, "data.content").String() != source || gjson.Get(fetched, "data.size_bytes").Int() != int64(len(source)) {
			t.Fatal("Markdown source preview changed bytes")
		}
		saved := filepath.Join(evidence, "downloaded-source.md")
		user(t, "source-save", "markdown", "+fetch", "--file-token", token, "--output", saved)
		bytes, err := os.ReadFile(saved)
		if err != nil || string(bytes) != source {
			t.Fatal("saved Markdown source differs")
		}
		t.Log("private Markdown source upload/preview/read/save bytes verified")
	})
	t.Run("native_render", func(t *testing.T) {
		acl(t, "render-parent", parent, "folder")
		created := call(t, ctx, "render-create", clie2e.Request{Args: []string{"docs", "+create", "--parent-token", parent, "--title", name, "--doc-format", "markdown", "--content", "@" + sourcePath}, DefaultAs: "user"})
		token := gjson.Get(created.Stdout, "data.document.document_id").String()
		register(t, "render", token, "docx")
		ok(t, "render-create", created)
		acl(t, "render-child", token, "docx")
		xml := user(t, "render-xml", "docs", "+fetch", "--doc", token, "--doc-format", "xml", "--detail", "full")
		content := gjson.Get(xml, "data.document.content").String()
		for _, part := range []string{"你好", "😀", "<table", "001", "résumé"} {
			if !strings.Contains(content, part) {
				t.Fatal("native XML rendering omitted a required Markdown feature")
			}
		}
		native := user(t, "render-native", "api", "GET", "/open-apis/docx/v1/documents/"+token+"/blocks", "--params", `{"page_size":500}`)
		if err := privateMarkdownNative(native); err != nil {
			t.Fatal(err)
		}
		t.Log("private Markdown-to-native heading/bold/link/table/Unicode/code rendering verified")
	})
}

func privateMarkdownDeleted(result *clie2e.Result, kind string) bool {
	if result == nil || result.ExitCode == 0 || gjson.Get(result.Stderr, "error.type").String() != "api" || gjson.Get(result.Stderr, "error.code").Type != gjson.Number {
		return false
	}
	code, message := gjson.Get(result.Stderr, "error.code").Int(), gjson.Get(result.Stderr, "error.message").String()
	return kind == "docx" && code == 1770003 && message == "resource deleted" || kind == "file" && code == 1063005 && message == "Resource is deleted"
}

func privateMarkdownNative(raw string) error {
	items := gjson.Get(raw, "data.items")
	if !items.IsArray() || gjson.Get(raw, "data.has_more").Bool() {
		return fmt.Errorf("native Markdown block extent incomplete")
	}
	heading, table, code, bold, link, unicode, zero, accent := false, false, false, false, false, false, false, false
	for _, block := range items.Array() {
		kind := block.Get("block_type").Int()
		heading = heading || (kind >= 3 && kind <= 11)
		table = table || kind == 31
		codeText := ""
		for _, element := range block.Get("code.elements").Array() {
			codeText += element.Get("text_run.content").String()
		}
		code = code || kind == 14 && codeText == "001 & <literal>"
		for _, field := range []string{"text", "heading1", "heading2", "heading3", "code", "todo"} {
			for _, el := range block.Get(field + ".elements").Array() {
				text := el.Get("text_run.content").String()
				style := el.Get("text_run.text_element_style")
				bold = bold || text == "bold" && style.Get("bold").Bool()
				link = link || text == "link" && style.Get("link.url").String() == privateMarkdownLink
				unicode = unicode || strings.Contains(text, "你好") && strings.Contains(text, "😀")
				zero = zero || field == "text" && text == "001"
				accent = accent || strings.Contains(text, "résumé 😀")
			}
		}
	}
	if !heading || !table || !code || !bold || !link || !unicode || !zero || !accent {
		return fmt.Errorf("native Markdown heading/table/code/bold/link/Unicode/leading-zero invariant missing")
	}
	return nil
}

func TestMarkdownPrivateDeletedProof(t *testing.T) {
	for _, tc := range []struct {
		kind    string
		code    int
		message string
	}{{"file", 1063005, "Resource is deleted"}, {"docx", 1770003, "resource deleted"}} {
		raw, _ := json.Marshal(map[string]any{"error": map[string]any{"type": "api", "code": tc.code, "message": tc.message}})
		if !privateMarkdownDeleted(&clie2e.Result{ExitCode: 1, Stderr: string(raw)}, tc.kind) {
			t.Fatal("native deleted evidence rejected")
		}
	}
	if privateMarkdownDeleted(&clie2e.Result{ExitCode: 0, Stdout: `{"data":{"metas":[{"url":""}]}}`}, "file") {
		t.Fatal("empty metadata URL accepted as deletion")
	}
}

func TestMarkdownPrivateNativeVerifier(t *testing.T) {
	fixture := `{"data":{"has_more":false,"items":[{"block_type":3,"heading1":{"elements":[{"text_run":{"content":"你好 😀"}}]}},{"block_type":31},{"block_type":14,"code":{"elements":[{"text_run":{"content":"001 & <literal>"}}]}},{"block_type":2,"text":{"elements":[{"text_run":{"content":"001"}},{"text_run":{"content":"résumé 😀"}},{"text_run":{"content":"bold","text_element_style":{"bold":true}}},{"text_run":{"content":"link","text_element_style":{"link":{"url":"https://example.com/private-markdown"}}}}]}}]}}`
	if err := privateMarkdownNative(fixture); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(fixture, `"bold":true`, `"bold":false`, 1),
		strings.Replace(fixture, privateMarkdownLink, "https://example.com/wrong", 1),
		strings.Replace(fixture, `"block_type":31`, `"block_type":2`, 1),
		strings.Replace(fixture, `"has_more":false`, `"has_more":true`, 1),
		strings.Replace(fixture, "你好", "lost", 1),
		strings.Replace(fixture, "<literal>", "lost", 1),
	} {
		if privateMarkdownNative(bad) == nil {
			t.Fatal("incomplete native rendering accepted")
		}
	}
}
