// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var fixtureBinary string

// TestMain uses the existing production authsidecar build to drive real CLI
// commands against local handlers. The normal binary is tested separately by
// foundation_test.py. This avoids a product test hook or OS certificate changes.
func TestMain(m *testing.M) {
	home := os.Getenv("HOME")
	config := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
	if home == "" || config == "" || filepath.Dir(home) != filepath.Dir(config) ||
		!strings.HasPrefix(filepath.Base(filepath.Dir(home)), "next-tests-") {
		fmt.Fprintln(os.Stderr, "foundation E2E requires scripts/next_test.py's isolated environment")
		os.Exit(2)
	}
	directory, err := os.MkdirTemp("", "foundation-binary-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fixtureBinary = filepath.Join(directory, "feishu-cli-next-fixture")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	build := exec.CommandContext(ctx, "go", "build", "-tags", "authsidecar", "-o", fixtureBinary, ".")
	build.Dir = filepath.Join("..", "..")
	output, err := build.CombinedOutput()
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "build foundation authsidecar fixture: %v\n%s", err, output)
		_ = os.RemoveAll(directory)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(directory)
	os.Exit(code)
}

// Fixture owns separate private config/work directories and a local synthetic backend. Env overrides
// apply to its child CLI only; tests can change them to exercise auth failures.
// Handler requests carry the real API paths and sidecar target/identity headers.
// No request is forwarded to the target named in those headers.
type Fixture struct {
	Env       map[string]string
	ConfigDir string
	WorkDir   string
	t         *testing.T
}

func NewFixture(t *testing.T, handler http.Handler) *Fixture {
	t.Helper()
	if handler == nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("unexpected backend request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected fixture request", http.StatusInternalServerError)
		})
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	config := t.TempDir()
	return &Fixture{t: t, ConfigDir: config, WorkDir: t.TempDir(), Env: map[string]string{
		"LARKSUITE_CLI_CONFIG_DIR": config,
		"LARKSUITE_CLI_AUTH_PROXY": server.URL,
		"LARKSUITE_CLI_PROXY_KEY":  "synthetic-foundation-proxy-key",
		"LARKSUITE_CLI_APP_ID":     "synthetic-foundation-app",
		"LARKSUITE_CLI_BRAND":      "feishu",
		"LARKSUITE_CLI_DEFAULT_AS": "bot",
	}}
}

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
