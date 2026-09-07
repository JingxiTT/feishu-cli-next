// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"bytes"
	"encoding/json"
)

type taskReplacementFields struct {
	Summary        json.RawMessage `json:"summary"`
	Description    json.RawMessage `json:"description"`
	Due            json.RawMessage `json:"due"`
	Start          json.RawMessage `json:"start"`
	Extra          json.RawMessage `json:"extra"`
	RepeatRule     json.RawMessage `json:"repeat_rule"`
	CompletedAt    json.RawMessage `json:"completed_at"`
	Mode           json.RawMessage `json:"mode"`
	IsMilestone    json.RawMessage `json:"is_milestone"`
	CustomComplete json.RawMessage `json:"custom_complete"`
}

// Only fields with replacement semantics and lossless API representations are
// eligible. Members, custom fields, attachments and workflow effects are not
// inferred from a projected task response. An omitted field is never a clear.
func taskReplacementUnchanged(current, requested map[string]interface{}) bool {
	if len(requested) == 0 {
		return false
	}
	raw, err := json.Marshal(current)
	if err != nil {
		return false
	}
	var observed taskReplacementFields
	if json.Unmarshal(raw, &observed) != nil {
		return false
	}
	for field, desired := range requested {
		var actual json.RawMessage
		switch field {
		case "summary":
			actual = observed.Summary
		case "description":
			actual = observed.Description
		case "due":
			actual = observed.Due
		case "start":
			actual = observed.Start
		case "extra":
			actual = observed.Extra
		case "repeat_rule":
			actual = observed.RepeatRule
		case "completed_at":
			actual = observed.CompletedAt
		case "mode":
			actual = observed.Mode
		case "is_milestone":
			actual = observed.IsMilestone
		case "custom_complete":
			actual = observed.CustomComplete
		default:
			return false
		}
		want, err := json.Marshal(desired)
		if err != nil || !bytes.Equal(want, actual) {
			return false
		}
	}
	return true
}
