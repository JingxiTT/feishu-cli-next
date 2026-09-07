// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package guide exposes the single feishu-cli-next entry skill.
package guide

import (
	"io/fs"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/nextskill"
	"github.com/larksuite/cli/internal/output"
	"github.com/spf13/cobra"
)

// NewCmd accepts a filesystem rooted at agent-skills, independent of the
// inherited upstream skills available through the compatibility skills command.
func NewCmd(f *cmdutil.Factory, content fs.FS) *cobra.Command {
	show := func(cmd *cobra.Command, args []string) error {
		data, err := nextskill.Read(content)
		if err != nil {
			return err
		}
		if _, err := f.IOStreams.Out.Write(data); err != nil {
			return errs.NewInternalError(errs.SubtypeFileIO, "write entry skill failed").WithCause(err)
		}
		return nil
	}
	cmd := &cobra.Command{Use: "guide", Short: "Read or install the single project agent skill", Args: cobra.NoArgs, RunE: show}
	cmdutil.DisableAuthCheck(cmd)
	cmdutil.SetRisk(cmd, "read")
	showCmd := &cobra.Command{Use: "show", Short: "Print the embedded entry skill as Markdown", Args: cobra.NoArgs, RunE: show}
	cmdutil.DisableAuthCheck(showCmd)
	cmdutil.SetRisk(showCmd, "read")
	cmd.AddCommand(showCmd)
	for _, action := range []string{"install", "update", "uninstall"} {
		var project string
		child := &cobra.Command{
			Use:   action,
			Short: action + " the project-local agent skill, preserving local edits",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				result, err := nextskill.Apply(content, action, project)
				if err != nil {
					return err
				}
				output.PrintJson(f.IOStreams.Out, result)
				return nil
			},
		}
		child.Flags().StringVar(&project, "project", ".", "Existing local project directory (installs under .agents/skills)")
		cmdutil.DisableAuthCheck(child)
		cmdutil.SetRisk(child, "write")
		cmd.AddCommand(child)
	}
	return cmd
}
