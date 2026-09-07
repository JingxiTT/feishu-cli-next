// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"bytes"
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

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func successful(t *testing.T, got nextfixture.Result) map[string]any {
	t.Helper()
	if got.ExitCode != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", got.ExitCode, got.Stdout, got.Stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(got.Stdout), &out); err != nil {
		t.Fatalf("JSON stdout: %v: %s", err, got.Stdout)
	}
	if out["ok"] != true {
		t.Fatalf("unsuccessful envelope: %s", got.Stdout)
	}
	return out
}
func body(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var b map[string]any
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		t.Error(err)
	}
	return b
}

func TestUploadDownloadAndVersionBytes(t *testing.T) {
	payload := append([]byte{0, 1, 255}, bytes.Repeat([]byte("中文 Ω\r\n"), 1000)...)
	var stored []byte
	var uploads atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/files/upload_all":
			uploads.Add(1)
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				http.Error(w, "multipart", 400)
				return
			}
			defer file.Close()
			stored, err = io.ReadAll(file)
			if err != nil {
				t.Error(err)
			}
			if r.FormValue("parent_node") != "root" || r.FormValue("file_name") != "原文.bin" {
				t.Errorf("wrong upload destination/name: %v", r.MultipartForm.Value)
			}
			reply(w, map[string]any{"file_token": "file1", "file_version": "7"})
		case "/open-apis/drive/v1/lark_cli_file_event/report":
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/metas/batch_query":
			reply(w, map[string]any{"metas": []any{map[string]any{"doc_token": "file1", "doc_type": "file", "title": "原文.bin", "url": "https://example.feishu.cn/file/file1"}}})
		case "/open-apis/drive/v1/permissions/file1/members/auth":
			reply(w, map[string]any{"auth_result": true})
		case "/open-apis/drive/v1/files/file1/download", "/open-apis/drive/v1/export_tasks/file/export1/download":
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="source.bin"`)
			_, _ = w.Write(stored)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	source := filepath.Join(f.WorkDir, "input.bin")
	if err := os.WriteFile(source, payload, 0600); err != nil {
		t.Fatal(err)
	}
	successful(t, f.Run("drive", "+upload", "--file", source, "--folder-token", "root", "--name", "原文.bin", "--as", "user"))
	if !bytes.Equal(stored, payload) || uploads.Load() != 1 {
		t.Fatal("upload changed bytes or retried")
	}
	commands := [][]string{
		{"+download", "--file-token", "file1", "--output", "download.bin"},
		{"+version-get", "--file-token", "file1", "--version", "7", "--output", "version.bin"},
		{"+export-download", "--file-token", "export1", "--file-name", "export.bin"},
	}
	for _, args := range commands {
		successful(t, f.Run(append(append([]string{"drive"}, args...), "--as", "user")...))
	}
	for _, name := range []string{"download.bin", "version.bin", "export.bin"} {
		got, err := os.ReadFile(filepath.Join(f.WorkDir, name))
		if err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("%s bytes changed: %v", name, err)
		}
	}
	failed := f.Run("drive", "+download", "--file-token", "file1", "--output", "download.bin", "--as", "user")
	if failed.ExitCode == 0 {
		t.Fatal("existing output must be protected")
	}
	got, _ := os.ReadFile(filepath.Join(f.WorkDir, "download.bin"))
	if !bytes.Equal(got, payload) {
		t.Fatal("conflicting download changed local file")
	}
}

func TestWikiCopyMoveAndGuardedRename(t *testing.T) {
	var copied, moved, patches atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/wiki/v2/spaces/get_node":
			reply(w, map[string]any{"node": map[string]any{"node_token": "wiki1", "obj_token": "source1", "obj_type": "file", "title": "原文.bin"}})
		case "/open-apis/drive/explorer/v2/root_folder/meta":
			reply(w, map[string]any{"token": "root"})
		case "/open-apis/drive/v1/files/source1/copy":
			b := body(t, r)
			if b["folder_token"] != "root" || b["type"] != "file" || b["name"] != "副本.bin" {
				t.Errorf("wrong copy: %#v", b)
			}
			copied.Add(1)
			reply(w, map[string]any{"file": map[string]any{"token": "copy1", "name": "副本.bin", "type": "file"}})
		case "/open-apis/drive/v1/files/copy1/move":
			b := body(t, r)
			if b["folder_token"] != "destination" || b["type"] != "file" {
				t.Errorf("wrong move: %#v", b)
			}
			moved.Add(1)
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/metas/batch_query":
			reply(w, map[string]any{"metas": []any{map[string]any{"doc_token": "copy1", "doc_type": "file", "title": "副本.bin"}}})
		case "/open-apis/drive/v1/files/copy1":
			patches.Add(1)
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	successful(t, f.Run("drive", "+copy", "--url", "https://example.feishu.cn/wiki/wiki1", "--name", "副本.bin", "--folder-token", "my_space", "--as", "user"))
	successful(t, f.Run("drive", "+move", "--file-token", "copy1", "--type", "file", "--folder-token", "destination", "--as", "user"))
	got := f.Run("drive", "+update-title", "--token", "copy1", "--type", "file", "--title", "副本", "--as", "user")
	out := successful(t, got)
	data := out["data"].(map[string]any)
	if data["unchanged"] != true || data["updated"] != false || patches.Load() != 0 || copied.Load() != 1 || moved.Load() != 1 {
		t.Fatalf("wrong effects: %s copies=%d moves=%d patches=%d", got.Stdout, copied.Load(), moved.Load(), patches.Load())
	}
}

func TestMirrorExactHashNoopAndIncompleteListing(t *testing.T) {
	for _, mode := range []string{"same", "missing_cursor", "repeated_cursor", "cycle", "duplicate"} {
		t.Run(mode, func(t *testing.T) {
			var reads, writes atomic.Int32
			payload := []byte("保持原文\n")
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					writes.Add(1)
					http.Error(w, "no mutations permitted", 500)
					return
				}
				switch r.URL.Path {
				case "/open-apis/drive/v1/files":
					reads.Add(1)
					data := map[string]any{"files": []any{map[string]any{"token": "file1", "name": "same.txt", "type": "file"}}, "has_more": false}
					switch mode {
					case "missing_cursor":
						data["has_more"] = true
					case "repeated_cursor":
						data["has_more"] = true
						data["next_page_token"] = "same"
					case "cycle":
						data["files"] = []any{map[string]any{"token": "root", "name": "loop", "type": "folder"}}
					case "duplicate":
						data["files"] = []any{map[string]any{"token": "a", "name": "same.txt", "type": "file"}, map[string]any{"token": "b", "name": "same.txt", "type": "file"}}
					}
					reply(w, data)
				case "/open-apis/drive/v1/files/file1/download":
					w.Header().Set("Content-Type", "application/octet-stream")
					_, _ = w.Write(payload)
				default:
					t.Errorf("unexpected %s", r.URL)
					http.Error(w, "unexpected", 500)
				}
			}))
			if err := os.WriteFile(filepath.Join(f.WorkDir, "same.txt"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "same" {
				successful(t, f.Run("drive", "+status", "--local-dir", ".", "--folder-token", "root", "--as", "user"))
			}
			got := f.Run("drive", "+sync", "--local-dir", ".", "--folder-token", "root", "--as", "user")
			if mode == "same" {
				successful(t, got)
				if !strings.Contains(got.Stdout, `"unchanged"`) {
					t.Fatal(got.Stdout)
				}
			} else if got.ExitCode == 0 {
				t.Fatalf("invalid listing succeeded: %s", got.Stdout)
			}
			if reads.Load() > 2 || writes.Load() != 0 {
				t.Fatalf("unbounded reads or side effect: reads=%d writes=%d", reads.Load(), writes.Load())
			}
			saved, _ := os.ReadFile(filepath.Join(f.WorkDir, "same.txt"))
			if !bytes.Equal(saved, payload) {
				t.Fatal("local file changed")
			}
		})
	}
}

func TestSearchArtifactPreservesContinuation(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/search/v2/doc_wiki/search" {
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
			return
		}
		calls.Add(1)
		b := body(t, r)
		if b["query"] != "原文" || b["page_token"] != "previous" {
			t.Errorf("wrong search: %#v", b)
		}
		reply(w, map[string]any{"total": 2, "has_more": true, "page_token": "continue", "res_units": []any{map[string]any{"id": "result1", "title": "原文 Ω", "empty": "", "null": nil, "leading": "001"}}})
	}))
	got := f.Run("drive", "+search", "--query", "原文", "--page-token", "previous", "--page-size", "1", "--save-result", "search.json", "--as", "user")
	successful(t, got)
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "search.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"page_token":"continue"`, `"has_more":true`, `"leading":"001"`, `"empty":""`, `"null":null`, `"user_verified":true`, `"tenant_verified":true`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Fatalf("artifact missing %s: %s", want, raw)
		}
	}
	if calls.Load() != 1 || !strings.Contains(got.Stdout, "continue") {
		t.Fatalf("bounded search lost continuation: %s", got.Stdout)
	}
}

func TestMutationUnknownIsNotReplayed(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/open-apis/drive/v1/files/create_folder" {
			t.Errorf("unexpected %s", r.URL)
		}
		// A response lost after application cannot authorize another create.
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	got := f.Run("drive", "+create-folder", "--name", "fixture", "--folder-token", "root", "--as", "user")
	if got.ExitCode != 22 || calls.Load() != 1 || !strings.Contains(got.Stderr, `"outcome":"unknown"`) {
		t.Fatalf("unknown mutation: exit=%d calls=%d stdout=%s stderr=%s", got.ExitCode, calls.Load(), got.Stdout, got.Stderr)
	}
	receipts := f.Run("receipts", "list")
	if receipts.ExitCode != 0 || !strings.Contains(receipts.Stdout, "unknown") {
		t.Fatalf("missing durable receipt: %+v", receipts)
	}
	if calls.Load() != 1 {
		t.Fatal("receipt inspection contacted backend")
	}
}
