// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandNameHelpRendersBuildCommandName(t *testing.T) {
	root := &cobra.Command{Use: "lark-cli"}
	child := &cobra.Command{
		Use:  "apps",
		Long: "Domain guide: lark-cli skills read lark-apps\nUse github.com/larksuite/cli and ~/.lark-cli",
		Run:  func(*cobra.Command, []string) {},
	}
	root.AddCommand(child)
	installCommandNameHelp(root, "feishu-cli-next")

	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"apps", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Domain guide: feishu-cli-next skills read lark-apps") {
		t.Fatalf("help did not render the build command name:\n%s", got)
	}
	if !strings.Contains(got, "github.com/larksuite/cli and ~/.lark-cli") {
		t.Fatalf("help rewrote a non-command token:\n%s", got)
	}
}

func TestCommandNameHelpIsIdentityForUpstreamName(t *testing.T) {
	root := &cobra.Command{Use: "lark-cli"}
	root.AddCommand(&cobra.Command{Use: "apps", Long: "run lark-cli skills read"})
	installCommandNameHelp(root, "lark-cli")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"apps", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "run lark-cli skills read") {
		t.Fatalf("upstream name changed:\n%s", out.String())
	}
}
