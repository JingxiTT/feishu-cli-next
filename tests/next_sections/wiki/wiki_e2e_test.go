// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package wikie2e

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const nodeToken = "wikcnSyntheticNode1234567890"
const objToken = "doccnSyntheticObj12345678901"
const parentToken = "wikcnSyntheticParent12345678"

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func node() map[string]any {
	return map[string]any{"node_token": nodeToken, "obj_token": objToken, "space_id": "123", "obj_type": "docx", "node_type": "origin", "title": "你好 😀", "parent_node_token": parentToken, "has_child": false}
}
func success(t *testing.T, r nextfixture.Result) {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "%+v", r)
	require.True(t, gjson.Get(r.Stdout, "ok").Bool(), "%+v", r)
}

// Each inherited shortcut executes through the real process and transport. The
// handlers assert requests and observed mutations, independently of CLI messages.
func TestWikiNodeLifecycleAndMemberPermissions(t *testing.T) {
	nodes := map[string]map[string]any{}
	members := map[string]any{}
	var stateMu sync.Mutex
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stateMu.Lock()
		defer stateMu.Unlock()
		p := r.URL.Path
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		switch {
		case r.Method == "POST" && p == "/open-apis/wiki/v2/spaces":
			require.Equal(t, "测试空间", body["name"])
			require.Equal(t, "说明 😀", body["description"])
			reply(w, map[string]any{"space": map[string]any{"space_id": "123", "name": body["name"], "description": body["description"], "visibility": "private"}})
		case r.Method == "GET" && p == "/open-apis/wiki/v2/spaces/my_library":
			reply(w, map[string]any{"space": map[string]any{"space_id": "123", "visibility": "private"}})
		case r.Method == "POST" && p == "/open-apis/wiki/v2/spaces/123/nodes":
			n := node()
			n["title"] = body["title"]
			require.Equal(t, "docx", body["obj_type"])
			require.Equal(t, "origin", body["node_type"])
			nodes[nodeToken] = n
			reply(w, map[string]any{"node": n})
		case r.Method == "GET" && p == "/open-apis/wiki/v2/spaces/node_by_token":
			token := r.URL.Query().Get("token")
			require.Contains(t, nodes, token)
			reply(w, map[string]any{"node": nodes[token]})
		case r.Method == "GET" && p == "/open-apis/wiki/v2/spaces/123/nodes":
			require.Equal(t, parentToken, r.URL.Query().Get("parent_node_token"))
			reply(w, map[string]any{"items": []any{nodes[nodeToken]}, "has_more": false})
		case r.Method == "POST" && strings.HasSuffix(p, "/copy"):
			require.Equal(t, "123", body["target_space_id"])
			require.Equal(t, "副本 😀", body["title"])
			n := node()
			n["node_token"] = "copySynthetic1234567890"
			n["obj_token"] = "copyObjSynthetic1234567890"
			n["title"] = body["title"]
			nodes[n["node_token"].(string)] = n
			reply(w, map[string]any{"node": n})
		case r.Method == "POST" && strings.HasSuffix(p, "/move"):
			require.Equal(t, "123", body["target_space_id"])
			nodes[nodeToken]["parent_node_token"] = ""
			reply(w, map[string]any{"node": nodes[nodeToken]})
		case r.Method == "POST" && p == "/open-apis/wiki/v2/spaces/123/members":
			require.Equal(t, "false", r.URL.Query().Get("need_notification"))
			require.Equal(t, "openid", body["member_type"])
			require.Equal(t, "member", body["member_role"])
			members[body["member_id"].(string)] = body
			reply(w, map[string]any{"member": body})
		case r.Method == "GET" && p == "/open-apis/wiki/v2/spaces/123/members":
			reply(w, map[string]any{"members": []any{members["ou_synthetic"]}, "has_more": false})
		case r.Method == "DELETE" && p == "/open-apis/wiki/v2/spaces/123/members/ou_synthetic":
			require.Equal(t, "member", body["member_role"])
			delete(members, "ou_synthetic")
			reply(w, map[string]any{})
		case r.Method == "DELETE" && strings.Contains(p, "/nodes/"):
			require.Equal(t, false, body["include_children"])
			delete(nodes, strings.TrimPrefix(p, "/open-apis/wiki/v2/spaces/123/nodes/"))
			reply(w, map[string]any{})
		case r.Method == "DELETE" && p == "/open-apis/wiki/v2/spaces/123":
			require.Empty(t, nodes)
			require.Empty(t, members)
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	run := func(args ...string) nextfixture.Result {
		r := f.Run(append([]string{"wiki", "--as", "user"}, args...)...)
		stateMu.Lock()
		stateMu.Unlock()
		return r
	}
	success(t, run("+space-create", "--name", "测试空间", "--description", "说明 😀"))
	created := run("+node-create", "--space-id", "my_library", "--title", "你好 😀")
	success(t, created)
	require.Equal(t, nodeToken, gjson.Get(created.Stdout, "data.node_token").String())
	require.Len(t, nodes, 1)
	got := run("+node-get", "--node-token", "https://example.feishu.cn/wiki/"+nodeToken, "--space-id", "123")
	success(t, got)
	require.Equal(t, objToken, gjson.Get(got.Stdout, "data.obj_token").String())
	success(t, run("+node-list", "--space-id", "123", "--parent-node-token", parentToken))
	copied := run("+node-copy", "--space-id", "123", "--node-token", nodeToken, "--target-space-id", "123", "--title", "副本 😀", "--yes")
	success(t, copied)
	require.Len(t, nodes, 2)
	success(t, run("+move", "--node-token", nodeToken, "--source-space-id", "123", "--target-space-id", "123"))
	require.Equal(t, "", nodes[nodeToken]["parent_node_token"])
	success(t, run("+member-add", "--space-id", "123", "--member-id", "ou_synthetic", "--member-type", "openid", "--member-role", "member", "--need-notification=false"))
	require.Len(t, members, 1)
	success(t, run("+member-list", "--space-id", "my_library"))
	removed := run("+member-remove", "--space-id", "123", "--member-id", "ou_synthetic", "--member-type", "openid", "--member-role", "member")
	success(t, removed)
	require.Equal(t, "ou_synthetic", gjson.Get(removed.Stdout, "data.member_id").String())
	require.Empty(t, members)
	for token := range nodes {
		success(t, run("+node-delete", "--node-token", token, "--obj-type", "wiki", "--space-id", "123", "--include-children=false", "--yes"))
	}
	require.Empty(t, nodes)
	success(t, run("+delete-space", "--space-id", "123", "--yes"))
}

func TestWikiPaginationArtifactAndCycle(t *testing.T) {
	for _, kind := range []string{"space", "node", "member"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "GET", r.Method)
				next := "A"
				if r.URL.Query().Get("page_token") == "A" {
					next = "B"
				}
				data := map[string]any{"has_more": true, "page_token": next}
				switch kind {
				case "space":
					data["items"] = []any{map[string]any{"space_id": "123", "name": "你好 😀", "description": "full text", "visibility": "private"}}
				case "node":
					data["items"] = []any{node()}
				case "member":
					data["members"] = []any{map[string]any{"member_id": "ou_synthetic", "member_type": "openid", "member_role": "admin"}}
				}
				reply(w, data)
			}))
			args := []string{"wiki", "+" + kind + "-list", "--as", "user", "--page-all", "--page-limit", "0"}
			if kind != "space" {
				args = append(args, "--space-id", "123")
			}
			result := f.Run(args...)
			success(t, result)
			require.Equal(t, int32(3), calls.Load())
			require.True(t, gjson.Get(result.Stdout, "data.has_more").Bool())
			require.Contains(t, result.Stderr, "results are incomplete")
			artifact := filepath.Join(f.WorkDir, "wiki.json")
			result = f.Run(append(args, "--save-result", artifact)...)
			success(t, result)
			saved, err := os.ReadFile(artifact)
			require.NoError(t, err)
			require.Equal(t, "feishu-cli-next/result/v1", gjson.GetBytes(saved, "schema").String())
			require.True(t, gjson.GetBytes(saved, "binding.tenant_verified").Bool())
			require.True(t, gjson.GetBytes(saved, "result.data.has_more").Bool())
			before := calls.Load()
			again := f.Run(append(args, "--save-result", artifact)...)
			require.NotEqual(t, 0, again.ExitCode)
			require.Equal(t, before, calls.Load())
			resumed := f.Run(append(args, "--page-token", "A")...)
			success(t, resumed)
			require.Equal(t, before+1, calls.Load())
			require.Contains(t, resumed.Stderr, "--page-all is ignored")
		})
	}
}

func TestWikiInvalidInputAndIdentityBeforeBusinessRequests(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{
		{"+node-get", "--node-token", "short/path"},
		{"+node-get", "--node-token", nodeToken, "--token", parentToken},
		{"+node-get", "--node-token", "https://example.feishu.cn/unknown/" + objToken, "--obj-type", "sheet"},
		{"+node-create", "--as", "bot", "--space-id", "my_library"},
		{"+node-create", "--space-id", "123", "--obj-type", "file"},
		{"+node-create", "--space-id", "123", "--node-type", "shortcut"},
		{"+node-list", "--as", "bot", "--space-id", "my_library"},
		{"+node-copy", "--space-id", "123", "--node-token", nodeToken, "--target-space-id", "123", "--target-parent-node-token", parentToken, "--yes"},
		{"+move", "--node-token", nodeToken, "--obj-type", "docx", "--obj-token", objToken},
		{"+member-add", "--as", "bot", "--space-id", "123", "--member-id", "dep_synthetic", "--member-type", "opendepartmentid", "--member-role", "member"},
		{"+space-list", "--page-size", "51"},
		{"+space-list", "--page-limit", "-1"},
		{"+space-create", "--as", "bot", "--name", "invalid identity"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			r := f.Run(append([]string{"wiki"}, args...)...)
			require.NotEqual(t, 0, r.ExitCode, "%+v", r)
			require.Empty(t, r.Stdout)
			require.Contains(t, r.Stderr, "error")
		})
	}
}

func TestWikiMutationUnknownNeverReplaysAndPartialGrant(t *testing.T) {
	t.Run("unknown copy response", func(t *testing.T) {
		var writes atomic.Int32
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "POST", r.Method)
			writes.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"broken":`)
		}))
		r := f.Run("wiki", "+node-copy", "--as", "user", "--space-id", "123", "--node-token", nodeToken, "--target-space-id", "456", "--yes")
		require.Equal(t, 22, r.ExitCode, "%+v", r)
		require.Equal(t, int32(1), writes.Load())
		require.Contains(t, r.Stderr, `"unknown"`)
		listed := f.Run("receipts", "list")
		success(t, listed)
		require.Contains(t, listed.Stdout, "unknown")
		require.Equal(t, int32(1), writes.Load())
	})
	t.Run("node survives rejected auto grant", func(t *testing.T) {
		var writes atomic.Int32
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writes.Add(1)
			if strings.Contains(r.URL.Path, "/permissions/") {
				w.WriteHeader(403)
				_, _ = io.WriteString(w, `{"code":99991679,"msg":"missing scope"}`)
				return
			}
			require.Equal(t, "/open-apis/wiki/v2/spaces/123/nodes", r.URL.Path)
			reply(w, map[string]any{"node": node()})
		}))
		r := f.Run("wiki", "+node-create", "--as", "bot", "--space-id", "123", "--title", "你好 😀")
		require.Equal(t, 21, r.ExitCode, "%+v", r)
		require.Equal(t, int32(2), writes.Load())
		require.False(t, gjson.Get(r.Stdout, "ok").Bool())
		require.Equal(t, nodeToken, gjson.Get(r.Stdout, "data.node_token").String())
		require.Contains(t, r.Stderr, `"partial"`)
	})
}

func TestWikiAuthorizedLiveFixture(t *testing.T) {
	t.Skip("BLOCKED: no verified current-account/tenant Wiki creation visibility fixture. Legacy any-host tests may inherit other-user access; do not run them live. Need a run-owned private space, private Drive source/destination and unambiguous Ying Wu identity before permission delivery.")
}
