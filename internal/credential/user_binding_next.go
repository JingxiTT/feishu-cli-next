// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/dpop"
)

// UserBinding projects non-secret identity evidence without changing config or
// account selection. Verified distinguishes a current user_info response from
// the existing persisted/provider account identity.
type UserBinding struct {
	OpenID, TenantKey string
	Verified          bool
}

// ResolveUserBinding reuses account enrichment and only queries user_info when
// no user identity was resolved. A missing tenant alone does not cause an extra
// network call. Resolution is cached for this credential-provider invocation.
func (p *CredentialProvider) ResolveUserBinding(ctx context.Context) (UserBinding, error) {
	p.userBindingOnce.Do(func() { p.userBinding, p.userBindingErr = p.resolveUserBinding(ctx) })
	return p.userBinding, p.userBindingErr
}
func (p *CredentialProvider) resolveUserBinding(ctx context.Context) (UserBinding, error) {
	account, err := p.ResolveAccount(ctx)
	if err != nil {
		return UserBinding{}, err
	}
	if account.UserOpenId != "" {
		return UserBinding{OpenID: account.UserOpenId, TenantKey: account.TenantKey, Verified: account.UserInfoVerified}, nil
	}
	token, err := p.ResolveToken(ctx, TokenSpec{Type: TokenTypeUAT, AppID: account.AppID})
	if err != nil {
		return UserBinding{}, errs.NewAuthenticationError(errs.SubtypeTokenMissing, "cannot resolve user identity for mutation evidence").WithCause(err)
	}
	if p.httpClient == nil {
		return UserBinding{}, errs.NewInternalError(errs.SubtypeUnknown, "user identity lookup has no HTTP client")
	}
	client, err := p.httpClient()
	if err != nil {
		return UserBinding{}, err
	}
	info, err := fetchUserInfo(dpop.WithBinding(core.WithCredentialSource(ctx, token.Source), token.DPoP), client, account.Brand, token.Token)
	if err != nil {
		return UserBinding{}, errs.NewNetworkError(errs.SubtypeNetworkTransport, "cannot verify user identity for mutation evidence").WithCause(err)
	}
	if info.OpenID == "" {
		return UserBinding{}, errs.NewInternalError(errs.SubtypeInvalidResponse, "user identity response has no open_id")
	}
	return UserBinding{OpenID: info.OpenID, TenantKey: info.TenantKey, Verified: true}, nil
}
