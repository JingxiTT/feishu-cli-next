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

const docsTaskCompletion = `{"code":0,"data":{"task":{"task_id":"synthetic-task","status":"succeeded","result":{"create_document":"{\"document\":{\"document_id\":\"synthetic-document\"}}"}}}}`

func TestDocsCreateReconciliationUsesActualRequestIdentity(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	cfg := &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu, UserOpenId: "user", ProfileName: "selected"}
	f, _, _, _ := TestFactory(t, cfg)
	f.EnableAgentAttempts, f.ResolvedIdentity = true, core.AsUser
	calls := 0
	tr := f.agentAttemptTransport(attemptRoundTrip(func(req *http.Request) (*http.Response, error) {
		calls++
		body := docsTaskCompletion
		if req.Method == "POST" {
			body = `{"code":0,"data":{"task":{"task_id":"synthetic-task","status":"processing"}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	origin := core.ResolveEndpoints(cfg.Brand).Open
	botCtx := core.WithRequestIdentity(context.Background(), core.AsBot)
	req, _ := http.NewRequestWithContext(botCtx, "POST", origin+"/open-apis/docs_ai/v1/documents", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	for _, ctx := range []context.Context{context.Background(), botCtx} {
		req, _ = http.NewRequestWithContext(ctx, "GET", origin+"/open-apis/docs_ai/v1/async_tasks/synthetic-task", nil)
		resp, err = tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		wantApplied := ctx == botCtx
		if (f.AgentAttemptSummary().Applied == 1) != wantApplied {
			t.Fatal("request identity did not scope completion", f.AgentAttemptSummary())
		}
	}
	if calls != 3 || f.ResolvedIdentity != core.AsUser || f.AgentAttemptExitCode() != 0 {
		t.Fatal("completion changed calls or invocation identity", calls, f.AgentAttemptSummary())
	}
}

func TestDocsCreateRedirectCannotSupplyCompletionProof(t *testing.T) {
	for _, target := range []string{"https://other.invalid/completed", core.ResolveEndpoints(core.BrandFeishu).Open + "/open-apis/docs_ai/v1/async_tasks/other-task"} {
		t.Run(target, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			cfg := &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu, ProfileName: "selected"}
			f, _, _, _ := TestFactory(t, cfg)
			f.EnableAgentAttempts, f.ResolvedIdentity = true, core.AsBot
			origin := core.ResolveEndpoints(cfg.Brand).Open
			tr := f.agentAttemptTransport(attemptRoundTrip(func(req *http.Request) (*http.Response, error) {
				if req.Method == "POST" {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"task":{"task_id":"synthetic-task","status":"processing"}}}`))}, nil
				}
				if req.URL.String() == target {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(docsTaskCompletion))}, nil
				}
				return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{target}}, Body: http.NoBody}, nil
			}))
			client := &http.Client{Transport: tr, CheckRedirect: safeRedirectPolicy}
			req, _ := http.NewRequest("POST", origin+"/open-apis/docs_ai/v1/documents", nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			resp, err = client.Get(origin + "/open-apis/docs_ai/v1/async_tasks/synthetic-task")
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if f.AgentAttemptSummary().Unknown != 1 || f.AgentAttemptExitCode() != 22 {
				t.Fatal("redirect upgraded pending task", f.AgentAttemptSummary())
			}
		})
	}
}
