// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"net/http"
	"strings"
	"testing"
)

// Dry-run stays offline, so the fixture rejects any request.
func TestSingleSupportedIdentityUsedWhenNoneChosen(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run sent a request: %s %s", r.Method, r.URL.Path)
	}))
	f.Env["LARKSUITE_CLI_DEFAULT_AS"] = ""

	auto := f.Run("docs", "+search", "--query", "x", "--dry-run")
	if auto.ExitCode != 0 {
		t.Fatalf("auto-detected identity: exit %d, stdout %s, stderr %s", auto.ExitCode, auto.Stdout, auto.Stderr)
	}
	if !strings.Contains(auto.Stdout, `"identity":"user"`) {
		t.Errorf("dry-run output does not report user identity: %s", auto.Stdout)
	}

	explicit := f.Run("docs", "+search", "--query", "x", "--as", "bot", "--dry-run")
	if explicit.ExitCode == 0 {
		t.Fatalf("explicit --as bot succeeded: stdout %s", explicit.Stdout)
	}
	out := explicit.Stdout + explicit.Stderr
	if !strings.Contains(out, "--as bot is not supported, this command only supports: user") {
		t.Errorf("explicit --as bot error changed: %s", out)
	}
}

func TestConfiguredDefaultAsBotKeepsIdentityError(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run sent a request: %s %s", r.Method, r.URL.Path)
	}))

	got := f.Run("docs", "+search", "--query", "x", "--dry-run")
	if got.ExitCode == 0 {
		t.Fatalf("configured default-as bot succeeded: stdout %s", got.Stdout)
	}
	if !strings.Contains(got.Stdout+got.Stderr, "--as bot is not supported, this command only supports: user") {
		t.Errorf("configured default-as bot error changed: %s", got.Stdout+got.Stderr)
	}
}
