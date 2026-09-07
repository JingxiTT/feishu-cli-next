// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package wikie2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestWikiAsyncTransfersAndDeletesKeepTaskEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, taskType, resultKey string
		args                                    []string
		result                                  any
	}{
		{"drive to wiki", "POST", "/open-apis/wiki/v2/spaces/123/nodes/move_docs_to_wiki", "move", "move_result", []string{"+move", "--obj-type", "docx", "--obj-token", objToken, "--target-space-id", "123"}, []any{map[string]any{"status": 0, "node": node()}}},
		{"wiki to drive", "POST", "/open-apis/wiki/v2/nodes/" + nodeToken + "/move_wiki_to_docs", "move_wiki_to_docs", "move_wiki_to_docs_result", []string{"+move-to-drive", "--node-token", nodeToken, "--folder-token", "folderSynthetic"}, map[string]any{"status": 0, "obj_type": "docx", "obj_token": objToken, "url": "https://example.feishu.cn/docx/" + objToken}},
		{"delete node", "DELETE", "/open-apis/wiki/v2/spaces/123/nodes/" + nodeToken, "delete_node", "simple_task_result", []string{"+node-delete", "--node-token", nodeToken, "--obj-type", "wiki", "--space-id", "123", "--yes"}, map[string]any{"status": "success"}},
		{"delete space", "DELETE", "/open-apis/wiki/v2/spaces/123", "delete_space", "delete_space_result", []string{"+delete-space", "--space-id", "123", "--yes"}, map[string]any{"status": "success"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var writes, polls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == tc.method && r.URL.Path == tc.path {
					writes.Add(1)
					if tc.taskType == "move_wiki_to_docs" {
						var b map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
						require.Equal(t, "folderSynthetic", b["folder_token"])
					}
					reply(w, map[string]any{"task_id": "taskSynthetic"})
					return
				}
				require.Equal(t, "GET", r.Method)
				require.Equal(t, "/open-apis/wiki/v2/tasks/taskSynthetic", r.URL.Path)
				require.Equal(t, tc.taskType, r.URL.Query().Get("task_type"))
				polls.Add(1)
				reply(w, map[string]any{"task": map[string]any{"task_id": "taskSynthetic", tc.resultKey: tc.result}})
			}))
			r := f.Run(append([]string{"wiki", "--as", "user"}, tc.args...)...)
			// The transport conservatively retains async acceptance as unknown even
			// when the domain poll reports ready. Do not mistake it for failed dispatch.
			require.Equal(t, 22, r.ExitCode, "%+v", r)
			require.False(t, gjson.Get(r.Stdout, "ok").Bool())
			require.True(t, gjson.Get(r.Stdout, "data.ready").Bool(), "%+v", r)
			require.Equal(t, "taskSynthetic", gjson.Get(r.Stdout, "data.task_id").String())
			require.Contains(t, r.Stderr, "unknown")
			require.Equal(t, int32(1), writes.Load())
			require.Equal(t, int32(1), polls.Load())
			result := f.Run("api", "get", "/open-apis/wiki/v2/tasks/taskSynthetic", "--params", `{"task_type":"`+tc.taskType+`"}`, "--as", "user")
			success(t, result)
			require.Equal(t, int32(1), writes.Load())
			require.Equal(t, int32(2), polls.Load())
		})
	}
}

func TestWikiReadPermissionAndSpaceMismatch(t *testing.T) {
	for _, mode := range []string{"scope", "space mismatch"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "GET", r.Method)
				if mode == "scope" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"code":99991679,"msg":"missing scope","error":{"permission_violations":[{"type":"scope","subject":"wiki:node:read"}]}}`))
					return
				}
				reply(w, map[string]any{"node": node()})
			}))
			r := f.Run("wiki", "+node-get", "--as", "user", "--node-token", nodeToken, "--space-id", "456")
			require.NotEqual(t, 0, r.ExitCode, "%+v", r)
			require.Empty(t, r.Stdout)
			require.Equal(t, int32(1), calls.Load())
			require.Contains(t, r.Stderr, "error")
			if mode == "space mismatch" {
				require.True(t, strings.Contains(r.Stderr, "space"))
			}
		})
	}
}

func TestWikiAsyncFailureIsNotReplayed(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			writes.Add(1)
			reply(w, map[string]any{"task_id": "taskSynthetic"})
			return
		}
		require.Equal(t, "GET", r.Method)
		reply(w, map[string]any{"task": map[string]any{"task_id": "taskSynthetic", "move_wiki_to_docs_result": map[string]any{"status": -1, "status_msg": "synthetic permission changed"}}})
	}))
	r := f.Run("wiki", "+move-to-drive", "--as", "user", "--node-token", nodeToken)
	require.Equal(t, 22, r.ExitCode, "%+v", r)
	require.Contains(t, r.Stderr, "taskSynthetic")
	require.Contains(t, r.Stderr, "unknown")
	require.Equal(t, int32(1), writes.Load())
}

func TestWikiRejectedMutationIsConfirmedUnapplied(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(403)
		_, _ = w.Write([]byte(`{"code":99991679,"msg":"missing scope"}`))
	}))
	r := f.Run("wiki", "+space-create", "--as", "user", "--name", "synthetic")
	require.Equal(t, 20, r.ExitCode, "%+v", r)
	require.Contains(t, r.Stderr, "confirmed_unapplied")
	require.Equal(t, int32(1), writes.Load())
}
