// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestImportExportPreservesSourceAndAsyncRecovery(t *testing.T) {
	markdown := []byte("# 原文 Ω\n\n| Name | Value |\n| --- | --- |\n| 零 | 001 |\n")
	var imported []byte
	var tasks atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/medias/upload_all":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				http.Error(w, "file", 400)
				return
			}
			defer file.Close()
			imported, _ = io.ReadAll(file)
			reply(w, map[string]any{"file_token": "media1"})
		case "/open-apis/drive/v1/lark_cli_file_event/report":
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/import_tasks":
			b := body(t, r)
			if b["file_token"] != "media1" || b["type"] != "docx" || b["file_extension"] != "md" {
				t.Errorf("wrong import %#v", b)
			}
			tasks.Add(1)
			reply(w, map[string]any{"ticket": "import1"})
		case "/open-apis/drive/v1/import_tasks/import1":
			reply(w, map[string]any{"result": map[string]any{"token": "doc1", "type": "docx", "job_status": 0, "url": "https://example.feishu.cn/docx/doc1"}})
		case "/open-apis/docs_ai/v1/documents/doc1/fetch":
			b := body(t, r)
			if b["format"] != "markdown" {
				t.Errorf("wrong export format %#v", b)
			}
			reply(w, map[string]any{"document": map[string]any{"content": string(imported)}})
		case "/open-apis/drive/v1/export_tasks":
			b := body(t, r)
			if b["token"] != "doc1" || b["file_extension"] != "pdf" {
				t.Errorf("wrong export %#v", b)
			}
			reply(w, map[string]any{"ticket": "export1"})
		case "/open-apis/drive/v1/export_tasks/export1":
			reply(w, map[string]any{"result": map[string]any{"job_status": 0, "file_token": "pdf1", "file_name": "out", "file_extension": "pdf", "type": "docx", "file_size": 4}})
		case "/open-apis/drive/v1/export_tasks/file/pdf1/download":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="out.pdf"`)
			_, _ = w.Write([]byte("%PDF"))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	if err := os.WriteFile(filepath.Join(f.WorkDir, "source.md"), markdown, 0600); err != nil {
		t.Fatal(err)
	}
	importedResult := f.Run("drive", "+import", "--file", "source.md", "--type", "docx", "--as", "user")
	// Exact same-invocation task completion reconciles the accepted create;
	// an acknowledgement alone must never authorize replay or completion.
	successful(t, importedResult)
	if !strings.Contains(importedResult.Stdout, `"outcome":"confirmed_applied"`) || !strings.Contains(importedResult.Stdout, `"ready":true`) {
		t.Fatalf("completed import outcome lost: %+v", importedResult)
	}
	if tasks.Load() != 1 || !bytes.Equal(imported, markdown) {
		t.Fatal("import bytes changed or task replayed")
	}
	successful(t, f.Run("drive", "+task_result", "--scenario", "import", "--ticket", "import1", "--as", "user"))
	successful(t, f.Run("drive", "+export", "--token", "doc1", "--doc-type", "docx", "--file-extension", "markdown", "--file-name", "roundtrip.md", "--as", "user"))
	result := f.Run("drive", "+export", "--token", "doc1", "--doc-type", "docx", "--file-extension", "pdf", "--as", "user")
	successful(t, result) // Export is declared read-only and preserves its existing artifact workflow.
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "roundtrip.md"))
	if err != nil || !bytes.Equal(raw, markdown) {
		t.Fatalf("markdown transfer corrupted: %v %q", err, raw)
	}
	raw, err = os.ReadFile(filepath.Join(f.WorkDir, "out.pdf"))
	if err != nil || string(raw) != "%PDF" {
		t.Fatalf("export download: %v %q", err, raw)
	}
}

func TestPushPullPreservesDirectoryFiles(t *testing.T) {
	payload := []byte("路径/原文 Ω\n001\n")
	var stored []byte
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/files":
			files := []any{}
			if stored != nil {
				files = append(files, map[string]any{"token": "file1", "name": "original.txt", "type": "file", "size": len(stored)})
			}
			reply(w, map[string]any{"files": files, "has_more": false})
		case "/open-apis/drive/v1/files/upload_all":
			writes.Add(1)
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				http.Error(w, "file", 400)
				return
			}
			defer file.Close()
			stored, _ = io.ReadAll(file)
			if r.FormValue("parent_node") != "root" {
				t.Errorf("wrong push destination: %v", r.MultipartForm.Value)
			}
			reply(w, map[string]any{"file_token": "file1"})
		case "/open-apis/drive/v1/lark_cli_file_event/report":
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/files/file1/download":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(stored)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	for _, dir := range []string{"source", "destination"} {
		if err := os.Mkdir(filepath.Join(f.WorkDir, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.WorkDir, "source", "original.txt"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	successful(t, f.Run("drive", "+push", "--local-dir", "source", "--folder-token", "root", "--as", "user"))
	successful(t, f.Run("drive", "+pull", "--local-dir", "destination", "--folder-token", "root", "--as", "user"))
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "destination", "original.txt"))
	if err != nil || !bytes.Equal(raw, payload) || !bytes.Equal(stored, payload) || writes.Load() != 1 {
		t.Fatalf("directory transfer corruption/replay: %v writes=%d", err, writes.Load())
	}
}

func TestCoverAndSourcePreviewArtifacts(t *testing.T) {
	payload := []byte("synthetic preview bytes")
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/open-apis/drive/v1/medias/file1/preview_download" || r.URL.Query().Get("preview_type") == "" {
			t.Errorf("wrong preview request %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(payload)
	}))
	successful(t, f.Run("drive", "+cover", "--file-token", "file1", "--list-only", "--as", "user"))
	if calls.Load() != 0 {
		t.Fatal("built-in cover listing made a backend request")
	}
	successful(t, f.Run("drive", "+cover", "--file-token", "file1", "--spec", "square", "--output", "cover.png", "--as", "user"))
	successful(t, f.Run("drive", "+preview", "--file-token", "file1", "--type", "source_file", "--output", "source.bin", "--as", "user"))
	for _, name := range []string{"cover.png", "source.bin"} {
		raw, err := os.ReadFile(filepath.Join(f.WorkDir, name))
		if err != nil || !bytes.Equal(raw, payload) {
			t.Fatalf("preview byte corruption %s: %v", name, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("unexpected preview replay")
	}
}
