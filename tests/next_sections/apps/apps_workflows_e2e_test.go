// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"archive/tar"
	"bytes"
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
	"github.com/tidwall/gjson"
)

func TestAppsLocalPluginLifecycleAndCredentialIsolation(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	writeFile(t, filepath.Join(f.WorkDir, "package.json"), `{"name":"fixture","custom":{"keep":true},"actionPlugins":{"@official-plugins/other":"2.0.0"}}`)
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	files := map[string]string{"package/package.json": `{"name":"@official-plugins/fixture","version":"1.0.0"}`, "package/index.js": "export const fixture = '中文';\n"}
	for name, contents := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(contents))}); err != nil {
			t.Fatal(err)
		}
		_, _ = io.WriteString(tw, contents)
	}
	_ = tw.Close()
	_ = gz.Close()
	path := filepath.Join(f.WorkDir, "fixture.tgz")
	if err := os.WriteFile(path, archive.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	expect(t, run(f, "+plugin-install", "--file", path), 0, "data.key", "@official-plugins/fixture")
	installed := filepath.Join(f.WorkDir, "node_modules/@official-plugins/fixture/index.js")
	if b, err := os.ReadFile(installed); err != nil || string(b) != files["package/index.js"] {
		t.Fatalf("installed content %s err %v", b, err)
	}
	listed := run(f, "+plugin-list")
	expect(t, listed, 0, "meta.count", "2")
	expect(t, run(f, "+plugin-install", "--name", "@official-plugins/fixture", "--version", "1.0.0"), 0, "data.status", "already_installed")
	expect(t, run(f, "+plugin-uninstall", "--name", "@official-plugins/fixture"), 0, "data.removed", "true")
	if _, err := os.Stat(installed); !os.IsNotExist(err) {
		t.Fatal("plugin remained")
	}
	b, err := os.ReadFile(filepath.Join(f.WorkDir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "@official-plugins/other") || !gjson.GetBytes(b, "custom.keep").Bool() {
		t.Fatalf("unrelated package state lost: %s", b)
	}
	expect(t, run(f, "+git-credential-list"), 0, "data.count", "0")
	// No metadata exists, so cleanup never resolves or deletes a keychain secret.
	expect(t, run(f, "+git-credential-remove", "--app-id", "next_fixture_missing"), 0, "data.removed", "false")
	helper := f.Run("apps", "git-credential-helper", "store")
	expect(t, helper, 0, "", "")
	if helper.Stdout != "" {
		t.Fatal("Git store helper printed credentials")
	}
}

func TestAppsCredentialArtifactsRefusedBeforeIssuance(t *testing.T) {
	for _, args := range [][]string{
		{"+git-credential-init", "--app-id", "app_fixture"},
		{"+openapi-key-create", "--app-id", "app_fixture", "--name", "fixture", "--scope-all"},
		{"+openapi-key-reset", "--app-id", "app_fixture", "--key-id", "key_fixture", "--yes"},
		{"+automation-update", "--app-id", "app_fixture", "--name", "hook", "--reset-token", "--yes"},
		{"+env-pull", "--app-id", "app_fixture"},
		{"+env-list", "--app-id", "app_fixture", "--include-values"},
	} {
		t.Run(args[0], func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			path := filepath.Join(f.WorkDir, "secret.json")
			result := run(f, append(args, "--save-result", path)...)
			if result.ExitCode != 2 || gjson.Get(result.Stderr, "error.param").String() != "--save-result" {
				t.Fatalf("credential preflight %+v", result)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("credential artifact exists")
			}
		})
	}
}

func TestAppsAutomationAllTriggerTypesAndRotations(t *testing.T) {
	for _, tc := range []struct {
		kind      string
		flags     []string
		condition string
	}{
		{"cron", []string{"--cron", "0 10 * * *", "--timezone", "Asia/Shanghai"}, "cron_condition"},
		{"record-change", []string{"--table", "orders", "--event", "UPDATE", "--fields", `["price","status"]`}, "record_change_condition"},
		{"webhook", []string{"--white-ip-list", `["127.0.0.1"]`}, "webhook_condition"},
		{"feishu-approval", []string{"--event-type", "approval_instance", "--instance-status", "APPROVED", "--approval-code", "approval_fixture"}, "feishu_approval_condition"},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			var writes atomic.Int32
			var created map[string]interface{}
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					reply(w, map[string]interface{}{"trigger": created})
					return
				}
				writes.Add(1)
				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if r.Method == "POST" {
					created = body
					if _, ok := body[tc.condition]; !ok {
						t.Errorf("missing condition %s: %#v", tc.condition, body)
					}
					created["status"] = "disabled"
					reply(w, map[string]interface{}{"trigger": created})
					return
				}
				if r.Method != "PATCH" {
					t.Errorf("wrong status method %s", r.Method)
				}
				created["status"] = body["status"]
				reply(w, map[string]interface{}{"success": true})
			}))
			args := append([]string{"+automation-create", "--app-id", "app_fixture", "--name", "fixture_trigger", "--trigger-type", tc.kind}, tc.flags...)
			expect(t, run(f, args...), 0, "data.trigger.status", "disabled")
			expect(t, run(f, "+automation-get", "--app-id", "app_fixture", "--name", "fixture_trigger"), 0, "data.trigger.name", "fixture_trigger")
			expect(t, run(f, "+automation-enable", "--app-id", "app_fixture", "--name", "fixture_trigger"), 0, "", "")
			expect(t, run(f, "+automation-disable", "--app-id", "app_fixture", "--name", "fixture_trigger"), 0, "", "")
			if writes.Load() != 3 || created["status"] != "disabled" {
				t.Fatal("automation lifecycle effects differ")
			}
		})
	}
	for _, action := range []string{"reset-url", "enable-token", "disable-token", "reset-token"} {
		t.Run(action, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				if !strings.Contains(r.URL.Path, "/triggers/hook/") {
					t.Errorf("rotation target %s", r.URL.Path)
				}
				reply(w, map[string]interface{}{"token_value": "synthetic-rotation-secret", "preview_url": "https://example.invalid/hook"})
			}))
			args := []string{"+automation-update", "--app-id", "app_fixture", "--name", "hook", "--" + action, "--yes"}
			if action == "reset-url" {
				args = append(args, "--app-env", "preview")
			}
			expect(t, run(f, args...), 0, "", "")
			if writes.Load() != 1 {
				t.Fatal("rotation replayed")
			}
			receipts, err := filepath.Glob(filepath.Join(f.ConfigDir, "next/receipts/*.json"))
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range receipts {
				b, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), "synthetic-rotation-secret") || strings.Contains(string(b), "example.invalid") {
					t.Fatal("secret in receipt")
				}
			}
		})
	}
}

func TestAppsDBPreviewHasNoMutationReceipt(t *testing.T) {
	config := `{"mode":"batch","source":{"type":"base","base_url":"https://base.example/base/fixture?table=tbl_fixture","table":{"name":"orders"}},"target":{"type":"postgresql","table":{"name":"orders","action":"create"}},"schema_only":true}`
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/db/sync_create") || body["preview"] != true {
			t.Errorf("preview request %#v", body)
		}
		reply(w, map[string]interface{}{"config": body["config"], "estimated_record_count": 1})
	}))
	result := run(f, "+db-sync-create", "--app-id", "app_fixture", "--config", config, "--preview", "--output", "resolved.json")
	expect(t, result, 0, "", "")
	if strings.Contains(result.Stderr, "mutation") || calls.Load() != 1 {
		t.Fatalf("read preview journaled: %+v", result)
	}
	b, err := os.ReadFile(filepath.Join(f.WorkDir, "resolved.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !gjson.GetBytes(b, "schema_only").Bool() || gjson.GetBytes(b, "source.table.name").String() != "orders" {
		t.Fatalf("preview transfer lost config %s", b)
	}
}

func TestAppsAmbiguityAndIdentityFailBeforeEffects(t *testing.T) {
	for _, args := range [][]string{
		{"+create", "--name", "fixture", "--app-type", "HTML"},
		{"+update", "--app-id", "app_fixture"},
		{"+automation-update", "--app-id", "app_fixture", "--name", "hook", "--cron", "0 10 * * *", "--reset-token", "--yes"},
		{"+automation-update", "--app-id", "app_fixture", "--name", "hook", "--cron", "0 10 * * *", "--white-ip-list", "[]", "--yes"},
		{"+role-member-remove", "--app-id", "app_fixture", "--role-id", "role_fixture", "--users", "ou_fixture", "--all", "--yes"},
		{"+member-add", "--app-id", "app_fixture", "--member-id", "ou_fixture", "--member-type", "unknown", "--perm", "view", "--yes"},
		{"+db-execute", "--app-id", "app_fixture", "--sql", "SELECT 1", "--file", "query.sql", "--yes"},
	} {
		t.Run(args[0]+strings.Join(args[1:], "_"), func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			r := run(f, args...)
			if r.ExitCode != 2 {
				t.Fatalf("invalid input %+v", r)
			}
		})
	}
	f := nextfixture.NewFixture(t, nil)
	r := f.Run("apps", "+get", "--app-id", "app_fixture", "--as", "bot")
	if r.ExitCode == 0 {
		t.Fatal("bot accepted for user-only app read")
	}
}

func TestAppsLivePrivateLifecycle(t *testing.T) {
	t.Skip("blocked: no verified fixture proves current account and tenant, private app creation visibility, run-owned deletion, or downstream automation recipient scope; legacy live fixtures are unauthorized here")
}
