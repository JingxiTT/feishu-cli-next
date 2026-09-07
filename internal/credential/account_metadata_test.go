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

	extcred "github.com/larksuite/cli/extension/credential"
	"github.com/larksuite/cli/internal/core"
)

type metadataProvider struct {
	account    *extcred.Account
	err        error
	selections atomic.Int32
	tokens     atomic.Int32
}

func (*metadataProvider) Name() string { return "metadata-fixture" }
func (p *metadataProvider) ResolveAccount(context.Context) (*extcred.Account, error) {
	p.selections.Add(1)
	return p.account, p.err
}
func (p *metadataProvider) ResolveToken(context.Context, extcred.TokenSpec) (*extcred.Token, error) {
	p.tokens.Add(1)
	return &extcred.Token{Value: "synthetic-metadata-uat"}, nil
}

func TestSupportedIdentitiesDoesNotResolveTokenOrUser(t *testing.T) {
	source := &metadataProvider{account: &extcred.Account{AppID: "app", Brand: "feishu", OpenID: "provided-user", SupportedIdentities: extcred.SupportsUser}}
	reads := 0
	provider := NewCredentialProvider([]extcred.Provider{source}, nil, nil, func() (*http.Client, error) {
		return &http.Client{Transport: bindingRoundTrip(func(r *http.Request) (*http.Response, error) {
			reads++
			if r.URL.Path != "/open-apis/authen/v1/user_info" {
				t.Fatalf("unexpected endpoint %s", r.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"observed-user","tenant_key":"observed-tenant"}}`))}, nil
		})}, nil
	})
	for i := 0; i < 3; i++ {
		ids, err := provider.ResolveSupportedIdentities(context.Background())
		if err != nil || ids != uint8(extcred.SupportsUser) {
			t.Fatalf("metadata result: %d %v", ids, err)
		}
	}
	if source.selections.Load() != 1 || source.tokens.Load() != 0 || reads != 0 {
		t.Fatalf("metadata authenticated: selections=%d tokens=%d HTTP=%d", source.selections.Load(), source.tokens.Load(), reads)
	}
	account, err := provider.ResolveAccount(context.Background())
	if err != nil || !account.UserInfoVerified || account.UserOpenId != "observed-user" || account.TenantKey != "observed-tenant" {
		t.Fatalf("platform account skipped enrichment: %+v %v", account, err)
	}
	if source.account.OpenID != "provided-user" {
		t.Fatal("enrichment mutated provider metadata")
	}
	artifact, err := provider.ResolveArtifactIdentity(context.Background(), core.AsUser)
	if err != nil || artifact.OpenID != "observed-user" || artifact.TenantKey != "observed-tenant" {
		t.Fatalf("artifact evidence changed: %+v %v", artifact, err)
	}
	if source.selections.Load() != 1 || source.tokens.Load() != 1 || reads != 1 {
		t.Fatalf("selection/enrichment not reused: selections=%d tokens=%d HTTP=%d", source.selections.Load(), source.tokens.Load(), reads)
	}
}

func TestSupportedIdentitiesAndEnrichmentConcurrentCache(t *testing.T) {
	source := &metadataProvider{account: &extcred.Account{AppID: "app", Brand: "feishu", SupportedIdentities: extcred.SupportsUser}}
	var reads atomic.Int32
	provider := NewCredentialProvider([]extcred.Provider{source}, nil, nil, func() (*http.Client, error) {
		return &http.Client{Transport: bindingRoundTrip(func(*http.Request) (*http.Response, error) {
			reads.Add(1)
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"user","tenant_key":"tenant"}}`))}, nil
		})}, nil
	})
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func(full bool) {
			defer workers.Done()
			if full {
				account, err := provider.ResolveAccount(context.Background())
				if err != nil || account.UserOpenId != "user" {
					t.Errorf("enrichment failed: %+v %v", account, err)
				}
			} else {
				ids, err := provider.ResolveSupportedIdentities(context.Background())
				if err != nil || ids != uint8(extcred.SupportsUser) {
					t.Errorf("metadata failed: %d %v", ids, err)
				}
			}
		}(i%2 == 0)
	}
	workers.Wait()
	if source.selections.Load() != 1 || source.tokens.Load() != 1 || reads.Load() != 1 {
		t.Fatalf("concurrent selection/enrichment repeated: %d %d %d", source.selections.Load(), source.tokens.Load(), reads.Load())
	}
}

func TestSupportedIdentitiesPreservesSelectedProviderError(t *testing.T) {
	sentinel := errors.New("synthetic account metadata error")
	source := &metadataProvider{err: sentinel}
	fallback := &metadataProvider{account: &extcred.Account{AppID: "other-app"}}
	provider := NewCredentialProvider([]extcred.Provider{source, fallback}, nil, nil, nil)
	if _, err := provider.ResolveSupportedIdentities(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("metadata error changed: %v", err)
	}
	if _, err := provider.ResolveAccount(context.Background()); !errors.Is(err, sentinel) {
		t.Fatalf("account ignored metadata failure: %v", err)
	}
	if source.selections.Load() != 1 || source.tokens.Load() != 0 || fallback.selections.Load() != 0 {
		t.Fatal("failed metadata selection retried or changed account")
	}
}
