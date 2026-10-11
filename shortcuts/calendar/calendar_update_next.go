// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar

import (
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

// calendarUpdateUnchanged compares only exact replacements against the already
// fetched occurrence. Explicit clears, attendee effects and explicit notification
// requests retain their own semantics. A master fallback is not occurrence proof.
// This read does not provide a conditional-write or concurrency guarantee.
func calendarUpdateUnchanged(runtime *common.RuntimeContext, current *calendarEvent, eventID string) bool {
	if current == nil || current.inferred || current.EventID != eventID ||
		strings.TrimSpace(runtime.Str("add-attendee-ids")) != "" ||
		strings.TrimSpace(runtime.Str("remove-attendee-ids")) != "" ||
		(runtime.Cmd.Flags().Changed("notify") && runtime.Bool("notify")) {
		return false
	}
	body, fields, err := buildCalendarUpdateEventData(runtime)
	if err != nil || !fields {
		return false
	}
	for key, value := range body {
		switch key {
		case "need_notification":
		case "summary":
			if value == "" || value != current.Summary {
				return false
			}
		case "description_rich":
			if value == "" || value != current.DescriptionRich {
				return false
			}
		case "recurrence":
			if value == "" || value != current.Recurrence {
				return false
			}
		case "start_time":
			if current.StartTime == nil || current.StartTime.Date != "" || value.(map[string]string)["timestamp"] != current.StartTime.Timestamp {
				return false
			}
		case "end_time":
			if current.EndTime == nil || current.EndTime.Date != "" || value.(map[string]string)["timestamp"] != current.EndTime.Timestamp {
				return false
			}
		default:
			return false
		}
	}
	return true
}
