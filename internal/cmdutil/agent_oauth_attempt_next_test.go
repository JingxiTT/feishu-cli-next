// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	exttransport "github.com/larksuite/cli/extension/transport"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/envvars"
	"github.com/larksuite/cli/internal/nextattempt"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/spf13/cobra"
)

type oauthFixtureRoute struct {
	t      *testing.T
	server *url.URL
	brand  core.LarkBrand
}

func (route *oauthFixtureRoute) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	endpoints := core.ResolveEndpoints(route.brand)
	origin := req.URL.Scheme + "://" + req.URL.Host
	if origin != endpoints.Accounts && origin != endpoints.Open {
		route.t.Error("unexpected logical host in isolated OAuth fixture")
	}
	// Redirect only the test transport to loopback; the outer Factory receipt
	// classifier still observes the production logical URL before this rewrite.
	req.URL.Scheme, req.URL.Host = route.server.Scheme, route.server.Host
	return nil
}

func TestFactoryColdUnifiedOAuthTokenIsNotContentMutation(t *testing.T) {
	for _, brand := range []core.LarkBrand{core.BrandFeishu, core.BrandLark} {
		for _, codeField := range []string{"", `"code":0,`} {
			t.Run(string(brand)+codeField, func(t *testing.T) {
				for _, name := range []string{envvars.CliAppID, envvars.CliAppSecret, envvars.CliUserAccessToken, envvars.CliTenantAccessToken, envvars.CliTenantAccessTokenSource, envvars.CliBrand, envvars.CliDefaultAs, envvars.CliStrictMode, envvars.CliProfile} {
					t.Setenv(name, "")
				}
				t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
				t.Setenv("LARKSUITE_CLI_NO_PROXY", "1")
				noRiskControl := false
				if err := core.SaveMultiAppConfig(&core.MultiAppConfig{CurrentApp: "selected", RiskControl: &noRiskControl, Apps: []core.AppConfig{{Name: "selected", AppId: "synthetic-app", AppSecret: core.PlainSecret("synthetic-private-secret"), Brand: brand}}}); err != nil {
					t.Fatal(err)
				}
				var factory *Factory
				var tokens, writes atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					receipts, err := factory.AgentReceiptStore().List()
					if err != nil {
						t.Error(err)
					}
					switch req.URL.Path {
					case core.OAuthTokenV3Path:
						tokens.Add(1)
						if req.Method != "POST" || req.ParseForm() != nil || req.Form.Get("grant_type") != "client_credentials" || req.Form.Get("client_id") != "synthetic-app" || req.Form.Get("client_secret") != "synthetic-private-secret" || len(receipts) != 0 {
							t.Error("cold token request changed or created a mutation receipt")
						}
						_, _ = io.WriteString(w, `{`+codeField+`"access_token":"synthetic-private-tat","token_type":"Bearer","expires_in":7200}`)
					case "/open-apis/fixture/write":
						writes.Add(1)
						if req.Header.Get("Authorization") != "Bearer synthetic-private-tat" || len(receipts) != 1 || receipts[0].Outcome != nextattempt.Unknown || receipts[0].Binding.Identity != "bot" {
							t.Error("content write lacks selected token or durable pre-dispatch evidence")
						}
						_, _ = io.WriteString(w, `{"code":0,"data":{"id":"written"}}`)
					default:
						t.Error("unexpected request in isolated OAuth fixture")
						w.WriteHeader(404)
					}
				}))
				t.Cleanup(server.Close)
				target, err := url.Parse(server.URL)
				if err != nil {
					t.Fatal(err)
				}
				exttransport.Register(&platformOnlyStubProvider{stubTransportProvider: &stubTransportProvider{interceptor: &oauthFixtureRoute{t: t, server: target, brand: brand}}})
				t.Cleanup(func() { exttransport.Register(nil) })
				factory = NewDefault(nil, InvocationContext{})
				factory.EnableAgentAttempts = true
				factory.CurrentCommand = &cobra.Command{Use: "write"}
				SetRisk(factory.CurrentCommand, "write")
				// No global identity is resolved yet: credential acquisition must not
				// recurse into receipt binding. The final SDK write carries bot identity.
				ac, err := factory.NewAPIClient()
				if err != nil {
					t.Fatal(err)
				}
				_, err = ac.DoSDKRequest(context.Background(), &larkcore.ApiReq{HttpMethod: "POST", ApiPath: "/open-apis/fixture/write", Body: map[string]string{"text": "synthetic-content"}}, core.AsBot)
				if err != nil {
					t.Fatal(err)
				}
				summary := factory.AgentAttemptSummary()
				if tokens.Load() != 1 || writes.Load() != 1 || len(summary.Receipts) != 1 || summary.Applied != 1 || factory.FinalizeAgentAttempt(nil) != nil {
					t.Fatalf("cold token/write outcome: tokens=%d writes=%d summary=%+v", tokens.Load(), writes.Load(), summary)
				}
				raw, err := os.ReadFile(filepath.Join(factory.AgentReceiptStore().Dir, summary.Receipts[0].ID+".json"))
				if err != nil || strings.Contains(string(raw), "synthetic-private") || strings.Contains(string(raw), core.OAuthTokenV3Path) {
					t.Fatal("receipt missing or contains credential acquisition evidence")
				}
			})
		}
	}
}

func TestUnifiedOAuthBypassRequiresExactTrustedEndpoint(t *testing.T) {
	for _, tt := range []struct {
		method, endpoint string
		skip             bool
	}{
		{"POST", "https://accounts.feishu.cn/oauth/v3/token", true},
		{"POST", "https://accounts.larksuite.com:443/oauth/v3/token", true},
		{"POST", "https://open.feishu.cn/oauth/v3/token", false},
		{"POST", "https://accounts.example.test/oauth/v3/token", false},
		{"POST", "https://accounts.feishu.cn.example.test/oauth/v3/token", false},
		{"POST", "http://accounts.feishu.cn/oauth/v3/token", false},
		{"POST", "https://accounts.feishu.cn:444/oauth/v3/token", false},
		{"POST", "https://accounts.feishu.cn/oauth/v3/token/other", false},
		{"POST", "https://accounts.feishu.cn/oauth/v3/%74oken", false},
		{"PUT", "https://accounts.feishu.cn/oauth/v3/token", false},
	} {
		t.Run(tt.endpoint+tt.method, func(t *testing.T) {
			f, _, _, _ := TestFactory(t, nil)
			f.EnableAgentAttempts = true
			tr := f.agentAttemptTransport(nil).(*nextattempt.Transport)
			req, err := http.NewRequest(tt.method, tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tr.IsMutation(req) == tt.skip {
				t.Fatal("token bypass changed an unrelated request classification")
			}
		})
	}
}
