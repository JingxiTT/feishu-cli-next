// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/flagalias"
)

// cobraUsageError types a cobra usage mistake that reached the root dispatcher
// as plain text. A missing required flag lists each flag's type and usage and
// the invocation to retry with, so an agent does not need a --help round trip.
func cobraUsageError(c *cobra.Command, err error) error {
	if c != nil && strings.Contains(err.Error(), "required flag(s) ") {
		if missing := missingRequiredFlags(c); len(missing) > 0 {
			return requiredFlagError(c, err, missing)
		}
	}
	return errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", err.Error())
}

func missingRequiredFlags(c *cobra.Command) []*pflag.Flag {
	var missing []*pflag.Flag
	c.Flags().VisitAll(func(fl *pflag.Flag) {
		req, marked := fl.Annotations[cobra.BashCompOneRequiredFlag]
		if marked && len(req) > 0 && req[0] == "true" && !fl.Changed {
			missing = append(missing, fl)
		}
	})
	return missing
}

func requiredFlagError(c *cobra.Command, err error, missing []*pflag.Flag) *errs.ValidationError {
	params := make([]errs.InvalidParam, 0, len(missing))
	retry := []string{c.CommandPath()}
	for _, fl := range missing {
		reason := "required; " + fl.Value.Type()
		if fl.Usage != "" {
			reason += ": " + fl.Usage
		}
		params = append(params, errs.InvalidParam{Name: "--" + fl.Name, Reason: reason})
		retry = append(retry, fmt.Sprintf("--%s <%s>", fl.Name, fl.Name))
	}
	return errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", err.Error()).
		WithParam("--"+missing[0].Name).
		WithParams(params...).
		WithHint("retry with `%s`", strings.Join(retry, " "))
}

// invalidFlagValueError rewrites a pflag value-parse failure on a non-alias
// flag, so the message names the flag and the expected type instead of the Go
// parser's text. Alias-backed flags keep the caller's spelling via flagDidYouMean.
func invalidFlagValueError(c *cobra.Command, ferr error) (*errs.ValidationError, bool) {
	var invalid *pflag.InvalidValueError
	if !errors.As(ferr, &invalid) {
		return nil, false
	}
	if _, aliased := flagalias.InvalidValueAttributionOf(ferr); aliased {
		return nil, false
	}
	expected, ok := expectedFlagValue(invalid.GetFlag().Value.Type())
	if !ok {
		return nil, false
	}
	return errs.NewValidationError(errs.SubtypeInvalidArgument,
		"invalid value %q for --%s: expected %s", invalid.GetValue(), invalid.GetFlag().Name, expected).
		WithHint("run `%s --help` for valid flags", c.CommandPath()).
		WithCause(ferr), true
}

func expectedFlagValue(valueType string) (string, bool) {
	switch valueType {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "an integer", true
	case "float32", "float64":
		return "a number", true
	case "bool":
		return "true or false", true
	case "duration":
		return "a duration such as 30s", true
	}
	return "", false
}

// rootUnknownCommandError replaces cobra's root-only "unknown command" error,
// raised by Find before flags are handled, with the unknown_subcommand envelope
// a domain group uses. Its text is the only handle cobra leaves on this error.
func rootUnknownCommandError(root *cobra.Command, err error) error {
	const prefix = "unknown command "
	if err == nil || !strings.HasPrefix(err.Error(), prefix) {
		return err
	}
	quoted, qErr := strconv.QuotedPrefix(err.Error()[len(prefix):])
	if qErr != nil {
		return err
	}
	unknown, uErr := strconv.Unquote(quoted)
	if uErr != nil {
		return err
	}
	return unknownSubcommandRunE(root, []string{unknown})
}
