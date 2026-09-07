// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestAppsFileTransferAndCallbackPartial(t *testing.T) {
	for _, mode := range []string{"complete", "callback-denied", "lost-put"} {
		t.Run(mode, func(t *testing.T) {
			var bytesReceived []byte
			var uploads, callbacks atomic.Int32
			storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "PUT" {
					uploads.Add(1)
					bytesReceived, _ = io.ReadAll(r.Body)
					if r.Header.Get("Authorization") != "" {
						t.Error("gateway authorization leaked to storage")
					}
					for _, values := range r.Header {
						for _, value := range values {
							if strings.Contains(value, "synthetic-foundation-proxy-key") {
								t.Error("proxy key leaked to storage")
							}
						}
					}
					if mode == "lost-put" {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err == nil {
							_ = conn.Close()
						}
						return
					}
					w.Header().Set("ETag", "fixture-etag")
					return
				}
				if r.Method == "GET" {
					_, _ = w.Write(bytesReceived)
					return
				}
				t.Errorf("storage method %s", r.Method)
			}))
			defer storage.Close()
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasSuffix(r.URL.Path, "/storage/file_pre_upload"):
					reply(w, map[string]interface{}{"upload_id": "upload_fixture", "upload_url": storage.URL + "/object?signature=synthetic-storage-signature"})
				case strings.HasSuffix(r.URL.Path, "/storage/file_upload_callback"):
					callbacks.Add(1)
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["upload_id"] != "upload_fixture" || body["etag"] != "fixture-etag" {
						t.Errorf("callback data %#v", body)
					}
					if mode == "callback-denied" {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"code":99991679,"msg":"fixture scope denied"}`)
						return
					}
					reply(w, map[string]interface{}{"file_name": "source.txt", "path": "/object", "size": len(bytesReceived)})
				case strings.HasSuffix(r.URL.Path, "/storage/file_sign"):
					reply(w, map[string]interface{}{"signed_url": storage.URL + "/object?signature=synthetic-storage-signature"})
				case strings.HasSuffix(r.URL.Path, "/storage/file_batch_remove"):
					bytesReceived = nil
					reply(w, map[string]interface{}{"results": []interface{}{map[string]interface{}{"status": "ok"}}})
				default:
					t.Errorf("unexpected API %s", r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			content := "中文\x00binary\nline two"
			path := filepath.Join(f.WorkDir, "source.txt")
			writeFile(t, path, content)
			result := run(f, "+file-upload", "--app-id", "app_fixture", "--file", path)
			if mode == "lost-put" {
				expect(t, result, 22, "", "")
				if string(bytesReceived) != content || uploads.Load() != 1 || callbacks.Load() != 0 {
					t.Fatal("lost PUT replayed or callback ran")
				}
				receipts, err := filepath.Glob(filepath.Join(f.ConfigDir, "next/receipts/*.json"))
				if err != nil {
					t.Fatal(err)
				}
				unknown := false
				for _, receipt := range receipts {
					b, err := os.ReadFile(receipt)
					if err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(b), "synthetic-storage-signature") || strings.Contains(string(b), "中文") {
						t.Fatal("receipt persisted signature/content")
					}
					if gjson.GetBytes(b, "method").String() == "PUT" && strings.Contains(string(b), "unknown") {
						unknown = true
					}
				}
				if !unknown {
					t.Fatal("lost PUT lacked durable unknown receipt")
				}
				return
			}
			if mode == "callback-denied" {
				expect(t, result, 21, "", "")
				if string(bytesReceived) != content || uploads.Load() != 1 || callbacks.Load() != 1 {
					t.Fatal("partial upload replayed/lost")
				}
				return
			}
			expect(t, result, 0, "data.path", "/object")
			expect(t, run(f, "+file-download", "--app-id", "app_fixture", "--path", "/object", "--output", "download.bin"), 0, "", "")
			downloaded, err := os.ReadFile(filepath.Join(f.WorkDir, "download.bin"))
			if err != nil || string(downloaded) != content {
				t.Fatalf("bytes changed: %q %v", downloaded, err)
			}
			deletion := run(f, "+file-delete", "--app-id", "app_fixture", "--path", "/object", "--yes")
			if deletion.ExitCode != 0 && deletion.ExitCode != 22 {
				t.Fatalf("delete %+v", deletion)
			}
			if len(bytesReceived) != 0 {
				t.Fatal("delete effect absent")
			}
		})
	}
}

func TestAppsDataExportImportPreservesRawTypesAndExtent(t *testing.T) {
	content := `[{"id":9007199254740993,"decimal":123.1234567890123456789,"name":"中文","empty":"","null":null}]`
	var imported []byte
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/records"):
			reply(w, map[string]interface{}{"total": 3})
		case strings.HasSuffix(r.URL.Path, "/db/data_export"):
			if r.URL.Query().Get("limit") != "1" || r.URL.Query().Get("format") != "json" {
				t.Errorf("export params %s", r.URL)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, content)
		case strings.HasSuffix(r.URL.Path, "/db/data_import"):
			writes.Add(1)
			if err := r.ParseMultipartForm(1024 * 1024); err != nil {
				t.Error(err)
				return
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
				return
			}
			defer file.Close()
			imported, _ = io.ReadAll(file)
			if r.URL.Query().Get("table") != "copy" {
				t.Error("import target changed")
			}
			reply(w, map[string]interface{}{"table": "copy", "rows": 1})
		default:
			t.Errorf("unexpected export/import route %s", r.URL.Path)
			w.WriteHeader(500)
		}
	}))
	result := run(f, "+db-data-export", "--app-id", "app_fixture", "--table", "orders", "--output", "rows.json", "--limit", "1")
	expect(t, result, 0, "data.rows", "1")
	expect(t, result, 0, "data.complete", "false")
	expect(t, result, 0, "data.has_more", "true")
	b, err := os.ReadFile(filepath.Join(f.WorkDir, "rows.json"))
	if err != nil || string(b) != content {
		t.Fatalf("export bytes changed %s %v", b, err)
	}
	result = run(f, "+db-data-import", "--app-id", "app_fixture", "--table", "copy", "--file", "rows.json", "--yes")
	expect(t, result, 0, "data.rows", "1")
	if string(imported) != content || writes.Load() != 1 {
		t.Fatal("raw import data changed or replayed")
	}
}

func TestAppsMissingDeleteItemsNeverClaimSuccess(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(w, map[string]interface{}{"results": []interface{}{map[string]interface{}{"status": "ok"}}})
	}))
	result := run(f, "+file-delete", "--app-id", "app_fixture", "--path", "/a", "--path", "/b", "--yes")
	if gjson.Get(result.Stdout, "data.results.1.status").String() != "unknown" {
		t.Fatalf("missing item claimed deleted %+v", result)
	}
}

func TestAppsDataImportRejectsArtifactBeforeDispatch(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	writeFile(t, filepath.Join(f.WorkDir, "artifact.json"), `{"schema":"feishu-cli-next/result/v1","binding":{"app_id":"other"},"result":{"data":[]}}`)
	result := run(f, "+db-data-import", "--app-id", "app_fixture", "--file", "artifact.json", "--yes")
	expect(t, result, 2, "", "")
	if !strings.Contains(result.Stderr, "native database import") {
		t.Fatalf("wrong artifact error: %+v", result)
	}
}

func TestAppsHTMLStorageAndReleaseRemainSeparateEffects(t *testing.T) {
	for _, mode := range []string{"complete", "lost-put", "release-denied"} {
		t.Run(mode, func(t *testing.T) {
			var uploads, releases atomic.Int32
			storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				uploads.Add(1)
				body, _ := io.ReadAll(r.Body)
				if r.Method != "PUT" || len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
					t.Errorf("expected gzip PUT, got %s %d bytes", r.Method, len(body))
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("gateway auth leaked to TOS")
				}
				if mode == "lost-put" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
					return
				}
				w.WriteHeader(204)
			}))
			defer storage.Close()
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && r.URL.Path == "/open-apis/spark/v1/apps/app_fixture":
					reply(w, map[string]interface{}{"app": map[string]interface{}{"app_id": "app_fixture", "app_type": "HTML"}})
				case strings.HasSuffix(r.URL.Path, "/pre_release"):
					reply(w, map[string]interface{}{"kvs": []interface{}{map[string]interface{}{"key": "upload_url", "value": storage.URL + "/archive?signature=synthetic-html-signature"}, map[string]interface{}{"key": "tos_path", "value": "tos://fixture/archive"}}})
				case strings.HasSuffix(r.URL.Path, "/releases"):
					releases.Add(1)
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["tos_path"] != "tos://fixture/archive" {
						t.Errorf("release body %#v", body)
					}
					if mode == "release-denied" {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"code":99991679,"msg":"fixture denied"}`)
						return
					}
					reply(w, map[string]interface{}{"release_id": "release_fixture", "status": "publishing"})
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			path := filepath.Join(f.WorkDir, "index.html")
			writeFile(t, path, "<!doctype html><title>Fixture</title>")
			result := run(f, "+html-publish", "--app-id", "app_fixture", "--path", path)
			switch mode {
			case "complete":
				expect(t, result, 0, "data.release_id", "release_fixture")
			case "lost-put":
				expect(t, result, 22, "", "")
			case "release-denied":
				expect(t, result, 21, "", "")
			}
			expectedReleases := int32(1)
			if mode == "lost-put" {
				expectedReleases = 0
			}
			if uploads.Load() != 1 || releases.Load() != expectedReleases {
				t.Fatalf("effects uploads=%d releases=%d", uploads.Load(), releases.Load())
			}
			files, _ := filepath.Glob(filepath.Join(f.ConfigDir, "next/receipts/*.json"))
			found := false
			for _, file := range files {
				b, err := os.ReadFile(file)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), "synthetic-html-signature") {
					t.Fatal("persisted signed URL")
				}
				if gjson.GetBytes(b, "method").String() == "PUT" {
					found = true
					if mode == "lost-put" && !strings.Contains(string(b), "unknown") {
						t.Fatal("lost PUT not unknown")
					}
				}
			}
			if !found {
				t.Fatal("storage PUT receipt missing")
			}
		})
	}
}
