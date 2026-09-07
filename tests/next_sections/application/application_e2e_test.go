// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const endpoint = "/open-apis/application/v7/app_slash_commands"

func run(f *nextfixture.Fixture, command string, flags ...string) nextfixture.Result {
	return f.Run(append([]string{"application", "+slash-command-" + command, "--as", "bot"}, flags...)...)
}

func expect(t *testing.T, r nextfixture.Result, exit int, key, value string) {
	t.Helper()
	if r.ExitCode != exit || (key != "" && gjson.Get(r.Stdout, key).String() != value) {
		t.Fatalf("want exit=%d %s=%s: %+v", exit, key, value, r)
	}
}

func TestApplicationLifecycleAndNoop(t *testing.T) {
	var writes atomic.Int32
	state := map[string]interface{}{}
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, endpoint) {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		var data interface{}
		switch r.Method {
		case "GET":
			items := []interface{}{}
			if len(state) > 0 {
				items = append(items, state)
			}
			data = map[string]interface{}{"items": items, "has_more": false}
		case "POST":
			writes.Add(1)
			if err := json.NewDecoder(r.Body).Decode(&state); err != nil {
				t.Error(err)
			}
			state["command_id"] = "cmd-001"
			state["create_time"] = "001"
			state["unknown_field"] = nil
			data = state
		case "PATCH":
			writes.Add(1)
			if r.URL.Path != endpoint+"/cmd-001" {
				t.Errorf("target changed: %s", r.URL.Path)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			for key, value := range body {
				state[key] = value
			}
			data = state
		case "DELETE":
			writes.Add(1)
			state = map[string]interface{}{}
			data = map[string]interface{}{}
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": data})
	}))
	created := run(f, "create", "--command", " greet ", "--description", "你好 😀", "--description-i18n", "en_us=Hi=there", "--icon-key", "skill_outlined")
	expect(t, created, 0, "data.action", "created")
	expect(t, run(f, "list"), 0, "data.items.0.description.i18n.en_us", "Hi=there")
	expect(t, run(f, "update", "--command", "greet", "--description", "你好 😀", "--description-i18n", "en_us=Hi=there"), 0, "data.action", "unchanged")
	if writes.Load() != 1 {
		t.Fatalf("no-op wrote: %d", writes.Load())
	}
	expect(t, run(f, "update", "--command-id", "cmd-001", "--icon-key", "new_icon"), 0, "data.description.i18n.en_us", "Hi=there")
	// A supplied description replaces its complete object, dropping omitted languages.
	expect(t, run(f, "update", "--command", "greet", "--description", "你好 😀"), 0, "data.action", "updated")
	listed := run(f, "list")
	expect(t, listed, 0, "data.items.0.command_id", "cmd-001")
	if gjson.Get(listed.Stdout, "data.items.0.description.i18n").Exists() {
		t.Fatalf("omitted language preserved by replacement: %+v", listed)
	}
	denied := run(f, "delete", "--command-id", "cmd-001")
	expect(t, denied, 10, "", "")
	if writes.Load() != 3 {
		t.Fatalf("confirmation gate wrote: %d", writes.Load())
	}
	expect(t, run(f, "delete", "--command", "greet", "--yes"), 0, "data.action", "deleted")
	expect(t, run(f, "list"), 0, "data.count", "0")
	if writes.Load() != 4 {
		t.Fatalf("writes=%d", writes.Load())
	}
}

func TestApplicationListArtifactsAndSensitiveFields(t *testing.T) {
	for _, sensitive := range []bool{false, true} {
		t.Run(map[bool]string{false: "lossless", true: "credential-rejected"}[sensitive], func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				item := map[string]interface{}{"command": "greet", "command_id": "cmd-001", "description": map[string]interface{}{"default_value": "你好", "i18n": map[string]string{"en_us": "Hi"}}, "extra": nil, "create_time": "001"}
				if sensitive {
					item["app_secret"] = "synthetic-secret"
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": []interface{}{item}, "has_more": true, "page_token": "next-001"}})
			}))
			path := filepath.Join(f.WorkDir, "commands.json")
			result := run(f, "list", "--save-result", path, "--result-fields", "command")
			if sensitive {
				if result.ExitCode == 0 {
					t.Fatalf("saved credentials: %+v", result)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("secret artifact exists: %v", err)
				}
				return
			}
			expect(t, result, 0, "artifact.binding.app_id", "synthetic-foundation-app")
			expect(t, result, 0, "artifact.complete", "false")
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if gjson.GetBytes(saved, "result.data.items.0.description.i18n.en_us").String() != "Hi" || gjson.GetBytes(saved, "result.data.page_token").String() != "next-001" {
				t.Fatalf("lossy artifact %s", saved)
			}
			if gjson.GetBytes(saved, "binding.identity").String() != "bot" {
				t.Fatalf("unbound artifact %s", saved)
			}
			if run(f, "list", "--save-result", path).ExitCode == 0 {
				t.Fatal("overwrote artifact")
			}
		})
	}
}

func TestApplicationValidationAndIdentityBeforeWrites(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	cases := [][]string{
		{"create", "--command", "/greet", "--description", "hi"},
		{"create", "--command", "greet", "--description", "hi", "--description-i18n", "en_us=a", "--description-i18n", "en_us=b"},
		{"update", "--command-id", "id", "--command", "greet", "--description", "x"},
		{"update", "--command-id", "id", "--description-i18n", "en_us=x"},
		{"delete", "--command-id", "id", "--command", "greet", "--yes"},
		{"create", "--command", "greet", "--description", "hi", "--save-result", "result.json"},
	}
	for _, args := range cases {
		r := run(f, args[0], args[1:]...)
		expect(t, r, 2, "", "")
	}
	bad := f.Run("application", "+slash-command-list", "--as", "invalid")
	expect(t, bad, 2, "", "")
}

func TestApplicationMutationFaultsNeverRetry(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		exit       int
	}{
		{"scope", `{"code":99991679,"msg":"synthetic missing scope"}`, 20},
		{"malformed", `{"broken":`, 22},
		{"partial-item", `{"code":0,"data":{"errors":[{"code":7}],"command_id":"cmd-1"}}`, 22},
		{"lost", "", 22},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				if tc.name == "lost" {
					conn, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = conn.Close()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			r := run(f, "create", "--command", "greet", "--description", "hi", "--force")
			expect(t, r, tc.exit, "", "")
			if writes.Load() != 1 {
				t.Fatalf("unsafe retry/resolution: %d %+v", writes.Load(), r)
			}
			if !strings.Contains(r.Stderr, "receipts") {
				t.Fatalf("missing receipt: %+v", r)
			}
			if tc.name == "partial-item" {
				expect(t, r, 22, "ok", "false")
				expect(t, r, 22, "data.errors.0.code", "7")
			}
		})
	}
}

func TestApplicationForceCollisionAndAmbiguity(t *testing.T) {
	for _, mode := range []string{"update", "noop", "ambiguous", "incomplete", "missing"} {
		t.Run(mode, func(t *testing.T) {
			var patches atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case "POST":
					_, _ = w.Write([]byte(`{"code":40000000,"msg":"command already exists"}`))
				case "GET":
					desc := "before"
					if mode == "noop" {
						desc = "after"
					}
					item := map[string]interface{}{"command": "greet", "command_id": "cmd-001", "description": map[string]string{"default_value": desc}}
					items := []interface{}{item}
					if mode == "ambiguous" {
						items = append(items, item)
					}
					if mode == "missing" {
						items = nil
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": items, "has_more": mode == "incomplete"}})
				case "PATCH":
					patches.Add(1)
					_, _ = w.Write([]byte(`{"code":0,"data":{"command_id":"cmd-001","command":"greet"}}`))
				}
			}))
			r := run(f, "create", "--command", "greet", "--description", "after", "--force")
			// The collision receipt remains conservatively unknown; a successful PATCH
			// does not erase that earlier evidence or authorize retrying the invocation.
			expect(t, r, 22, "", "")
			want := int32(0)
			if mode == "update" {
				want = 1
				expect(t, r, 22, "data.action", "updated")
			}
			if mode == "noop" {
				expect(t, r, 22, "data.action", "unchanged")
			}
			if patches.Load() != want {
				t.Fatalf("patches=%d want=%d", patches.Load(), want)
			}
		})
	}
}

func TestApplicationLivePrivateLifecycle(t *testing.T) {
	t.Skip("BLOCKED: app slash commands are visible to app users; current-account-only app installation/visibility and cleanable private app creation are not verified. No live credentials or writes are used. Required scenario: verify effective account/tenant and app visibility, create localized command, list, no-op update, replace description/icon, delete only the created command, verify removal and client cache behavior.")
}

func TestApplicationPublicHelpAndDryRun(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for command, flags := range map[string][]string{
		"list":   {},
		"create": {"--command", "--description", "--description-i18n", "--icon-key", "--force"},
		"update": {"--command-id", "--command", "--description", "--description-i18n", "--icon-key"},
		"delete": {"--command-id", "--command"},
	} {
		help := run(f, command, "--help")
		expect(t, help, 0, "", "")
		for _, flag := range flags {
			if !strings.Contains(help.Stdout, flag) {
				t.Fatalf("%s lost %s: %+v", command, flag, help)
			}
		}
	}
	for _, command := range []string{"update", "delete"} {
		flags := []string{"--command", "greet", "--dry-run"}
		if command == "update" {
			flags = append(flags, "--description", "new")
		}
		r := run(f, command, flags...)
		expect(t, r, 0, "data.api.0.method", "GET")
		expect(t, r, 0, "data.api.0.url", endpoint)
		expect(t, r, 0, "data.api.1.url", endpoint+"/<resolved_command_id>")
	}
	r := run(f, "create", "--command", "greet", "--description", "你好", "--description-i18n", "en_us=Hi=there", "--force", "--dry-run")
	expect(t, r, 0, "data.api.0.method", "POST")
	expect(t, r, 0, "data.api.0.body.description.i18n.en_us", "Hi=there")
}

func TestApplicationUserIdentityAndProjection(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"command":"greet","command_id":"cmd-1","description":{"default_value":"Hi"},"extra":"not-selected"}]}}`))
	}))
	r := f.Run("application", "+slash-command-list", "--as", "user", "--result-fields", "command")
	expect(t, r, 0, "identity", "user")
	expect(t, r, 0, "data.items.0.command_id", "cmd-1")
	if gjson.Get(r.Stdout, "data.items.0.extra").Exists() {
		t.Fatalf("projection ignored: %+v", r)
	}
	if f.IdentityRequests.Load() == 0 {
		t.Fatal("no user identity evidence")
	}
}
