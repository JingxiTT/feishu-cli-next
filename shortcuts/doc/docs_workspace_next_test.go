// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/larksuite/cli/shortcuts/doc/internal/docworkspace"
)

type docsWorkspaceAccount struct{}

func (docsWorkspaceAccount) ResolveAccount(context.Context) (*credential.Account, error) {
	return &credential.Account{AppID: "workspace-app", Brand: core.BrandFeishu, UserOpenId: "user", TenantKey: "tenant", UserInfoVerified: true}, nil
}
func workspaceFactory(t *testing.T) (*cmdutil.Factory, *httpmock.Registry, *bytes.Buffer) {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	withDocsWorkingDir(t, t.TempDir())
	cfg := docsTestConfigWithAppID("workspace-app")
	cfg.UserOpenId = "user"
	f, out, _, reg := cmdutil.TestFactory(t, cfg)
	f.Credential = credential.NewCredentialProvider(nil, docsWorkspaceAccount{}, &docMediaScopedTokenResolver{scopes: "docx:document:readonly docx:document:write_only"}, nil)
	return f, reg, out
}
func workspaceBlocks() []json.RawMessage {
	return []json.RawMessage{
		json.RawMessage(`{"block_id":"doc1","block_type":1,"children":["p1","p2","image"],"page":{"elements":[]}}`),
		json.RawMessage(`{"block_id":"p1","block_type":2,"parent_id":"doc1","text":{"elements":[{"text_run":{"content":"Old 中文","text_element_style":{"bold":true,"link":{"url":"https://example.org"}}}},{"text_run":{"content":" rest"}}],"style":{"align":1}}}`),
		json.RawMessage(`{"block_id":"p2","block_type":2,"parent_id":"doc1","text":{"elements":[{"text_run":{"content":"Second"}}]}}`),
		json.RawMessage(`{"block_id":"image","block_type":27,"parent_id":"doc1","image":{"token":"opaque","future":9007199254740993}}`),
	}
}

type workspaceServer struct {
	metadataHook                       func(*workspaceServer)
	revision                           int64
	blocks                             []json.RawMessage
	metadataCalls, blockCalls, patches int
	changeReadAt                       int
	failPatchAt                        int
	mismatchReadback                   bool
}

func (s *workspaceServer) mount(t *testing.T, reg *httpmock.Registry) {
	t.Helper()
	envelope := func(data interface{}) interface{} { return map[string]interface{}{"code": 0, "data": data} }
	block := &httpmock.Stub{Method: "GET", URL: "/documents/doc1/blocks", Reusable: true, Optional: true}
	block.OnMatch = func(r *http.Request) {
		s.blockCalls++
		if r.URL.Query().Get("document_revision_id") != fmt.Sprint(s.revision) {
			t.Errorf("historical read used: %s at %d", r.URL.RawQuery, s.revision)
		}
		block.Body = envelope(map[string]interface{}{"items": s.blocks, "has_more": false})
	}
	reg.Register(block)
	metadata := &httpmock.Stub{Method: "GET", URL: "/documents/doc1", Reusable: true, Optional: true}
	metadata.OnMatch = func(*http.Request) {
		s.metadataCalls++
		if s.metadataHook != nil {
			s.metadataHook(s)
		}
		if s.changeReadAt == s.metadataCalls {
			s.revision++
		}
		metadata.Body = envelope(map[string]interface{}{"document": map[string]interface{}{"document_id": "doc1", "revision_id": s.revision}})
	}
	reg.Register(metadata)
	patch := &httpmock.Stub{Method: "PATCH", URL: "/documents/doc1/blocks/", Reusable: true, Optional: true}
	patch.OnMatch = func(r *http.Request) {
		s.patches++
		if s.failPatchAt == s.patches {
			patch.Body = map[string]interface{}{"code": 99991663, "msg": "synthetic denied"}
			return
		}
		if r.URL.Query().Get("document_revision_id") != fmt.Sprint(s.revision) {
			t.Error("patch base is not latest observed revision")
		}
		var body struct {
			Update struct {
				Elements []json.RawMessage `json:"elements"`
			} `json:"update_text_elements"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		// Assert the actual write body contains no fields beyond the planned text elements.
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		if len(fields) != 1 || body.Update.Elements == nil {
			t.Errorf("invalid patch body: %s", raw)
		}
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if !s.mismatchReadback {
			var err error
			s.blocks, err = docsWorkspaceEditedBlocks(s.blocks, docworkspace.Edit{BlockID: id, Elements: body.Update.Elements})
			if err != nil {
				t.Fatal(err)
			}
		}
		s.revision++
		patch.Body = envelope(map[string]interface{}{"document_revision_id": s.revision})
	}
	reg.Register(patch)
}
func workspaceRun(t *testing.T, s common.Shortcut, f *cmdutil.Factory, out *bytes.Buffer, args ...string) error {
	t.Helper()
	return mountAndRunDocs(t, s, append(args, "--as", "user"), f, out)
}
func workspaceSave(t *testing.T, f *cmdutil.Factory, blocks []json.RawMessage, edited string) []byte {
	t.Helper()
	binding, err := f.ResolveArtifactBinding(context.Background(), core.AsUser)
	if err != nil {
		t.Fatal(err)
	}
	saved := docsWorkspace{Schema: docsWorkspaceSchema, Binding: binding, DocumentID: "doc1", RevisionID: 3, Blocks: blocks}
	raw, _ := json.Marshal(saved)
	if err := os.WriteFile("note.md.feishu.json", raw, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("note.md", []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	return raw
}
func workspaceMarkdown(t *testing.T, blocks []json.RawMessage) string {
	t.Helper()
	s, e := docworkspace.Render(blocks)
	if e != nil {
		t.Fatal(e)
	}
	return s
}

func TestDocsWorkspacePullAndPushLifecycle(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	server := &workspaceServer{revision: 3, blocks: workspaceBlocks()}
	server.mount(t, reg)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "note.md"); err != nil {
		t.Fatal(err)
	}
	sidecar, err := os.ReadFile("note.md.feishu.json")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(sidecar, []byte("9007199254740993")) {
		t.Fatal("lost native integer precision")
	}
	markdown, err := os.ReadFile("note.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(markdown, []byte("<!-- block:p1 -->")) {
		t.Fatal("missing stable identity")
	}
	edited := strings.Replace(string(markdown), "Old 中文", "New 🐈", 1)
	if err := os.WriteFile("note.md", []byte(edited), 0600); err != nil {
		t.Fatal(err)
	}
	before := append([]json.RawMessage(nil), server.blocks...)
	if err := workspaceRun(t, DocsPush, f, out, "+push", "--file", "note.md"); err != nil {
		t.Fatalf("push: %v\n%s", err, out)
	}
	if server.patches != 1 || server.revision != 4 || !strings.Contains(out.String(), `"verified": true`) || !strings.Contains(out.String(), "New 🐈 rest") {
		t.Fatalf("wrong verified output: %s", out)
	}
	if !docsWorkspaceJSONEqual(before[0], server.blocks[0]) || !docsWorkspaceJSONEqual(before[2:], server.blocks[2:]) {
		t.Fatal("unrelated native blocks changed")
	}
	if !bytes.Contains(server.blocks[1], []byte(`"bold":true`)) || !bytes.Contains(server.blocks[1], []byte(`https://example.org`)) {
		t.Fatal("lost native marks")
	}
	after, _ := os.ReadFile("note.md.feishu.json")
	if !bytes.Equal(sidecar, after) {
		t.Fatal("sidecar advanced")
	}
	if err := workspaceRun(t, DocsPush, f, out, "+push", "--file", "note.md"); err == nil || server.patches != 1 {
		t.Fatalf("repeat push did not detect stale workspace: %v", err)
	}
}
func TestDocsWorkspacePushNoopConflictAndOfflineValidation(t *testing.T) {
	for _, mode := range []string{"noop", "revision-conflict", "content-conflict", "unsupported-last", "dry-run", "bad-binding", "missing-revision", "unknown-sidecar"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			blocks := workspaceBlocks()
			markdown := workspaceMarkdown(t, blocks)
			if mode != "noop" {
				markdown = strings.Replace(markdown, "Old 中文", "New 中文", 1)
			}
			if mode == "unsupported-last" {
				markdown = strings.Replace(markdown, "Second", "**Second**", 1)
			}
			raw := workspaceSave(t, f, blocks, markdown)
			if mode == "bad-binding" {
				raw = bytes.ReplaceAll(raw, []byte(`"tenant"`), []byte(`"other"`))
			}
			if mode == "missing-revision" {
				raw = bytes.ReplaceAll(raw, []byte(`"revision_id":3,`), nil)
			}
			if mode == "unknown-sidecar" {
				raw = append([]byte(`{"unknown":1,`), raw[1:]...)
			}
			if err := os.WriteFile("note.md.feishu.json", raw, 0600); err != nil {
				t.Fatal(err)
			}
			server := &workspaceServer{revision: 3, blocks: blocks}
			server.mount(t, reg)
			if mode == "revision-conflict" {
				server.revision = 4
			}
			if mode == "content-conflict" {
				server.blocks = append([]json.RawMessage(nil), blocks...)
				server.blocks[2] = bytes.ReplaceAll(blocks[2], []byte("Second"), []byte("Other"))
			}
			args := []string{"+push", "--file", "note.md"}
			if mode == "dry-run" {
				args = append(args, "--dry-run")
			}
			err := workspaceRun(t, DocsPush, f, out, args...)
			if mode == "noop" || mode == "dry-run" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("expected rejection")
			}
			if server.patches != 0 {
				t.Fatal("unexpected write")
			}
			if mode == "unsupported-last" || mode == "dry-run" || mode == "bad-binding" || mode == "missing-revision" || mode == "unknown-sidecar" {
				if server.metadataCalls+server.blockCalls != 0 {
					t.Fatal("local validation/dry-run performed native read")
				}
			}
		})
	}
}
func TestDocsWorkspacePushStopsAfterUnknownOrPartial(t *testing.T) {
	for _, mode := range []string{"denied-second", "conflict-second", "readback-mismatch"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			blocks := workspaceBlocks()
			markdown := workspaceMarkdown(t, blocks)
			markdown = strings.Replace(strings.Replace(markdown, "Old 中文", "New 中文", 1), "Second", "Changed", 1)
			saved := workspaceSave(t, f, blocks, markdown)
			server := &workspaceServer{revision: 3, blocks: blocks}
			server.mount(t, reg)
			switch mode {
			case "denied-second":
				server.failPatchAt = 2
			case "conflict-second":
				server.changeReadAt = 5
			case "readback-mismatch":
				server.mismatchReadback = true
			}
			err := workspaceRun(t, DocsPush, f, out, "+push", "--file", "note.md")
			if err == nil {
				t.Fatal("expected failure")
			}
			var outcome *errs.MutationOutcomeError
			if !errors.As(err, &outcome) {
				t.Fatalf("missing mutation outcome: %T %v", err, err)
			}
			if mode == "denied-second" && server.patches != 2 || mode != "denied-second" && server.patches != 1 {
				t.Fatalf("unexpected patch count: %d", server.patches)
			}
			if !strings.Contains(out.String(), `"verified": false`) {
				t.Fatalf("missing unverified output: %s", out)
			}
			if mode != "readback-mismatch" && !strings.Contains(out.String(), "New 中文 rest") {
				t.Fatalf("lost prior verified change: %s", out)
			}
			after, _ := os.ReadFile("note.md.feishu.json")
			if !bytes.Equal(saved, after) {
				t.Fatal("failure advanced sidecar")
			}
		})
	}
}
func TestDocsWorkspacePullRejectsIncompleteSnapshots(t *testing.T) {
	for _, mode := range []string{"empty", "missing-pagination", "duplicate", "cursor-loop", "missing-revision", "drift"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			if mode == "drift" {
				s := &workspaceServer{revision: 3, blocks: workspaceBlocks(), metadataHook: func(s *workspaceServer) {
					if s.metadataCalls%2 == 0 {
						s.revision++
					}
				}}
				s.mount(t, reg)
			} else {
				data := map[string]interface{}{"items": workspaceBlocks(), "has_more": false}
				switch mode {
				case "empty":
					data["items"] = []interface{}{}
				case "missing-pagination":
					delete(data, "has_more")
				case "duplicate":
					data["items"] = []json.RawMessage{workspaceBlocks()[1], workspaceBlocks()[1]}
				case "cursor-loop":
					data["has_more"] = true
					data["page_token"] = "same"
				}
				reg.Register(&httpmock.Stub{Method: "GET", URL: "/documents/doc1/blocks", Reusable: true, Optional: mode == "missing-revision", Body: map[string]interface{}{"code": 0, "data": data}})
				document := map[string]interface{}{"document_id": "doc1", "revision_id": 3}
				if mode == "missing-revision" {
					delete(document, "revision_id")
				}
				reg.Register(&httpmock.Stub{Method: "GET", URL: "/documents/doc1", Reusable: true, Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"document": document}}})
			}
			if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "note.md"); err == nil {
				t.Fatal("expected failure")
			}
			for _, name := range []string{"note.md", "note.md.feishu.json"} {
				if _, err := os.Stat(name); !os.IsNotExist(err) {
					t.Fatalf("published incomplete snapshot: %s", name)
				}
			}
		})
	}
}
func TestDocsWorkspacePullOfflineAndExclusive(t *testing.T) {
	f, _, out := workspaceFactory(t)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "https://example.feishu.cn/wiki/wiki1", "--output", "note.md", "--dry-run"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "node_by_token") || !strings.Contains(out.String(), "binding") {
		t.Fatalf("missing planned conditional effects: %s", out)
	}
	if _, err := os.Stat("note.md"); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote file")
	}
	if err := os.WriteFile("note.md.feishu.json", []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "note.md"); err == nil {
		t.Fatal("overwrote sidecar")
	}
	raw, _ := os.ReadFile("note.md.feishu.json")
	if string(raw) != "retain" {
		t.Fatal("existing sidecar altered")
	}
}

func TestDocsWorkspacePullPagesUseCurrentRevision(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/documents/doc1/blocks", OnMatch: func(r *http.Request) {
		if r.URL.Query().Get("document_revision_id") != "3" || r.URL.Query().Get("page_token") != "" {
			t.Errorf("wrong first page: %s", r.URL.RawQuery)
		}
	}, Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": workspaceBlocks()[:2], "has_more": true, "page_token": "next"}}})
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/documents/doc1/blocks", OnMatch: func(r *http.Request) {
		if r.URL.Query().Get("document_revision_id") != "3" || r.URL.Query().Get("page_token") != "next" {
			t.Errorf("wrong second page: %s", r.URL.RawQuery)
		}
	}, Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": workspaceBlocks()[2:], "has_more": false}}})
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/documents/doc1", Reusable: true, Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"document": map[string]interface{}{"document_id": "doc1", "revision_id": 3}}}})
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"block:p1", "block:p2", "block:image", "markdown"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("missing page content %s", want)
		}
	}
}

func TestDocsWorkspacePullRejectsExpandedMarkdownBeforeSaving(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	// Each native '#' requires two Markdown bytes, while the native payload fits.
	content := strings.Repeat("#", docsWorkspaceLimit/2)
	raw, _ := json.Marshal(map[string]interface{}{"block_id": "p1", "block_type": 2, "text": map[string]interface{}{"elements": []interface{}{map[string]interface{}{"text_run": map[string]interface{}{"content": content}}}}})
	server := &workspaceServer{revision: 3, blocks: []json.RawMessage{raw}}
	server.mount(t, reg)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "note.md"); err == nil || !strings.Contains(err.Error(), "size limits") {
		t.Fatalf("expected expanded file limit, got %v", err)
	}
	for _, name := range []string{"note.md", "note.md.feishu.json"} {
		if _, err := os.Stat(name); !os.IsNotExist(err) {
			t.Fatalf("published unreadable workspace: %s", name)
		}
	}
}

func TestDocsWorkspaceExpectedEditsUseNativeBlockType(t *testing.T) {
	blocks := []json.RawMessage{json.RawMessage(`{"block_id":"h","block_type":3,"text":{"elements":[]},"heading1":{"elements":[{"text_run":{"content":"old"}}]}}`)}
	elements := []json.RawMessage{json.RawMessage(`{"text_run":{"content":"new"}}`)}
	result, err := docsWorkspaceEditedBlocks(blocks, docworkspace.Edit{BlockID: "h", Elements: elements})
	if err != nil {
		t.Fatal(err)
	}
	var b map[string]json.RawMessage
	_ = json.Unmarshal(result[0], &b)
	if !bytes.Contains(b["heading1"], []byte("new")) || !bytes.Equal(b["text"], []byte(`{"elements":[]}`)) {
		t.Fatalf("wrong native field changed: %s", result[0])
	}
	blocks[0] = json.RawMessage(`{"block_id":"h","block_type":12,"text":{"elements":[]}}`)
	if _, err := docsWorkspaceEditedBlocks(blocks, docworkspace.Edit{BlockID: "h", Elements: elements}); err == nil {
		t.Fatal("unsupported block type accepted")
	}
}

func TestDocsWorkspacePullRejectsFinalSidecarEnvelopeBeforeSaving(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	binding, err := f.ResolveArtifactBinding(context.Background(), core.AsUser)
	if err != nil {
		t.Fatal(err)
	}
	makeBlock := func(content string) json.RawMessage {
		raw, _ := json.Marshal(map[string]interface{}{"block_id": "p1", "block_type": 2, "text": map[string]interface{}{"elements": []interface{}{map[string]interface{}{"text_run": map[string]interface{}{"content": content}}}}})
		return raw
	}
	saved := docsWorkspace{Schema: docsWorkspaceSchema, Binding: binding, DocumentID: "doc1", RevisionID: 3, Blocks: []json.RawMessage{makeBlock("")}}
	envelope, _ := json.Marshal(saved)
	block := makeBlock(strings.Repeat("a", docsWorkspaceLimit-len(envelope)+1))
	response, _ := json.Marshal(map[string]interface{}{"items": []json.RawMessage{block}, "has_more": false})
	if len(response) > docsWorkspaceLimit {
		t.Fatal("fixture must fit the API response boundary")
	}
	server := &workspaceServer{revision: 3, blocks: []json.RawMessage{block}}
	server.mount(t, reg)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "note.md"); err == nil || !strings.Contains(err.Error(), "size limits") {
		t.Fatalf("expected final sidecar size rejection, got %v", err)
	}
	for _, name := range []string{"note.md", "note.md.feishu.json"} {
		if _, err := os.Stat(name); !os.IsNotExist(err) {
			t.Fatalf("published oversized sidecar: %s", name)
		}
	}
}

func TestDocsWorkspaceSnapshotReconciliationNeverReplaysMutation(t *testing.T) {
	for _, mode := range []string{"transient", "persistent", "changed-content", "malformed-response"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			blocks := workspaceBlocks()
			workspaceSave(t, f, blocks, strings.Replace(workspaceMarkdown(t, blocks), "Old 中文", "New 中文", 1))
			server := &workspaceServer{revision: 3, blocks: blocks}
			server.metadataHook = func(s *workspaceServer) {
				if s.patches == 0 {
					return
				}
				switch mode {
				case "transient", "changed-content":
					if s.metadataCalls == 4 {
						s.revision++
						if mode == "changed-content" {
							s.blocks[2] = bytes.ReplaceAll(s.blocks[2], []byte("Second"), []byte("Concurrent"))
						}
					}
				case "persistent":
					if s.metadataCalls%2 == 0 {
						s.revision++
					}
				case "malformed-response":
					s.blocks = nil
				}
			}
			server.mount(t, reg)
			err := workspaceRun(t, DocsPush, f, out, "+push", "--file", "note.md")
			if server.patches != 1 {
				t.Fatalf("mutation repeated: %d", server.patches)
			}
			if mode == "transient" {
				if err != nil || !strings.Contains(out.String(), `"verified": true`) || server.metadataCalls != 6 || server.blockCalls != 3 {
					t.Fatalf("stable reconciliation failed: %v; reads %d/%d; %s", err, server.metadataCalls, server.blockCalls, out)
				}
			} else {
				var outcome *errs.MutationOutcomeError
				if !errors.As(err, &outcome) || outcome.Outcome != "unknown" {
					t.Fatalf("read uncertainty lost: %T %v", err, err)
				}
				if mode == "persistent" {
					if server.metadataCalls != 8 || server.blockCalls != 4 {
						t.Fatalf("read attempts unbounded: %d/%d", server.metadataCalls, server.blockCalls)
					}
					var conflict *errs.ValidationError
					if !errors.As(err, &conflict) || !strings.Contains(conflict.Hint, "revision 6 before reading and 7 after reading") {
						t.Fatalf("missing final observed revisions: %v", err)
					}
				}
				if mode == "changed-content" {
					var conflict *errs.ValidationError
					if !errors.As(err, &conflict) || !strings.Contains(conflict.Error(), "does not match") {
						t.Fatalf("accepted changed content: %v", err)
					}
				}
				if mode == "malformed-response" && (server.metadataCalls != 3 || server.blockCalls != 2) {
					t.Fatalf("invalid response retried: %d/%d", server.metadataCalls, server.blockCalls)
				}
			}
		})
	}
}

func TestDocsWorkspacePhysicalNewlineNoopAndDryRun(t *testing.T) {
	for _, mode := range []string{"noop", "dry-run"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			blocks := workspaceBlocks()
			markdown := workspaceMarkdown(t, blocks)
			if mode == "dry-run" {
				markdown = strings.Replace(markdown, "Old 中文", "New 中文", 1)
			}
			markdown = "\ufeff" + strings.ReplaceAll(strings.TrimRight(markdown, "\n"), "\n", "\r\n")
			workspaceSave(t, f, blocks, markdown)
			server := &workspaceServer{revision: 3, blocks: blocks}
			server.mount(t, reg)
			args := []string{"+push", "--file", "note.md"}
			if mode == "dry-run" {
				args = append(args, "--dry-run")
			}
			if err := workspaceRun(t, DocsPush, f, out, args...); err != nil {
				t.Fatal(err)
			}
			if server.patches != 0 {
				t.Fatal("physical formatting dispatched a write")
			}
			if mode == "dry-run" && server.metadataCalls+server.blockCalls != 0 {
				t.Fatal("dry-run performed native reads")
			}
		})
	}
}

func TestDocsWorkspaceAtPathRecoveryPreservesLiteralFiles(t *testing.T) {
	for _, mode := range []string{"missing-prefix", "literal-prefix"} {
		t.Run(mode, func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			blocks := workspaceBlocks()
			workspaceSave(t, f, blocks, workspaceMarkdown(t, blocks))
			server := &workspaceServer{revision: 3, blocks: blocks}
			server.mount(t, reg)
			file := "@./note.md"
			if mode == "literal-prefix" {
				for _, suffix := range []string{"", ".feishu.json"} {
					if err := os.Rename("note.md"+suffix, "@note.md"+suffix); err != nil {
						t.Fatal(err)
					}
				}
				file = "@note.md"
			}
			err := workspaceRun(t, DocsPush, f, out, "+push", "--file", file, "--dry-run")
			if mode == "literal-prefix" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				var input *errs.ValidationError
				if !errors.As(err, &input) || input.Subtype != errs.SubtypeInvalidArgument || !errors.Is(err, os.ErrNotExist) || !strings.Contains(input.Hint, "--file ./note.md without @") {
					t.Fatalf("missing actionable typed recovery: %v", err)
				}
			}
			if server.patches+server.metadataCalls+server.blockCalls != 0 {
				t.Fatal("local path/dry-run performed native requests")
			}
		})
	}
}
