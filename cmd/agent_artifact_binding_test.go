// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	extcred "github.com/larksuite/cli/extension/credential"
	credenv "github.com/larksuite/cli/extension/credential/env"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
	"github.com/spf13/cobra"
)

type artifactBindingRoundTrip func(*http.Request) (*http.Response, error)

func (f artifactBindingRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type artifactBindingAccount struct{ account *credential.Account }

func (a artifactBindingAccount) ResolveAccount(context.Context) (*credential.Account, error) {
	return a.account, nil
}

type artifactBindingToken struct{}

func (artifactBindingToken) ResolveToken(context.Context, credential.TokenSpec) (*credential.TokenResult, error) {
	return &credential.TokenResult{Token: "synthetic-selected-token"}, nil
}

type artifactBindingContextKey struct{}

func TestArtifactBindingUsesCachedEnvironmentUserAndTenant(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for key, value := range map[string]string{
		"LARKSUITE_CLI_APP_ID": "environment-app", "LARKSUITE_CLI_APP_SECRET": "",
		"LARKSUITE_CLI_USER_ACCESS_TOKEN": "synthetic-env-token", "LARKSUITE_CLI_TENANT_ACCESS_TOKEN": "",
		"LARKSUITE_CLI_TENANT_ACCESS_TOKEN_SOURCE": "", "LARKSUITE_CLI_BRAND": "lark",
		"LARKSUITE_CLI_DEFAULT_AS": "user", "LARKSUITE_CLI_STRICT_MODE": "user",
	} {
		t.Setenv(key, value)
	}
	calls := 0
	ctx := context.WithValue(context.Background(), artifactBindingContextKey{}, "current-command")
	provider := credential.NewCredentialProvider([]extcred.Provider{&credenv.Provider{}}, nil, nil, func() (*http.Client, error) {
		return &http.Client{Transport: artifactBindingRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Context().Value(artifactBindingContextKey{}) != "current-command" || r.URL.Path != "/open-apis/authen/v1/user_info" {
				t.Fatalf("wrong binding request context/path: %s", r.URL)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"environment-user","tenant_key":"observed-tenant"}}`))}, nil
		})}, nil
	})
	// Normal account enrichment happens before output. Artifact resolution must
	// reuse its evidence even though the separate config view has no user ID.
	if _, err := provider.ResolveAccount(ctx); err != nil {
		t.Fatal(err)
	}
	f, _, _, _ := cmdutil.TestFactory(t, &core.CliConfig{Brand: core.BrandLark, AppID: "environment-app", AppSecret: "synthetic-secret", ProfileName: "selected"})
	f.Credential = provider
	f.ResolvedIdentity = core.AsUser
	f.CurrentCommand = &cobra.Command{Use: "read"}
	f.CurrentCommand.SetContext(ctx)
	opts := configureAgentFoundation(f, &cobra.Command{})
	for i := 0; i < 2; i++ {
		binding, err := opts.ResolveBinding()
		if err != nil {
			t.Fatal(err)
		}
		if binding.Environment != "lark" || binding.Endpoint != core.ResolveEndpoints(core.BrandLark).Open || binding.UserID != "environment-user" || binding.TenantKey != "observed-tenant" || !binding.UserVerified || !binding.TenantVerified || binding.Profile != "selected" {
			t.Fatalf("wrong effective binding: %+v", binding)
		}
		encoded, err := json.Marshal(binding)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{"synthetic-env-token", "synthetic-secret", "access_token", "app_secret"} {
			if strings.Contains(string(encoded), secret) {
				t.Fatalf("binding leaked credentials: %s", encoded)
			}
		}
	}
	if calls != 1 {
		t.Fatalf("repeated identity requests: %d", calls)
	}
}

func TestArtifactBindingSelectedUserDoesNotFetchMissingTenant(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, &core.CliConfig{Brand: core.BrandFeishu, AppID: "selected-app", ProfileName: "selected"})
	f.Credential = credential.NewCredentialProvider(nil, artifactBindingAccount{&credential.Account{Brand: core.BrandFeishu, AppID: "selected-app", UserOpenId: "selected-user"}}, nil, func() (*http.Client, error) { t.Fatal("missing tenant caused identity lookup"); return nil, nil })
	f.ResolvedIdentity = core.AsUser
	opts := configureAgentFoundation(f, &cobra.Command{})
	binding, err := opts.ResolveBinding()
	if err != nil {
		t.Fatal(err)
	}
	if binding.UserID != "selected-user" || binding.UserVerified || binding.TenantVerified || binding.TenantKey != "" || binding.Endpoint != core.ResolveEndpoints(core.BrandFeishu).Open {
		t.Fatalf("unobserved verification claimed: %+v", binding)
	}
}

func TestArtifactBindingResolvesSelectedTokenWithCurrentContextOnce(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, &core.CliConfig{Brand: core.BrandFeishu, AppID: "selected-app"})
	calls := 0
	f.Credential = credential.NewCredentialProvider(nil, artifactBindingAccount{&credential.Account{Brand: core.BrandFeishu, AppID: "selected-app"}}, artifactBindingToken{}, func() (*http.Client, error) {
		return &http.Client{Transport: artifactBindingRoundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Context().Value(artifactBindingContextKey{}) != "current-command" {
				t.Fatal("output ignored current command context")
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"open_id":"selected-user"}}`))}, nil
		})}, nil
	})
	f.ResolvedIdentity = core.AsUser
	f.CurrentCommand = &cobra.Command{}
	f.CurrentCommand.SetContext(context.WithValue(context.Background(), artifactBindingContextKey{}, "current-command"))
	opts := configureAgentFoundation(f, &cobra.Command{})
	for i := 0; i < 2; i++ {
		binding, err := opts.ResolveBinding()
		if err != nil || binding.UserID != "selected-user" || !binding.UserVerified || binding.TenantVerified || binding.TenantKey != "" {
			t.Fatalf("selected-token binding: %+v, %v", binding, err)
		}
	}
	if calls != 1 {
		t.Fatalf("uncached binding lookup: %d", calls)
	}
}

func TestArtifactBindingBotDoesNotClaimConfiguredUserOrTenant(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := cmdutil.TestFactory(t, &core.CliConfig{Brand: core.BrandFeishu, AppID: "bot-app", UserOpenId: "other-user"})
	f.Credential = credential.NewCredentialProvider(nil, artifactBindingAccount{&credential.Account{}}, nil, func() (*http.Client, error) { t.Fatal("bot artifact resolved user credentials"); return nil, nil })
	f.ResolvedIdentity = core.AsBot
	binding, err := configureAgentFoundation(f, &cobra.Command{}).ResolveBinding()
	if err != nil || binding.Identity != "bot" || binding.UserID != "" || binding.UserVerified || binding.TenantKey != "" || binding.TenantVerified {
		t.Fatalf("invented bot user/tenant: %+v %v", binding, err)
	}
}
