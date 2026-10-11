// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
)

type tokenMintFunc func(*http.Request) (*http.Response, error)

func (f tokenMintFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func tokenProviderForTest(t *testing.T, transport http.RoundTripper) *DefaultTokenProvider {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	if err := core.SaveMultiAppConfig(&core.MultiAppConfig{Apps: []core.AppConfig{{
		AppId: "fixture_app", AppSecret: core.PlainSecret("fixture_secret"), Brand: core.BrandFeishu,
	}}}); err != nil {
		t.Fatal(err)
	}
	return NewDefaultTokenProvider(NewDefaultAccountProvider(nil, "", core.ProfileFromConfig),
		func() (*http.Client, error) { return &http.Client{Transport: transport}, nil }, io.Discard)
}

func TestDefaultTokenProvider_TATRefresh(t *testing.T) {
	var calls atomic.Int32
	p := tokenProviderForTest(t, tokenMintFunc(func(req *http.Request) (*http.Response, error) {
		if err := req.ParseForm(); err != nil || req.Form.Get("client_id") != "fixture_app" {
			t.Errorf("refresh switched apps: client_id=%q, error=%v", req.Form.Get("client_id"), err)
		}
		calls.Add(1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(
			`{"access_token":"fixture_token","expires_in":1}`))}, nil
	}))
	spec := TokenSpec{Type: TokenTypeTAT, AppID: "fixture_app"}
	resolveConcurrently := func() {
		t.Helper()
		var wg sync.WaitGroup
		for i := 0; i < 12; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				result, err := p.ResolveToken(context.Background(), spec)
				if err != nil || result.Token != "fixture_token" {
					t.Errorf("token = %v, error = %v", result, err)
				}
			}()
		}
		wg.Wait()
	}
	resolveConcurrently()
	if calls.Load() != 1 {
		t.Fatalf("concurrent callers minted %d tokens", calls.Load())
	}
	if err := core.SaveMultiAppConfig(&core.MultiAppConfig{Apps: []core.AppConfig{{
		AppId: "other_app", AppSecret: core.PlainSecret("other_fixture_secret"), Brand: core.BrandFeishu,
	}}}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	resolveConcurrently()
	if calls.Load() != 2 {
		t.Fatalf("expired token reused or refresh duplicated: mint calls = %d", calls.Load())
	}
}

func TestDefaultTokenProvider_TATFailureDoesNotPoisonCache(t *testing.T) {
	cause := errors.New("temporary mint failure")
	calls := 0
	p := tokenProviderForTest(t, tokenMintFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, cause
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(
			`{"access_token":"fixture_token","expires_in":7200}`))}, nil
	}))
	spec := TokenSpec{Type: TokenTypeTAT}
	if _, err := p.ResolveToken(context.Background(), spec); !errors.Is(err, cause) {
		t.Fatalf("mint cause lost: %v", err)
	}
	result, err := p.ResolveToken(context.Background(), spec)
	if err != nil || result.Token != "fixture_token" || calls != 2 {
		t.Fatalf("token = %v, calls = %d, error = %v", result, calls, err)
	}
}

func TestDefaultTokenProvider_Dispatches(t *testing.T) {
	// Just verify the type implements DefaultTokenResolver
	var _ DefaultTokenResolver = &DefaultTokenProvider{}
}

func TestDefaultAccountProvider_Implements(t *testing.T) {
	var _ DefaultAccountResolver = &DefaultAccountProvider{}
}

// TestClassifyTATResponseCode_InvalidClient_MapsToInvalidClient pins that the
// unified Token Endpoint's OAuth2 invalid_client error surfaces as
// CategoryConfig/InvalidClient — the configured app_id/app_secret cannot mint a
// tenant access token, the same actionable failure the legacy 10003/10014 codes
// produced. The numeric code is intentionally not asserted: the v3 endpoint may
// return invalid_client with no Lark code (code defaults to 0).
func TestClassifyTATResponseCode_InvalidClient_MapsToInvalidClient(t *testing.T) {
	err := classifyTATResponseCode(0, "invalid_client", "client authentication failed", "feishu", "cli_app_x")
	if err == nil {
		t.Fatal("expected non-nil error for invalid_client")
	}
	var cfgErr *errs.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected *errs.ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Category != errs.CategoryConfig {
		t.Errorf("Category = %q, want %q", cfgErr.Category, errs.CategoryConfig)
	}
	if cfgErr.Subtype != errs.SubtypeInvalidClient {
		t.Errorf("Subtype = %q, want %q", cfgErr.Subtype, errs.SubtypeInvalidClient)
	}
	if cfgErr.Hint == "" {
		t.Error("Hint must be non-empty so the user gets a recovery action")
	}
}

// TestClassifyTATResponseCode_UnauthorizedClient_MapsToInvalidClient pins that
// unauthorized_client is treated as the same credential failure as
// invalid_client.
func TestClassifyTATResponseCode_UnauthorizedClient_MapsToInvalidClient(t *testing.T) {
	err := classifyTATResponseCode(0, "unauthorized_client", "client not authorized", "feishu", "cli_app_x")
	var cfgErr *errs.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Fatalf("expected *errs.ConfigError, got %T: %v", err, err)
	}
	if cfgErr.Subtype != errs.SubtypeInvalidClient {
		t.Errorf("Subtype = %q, want %q", cfgErr.Subtype, errs.SubtypeInvalidClient)
	}
}

// TestClassifyTATResponseCode_OtherErrorFallsThrough pins that OAuth errors
// outside the credential set fall through to the generic BuildAPIError fallback
// — still typed, but not a ConfigError. The mapping is narrow and intentional.
func TestClassifyTATResponseCode_OtherErrorFallsThrough(t *testing.T) {
	err := classifyTATResponseCode(20068, "invalid_scope", "unauthorized scope", "feishu", "cli_app_x")
	if err == nil {
		t.Fatal("expected non-nil error for invalid_scope")
	}
	var cfgErr *errs.ConfigError
	if errors.As(err, &cfgErr) {
		t.Fatalf("invalid_scope must not be classified as ConfigError, got %T", err)
	}
}

// TestClassifyTATResponseCode_CodeZeroOtherError_StillTyped pins the code-0
// backstop: a non-credential OAuth error (e.g. invalid_scope) that arrives with no
// numeric code (code 0) must still produce a non-nil typed error. BuildAPIError
// returns nil for code 0 (Feishu's success convention); without the backstop,
// FetchTAT would surface this deterministic rejection as an empty result with no
// error.
func TestClassifyTATResponseCode_CodeZeroOtherError_StillTyped(t *testing.T) {
	err := classifyTATResponseCode(0, "invalid_scope", "the requested scope is not granted", "feishu", "cli_app_x")
	if err == nil {
		t.Fatal("expected non-nil error for code-0 invalid_scope (must not be swallowed as success)")
	}
	if !errs.IsTyped(err) {
		t.Fatalf("expected a typed errs.* error, got %T %v", err, err)
	}
	var cfgErr *errs.ConfigError
	if errors.As(err, &cfgErr) {
		t.Fatalf("code-0 invalid_scope must not be a ConfigError, got %T", err)
	}
}
