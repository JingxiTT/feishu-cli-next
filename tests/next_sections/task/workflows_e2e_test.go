// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package task_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestShortcutResourcesAndRecipients(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		paths  []string
		checks map[string]string
	}{
		{"create", []string{"+create", "--summary", "中文 exact", "--description", "body", "--assignee", "cli_fixture", "--follower", "ou_fixture", "--due", "2026-09-08", "--tasklist-id", "list-fixture", "--idempotency-key", "token-fixture"}, []string{"POST /tasks"}, map[string]string{"summary": "中文 exact", "members.0.type": "app", "members.1.role": "follower", "client_token": "token-fixture", "tasklists.0.tasklist_guid": "list-fixture"}},
		{"assign", []string{"+assign", "--task-id", "task-fixture", "--add", "ou_fixture,cli_fixture", "--remove", "ou_remove", "--idempotency-key", "once"}, []string{"POST /tasks/task-fixture/add_members", "POST /tasks/task-fixture/remove_members"}, map[string]string{"members.0.role": "assignee"}},
		{"followers", []string{"+followers", "--task-id", "task-fixture", "--add", "ou_fixture", "--remove", "cli_fixture"}, []string{"POST /tasks/task-fixture/add_members", "POST /tasks/task-fixture/remove_members"}, map[string]string{"members.0.role": "follower"}},
		{"comment", []string{"+comment", "--task-id", "task-fixture", "--content", "literal @mention text"}, []string{"POST /comments"}, map[string]string{"content": "literal @mention text", "resource_id": "task-fixture"}},
		{"ancestor", []string{"+set-ancestor", "--task-id", "task-fixture", "--ancestor-id", "parent-fixture"}, []string{"POST /tasks/task-fixture/set_ancestor_task"}, map[string]string{"ancestor_guid": "parent-fixture"}},
		{"detach", []string{"+set-ancestor", "--task-id", "task-fixture"}, []string{"POST /tasks/task-fixture/set_ancestor_task"}, map[string]string{"ancestor_guid": ""}},
		{"list-create", []string{"+tasklist-create", "--name", "fixture", "--data", `[{"summary":"subtask","assignee":"cli_fixture","extra_number":9007199254740993}]`}, []string{"POST /tasklists", "POST /tasks"}, nil},
		{"list-add", []string{"+tasklist-task-add", "--tasklist-id", "list-fixture", "--task-id", "task-fixture", "--section-guid", "section-fixture"}, []string{"POST /tasks/task-fixture/add_tasklist"}, map[string]string{"tasklist_guid": "list-fixture", "section_guid": "section-fixture"}},
		{"list-members", []string{"+tasklist-members", "--tasklist-id", "list-fixture", "--add", "ou_fixture", "--remove", "ou_remove"}, []string{"POST /tasklists/list-fixture/add_members", "POST /tasklists/list-fixture/remove_members"}, map[string]string{"members.0.role": "editor"}},
		{"list-set-noop", []string{"+tasklist-members", "--tasklist-id", "list-fixture", "--set", "ou_fixture"}, []string{"GET /tasklists/list-fixture"}, nil},
		{"list-read", []string{"+tasklist-members", "--tasklist-id", "list-fixture"}, []string{"GET /tasklists/list-fixture"}, nil},
		{"reminder-set", []string{"+reminder", "--task-id", "task-fixture", "--set", "1h"}, []string{"GET /tasks/task-fixture", "POST /tasks/task-fixture/remove_reminders", "POST /tasks/task-fixture/add_reminders"}, nil},
		{"reminder-noop", []string{"+reminder", "--task-id", "task-fixture", "--set", "30m"}, []string{"GET /tasks/task-fixture"}, nil},
		{"reminder-remove", []string{"+reminder", "--task-id", "task-fixture", "--remove"}, []string{"GET /tasks/task-fixture", "POST /tasks/task-fixture/remove_reminders"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				i := int(calls.Add(1)) - 1
				got := r.Method + " " + strings.TrimPrefix(r.URL.Path, "/open-apis/task/v2")
				if i >= len(tc.paths) || got != tc.paths[i] {
					t.Errorf("call %d: %s want %v", i, got, tc.paths)
				}
				body, _ := io.ReadAll(r.Body)
				for path, want := range tc.checks {
					if gjson.GetBytes(body, path).String() != want {
						t.Errorf("%s: body %s want %q", path, body, want)
					}
				}
				if tc.name == "list-create" && i == 1 {
					if gjson.GetBytes(body, "extra_number").Raw != "9007199254740993" || gjson.GetBytes(body, "members.0.type").String() != "app" {
						t.Errorf("task data lost: %s", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":{"task":{"guid":"task-fixture","summary":"fixture","reminders":[{"id":"reminder-fixture","relative_fire_minute":30}]},"tasklist":{"guid":"list-fixture","name":"fixture","members":[{"id":"ou_fixture","role":"editor","type":"user"}]},"comment":{"id":"comment-fixture","content":"literal @mention text"}}}`)
			}))
			args := append([]string{"task"}, tc.args...)
			args = append(args, "--as", "user")
			dry := f.Run(append(args, "--dry-run")...)
			if dry.ExitCode != 0 || calls.Load() != 0 {
				t.Fatalf("dry run %+v", dry)
			}
			requireSuccess(t, f.Run(args...))
			if int(calls.Load()) != len(tc.paths) {
				t.Fatalf("calls %d want %d", calls.Load(), len(tc.paths))
			}
		})
	}
}

func TestShortcutQueriesPaginateAndPreserveContinuation(t *testing.T) {
	for _, command := range []string{"+search", "+tasklist-search", "+get-my-tasks", "+get-related-tasks"} {
		t.Run(command, func(t *testing.T) {
			var pages atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/task-fixture") || strings.HasSuffix(r.URL.Path, "/list-fixture") {
					_, _ = io.WriteString(w, `{"code":0,"data":{"task":{"guid":"task-fixture","summary":"fixture"},"tasklist":{"guid":"list-fixture","name":"fixture"}}}`)
					return
				}
				i := pages.Add(1)
				if i == 2 && r.URL.Query().Get("page_token") != "next-fixture" {
					t.Errorf("lost continuation %s", r.URL.RawQuery)
				}
				item := map[string]interface{}{"guid": "task-fixture", "id": "task-fixture", "summary": "fixture"}
				if command == "+tasklist-search" {
					item["id"] = "list-fixture"
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": []interface{}{item}, "has_more": i == 1, "page_token": "next-fixture"}})
			}))
			args := []string{"task", command, "--page-all", "--as", "user"}
			if command == "+search" || command == "+tasklist-search" {
				args = append(args, "--query", "fixture")
			}
			result := f.Run(args...)
			requireSuccess(t, result)
			if pages.Load() != 2 {
				t.Fatalf("pages %d: %+v", pages.Load(), result)
			}
		})
	}
}

func TestAttachmentPreservesBytesFilenameAndResource(t *testing.T) {
	content := "synthetic attachment\n中文\x00"
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/open-apis/task/v2/attachments/upload" {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Error(err)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Error(err)
			return
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		if string(body) != content || header.Filename != "fixture.txt" || r.FormValue("resource_id") != "task-fixture" || r.FormValue("resource_type") != "task_delivery" {
			t.Errorf("multipart body %q file %s values %v", body, header.Filename, r.Form)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"items":[{"guid":"attachment-fixture","name":"fixture.txt","size":30}]}}`)
	}))
	file := filepath.Join(f.WorkDir, "fixture.txt")
	if err := os.WriteFile(file, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	requireSuccess(t, f.Run("task", "+upload-attachment", "--resource-id", "https://example.test/task?guid=task-fixture", "--resource-type", "task_delivery", "--file", file, "--as", "user"))
}

func TestMutationPartialAndUnknownDoNotReplay(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial", true: "unknown"}[unknown], func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				i := calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if unknown {
					_, _ = io.WriteString(w, `{`)
					return
				}
				if i == 1 {
					_, _ = io.WriteString(w, `{"code":0,"data":{"task":{"guid":"task-fixture"}}}`)
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, `{"code":99991672,"msg":"forbidden"}`)
			}))
			args := []string{"task", "+assign", "--task-id", "task-fixture", "--add", "ou_fixture", "--as", "user"}
			if !unknown {
				args = append(args, "--remove", "ou_remove")
			}
			result := f.Run(args...)
			want := 21
			if unknown {
				want = 22
			}
			if result.ExitCode != want {
				t.Fatalf("outcome %+v", result)
			}
			if (unknown && calls.Load() != 1) || (!unknown && calls.Load() != 2) {
				t.Fatalf("replayed: %d", calls.Load())
			}
			if !strings.Contains(result.Stderr, map[bool]string{false: `"outcome":"partial"`, true: `"outcome":"unknown"`}[unknown]) {
				t.Fatalf("receipt %s", result.Stderr)
			}
		})
	}
}
