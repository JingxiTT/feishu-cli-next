// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package wiki

import (
	"encoding/json"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestWikiListStopsCursorCyclesWithoutClaimingComplete(t *testing.T) {
	for _, tc := range []struct {
		shortcut common.Shortcut
		path     string
		args     []string
	}{
		{WikiSpaceList, "/open-apis/wiki/v2/spaces", nil},
		{WikiNodeList, "/open-apis/wiki/v2/spaces/123/nodes", []string{"--space-id", "123"}},
		{WikiMemberList, "/open-apis/wiki/v2/spaces/123/members", []string{"--space-id", "123"}},
	} {
		t.Run(tc.shortcut.Command, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			factory, stdout, stderr, reg := cmdutil.TestFactory(t, wikiTestConfig())
			calls := 0
			for _, next := range []string{"A", "B", "A"} {
				reg.Register(&httpmock.Stub{Method: "GET", URL: tc.path, OnMatch: func(r *http.Request) { calls++ },
					Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": []interface{}{}, "members": []interface{}{}, "has_more": true, "page_token": next}}})
			}
			args := append([]string{tc.shortcut.Command, "--as", "bot", "--page-all", "--page-limit", "0"}, tc.args...)
			require.NoError(t, mountAndRunWiki(t, tc.shortcut, args, factory, stdout))
			require.Equal(t, 3, calls)
			var out struct {
				Data struct {
					HasMore   bool   `json:"has_more"`
					PageToken string `json:"page_token"`
				}
			}
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &out))
			require.True(t, out.Data.HasMore)
			require.Equal(t, "A", out.Data.PageToken)
			require.Contains(t, stderr.String(), "results are incomplete")
		})
	}
}
