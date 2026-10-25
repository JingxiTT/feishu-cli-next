// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/flagalias"
)

// InvalidFlagValueError types a pflag value-parse failure on an integer, number,
// bool, or duration flag. It names the flag the caller typed and the value it
// expects, so the Go parser's text never reaches the envelope. ok is false for
// any other error, which the caller keeps on its generic path.
func InvalidFlagValueError(c *cobra.Command, ferr error) (*errs.ValidationError, bool) {
	var invalid *pflag.InvalidValueError
	if !errors.As(ferr, &invalid) {
		return nil, false
	}
	expected, ok := expectedFlagValue(invalid.GetFlag().Value.Type())
	if !ok {
		return nil, false
	}
	name := invalid.GetFlag().Name
	attribution, aliased := flagalias.InvalidValueAttributionOf(ferr)
	if aliased {
		name = attribution.Source
	}
	verr := errs.NewValidationError(errs.SubtypeInvalidArgument,
		"invalid value %q for --%s: expected %s", invalid.GetValue(), name, expected).
		WithCause(ferr)
	if !aliased {
		return verr.WithHint("run `%s --help` for valid flags", c.CommandPath()), true
	}
	verr.WithParam("--" + name)
	if attribution.Source != attribution.Canonical {
		return verr.WithHint("--%s maps to canonical flag --%s; run `%s --help` for valid values",
			attribution.Source, attribution.Canonical, c.CommandPath()), true
	}
	return verr.WithHint("run `%s --help` for valid flags", c.CommandPath()), true
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
