// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package nextfixture drives production CLI processes against isolated local fixtures.
package nextfixture

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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
)

var fixtureBinary string

// RunMain builds the production authsidecar CLI and runs a package's tests.
// Call it from TestMain with the repository root relative to the test package:
//
//	os.Exit(nextfixture.RunMain(m, "../../.."))
//
// scripts/next_test.py supplies the required isolated parent environment.
func RunMain(m *testing.M, repoRoot string) int {
	home := os.Getenv("HOME")
	config := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
	if home == "" || config == "" || filepath.Dir(home) != filepath.Dir(config) ||
		!strings.HasPrefix(filepath.Base(filepath.Dir(home)), "next-tests-") {
		fmt.Fprintln(os.Stderr, "foundation E2E requires scripts/next_test.py's isolated environment")
		return 2
	}
	directory, err := os.MkdirTemp("", "foundation-binary-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	fixtureBinary = filepath.Join(directory, "feishu-cli-next-fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	build := exec.CommandContext(ctx, "go", "build", "-tags", "authsidecar", "-o", fixtureBinary, ".")
	build.Dir = repoRoot
	output, err := build.CombinedOutput()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build foundation authsidecar fixture: %v\n%s", err, output)
		_ = os.RemoveAll(directory)
		return 1
	}
	code := m.Run()
	_ = os.RemoveAll(directory)
	return code
}

// Fixture owns separate private config/work directories and a local synthetic backend. Env overrides
// apply to its child CLI only; tests can change them to exercise auth failures.
// Handler requests carry the real API paths and sidecar target/identity headers.
// No request is forwarded to the target named in those headers.
type Fixture struct {
	Env                    map[string]string
	ConfigDir              string
	WorkDir                string
	IdentityRequests       atomic.Int32
	TenantIdentityRequests atomic.Int32
	t                      *testing.T
}

// NewFixture creates a private working/config directory pair and local backend.
// RunMain must initialize the package's CLI binary before tests call Run.
func NewFixture(t *testing.T, handler http.Handler) *Fixture {
	t.Helper()
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected backend request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected fixture request", http.StatusInternalServerError)
		})
	}
	config := t.TempDir()
	fixture := &Fixture{t: t, ConfigDir: config, WorkDir: t.TempDir()}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The sidecar credential source enriches the current account before the
		// invocation selects bot/user. Keep that read visible separately from
		// business requests, whose handlers and request counts stay unchanged.
		if r.Method == http.MethodGet && r.URL.Path == "/open-apis/authen/v1/user_info" && r.URL.RawQuery == "" {
			fixture.IdentityRequests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"open_id":"ou_foundation_fixture","name":"Synthetic Fixture","tenant_key":"foundation-fixture-tenant"}}`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/open-apis/tenant/v2/tenant/query" && r.URL.RawQuery == "" {
			fixture.TenantIdentityRequests.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":0,"data":{"tenant":{"tenant_key":"foundation-fixture-tenant"}}}`)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() {
		t.Logf("fixture identity requests: user=%d tenant=%d", fixture.IdentityRequests.Load(), fixture.TenantIdentityRequests.Load())
	})
	fixture.Env = map[string]string{
		"LARKSUITE_CLI_CONFIG_DIR": config,
		"LARKSUITE_CLI_AUTH_PROXY": server.URL,
		"LARKSUITE_CLI_PROXY_KEY":  "synthetic-foundation-proxy-key",
		"LARKSUITE_CLI_APP_ID":     "synthetic-foundation-app",
		"LARKSUITE_CLI_BRAND":      "feishu",
		"LARKSUITE_CLI_DEFAULT_AS": "bot",
	}
	return fixture
}

// Result preserves the process output and exit code without retries.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run executes once without retrying errors or unknown mutation outcomes.
func (f *Fixture) Run(args ...string) Result {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, fixtureBinary, args...)
	command.Dir = f.WorkDir
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, overridden := f.Env[key]; !overridden {
			command.Env = append(command.Env, entry)
		}
	}
	for key, value := range f.Env {
		command.Env = append(command.Env, key+"="+value)
	}
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			f.t.Fatalf("start fixture CLI: %v", err)
		}
		result.ExitCode = exit.ExitCode()
	}
	return result
}
