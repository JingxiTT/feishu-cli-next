// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

func run(f *nextfixture.Fixture, args ...string) nextfixture.Result {
	return f.Run(append([]string{"apps", "--as", "user"}, args...)...)
}
func expect(t *testing.T, r nextfixture.Result, exit int, key, value string) {
	t.Helper()
	if r.ExitCode != exit || key != "" && gjson.Get(r.Stdout, key).String() != value {
		t.Fatalf("want exit=%d %s=%s; got %+v", exit, key, value, r)
	}
}
func reply(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": data})
}
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

// Each expected request was captured from an isolated build of frozen 7fd6ef3,
// not generated from the implementation under test. This covers every API
// shortcut's process/validation/dry-run routing, alongside the inherited units.
func TestAppsFrozenRequestMapping(t *testing.T) {
	bytes, err := os.ReadFile("testdata/reference_dryrun.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name string
		Args []string
		APIs []map[string]interface{} `json:"apis"`
	}
	if err = json.Unmarshal(bytes, &cases); err != nil {
		t.Fatal(err)
	}
	covered := map[string]bool{"apps +git-credential-list": true, "apps +git-credential-remove": true, "apps +plugin-list": true, "apps +plugin-uninstall": true, "apps git-credential-helper": true}
	f := nextfixture.NewFixture(t, nil)
	writeFile(t, filepath.Join(f.WorkDir, "fixture.csv"), "id,name\n1,fixture\n")
	writeFile(t, filepath.Join(f.WorkDir, "fixture.html"), "<html><body>fixture</body></html>")
	writeFile(t, filepath.Join(f.WorkDir, "package.json"), `{"name":"fixture","actionPlugins":{}}`)
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			covered["apps "+tc.Name] = true
			result := run(f, append(tc.Args, "--dry-run")...)
			expect(t, result, 0, "dry_run", "true")
			var actual struct {
				Data struct {
					API []map[string]interface{} `json:"api"`
				}
			}
			if err := json.Unmarshal([]byte(result.Stdout), &actual); err != nil {
				t.Fatal(err)
			}
			for _, api := range actual.Data.API {
				for k := range api {
					if k != "method" && k != "url" && k != "body" && k != "params" {
						delete(api, k)
					}
				}
			}
			if !reflect.DeepEqual(actual.Data.API, tc.APIs) {
				t.Fatalf("request drift\nactual=%#v\nreference=%#v", actual.Data.API, tc.APIs)
			}
		})
	}
	file, err := os.Open("../../../docs/reference/registrations.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	var frozen map[string]struct{ Commands map[string]json.RawMessage }
	if err = json.NewDecoder(compressed).Decode(&frozen); err != nil {
		t.Fatal(err)
	}
	for command := range frozen["feishu"].Commands {
		if strings.HasPrefix(command, "apps ") && !covered[command] {
			t.Errorf("unmapped frozen command %s", command)
		}
	}
	if len(covered) != 97 {
		t.Fatalf("mapped %d, want 97 frozen leaves", len(covered))
	}
}

func TestAppsMetadataReplacementAndUnknown(t *testing.T) {
	for _, mode := range []string{"same", "change", "missing", "wrong-target", "denied", "lost"} {
		t.Run(mode, func(t *testing.T) {
			var writes atomic.Int32
			state := map[string]interface{}{"app_id": "app_fixture", "name": "old", "description": "preserve 😀", "extra": nil}
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/open-apis/spark/v1/apps/app_fixture" {
					t.Errorf("wrong target: %s", r.URL.Path)
				}
				if r.Method == "GET" {
					if mode == "denied" {
						w.WriteHeader(403)
						_, _ = io.WriteString(w, `{"code":99991679,"msg":"fixture missing scope"}`)
						return
					}
					if mode == "wrong-target" {
						reply(w, map[string]interface{}{"app": map[string]interface{}{"app_id": "app_other", "name": "new"}})
						return
					}
					if mode == "missing" {
						reply(w, map[string]interface{}{})
						return
					}
					reply(w, map[string]interface{}{"app": state})
					return
				}
				if r.Method != "PATCH" {
					t.Errorf("wrong method %s", r.Method)
				}
				writes.Add(1)
				var patch map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&patch)
				if len(patch) != 1 || patch["name"] != "new" {
					t.Errorf("unrequested fields: %#v", patch)
				}
				state["name"] = patch["name"]
				if mode == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
					return
				}
				reply(w, map[string]interface{}{"app": state})
			}))
			name := "new"
			if mode == "same" {
				name = "old"
			}
			result := run(f, "+update", "--app-id", "app_fixture", "--name", name)
			switch mode {
			case "same":
				expect(t, result, 0, "data.app.description", "preserve 😀")
				if writes.Load() != 0 || strings.Contains(result.Stderr, "receipts") {
					t.Fatalf("no-op wrote: %+v", result)
				}
			case "change":
				expect(t, result, 0, "data.app.name", "new")
				if writes.Load() != 1 {
					t.Fatal("missing mutation")
				}
			case "missing", "wrong-target", "denied":
				if result.ExitCode == 0 || writes.Load() != 0 {
					t.Fatalf("invalid preflight wrote: %+v", result)
				}
			case "lost":
				expect(t, result, 22, "", "")
				if writes.Load() != 1 || !strings.Contains(result.Stderr, `"outcome":"unknown"`) {
					t.Fatalf("unknown replayed/hidden: %+v", result)
				}
			}
			if state["description"] != "preserve 😀" {
				t.Fatal("unrelated description altered")
			}
		})
	}
}

func TestAppsAutomationPaginationAndArtifacts(t *testing.T) {
	for _, mode := range []string{"complete", "missing-cursor", "repeated-cursor", "failed-page"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				if n > 3 {
					t.Error("pagination runaway")
				}
				if !strings.HasSuffix(r.URL.Path, "/triggers") || r.Method != "GET" {
					t.Errorf("wrong request %s %s", r.Method, r.URL.Path)
				}
				if n == 2 && mode == "failed-page" {
					w.WriteHeader(403)
					_, _ = io.WriteString(w, `{"code":99991679,"msg":"denied"}`)
					return
				}
				more := n == 1 || mode == "repeated-cursor"
				cursor := "next"
				if mode == "missing-cursor" {
					cursor = ""
				}
				reply(w, map[string]interface{}{"items": []interface{}{map[string]interface{}{"name": "trigger-" + string('0'+n), "trigger_type": "webhook", "trigger_condition": map[string]interface{}{"token_value": "synthetic-sensitive-token", "token_enabled": true}}}, "has_more": more, "page_token": cursor})
			}))
			path := filepath.Join(f.WorkDir, "triggers.json")
			result := run(f, "+automation-list", "--app-id", "app_fixture", "--all", "--save-result", path)
			if mode != "complete" {
				if result.ExitCode == 0 {
					t.Fatalf("truncated success %+v", result)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatal("incomplete artifact committed")
				}
				return
			}
			expect(t, result, 0, "artifact.complete", "true")
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "synthetic-sensitive-token") || gjson.GetBytes(b, "result.data.items.#").Int() != 2 {
				t.Fatalf("artifact integrity: %s", b)
			}
			if strings.Contains(result.Stderr, "mutation") {
				t.Fatal("read journaled")
			}
		})
	}
}

func TestAppsSQLTypePreservationAndFailures(t *testing.T) {
	for _, mode := range []string{"types", "malformed", "partial", "lost"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/sql_commands") || r.URL.Query().Get("transactional") != "false" {
					t.Errorf("SQL routing %s %s", r.Method, r.URL)
				}
				if mode == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err == nil {
						_ = conn.Close()
					}
					return
				}
				result := `[{"sql_type":"SELECT","data":"[{\"id\":9007199254740993,\"decimal\":123.1234567890123456789,\"s\":\"001\",\"e\":\"\",\"n\":null,\"b\":false}]"}]`
				if mode == "malformed" {
					result = `[{"sql_type":"SELECT","data":"{bad}"}]`
				}
				if mode == "partial" {
					result = `[{"sql_type":"INSERT","data":"","affected_rows":1},{"sql_type":"ERROR","data":"{\"code\":1300002,\"message\":\"fixture SQL failure\"}"}]`
				}
				reply(w, map[string]interface{}{"result": result})
			}))
			result := run(f, "+db-execute", "--app-id", "app_fixture", "--sql", "SELECT fixture", "--yes")
			if calls.Load() != 1 {
				t.Fatalf("SQL replay count %d", calls.Load())
			}
			if mode == "types" {
				if result.ExitCode != 0 && result.ExitCode != 22 {
					t.Fatalf("SQL read failed %+v", result)
				}
				if !strings.Contains(result.Stdout, `9007199254740993`) || !strings.Contains(result.Stdout, `123.1234567890123456789`) || !strings.Contains(result.Stdout, `"s":"001"`) {
					t.Fatalf("SQL precision/type loss: %+v", result)
				}
				return
			}
			if mode == "lost" {
				expect(t, result, 22, "", "")
				return
			}
			if result.ExitCode == 0 {
				t.Fatalf("SQL false success %+v", result)
			}
			if mode == "partial" && strings.Contains(result.Stderr, `"outcome":"confirmed_applied"`) {
				t.Fatalf("partial SQL claimed applied %+v", result)
			}
		})
	}
}

// Exercise the real execution/auth/transport path for every API shortcut using
// a rejected synthetic backend. Unlike --dry-run this independently observes
// dispatch and verifies that failures stop the workflow without duplicate writes.
// Success/composition semantics have dedicated stateful cases and domain units.
func TestAppsEveryAPIShortcutRejectionPath(t *testing.T) {
	bytes, err := os.ReadFile("testdata/reference_dryrun.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name string
		Args []string
		APIs []map[string]interface{} `json:"apis"`
	}
	if err = json.Unmarshal(bytes, &cases); err != nil {
		t.Fatal(err)
	}
	metadata := frozenAppsMetadata(t)
	for _, tc := range cases {
		if len(tc.APIs) == 0 {
			continue
		} // +init is a local multi-tool orchestration.
		t.Run(tc.Name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				path := tc.APIs[0]["url"].(string)
				path = strings.ReplaceAll(path, ":app_id", "app_fixture")
				if tc.Name == "+html-publish" && r.Method == "GET" && r.URL.Path == "/open-apis/spark/v1/apps/app_fixture" {
					// Publishing verifies the app type before obtaining upload credentials.
				} else if tc.Name == "+update" && r.Method == "GET" && r.URL.Path == path {
					// Agent replacement compares the existing app before patching.
				} else if tc.Name == "+db-data-export" && strings.HasSuffix(r.URL.Path, "/records") { // read-only optional extent probe
					if r.Method != "GET" {
						t.Errorf("export extent probe writes")
					}
				} else {
					if r.Method != tc.APIs[0]["method"] || r.URL.Path != path {
						t.Errorf("execution differs from frozen request: %s %s; want %v %s", r.Method, r.URL.Path, tc.APIs[0]["method"], path)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(403)
				_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic scope rejection"}`)
			}))
			writeFile(t, filepath.Join(f.WorkDir, "fixture.csv"), "id,name\n1,fixture\n")
			writeFile(t, filepath.Join(f.WorkDir, "fixture.html"), "<html><body>fixture</body></html>")
			writeFile(t, filepath.Join(f.WorkDir, "package.json"), `{"name":"fixture","actionPlugins":{}}`)
			args := append([]string{}, tc.Args...)
			// Frozen metadata determines which commands actually register confirmation.
			for _, sc := range metadata {
				if sc.Name == tc.Name && sc.Confirmation {
					args = append(args, "--yes")
				}
			}
			r := run(f, args...)
			if r.ExitCode == 0 || calls.Load() == 0 {
				t.Fatalf("API rejection not exercised: calls=%d result=%+v", calls.Load(), r)
			}
			maxCalls := int32(1)
			if tc.Name == "+db-data-export" || tc.Name == "+db-env-migrate" {
				maxCalls = 2
			}
			if calls.Load() > maxCalls {
				t.Fatalf("request replayed %d times: %+v", calls.Load(), r)
			}
			if strings.Contains(r.Stdout, `"ok":true`) {
				t.Fatalf("rejection claimed success: %+v", r)
			}
		})
	}
}

type frozenAppMetadata struct {
	Name         string
	Confirmation bool
}

func frozenAppsMetadata(t *testing.T) []frozenAppMetadata {
	t.Helper()
	file, err := os.Open("../../../docs/reference/registrations.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	var frozen map[string]struct {
		Commands map[string]struct{ Flags map[string]json.RawMessage }
	}
	if err = json.NewDecoder(compressed).Decode(&frozen); err != nil {
		t.Fatal(err)
	}
	var out []frozenAppMetadata
	for name, meta := range frozen["feishu"].Commands {
		if strings.HasPrefix(name, "apps +") {
			_, yes := meta.Flags["yes"]
			out = append(out, frozenAppMetadata{strings.TrimPrefix(name, "apps "), yes})
		}
	}
	return out
}

func TestAppsListPreservesFullAppMetadata(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply(w, map[string]interface{}{"items": []interface{}{map[string]interface{}{"app_id": "app_fixture", "name": "Journal", "owner": "self", "created_at": "001", "icon_url": "https://example.invalid/icon.png", "extra": nil}}, "has_more": false})
	}))
	result := run(f, "+list", "--keyword", "Journal", "--ownership", "mine")
	expect(t, result, 0, "data.items.0.created_at", "001")
	expect(t, result, 0, "data.items.0.icon_url", "https://example.invalid/icon.png")
	expect(t, result, 0, "data.items.0.owner", "self")
}
