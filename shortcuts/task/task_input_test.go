// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/spf13/cobra"
)

func TestTaskObjectRejectsNonObjectsAndPreservesNumbers(t *testing.T) {
	for _, input := range []string{"null", "[]", "1", "{} {}", "{", "true"} {
		_, err := parseTaskObject(input)
		var validation *errs.ValidationError
		if !errors.As(err, &validation) || validation.Param != "--data" {
			t.Fatalf("input %s: %v", input, err)
		}
	}
	object, err := parseTaskObject(`{"extra_number":9007199254740993,"description":"","due":null}`)
	if err != nil || object["extra_number"] != json.Number("9007199254740993") || object["description"] != "" || object["due"] != nil {
		t.Fatalf("object %#v: %v", object, err)
	}
}

func TestTaskUpdateEmptyDescriptionIsExplicitClear(t *testing.T) {
	cmd := &cobra.Command{}
	for _, name := range []string{"data", "summary", "description", "due"} {
		cmd.Flags().String(name, "", "")
	}
	if err := cmd.Flags().Set("description", ""); err != nil {
		t.Fatal(err)
	}
	body, err := buildTaskUpdateBody(&common.RuntimeContext{Cmd: cmd})
	if err != nil {
		t.Fatal(err)
	}
	object := body["task"].(map[string]interface{})
	if len(object) != 1 || object["description"] != "" {
		t.Fatalf("clear: %#v", body)
	}
}

func TestTaskCreateExplicitFlagsOverrideDataEvenWhenEmpty(t *testing.T) {
	cmd := &cobra.Command{}
	for _, name := range []string{"data", "summary", "description", "due", "assignee", "follower", "tasklist-id", "idempotency-key"} {
		cmd.Flags().String(name, "", "")
	}
	if err := cmd.Flags().Set("data", `{"summary":"valid","description":"original"}`); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("description", ""); err != nil {
		t.Fatal(err)
	}
	body, err := buildTaskCreateBody(&common.RuntimeContext{Cmd: cmd})
	if err != nil || body["description"] != "" {
		t.Fatalf("body %#v: %v", body, err)
	}
	if err := cmd.Flags().Set("summary", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := buildTaskCreateBody(&common.RuntimeContext{Cmd: cmd}); err == nil {
		t.Fatal("explicit empty summary silently used data summary")
	}
}

func TestTasklistRejectsEveryInvalidTaskBeforeWrites(t *testing.T) {
	for _, input := range []string{"null", "[null]", `[{}, {"summary":"ok"}]`, `[{"summary":"ok"},{"summary":" "}]`} {
		if _, err := parseTasklistTasks(input); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
	tasks, err := parseTasklistTasks(`[{"summary":"你好","extra_number":9007199254740993}]`)
	if err != nil || tasks[0]["extra_number"] != json.Number("9007199254740993") {
		t.Fatalf("tasks %#v: %v", tasks, err)
	}
}
