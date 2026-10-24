// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/internal/transport"
	"github.com/spf13/cobra"
)

type hostMutationRoundTrip func(*http.Request) (*http.Response, error)

func (f hostMutationRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAgentHostExecutionReturnsUnknownMutation(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	f := cmdutil.NewDefault(cmdutil.NewIOStreams(strings.NewReader(""), &stdout, &stderr), cmdutil.InvocationContext{})
	f.Config = func() (*core.CliConfig, error) {
		return &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu}, nil
	}
	f.ResolvedIdentity = core.AsBot
	client, err := f.HttpClient()
	if err != nil {
		t.Fatal(err)
	}
	client.Transport = client.Transport.(transport.RoundTripperDecorator).WithBaseRoundTripper(hostMutationRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"accepted":true}}`))}, nil
	}))
	root := &cobra.Command{Use: "host", SilenceUsage: true, SilenceErrors: true}
	child := &cobra.Command{Use: "mutate", RunE: func(c *cobra.Command, args []string) error {
		req, err := http.NewRequestWithContext(c.Context(), "POST", "https://fixture.invalid/create", nil)
		if err != nil {
			return err
		}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		response.Body.Close()
		return output.WriteJSON(f.IOStreams.Out, map[string]any{"ok": true, "data": map[string]any{"accepted": true}})
	}}
	root.AddCommand(child)
	f.CurrentCommand = child
	cmdutil.SetRisk(child, "write")
	// These are the same two boundary functions used by Build(WithAgentMode()).
	opts := configureAgentFoundation(f, root)
	finalizeAgentFoundation(f, root, opts)
	root.SetArgs([]string{"mutate"})
	root.SetContext(context.Background())
	err = root.Execute()
	var outcome *errs.MutationOutcomeError
	if !errors.As(err, &outcome) || outcome.Outcome != "unknown" || output.ExitCodeOf(err) != 22 || len(outcome.Receipts) != 1 {
		t.Fatalf("host received misleading success: %v", err)
	}
	if !strings.Contains(stdout.String(), `"ok":false`) {
		t.Fatalf("misleading stdout: %s", &stdout)
	}
}

func TestReportAgentMutationsWritesStderrDiagnosticAndExitCode(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	var stdout, stderr bytes.Buffer
	f := cmdutil.NewDefault(cmdutil.NewIOStreams(strings.NewReader(""), &stdout, &stderr), cmdutil.InvocationContext{})
	f.Config = func() (*core.CliConfig, error) {
		return &core.CliConfig{AppID: "synthetic-app", Brand: core.BrandFeishu}, nil
	}
	f.ResolvedIdentity = core.AsBot
	f.EnableAgentAttempts = true
	if got := reportAgentMutations(f, 7); got != 7 || stderr.Len() != 0 {
		t.Fatalf("no receipts: exit %d, stderr %q", got, &stderr)
	}
	client, err := f.HttpClient()
	if err != nil {
		t.Fatal(err)
	}
	client.Transport = client.Transport.(transport.RoundTripperDecorator).WithBaseRoundTripper(hostMutationRoundTrip(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"accepted":true}}`))}, nil
	}))
	cmd := &cobra.Command{Use: "mutate"}
	cmdutil.SetRisk(cmd, "write")
	f.CurrentCommand = cmd
	req, err := http.NewRequestWithContext(context.Background(), "POST", "https://fixture.invalid/create", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if got := reportAgentMutations(f, 0); got != output.ExitMutationUnknown {
		t.Fatalf("exit code = %d, want %d", got, output.ExitMutationUnknown)
	}
	ids := f.AgentAttemptSummary().ReceiptIDs()
	if len(ids) != 1 || ids[0] == "" {
		t.Fatalf("receipt ids: %v", ids)
	}
	want := `{"mutation":{"applied":0,"outcome":"unknown","receipts":["` + ids[0] + `"],"unapplied":0,"unknown":1}}`
	if strings.TrimSpace(stderr.String()) != want || stdout.Len() != 0 {
		t.Fatalf("diagnostic:\n got %s\nwant %s\nstdout %q", &stderr, want, &stdout)
	}
}
