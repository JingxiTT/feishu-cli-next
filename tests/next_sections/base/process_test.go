// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/shortcuts/base"
	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

type exchange struct{ method, path, query, body, data string }
type scenario struct {
	name     string
	args     []string
	requests []exchange
	output   string
}

const bp = "/open-apis/base/v3/bases/bas_fixture"
const tp = bp + "/tables/tbl_fixture"
const ap = "/open-apis/base/v3/base_apps/app_fixture"

func args(command string, flags ...string) []string {
	return append([]string{"base", "+" + command, "--as", "user"}, flags...)
}
func scoped(command string, flags ...string) []string {
	return args(command, append([]string{"--base-token", "bas_fixture"}, flags...)...)
}
func table(command string, flags ...string) []string {
	return scoped(command, append([]string{"--table-id", "tbl_fixture"}, flags...)...)
}
func view(command string, flags ...string) []string {
	return table(command, append([]string{"--view-id", "vew_fixture"}, flags...)...)
}
func app(command string, flags ...string) []string {
	return args(command, append([]string{"--app-token", "app_fixture"}, flags...)...)
}
func page(command string, flags ...string) []string {
	return app(command, append([]string{"--page-id", "pag_fixture"}, flags...)...)
}
func dash(command string, flags ...string) []string {
	return scoped(command, append([]string{"--dashboard-id", "dsh_fixture"}, flags...)...)
}
func ex(method, path, query, body, data string) exchange {
	return exchange{method, path, query, body, data}
}
func one(a []string, e exchange, out string) scenario {
	return scenario{strings.TrimPrefix(a[1], "+"), a, []exchange{e}, out}
}

// Expectations are authored independently of DryRun/Execute. Every expected business
// request is checked for method, path, complete query and complete JSON body.
func runScenario(t *testing.T, s scenario) {
	t.Helper()
	var mu sync.Mutex
	calls := 0
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		i := calls
		calls++
		if i >= len(s.requests) {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
			http.Error(w, "unexpected request", 500)
			return
		}
		want := s.requests[i]
		if r.Method != want.method || r.URL.Path != want.path {
			t.Errorf("request %d: %s %s, want %s %s", i, r.Method, r.URL.Path, want.method, want.path)
		}
		query, err := url.ParseQuery(want.query)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(r.URL.Query(), query) {
			t.Errorf("request %d query=%v want=%v", i, r.URL.Query(), query)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if want.body == "" {
			if len(raw) != 0 {
				t.Errorf("request %d unwanted body %s", i, raw)
			}
		} else {
			assertJSON(t, string(raw), want.body, false)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("fixture received raw credentials")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"code":0,"data":%s}`, want.data)
	}))
	result := fixture.Run(s.args...)
	wantExit := 0
	switch s.args[1] {
	case "+record-batch-update", "+record-delete":
		// Generic batch success lacks item reconciliation. Retain the result and
		// report unknown honestly, without replaying writes.
		if !strings.Contains(strings.Join(s.args, " "), "--skip-unchanged") {
			wantExit = 22
		}
	}
	if result.ExitCode == 22 && !strings.Contains(result.Stderr, `"unknown":1`) {
		t.Errorf("missing unknown batch receipt: %s", result.Stderr)
	}
	if result.ExitCode != wantExit {
		t.Fatalf("exit=%d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	mu.Lock()
	count := calls
	mu.Unlock()
	if count != len(s.requests) {
		t.Fatalf("business requests=%d want=%d", count, len(s.requests))
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil {
		t.Fatalf("output JSON: %v: %s", err, result.Stdout)
	}
	if raw, exists := envelope["ok"]; exists {
		wantOK := "true"
		if wantExit != 0 {
			wantOK = "false"
		}
		if string(raw) != wantOK {
			t.Errorf("ok=%s for exit=%d", raw, result.ExitCode)
		}
	}
	if raw, exists := envelope["identity"]; exists {
		identity := "user"
		for i, arg := range s.args {
			if arg == "--as" && i+1 < len(s.args) {
				identity = s.args[i+1]
			}
		}
		if string(raw) != `"`+identity+`"` {
			t.Errorf("identity=%s want=%s", raw, identity)
		}
	}
	assertJSON(t, result.Stdout, s.output, true)
}

func assertJSON(t *testing.T, raw, expected string, subset bool) {
	t.Helper()
	decode := func(s string) any {
		d := json.NewDecoder(strings.NewReader(s))
		d.UseNumber()
		var v any
		if err := d.Decode(&v); err != nil {
			t.Fatalf("decode JSON: %v: %s", err, s)
		}
		var extra any
		if err := d.Decode(&extra); err != io.EOF {
			t.Fatalf("trailing JSON: %s", s)
		}
		return v
	}
	got, want := decode(raw), decode(expected)
	if subset { // CLI success envelopes retain the domain result under data.
		if obj, ok := got.(map[string]any); ok {
			if _, envelope := obj["ok"].(bool); envelope {
				if data, exists := obj["data"]; exists {
					got = data
				}
			}
		}
		if !containsJSON(got, want) {
			t.Errorf("output=%s does not contain %s", raw, expected)
		}
	} else if !reflect.DeepEqual(got, want) {
		t.Errorf("JSON=%s want=%s", raw, expected)
	}
}
func containsJSON(got, want any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for key, value := range w {
			v, exists := g[key]
			if !exists || !containsJSON(v, value) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !containsJSON(g[i], w[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(got, want)
	}
}

func TestBaseProcessWorkflows(t *testing.T) {
	for _, s := range scenarios() {
		t.Run(s.name, func(t *testing.T) { runScenario(t, s) })
	}
}

func TestEveryRegisteredBaseShortcutHasProcessScenario(t *testing.T) {
	covered := map[string]bool{}
	for _, s := range scenarios() {
		covered[s.args[1]] = true
	}
	// Function references keep the dedicated test mapping checked by the compiler.
	for command := range map[string]func(*testing.T){
		"+record-upload-attachment":   TestBaseAttachmentUploadDownloadRemove,
		"+record-download-attachment": TestBaseAttachmentUploadDownloadRemove,
		"+record-remove-attachment":   TestBaseAttachmentUploadDownloadRemove,
		"+record-import":              TestRecordImportTenThousandRowsPreservesTypedValuesOutsideStdout,
	} {
		covered[command] = true
	}
	for _, shortcut := range base.Shortcuts() {
		if !covered[shortcut.Command] {
			t.Errorf("registered shortcut lacks an executing process scenario: %s", shortcut.Command)
		}
	}
}
