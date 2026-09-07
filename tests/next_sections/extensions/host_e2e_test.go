// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

func buildHost(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	// Outside the CLI module: importing an internal package would fail.
	directory, err := os.MkdirTemp("", "external-host-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	for _, name := range []string{"go.mod", "go.sum"} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if name == "go.mod" {
			content = []byte(strings.Replace(string(content), "module github.com/larksuite/cli", "module hostfixture", 1) + "\nrequire github.com/larksuite/cli v0.0.0\nreplace github.com/larksuite/cli => " + root + "\n")
		}
		if err := os.WriteFile(filepath.Join(directory, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
	content, err := os.ReadFile("testdata/host/main.go")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "main.go"), content, 0600); err != nil {
		t.Fatal(err)
	}
	hostBinary := filepath.Join(directory, "host-cli")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-mod=mod", "-o", hostBinary, ".")
	build.Dir = directory
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("external host build: %v\n%s", err, output)
	}
	return hostBinary
}

type hostReport struct {
	Files map[string]string `json:"files"`
	Saves int               `json:"saves"`
}

func runHost(t *testing.T, binary, backend string, env map[string]string, args ...string) (nextfixture.Result, hostReport) {
	t.Helper()
	directory := t.TempDir()
	reportPath := filepath.Join(directory, "report.json")
	overrides := map[string]string{"HOST_BACKEND": backend, "HOST_REPORT": reportPath, "HOST_AGENT": "agent", "HOST_FILES": "exclusive", "LARKSUITE_CLI_CONFIG_DIR": filepath.Join(directory, "config")}
	for key, value := range env {
		overrides[key] = value
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	process := exec.CommandContext(ctx, binary, args...)
	process.Dir = directory
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, exists := overrides[key]; !exists {
			process.Env = append(process.Env, entry)
		}
	}
	for key, value := range overrides {
		process.Env = append(process.Env, key+"="+value)
	}
	var stdout, stderr bytes.Buffer
	process.Stdout, process.Stderr = &stdout, &stderr
	err := process.Run()
	result := nextfixture.Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		result.ExitCode = exit.ExitCode()
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("host report: %v; result=%+v", err, result)
	}
	var report hostReport
	if err := json.Unmarshal(content, &report); err != nil {
		t.Fatal(err)
	}
	return result, report
}

func envelope(t *testing.T, text string) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatalf("JSON: %v: %s", err, text)
	}
	return result
}

func TestExternalHostContracts(t *testing.T) {
	binary := buildHost(t)
	var requests atomic.Int32
	var tenantRequests atomic.Int32
	var status atomic.Int32
	status.Store(http.StatusOK)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/open-apis/tenant/v2/tenant/query" {
			tenantRequests.Add(1)
			if r.Method != http.MethodGet || r.Header.Get("Authorization") != "Bearer synthetic-host-token" {
				t.Errorf("unexpected tenant verification: %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"tenant":{"tenant_key":"synthetic-host-tenant"}}}`)
			return
		}
		requests.Add(1)
		if r.URL.Path != "/open-apis/im/v1/host-fixture" || r.Header.Get("Authorization") != "Bearer synthetic-host-token" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Method == http.MethodPost {
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"title":"synthetic"}` {
				t.Errorf("body=%s", body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(status.Load()))
		_, _ = io.WriteString(w, `{"code":0,"data":{"id":"001","title":"标题","content":"full 内容","number":42.5}}`)
	}))
	defer server.Close()
	for _, mode := range []string{"legacy", "agent"} {
		t.Run(mode, func(t *testing.T) {
			result, report := runHost(t, binary, server.URL, map[string]string{"HOST_AGENT": mode}, "im", "+host-read", "--as", "bot")
			if result.ExitCode != 0 || result.Stderr != "" || len(report.Files) != 0 || tenantRequests.Load() != 0 {
				t.Fatalf("read: %+v report=%+v", result, report)
			}
			data := envelope(t, result.Stdout)["data"].(map[string]any)
			if data["id"] != "001" || data["content"] != "full 内容" || data["number"] != 42.5 {
				t.Fatalf("lossy data: %v", data)
			}
			if (strings.Count(result.Stdout, "\n") == 1) != (mode == "agent") {
				t.Fatalf("format %s: %s", mode, result.Stdout)
			}
		})
	}
	t.Run("legacy FileIO remains usable for ordinary reads", func(t *testing.T) {
		result, report := runHost(t, binary, server.URL, map[string]string{"HOST_FILES": "legacy"}, "im", "+host-read", "--as", "bot")
		if result.ExitCode != 0 || result.Stderr != "" || report.Saves != 0 || tenantRequests.Load() != 0 || envelope(t, result.Stdout)["ok"] != true {
			t.Fatalf("legacy FileIO read: %+v", result)
		}
	})
	t.Run("artifact full content and canonical host reference", func(t *testing.T) {
		before := tenantRequests.Load()
		result, report := runHost(t, binary, server.URL, nil, "im", "+host-read", "--as", "bot", "--result-fields", "title", "--save-result", "result.json")
		if result.ExitCode != 0 || result.Stderr != "" || report.Saves != 1 || tenantRequests.Load()-before != 1 {
			t.Fatalf("artifact: %+v report=%+v", result, report)
		}
		output := envelope(t, result.Stdout)
		artifact := output["artifact"].(map[string]any)
		if artifact["ref"] != "memory://result.json" {
			t.Fatalf("host reference: %v", artifact)
		}
		saved := envelope(t, report.Files["memory://result.json"])
		if saved["schema"] != "feishu-cli-next/result/v1" {
			t.Fatalf("schema: %v", saved)
		}
		binding := saved["binding"].(map[string]any)
		if binding["app_id"] != "synthetic-host-app" || binding["identity"] != "bot" || binding["tenant_verified"] != true || binding["tenant_key"] != "synthetic-host-tenant" {
			t.Fatalf("binding: %v", binding)
		}
		data := saved["result"].(map[string]any)["data"].(map[string]any)
		if data["content"] != "full 内容" || data["id"] != "001" {
			t.Fatalf("artifact lost content: %v", data)
		}
	})
	for _, test := range []struct {
		name, mode, path string
		calls, saves     int32
	}{
		{"unsupported exclusive provider", "legacy", "result.json", 0, 0},
		{"host denied path", "exclusive", "denied.json", 0, 0},
		{"concurrent destination", "race", "result.json", 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := requests.Load()
			tenantBefore := tenantRequests.Load()
			result, report := runHost(t, binary, server.URL, map[string]string{"HOST_FILES": test.mode}, "im", "+host-read", "--as", "bot", "--save-result", test.path)
			if result.ExitCode == 0 || result.Stdout != "" || requests.Load()-before != test.calls || report.Saves != int(test.saves) {
				t.Fatalf("failure: %+v calls=%d report=%+v", result, requests.Load()-before, report)
			}
			if tenantRequests.Load()-tenantBefore != test.calls {
				t.Fatalf("tenant verification calls = %d, want %d", tenantRequests.Load()-tenantBefore, test.calls)
			}
			failure, ok := envelope(t, result.Stderr)["error"].(map[string]any)
			if !ok || failure["param"] != "--save-result" {
				t.Fatalf("missing typed error: %s", result.Stderr)
			}
			if test.mode == "race" && report.Files["memory://result.json"] != "concurrent owner" {
				t.Fatalf("overwritten concurrent file: %+v", report)
			}
		})
	}
	t.Run("dry run has no effects", func(t *testing.T) {
		before := requests.Load()
		result, report := runHost(t, binary, server.URL, nil, "im", "+host-write", "--as", "bot", "--dry-run")
		if result.ExitCode != 0 || requests.Load() != before || report.Saves != 0 || envelope(t, result.Stdout)["dry_run"] != true {
			t.Fatalf("dry run: %+v", result)
		}
	})
	for _, test := range []struct {
		name, outcome string
		status, exit  int
	}{
		{"acknowledged mutation", "confirmed_applied", http.StatusOK, 0},
		{"unknown mutation is not success or retried", "unknown", http.StatusAccepted, 22},
		{"rejected mutation is not retried", "confirmed_unapplied", http.StatusForbidden, 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			status.Store(int32(test.status))
			before := requests.Load()
			result, _ := runHost(t, binary, server.URL, nil, "im", "+host-write", "--as", "bot")
			if result.ExitCode != test.exit || requests.Load()-before != 1 {
				t.Fatalf("mutation: %+v requests=%d", result, requests.Load()-before)
			}
			if result.Stdout != "" {
				output := envelope(t, result.Stdout)
				mutation := output["mutation"].(map[string]any)
				if output["ok"] != (test.exit == 0) || mutation["outcome"] != test.outcome {
					t.Fatalf("incorrect outcome: %v", output)
				}
			}
			if test.exit != 0 && (!strings.Contains(result.Stderr, `"mutation_outcome"`) || !strings.Contains(result.Stderr, test.outcome)) {
				t.Fatalf("missing typed outcome: %s", result.Stderr)
			}
		})
	}
}
