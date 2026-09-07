// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/larksuite/cli/internal/client"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
)

type requestIdentityTokens struct{}

func (requestIdentityTokens) ResolveToken(_ context.Context, spec credential.TokenSpec) (*credential.TokenResult, error) {
	return &credential.TokenResult{Token: "synthetic-private-" + string(spec.Type), Source: core.CredentialSourceLocal}, nil
}

func TestAgentSDKRequestsBindActualIdentityWithoutChangingInvocation(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	cfg := &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu, ProfileName: "selected-account", UserOpenId: "selected-user"}
	f, _, _, _ := TestFactory(t, cfg)
	f.EnableAgentAttempts, f.ResolvedIdentity = true, core.AsUser
	f.Credential = credential.NewCredentialProvider(nil, &testDefaultAcct{config: cfg}, requestIdentityTokens{}, nil)
	var overlap *sync.WaitGroup
	tr := f.agentAttemptTransport(attemptRoundTrip(func(req *http.Request) (*http.Response, error) {
		var body struct {
			Identity core.Identity `json:"identity"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("request payload was not preserved: %v", err)
		}
		identity, ok := core.RequestIdentityFromContext(req.Context())
		if !ok || identity != body.Identity {
			t.Error("effective identity does not match this request payload")
		}
		spec := credential.NewTokenSpec(body.Identity, cfg.AppID)
		if req.Header.Get("Authorization") != "Bearer synthetic-private-"+string(spec.Type) {
			t.Error("platform authorization does not match this request identity")
		}
		if overlap != nil {
			overlap.Done()
			overlap.Wait()
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"code":0}`))}, nil
	}))
	ac := &client.APIClient{
		Config: cfg, Credential: f.Credential, ErrOut: io.Discard, HTTP: &http.Client{Transport: tr},
		SDK: lark.NewClient("synthetic-app", "synthetic-secret", lark.WithEnableTokenCache(false), lark.WithLogLevel(larkcore.LogLevelError), lark.WithHttpClient(&http.Client{Transport: tr})),
	}
	// An inherited user identity must be shadowed by an explicit bot call.
	ctx := core.WithRequestIdentity(context.Background(), core.AsUser)
	run := func(identity core.Identity) error {
		_, err := ac.DoSDKRequest(ctx, &larkcore.ApiReq{HttpMethod: "POST", ApiPath: "/open-apis/fixture/write", Body: map[string]string{"identity": string(identity)}}, identity)
		return err
	}
	for _, identity := range []core.Identity{core.AsBot, core.AsUser, core.AsBot, core.AsUser} {
		if err := run(identity); err != nil {
			t.Fatal(err)
		}
		receipts := f.AgentAttemptSummary().Receipts
		if receipts[len(receipts)-1].Binding.Identity != string(identity) {
			t.Fatal("alternating request retained invocation identity")
		}
	}
	for _, identity := range []core.Identity{core.AsBot, core.AsUser} {
		resp, err := ac.DoStream(ctx, &larkcore.ApiReq{HttpMethod: "POST", ApiPath: "/open-apis/fixture/write", Body: map[string]string{"identity": string(identity)}}, identity)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		receipts := f.AgentAttemptSummary().Receipts
		if receipts[len(receipts)-1].Binding.Identity != string(identity) {
			t.Fatal("stream request retained invocation identity")
		}
	}
	const concurrent = 12
	overlap = &sync.WaitGroup{}
	overlap.Add(concurrent)
	errors := make(chan error, concurrent)
	for i := 0; i < concurrent; i++ {
		identity := core.AsUser
		if i%2 == 0 {
			identity = core.AsBot
		}
		go func() { errors <- run(identity) }()
	}
	for i := 0; i < concurrent; i++ {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	counts := map[string]int{}
	for _, receipt := range f.AgentAttemptSummary().Receipts {
		binding := receipt.Binding
		counts[binding.Identity]++
		if binding.Account != cfg.ProfileName || binding.AppID != cfg.AppID || (binding.Identity == "bot" && binding.UserID != "") || (binding.Identity == "user" && binding.UserID != cfg.UserOpenId) {
			t.Fatalf("incorrect per-request binding: %+v", binding)
		}
		raw, err := os.ReadFile(filepath.Join(f.AgentReceiptStore().Dir, receipt.ID+".json"))
		if err != nil || strings.Contains(string(raw), "synthetic-private-") {
			t.Fatal("receipt missing or contains a credential")
		}
	}
	if counts["user"] != 9 || counts["bot"] != 9 || f.ResolvedIdentity != core.AsUser {
		t.Fatalf("per-request identities changed invocation or were lost: %v %v", counts, f.ResolvedIdentity)
	}
}
