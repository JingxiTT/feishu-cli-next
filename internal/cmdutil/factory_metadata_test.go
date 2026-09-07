// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"net/http"
	"testing"

	extcred "github.com/larksuite/cli/extension/credential"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
)

type strictMetadataProvider struct {
	t       *testing.T
	calls   int
	support extcred.IdentitySupport
}

func (*strictMetadataProvider) Name() string { return "strict-metadata" }
func (p *strictMetadataProvider) ResolveAccount(context.Context) (*extcred.Account, error) {
	p.calls++
	return &extcred.Account{AppID: "metadata-app", Brand: extcred.BrandFeishu, SupportedIdentities: p.support}, nil
}
func (p *strictMetadataProvider) ResolveToken(context.Context, extcred.TokenSpec) (*extcred.Token, error) {
	p.t.Fatal("strict-mode discovery resolved a token")
	return nil, nil
}

func TestResolveStrictModeUsesOnlyAccountMetadata(t *testing.T) {
	for _, tt := range []struct {
		support extcred.IdentitySupport
		want    core.StrictMode
	}{{extcred.SupportsUser, core.StrictModeUser}, {extcred.SupportsBot, core.StrictModeBot}, {extcred.SupportsUser | extcred.SupportsBot, core.StrictModeOff}} {
		t.Run(string(tt.want), func(t *testing.T) {
			source := &strictMetadataProvider{t: t, support: tt.support}
			f := &Factory{Credential: credential.NewCredentialProvider([]extcred.Provider{source}, nil, nil, func() (*http.Client, error) {
				t.Fatal("strict-mode discovery initialized identity HTTP client")
				return nil, nil
			})}
			for i := 0; i < 2; i++ {
				if mode := f.ResolveStrictMode(context.Background()); mode != tt.want {
					t.Fatalf("strict mode %q, want %q", mode, tt.want)
				}
			}
			if source.calls != 1 {
				t.Fatalf("account metadata selection repeated: %d", source.calls)
			}
		})
	}
}
