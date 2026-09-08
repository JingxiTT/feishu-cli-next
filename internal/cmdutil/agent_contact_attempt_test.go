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

	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
	"github.com/spf13/cobra"
)

func TestContactBasicBatchReadRequiresExactTrustedEndpoint(t *testing.T) {
	const path = "/open-apis/contact/v3/users/basic_batch"
	for _, tt := range []struct {
		method, endpoint string
		read             bool
	}{
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Open + path, true},
		{"POST", core.ResolveEndpoints(core.BrandLark).Open + ":443" + path + "?user_id_type=open_id", true},
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Accounts + path, false},
		{"POST", "https://fixture.invalid" + path, false},
		{"POST", "https://open.feishu.cn.example.test" + path, false},
		{"POST", "http://open.feishu.cn" + path, false},
		{"POST", "https://open.feishu.cn:444" + path, false},
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Open + path + "/other", false},
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Open + path + "/", false},
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Open + "/open-apis/contact/v3/users/%62asic_batch", false},
		{"POST", core.ResolveEndpoints(core.BrandFeishu).Open + "/open-apis/contact/v3/users/batch_update", false},
		{"PATCH", core.ResolveEndpoints(core.BrandFeishu).Open + path, false},
		{"CUSTOM", core.ResolveEndpoints(core.BrandFeishu).Open + path, false},
	} {
		t.Run(tt.method+tt.endpoint, func(t *testing.T) {
			f := &Factory{EnableAgentAttempts: true, CurrentCommand: &cobra.Command{Use: "api"}}
			SetRisk(f.CurrentCommand, "write")
			tr := f.agentAttemptTransport(nil).(*nextattempt.Transport)
			req, err := http.NewRequest(tt.method, tt.endpoint, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tr.IsMutation(req) == tt.read {
				t.Fatal("Contact read bypass changed an unrelated request classification")
			}
			if tt.read {
				forced := req.WithContext(WithMutationRequest(req.Context(), req.Method, req.URL.Path))
				if !tr.IsMutation(forced) {
					t.Fatal("Contact read bypass overrode an explicit mutation declaration")
				}
			}
		})
	}
}

func TestContactBasicBatchReadPreservesResponseAndTransportError(t *testing.T) {
	for _, transportErr := range []error{nil, context.Canceled} {
		f := &Factory{EnableAgentAttempts: true, Config: func() (*core.CliConfig, error) {
			t.Fatal("Contact read resolved mutation credentials")
			return nil, nil
		}}
		const payload = `{"code":0,"data":{"users":[{"user_id":"ou_synthetic","name":"测试"}]}}`
		calls := 0
		tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(payload))}, transportErr
		}))
		req, _ := http.NewRequest("POST", core.ResolveEndpoints(core.BrandFeishu).Open+"/open-apis/contact/v3/users/basic_batch", nil)
		resp, err := tr.RoundTrip(req)
		if !errors.Is(err, transportErr) {
			t.Fatalf("transport error changed: %v", err)
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil || string(data) != payload || calls != 1 {
			t.Fatalf("read response changed: %s, %v; calls=%d", data, readErr, calls)
		}
		if len(f.AgentAttemptSummary().Receipts) != 0 || f.AgentAttemptExitCode() != 0 || f.FinalizeAgentAttempt(err) != err {
			t.Fatal("read acquired a mutation receipt or changed final error")
		}
	}
}
