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

	"github.com/larksuite/cli/internal/core"
)

type tokenMintFunc func(*http.Request) (*http.Response, error)

func (f tokenMintFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func tokenProviderForTest(t *testing.T, transport http.RoundTripper) *DefaultTokenProvider {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	if err := core.SaveMultiAppConfig(&core.MultiAppConfig{Apps: []core.AppConfig{{
		AppId: "fixture_app", AppSecret: core.PlainSecret("fixture_secret"), Brand: core.BrandFeishu, DPoPMode: core.DPoPModeDisabled,
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
		AppId: "other_app", AppSecret: core.PlainSecret("other_fixture_secret"), Brand: core.BrandFeishu, DPoPMode: core.DPoPModeDisabled,
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
