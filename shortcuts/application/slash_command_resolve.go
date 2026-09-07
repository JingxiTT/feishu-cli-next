// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

// commandNotFoundError reports a resolution miss against the live list as an
// API-category not-found error (the name is a valid argument shape; the
// resource simply does not exist server-side - this is not a validation
// failure of caller input).
func commandNotFoundError(name string) error {
	return errs.NewAPIError(errs.SubtypeNotFound,
		"slash command %q not found in the current bound app", name).
		WithHint("run `lark-cli application +slash-command-list` to see registered commands")
}

// resolveCommandID resolves a command name to its command_id via the live
// list endpoint (in-memory only; never touches local files). Requires the
// read scope on the current identity.
func resolveCommandID(runtime *common.RuntimeContext, name string) (string, error) {
	item, err := resolveCommand(runtime, name)
	if err != nil {
		return "", err
	}
	state, err := decodeCommandState(item)
	return state.ID, err
}
