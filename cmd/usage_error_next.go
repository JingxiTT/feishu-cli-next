// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/larksuite/cli/errs"
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
