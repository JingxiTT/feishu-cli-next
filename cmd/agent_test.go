// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
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
