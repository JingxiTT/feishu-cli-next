// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr_test

import (
	"compress/gzip"
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
func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func success(t *testing.T, r nextfixture.Result, want string) {
	t.Helper()
	if r.ExitCode != 0 || !json.Valid([]byte(r.Stdout)) || !strings.Contains(r.Stdout, want) {
		t.Fatalf("want %s: %+v", want, r)
	}
}

func TestFrozenOKRDiscovery(t *testing.T) {
	f, e := os.Open("../../../docs/reference/v1.0.97/registrations.json.gz")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	z, e := gzip.NewReader(f)
	if e != nil {
		t.Fatal(e)
	}
	defer z.Close()
	var snapshot map[string]struct {
		Commands map[string]struct {
			Flags map[string]struct {
				Hidden bool `json:"hidden"`
			} `json:"flags"`
		} `json:"commands"`
	}
	if e = json.NewDecoder(z).Decode(&snapshot); e != nil {
		t.Fatal(e)
	}
	fxt := nextfixture.NewFixture(t, nil)
	count := 0
	for name, contract := range snapshot["feishu"].Commands {
		if name != "okr" && !strings.HasPrefix(name, "okr ") {
			continue
		}
		count++
		t.Run(name, func(t *testing.T) {
			r := fxt.Run(append(strings.Fields(name), "--help")...)
			if r.ExitCode != 0 {
				t.Fatalf("%+v", r)
			}
			for flag, v := range contract.Flags {
				// authsidecar intentionally hides local profile selection.
				if flag != "profile" && !v.Hidden && !strings.Contains(r.Stdout, "--"+flag) {
					t.Errorf("missing %s", flag)
				}
			}
		})
	}
	if count != 57 {
		t.Fatalf("frozen count %d", count)
	}
}

// Every shortcut executes through the production process. The backend checks
// concrete dispatch and returns independent IDs, not the command's own output.
func TestOKRShortcutDispatch(t *testing.T) {
	const simple = `{"text":"目标 中文 😀","mention":["ou_synthetic"]}`
	cases := []struct {
		name                               string
		args                               []string
		method, path, response, want, body string
	}{
		{"cycle-list", []string{"--user-id", "ou_synthetic"}, "GET", "/open-apis/okr/v2/cycles", `{"items":[{"id":"101"}],"has_more":true,"page_token":"next+="}`, `"page_token":"next+="`, ""},
		{"progress-list", []string{"--target-id", "101", "--target-type", "objective"}, "GET", "/open-apis/okr/v2/objectives/101/progresses", `{"items":[{"id":"301"}],"has_more":true,"page_token":"next+="}`, `"301"`, ""},
		{"progress-get", []string{"--progress-id", "301", "--style", "richtext"}, "GET", "/open-apis/okr/v1/progress_records/301", `{"progress_id":"301"}`, `"301"`, ""},
		{"create", []string{"--level", "objective", "--cycle-id", "101", "--content", simple, "--notes", simple, "--category-id", "901", "--user-id-type", "union_id"}, "POST", "/open-apis/okr/v2/cycles/101/objectives", `{"objective_id":"201"}`, `"201"`, `"category_id":"901"`},
		{"create", []string{"--level", "key-result", "--objective-id", "201", "--content", simple}, "POST", "/open-apis/okr/v2/objectives/201/key_results", `{"key_result_id":"202"}`, `"202"`, "目标 中文 😀"},
		{"patch", []string{"--level", "objective", "--target-id", "201", "--content", simple, "--notes", simple, "--score", "0", "--deadline", "1750000000000"}, "PATCH", "/open-apis/okr/v2/objectives/201", `{}`, `"target_id":"201"`, `"score":0`},
		{"patch", []string{"--level", "key-result", "--target-id", "202", "--content", simple}, "PATCH", "/open-apis/okr/v2/key_results/202", `{}`, `"target_id":"202"`, "目标 中文 😀"},
		{"progress-create", []string{"--target-id", "201", "--target-type", "objective", "--content", simple, "--progress-percent", "0", "--progress-status", "normal", "--source-title", "Synthetic source", "--source-url", "https://example.test/source"}, "POST", "/open-apis/okr/v1/progress_records/", `{"progress_id":"301"}`, `"301"`, "Synthetic source"},
		{"progress-update", []string{"--progress-id", "301", "--content", simple, "--progress-percent", "100", "--progress-status", "done"}, "PUT", "/open-apis/okr/v1/progress_records/301", `{"progress_id":"301"}`, `"301"`, `"percent":100`},
		{"progress-delete", []string{"--progress-id", "301", "--yes"}, "DELETE", "/open-apis/okr/v1/progress_records/301", `{}`, `"301"`, ""},
		{"comment-list", []string{"--target-id", "201", "--target-type", "objective", "--user-id-type", "user_key", "--page-token", "opaque+=", "--page-size", "1"}, "GET", "/open-apis/okr/v2/comments", `{"items":[{"id":"401"}],"has_more":true,"page_token":"next+="}`, `"401"`, ""},
		{"comment-get", []string{"--comment-id", "401", "--style", "richtext"}, "GET", "/open-apis/okr/v2/comments/401", `{"comment":{"id":"401"}}`, `"401"`, ""},
		{"comment-create", []string{"--target-id", "201", "--target-type", "objective", "--content", simple, "--selected-text", "目标"}, "POST", "/open-apis/okr/v2/comments", `{"comment_id":"401","selection_id":"501"}`, `"501"`, `"selected_text":"目标"`},
		{"comment-create", []string{"--target-id", "201", "--target-type", "objective", "--content", simple, "--ref-comment-id", "401"}, "POST", "/open-apis/okr/v2/comments", `{"comment_id":"402"}`, `"402"`, `"ref_comment_id":"401"`},
		{"comment-patch", []string{"--comment-id", "401", "--content", simple}, "PATCH", "/open-apis/okr/v2/comments/401", `{"comment":{"id":"401"}}`, `"401"`, "目标 中文 😀"},
		{"comment-solve", []string{"--comment-id", "401"}, "POST", "/open-apis/okr/v2/comments/401/solve", `{"affected_comments":[{"id":"401","solved":true}]}`, `"401"`, ""},
		{"comment-reopen", []string{"--comment-id", "401"}, "POST", "/open-apis/okr/v2/comments/401/reopen", `{"affected_comments":[{"id":"401","solved":false}]}`, `"401"`, ""},
		{"comment-delete", []string{"--comment-id", "401", "--yes"}, "DELETE", "/open-apis/okr/v2/comments/401", `{}`, `"401"`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name+strings.Join(tc.args[:2], "-"), func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("dispatch %s %s", r.Method, r.URL)
				}
				b, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(b), tc.body) {
					t.Errorf("body %s lacks %s", b, tc.body)
				}
				var d any
				_ = json.Unmarshal([]byte(tc.response), &d)
				reply(w, d)
			}))
			args := append([]string{"okr", "+" + tc.name, "--as", "user"}, tc.args...)
			success(t, f.Run(append(args, "--dry-run")...), "/open-apis/okr/")
			if calls.Load() != 0 {
				t.Fatal("dry run dispatched")
			}
			success(t, f.Run(args...), tc.want)
			if calls.Load() != 1 {
				t.Fatalf("calls %d", calls.Load())
			}
		})
	}
}

func TestOKRCompositionAndNoop(t *testing.T) {
	for _, level := range []string{"objective", "key-result"} {
		for _, op := range []string{"reorder", "weight", "indicator-update"} {
			t.Run(level+op, func(t *testing.T) {
				var writes atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == "GET" {
						if strings.HasSuffix(r.URL.Path, "/indicators") {
							reply(w, map[string]any{"indicator": map[string]any{"id": "901"}})
						} else {
							reply(w, map[string]any{"items": []any{map[string]any{"id": "201", "position": 1, "weight": 0.5}, map[string]any{"id": "202", "position": 2, "weight": 0.5}}, "has_more": false})
						}
						return
					}
					writes.Add(1)
					b, _ := io.ReadAll(r.Body)
					if !strings.Contains(string(b), "201") && op != "indicator-update" {
						t.Errorf("body %s", b)
					}
					reply(w, map[string]any{})
				}))
				args := []string{"okr", "+" + op, "--level", level}
				want := `"total":2`
				switch op {
				case "reorder":
					args = append(args, "--cycle-id", "101", "--ops", `[{"id":"201","position":1}]`)
				case "weight":
					args = append(args, "--cycle-id", "101", "--weights", `[{"id":"201","weight":0.25}]`)
				case "indicator-update":
					args = append(args, "--id", "201", "--value", "-12.5")
					want = `"current_value":-12.5`
				}
				if level == "key-result" && op != "indicator-update" {
					args = append(args, "--objective-id", "101")
				}
				success(t, f.Run(args...), want)
				expected := int32(1)
				if op == "reorder" {
					expected = 0
				}
				if writes.Load() != expected {
					t.Fatalf("writes %d want %d", writes.Load(), expected)
				}
				if op == "weight" {
					for i := range args {
						if args[i] == `[{"id":"201","weight":0.25}]` {
							args[i] = `[{"id":"201","weight":0.5}]`
						}
					}
					success(t, f.Run(args...), `"noop":true`)
					if writes.Load() != 1 {
						t.Fatal("unchanged weights dispatched")
					}
				}
				if op == "reorder" {
					for i := range args {
						if args[i] == `[{"id":"201","position":1}]` {
							args[i] = `[{"id":"202","position":1}]`
						}
					}
					success(t, f.Run(args...), `"noop":false`)
					if writes.Load() != 1 {
						t.Fatal("changed order not dispatched")
					}
				}
			})
		}
	}
}

func TestOKRCycleAndCommentDetail(t *testing.T) {
	for _, name := range []string{"cycle-detail", "comment-detail"} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" {
					t.Errorf("write %s", r.Method)
				}
				switch {
				case strings.HasSuffix(r.URL.Path, "/objectives"):
					reply(w, map[string]any{"items": []any{map[string]any{"id": "201", "position": 1, "notes": map[string]any{"blocks": []any{}}}}, "has_more": false})
				case strings.HasSuffix(r.URL.Path, "/key_results"):
					reply(w, map[string]any{"items": []any{map[string]any{"id": "202", "position": 1}}, "has_more": false})
				case strings.HasSuffix(r.URL.Path, "/progresses"):
					reply(w, map[string]any{"items": []any{map[string]any{"id": "301"}}, "has_more": false})
				case strings.HasSuffix(r.URL.Path, "/comments"):
					reply(w, map[string]any{"items": []any{map[string]any{"id": "401", "create_time": "1750000000000"}}, "has_more": false})
				default:
					t.Errorf("unexpected %s", r.URL)
					reply(w, map[string]any{})
				}
			}))
			success(t, f.Run("okr", "+"+name, "--cycle-id", "101", "--style", "richtext"), `"201"`)
			if calls.Load() < 2 {
				t.Fatal("composition omitted")
			}
		})
	}
}

func TestOKRBatchRollbackAndUnknownOutcome(t *testing.T) {
	for _, mode := range []string{"success", "rollback", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			var creates, deletes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "DELETE" {
					deletes.Add(1)
					reply(w, map[string]any{})
					return
				}
				creates.Add(1)
				if strings.HasSuffix(r.URL.Path, "/key_results") {
					if mode == "rollback" {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"code":99991672,"msg":"synthetic denied"}`)
						return
					}
					if mode == "unknown" {
						h := w.(http.Hijacker)
						c, _, _ := h.Hijack()
						_ = c.Close()
						return
					}
					reply(w, map[string]any{"key_result_id": "202"})
					return
				}
				reply(w, map[string]any{"objective_id": "201"})
			}))
			input := filepath.Join(f.WorkDir, "batch.json")
			if e := os.WriteFile(input, []byte(`[{"text":"目标","notes":"说明","category_id":"901","krs":[{"text":"结果"}]}]`), 0600); e != nil {
				t.Fatal(e)
			}
			r := f.Run("okr", "+batch-create", "--cycle-id", "101", "--input", "@"+input)
			if mode == "success" {
				success(t, r, `"202"`)
				if deletes.Load() != 0 {
					t.Fatal("unexpected rollback")
				}
			} else {
				if r.ExitCode == 0 || !strings.Contains(r.Stderr, `"mutation"`) {
					t.Fatalf("%+v", r)
				}
				if deletes.Load() != 1 {
					t.Fatalf("rollback count %d", deletes.Load())
				}
			}
			if creates.Load() != 2 {
				t.Fatalf("unsafe replay %d", creates.Load())
			}
		})
	}
}

func TestOKRUploadImageMultipart(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/open-apis/okr/v1/images/upload" || r.Method != "POST" {
			t.Errorf("unexpected %s %s", r.Method, r.URL)
		}
		if e := r.ParseMultipartForm(1024 * 1024); e != nil {
			t.Error(e)
			http.Error(w, "invalid multipart", http.StatusBadRequest)
			return
		}
		if r.FormValue("target_id") != "201" {
			t.Error("target lost")
		}
		file, _, e := r.FormFile("data")
		if e != nil {
			t.Error(e)
		} else {
			defer file.Close()
			b, _ := io.ReadAll(file)
			if string(b) != "synthetic image bytes" {
				t.Error("image corrupted")
			}
		}
		reply(w, map[string]any{"file_token": "img_synthetic", "url": "https://example.test/image"})
	}))
	file := filepath.Join(f.WorkDir, "fixture.png")
	if e := os.WriteFile(file, []byte("synthetic image bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	r := f.Run("okr", "+upload-image", "--file", file, "--target-id", "201", "--target-type", "key_result")
	success(t, r, `"file_token":"img_synthetic"`)
	if !strings.Contains(r.Stderr, "Uploading:") {
		t.Fatal("progress absent from stderr")
	}
}

func TestOKRPaginationFailurePreventsReplacement(t *testing.T) {
	for _, response := range []string{`{"items":[{"id":"201"}],"has_more":true}`, `{"items":[{"id":"201"}],"has_more":true,"page_token":"loop"}`, `{"items":[{"id":9}],"has_more":false}`} {
		for _, command := range []string{"reorder", "cycle-detail", "comment-detail"} {
			t.Run(command+response, func(t *testing.T) {
				var calls, writes atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != "GET" {
						writes.Add(1)
					}
					var d any
					_ = json.Unmarshal([]byte(response), &d)
					reply(w, d)
				}))
				args := []string{"okr", "+" + command, "--cycle-id", "101"}
				if command == "reorder" {
					args = append(args, "--level", "objective", "--ops", `[{"id":"201","position":1}]`)
				}
				r := f.Run(args...)
				if r.ExitCode == 0 || r.Stdout != "" || !strings.Contains(r.Stderr, `"invalid_response"`) {
					t.Fatalf("%+v", r)
				}
				if calls.Load() > 2 || writes.Load() != 0 {
					t.Fatalf("calls %d writes %d", calls.Load(), writes.Load())
				}
			})
		}
	}
}

func TestOKRInvalidInputAndIdentityNeverDispatch(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{{"+create", "--level", "objective", "--cycle-id", "101", "--content", `{"text":"x"} {"text":"y"}`}, {"+comment-create", "--target-id", "201", "--target-type", "objective", "--content", `{"text":"x"}`}, {"+progress-update", "--progress-id", "301", "--content", `{"text":"x"}`, "--progress-status", "done"}, {"+comment-delete", "--comment-id", "401", "--as", "bot", "--yes"}} {
		r := f.Run(append([]string{"okr"}, args...)...)
		if r.ExitCode == 0 || r.Stdout != "" || !strings.Contains(r.Stderr, `"error"`) {
			t.Fatalf("%+v", r)
		}
	}
}

func TestOKRLivePrivateLifecycle(t *testing.T) {
	t.Skip("Live blocked: no verified current-account/tenant private OKR cycle, non-notifying mutation semantics, and run-owned cleanup target. PLAN forbids using existing unrelated objectives; inherited live scenarios remain excluded.")
}

func TestOKRRichContentFileAndArtifact(t *testing.T) {
	const rich = `{"blocks":[{"block_element_type":"paragraph","paragraph":{"style":{"list":{"list_type":"number","indent_level":0,"number":1}},"elements":[{"paragraph_element_type":"textRun","text_run":{"text":"中文 😀\nsecond","style":{"bold":false,"strike_through":true,"back_color":{"red":0,"green":128,"blue":255,"alpha":0.5},"text_color":{"red":255,"green":0,"blue":0,"alpha":1},"link":{"url":"https://example.test/link"}}}},{"paragraph_element_type":"mention","mention":{"user_id":"ou_synthetic"}},{"paragraph_element_type":"docsLink","docs_link":{"url":"https://example.test/doc","title":"文档"}}]}},{"block_element_type":"gallery","gallery":{"images":[{"file_token":"img_synthetic","src":"https://example.test/image","width":12.5,"height":20}]}}]}`
	var content any
	_ = json.Unmarshal([]byte(rich), &content)
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			writes.Add(1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			a, _ := json.Marshal(body["content"])
			b, _ := json.Marshal(content)
			if string(a) != string(b) {
				t.Errorf("rich input changed: %s", a)
			}
			reply(w, map[string]any{"objective_id": "201"})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/key_results") {
			reply(w, map[string]any{"items": []any{}, "has_more": false})
			return
		}
		reply(w, map[string]any{"items": []any{map[string]any{"id": "201", "content": content, "notes": content, "score": 0, "category_id": "901", "position": 1}}, "has_more": false})
	}))
	input := filepath.Join(f.WorkDir, "rich.json")
	if err := os.WriteFile(input, []byte(rich), 0600); err != nil {
		t.Fatal(err)
	}
	success(t, f.Run("okr", "+create", "--level", "objective", "--cycle-id", "101", "--style", "richtext", "--content", "@"+input), `"201"`)
	artifact := filepath.Join(f.WorkDir, "cycle.json")
	success(t, f.Run("okr", "+cycle-detail", "--cycle-id", "101", "--style", "richtext", "--save-result", artifact), `"artifact"`)
	b, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Result struct {
			Data struct {
				Objectives []struct {
					Content string `json:"content"`
				} `json:"objectives"`
			} `json:"data"`
		} `json:"result"`
	}
	if err = json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Result.Data.Objectives) != 1 {
		t.Fatalf("artifact lost objectives: %s", b)
	}
	var got any
	if err = json.Unmarshal([]byte(saved.Result.Data.Objectives[0].Content), &got); err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	expected, _ := json.Marshal(content)
	if string(a) != string(expected) {
		t.Fatalf("rich artifact changed: %s", a)
	}
	r := f.Run("okr", "+cycle-detail", "--cycle-id", "101", "--save-result", artifact)
	if r.ExitCode == 0 {
		t.Fatal("artifact overwritten")
	}
	if writes.Load() != 1 {
		t.Fatalf("writes %d", writes.Load())
	}
}

func TestOKRCompletePaginationKeepsOpaqueToken(t *testing.T) {
	var pages atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/key_results") {
			reply(w, map[string]any{"items": []any{}, "has_more": false})
			return
		}
		n := pages.Add(1)
		token := r.URL.Query().Get("page_token")
		if n == 1 {
			if token != "" {
				t.Error("unexpected initial continuation")
			}
			reply(w, map[string]any{"items": []any{map[string]any{"id": "201"}}, "has_more": true, "page_token": "opaque+/="})
		} else {
			if token != "opaque+/=" {
				t.Errorf("token changed %q", token)
			}
			reply(w, map[string]any{"items": []any{map[string]any{"id": "202"}}, "has_more": false})
		}
	}))
	r := f.Run("okr", "+cycle-detail", "--cycle-id", "101")
	success(t, r, `"201"`)
	if !strings.Contains(r.Stdout, `"202"`) || pages.Load() != 2 {
		t.Fatalf("incomplete %s", r.Stdout)
	}
}

func TestOKRReplacementRejectsMalformedNativeIDs(t *testing.T) {
	for _, command := range []string{"reorder", "weight"} {
		for _, level := range []string{"objective", "key-result"} {
			for _, kind := range []string{"duplicate", "duplicate_across_pages", "missing", "nonnumeric", "nonpositive", "overflow"} {
				t.Run(command+"/"+level+"/"+kind, func(t *testing.T) {
					var reads, writes atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != "GET" {
							writes.Add(1)
							reply(w, map[string]any{})
							return
						}
						reads.Add(1)
						native := map[string]any{"id": "1", "position": 1, "weight": 0.5}
						bad := map[string]any{"id": "1", "position": 2, "weight": 0.5}
						switch kind {
						case "missing":
							delete(bad, "id")
						case "nonnumeric":
							bad["id"] = "invalid"
						case "nonpositive":
							bad["id"] = "0"
						case "overflow":
							bad["id"] = "9223372036854775808"
						}
						if kind == "duplicate_across_pages" {
							if r.URL.Query().Get("page_token") == "" {
								reply(w, map[string]any{"items": []any{native}, "has_more": true, "page_token": "next-native-page"})
								return
							}
							reply(w, map[string]any{"items": []any{bad}, "has_more": false})
							return
						}
						reply(w, map[string]any{"items": []any{native, bad}, "has_more": false})
					}))
					args := []string{"okr", "+" + command, "--level", level, "--cycle-id", "123"}
					if level == "key-result" {
						args = append(args, "--objective-id", "456")
					}
					if command == "reorder" {
						args = append(args, "--ops", `[{"id":"1","position":1}]`)
					} else {
						args = append(args, "--weights", `[{"id":"1","weight":1}]`)
					}
					result := f.Run(args...)
					var envelope struct {
						Error struct {
							Type    string `json:"type"`
							Subtype string `json:"subtype"`
							Hint    string `json:"hint"`
						} `json:"error"`
					}
					if err := json.Unmarshal([]byte(result.Stderr), &envelope); err != nil {
						t.Fatalf("expected typed JSON, not panic: %+v", result)
					}
					if result.ExitCode != 5 || result.Stdout != "" || envelope.Error.Type != "internal" || envelope.Error.Subtype != "invalid_response" || envelope.Error.Hint == "" {
						t.Fatalf("unsafe malformed snapshot result: %+v", result)
					}
					want := int32(1)
					if kind == "duplicate_across_pages" {
						want = 2
					}
					if writes.Load() != 0 || reads.Load() != want {
						t.Fatalf("reads=%d writes=%d", reads.Load(), writes.Load())
					}
				})
			}
		}
	}
}
