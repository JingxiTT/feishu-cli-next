// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docs_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const documentID = "docxSyntheticFixture"
const richXML = `<h1 block-id="heading">你好 😀</h1><p><b>Bold</b> &amp; <a href="https://example.com">link</a></p><table><tr><td><p>001</p></td></tr></table><img token="syntheticImage"/>`

func run(f *nextfixture.Fixture, args ...string) nextfixture.Result {
	return f.Run(append([]string{"docs"}, append(args, "--as", "user")...)...)
}
func success(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("CLI failed: %+v", r)
	}
	var value map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &value); err != nil {
		t.Fatalf("invalid JSON: %v: %+v", err, r)
	}
	return value
}
func body(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var value map[string]any
	d := json.NewDecoder(r.Body)
	d.UseNumber()
	if err := d.Decode(&value); err != nil {
		t.Errorf("decode request: %v", err)
	}
	return value
}
func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func require(t *testing.T, got any, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestDocsCreateFetchRichArtifact(t *testing.T) {
	var requests atomic.Int32
	references := `{"custom":{"integer":9007199254740993,"decimal":1.234567890123456789,"empty":null,"id":"001"}}`
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		b := body(t, r)
		require(t, r.Method, "POST")
		switch r.URL.Path {
		case "/open-apis/docs_ai/v1/documents":
			require(t, b["format"], "xml")
			require(t, b["content"], "<title>A &amp; B</title>\n"+richXML)
			ref := b["reference_map"].(map[string]any)["custom"].(map[string]any)
			require(t, ref["integer"], json.Number("9007199254740993"))
			require(t, ref["decimal"], json.Number("1.234567890123456789"))
			reply(w, map[string]any{"document": map[string]any{"document_id": documentID}})
		case "/open-apis/docs_ai/v1/documents/" + documentID + "/fetch":
			require(t, b["revision_id"], json.Number("0"))
			require(t, b["export_option"].(map[string]any)["export_style_attrs"], true)
			reply(w, map[string]any{"document": map[string]any{"document_id": documentID, "content": richXML, "reference_map": map[string]any{"custom": map[string]any{"id": "001", "empty": nil}}}})
		default:
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	if err := os.WriteFile(filepath.Join(f.WorkDir, "content.xml"), []byte(richXML), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.WorkDir, "references.json"), []byte(references), 0600); err != nil {
		t.Fatal(err)
	}
	created := success(t, run(f, "+create", "--title", "A & B", "--content", "@content.xml", "--reference-map", "@references.json"))
	require(t, created["data"].(map[string]any)["document"].(map[string]any)["document_id"], documentID)
	artifactPath := filepath.Join(f.WorkDir, "document-result.json")
	fetched := success(t, run(f, "+fetch", "--doc", documentID, "--detail", "full", "--revision-id", "0", "--save-result", artifactPath, "--format", "json"))
	if fetched["artifact"] == nil {
		t.Fatalf("missing artifact: %v", fetched)
	}
	saved, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	var artifact struct {
		Result struct {
			Data struct {
				Document struct {
					Content   string         `json:"content"`
					Reference map[string]any `json:"reference_map"`
				} `json:"document"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(saved, &artifact); err != nil {
		t.Fatal(err)
	}
	require(t, artifact.Result.Data.Document.Content, richXML)
	if artifact.Result.Data.Document.Reference["custom"] == nil {
		t.Fatalf("lost reference map: %s", saved)
	}
	require(t, requests.Load(), int32(2))
}

func TestDocsEveryUpdateOperation(t *testing.T) {
	cases := []struct {
		name, wire string
		flags      []string
		key        string
		value      any
	}{
		{"str_replace", "str_replace", []string{"--pattern", "old", "--content", "new"}, "pattern", "old"},
		{"block_delete", "block_delete", []string{"--start-block-id", "0", "--end-block-id", "-1"}, "start_block_id", "0"},
		{"block_insert_after", "block_insert_after", []string{"--block-id", "anchor", "--content", "<p>new</p>"}, "block_id", "anchor"},
		{"block_copy_insert_after", "block_copy_insert_after", []string{"--block-id", "anchor", "--src-block-ids", "first,second"}, "src_block_ids", "first,second"},
		{"block_replace", "block_replace", []string{"--block-id", "first,second", "--content", "<p>new</p>"}, "block_id", "first,second"},
		{"block_move_after", "block_move_after", []string{"--block-id", "anchor", "--src-block-ids", "first,second"}, "src_block_ids", "first,second"},
		{"overwrite", "overwrite", []string{"--content", "<p>new</p>"}, "content", "<p>new</p>"},
		{"append", "block_insert_after", []string{"--content", "<p>new</p>"}, "block_id", "-1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/fetch") {
					reply(w, map[string]any{"document": map[string]any{"content": "<p>old</p>"}})
					return
				}
				writes.Add(1)
				require(t, r.Method, "PUT")
				require(t, r.URL.Path, "/open-apis/docs_ai/v1/documents/"+documentID)
				b := body(t, r)
				require(t, b["command"], tc.wire)
				require(t, b[tc.key], tc.value)
				require(t, b["revision_id"], json.Number("0"))
				reply(w, map[string]any{"document_id": documentID, "revision_id": 1})
			}))
			args := append([]string{"+update", "--doc", documentID, "--command", tc.name, "--revision-id", "0"}, tc.flags...)
			success(t, run(f, args...))
			require(t, writes.Load(), int32(1))
		})
	}
}

func TestDocsBoundedReadScopes(t *testing.T) {
	for _, scope := range []string{"outline", "range", "section", "keyword"} {
		t.Run(scope, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require(t, r.Method, "POST")
				require(t, r.URL.Path, "/open-apis/docs_ai/v1/documents/"+documentID+"/fetch")
				b := body(t, r)
				ro := b["read_option"].(map[string]any)
				require(t, ro["read_mode"], scope)
				require(t, ro["max_depth"], "2")
				require(t, b["lang"], "zh-CN")
				if scope != "outline" {
					require(t, ro["context_before"], "1")
					require(t, ro["context_after"], "2")
				}
				if scope == "keyword" {
					require(t, ro["keyword"], "中文|error")
				}
				if scope == "range" || scope == "section" {
					require(t, ro["start_block_id"], "anchor")
				}
				reply(w, map[string]any{"document": map[string]any{"content": "<h1>bounded</h1>"}, "has_more": true})
			}))
			args := []string{"+fetch", "--doc", documentID, "--scope", scope, "--max-depth", "2", "--lang", "zh-CN"}
			if scope == "keyword" {
				args = append(args, "--keyword", "中文|error")
			}
			if scope == "range" || scope == "section" {
				args = append(args, "--start-block-id", "anchor")
			}
			if scope != "outline" {
				args = append(args, "--context-before", "1", "--context-after", "2")
			}
			r := run(f, args...)
			success(t, r)
			if strings.Contains(r.Stderr, `"mutation"`) {
				t.Fatalf("read journaled as mutation: %s", r.Stderr)
			}
		})
	}
}

func TestDocsSearchPaginationAndHistory(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/search/v2/doc_wiki/search":
			b := body(t, r)
			require(t, b["page_size"], json.Number("20"))
			require(t, b["page_token"], "page-one")
			if b["wiki_filter"] != nil {
				t.Error("folder scope leaked into wiki search")
			}
			reply(w, map[string]any{"res_units": []any{map[string]any{"title_highlighted": "<h>Title</h>", "result_meta": map[string]any{"create_time": "1700000000"}}}, "total": 2, "has_more": true, "page_token": "page-two"})
		case "/open-apis/docs_ai/v1/documents/" + documentID + "/histories":
			require(t, r.Method, "GET")
			require(t, r.URL.Query().Get("page_size"), "2")
			require(t, r.URL.Query().Get("page_token"), "history-page")
			reply(w, map[string]any{"items": []any{map[string]any{"history_version_id": "42"}}, "has_more": true, "page_token": "next-history"})
		case "/open-apis/docs_ai/v1/documents/" + documentID + "/history/revert":
			writes.Add(1)
			b := body(t, r)
			require(t, b["history_version_id"], "42")
			require(t, b["wait_timeout_ms"], json.Number("0"))
			reply(w, map[string]any{"task_id": "synthetic-task", "status": "pending"})
		case "/open-apis/docs_ai/v1/documents/" + documentID + "/history/revert_status":
			require(t, r.Method, "GET")
			require(t, r.URL.Query().Get("task_id"), "synthetic-task")
			reply(w, map[string]any{"task_id": "synthetic-task", "status": "success"})
		default:
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	search := success(t, run(f, "+search", "--query", "Title", "--page-size", "99", "--page-token", "page-one", "--filter", `{"folder_tokens":["syntheticFolder"]}`))
	require(t, search["data"].(map[string]any)["page_token"], "page-two")
	require(t, search["data"].(map[string]any)["has_more"], true)
	success(t, run(f, "+history-list", "--doc", documentID, "--page-size", "2", "--page-token", "history-page"))
	revert := run(f, "+history-revert", "--doc", documentID, "--history-version-id", "42", "--wait-timeout-ms", "0")
	if revert.ExitCode != 22 {
		t.Fatalf("async revert must remain unknown: %+v", revert)
	}
	success(t, run(f, "+history-revert-status", "--doc", documentID, "--task-id", "synthetic-task"))
	require(t, writes.Load(), int32(1))
}

func TestDocsLocalScriptDraftAndParse(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	decision := `{"audience":"reader","reader_task":"understand","genre_contract":null,"adapter":null,"presentation_mode":"normal","visual_plan":{"reason":"plain text is sufficient","blocks":[]}}`
	draft := success(t, run(f, "+script", "--command", "init-draft", "--presentation-decision", decision))
	path := draft["data"].(map[string]any)["draft_path"].(string)
	if !filepath.IsAbs(path) {
		path = filepath.Join(f.WorkDir, path)
	}
	if err := os.WriteFile(path, []byte(`<h1>Fixture</h1><p>Hello 世界</p>`), 0600); err != nil {
		t.Fatal(err)
	}
	relative, err := filepath.Rel(f.WorkDir, path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := success(t, run(f, "+script", "--command", "parse", "--content", "@"+relative))
	data := parsed["data"].(map[string]any)
	require(t, data["assessment"].(map[string]any)["status"], "passed")
	if data["profile"] == nil {
		t.Fatalf("missing profile: %v", data)
	}
}

func TestDocsMediaAndResourcesDryRun(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	if err := os.WriteFile(filepath.Join(f.WorkDir, "fixture.png"), []byte("synthetic image bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"media-insert", []string{"--doc", documentID, "--file", "fixture.png", "--width", "100", "--height", "50", "--caption", "图片"}, []string{"POST", "PATCH", "/blocks/batch_update", "upload_all"}},
		{"media-upload", []string{"--file", "fixture.png", "--parent-type", "docx_image", "--parent-node", "syntheticBlock", "--doc-id", documentID}, []string{"POST", "upload_all", "docx_image"}},
		{"media-preview", []string{"--token", "syntheticMedia", "--output", "preview.png"}, []string{"GET", "preview_download"}},
		{"media-download", []string{"--token", "syntheticMedia", "--output", "download.png"}, []string{"GET", "/download"}},
		{"media-download", []string{"--token", "syntheticBoard", "--output", "board.png", "--type", "whiteboard"}, []string{"GET", "download_as_image"}},
		{"resource-download", []string{"--doc", documentID, "--output", "cover.png"}, []string{"GET", "cover_token"}},
		{"resource-update", []string{"--doc", documentID, "--file", "fixture.png", "--offset-ratio-x", "0.25"}, []string{"POST", "PATCH", "update_cover", "0.25"}},
		{"resource-delete", []string{"--doc", documentID}, []string{"GET", "PATCH", "update_cover"}},
		// Registered under Docs; implementation and full lifecycle are owned by Whiteboards.
		{"whiteboard-update", []string{"--whiteboard-token", "syntheticBoard", "--source", "graph TD; A-->B", "--input_format", "mermaid"}, []string{"POST", "/nodes/plantuml", "graph TD; A--"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := run(f, append([]string{"+" + tc.name, "--dry-run"}, tc.args...)...)
			success(t, r)
			for _, want := range tc.want {
				if !strings.Contains(r.Stdout, want) {
					t.Errorf("missing %q in %s", want, r.Stdout)
				}
			}
		})
	}
	for _, name := range []string{"preview.png", "download.png", "board.png", "cover.png"} {
		if _, err := os.Stat(filepath.Join(f.WorkDir, name)); !os.IsNotExist(err) {
			t.Errorf("dry run created %s: %v", name, err)
		}
	}
}

func TestDocsInvalidInputsBeforeBackend(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	cases := [][]string{
		{"+create", "--title", " "},
		{"+create", "--title", "fixture", "--parent-token", "folder", "--parent-position", "my_library"},
		{"+create", "--content", "<p>text</p>", "--reference-map", "[]"},
		{"+fetch", "--doc", documentID, "--scope", "section"},
		{"+fetch", "--doc", documentID, "--scope", "keyword"},
		{"+fetch", "--doc", documentID, "--context-before", "-1"},
		{"+fetch", "--doc", documentID, "--scope", "full", "--max-depth", "2"},
		{"+search", "--filter", "[]"},
		{"+search", "--filter", `{"folder_tokens":["folder"],"space_ids":["space"]}`},
		{"+update", "--doc", documentID, "--command", "block_replace", "--content", "<p>x</p>", "--start-block-id", "a"},
		{"+update", "--doc", documentID, "--command", "block_move_after", "--block-id", "a"},
		{"+history-list", "--doc", documentID, "--page-size", "21"},
		{"+history-revert", "--doc", documentID, "--history-version-id", "0"},
		{"+history-revert-status", "--doc", documentID, "--task-id", " "},
		{"+script", "--command", "parse", "--doc", documentID, "--content", "<p>x</p>"},
		{"+media-insert", "--doc", documentID},
		{"+resource-update", "--doc", documentID, "--file", "missing.png", "--url", "https://example.com/image.png"},
		{"+resource-delete", "--doc", documentID, "--type", "unsupported"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			r := run(f, append(args, "--dry-run")...)
			if r.ExitCode != 2 || r.Stdout != "" {
				t.Fatalf("invalid input accepted: %+v", r)
			}
			var e struct {
				Error struct {
					Type string `json:"type"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(r.Stderr), &e); err != nil {
				t.Fatalf("invalid failure envelope: %+v", r)
			}
			require(t, e.Error.Type, "validation")
		})
	}
}

func TestDocsUnknownMutationReceiptAndReadOnlyRecovery(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/fetch") {
			reply(w, map[string]any{"document": map[string]any{"content": "<p>possibly applied</p>"}})
			return
		}
		writes.Add(1)
		require(t, r.Method, "PUT")
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}))
	r := run(f, "+update", "--doc", documentID, "--command", "append", "--content", "<p>possibly applied</p>")
	if r.ExitCode != 22 {
		t.Fatalf("lost response not unknown: %+v", r)
	}
	var id string
	d := json.NewDecoder(strings.NewReader(r.Stderr))
	for {
		var row struct {
			Mutation struct {
				Outcome  string   `json:"outcome"`
				Receipts []string `json:"receipts"`
			} `json:"mutation"`
		}
		err := d.Decode(&row)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if len(row.Mutation.Receipts) > 0 {
			require(t, row.Mutation.Outcome, "unknown")
			id = row.Mutation.Receipts[0]
		}
	}
	if id == "" {
		t.Fatalf("missing receipt: %+v", r)
	}
	inspect := f.Run("receipts", "inspect", id)
	success(t, inspect)
	if !strings.Contains(inspect.Stdout, `"unknown"`) {
		t.Fatalf("unknown evidence lost: %+v", inspect)
	}
	success(t, run(f, "+fetch", "--doc", documentID))
	success(t, f.Run("receipts", "cleanup", id))
	require(t, writes.Load(), int32(1))
}

// This named scenario is independently selectable. Environment opt-ins never bypass
// the independently verified current-account/private-creation prerequisite.
func TestDocsLivePrivateLifecycle(t *testing.T) {
	t.Skip("BLOCKED: no independently verified current-account and tenant-bound fixture proves private creation visibility before writing. Required live flow: private create; rich fetch/update and no-op; script/search; media/cover lifecycle; history revert/status; receipt recovery; delete only run-owned resources. No legacy live helpers or personal credentials are used. Other-recipient actions require independently resolved Ying Wu identity.")
}

func TestDocsCoverClearNoop(t *testing.T) {
	var reads atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		require(t, r.Method, "GET")
		require(t, r.URL.Path, "/open-apis/docx/v1/documents/"+documentID)
		reply(w, map[string]any{"document": map[string]any{"document_id": documentID}})
	}))
	r := run(f, "+resource-delete", "--doc", documentID)
	result := success(t, r)
	require(t, result["data"].(map[string]any)["already_empty"], true)
	require(t, result["data"].(map[string]any)["deleted"], false)
	require(t, reads.Load(), int32(1))
	if strings.Contains(r.Stderr, `"mutation"`) {
		t.Fatalf("no-op emitted mutation evidence: %s", r.Stderr)
	}
}

func TestDocsServiceFailureKeepsPartialResult(t *testing.T) {
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				reply(w, map[string]any{"result": "failed", "warnings": []string{"synthetic block could not be replaced"}})
			}))
			args := []string{"+" + operation, "--content", "<p>new</p>"}
			if operation == "update" {
				args = append(args, "--doc", documentID, "--command", "append")
			}
			r := run(f, args...)
			if r.ExitCode != 22 {
				t.Fatalf("partial service result must remain unknown: %+v", r)
			}
			var envelope struct {
				OK   bool `json:"ok"`
				Data struct {
					Result   string   `json:"result"`
					Warnings []string `json:"warnings"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(r.Stdout), &envelope); err != nil {
				t.Fatal(err)
			}
			require(t, envelope.OK, false)
			require(t, envelope.Data.Result, "failed")
			if len(envelope.Data.Warnings) != 1 {
				t.Fatalf("warnings lost: %s", r.Stdout)
			}
			require(t, writes.Load(), int32(1))
		})
	}
}

func TestDocsDownloadArtifactsAndNoClobber(t *testing.T) {
	payload := "synthetic\x00image\xffbytes"
	cases := []struct {
		name string
		args []string
		path string
	}{
		{"preview", []string{"+media-preview", "--token", "syntheticMedia"}, "/open-apis/drive/v1/medias/syntheticMedia/preview_download"},
		{"media", []string{"+media-download", "--token", "syntheticMedia"}, "/open-apis/drive/v1/medias/syntheticMedia/download"},
		{"board", []string{"+media-download", "--token", "syntheticBoard", "--type", "whiteboard"}, "/open-apis/board/v1/whiteboards/syntheticBoard/download_as_image"},
		{"cover", []string{"+resource-download", "--doc", documentID}, "/open-apis/drive/v1/medias/syntheticCover/download"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require(t, r.Method, "GET")
				switch r.URL.Path {
				case tc.path:
					w.Header().Set("Content-Type", "image/png")
					_, _ = io.WriteString(w, payload)
				case "/open-apis/docx/v1/documents/" + documentID:
					reply(w, map[string]any{"document": map[string]any{"cover": map[string]any{"token": "syntheticCover", "offset_ratio_x": 0.25}}})
				case "/open-apis/drive/v1/permissions/syntheticMedia/members/auth":
					reply(w, map[string]any{"auth_result": true})
				default:
					t.Errorf("unexpected download request %s", r.URL)
					http.Error(w, "unexpected", 500)
				}
			}))
			args := append(tc.args, "--output", "artifact.png")
			result := success(t, run(f, args...))
			require(t, result["data"].(map[string]any)["size_bytes"], float64(len(payload)))
			path := filepath.Join(f.WorkDir, "artifact.png")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			require(t, string(data), payload)
			if err := os.WriteFile(path, []byte("preserve existing"), 0600); err != nil {
				t.Fatal(err)
			}
			refused := run(f, args...)
			if refused.ExitCode != 2 {
				t.Fatalf("existing artifact overwritten: %+v", refused)
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			require(t, string(data), "preserve existing")
			success(t, run(f, append(args, "--overwrite")...))
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			require(t, string(data), payload)
		})
	}
}

func TestDocsUnchangedXMLOverwriteNoop(t *testing.T) {
	var reads atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		require(t, r.Method, "POST")
		require(t, r.URL.Path, "/open-apis/docs_ai/v1/documents/"+documentID+"/fetch")
		b := body(t, r)
		require(t, b["format"], "xml")
		require(t, b["export_option"].(map[string]any)["export_style_attrs"], true)
		reply(w, map[string]any{"document": map[string]any{"document_id": documentID, "content": richXML, "revision_id": 0}})
	}))
	r := run(f, "+update", "--doc", documentID, "--command", "overwrite", "--content", richXML, "--revision-id", "0")
	result := success(t, r)
	require(t, result["data"].(map[string]any)["unchanged"], true)
	require(t, result["data"].(map[string]any)["result"], "unchanged")
	require(t, reads.Load(), int32(1))
	if strings.Contains(r.Stderr, `"mutation"`) {
		t.Fatalf("no-op emitted mutation evidence: %s", r.Stderr)
	}
	files, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("no-op persisted mutation receipts: %v", files)
	}
}

func TestDocsCreatePartialCurrentUserGrantReceipt(t *testing.T) {
	var creates, grants atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require(t, r.Method, "POST")
		switch r.URL.Path {
		case "/open-apis/docs_ai/v1/documents":
			creates.Add(1)
			reply(w, map[string]any{"document": map[string]any{"document_id": documentID}})
		case "/open-apis/drive/v1/permissions/" + documentID + "/members":
			grants.Add(1)
			b := body(t, r)
			require(t, b["member_id"], "ou_foundation_fixture")
			require(t, r.URL.Query().Get("need_notification"), "false")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic grant rejected"}`)
		default:
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	r := f.Run("docs", "+create", "--title", "synthetic private fixture", "--as", "bot")
	if r.ExitCode != 21 {
		t.Fatalf("mixed outcome not partial: %+v", r)
	}
	var result struct {
		OK       bool `json:"ok"`
		Mutation struct {
			Outcome  string   `json:"outcome"`
			Receipts []string `json:"receipts"`
		} `json:"mutation"`
		Data struct {
			Document struct {
				ID string `json:"document_id"`
			} `json:"document"`
			Grant struct {
				Status string `json:"status"`
			} `json:"permission_grant"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(r.Stdout), &result); err != nil {
		t.Fatal(err)
	}
	require(t, result.OK, false)
	require(t, result.Mutation.Outcome, "partial")
	require(t, result.Data.Document.ID, documentID)
	require(t, result.Data.Grant.Status, "failed")
	require(t, len(result.Mutation.Receipts), 2)
	for _, id := range result.Mutation.Receipts {
		success(t, f.Run("receipts", "inspect", id))
	}
	require(t, creates.Load(), int32(1))
	require(t, grants.Load(), int32(1))
}

func TestDocsIdentityAndScopeFailures(t *testing.T) {
	t.Run("unsupported bot search", func(t *testing.T) {
		f := nextfixture.NewFixture(t, nil)
		r := f.Run("docs", "+search", "--query", "synthetic", "--as", "bot")
		if r.ExitCode == 0 || r.Stdout != "" {
			t.Fatalf("unsupported identity accepted: %+v", r)
		}
		var e struct {
			Error struct {
				Type string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(r.Stderr), &e); err != nil || e.Error.Type == "" {
			t.Fatalf("missing typed identity failure: %+v", r)
		}
	})
	t.Run("read scope rejection", func(t *testing.T) {
		var requests atomic.Int32
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			require(t, r.Method, "POST")
			require(t, r.URL.Path, "/open-apis/docs_ai/v1/documents/"+documentID+"/fetch")
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":99991679,"msg":"permission denied: docx:document:readonly","data":{}}`)
		}))
		r := run(f, "+fetch", "--doc", documentID)
		if r.ExitCode != 3 || r.Stdout != "" {
			t.Fatalf("scope failure misreported: %+v", r)
		}
		if strings.Contains(r.Stderr, `"mutation"`) {
			t.Fatalf("read rejection journaled as mutation: %s", r.Stderr)
		}
		require(t, requests.Load(), int32(1))
	})
}

func TestDocsExplicitMarkdownConversionsPreserveSidecars(t *testing.T) {
	const source = "<title>路线图 😀</title>\n<sheet token=\"syntheticSheet\" sheet-id=\"S1\"></sheet>"
	for _, format := range []string{"markdown", "im-markdown"} {
		t.Run(format, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b := body(t, r)
				require(t, b["format"], "markdown")
				require(t, b["export_option"].(map[string]any)["export_block_id"], false)
				reply(w, map[string]any{"document": map[string]any{"document_id": documentID, "revision_id": 3, "content": source, "reference_map": map[string]any{"comments": map[string]any{"c1": map[string]any{"data": "synthetic comment 中文"}}}}})
			}))
			result := success(t, run(f, "+fetch", "--doc", "https://fixture.example/docx/"+documentID, "--doc-format", format, "--detail", "full"))
			data := result["data"].(map[string]any)
			doc := data["document"].(map[string]any)
			wanted := source
			if format == "im-markdown" {
				wanted = "# 路线图 😀\n[sheet S1](https://fixture.example/sheets/syntheticSheet)"
			}
			require(t, doc["content"], wanted)
			require(t, doc["reference_map"].(map[string]any)["comments"].(map[string]any)["c1"].(map[string]any)["data"], "synthetic comment 中文")
			if len(data["warnings"].([]any)) == 0 {
				t.Fatalf("missing explicit detail downgrade warning: %#v", data)
			}
		})
	}
}
