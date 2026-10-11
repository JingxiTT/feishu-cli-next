// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/larksuite/cli/shortcuts/im"
	"github.com/spf13/cobra"
)

func TestAgentModePreservesLegacyAndDomainFlags(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	opts := []BuildOption{WithIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}), WithoutPlugins(), WithoutStrictMode()}
	legacy := Build(context.Background(), cmdutil.InvocationContext{}, opts...)
	next := Build(context.Background(), cmdutil.InvocationContext{}, append(opts, WithAgentMode())...)
	if legacy.Name() != "lark-cli" || legacy.PersistentFlags().Lookup("result-fields") != nil {
		t.Fatal("agent options leaked into legacy build")
	}
	if next.Name() != "feishu-cli-next" || next.PersistentFlags().Lookup("result-fields") == nil {
		t.Fatal("agent mode not registered")
	}
	for _, path := range [][]string{{"base", "+record-list"}, {"apps", "+automation-create"}, {"docs", "+fetch"}} {
		before, _, err := legacy.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		after, _, err := next.Find(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"fields", "detail"} {
			original := before.LocalNonPersistentFlags().Lookup(name)
			if original == nil {
				continue
			}
			current := after.LocalNonPersistentFlags().Lookup(name)
			if current == nil || current.Value.Type() != original.Value.Type() || current.DefValue != original.DefValue {
				t.Fatalf("changed %v --%s", path, name)
			}
		}
	}
}

func TestAgentDiscoveryCompactAndFullDetail(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for _, detail := range []bool{false, true} {
		var out, errOut bytes.Buffer
		root := Build(context.Background(), cmdutil.InvocationContext{}, WithIO(strings.NewReader(""), &out, &errOut), WithoutPlugins(), WithoutStrictMode(), WithAgentMode())
		args := []string{"discover", "docs", "--limit", "1"}
		if detail {
			args = append(args, "--result-detail")
		}
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatalf("%v: %s", err, &out)
		}
		if !detail && strings.Count(out.String(), "\n") != 1 {
			t.Fatalf("not compact: %s", &out)
		}
		if result["meta"].(map[string]any)["query"] != "docs" || len(result["data"].([]any)) != 1 {
			t.Fatalf("missing discovery context: %s", &out)
		}
	}
}

func TestAgentRejectsUnsupportedResultModesBeforeExecution(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	for _, test := range []struct {
		name     string
		opts     output.AgentOptions
		business bool
		raw      bool
		risk     string
	}{
		{"invalid field", output.AgentOptions{Fields: "a..b"}, true, false, "read"},
		{"raw projection", output.AgentOptions{Fields: "id"}, true, true, "read"},
		{"auth artifact", output.AgentOptions{SaveResult: "secret.json"}, false, false, "read"},
		{"mutation artifact", output.AgentOptions{SaveResult: "result.json"}, true, false, "write"},
		{"detail projection", output.AgentOptions{Fields: "id", Detail: true}, true, false, "read"},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := &cobra.Command{Use: "operation"}
			if test.business {
				cmdmeta.SetSource(c, cmdmeta.SourceShortcut, false)
			}
			cmdutil.SetRisk(c, test.risk)
			c.Flags().Bool("raw", false, "")
			if test.raw {
				_ = c.Flags().Set("raw", "true")
			}
			if err := validateAgentOutput(f, c, &test.opts); err == nil {
				t.Fatal("invalid result mode accepted")
			}
		})
	}
}

func TestAgentOutputModesStopBeforeCommandExecution(t *testing.T) {
	for _, test := range []struct {
		name   string
		mode   string
		value  string
		opts   output.AgentOptions
		reject bool
	}{
		{"concise projection", "concise", "true", output.AgentOptions{Fields: "id"}, true},
		{"concise artifact", "concise", "true", output.AgentOptions{SaveResult: "result.json"}, true},
		{"concise disabled", "concise", "false", output.AgentOptions{Fields: "id"}, false},
		{"concise alone", "concise", "true", output.AgentOptions{}, false},
		{"raw projection", "raw", "true", output.AgentOptions{Fields: "id"}, true},
		{"raw disabled", "raw", "false", output.AgentOptions{Fields: "id"}, false},
		{"jq projection", "jq", ".data", output.AgentOptions{Fields: "id"}, true},
		{"stream projection", "stream", "true", output.AgentOptions{Fields: "id"}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			f, _, _, _ := cmdutil.TestFactory(t, nil)
			f.IOStreams.Out = output.WithAgentOptions(f.IOStreams.Out, &test.opts)
			executions := 0
			c := &cobra.Command{Use: "operation", RunE: func(c *cobra.Command, args []string) error {
				executions++
				return output.WriteJSON(f.IOStreams.Out, map[string]any{"id": "fixture"})
			}}
			cmdmeta.SetSource(c, cmdmeta.SourceShortcut, false)
			cmdutil.SetRisk(c, "read")
			if test.mode == "jq" {
				c.Flags().String(test.mode, "", "")
			} else {
				c.Flags().Bool(test.mode, false, "")
			}
			root := &cobra.Command{Use: "next", SilenceErrors: true, SilenceUsage: true}
			root.AddCommand(c)
			finalizeAgentFoundation(f, root, &test.opts)
			root.SetArgs([]string{"operation", "--" + test.mode + "=" + test.value})
			err := root.Execute()
			if !test.reject {
				if err != nil || executions != 1 {
					t.Fatalf("valid mode did not execute: executions=%d, err=%v", executions, err)
				}
				return
			}
			var validation *errs.ValidationError
			if !errors.As(err, &validation) || validation.Subtype != errs.SubtypeInvalidArgument || validation.Param != "--"+test.mode {
				t.Fatalf("expected typed mode conflict, got %#v", err)
			}
			if executions != 0 {
				t.Fatalf("invalid output mode executed command %d times", executions)
			}
		})
	}
}

func TestAgentConciseDryRunKeepsStructuredProjection(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for _, test := range []struct {
		shortcut common.Shortcut
		args     []string
	}{
		{im.ImChatMessageList, []string{"--chat-id", "oc_fixture"}},
		{im.ImThreadsMessagesList, []string{"--thread", "omt_fixture"}},
	} {
		t.Run(test.shortcut.Command, func(t *testing.T) {
			f, stdout, _, _ := cmdutil.TestFactory(t, &core.CliConfig{})
			root := &cobra.Command{Use: "next", SilenceErrors: true, SilenceUsage: true}
			opts := configureAgentFoundation(f, root)
			test.shortcut.Mount(root, f)
			finalizeAgentFoundation(f, root, opts)
			args := append([]string{test.shortcut.Command}, test.args...)
			root.SetArgs(append(args, "--dry-run", "--concise", "--no-reactions", "--result-fields", "method,url"))
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			var result map[string]any
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatalf("dry-run did not emit JSON: %v: %s", err, stdout)
			}
			if result["ok"] != true || result["dry_run"] != true {
				t.Fatalf("dry-run envelope lost metadata: %s", stdout)
			}
		})
	}
}

func TestAgentUpdateDoesNotUseUpstreamChannel(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for _, check := range []bool{false, true} {
		var out, errOut bytes.Buffer
		root := Build(context.Background(), cmdutil.InvocationContext{}, WithIO(strings.NewReader(""), &out, &errOut), WithoutPlugins(), WithoutStrictMode(), WithAgentMode())
		args := []string{"update", "--json"}
		if check {
			args = append(args, "--check")
		}
		root.SetArgs(args)
		err := root.Execute()
		if !check && err == nil {
			t.Fatal("unconfigured distribution update accepted")
		}
		if check && (err != nil || !strings.Contains(out.String(), `"update_channel":"unconfigured"`)) {
			t.Fatalf("unexpected channel status: %v %s", err, &out)
		}
	}
}

func TestAgentRawReadCannotPersistCredentialArtifacts(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	for _, path := range []string{"/open-apis/apps/v1/apps/fixture/oapi_apikeys/fixture", "/open-apis/apps/v1/apps/fixture/git_info"} {
		c := &cobra.Command{Use: "api"}
		cmdutil.SetRisk(c, "read")
		if err := c.ParseFlags([]string{"GET", path}); err != nil {
			t.Fatal(err)
		}
		opts := output.AgentOptions{SaveResult: "credentials.json"}
		err := validateAgentOutput(f, c, &opts)
		var validation *errs.ValidationError
		if !errors.As(err, &validation) || validation.Subtype != errs.SubtypeInvalidArgument || validation.Param != "--save-result" || !strings.Contains(validation.Message, "non-raw read operation") {
			t.Fatalf("raw credential artifact guard for %s: %#v", path, err)
		}
	}
}

func TestAgentDiscoveryWithInvocationArgsKeepsDomainCommands(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	var out, errOut bytes.Buffer
	root := Build(context.Background(), cmdutil.InvocationContext{},
		WithIO(strings.NewReader(""), &out, &errOut), WithoutPlugins(), WithoutStrictMode(),
		WithAgentMode(), WithInvocationArgs([]string{"discover", "docs +pull", "--limit", "1"}))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var result struct {
		Data []struct {
			Command string `json:"command"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != 1 || result.Data[0].Command != "docs +pull" {
		t.Fatalf("target-aware discovery lost domain commands: %s", &out)
	}
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("target-aware build lost compact output: %s", &out)
	}
}
