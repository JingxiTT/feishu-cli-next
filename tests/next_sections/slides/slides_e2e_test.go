// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package slides_test

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }
func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func success(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("exit %d stdout=%s stderr=%s", r.ExitCode, r.Stdout, r.Stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &out); err != nil {
		t.Fatal(err, r.Stdout)
	}
	return out
}
func run(f *nextfixture.Fixture, args ...string) nextfixture.Result {
	return f.Run(append([]string{"slides"}, append(args, "--as", "user")...)...)
}

func TestPageLifecyclePreservesRichContentAndNoop(t *testing.T) {
	var mu sync.Mutex
	page := `<slide id="p1"><data><shape id="b1" type="text"><content><p>你好  café</p></content></shape><svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0"/></svg><line id="ln1"/><polyline id="pl1"/><img id="im1" src="media1"/><icon id="ic1" type="check"/><table id="tb1"><tr><td><content><p>001</p></content></td></tr></table><chart id="ch1" type="bar"><data>1,2</data></chart><embed id="em1" src="resource1"/></data><note id="n1">notes</note></slide>`
	writes := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/slide") {
			if r.URL.Query().Get("slide_id") != "p1" {
				t.Error(r.URL)
			}
			reply(w, map[string]any{"revision_id": 8, "slide": map[string]any{"slide_id": "p1", "content": page}})
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/slide/replace") {
			writes++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			parts := body["parts"].([]any)
			part := parts[0].(map[string]any)
			if len(parts) != 1 || part["block_id"] != "p1" {
				t.Error(body)
			}
			page = part["replacement"].(string)
			reply(w, map[string]any{"revision_id": 9})
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", 500)
	}))
	out := success(t, run(f, "+xml-get", "--presentation", "pres1", "--slide-id", "p1", "--save-result", "page.json"))
	if out["artifact"] == nil {
		t.Fatal(out)
	}
	artifact, err := os.ReadFile(filepath.Join(f.WorkDir, "page.json"))
	if err != nil || !strings.Contains(string(artifact), "你好") {
		t.Fatal(err, string(artifact))
	}
	success(t, run(f, "+update-slide", "--presentation", "pres1", "--slide-id", "p1", "--content", page, "--skip-unchanged"))
	if writes != 0 {
		t.Fatal("unchanged page wrote")
	}
	changed := strings.Replace(page, "你好", "世界", 1)
	success(t, run(f, "+update", "--presentation", "pres1", "--slide-id", "p1", "--xml", changed, "--skip-unchanged", "--no-lint"))
	if writes != 1 || page != strings.Replace(changed, ` id="n1"`, "", 1) {
		t.Fatal(writes, page)
	}
	// Default write-only behavior performs no read; explicit transaction semantics stay writes.
	success(t, run(f, "+update-slide", "--presentation", "pres1", "--slide-id", "p1", "--content", page, "--tid", "txn", "--skip-unchanged"))
	if writes != 2 {
		t.Fatal(writes)
	}
	r := run(f, "+update-slide", "--presentation", "pres1", "--slide-id", "other", "--content", page)
	var problem struct {
		Error struct {
			Param string `json:"param"`
		} `json:"error"`
	}
	_ = json.Unmarshal([]byte(r.Stderr), &problem)
	if r.ExitCode != 2 || problem.Error.Param != "--content" {
		t.Fatal(r)
	}
	raw := run(f, "+xml-get", "--presentation", "pres1", "--slide-id", "p1", "--raw")
	if raw.ExitCode != 0 || raw.Stdout != page {
		t.Fatal(raw)
	}
	success(t, run(f, "+xml-get", "--presentation", "pres1", "--slide-id", "p1", "--output", "page.xml"))
	saved, _ := os.ReadFile(filepath.Join(f.WorkDir, "page.xml"))
	if string(saved) != page {
		t.Fatal(string(saved))
	}
}

func TestCreateAddDeleteAndReplaceOrdering(t *testing.T) {
	var calls []string
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		calls = append(calls, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/open-apis/slides_ai/v1/xml_presentations" {
			if !strings.Contains(body["xml_presentation"].(map[string]any)["content"].(string), "Deck &amp; 你好") {
				t.Error(body)
			}
			reply(w, map[string]any{"xml_presentation_id": "pres1", "revision_id": 1})
			return
		}
		if r.Method == "POST" {
			slide := body["slide"].(map[string]any)
			if !strings.Contains(slide["content"].(string), "<slide") {
				t.Error(body)
			}
			reply(w, map[string]any{"slide_id": "new1", "revision_id": 3})
			return
		}
		if r.URL.Query().Get("slide_id") == "" {
			t.Error(r.URL)
		}
		reply(w, map[string]any{"revision_id": 4})
	}))
	success(t, run(f, "+create", "--title", "Deck & 你好", "--slide", "<slide><data/></slide>"))
	success(t, run(f, "+add-slide", "--presentation", "pres1", "--slide", "<slide><data/></slide>", "--before-slide-id", "old1", "--revision-id", "2"))
	success(t, run(f, "+delete-slide", "--presentation", "pres1", "--slide-id", "new1", "--revision-id", "3"))
	pages := `[{"slide_id":"old1","content":"<slide><data/></slide>"}]`
	before := len(calls)
	success(t, run(f, "+replace-pages", "--presentation", "pres1", "--pages", pages, "--validate-only"))
	if len(calls) != before {
		t.Fatal(calls)
	}
	success(t, run(f, "+replace-pages", "--presentation", "pres1", "--pages", pages))
	tail := calls[len(calls)-2:]
	want := []string{"POST /open-apis/slides_ai/v1/xml_presentations/pres1/slide", "DELETE /open-apis/slides_ai/v1/xml_presentations/pres1/slide"}
	if !reflect.DeepEqual(tail, want) {
		t.Fatal(calls)
	}
}

func TestHistoryPaginationRevertAndStatus(t *testing.T) {
	calls := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch {
		case strings.HasSuffix(r.URL.Path, "/histories"):
			if r.URL.Query().Get("page_size") != "1" {
				t.Error(r.URL)
			}
			token := r.URL.Query().Get("page_token")
			if token == "" {
				reply(w, map[string]any{"items": []any{map[string]any{"history_version_id": "9007199254740993", "title": "你好"}}, "has_more": true, "page_token": "next"})
			} else {
				if token != "next" {
					t.Error(token)
				}
				reply(w, map[string]any{"items": []any{map[string]any{"history_version_id": "2"}}, "has_more": false})
			}
		case strings.HasSuffix(r.URL.Path, "/history/revert"):
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["history_version_id"] != "9007199254740993" {
				t.Error(b)
			}
			reply(w, map[string]any{"task_id": "task1"})
		case strings.HasSuffix(r.URL.Path, "/history/revert_status"):
			if r.URL.Query().Get("task_id") != "task1" {
				t.Error(r.URL)
			}
			reply(w, map[string]any{"status": "completed", "task_id": "task1"})
		default:
			t.Error(r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	r := run(f, "+history-list", "--presentation", "pres1", "--page-size", "1", "--result-fields", "title")
	success(t, r)
	if !strings.Contains(r.Stdout, `"has_more":true`) || !strings.Contains(r.Stdout, "9007199254740993") || !strings.Contains(r.Stdout, "next") {
		t.Fatal(r)
	}
	success(t, run(f, "+history-list", "--presentation", "pres1", "--page-size", "1", "--page-token", "next"))
	r = run(f, "+history-revert", "--presentation", "pres1", "--history-version-id", "9007199254740993")
	if r.ExitCode != 22 || !strings.Contains(r.Stdout, "task1") || !strings.Contains(r.Stderr, "unknown") {
		t.Fatal(r)
	}
	success(t, run(f, "+history-revert-status", "--presentation", "pres1", "--task-id", "task1"))
	if calls != 4 {
		t.Fatal(calls)
	}
}

func TestStructuralPartialAndUnknownNeverRetry(t *testing.T) {
	for _, mode := range []string{"partial", "lost", "scope"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				b, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(b), "block_replace") || !strings.Contains(string(b), `id=\"b1\"`) {
					t.Error(string(b))
				}
				switch mode {
				case "partial":
					reply(w, map[string]any{"revision_id": 3, "failed_part_index": 1, "failed_reason": "fixture failure"})
				case "scope":
					w.WriteHeader(403)
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 99991672, "msg": "missing scope"})
				case "lost":
					h := w.(http.Hijacker)
					c, _, _ := h.Hijack()
					_ = c.Close()
				}
			}))
			r := run(f, "+replace-slide", "--presentation", "pres1", "--slide-id", "p1", "--parts", `[{"action":"replace","target_id":"b1","content":"<shape type=\"text\"><content><p>你好</p></content></shape>"}]`)
			want := 22
			if mode == "scope" {
				want = 20
			}
			if r.ExitCode != want || calls != 1 {
				t.Fatal(r, calls)
			}
			if mode == "partial" && !strings.Contains(r.Stdout, "failed_part_index") {
				t.Fatal(r)
			}
			success(t, f.Run("receipts", "list"))
		})
	}
}

func TestScreenshotAndMediaBytes(t *testing.T) {
	payload := []byte{137, 80, 78, 71, 13, 10, 26, 10, 0, 1, 2}
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/lark_cli_file_event/report"):
			reply(w, map[string]any{})
		case strings.HasSuffix(r.URL.Path, "upload_all"):
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Error(err)
			}
			if r.FormValue("parent_type") != "slide_file" || r.FormValue("parent_node") != "pres1" {
				t.Error(r.Form)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Error(err)
			} else {
				b, _ := io.ReadAll(file)
				_ = file.Close()
				if !reflect.DeepEqual(b, payload) {
					t.Error(b)
				}
			}
			reply(w, map[string]any{"file_token": "media1"})
		case strings.HasSuffix(r.URL.Path, "/download"):
			if strings.Contains(r.URL.Path, "/fallback/") {
				w.WriteHeader(403)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 1061104, "msg": "permission denied"})
				return
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(payload)
		case strings.HasSuffix(r.URL.Path, "/preview_download"):
			if r.URL.Query().Get("preview_type") != "16" {
				t.Error(r.URL)
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(payload)
		case strings.HasSuffix(r.URL.Path, "/slide_images"):
			reply(w, map[string]any{"slide_images": []any{map[string]any{"slide_id": "p1", "format": 1, "data": base64.StdEncoding.EncodeToString(payload)}}})
		case strings.HasSuffix(r.URL.Path, "/render"):
			reply(w, map[string]any{"slide_image": map[string]any{"format": 1, "data": base64.StdEncoding.EncodeToString(payload)}})
		default:
			t.Error(r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	if err := os.WriteFile(filepath.Join(f.WorkDir, "input.png"), payload, 0600); err != nil {
		t.Fatal(err)
	}
	success(t, run(f, "+media-upload", "--presentation", "pres1", "--file", "input.png"))
	success(t, run(f, "+media-download", "--file-token", "media1", "--output", "download.png"))
	success(t, run(f, "+media-download", "--file-token", "fallback", "--output", "fallback.png"))
	success(t, run(f, "+screenshot", "--presentation", "pres1", "--slide", "1", "--output", "shot.png"))
	r := run(f, "+screenshot", "--content", "<slide/>", "--output", "render.png")
	success(t, r)
	if strings.Contains(r.Stdout, base64.StdEncoding.EncodeToString(payload)) || strings.Contains(r.Stderr, "receipts") {
		t.Fatal(r)
	}
	for _, name := range []string{"download.png", "fallback.png", "shot.png", "render.png"} {
		b, err := os.ReadFile(filepath.Join(f.WorkDir, name))
		if err != nil || !reflect.DeepEqual(b, payload) {
			t.Fatal(name, err, b)
		}
	}
}

func TestLivePrivateSlideLifecycle(t *testing.T) {
	t.Skip("BLOCKED: selectable live scenario requires verified current account and tenant plus private creation visibility BEFORE any content write; no verified private fixture/cleanup harness is available. Then create owned deck, upload image, add/update/no-op/replace/read/screenshot/history/delete and clean up only run-owned resources; any other recipient must be unambiguous Ying Wu.")
}

func TestSkipUnchangedDryRunAndReadFailure(t *testing.T) {
	reads := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads++
		if r.Method != "GET" {
			t.Error("unexpected write", r.Method)
		}
		w.WriteHeader(403)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 99991672, "msg": "missing scope"})
	}))
	args := []string{"+update-slide", "--presentation", "pres1", "--slide-id", "p1", "--content", `<slide id="p1"/>`, "--skip-unchanged"}
	r := run(f, append(args, "--dry-run")...)
	success(t, r)
	if reads != 0 || !strings.Contains(r.Stdout, "GET") || !strings.Contains(r.Stdout, "POST") || !strings.Contains(r.Stdout, "[2/2] Replace slide") {
		t.Fatal(r, reads)
	}
	r = run(f, args...)
	if r.ExitCode == 0 || reads != 1 || strings.Contains(r.Stderr, `"receipts"`) {
		t.Fatal(r, reads)
	}
}

func TestReplacementDeleteFailureRecovery(t *testing.T) {
	for _, keepGoing := range []bool{false, true} {
		t.Run(map[bool]string{false: "stop", true: "continue"}[keepGoing], func(t *testing.T) {
			var operations []string
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				operations = append(operations, r.Method)
				if r.Method == "POST" {
					reply(w, map[string]any{"slide_id": "new1", "revision_id": 3})
					return
				}
				w.WriteHeader(403)
				_ = json.NewEncoder(w).Encode(map[string]any{"code": 99991672, "msg": "missing scope"})
			}))
			args := []string{"+replace-pages", "--presentation", "pres1", "--pages", `[{"slide_id":"old1","content":"<slide/>"},{"slide_id":"old2","content":"<slide/>"}]`}
			if keepGoing {
				args = append(args, "--continue-on-error")
			}
			r := run(f, args...)
			want := []string{"POST", "DELETE"}
			if keepGoing {
				want = append(want, "POST", "DELETE")
				if !strings.Contains(r.Stdout, "delete_failed") || !strings.Contains(r.Stdout, "new1") {
					t.Fatal(r)
				}
			} else {
				if !strings.Contains(r.Stdout, "new1") || !strings.Contains(r.Stdout, "old1") || !strings.Contains(r.Stdout, "delete_failed") {
					t.Fatal(r)
				}
			}
			if r.ExitCode != 21 || !reflect.DeepEqual(operations, want) {
				t.Fatal(r, operations)
			}
		})
	}
}

func TestWikiResolutionAndEffectiveIdentities(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		for _, kind := range []string{"slides", "docx"} {
			t.Run(identity+"/"+kind, func(t *testing.T) {
				calls := 0
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					if r.Header.Get("X-Lark-Proxy-Identity") != identity {
						t.Error(r.Header.Get("X-Lark-Proxy-Identity"))
					}
					if strings.HasSuffix(r.URL.Path, "/get_node") {
						if r.URL.Query().Get("token") != "wikcn123" {
							t.Error(r.URL)
						}
						reply(w, map[string]any{"node": map[string]any{"obj_type": kind, "obj_token": "resolved1"}})
						return
					}
					if r.Method != "GET" || r.URL.Path != "/open-apis/slides_ai/v1/xml_presentations/resolved1" {
						t.Error(r.Method, r.URL)
					}
					reply(w, map[string]any{"xml_presentation": map[string]any{"content": "<presentation><slide/></presentation>", "revision_id": 1}})
				}))
				r := f.Run("slides", "+xml-get", "--presentation", "https://x.feishu.cn/wiki/wikcn123", "--as", identity)
				if kind == "slides" {
					success(t, r)
					if calls != 2 || !strings.Contains(r.Stdout, `"identity":"`+identity+`"`) {
						t.Fatal(r, calls)
					}
				} else {
					if r.ExitCode != 2 || calls != 1 {
						t.Fatal(r, calls)
					}
				}
			})
		}
	}
}
