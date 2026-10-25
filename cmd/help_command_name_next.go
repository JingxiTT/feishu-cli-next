// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"io"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/internal/recovery"
)

// installCommandNameHelp renders help pages with this build's executable name.
// Cobra writes help and usage through one writer per command, so the default
// help is buffered and rewritten once at the root, which every command inherits.
func installCommandNameHelp(root *cobra.Command, name string) {
	if name == "" || name == "lark-cli" {
		return
	}
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		out := cmd.OutOrStdout()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		defaultHelp(cmd, args)
		cmd.SetOut(out)
		_, _ = io.WriteString(out, recovery.RewriteCommandName(buf.String(), name))
	})
}
