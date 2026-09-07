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
	"github.com/spf13/cobra"
)

func TestDeclaredReadRequestSkipsJournalOnlyForExactTuple(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu})
	f.EnableAgentAttempts = true
	f.ResolvedIdentity = core.AsBot
	f.CurrentCommand = &cobra.Command{Use: "write"}
	SetRisk(f.CurrentCommand, "write")
	calls := 0
	tr := f.agentAttemptTransport(attemptRoundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
	}))
	base := context.Background()
	marked := WithReadOnlyRequest(base, "POST", "/open-apis/fixture/fetch")
	tests := []struct {
		ctx          context.Context
		method, path string
		receipts     int
	}{
		{marked, "POST", "/open-apis/fixture/fetch", 0},
		{marked, "POST", "/open-apis/fixture/write", 1},
		{marked, "PATCH", "/open-apis/fixture/fetch", 2},
		{base, "POST", "/open-apis/fixture/fetch", 3},
	}
	for _, tt := range tests {
		req, err := http.NewRequestWithContext(tt.ctx, tt.method, "https://fixture.invalid"+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		records, err := f.AgentReceiptStore().List()
		if err != nil || len(records) != tt.receipts {
			t.Fatalf("journal for %s %s: %v %v", tt.method, tt.path, records, err)
		}
	}
	if calls != len(tests) {
		t.Fatal(calls)
	}
	if risk, _ := GetRisk(f.CurrentCommand); risk != "write" {
		t.Fatal("command risk changed")
	}
}
func TestDeclaredReadPreservesContextAndSkipsCredentialResolution(t *testing.T) {
	type key struct{}
	original, cancel := context.WithCancel(context.WithValue(context.Background(), key{}, "preserved"))
	defer cancel()
	marked := WithReadOnlyRequest(original, "POST", "/fetch")
	f := &Factory{EnableAgentAttempts: true, Config: func() (*core.CliConfig, error) { t.Fatal("read resolved mutation credentials"); return nil, nil }}
	tr := f.agentAttemptTransport(attemptRoundTrip(func(req *http.Request) (*http.Response, error) {
		if req.Context().Value(key{}) != "preserved" {
			t.Fatal("context value lost")
		}
		return nil, req.Context().Err()
	}))
	cancel()
	req, err := http.NewRequestWithContext(marked, "POST", "https://fixture.invalid/fetch", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(f.AgentAttemptSummary().Receipts) != 0 {
		t.Fatal("read journaled")
	}
	if original.Value(readOnlyRequestKey{}) != nil {
		t.Fatal("original context changed")
	}
}
