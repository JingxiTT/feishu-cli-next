// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"testing"

	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
)

func TestTaskReplacementRequiresKnownPresentEqualFields(t *testing.T) {
	for _, test := range []struct {
		name             string
		current, desired map[string]interface{}
		want             bool
	}{
		{"equal description clear", map[string]interface{}{"description": ""}, map[string]interface{}{"description": ""}, true},
		{"omitted is not clear", map[string]interface{}{}, map[string]interface{}{"description": ""}, false},
		{"null is not empty", map[string]interface{}{"description": nil}, map[string]interface{}{"description": ""}, false},
		{"assignment is effect", map[string]interface{}{"members": []string{}}, map[string]interface{}{"members": []string{}}, false},
		{"changes one field", map[string]interface{}{"summary": "old", "description": "same"}, map[string]interface{}{"summary": "new", "description": "same"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := taskReplacementUnchanged(test.current, test.desired); got != test.want {
				t.Fatalf("unchanged=%v", got)
			}
		})
	}
}

func TestTaskReplacementSkipsWriteWhenUnchanged(t *testing.T) {
	for _, test := range []struct {
		shortcut common.Shortcut
		args     []string
		current  map[string]interface{}
	}{
		{UpdateTask, []string{"+update", "--task-id", "task-1", "--summary", "same"}, map[string]interface{}{"guid": "task-1", "summary": "same"}},
		{ReopenTask, []string{"+reopen", "--task-id", "task-1"}, map[string]interface{}{"guid": "task-1", "completed_at": "0"}},
		{ReminderTask, []string{"+reminder", "--task-id", "task-1", "--set", "1h"}, map[string]interface{}{"guid": "task-1", "reminders": []map[string]interface{}{{"id": "reminder-1", "relative_fire_minute": 60}}}},
	} {
		t.Run(test.shortcut.Command, func(t *testing.T) {
			f, stdout, _, reg := taskShortcutTestFactory(t)
			warmTenantToken(t, f, reg)
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/task/v2/tasks/task-1", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"task": test.current}}})
			if err := runMountedTaskShortcut(t, test.shortcut, append(test.args, "--as", "bot", "--format", "json"), f, stdout); err != nil {
				t.Fatal(err)
			}
			reg.Verify(t)
			data := decodeTaskOutputData(t, stdout.Bytes())
			if test.shortcut.Command == "+update" {
				data = data["tasks"].([]interface{})[0].(map[string]interface{})
			}
			if data["unchanged"] != true {
				t.Fatalf("output %#v", data)
			}
		})
	}
}
