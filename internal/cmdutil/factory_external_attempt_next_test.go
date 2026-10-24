// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	exttransport "github.com/larksuite/cli/extension/transport"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
)

type gatewaySecretInterceptor struct{}

func (*gatewaySecretInterceptor) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	req.Header.Set("Authorization", "Bearer synthetic-gateway-token")
	req.Header.Set("X-Gateway-Signature", "synthetic-gateway-signature")
	return nil
}

func TestFactoryExternalUploadJournalsBeforeDispatchWithoutPersistingPayload(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		outcome string
	}{
		{"storage accepted", http.StatusNoContent, nextattempt.Applied},
		{"response lost", 0, nextattempt.Unknown},
		{"storage denied", http.StatusForbidden, nextattempt.Unapplied},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			t.Setenv("LARKSUITE_CLI_NO_PROXY", "1")
			exttransport.Register(&platformOnlyStubProvider{stubTransportProvider: &stubTransportProvider{interceptor: &gatewaySecretInterceptor{}}})
			t.Cleanup(func() { exttransport.Register(nil) })

			cfg := &core.CliConfig{AppID: "synthetic-app", AppSecret: "synthetic-app-secret", Brand: core.BrandLark, ProfileName: "selected-account", UserOpenId: "selected-user"}
			factory, _, _, _ := TestFactory(t, cfg)
			factory.EnableAgentAttempts = true
			factory.ResolvedIdentity = core.AsUser
			riskEnabled := false
			factory.HttpClient = cachedHttpClientFunc(factory, staticWorkspaceConfig{config: &core.MultiAppConfig{RiskControl: &riskEnabled}})
			base, err := factory.HttpClient()
			if err != nil {
				t.Fatal(err)
			}
			external, err := factory.ExternalHTTPClient()
			if err != nil {
				t.Fatal(err)
			}
			external.Timeout = 0
			if base.Timeout == 0 || external == base {
				t.Fatal("transfer timeout change mutated the cached invocation client")
			}

			const payload = "synthetic-private-upload-body"
			const uploadHeader = "synthetic-private-upload-header"
			const querySecret = "synthetic-presigned-query-secret"
			dispatched := make(chan nextattempt.Receipt, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				receipts, listErr := factory.AgentReceiptStore().List()
				if listErr != nil || len(receipts) != 1 {
					t.Errorf("before PUT dispatch receipts = %#v, %v, want one durable attempt", receipts, listErr)
					dispatched <- nextattempt.Receipt{}
				} else {
					dispatched <- receipts[0]
				}
				for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Gateway-Signature"} {
					if req.Header.Get(name) != "" {
						t.Errorf("external PUT leaked gateway header %s", name)
					}
				}
				body, readErr := io.ReadAll(req.Body)
				if readErr != nil || string(body) != payload || req.Header.Get("X-Upload-Metadata") != uploadHeader || req.URL.Query().Get("signature") != querySecret {
					t.Error("external PUT changed the signed request payload")
				}
				if tt.status == 0 {
					conn, _, hijackErr := w.(http.Hijacker).Hijack()
					if hijackErr != nil {
						t.Error(hijackErr)
						return
					}
					conn.Close()
					return
				}
				w.WriteHeader(tt.status)
			}))
			t.Cleanup(server.Close)

			uploadURL := server.URL + "/synthetic-private-object?signature=" + querySecret
			req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, uploadURL, strings.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			req = req.WithContext(WithStoragePutRequest(req.Context(), req.URL.Path))
			req.Header.Set("X-Upload-Metadata", uploadHeader)
			resp, err := external.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
			if tt.status == 0 && err == nil || tt.status != 0 && err != nil {
				t.Fatalf("PUT error = %v for status %d", err, tt.status)
			}
			var before nextattempt.Receipt
			select {
			case before = <-dispatched:
			default:
				t.Fatal("upload request did not reach the external server")
			}
			if before.ID == "" || before.Method != http.MethodPut || before.Outcome != nextattempt.Unknown || before.Evidence != "dispatch_started" {
				t.Fatalf("pre-dispatch receipt = %#v", before)
			}
			summary := factory.AgentAttemptSummary()
			if len(summary.Receipts) != 1 || summary.Receipts[0].ID != before.ID || summary.Outcome != tt.outcome {
				t.Fatalf("final receipt summary = %#v, want %s", summary, tt.outcome)
			}
			binding := summary.Receipts[0].Binding
			if binding.AppID != cfg.AppID || binding.Account != cfg.ProfileName || binding.UserID != cfg.UserOpenId || binding.Identity != "user" || binding.Environment != core.ResolveEndpoints(cfg.Brand).Open {
				t.Fatalf("receipt is not bound to the effective invocation account: %#v", binding)
			}
			stored, err := os.ReadFile(filepath.Join(factory.AgentReceiptStore().Dir, before.ID+".json"))
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{server.URL, "synthetic-private-object", querySecret, payload, uploadHeader, cfg.AppSecret, "synthetic-gateway-token", "synthetic-gateway-signature"} {
				if strings.Contains(string(stored), secret) {
					t.Fatal("durable receipt persisted request payload or credential material")
				}
			}
		})
	}
}
