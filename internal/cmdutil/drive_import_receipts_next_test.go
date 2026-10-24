// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/nextattempt"
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
					ctx = nextattempt.WithDriveImportStatusRequest(ctx, "/open-apis/drive/v1/import_tasks/"+ticket)
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
		req, _ := http.NewRequestWithContext(nextattempt.WithDriveImportStatusRequest(context.Background(), "/open-apis/drive/v1/import_tasks/ticket"), "GET", endpoint, nil)
		if nextattempt.IsDriveImportStatusRequest(req) {
			t.Fatal("scoped marker changed an unrelated origin or path")
		}
	}
}

func TestDriveImportLocalProofErrorsKeepPendingWithoutPolling(t *testing.T) {
	for _, failure := range []string{"config", "request_context", "binding", "marshal"} {
		t.Run(failure, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			cfg := &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu, ProfileName: "selected"}
			f, _, _, _ := TestFactory(t, cfg)
			f.EnableAgentAttempts, f.ResolvedIdentity = true, core.AsBot
			dispatches := 0
			tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
				dispatches++
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"ticket":"synthetic-ticket"}}`))}, nil
			}))
			req, err := http.NewRequest("POST", core.ResolveEndpoints(cfg.Brand).Open+"/open-apis/drive/v1/import_tasks", strings.NewReader(`{"file_token":"source","file_extension":"md","type":"docx","point":{"mount_type":1,"mount_key":""}}`))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			store := f.AgentReceiptStore()
			before, err := store.Inspect(f.AgentAttemptSummary().Receipts[0].ID)
			if err != nil {
				t.Fatal(err)
			}
			if before.Evidence != "drive_import_task_pending" {
				t.Fatal("test requires captured pending task", before)
			}
			ctx := context.Background()
			proof := map[string]interface{}{"result": map[string]interface{}{"job_status": 0, "type": "docx", "token": "target"}}
			switch failure {
			case "config":
				f.Config = func() (*core.CliConfig, error) { return nil, errors.New("synthetic config failure") }
			case "request_context":
				ctx = nil // http.NewRequestWithContext rejects this before identity resolution.
			case "binding":
				f.ResolvedIdentity = ""
			case "marshal":
				proof["unsupported"] = make(chan int)
			}
			if err := f.ReconcileDriveImport(ctx, "synthetic-ticket", proof); err != nil {
				t.Fatalf("local proof error would enter domain poll retry: %v", err)
			}
			summary := f.AgentAttemptSummary()
			if dispatches != 1 || summary.Unknown != 1 || summary.Applied != 0 || summary.PersistenceFailed || f.AgentAttemptExitCode() != 22 {
				t.Fatalf("local proof failure dispatched or changed outcome: dispatches=%d summary=%+v", dispatches, summary)
			}
			persisted, err := store.Inspect(before.ID)
			if err != nil || !reflect.DeepEqual(persisted, before) || persisted.Outcome != nextattempt.Unknown {
				t.Fatalf("pending durable evidence changed: %+v %v", persisted, err)
			}
		})
	}
}
