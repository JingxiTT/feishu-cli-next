// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package whiteboard_test

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
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const board = "synthetic-whiteboard"
const nodes = `[{"id":"001","type":"shape","x":9007199254740993,"y":1.234567890123456789,"text":{"text":"你好 😀","bold":true},"image":{"token":"fixture-image"},"children":[],"empty":"","null":null,"unknown":{"flag":false}}]`

func decode(t *testing.T, input string) map[string]any {
	t.Helper()
	var out map[string]any
	d := json.NewDecoder(strings.NewReader(input))
	d.UseNumber()
	if err := d.Decode(&out); err != nil {
		t.Fatalf("decode: %v: %s", err, input)
	}
	return out
}
func success(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("CLI failed: %+v", r)
	}
	return decode(t, r.Stdout)
}
func reply(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"code":0,"data":`+data+`}`)
}
func run(f *nextfixture.Fixture, command string, args ...string) nextfixture.Result {
	return f.Run(append([]string{"whiteboard", command, "--whiteboard-token", board, "--as", "user"}, args...)...)
}
func readBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	return decode(t, string(data))
}
func equal(t *testing.T, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestUpdateEveryFormatAliasAndIdentity(t *testing.T) {
	for _, domain := range []string{"whiteboard", "docs"} {
		for _, identity := range []string{"user", "bot"} {
			for _, format := range []string{"raw", "plantuml", "mermaid", "svg"} {
				t.Run(domain+"/"+identity+"/"+format, func(t *testing.T) {
					var calls atomic.Int32
					source := map[string]string{"raw": `{"nodes":` + nodes + `}`, "plantuml": "@startuml\nA -> B: 你好\n@enduml", "mermaid": "graph TD\nA[你好]-->B", "svg": `<svg xmlns="http://www.w3.org/2000/svg"><text>你好</text></svg>`}[format]
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						equal(t, r.Method, "POST")
						equal(t, r.URL.Query().Get("client_token"), "synthetic-idempotent-token")
						b := readBody(t, r)
						equal(t, b["overwrite"], true)
						if format == "raw" {
							equal(t, r.URL.Path, "/open-apis/board/v1/whiteboards/"+board+"/nodes")
							equal(t, b["nodes"], decode(t, `{"nodes":`+nodes+`}`)["nodes"])
							reply(w, `{"ids":["created-001"]}`)
						} else {
							equal(t, r.URL.Path, "/open-apis/board/v1/whiteboards/"+board+"/nodes/plantuml")
							equal(t, b["plant_uml_code"], source)
							equal(t, b["parse_mode"], json.Number("1"))
							equal(t, b["syntax_type"], json.Number(map[string]string{"plantuml": "1", "mermaid": "2", "svg": "3"}[format]))
							reply(w, `{"node_id":"created-001"}`)
						}
					}))
					if err := os.WriteFile(filepath.Join(f.WorkDir, "source.txt"), []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
					command := "+update"
					if domain == "docs" {
						command = "+whiteboard-update"
					}
					args := []string{domain, command, "--whiteboard-token", board, "--as", identity, "--source", "@source.txt", "--input_format", format, "--overwrite", "--idempotent-token", "synthetic-idempotent-token"}
					dry := success(t, f.Run(append(args, "--dry-run")...))
					api := dry["data"].(map[string]any)["api"].([]any)[0].(map[string]any)
					equal(t, api["params"].(map[string]any)["client_token"], "synthetic-idempotent-token")
					equal(t, calls.Load(), int32(0))
					result := success(t, f.Run(args...))
					equal(t, result["identity"], identity)
					if !strings.Contains(f.Run("receipts", "list").Stdout, "confirmed_applied") {
						t.Fatal("missing applied receipt")
					}
					equal(t, calls.Load(), int32(1))
				})
			}
		}
	}
}

func TestRawExportImportAndExplicitClear(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		equal(t, r.URL.Path, "/open-apis/board/v1/whiteboards/"+board+"/nodes")
		if r.Method == "GET" {
			reply(w, `{"nodes":`+nodes+`}`)
			return
		}
		b := readBody(t, r)
		writes.Add(1)
		if writes.Load() <= 2 {
			equal(t, b["nodes"], decode(t, `{"nodes":`+nodes+`}`)["nodes"])
			if _, present := b["overwrite"]; present {
				t.Error("append unexpectedly overwrites")
			}
		} else {
			equal(t, b["nodes"], []any{})
			equal(t, b["overwrite"], true)
		}
		reply(w, `{"ids":[]}`)
	}))
	for _, alias := range []bool{false, true} {
		var result nextfixture.Result
		if alias {
			result = run(f, "+query", "--output_as", "raw", "--output", "nodes.json", "--overwrite")
		} else {
			result = run(f, "+export", "--output-type", "raw", "--output", "nodes.json")
		}
		success(t, result)
		saved, err := os.ReadFile(filepath.Join(f.WorkDir, "nodes.json"))
		if err != nil {
			t.Fatal(err)
		}
		equal(t, decode(t, string(saved)), decode(t, `{"nodes":`+nodes+`}`))
		success(t, run(f, "+update", "--source", "@nodes.json"))
	}
	success(t, run(f, "+update", "--source", `{"nodes":[]}`, "--overwrite"))
	equal(t, writes.Load(), int32(3))
	artifact := filepath.Join(f.WorkDir, "result.json")
	success(t, run(f, "+export", "--output-type", "raw", "--save-result", artifact, "--result-fields", "id"))
	saved, err := os.ReadFile(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "9007199254740993") {
		t.Fatal("artifact lost precision")
	}
	rejected := run(f, "+update", "--source", "@result.json", "--overwrite")
	equal(t, rejected.ExitCode, 2)
	equal(t, writes.Load(), int32(3))
}

func TestSourceExportEverySyntaxAndAmbiguity(t *testing.T) {
	for _, test := range []struct{ name, payload, expected, extension string }{
		{"mermaid", `{"nodes":[{"syntax":{"code":"graph TD\nA-->B","syntax_type":2}}]}`, "graph TD\nA-->B", ".mmd"},
		{"plantuml", `{"nodes":[{"syntax":{"code":"@startuml\nA -> B\n@enduml","syntax_type":1}}]}`, "@startuml\nA -> B\n@enduml", ".puml"},
		{"empty", `{}`, "whiteboard is empty", ""},
		{"no-code", `{"nodes":[{"syntax":{"code":"unsupported","syntax_type":3}},{"text":"shape"}]}`, "no code blocks found", ""},
		{"ambiguous", `{"nodes":[{"syntax":{"code":"a","syntax_type":1}},{"syntax":{"code":"b","syntax_type":2}}]}`, "multiple code blocks found", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { equal(t, r.Method, "GET"); reply(w, test.payload) }))
			data := success(t, run(f, "+export", "--output-type", "source"))["data"].(map[string]any)
			if test.extension == "" {
				if !strings.Contains(data["msg"].(string), test.expected) {
					t.Fatal(data)
				}
				return
			}
			equal(t, data["code"], test.expected)
			success(t, run(f, "+query", "--output_as", "code", "--output", "code"))
			saved, err := os.ReadFile(filepath.Join(f.WorkDir, "code"+test.extension))
			if err != nil {
				t.Fatal(err)
			}
			equal(t, string(saved), test.expected)
		})
	}
}

func TestRenderExportsAndCollisionProtection(t *testing.T) {
	const svg = `<svg><text>你好 😀</text></svg>`
	image := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 1, 2, 3}
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/export") {
			equal(t, r.Method, "POST")
			equal(t, readBody(t, r)["export_type"], "svg")
			reply(w, `{"content":"`+base64.StdEncoding.EncodeToString([]byte(svg))+`","mime_type":"image/svg+xml"}`)
			return
		}
		equal(t, r.Method, "GET")
		equal(t, r.URL.Path, "/open-apis/board/v1/whiteboards/"+board+"/download_as_image")
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(image)
	}))
	equal(t, success(t, run(f, "+export", "--output-type", "svg"))["data"].(map[string]any)["svg_content"], svg)
	success(t, run(f, "+query", "--output_as", "svg", "--output", "render"))
	saved, err := os.ReadFile(filepath.Join(f.WorkDir, "render.svg"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, string(saved), svg)
	success(t, run(f, "+export", "--output-type", "preview", "--output", "preview"))
	saved, err = os.ReadFile(filepath.Join(f.WorkDir, "preview.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	equal(t, saved, image)
	equal(t, run(f, "+query", "--output_as", "image", "--output", "preview").ExitCode, 2)
	success(t, run(f, "+query", "--output_as", "image", "--output", "preview", "--overwrite"))
	mismatch := run(f, "+export", "--output-type", "preview", "--output", "preview.png", "--overwrite")
	if mismatch.ExitCode == 0 || !strings.Contains(mismatch.Stderr, "failed_precondition") {
		t.Fatal(mismatch)
	}
	if _, err = os.Stat(filepath.Join(f.WorkDir, "preview.png")); !os.IsNotExist(err) {
		t.Fatal("mismatched file written")
	}
	if strings.Contains(f.Run("receipts", "list").Stdout, "confirmed_applied") {
		t.Fatal("read-only SVG export journaled as mutation")
	}
}

func TestValidationAndMutationFailureEvidence(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{
		{"--source", `{"nodes":[]} {}`}, {"--source", `{"nodes":[]}`, "--input_format", "bogus"}, {"--source", `{"nodes":[]}`, "--idempotent-token", "short"},
	} {
		r := run(f, "+update", args...)
		equal(t, r.ExitCode, 2)
		if !strings.Contains(r.Stderr, "validation") {
			t.Fatal(r)
		}
	}
	equal(t, run(f, "+export", "--output-type", "preview").ExitCode, 2)
	for _, test := range []struct {
		name, body, outcome string
		status, exit        int
	}{
		{"permission", `{"code":99991672,"msg":"scope required"}`, "confirmed_unapplied", 403, 20},
		{"malformed", `not-json`, "unknown", 200, 22},
		{"partial", `{"code":0,"data":{"ids":["one"],"failed_items":[{"id":"two","code":1}]}}`, "unknown", 200, 22},
		{"lost-response", "", "unknown", 0, 22},
	} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if test.status == 0 {
					connection, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = connection.Close()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			result := run(f, "+update", "--source", `{"nodes":`+nodes+`}`, "--overwrite")
			equal(t, result.ExitCode, test.exit)
			equal(t, calls.Load(), int32(1))
			if !strings.Contains(result.Stderr, test.outcome) {
				t.Fatal(result)
			}
			receipts := f.Run("receipts", "list")
			equal(t, receipts.ExitCode, 0)
			if !strings.Contains(receipts.Stdout, test.outcome) {
				t.Fatal(receipts)
			}
			equal(t, calls.Load(), int32(1))
		})
	}
}

func TestWhiteboardLivePrivateFixture(t *testing.T) {
	t.Skip("BLOCKED: no verified account/tenant-bound private whiteboard creation harness. Required scenario: create a private run-owned document and board after verifying creation visibility; import every format, read nodes/source/SVG/preview, verify identities and permissions after overwrite and explicit clear, and clean up only run-owned resources. No legacy external whiteboard token is accepted as proof of private run ownership.")
}

func TestRawAppendOverwriteClearObservedState(t *testing.T) {
	var mu sync.Mutex
	state := []any{}
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		equal(t, r.URL.Path, "/open-apis/board/v1/whiteboards/"+board+"/nodes")
		if r.Method == "GET" {
			encoded, err := json.Marshal(map[string]any{"nodes": state})
			if err != nil {
				t.Error(err)
				return
			}
			reply(w, string(encoded))
			return
		}
		writes.Add(1)
		b := readBody(t, r)
		incoming := b["nodes"].([]any)
		if b["overwrite"] == true {
			state = append([]any{}, incoming...)
		} else {
			state = append(state, incoming...)
		}
		reply(w, `{"ids":[]}`)
	}))
	observed := func(want int) {
		t.Helper()
		data := success(t, run(f, "+export", "--output-type", "raw"))["data"].(map[string]any)
		equal(t, len(data["nodes"].([]any)), want)
	}
	// Each append is a new effect. Equality must not suppress either invocation.
	success(t, run(f, "+update", "--source", `{"nodes":`+nodes+`}`))
	observed(1)
	success(t, run(f, "+update", "--source", `{"nodes":`+nodes+`}`))
	observed(2)
	success(t, run(f, "+update", "--source", `{"nodes":`+nodes+`}`, "--overwrite"))
	observed(1)
	// Explicit clear remains a dispatched permission-checked request even when empty.
	success(t, run(f, "+update", "--source", `{"nodes":[]}`, "--overwrite"))
	observed(0)
	success(t, run(f, "+update", "--source", `{"nodes":[]}`, "--overwrite"))
	observed(0)
	equal(t, writes.Load(), int32(5))
}
