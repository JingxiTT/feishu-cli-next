// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
	"github.com/larksuite/cli/internal/keychain"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

type artifactLifecycleTransport func(*http.Request) (*http.Response, error)

func (f artifactLifecycleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type artifactLifecycleKeychain struct{}

func (artifactLifecycleKeychain) Get(string, string) (string, error) {
	return "synthetic-config-secret", nil
}
func (artifactLifecycleKeychain) Set(string, string, string) error { return nil }
func (artifactLifecycleKeychain) Remove(string, string) error      { return nil }

// UAT storage is a package-level OS keychain dependency, so inject only that
// token boundary. The real default account provider still loads the persisted
// user ID and keychain-backed app secret; identity evidence must come from HTTP.
type artifactLifecycleUAT struct{}

func (artifactLifecycleUAT) ResolveToken(_ context.Context, spec credential.TokenSpec) (*credential.TokenResult, error) {
	if spec.Type != credential.TokenTypeUAT {
		return nil, errs.NewInternalError(errs.SubtypeUnknown, "unexpected synthetic token kind")
	}
	return &credential.TokenResult{Token: "synthetic-uat"}, nil
}

func TestArtifactDefaultConfigExportReuseLifecycle(t *testing.T) {
	for _, identity := range []core.Identity{core.AsUser, core.AsBot} {
		t.Run(string(identity), func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			cwd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err = os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(cwd) })
			cfg := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "config-app", AppSecret: core.SecretInput{Ref: &core.SecretRef{Source: "keychain", ID: "appsecret:config-app"}}, Brand: core.BrandFeishu, DPoPMode: core.DPoPModeDisabled, Users: []core.AppUser{{UserOpenId: "persisted-user"}}}}}
			if err = core.SaveMultiAppConfig(cfg); err != nil {
				t.Fatal(err)
			}
			tenant, user, denied := "tenant", "persisted-user", false
			identityReads, tokenReads := 0, 0
			client := func() (*http.Client, error) {
				return &http.Client{Transport: artifactLifecycleTransport(func(r *http.Request) (*http.Response, error) {
					body := ""
					switch r.URL.Path {
					case core.OAuthTokenV3Path:
						tokenReads++
						if r.Method != "POST" {
							t.Fatal("wrong token method")
						}
						_ = r.ParseForm()
						if r.Form.Get("client_id") != "config-app" || r.Form.Get("client_secret") != "synthetic-config-secret" || r.Form.Get("scope") != "" {
							t.Fatal("wrong account or unsolicited scope grant")
						}
						body = `{"code":0,"access_token":"synthetic-tat","token_type":"Bearer","expires_in":7200}`
					case "/open-apis/authen/v1/user_info", "/open-apis/tenant/v2/tenant/query":
						identityReads++
						if r.Method != "GET" {
							t.Fatal("identity lookup must be read-only")
						}
						if identity == core.AsUser {
							if r.URL.Path != "/open-apis/authen/v1/user_info" || r.Header.Get("Authorization") != "Bearer synthetic-uat" {
								t.Fatal("wrong user credential")
							}
							body = `{"code":0,"data":{"open_id":"` + user + `","tenant_key":"` + tenant + `"}}`
						} else {
							if r.URL.Path != "/open-apis/tenant/v2/tenant/query" || r.Header.Get("Authorization") != "Bearer synthetic-tat" {
								t.Fatal("wrong bot credential")
							}
							body = `{"code":0,"data":{"tenant":{"tenant_key":"` + tenant + `"}}}`
						}
						if denied {
							body = `{"code":99991672,"msg":"permission denied","error":{"permission_violations":[{"type":"action","subject":"tenant:tenant:readonly"}]}}`
						}
					default:
						t.Fatalf("unexpected network request %s", r.URL)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}, nil
			}
			newFactory := func() *Factory {
				acct := credential.NewDefaultAccountProvider(func() keychain.KeychainAccess { return artifactLifecycleKeychain{} }, "", core.ProfileFromConfig)
				var tokens credential.DefaultTokenResolver = credential.NewDefaultTokenProvider(acct, client, io.Discard)
				if identity == core.AsUser {
					tokens = artifactLifecycleUAT{}
				}
				provider := credential.NewCredentialProvider(nil, acct, tokens, client)
				return &Factory{Credential: provider, Config: func() (*core.CliConfig, error) {
					a, e := provider.ResolveAccount(context.Background())
					if e != nil {
						return nil, e
					}
					return a.ToCliConfig(), nil
				}}
			}
			exporting := newFactory()
			path := filepath.Join(dir, "result.json")
			opts := &output.AgentOptions{SaveResult: path, FileIO: &localfileio.LocalFileIO{}, ResolveBinding: func() (output.ArtifactBinding, error) {
				return exporting.ResolveArtifactBinding(context.Background(), identity)
			}}
			if err = opts.Validate(); err != nil {
				t.Fatal(err)
			}
			var stdout bytes.Buffer
			if err = output.WriteJSON(output.WithAgentOptions(&stdout, opts), json.RawMessage(`{"ok":true,"data":{"items":[{"id":"one","value":9007199254740993}],"has_more":false}}`)); err != nil {
				t.Fatal(err)
			}
			saved, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var artifact output.ResultArtifact
			if err = json.Unmarshal(saved, &artifact); err != nil {
				t.Fatal(err)
			}
			if !artifact.Binding.TenantVerified || (identity == core.AsUser && !artifact.Binding.UserVerified) {
				t.Fatalf("export lost verification: %+v", artifact.Binding)
			}
			for _, secret := range []string{"synthetic-config-secret", "synthetic-uat", "synthetic-tat"} {
				if bytes.Contains(saved, []byte(secret)) || strings.Contains(stdout.String(), secret) {
					t.Fatal("artifact leaked credential")
				}
			}
			importing := newFactory()
			for i := 0; i < 2; i++ {
				current, e := importing.ResolveArtifactBinding(context.Background(), identity)
				if e != nil {
					t.Fatal(e)
				}
				if e = output.ValidateArtifactBinding(artifact.Binding, current, "--artifact", true); e != nil {
					t.Fatal(e)
				}
			}
			if identityReads != 2 {
				t.Fatalf("identity lookups not cached per invocation: %d", identityReads)
			}
			if identity == core.AsBot && tokenReads != 2 {
				t.Fatalf("bot did not use normal token lifecycle: %d", tokenReads)
			}
			tenant = "different-tenant"
			current, e := newFactory().ResolveArtifactBinding(context.Background(), identity)
			if e != nil {
				t.Fatal(e)
			}
			if e = output.ValidateArtifactBinding(artifact.Binding, current, "--artifact", true); !errs.IsValidation(e) {
				t.Fatalf("tenant change accepted: %v", e)
			}
			if identity == core.AsUser {
				user = "different-user"
				if _, e = newFactory().ResolveArtifactBinding(context.Background(), identity); !errs.IsValidation(e) {
					t.Fatalf("persisted user mismatch accepted: %v", e)
				}
				user = "persisted-user"
			}
			denied = true
			refused := newFactory()
			opts.SaveResult = filepath.Join(dir, "denied.json")
			opts.ResolveBinding = func() (output.ArtifactBinding, error) {
				return refused.ResolveArtifactBinding(context.Background(), identity)
			}
			stdout.Reset()
			if e = output.WriteJSON(output.WithAgentOptions(&stdout, opts), json.RawMessage(`{"ok":true,"data":{}}`)); !errs.IsPermission(e) {
				t.Fatalf("permission failure hidden: %v", e)
			}
			if _, e = os.Stat(opts.SaveResult); !os.IsNotExist(e) || stdout.Len() != 0 {
				t.Fatal("failed verification created artifact or success output")
			}
		})
	}
}
