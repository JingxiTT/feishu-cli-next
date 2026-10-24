// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
	"github.com/larksuite/cli/internal/output"
	"github.com/spf13/cobra"
)

type attemptRoundTrip func(*http.Request) (*http.Response, error)

func (f attemptRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestAgentAttemptsReadAndAuthDoNotResolveCredentials(t *testing.T) {
	tests := []struct {
		name, method, path, risk string
		enabled                  bool
	}{
		{"legacy", "POST", "/open-apis/docs/create", "write", false},
		{"read", "GET", "/open-apis/docs/documents", "write", true},
		{"post read", "POST", "/open-apis/search", "read", true},
		{"auth inside mutation", "POST", "/open-apis/auth/v3/tenant_access_token/internal", "write", true},
		{"oauth inside mutation", "POST", "/open-apis/authen/v2/oauth/token", "write", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{EnableAgentAttempts: tt.enabled, CurrentCommand: &cobra.Command{}, Config: func() (*core.CliConfig, error) { t.Fatal("credential recursion"); return nil, nil }}
			SetRisk(f.CurrentCommand, tt.risk)
			calls := 0
			tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
			}))
			req, _ := http.NewRequest(tt.method, "https://fixture.invalid"+tt.path, nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if calls != 1 || len(f.AgentAttemptSummary().Receipts) != 0 {
				t.Fatalf("%d %+v", calls, f.AgentAttemptSummary())
			}
		})
	}
}
func TestAgentAttemptBindingAndArbitraryMethod(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "test-app", Brand: core.BrandLark, ProfileName: "selected", UserOpenId: "verified-user"})
	f.EnableAgentAttempts = true
	f.ResolvedIdentity = core.AsUser
	tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
	}))
	req, _ := http.NewRequestWithContext(context.Background(), "CUSTOM", "https://fixture.invalid/write", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	s := f.AgentAttemptSummary()
	if len(s.Receipts) != 1 {
		t.Fatal(s)
	}
	b := s.Receipts[0].Binding
	if b.Account != "selected" || b.AppID != "test-app" || b.UserID != "verified-user" || b.Identity != "user" || b.Environment != core.ResolveEndpoints(core.BrandLark).Open || b.TenantVerified {
		t.Fatal(b)
	}
}

func TestFinalizeAgentAttemptPreservesOriginalOrReportsEvidence(t *testing.T) {
	original := errs.NewPermissionError(errs.SubtypeMissingScope, "synthetic denial").WithMissingScopes("synthetic:write")
	for _, tt := range []struct {
		name     string
		enabled  bool
		receipts []nextattempt.Receipt
		exit     int
	}{
		{"legacy", false, []nextattempt.Receipt{{ID: "id", Outcome: nextattempt.Unknown}}, 0},
		{"no attempts", true, nil, 0},
		{"applied", true, []nextattempt.Receipt{{ID: "id", Outcome: nextattempt.Applied}}, 0},
		{"unapplied", true, []nextattempt.Receipt{{ID: "id", Outcome: nextattempt.Unapplied}}, 20},
		{"partial", true, []nextattempt.Receipt{{ID: "a", Outcome: nextattempt.Applied}, {ID: "b", Outcome: nextattempt.Unapplied}}, 21},
		{"unknown", true, []nextattempt.Receipt{{ID: "id", Outcome: nextattempt.Unknown}}, 22},
		{"persistence", true, []nextattempt.Receipt{{ID: "id", Outcome: nextattempt.Applied, PersistenceFailed: true}}, 22},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{EnableAgentAttempts: tt.enabled}
			for _, receipt := range tt.receipts {
				f.agentAttempts.Add(receipt)
			}
			for _, cause := range []error{nil, original} {
				err := f.FinalizeAgentAttempt(cause)
				if tt.exit == 0 {
					if err != cause {
						t.Fatalf("changed original: %v", err)
					}
					continue
				}
				var outcome *errs.MutationOutcomeError
				if !errors.As(err, &outcome) || output.ExitCodeOf(err) != tt.exit {
					t.Fatalf("missing outcome: %v", err)
				}
				if cause != nil && !errors.Is(err, cause) {
					t.Fatal("lost original error")
				}
				if len(outcome.Receipts) != len(tt.receipts) {
					t.Fatal(outcome)
				}
			}
		})
	}
}
