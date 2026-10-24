// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/dpop"
)

func TestArtifactIdentityVerifiesPersistedUserAndBotOnce(t *testing.T) {
	for _, identity := range []core.Identity{core.AsUser, core.AsBot} {
		t.Run(string(identity), func(t *testing.T) {
			calls := 0
			account := &Account{AppID: "app", Brand: core.BrandFeishu, UserOpenId: "persisted-user"}
			p := NewCredentialProvider(nil, &mockDefaultAcct{account: account}, &mockDefaultToken{result: &TokenResult{Token: "synthetic-selected-token"}}, func() (*http.Client, error) {
				return &http.Client{Transport: bindingRoundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					path, body := "/open-apis/authen/v1/user_info", `{"code":0,"data":{"open_id":"persisted-user","tenant_key":"tenant"}}`
					if identity == core.AsBot {
						path, body = "/open-apis/tenant/v2/tenant/query", `{"code":0,"data":{"tenant":{"tenant_key":"tenant"}}}`
					}
					if r.Method != "GET" || r.URL.Path != path || r.Header.Get("Authorization") != "Bearer synthetic-selected-token" {
						t.Fatalf("wrong selected identity request: %s %s", r.Method, r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}, nil
			})
			for i := 0; i < 2; i++ {
				got, err := p.ResolveArtifactIdentity(context.Background(), identity)
				if err != nil || got.TenantKey != "tenant" || (identity == core.AsUser && got.OpenID != "persisted-user") || (identity == core.AsBot && got.OpenID != "") {
					t.Fatalf("unverified binding: %+v %v", got, err)
				}
			}
			if calls != 1 {
				t.Fatalf("identity calls=%d", calls)
			}
		})
	}
}

func TestArtifactIdentityRefusesInsufficientOrDifferentEvidence(t *testing.T) {
	tests := []struct {
		name     string
		identity core.Identity
		account  *Account
		body     string
		status   int
		category errs.Category
	}{
		{"user mismatch", core.AsUser, &Account{UserOpenId: "persisted-user"}, `{"code":0,"data":{"open_id":"different-user","tenant_key":"tenant"}}`, 200, errs.CategoryValidation},
		{"tenant mismatch", core.AsBot, &Account{TenantKey: "saved-tenant"}, `{"code":0,"data":{"tenant":{"tenant_key":"different-tenant"}}}`, 200, errs.CategoryValidation},
		{"missing user tenant", core.AsUser, &Account{UserOpenId: "persisted-user"}, `{"code":0,"data":{"open_id":"persisted-user"}}`, 200, errs.CategoryValidation},
		{"missing bot tenant", core.AsBot, &Account{}, `{"code":0,"data":{"tenant":{}}}`, 200, errs.CategoryValidation},
		{"unverified configured tenant", core.AsUser, &Account{UserOpenId: "persisted-user", TenantKey: "configured-tenant"}, `{"code":0,"data":{"open_id":"persisted-user"}}`, 200, errs.CategoryValidation},
		{"missing success code", core.AsBot, &Account{}, `{"data":{"tenant":{"tenant_key":"tenant"}}}`, 200, errs.CategoryInternal},
		{"HTTP permission", core.AsBot, &Account{}, `denied`, 403, errs.CategoryAuthorization},
		{"scope permission", core.AsBot, &Account{}, `{"code":99991672,"msg":"permission denied","error":{"permission_violations":[{"type":"action","subject":"tenant:tenant:readonly"}]}}`, 403, errs.CategoryAuthorization},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.account.AppID = "app"
			tt.account.Brand = core.BrandFeishu
			calls := 0
			p := NewCredentialProvider(nil, &mockDefaultAcct{account: tt.account}, &mockDefaultToken{result: &TokenResult{Token: "synthetic-secret"}}, func() (*http.Client, error) {
				return &http.Client{Transport: bindingRoundTrip(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: tt.status, Body: io.NopCloser(strings.NewReader(tt.body))}, nil
				})}, nil
			})
			for i := 0; i < 2; i++ {
				got, err := p.ResolveArtifactIdentity(context.Background(), tt.identity)
				problem, ok := errs.ProblemOf(err)
				if !ok || problem.Category != tt.category || got != (ArtifactIdentity{}) {
					t.Fatalf("accepted invalid identity: %+v %T %v", got, err, err)
				}
				if strings.Contains(err.Error(), "synthetic-secret") {
					t.Fatal("credential leak")
				}
				if tt.name == "scope permission" {
					var permission *errs.PermissionError
					if !errors.As(err, &permission) || len(permission.MissingScopes) != 1 || permission.MissingScopes[0] != "tenant:tenant:readonly" {
						t.Fatalf("missing scope evidence lost: %#v", err)
					}
				}
			}
			if calls != 1 {
				t.Fatalf("retried rejected lookup: %d", calls)
			}
		})
	}
}

func TestArtifactIdentityDoesNotRetryIncompleteCurrentEnrichment(t *testing.T) {
	p := NewCredentialProvider(nil, &mockDefaultAcct{account: &Account{AppID: "app", Brand: core.BrandFeishu, UserOpenId: "user", UserInfoVerified: true}}, nil, func() (*http.Client, error) { t.Fatal("current user_info evidence was queried again"); return nil, nil })
	if _, err := p.ResolveArtifactIdentity(context.Background(), core.AsUser); !errs.IsValidation(err) {
		t.Fatalf("missing tenant accepted: %v", err)
	}
}

func TestIdentityLookupsCarrySelectedTokenDPoPBinding(t *testing.T) {
	for _, lookup := range []string{"artifact-user", "artifact-bot", "receipt-user"} {
		t.Run(lookup, func(t *testing.T) {
			binding := &dpop.Binding{}
			calls := 0
			p := NewCredentialProvider(nil, &mockDefaultAcct{account: &Account{AppID: "app", Brand: core.BrandFeishu}},
				&mockDefaultToken{result: &TokenResult{Token: "synthetic", DPoP: binding}}, func() (*http.Client, error) {
					return &http.Client{Transport: bindingRoundTrip(func(r *http.Request) (*http.Response, error) {
						calls++
						if got, ok := dpop.BindingFromContext(r.Context()); !ok || got != binding {
							t.Fatal("identity lookup lost the selected token's DPoP binding")
						}
						body := `{"code":0,"data":{"open_id":"user","tenant_key":"tenant"}}`
						if lookup == "artifact-bot" {
							body = `{"code":0,"data":{"tenant":{"tenant_key":"tenant"}}}`
						}
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
					})}, nil
				})
			var err error
			switch lookup {
			case "artifact-user":
				_, err = p.ResolveArtifactIdentity(context.Background(), core.AsUser)
			case "artifact-bot":
				_, err = p.ResolveArtifactIdentity(context.Background(), core.AsBot)
			default:
				_, err = p.ResolveUserBinding(context.Background())
			}
			if err != nil || calls != 1 {
				t.Fatalf("calls=%d, error=%v", calls, err)
			}
		})
	}
}
