// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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
	if fixture.IdentityRequests.Load() != 1 {
		t.Fatalf("identity enrichment requests=%d, want 1", fixture.IdentityRequests.Load())
	}
	if requests.Load() != 1 || !strings.Contains(result.Stdout, `"synthetic-fixture-result"`) {
		t.Fatalf("fixture result not independently observed: requests=%d stdout=%s", requests.Load(), result.Stdout)
	}
}

func TestFixtureOnlyInterceptsExactIdentityRead(t *testing.T) {
	var business atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		business.Add(1)
		w.WriteHeader(http.StatusTeapot)
	}))
	base := fixture.Env["LARKSUITE_CLI_AUTH_PROXY"]
	response, err := http.Get(base + "/open-apis/authen/v1/user_info")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var payload struct {
		Code int `json:"code"`
		Data struct {
			OpenID string `json:"open_id"`
			Tenant string `json:"tenant_key"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || payload.Code != 0 || payload.Data.OpenID != "ou_foundation_fixture" || payload.Data.Tenant != "foundation-fixture-tenant" {
		t.Fatalf("unexpected identity binding response: status=%d payload=%+v", response.StatusCode, payload)
	}
	for _, request := range []struct{ method, path string }{
		{http.MethodPost, "/open-apis/authen/v1/user_info"},
		{http.MethodGet, "/open-apis/authen/v1/user_info/other"},
		{http.MethodGet, "/open-apis/authen/v1/user_info?query=business"},
		{http.MethodPost, "/open-apis/tenant/v2/tenant/query"},
		{http.MethodGet, "/open-apis/tenant/v2/tenant/query/other"},
		{http.MethodGet, "/open-apis/tenant/v2/tenant/query?query=business"},
	} {
		req, err := http.NewRequest(request.method, base+request.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusTeapot {
			t.Errorf("business request intercepted: %s %s", request.method, request.path)
		}
	}
	if fixture.IdentityRequests.Load() != 1 || business.Load() != 6 {
		t.Fatalf("request counts: identity=%d business=%d", fixture.IdentityRequests.Load(), business.Load())
	}
}

func TestFixtureTenantReadIsCountedSeparately(t *testing.T) {
	fixture := NewFixture(t, nil)
	result := fixture.Run("api", "GET", "/open-apis/tenant/v2/tenant/query", "--as", "bot")
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, `"tenant_key":"foundation-fixture-tenant"`) {
		t.Fatalf("tenant binding response: exit=%d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if fixture.IdentityRequests.Load() != 1 || fixture.TenantIdentityRequests.Load() != 1 {
		t.Fatalf("identity counts: user=%d tenant=%d", fixture.IdentityRequests.Load(), fixture.TenantIdentityRequests.Load())
	}
}

func TestFixtureRunContextCancelsHangingProcessWithoutRetry(t *testing.T) {
	started := make(chan struct{}, 1)
	var requests atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		started <- struct{}{}
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	finished := make(chan Result, 1)
	go func() {
		finished <- fixture.RunContext(ctx, "api", "GET", "/open-apis/fixture/v1/hang", "--as", "bot")
	}()
	select {
	case <-started:
		cancel()
	case <-ctx.Done():
		t.Fatal("fixture backend did not receive the request")
	}
	select {
	case result := <-finished:
		if result.ExitCode == 0 || strings.Contains(result.Stdout, `"ok":true`) {
			t.Fatalf("canceled process reported success: %+v", result)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("fixture process did not stop after cancellation")
	}
	if requests.Load() != 1 {
		t.Fatalf("business requests=%d, want one attempt", requests.Load())
	}
}

func TestFixtureRunContextCanceledBeforeStart(t *testing.T) {
	fixture := NewFixture(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := fixture.RunContext(ctx, "api", "GET", "/open-apis/fixture/v1/not-started")
	if result.ExitCode != -1 || result.Stdout != "" || result.Stderr != "" {
		t.Fatalf("canceled-before-start result=%+v", result)
	}
	if fixture.IdentityRequests.Load() != 0 || fixture.TenantIdentityRequests.Load() != 0 {
		t.Fatal("canceled-before-start command reached identity backend")
	}
}
