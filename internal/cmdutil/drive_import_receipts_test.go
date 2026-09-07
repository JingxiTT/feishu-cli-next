// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/core"
)

func TestDriveImportReconciliationUsesActualRequestIdentity(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	cfg := &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu, UserOpenId: "user", ProfileName: "selected"}
	f, _, _, _ := TestFactory(t, cfg)
	f.EnableAgentAttempts, f.ResolvedIdentity = true, core.AsUser
	dispatches := 0
	tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
		dispatches++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"ticket":"synthetic-ticket"}}`))}, nil
	}))
	botCtx := core.WithRequestIdentity(context.Background(), core.AsBot)
	req, err := http.NewRequestWithContext(botCtx, "POST", core.ResolveEndpoints(cfg.Brand).Open+"/open-apis/drive/v1/import_tasks", strings.NewReader(`{"file_token":"source","file_extension":"md","type":"docx","point":{"mount_type":1,"mount_key":""}}`))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	proof := map[string]interface{}{"result": map[string]interface{}{"job_status": 0, "type": "docx", "token": "target"}}
	if err := f.ReconcileDriveImport(context.Background(), "synthetic-ticket", proof); err != nil || f.AgentAttemptSummary().Unknown != 1 {
		t.Fatal("different invocation identity reconciled bot import", err)
	}
	if err := f.ReconcileDriveImport(botCtx, "synthetic-ticket", proof); err != nil || f.AgentAttemptSummary().Applied != 1 || f.AgentAttemptExitCode() != 0 {
		t.Fatal("matching bot proof failed to reconcile", f.AgentAttemptSummary(), err)
	}
	if dispatches != 1 || f.ResolvedIdentity != core.AsUser {
		t.Fatal("reconciliation dispatched or changed invocation identity")
	}
}

func TestStandaloneDriveImportStatusDoesNotResolveBinding(t *testing.T) {
	f := &Factory{EnableAgentAttempts: true, Config: func() (*core.CliConfig, error) {
		t.Fatal("standalone status resolved mutation binding")
		return nil, nil
	}}
	if err := f.ReconcileDriveImport(context.Background(), "other-invocation", map[string]interface{}{"result": map[string]interface{}{"job_status": 0, "type": "docx", "token": "target"}}); err != nil {
		t.Fatal(err)
	}
}

func TestDriveImportStatusRedirectCannotSupplyCompletionProof(t *testing.T) {
	for _, ticket := range []string{"ticket", "encoded%2Fticket"} {
		native := "https://open.feishu.cn/open-apis/drive/v1/import_tasks/" + ticket
		for _, target := range []string{"https://open.feishu.cn/open-apis/drive/v1/import_tasks/other", "https://other.example.test/completed"} {
			for _, marked := range []bool{false, true} {
				targetCalls := 0
				client := &http.Client{CheckRedirect: safeRedirectPolicy, Transport: attemptRoundTrip(func(req *http.Request) (*http.Response, error) {
					if req.URL.String() == native {
						return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{target}}, Body: http.NoBody}, nil
					}
					targetCalls++
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"result":{"job_status":0,"type":"docx","token":"target"}}}`))}, nil
				})}
				ctx := context.Background()
				if marked {
					ctx = WithDriveImportStatusRequest(ctx, "/open-apis/drive/v1/import_tasks/"+ticket)
				}
				req, err := http.NewRequestWithContext(ctx, "GET", native, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				if marked && (resp.StatusCode != 302 || targetCalls != 0) || !marked && (resp.StatusCode != 200 || targetCalls != 1) {
					t.Fatalf("marked=%v status=%d redirected calls=%d", marked, resp.StatusCode, targetCalls)
				}
			}
		}
	}
	for _, endpoint := range []string{"https://other.example.test/open-apis/drive/v1/import_tasks/ticket", "https://open.feishu.cn/open-apis/drive/v1/import_tasks/other", "https://open.feishu.cn/open-apis/drive/v1/export_tasks/ticket"} {
		req, _ := http.NewRequestWithContext(WithDriveImportStatusRequest(context.Background(), "/open-apis/drive/v1/import_tasks/ticket"), "GET", endpoint, nil)
		if isDriveImportStatusRequest(req) {
			t.Fatal("scoped marker changed an unrelated origin or path")
		}
	}
}
