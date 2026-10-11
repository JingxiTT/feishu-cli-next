// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task_test

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

// Cases are reviewed synthetic payloads for every frozen generated task method.
// They exercise all top-level body fields and query options through real processes.
//
//go:embed catalog_cases.json
var catalogCases []byte

func TestFrozenCatalogRequests(t *testing.T) {
	var cases []struct {
		Confirm      bool
		Name         string
		Args         []string
		Method, Path string
		Query        map[string]string
		Body         interface{}
	}
	if err := json.Unmarshal(catalogCases, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 34 {
		t.Fatalf("catalog coverage changed: %d", len(cases))
	}
	frozenFile, err := os.Open("../../../docs/reference/v1.0.97/registrations.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer frozenFile.Close()
	frozenReader, err := gzip.NewReader(frozenFile)
	if err != nil {
		t.Fatal(err)
	}
	defer frozenReader.Close()
	var frozen struct {
		Feishu struct {
			Commands map[string]struct {
				Runnable bool
				Source   string
			}
		}
	}
	if err := json.NewDecoder(frozenReader).Decode(&frozen); err != nil {
		t.Fatal(err)
	}
	covered := make(map[string]bool)
	for _, tc := range cases {
		command := "task " + strings.ReplaceAll(tc.Name, "/", " ")
		if covered[command] || !frozen.Feishu.Commands[command].Runnable || frozen.Feishu.Commands[command].Source != "service" {
			t.Fatalf("unexpected or duplicate catalog case %s", command)
		}
		covered[command] = true
	}
	for command, metadata := range frozen.Feishu.Commands {
		if strings.HasPrefix(command, "task ") && len(strings.Fields(command)) == 3 && metadata.Runnable && metadata.Source == "service" && !covered[command] {
			t.Errorf("uncovered frozen command %s", command)
		}
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.Method || r.URL.Path != tc.Path {
					t.Errorf("request %s %s; want %s %s", r.Method, r.URL.Path, tc.Method, tc.Path)
				}
				for key, value := range tc.Query {
					if r.URL.Query().Get(key) != value {
						t.Errorf("query %s=%q want %q", key, r.URL.Query().Get(key), value)
					}
				}
				if tc.Body != nil {
					var body interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if !reflect.DeepEqual(body, tc.Body) {
						t.Errorf("body %#v; want %#v", body, tc.Body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":{"guid":"synthetic-result","items":[],"has_more":false,"preserved":"你好"}}`)
			}))
			identity := "user"
			if strings.HasPrefix(tc.Name, "agent") {
				identity = "bot"
			}
			args := append(append([]string{}, tc.Args...), "--as", identity)
			if tc.Body != nil {
				body, err := json.Marshal(tc.Body)
				if err != nil {
					t.Fatal(err)
				}
				args = append(args, "--data", string(body))
			}
			if tc.Confirm {
				args = append(args, "--yes")
			}
			dry := f.Run(append(args, "--dry-run")...)
			if dry.ExitCode != 0 || calls.Load() != 0 {
				t.Fatalf("dry-run %+v calls=%d", dry, calls.Load())
			}
			result := f.Run(args...)
			if result.ExitCode != 0 || calls.Load() != 1 || !strings.Contains(result.Stdout, "你好") {
				t.Fatalf("run %+v calls=%d", result, calls.Load())
			}
		})
	}
}

func requireSuccess(t *testing.T, result nextfixture.Result) {
	t.Helper()
	if result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
		t.Fatalf("result %+v", result)
	}
}

func TestReplacementLifecycleAndClear(t *testing.T) {
	current := map[string]interface{}{"guid": "task-fixture", "summary": "最初", "description": "preserve", "completed_at": "0"}
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/open-apis/task/v2/tasks/task-fixture" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Method == "PATCH" {
			writes.Add(1)
			var body struct {
				Task         map[string]interface{}
				UpdateFields []string `json:"update_fields"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			for _, field := range body.UpdateFields {
				current[field] = body.Task[field]
			}
		} else if r.Method != "GET" {
			t.Errorf("method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"task": current}})
	}))
	run := func(args ...string) nextfixture.Result { return f.Run(append([]string{"task"}, args...)...) }
	projected := run("+update", "--task-id", "task-fixture", "--summary", "最初", "--result-fields", "summary")
	requireSuccess(t, projected)
	if gjson.Get(projected.Stdout, "data.tasks.0.guid").String() != "task-fixture" || !gjson.Get(projected.Stdout, "data.tasks.0.unchanged").Bool() || gjson.Get(projected.Stdout, "data.tasks.0.confirmed.summary").String() != "最初" {
		t.Fatalf("projection lost identity or observed outcome: %+v", projected)
	}
	if writes.Load() != 0 {
		t.Fatal("unchanged summary wrote")
	}
	requireSuccess(t, run("+update", "--task-id", "task-fixture", "--description="))
	if writes.Load() != 1 || current["description"] != "" || current["summary"] != "最初" {
		t.Fatalf("clear %#v, writes %d", current, writes.Load())
	}
	requireSuccess(t, run("+update", "--task-id", "task-fixture", "--data", `{"description":""}`))
	if writes.Load() != 1 {
		t.Fatal("unchanged clear wrote")
	}
	requireSuccess(t, run("+complete", "--task-id", "task-fixture"))
	requireSuccess(t, run("+complete", "--task-id", "task-fixture"))
	if writes.Load() != 2 {
		t.Fatal("complete repeated a write")
	}
	dry := run("+reopen", "--task-id", "task-fixture", "--dry-run")
	if dry.ExitCode != 0 || writes.Load() != 2 {
		t.Fatalf("reopen dry-run %+v", dry)
	}
	requireSuccess(t, run("+reopen", "--task-id", "task-fixture"))
	requireSuccess(t, run("+reopen", "--task-id", "task-fixture"))
	if writes.Load() != 3 {
		t.Fatal("reopen repeated a write")
	}
}

func TestTaskReadArtifactRetainsFullTypes(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/open-apis/task/v2/tasks/task-fixture" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"task":{"guid":"task-fixture","summary":"你好","custom_fields":[{"type":"number","number_value":"9007199254740993.01"},{"type":"text","text_value":""}],"due":null}}}`)
	}))
	path := filepath.Join(f.WorkDir, "task-result.json")
	result := f.Run("task", "tasks", "get", "--task-guid", "task-fixture", "--as", "user", "--save-result", path, "--result-fields", "task.summary")
	requireSuccess(t, result)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if gjson.GetBytes(raw, "schema").String() != "feishu-cli-next/result/v1" || gjson.GetBytes(raw, "result.data.task.custom_fields.0.number_value").String() != "9007199254740993.01" || gjson.GetBytes(raw, "result.data.task.due").Raw != "null" || gjson.GetBytes(raw, "binding.user_id").String() == "" {
		t.Fatalf("artifact %s", raw)
	}
}

func TestInvalidTasksNeverDispatch(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{
		{"task", "+create", "--summary", "x", "--data", "null"},
		{"task", "+update", "--task-id", "a,b", "--data", "null", "--summary", "x"},
		{"task", "+tasklist-create", "--name", "private fixture", "--data", `[{"summary":"valid"},null]`},
		{"task", "+reminder", "--task-id", "a", "--set", "-1h"},
	} {
		for _, dry := range []bool{false, true} {
			invocation := append([]string{}, args...)
			if dry {
				invocation = append(invocation, "--dry-run")
			}
			result := f.Run(invocation...)
			if result.ExitCode != 2 || gjson.Get(result.Stderr, "error.type").String() != "validation" {
				t.Fatalf("invalid %+v", result)
			}
		}
	}
}

func TestTaskSummaryInputProvenance(t *testing.T) {
	for _, shortcut := range []string{"+create", "+update"} {
		for _, test := range []struct {
			name, data, summary, param string
			explicitSummary            bool
		}{
			{name: "null data summary", data: `{"summary":null}`, param: "--data"},
			{name: "empty data summary", data: `{"summary":""}`, param: "--data"},
			{name: "numeric data summary", data: `{"summary":3}`, param: "--data"},
			{name: "empty override", data: `{"summary":"valid"}`, explicitSummary: true, param: "--summary"},
			{name: "invalid data and empty override", data: `{"summary":null}`, explicitSummary: true, param: "--summary"},
			{name: "valid override of null", data: `{"summary":null}`, explicitSummary: true, summary: "override"},
			{name: "valid override of empty", data: `{"summary":""}`, explicitSummary: true, summary: "override"},
		} {
			t.Run(shortcut+"/"+test.name, func(t *testing.T) {
				var writes atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if test.param != "" {
						t.Errorf("invalid summary dispatched %s %s", r.Method, r.URL.Path)
					}
					summary := "old"
					if r.Method != http.MethodGet {
						writes.Add(1)
						body, err := io.ReadAll(r.Body)
						if err != nil {
							t.Error(err)
						}
						field := "summary"
						if shortcut == "+update" {
							field = "task.summary"
						}
						summary = gjson.GetBytes(body, field).String()
						if summary != "override" {
							t.Errorf("override lost: %s", body)
						}
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"task": map[string]interface{}{"guid": "task-fixture", "summary": summary}}})
				}))
				args := []string{"task", shortcut, "--data", test.data, "--as", "user"}
				if shortcut == "+update" {
					args = append(args, "--task-id", "task-fixture")
				}
				if test.explicitSummary {
					args = append(args, "--summary", test.summary)
				}
				for _, dry := range []bool{true, false} {
					invocation := append([]string{}, args...)
					if dry {
						invocation = append(invocation, "--dry-run")
					}
					result := f.Run(invocation...)
					if test.param != "" {
						if result.ExitCode != 2 || gjson.Get(result.Stderr, "error.param").String() != test.param || gjson.Get(result.Stderr, "error.type").String() != "validation" {
							t.Fatalf("wrong input attribution: %+v; want %s", result, test.param)
						}
					} else if result.ExitCode != 0 {
						t.Fatalf("valid override: %+v", result)
					}
				}
				wantWrites := int32(0)
				if test.param == "" {
					wantWrites = 1
				}
				if writes.Load() != wantWrites {
					t.Fatalf("writes %d want %d", writes.Load(), wantWrites)
				}
			})
		}
	}
}
