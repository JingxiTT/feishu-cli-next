// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/errs"
)

func validateSQLStatements(statements []map[string]interface{}) error {
	var decoded []struct {
		Kind string  `json:"sql_type"`
		Data *string `json:"data"`
	}
	encoded, err := json.Marshal(statements)
	if err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode SQL statement results").WithCause(err)
	}
	if err = json.Unmarshal(encoded, &decoded); err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot decode SQL statement results").WithCause(err)
	}
	for _, statement := range decoded {
		if statement.Kind == "" || statement.Kind == "RAW" {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "SQL response has an invalid statement result")
		}
		if statement.Kind != "SELECT" {
			continue
		}
		if statement.Data == nil {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "SQL SELECT response is missing row data")
		}
		data := *statement.Data
		if data == "" || strings.TrimSpace(data) == "null" {
			continue
		}
		var rows []map[string]json.RawMessage
		if err := json.Unmarshal([]byte(data), &rows); err != nil {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "SQL SELECT response has invalid row data").WithCause(err)
		}
	}
	return nil
}
