// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFixtureRoutesToSyntheticBackend(t *testing.T) {
	var requests atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/open-apis/fixture/v1/objects" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "" {
			t.Error("sidecar fixture received an unstripped credential")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"marker":"synthetic-fixture-result"}}`)
	}))
	result := fixture.Run("api", "GET", "/open-apis/fixture/v1/objects", "--as", "bot")
	if result.ExitCode != 0 {
		t.Fatalf("fixture CLI failed: %d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if requests.Load() != 1 || !strings.Contains(result.Stdout, `"synthetic-fixture-result"`) {
		t.Fatalf("fixture result not independently observed: requests=%d stdout=%s", requests.Load(), result.Stdout)
	}
}
