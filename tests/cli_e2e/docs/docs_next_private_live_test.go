// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/tidwall/gjson"
)

const nextLiveDocsLink = "https://example.com/next-docs-private"

// Deliberately separate from the inherited workflows, which create unverified
// parent folders. The coordinator supplies prior private-creation evidence; this
// test freshly binds the account and folder ACL before its single create call.
func TestDocsNextPrivateLiveLifecycle(t *testing.T) {
	if os.Getenv("NEXT_DOCS_LIVE") != "1" {
		t.Skip("opt-in private Docs batch; requires coordinator-verified creation visibility and binding proof")
	}
	fixturePath, proofPath, evidenceDir := os.Getenv("NEXT_DOCS_LIVE_FIXTURE"), os.Getenv("NEXT_DOCS_LIVE_BINDING"), os.Getenv("NEXT_DOCS_LIVE_EVIDENCE_DIR")
	if !filepath.IsAbs(fixturePath) || !filepath.IsAbs(proofPath) || !filepath.IsAbs(evidenceDir) {
		t.Fatal("absolute private fixture, binding proof and evidence directory are required")
	}
	readPrivate := func(path string) string {
		t.Helper()
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0077 != 0 {
			t.Fatal("fixture evidence must exist with owner-only permissions")
		}
		raw, err := os.ReadFile(path)
		if err != nil || !gjson.ValidBytes(raw) {
			t.Fatal("invalid private fixture evidence")
		}
		return string(raw)
	}
	fixture, binding := readPrivate(fixturePath), readPrivate(proofPath)
	parent := gjson.Get(fixture, "folder_token").String()
	if parent == "" || !gjson.Get(fixture, "created_by_this_run").Bool() || gjson.Get(fixture, "invited_collaborators").Int() != 0 || !gjson.Get(fixture, "child_writes_require_cli_account_and_acl_preflight").Bool() {
		t.Fatal("private parent fixture is incomplete")
	}
	if gjson.Get(binding, "algorithm").String() != "sha256" || gjson.Get(binding, "digest").String() == "" {
		t.Fatal("invalid independent account binding proof")
	}
	if err := os.MkdirAll(evidenceDir, 0700); err != nil {
		t.Fatal("cannot create private evidence directory")
	}
	if err := os.Chmod(evidenceDir, 0700); err != nil {
		t.Fatal("cannot restrict evidence directory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	calls := 0
	call := func(callCtx context.Context, label string, req clie2e.Request) *clie2e.Result {
		t.Helper()
		calls++
		// RunCmd defaults to retries. Attempts:1 is essential for unknown creates.
		result, err := clie2e.RunCmdWithRetry(callCtx, req, clie2e.RetryOptions{Attempts: 1})
		if result != nil {
			private, _ := json.Marshal(map[string]any{"label": label, "exit_code": result.ExitCode, "stdout": result.Stdout, "stderr": result.Stderr})
			if writeErr := os.WriteFile(filepath.Join(evidenceDir, fmt.Sprintf("%02d-%s.json", calls, label)), private, 0600); writeErr != nil {
				t.Error("cannot retain private command evidence")
			}
		}
		if err != nil || result == nil {
			t.Fatal("CLI process could not run; no retry performed")
		}
		return result
	}
	requireOK := func(label string, result *clie2e.Result) string {
		t.Helper()
		if result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
			t.Fatalf("%s failed (exit %d); private evidence retained; no retry performed", label, result.ExitCode)
		}
		return result.Stdout
	}
	user := func(label string, args ...string) string {
		t.Helper()
		return requireOK(label, call(ctx, label, clie2e.Request{Args: args, DefaultAs: "user"}))
	}
	authResult := call(ctx, "auth-binding", clie2e.Request{Args: []string{"auth", "status", "--verify", "--json"}})
	auth := authResult.Stdout
	if authResult.ExitCode != 0 || !gjson.Get(auth, "verified").Bool() || !gjson.Get(auth, "identities.user.verified").Bool() {
		t.Fatal("verified user identity unavailable; no create performed")
	}
	self := user("self-binding", "contact", "+get-user")
	account, tenant := gjson.Get(self, "data.user.open_id").String(), gjson.Get(self, "data.user.tenant_key").String()
	sum := sha256.Sum256([]byte("contact-current-account-v1\x00" + account + "\x00" + tenant))
	if account == "" || tenant == "" || gjson.Get(auth, "identities.user.openId").String() != account || hex.EncodeToString(sum[:]) != gjson.Get(binding, "digest").String() {
		t.Fatal("independent account/tenant binding mismatch; no create performed")
	}
	checkACL := func(label, token, kind string) {
		t.Helper()
		settings := user(label+"-settings", "drive", "+permission-get-setting", "--token", token, "--type", kind)
		if gjson.Get(settings, "data.permission_public.link_share_entity").String() != "closed" {
			t.Fatal("private resource link access is not closed")
		}
		members := user(label+"-members", "drive", "+member-list", "--token", token, "--type", kind)
		items := gjson.Get(members, "data.items")
		if !items.IsArray() || len(items.Array()) != 1 {
			t.Fatal("private resource does not have exactly one explicit collaborator")
		}
		member := items.Array()[0]
		if member.Get("member_type").String() != "openid" || member.Get("member_id").String() != account || member.Get("perm").String() != "full_access" {
			t.Fatal("private resource sole collaborator is not current owner")
		}
		if kind == "folder" && member.Get("perm_type").String() != "container" {
			t.Fatal("private parent membership does not cover its container")
		}
	}
	checkACL("parent", parent, "folder")
	metaReq := clie2e.Request{Args: []string{"drive", "metas", "batch_query"}, DefaultAs: "user", Data: map[string]any{"request_docs": []any{map[string]string{"doc_token": parent, "doc_type": "folder"}}, "with_url": false}}
	parentMeta := requireOK("parent-owner", call(ctx, "parent-owner", metaReq))
	metas := gjson.Get(parentMeta, "data.metas").Array()
	if len(metas) != 1 || metas[0].Get("doc_token").String() != parent || metas[0].Get("doc_type").String() != "folder" || metas[0].Get("owner_id").String() != account {
		t.Fatal("private parent metadata owner mismatch; no create performed")
	}
	title := "next-docs-private-" + clie2e.GenerateSuffix()
	content := `<h1>私有验证 😀</h1><p>Old <a href="` + nextLiveDocsLink + `"><b>status</b></a> — 你好</p><p>Keep résumé 😀 and 001 unchanged.</p><table><tr><td><p>001</p></td><td><p>é</p></td></tr><tr><td><p>二</p></td><td><p>😀</p></td></tr></table>`
	createResult := call(ctx, "create", clie2e.Request{Args: []string{"docs", "+create", "--parent-token", parent, "--title", title, "--doc-format", "xml", "--content", content}, DefaultAs: "user"})
	// Preserve a returned ID and register guarded cleanup even if the CLI reports
	// a partial/unknown result after the backend created the document.
	docID := gjson.Get(createResult.Stdout, "data.document.document_id").String()
	if docID == "" || docID == parent {
		t.Fatal("create response lacks unambiguous run-owned document ID; no replay or guessed cleanup")
	}
	manifest, _ := json.Marshal(map[string]string{"document_id": docID, "title": title, "parent_token": parent})
	if err := os.WriteFile(filepath.Join(evidenceDir, "run-owned-document.json"), manifest, 0600); err != nil {
		t.Error("cannot retain run-owned cleanup binding")
	}
	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cleanCancel()
		// Fresh metadata prevents cleanup of a guessed or unrelated resource.
		owned := call(cleanCtx, "cleanup-owner", clie2e.Request{Args: []string{"api", "GET", "/open-apis/docx/v1/documents/" + docID}, DefaultAs: "user"})
		if owned.ExitCode != 0 || !gjson.Get(owned.Stdout, "ok").Bool() || gjson.Get(owned.Stdout, "data.document.document_id").String() != docID || gjson.Get(owned.Stdout, "data.document.title").String() != title {
			t.Error("cleanup ownership not established; retained run-owned manifest; no delete attempted")
			return
		}
		deleted := call(cleanCtx, "cleanup-delete", clie2e.Request{Args: []string{"drive", "+delete", "--file-token", docID, "--type", "docx"}, DefaultAs: "user", Yes: true})
		if deleted.ExitCode != 0 || !gjson.Get(deleted.Stdout, "ok").Bool() || !gjson.Get(deleted.Stdout, "data.deleted").Bool() {
			t.Error("cleanup is not confirmed; private diagnostics retained; no delete replay")
			return
		}
		absent := call(cleanCtx, "cleanup-absence", clie2e.Request{Args: []string{"api", "GET", "/open-apis/docx/v1/documents/" + docID}, DefaultAs: "user"})
		if !nextDocsDeleted(absent) {
			t.Error("delete accepted but independent exact-ID absence not established; no delete replay")
			return
		}
		t.Log("run-owned document deletion independently confirmed; parent folder left intact")
	})
	requireOK("create", createResult)
	checkACL("document", docID, "docx")
	fetch := func(label string) string {
		return user(label, "docs", "+fetch", "--doc", docID, "--doc-format", "xml", "--detail", "full")
	}
	initialXML := fetch("fetch-initial")
	if !strings.Contains(gjson.Get(initialXML, "data.document.content").String(), "你好") || !strings.Contains(gjson.Get(initialXML, "data.document.content").String(), "<table") {
		t.Fatal("rich XML export lacks Unicode or native table")
	}
	native := func(label, phrase string) nextDocsNativeSnapshot {
		t.Helper()
		blocks := user(label, "api", "GET", "/open-apis/docx/v1/documents/"+docID+"/blocks", "--params", `{"page_size":500}`)
		snapshot, err := nextDocsInspectNative(blocks, phrase)
		if err != nil {
			t.Fatalf("%s native fidelity check failed: %v", label, err)
		}
		return snapshot
	}
	before := native("native-initial", "Old status — 你好")
	replacement := `New <a href="` + nextLiveDocsLink + `"><b>status</b></a>`
	if mode := os.Getenv("NEXT_DOCS_LIVE_EDIT_MODE"); mode == "plain" {
		// Reproduces the separately retained native loss-of-formatting observation.
		replacement = "New status"
	} else if mode != "" && mode != "xml" {
		t.Fatal("unsupported private edit mode")
	}
	user("precise-edit", "docs", "+update", "--doc", docID, "--command", "str_replace", "--pattern", "Old status", "--content", replacement, "--doc-format", "xml")
	afterXML := fetch("fetch-updated")
	after := native("native-updated", "New status — 你好")
	delete(before.blocks, before.target)
	delete(after.blocks, after.target)
	if !reflect.DeepEqual(before.blocks, after.blocks) {
		t.Fatal("precise edit changed an unrelated native block")
	}
	parsed := user("script-parse", "docs", "+script", "--command", "parse", "--doc", docID)
	if gjson.Get(parsed, "data.profile.word_count").Int() <= 0 || gjson.Get(parsed, "data.profile.block_count").Int() <= 0 {
		t.Fatal("script parse profile is incomplete")
	}
	// No-op is separately opt-in: overwrite can mutate if native exports differ.
	// The default private batch does not exercise that fallback or infer zero writes.
	if os.Getenv("NEXT_DOCS_LIVE_CHECK_NOOP") == "1" {
		revision := gjson.Get(afterXML, "data.document.revision_id")
		xml := gjson.Get(afterXML, "data.document.content")
		if !revision.Exists() || revision.Int() < 0 || !xml.Exists() {
			t.Fatal("no-op preflight requires complete native revision/content")
		}
		draft := filepath.Join(evidenceDir, "same-content.xml")
		if err := os.WriteFile(draft, []byte(xml.String()), 0600); err != nil {
			t.Fatal("cannot retain no-op input")
		}
		args := []string{"docs", "+update", "--doc", docID, "--command", "overwrite", "--doc-format", "xml", "--revision-id", revision.String(), "--content", "@" + draft}
		if refs := gjson.Get(afterXML, "data.document.reference_map"); refs.Exists() && refs.Raw != "null" {
			args = append(args, "--reference-map", refs.Raw)
		}
		noOp := user("same-content-noop", args...)
		if !gjson.Get(noOp, "data.unchanged").Bool() {
			t.Fatal("same-content operation did not report unchanged; no zero-write claim")
		}
	}
	t.Log("private Docs create/fetch/precise-edit/native bold-link-Unicode-table/script checks passed; broader section and disabled no-op remain unverified")
}

func nextDocsDeleted(result *clie2e.Result) bool {
	// Observed on a read of this run's exact docx ID after its completed delete.
	// Permission denied, empty metadata URLs and generic errors are not absence.
	return result != nil && result.ExitCode != 0 &&
		gjson.Get(result.Stderr, "error.type").String() == "api" &&
		gjson.Get(result.Stderr, "error.code").Type == gjson.Number &&
		gjson.Get(result.Stderr, "error.code").Int() == 1770003 &&
		gjson.Get(result.Stderr, "error.message").String() == "resource deleted"
}

type nextDocsNativeSnapshot struct {
	blocks map[string]any
	target string
}

func nextDocsInspectNative(raw, phrase string) (nextDocsNativeSnapshot, error) {
	out := nextDocsNativeSnapshot{blocks: map[string]any{}}
	items := gjson.Get(raw, "data.items")
	if !items.IsArray() || gjson.Get(raw, "data.has_more").Bool() {
		return out, fmt.Errorf("native block extent incomplete")
	}
	table, keep := false, false
	for _, block := range items.Array() {
		id := block.Get("block_id").String()
		if id == "" {
			return out, fmt.Errorf("native block ID missing")
		}
		if _, exists := out.blocks[id]; exists {
			return out, fmt.Errorf("duplicate native block ID")
		}
		var value any
		if err := json.Unmarshal([]byte(block.Raw), &value); err != nil {
			return out, err
		}
		out.blocks[id] = value
		if block.Get("block_type").Int() == 31 {
			table = true
		}
		text := ""
		type character struct {
			r    rune
			bold bool
			url  string
		}
		var chars []character
		for _, element := range block.Get("text.elements").Array() {
			run := element.Get("text_run")
			if !run.Exists() {
				continue
			}
			s := run.Get("content").String()
			text += s
			for _, r := range s {
				chars = append(chars, character{r, run.Get("text_element_style.bold").Bool(), run.Get("text_element_style.link.url").String()})
			}
		}
		if text == "Keep résumé 😀 and 001 unchanged." {
			keep = true
		}
		if text != phrase {
			continue
		}
		if out.target != "" {
			return out, fmt.Errorf("ambiguous target paragraph")
		}
		out.target = id
		for i, c := range chars {
			bold := i >= 4 && i < 10
			url := ""
			if bold {
				url = nextLiveDocsLink
			}
			if c.bold != bold || c.url != url {
				return out, fmt.Errorf("target native bold/link mismatch")
			}
		}
	}
	if out.target == "" || !table || !keep {
		return out, fmt.Errorf("target, native table or unrelated Unicode paragraph missing")
	}
	return out, nil
}

func TestDocsNextPrivateNativeVerifier(t *testing.T) {
	base := `{"data":{"has_more":false,"items":[{"block_id":"target","block_type":2,"text":{"elements":[{"text_run":{"content":"New "}},{"text_run":{"content":"status","text_element_style":{"bold":true,"link":{"url":"https://example.com/next-docs-private"}}}},{"text_run":{"content":" — 你好"}}]}},{"block_id":"keep","block_type":2,"text":{"elements":[{"text_run":{"content":"Keep résumé 😀 and 001 unchanged."}}]}},{"block_id":"table","block_type":31}]}}`
	if _, err := nextDocsInspectNative(base, "New status — 你好"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(base, `"bold":true`, `"bold":false`, 1),
		strings.Replace(base, nextLiveDocsLink, "https://example.com/wrong", 1),
		strings.Replace(base, "你好", "你坏", 1),
		strings.Replace(base, `"has_more":false`, `"has_more":true`, 1),
		strings.Replace(base, `"block_type":31`, `"block_type":2`, 1),
	} {
		if _, err := nextDocsInspectNative(bad, "New status — 你好"); err == nil {
			t.Fatal("invalid native fidelity accepted")
		}
	}
}

func TestDocsNextPrivateDeletedProof(t *testing.T) {
	deleted := &clie2e.Result{ExitCode: 1, Stderr: `{"ok":false,"error":{"type":"api","code":1770003,"message":"resource deleted"}}`}
	if !nextDocsDeleted(deleted) {
		t.Fatal("explicit deleted response was rejected")
	}
	for _, result := range []*clie2e.Result{
		nil,
		{ExitCode: 0, Stdout: `{"ok":true,"data":{"metas":[{"url":""}]}}`},
		{ExitCode: 1, Stderr: `{"ok":false,"error":{"type":"api","code":1770002,"message":"permission denied"}}`},
		{ExitCode: 1, Stderr: `{"ok":false,"error":{"type":"api","code":"1770003","message":"resource deleted"}}`},
	} {
		if nextDocsDeleted(result) {
			t.Fatal("ambiguous cleanup accepted as deletion")
		}
	}
}
