// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/larksuite/cli/errs"
)

// parseDocsInputObject preserves decimal numbers in extensible rich-content
// sidecars and search filters. A trailing JSON value is never part of the input.
func parseDocsInputObject(raw, param string) (map[string]interface{}, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value map[string]interface{}
	if err := decoder.Decode(&value); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "%s must be a JSON object", param).WithParam(param).WithCause(err)
	}
	if value == nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "%s must be a JSON object, got null", param).WithParam(param)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "%s must contain exactly one JSON object", param).WithParam(param).WithCause(err)
	}
	return value, nil
}
