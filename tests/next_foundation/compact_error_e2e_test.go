// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTypedErrorEnvelopeIsOneLineOnStderr(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("usage error sent a request: %s %s", r.Method, r.URL.Path)
	}))
	got := f.Run("docs", "+pull")
	if got.ExitCode != 2 {
		t.Fatalf("exit %d, stderr %s", got.ExitCode, got.Stderr)
	}
	lines := strings.Split(strings.TrimSuffix(got.Stderr, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("stderr has %d lines, want 1: %q", len(lines), got.Stderr)
	}
	var envelope struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &envelope); err != nil {
		t.Fatalf("stderr is not JSON: %v: %q", err, got.Stderr)
	}
	if envelope.OK {
		t.Fatalf("envelope reports ok:true: %q", got.Stderr)
	}
}
