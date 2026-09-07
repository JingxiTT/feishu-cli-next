// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	extcred "github.com/larksuite/cli/extension/credential"
	"github.com/larksuite/cli/internal/core"
)

type bindingRoundTrip func(*http.Request) (*http.Response, error)

func (f bindingRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestUserBindingReusesEnrichmentAndTenant(t *testing.T) {
	calls := 0
	provider := NewCredentialProvider([]extcred.Provider{&mockExtProvider{name: "fixture", account: &extcred.Account{AppID: "app", Brand: "feishu"}, token: &extcred.Token{Value: "synthetic"}}}, nil, nil, func() (*http.Client, error) {
		return &http.Client{Transport: bindingRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"user","tenant_key":"tenant"}}`))}, nil
		})}, nil
	})
	for i := 0; i < 2; i++ {
		binding, err := provider.ResolveUserBinding(context.Background())
		if err != nil || binding.OpenID != "user" || binding.TenantKey != "tenant" || !binding.Verified {
			t.Fatalf("%+v %v", binding, err)
		}
	}
	if calls != 1 {
		t.Fatalf("repeated identity reads: %d", calls)
	}
}
func TestUserBindingResolvesMissingUserFromSelectedToken(t *testing.T) {
	calls := 0
	provider := NewCredentialProvider(nil, &mockDefaultAcct{account: &Account{AppID: "app", Brand: core.BrandFeishu}}, &mockDefaultToken{result: &TokenResult{Token: "synthetic"}}, func() (*http.Client, error) {
		return &http.Client{Transport: bindingRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Method != "GET" || r.URL.Path != "/open-apis/authen/v1/user_info" {
				t.Fatal(r.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"user"}}`))}, nil
		})}, nil
	})
	for i := 0; i < 2; i++ {
		binding, err := provider.ResolveUserBinding(context.Background())
		if err != nil || binding.OpenID != "user" || binding.TenantKey != "" || !binding.Verified {
			t.Fatalf("%+v %v", binding, err)
		}
	}
	if calls != 1 {
		t.Fatalf("repeated identity reads: %d", calls)
	}
}
