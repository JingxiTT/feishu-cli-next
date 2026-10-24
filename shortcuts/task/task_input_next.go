// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/larksuite/cli/errs"
)

// Decode arbitrary supported task fields without rounding JSON numbers.
func parseTaskObject(input string) (map[string]interface{}, error) {
	decoder := json.NewDecoder(strings.NewReader(input))
	decoder.UseNumber()
	var object map[string]interface{}
	if err := decoder.Decode(&object); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--data must be a valid JSON object").WithParam("--data").WithCause(err)
	}
	if object == nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--data must be a valid JSON object").WithParam("--data")
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--data must contain one JSON object").WithParam("--data").WithCause(err)
	}
	return object, nil
}
