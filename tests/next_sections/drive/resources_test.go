// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestResourceAndVersionOperations(t *testing.T) {
	rows := []struct {
		command, flags, method, path string
		request, response            map[string]any
		want                         string
	}{
		{"+apply-permission", "--token doc1 --type docx --perm view --remark synthetic", "POST", "/open-apis/drive/v1/permissions/doc1/members/apply", map[string]any{"perm": "view", "remark": "synthetic"}, map[string]any{}, "confirmed_applied"},
		{"+delete", "--file-token file1 --type file --yes", "DELETE", "/open-apis/drive/v1/files/file1", nil, map[string]any{}, `"deleted":true`},
		{"+inspect", "--url https://example.feishu.cn/file/file1", "POST", "/open-apis/drive/v1/metas/batch_query", map[string]any{"request_docs": []any{map[string]any{"doc_token": "file1", "doc_type": "file"}}}, map[string]any{"metas": []any{map[string]any{"doc_token": "file1", "doc_type": "file", "title": "原文"}}}, "原文"},
		{"+create-folder", "--name 独立 --folder-token root", "POST", "/open-apis/drive/v1/files/create_folder", map[string]any{"name": "独立", "folder_token": "root"}, map[string]any{"token": "created1"}, "created1"},
		{"+create-shortcut", "--file-token file1 --type slides --folder-token root", "POST", "/open-apis/drive/v1/files/create_shortcut", map[string]any{"parent_token": "root", "refer_entity": map[string]any{"refer_token": "file1", "refer_type": "slides"}}, map[string]any{"succ_shortcut_node": map[string]any{"token": "shortcut1", "name": "reference"}}, "shortcut1"},
		{"+version-history", "--file-token file1 --limit 1 --cursor 99", "GET", "/open-apis/drive/v1/files/file1/history", nil, map[string]any{"items": []any{map[string]any{"version": "7", "edit_time": "98", "name": "原文", "size": 3}}, "has_more": true}, `"next_cursor":"98"`},
		{"+version-revert", "--file-token file1 --version 7", "POST", "/open-apis/drive/v1/files/file1/revert", map[string]any{"version": "7"}, map[string]any{}, "confirmed_applied"},
		{"+version-delete", "--file-token file1 --version 7 --yes", "POST", "/open-apis/drive/v1/files/file1/version_del", map[string]any{"version": "7"}, map[string]any{}, "confirmed_applied"},
		{"+secure-label-list", "--page-size 1 --page-token before --lang en", "GET", "/open-apis/drive/v2/my_secure_labels", nil, map[string]any{"items": []any{map[string]any{"id": "7", "name": "Private"}}, "has_more": true, "page_token": "next"}, "next"},
		{"+secure-label-update", "--token file1 --type file --label-id 7", "PATCH", "/open-apis/drive/v2/files/file1/secure_label", map[string]any{"id": "7"}, map[string]any{"id": "7"}, "confirmed_applied"},
		{"+preview", "--file-token file1 --list-only", "POST", "/open-apis/drive/v1/medias/file1/preview_result", map[string]any{}, map[string]any{"preview_results": []any{map[string]any{"preview_type": "14", "preview_status": "0"}, map[string]any{"preview_type": "0", "preview_status": "1"}}}, "processing"},
		{"+task_result", "--scenario task_check --task-id task1", "GET", "/open-apis/drive/v1/files/task_check", nil, map[string]any{"task_id": "task1", "status": "success"}, "success"},
	}
	for _, row := range rows {
		t.Run(row.command, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != row.method || r.URL.Path != row.path {
					t.Errorf("wrong request %s %s", r.Method, r.URL)
				}
				if row.request != nil {
					got := body(t, r)
					if !reflect.DeepEqual(got, row.request) {
						t.Errorf("body changed: got %#v want %#v", got, row.request)
					}
				}
				reply(w, row.response)
			}))
			args := append([]string{"drive", row.command, "--as", "user"}, strings.Fields(row.flags)...)
			got := f.Run(args...)
			successful(t, got)
			if !strings.Contains(strings.ToLower(got.Stdout), strings.ToLower(row.want)) || calls.Load() != 1 {
				t.Fatalf("wrong result/count: %s calls=%d", got.Stdout, calls.Load())
			}
		})
	}
}
