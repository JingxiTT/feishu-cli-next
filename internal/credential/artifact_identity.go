// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package credential

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/errclass"
)

// ArtifactIdentity is observed account evidence, without credentials. Artifact
// reuse requires a tenant even when ordinary account selection does not.
type ArtifactIdentity struct{ OpenID, TenantKey string }

type artifactIdentityCache struct {
	once     sync.Once
	identity ArtifactIdentity
	err      error
}

// ResolveArtifactIdentity verifies the selected credential with a read-only
// identity endpoint, once per identity per invocation. It never changes accounts,
// grants scopes, or falls back to an unverified configured user or tenant.
func (p *CredentialProvider) ResolveArtifactIdentity(ctx context.Context, identity core.Identity) (ArtifactIdentity, error) {
	var cache *artifactIdentityCache
	switch identity {
	case core.AsUser:
		cache = &p.artifactUser
	case core.AsBot:
		cache = &p.artifactBot
	default:
		return ArtifactIdentity{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact identity must be user or bot")
	}
	cache.once.Do(func() { cache.identity, cache.err = p.resolveArtifactIdentity(ctx, identity) })
	return cache.identity, cache.err
}

func (p *CredentialProvider) resolveArtifactIdentity(ctx context.Context, identity core.Identity) (ArtifactIdentity, error) {
	account, err := p.ResolveAccount(ctx)
	if err != nil {
		return ArtifactIdentity{}, err
	}
	if account == nil || account.AppID == "" {
		return ArtifactIdentity{}, errs.NewConfigError(errs.SubtypeNotConfigured, "artifact identity requires a selected account")
	}
	if identity == core.AsUser && account.UserInfoVerified {
		if account.UserOpenId == "" || account.TenantKey == "" {
			return ArtifactIdentity{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact user identity lookup returned incomplete user or tenant evidence")
		}
		return ArtifactIdentity{OpenID: account.UserOpenId, TenantKey: account.TenantKey}, nil
	}
	kind, path := TokenTypeUAT, "/open-apis/authen/v1/user_info"
	if identity == core.AsBot {
		kind, path = TokenTypeTAT, "/open-apis/tenant/v2/tenant/query"
	}
	token, err := p.ResolveToken(ctx, TokenSpec{Type: kind, AppID: account.AppID})
	if err != nil {
		return ArtifactIdentity{}, err
	}
	if p.httpClient == nil {
		return ArtifactIdentity{}, errs.NewInternalError(errs.SubtypeUnknown, "artifact identity lookup has no HTTP client")
	}
	client, err := p.httpClient()
	if err != nil {
		return ArtifactIdentity{}, err
	}
	data, err := fetchArtifactIdentityData(core.WithCredentialSource(ctx, token.Source), client, account, identity, path, token.Token)
	if err != nil {
		return ArtifactIdentity{}, err
	}
	observed := ArtifactIdentity{}
	if identity == core.AsUser {
		observed.OpenID, _ = data["open_id"].(string)
		observed.TenantKey, _ = data["tenant_key"].(string)
		if observed.OpenID == "" {
			return ArtifactIdentity{}, errs.NewInternalError(errs.SubtypeInvalidResponse, "artifact user identity response has no open_id")
		}
		if account.UserOpenId != "" && account.UserOpenId != observed.OpenID {
			return ArtifactIdentity{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact credential user does not match the selected account")
		}
	} else {
		tenant, _ := data["tenant"].(map[string]any)
		observed.TenantKey, _ = tenant["tenant_key"].(string)
	}
	if observed.TenantKey == "" {
		return ArtifactIdentity{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact identity lookup returned no tenant_key; verified tenant evidence is required")
	}
	if account.TenantKey != "" && account.TenantKey != observed.TenantKey {
		return ArtifactIdentity{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact credential tenant does not match the selected account")
	}
	return observed, nil
}

// Both endpoints use the Open API envelope. Keep response bodies and bearer
// values out of local errors; the shared classifier retains structured scope
// failures without requesting additional privileges.
func fetchArtifactIdentityData(ctx context.Context, client *http.Client, account *Account, identity core.Identity, path, token string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, core.ResolveEndpoints(account.Brand).Open+path, nil)
	if err != nil {
		return nil, errs.NewInternalError(errs.SubtypeUnknown, "cannot create artifact identity request").WithCause(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errs.NewNetworkError(errs.SubtypeNetworkTransport, "cannot verify artifact identity").WithCause(err)
	}
	defer resp.Body.Close()
	var envelope map[string]any
	decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&envelope)
	if decodeErr == nil {
		if apiErr := errclass.BuildAPIError(envelope, errclass.ClassifyContext{Brand: string(account.Brand), AppID: account.AppID, Identity: string(identity)}); apiErr != nil {
			return nil, apiErr
		}
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, errs.NewAuthenticationError(errs.SubtypeTokenInvalid, "artifact identity lookup returned HTTP 401")
	case resp.StatusCode == http.StatusForbidden:
		return nil, errs.NewPermissionError(errs.SubtypePermissionDenied, "artifact identity lookup returned HTTP 403").WithIdentity(string(identity))
	case resp.StatusCode >= 500:
		return nil, errs.NewNetworkError(errs.SubtypeNetworkServer, "artifact identity lookup returned HTTP %d", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, errs.NewNetworkError(errs.SubtypeNetworkProtocol, "artifact identity lookup returned HTTP %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "artifact identity response is not valid JSON").WithCause(decodeErr)
	}
	code, ok := envelope["code"].(float64)
	if !ok || code != 0 {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "artifact identity response has no success code")
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "artifact identity response has no data")
	}
	return data, nil
}
