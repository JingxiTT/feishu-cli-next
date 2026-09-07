// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package wikie2e

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestWikiAllGeneratedEndpointsPreserveRequestsAndFullOutput(t *testing.T) {
	for _, tc := range []struct{ name, method, path, params, body string }{
		{"spaces list", "GET", "/open-apis/wiki/v2/spaces", `{"page_size":2,"page_token":"opaque_cursor"}`, ""},
		{"spaces get", "GET", "/open-apis/wiki/v2/spaces/123", `{"space_id":"123","lang":"en"}`, ""},
		{"spaces get_node", "GET", "/open-apis/wiki/v2/spaces/get_node", `{"token":"native_token","obj_type":"slides"}`, ""},
		{"spaces create", "POST", "/open-apis/wiki/v2/spaces", `{}`, `{"name":"你好 😀","description":"  preserved\n"}`},
		{"nodes list", "GET", "/open-apis/wiki/v2/spaces/123/nodes", `{"space_id":"123","parent_node_token":"parent","page_size":2}`, ""},
		{"nodes create", "POST", "/open-apis/wiki/v2/spaces/123/nodes", `{"space_id":"123"}`, `{"title":"你好 😀","node_type":"shortcut","obj_type":"file","origin_node_token":"origin"}`},
		{"nodes copy", "POST", "/open-apis/wiki/v2/spaces/123/nodes/native_token/copy", `{"space_id":"123","node_token":"native_token"}`, `{"target_space_id":"456","target_parent_token":"parent","title":"copy"}`},
		{"members list", "GET", "/open-apis/wiki/v2/spaces/123/members", `{"space_id":"123","page_size":2,"page_token":"opaque_cursor"}`, ""},
		{"members create", "POST", "/open-apis/wiki/v2/spaces/123/members", `{"space_id":"123","need_notification":false}`, `{"member_type":"appid","member_id":"cli_synthetic","member_role":"admin"}`},
		{"members delete", "DELETE", "/open-apis/wiki/v2/spaces/123/members/cli_synthetic", `{"space_id":"123","member_id":"cli_synthetic"}`, `{"member_type":"appid","member_role":"admin"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, tc.method, r.Method)
				require.Equal(t, tc.path, r.URL.Path)
				if tc.body != "" {
					var got any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
					var want any
					require.NoError(t, json.Unmarshal([]byte(tc.body), &want))
					require.Equal(t, want, got)
				}
				for _, key := range []string{"page_size", "page_token", "lang", "obj_type", "token", "parent_node_token", "need_notification"} {
					if v := gjson.Get(tc.params, key); v.Exists() {
						require.Equal(t, v.String(), r.URL.Query().Get(key), key)
					}
				}
				reply(w, map[string]any{"native_id": "001", "nullable": nil, "empty": "", "title": "你好 😀", "items": []any{}, "has_more": false})
			}))
			args := append([]string{"wiki"}, strings.Fields(tc.name)...)
			args = append(args, "--as", "user", "--params", tc.params)
			if tc.method == "GET" {
				args = append(args, "--format", "json")
			} else {
				args = append(args, "--data", tc.body)
				if tc.name == "spaces create" || strings.HasPrefix(tc.name, "members ") {
					args = append(args, "--yes")
				}
			}
			r := f.Run(args...)
			success(t, r)
			require.Equal(t, int32(1), calls.Load())
			require.Equal(t, "001", gjson.Get(r.Stdout, "data.native_id").String())
			require.True(t, gjson.Get(r.Stdout, "data.nullable").Exists())
			require.Equal(t, "你好 😀", gjson.Get(r.Stdout, "data.title").String())
		})
	}
}

func TestWikiSupportedKindsAndNotificationDryRun(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, kind := range []string{"sheet", "mindnote", "bitable", "docx", "slides", "file"} {
		t.Run("create_"+kind, func(t *testing.T) {
			args := []string{"wiki", "+node-create", "--as", "user", "--space-id", "123", "--obj-type", kind, "--title", "测试 😀", "--dry-run"}
			if kind == "file" {
				args = append(args, "--node-type", "shortcut", "--origin-node-token", nodeToken)
			}
			r := f.Run(args...)
			require.Equal(t, 0, r.ExitCode, "%+v", r)
			require.Contains(t, r.Stdout, kind)
		})
	}
	for _, kind := range []string{"doc", "sheet", "mindnote", "bitable", "docx", "slides", "file"} {
		t.Run("move_get_"+kind, func(t *testing.T) {
			for _, args := range [][]string{
				{"wiki", "+move", "--as", "user", "--target-space-id", "123", "--obj-type", kind, "--obj-token", objToken, "--apply", "--dry-run"},
				{"wiki", "+node-get", "--as", "user", "--node-token", objToken, "--obj-type", kind, "--dry-run"},
				{"wiki", "+node-delete", "--as", "user", "--node-token", objToken, "--obj-type", kind, "--dry-run"},
			} {
				r := f.Run(args...)
				require.Equal(t, 0, r.ExitCode, "%+v", r)
				require.Contains(t, r.Stdout, kind)
			}
		})
	}
	for _, kind := range []string{"openid", "userid", "email", "unionid", "openchat", "opendepartmentid", "appid"} {
		for _, role := range []string{"admin", "member"} {
			t.Run(kind+role, func(t *testing.T) {
				for _, notify := range []string{"", "--need-notification=false", "--need-notification=true"} {
					args := []string{"wiki", "+member-add", "--as", "user", "--space-id", "my_library", "--member-id", "synthetic", "--member-type", kind, "--member-role", role, "--dry-run"}
					if notify != "" {
						args = append(args, notify)
					}
					r := f.Run(args...)
					require.Equal(t, 0, r.ExitCode, "%+v", r)
					require.Contains(t, r.Stdout, kind)
					if notify == "" {
						require.NotContains(t, r.Stdout, `"need_notification"`)
					}
				}
			})
		}
	}
}

func TestWikiGenericJSONFileInputPreservesWhitespace(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "  标题\n", body["name"])
		require.Equal(t, "", body["description"])
		reply(w, map[string]any{"space": map[string]any{"space_id": "123"}})
	}))
	path := filepath.Join(f.WorkDir, "payload.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"name":"  标题\n","description":""}`), 0600))
	success(t, f.Run("wiki", "spaces", "create", "--as", "user", "--data", "@"+path, "--yes"))
}

func TestWikiEveryMappedCommandHasHelpAndDiscoveryOrSchema(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	groups := []string{"wiki", "wiki members", "wiki nodes", "wiki spaces"}
	leaves := []string{"+delete-space", "+member-add", "+member-list", "+member-remove", "+move", "+move-to-drive", "+node-copy", "+node-create", "+node-delete", "+node-get", "+node-list", "+space-create", "+space-list", "members create", "members delete", "members list", "nodes copy", "nodes create", "nodes list", "spaces create", "spaces get", "spaces get_node", "spaces list"}
	for _, command := range groups {
		r := f.Run(append(strings.Fields(command), "--help")...)
		require.Equal(t, 0, r.ExitCode, "%+v", r)
		require.NotEmpty(t, r.Stdout)
	}
	for _, command := range leaves {
		t.Run(command, func(t *testing.T) {
			args := append([]string{"wiki"}, strings.Fields(command)...)
			r := f.Run(append(args, "--help")...)
			require.Equal(t, 0, r.ExitCode, "%+v", r)
			require.NotEmpty(t, r.Stdout)
			if strings.HasPrefix(command, "+") {
				r = f.Run(append(append([]string{"discover"}, args...), "--flags")...)
				require.Equal(t, "wiki "+command, gjson.Get(r.Stdout, "data.0.command").String(), "%+v", r)
			} else {
				r = f.Run(append([]string{"schema"}, args...)...)
			}
			require.Equal(t, 0, r.ExitCode, "%+v", r)
			require.True(t, json.Valid([]byte(r.Stdout)), "%+v", r)
		})
	}
}
