// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
)

func newRequiredFlagPull(t *testing.T) *cobra.Command {
	t.Helper()
	root := &cobra.Command{Use: "feishu-cli-next", SilenceUsage: true, SilenceErrors: true}
	docs := &cobra.Command{Use: "docs"}
	pull := &cobra.Command{Use: "+pull", RunE: func(*cobra.Command, []string) error { return nil }}
	pull.Flags().String("doc", "", "document URL or token")
	if err := pull.MarkFlagRequired("doc"); err != nil {
		t.Fatal(err)
	}
	root.AddCommand(docs)
	docs.AddCommand(pull)
	return pull
}

func TestCobraUsageError_MissingRequiredFlagNamesFlagAndRetry(t *testing.T) {
	pull := newRequiredFlagPull(t)
	cobraErr := pull.ValidateRequiredFlags()
	if cobraErr == nil {
		t.Fatal("ValidateRequiredFlags() succeeded, want missing --doc error")
	}

	err := cobraUsageError(pull, cobraErr)
	var verr *errs.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T, want *errs.ValidationError", err)
	}
	if verr.Message != cobraErr.Error() {
		t.Errorf("message = %q, want cobra's %q", verr.Message, cobraErr.Error())
	}
	if verr.Param != "--doc" {
		t.Errorf("param = %q, want --doc", verr.Param)
	}
	if len(verr.Params) != 1 || verr.Params[0].Name != "--doc" {
		t.Fatalf("params = %+v, want one entry for --doc", verr.Params)
	}
	if reason := verr.Params[0].Reason; !strings.Contains(reason, "string") || !strings.Contains(reason, "document URL or token") {
		t.Errorf("reason = %q, want flag type and usage", reason)
	}
	if want := "retry with `feishu-cli-next docs +pull --doc <doc>`"; verr.Hint != want {
		t.Errorf("hint = %q, want %q", verr.Hint, want)
	}
	if code := output.ExitCodeOf(err); code != output.ExitValidation {
		t.Errorf("exit code = %d, want %d", code, output.ExitValidation)
	}
}

func TestHandleRootError_MissingRequiredFlagEnvelopeCarriesParams(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	pull := newRequiredFlagPull(t)
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	errOut := &bytes.Buffer{}
	f.IOStreams.ErrOut = errOut
	f.CurrentCommand = pull

	exit := handleRootError(f, pull.ValidateRequiredFlags(), nil)
	if exit != output.ExitValidation {
		t.Fatalf("exit = %d, want %d", exit, output.ExitValidation)
	}
	var envelope struct {
		Error struct {
			Type   string `json:"type"`
			Hint   string `json:"hint"`
			Params []struct {
				Name   string `json:"name"`
				Reason string `json:"reason"`
			} `json:"params"`
		} `json:"error"`
	}
	if err := json.Unmarshal(errOut.Bytes(), &envelope); err != nil {
		t.Fatalf("stderr is not JSON: %v\n%s", err, errOut.String())
	}
	if envelope.Error.Type != "validation" {
		t.Errorf("error.type = %q, want validation", envelope.Error.Type)
	}
	if len(envelope.Error.Params) != 1 || envelope.Error.Params[0].Name != "--doc" {
		t.Fatalf("error.params = %+v, want one entry for --doc", envelope.Error.Params)
	}
	if !strings.Contains(envelope.Error.Hint, "--doc <doc>") {
		t.Errorf("error.hint = %q, want a template invocation", envelope.Error.Hint)
	}
}

func TestFlagDidYouMean_InvalidIntValueNamesFlagAndType(t *testing.T) {
	c := &cobra.Command{Use: "demo"}
	c.Flags().Int("limit", 10, "")
	parseErr := c.ParseFlags([]string{"--limit=abc"})
	if parseErr == nil {
		t.Fatal("ParseFlags() succeeded, want invalid integer error")
	}

	err := flagDidYouMean(c, parseErr)
	var verr *errs.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T, want *errs.ValidationError", err)
	}
	want := `invalid value "abc" for --limit: expected an integer`
	if verr.Message != want {
		t.Errorf("message = %q, want %q", verr.Message, want)
	}
	if strings.Contains(verr.Message, "strconv") {
		t.Errorf("message leaks the Go parser text: %q", verr.Message)
	}
	if verr.Subtype != errs.SubtypeInvalidArgument {
		t.Errorf("subtype = %q, want invalid_argument", verr.Subtype)
	}
	if code := output.ExitCodeOf(err); code != output.ExitValidation {
		t.Errorf("exit code = %d, want %d", code, output.ExitValidation)
	}
}

func TestRootUnknownCommand_UsesStructuredSuggestions(t *testing.T) {
	root := &cobra.Command{Use: "feishu-cli-next", SilenceUsage: true, SilenceErrors: true}
	docs := &cobra.Command{Use: "docs", Short: "docs"}
	docs.AddCommand(&cobra.Command{Use: "+pull", RunE: func(*cobra.Command, []string) error { return nil }})
	doctor := &cobra.Command{Use: "doctor", Short: "doctor"}
	root.AddCommand(docs, doctor)
	installUnknownSubcommandGuard(root)

	root.SetArgs([]string{"doc"})
	err := rootUnknownCommandError(root, root.Execute())
	var verr *errs.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("error = %T (%v), want *errs.ValidationError", err, err)
	}
	if want := `unknown subcommand "doc" for "feishu-cli-next"`; verr.Message != want {
		t.Errorf("message = %q, want %q", verr.Message, want)
	}
	if len(verr.Params) != 1 || !containsString(verr.Params[0].Suggestions, "docs") {
		t.Fatalf("params = %+v, want a suggestion for docs", verr.Params)
	}
	if code := output.ExitCodeOf(err); code != output.ExitValidation {
		t.Errorf("exit code = %d, want %d", code, output.ExitValidation)
	}
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
