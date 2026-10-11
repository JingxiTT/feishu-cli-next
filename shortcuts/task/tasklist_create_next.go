// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/errs"
)

// Validate every task before creating its containing tasklist.
func parseTasklistTasks(input string) ([]map[string]interface{}, error) {
	if input == "" {
		return nil, nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "failed to parse --data as JSON array").WithParam("--data").WithCause(err)
	}
	if raw == nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--data must be a JSON array").WithParam("--data")
	}
	tasks := make([]map[string]interface{}, 0, len(raw))
	for i, item := range raw {
		task, err := parseTaskObject(string(item))
		if err != nil {
			return nil, err
		}
		summary, ok := task["summary"].(string)
		if !ok || strings.TrimSpace(summary) == "" {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "task at index %d requires a non-empty summary", i).WithParam("--data")
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
