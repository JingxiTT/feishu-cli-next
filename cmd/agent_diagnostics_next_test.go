// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TestAgentDeprecatedFlagDiagnosticsUseStderr(t *testing.T) {
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	var stdout, stderr bytes.Buffer
	f.IOStreams.Out, f.IOStreams.ErrOut = &stdout, &stderr
	root := &cobra.Command{Use: "fixture", SilenceUsage: true, SilenceErrors: true}
	failure := errs.NewValidationError(errs.SubtypeInvalidArgument, "conflicting flags").WithParam("--old")
	child := &cobra.Command{Use: "operation", RunE: func(*cobra.Command, []string) error { return failure }}
	child.Flags().String("old", "", "legacy input")
	if err := child.Flags().MarkDeprecated("old", "use --new"); err != nil {
		t.Fatal(err)
	}
	root.AddCommand(child)
	opts := configureAgentFoundation(f, root)
	root.SetOut(f.IOStreams.Out)
	root.SetErr(f.IOStreams.ErrOut)
	finalizeAgentFoundation(f, root, opts)
	root.SetArgs([]string{"operation", "--old", "value"})
	if err := root.Execute(); !errors.Is(err, failure) {
		t.Fatalf("typed command error lost: %v", err)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "Flag --old has been deprecated") {
		t.Fatalf("diagnostic placement: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}
