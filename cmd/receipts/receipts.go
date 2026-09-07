// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package receipts

import (
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
	"github.com/spf13/cobra"
)

func NewCmdReceipts(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{Use: "receipts", Short: "Inspect local mutation evidence; never replay requests"}
	cmdutil.DisableAuthCheck(cmd)
	list := &cobra.Command{Use: "list", Short: "List retained mutation receipts", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		receipts, err := f.AgentReceiptStore().List()
		if err != nil {
			return err
		}
		output.PrintJson(f.IOStreams.Out, output.Envelope{OK: true, Data: receipts})
		return nil
	}}
	inspect := &cobra.Command{Use: "inspect <id>", Short: "Read one receipt and its recorded account binding", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		receipt, err := f.AgentReceiptStore().Inspect(args[0])
		if err != nil {
			return err
		}
		output.PrintJson(f.IOStreams.Out, output.Envelope{OK: true, Data: receipt})
		return nil
	}}
	cleanup := &cobra.Command{Use: "cleanup <id>", Short: "Delete one local receipt (does not undo a remote mutation)", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := f.AgentReceiptStore().Cleanup(args[0]); err != nil {
			return err
		}
		output.PrintJson(f.IOStreams.Out, output.Envelope{OK: true, Data: map[string]string{"removed": args[0]}})
		return nil
	}}
	cmdutil.SetRisk(cmd, "read")
	cmdutil.SetRisk(list, "read")
	cmdutil.SetRisk(inspect, "read")
	cmdutil.SetRisk(cleanup, "write")
	cmd.AddCommand(list, inspect, cleanup)
	return cmd
}
