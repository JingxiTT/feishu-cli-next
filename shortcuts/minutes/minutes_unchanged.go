// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

// Optional equality reads do not expand the scopes required by ordinary writes.
// They are observations, not conditional writes or concurrency protection.
func minutesUnchangedScope(runtime *common.RuntimeContext, scope string) error {
	if !runtime.Bool("skip-unchanged") {
		return nil
	}
	return runtime.EnsureScopes([]string{scope})
}

func minutesReadSnapshot(runtime *common.RuntimeContext, token, suffix string, target any) error {
	data, err := runtime.CallAPITyped(http.MethodGet, fmt.Sprintf("/open-apis/minutes/v1/minutes/%s%s", validate.EncodePathSegment(token), suffix), nil, nil)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(data)
	if err == nil {
		err = json.Unmarshal(encoded, target)
	}
	if err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot compare current minute content").WithCause(err)
	}
	return nil
}

func minutesTitleUnchanged(runtime *common.RuntimeContext, token, title string) (bool, error) {
	var current struct {
		Minute *struct {
			Title *string `json:"title"`
		} `json:"minute"`
	}
	if err := minutesReadSnapshot(runtime, token, "", &current); err != nil {
		return false, err
	}
	if current.Minute == nil || current.Minute.Title == nil {
		return false, minutesMissingComparison()
	}
	return *current.Minute.Title == title, nil
}

func minutesSummaryUnchanged(runtime *common.RuntimeContext, token, summary string) (bool, error) {
	var current struct {
		Summary *string `json:"summary"`
	}
	if err := minutesReadSnapshot(runtime, token, "/artifacts", &current); err != nil {
		return false, err
	}
	if current.Summary == nil {
		return false, minutesMissingComparison()
	}
	return *current.Summary == summary, nil
}

func minutesTodosUnchanged(runtime *common.RuntimeContext, token string, ops []minuteTodoOp) (bool, error) {
	var current struct {
		Todos *[]struct {
			ID      string  `json:"todo_id"`
			Content *string `json:"content"`
			Done    *bool   `json:"is_done"`
		} `json:"minute_todos"`
	}
	if err := minutesReadSnapshot(runtime, token, "/artifacts", &current); err != nil {
		return false, err
	}
	if current.Todos == nil {
		return false, minutesMissingComparison()
	}
	for _, op := range ops {
		matches := 0
		equal := false
		for _, item := range *current.Todos {
			if item.ID == op.item["todo_id"] {
				matches++
				if item.Content == nil || item.Done == nil {
					return false, minutesMissingComparison()
				}
				equal = item.Content != nil && item.Done != nil && *item.Content == op.item["content"] && *item.Done == op.item["is_done"]
			}
		}
		if matches > 1 {
			return false, minutesMissingComparison()
		}
		if matches != 1 || !equal {
			return false, nil
		}
	}
	return true, nil
}

func minutesMissingComparison() error {
	return errs.NewInternalError(errs.SubtypeInvalidResponse, "current minute response lacks unambiguous content for --skip-unchanged")
}
