// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// TestSheets_PrivateUserLifecycle is deliberately opt-in. The operator must
// verify the account's private creation default and parent folder before setting
// the gate. It never creates a parent folder, invites anyone, or retries writes.
func TestSheets_PrivateUserLifecycle(t *testing.T) {
	if os.Getenv("FEISHU_NEXT_SHEETS_PRIVATE_LIVE") != "1" {
		t.Skip("live opt-in required; private account/tenant and folder preflight must precede this test")
	}
	folder := os.Getenv("FEISHU_NEXT_PRIVATE_LIVE_FOLDER_TOKEN")
	owner := os.Getenv("FEISHU_NEXT_PRIVATE_LIVE_OPEN_ID")
	tenant := os.Getenv("FEISHU_NEXT_PRIVATE_LIVE_TENANT_KEY")
	evidence := os.Getenv("FEISHU_NEXT_SHEETS_LIVE_EVIDENCE_DIR")
	for key, value := range map[string]string{"folder": folder, "owner": owner, "tenant": tenant, "evidence directory": evidence} {
		require.NotEmpty(t, value, "missing private live %s", key)
	}
	require.NoError(t, os.Mkdir(evidence, 0o700), "evidence directory must be new for this run")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(cancel)

	// Existing CLI E2E machinery, explicitly one process attempt for every call.
	// Private raw evidence stays outside the Go test log and repository history.
	run := func(callCtx context.Context, name string, args ...string) *clie2e.Result {
		t.Helper()
		result, err := clie2e.RunCmdWithRetry(callCtx, clie2e.Request{Args: args, DefaultAs: "user", WorkDir: evidence}, clie2e.RetryOptions{Attempts: 1})
		require.NoError(t, err, "%s process launch failed", name)
		require.NotNil(t, result)
		raw, err := json.MarshalIndent(map[string]any{"exit_code": result.ExitCode, "stdout": result.Stdout, "stderr": result.Stderr}, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(evidence, name+".json"), raw, 0o600))
		t.Logf("%s: exit=%d; private evidence saved", name, result.ExitCode)
		return result
	}
	ok := func(name string, result *clie2e.Result) {
		t.Helper()
		require.Zero(t, result.ExitCode, "%s failed; inspect private evidence (no automatic replay)", name)
		require.True(t, gjson.Valid(result.Stdout), "%s returned invalid JSON", name)
		require.True(t, gjson.Get(result.Stdout, "ok").Bool(), "%s did not return a successful envelope", name)
	}

	self := run(ctx, "01-self", "api", "GET", "/open-apis/authen/v1/user_info")
	ok("self", self)
	require.True(t, gjson.Get(self.Stdout, "data.open_id").String() == owner, "current account mismatch")
	require.True(t, gjson.Get(self.Stdout, "data.tenant_key").String() == tenant, "current tenant mismatch")

	private := func(label, token, kind string) {
		t.Helper()
		settings := run(ctx, label+"-settings", "drive", "+permission-get-setting", "--token", token, "--type", kind)
		ok(label+" settings", settings)
		require.Equal(t, "closed", gjson.Get(settings.Stdout, "data.permission_public.link_share_entity").String(), "link access must be closed before content writes")
		members := run(ctx, label+"-members", "drive", "+member-list", "--token", token, "--type", kind)
		ok(label+" members", members)
		items := gjson.Get(members.Stdout, "data.items")
		require.True(t, items.IsArray(), "explicit membership array required")
		require.Len(t, items.Array(), 1, "only the current owner may have access")
		member := items.Array()[0]
		require.Equal(t, "openid", member.Get("member_type").String())
		require.True(t, member.Get("member_id").String() == owner, "member is not current account")
		require.Equal(t, "full_access", member.Get("perm").String())
		if kind == "folder" {
			require.Equal(t, "container", member.Get("perm_type").String())
		}
		body, err := json.Marshal(map[string]any{"request_docs": []any{map[string]any{"doc_token": token, "doc_type": kind}}, "with_url": false})
		require.NoError(t, err)
		metadata := run(ctx, label+"-owner", "drive", "metas", "batch_query", "--data", string(body))
		ok(label+" owner", metadata)
		metas := gjson.Get(metadata.Stdout, "data.metas").Array()
		require.Len(t, metas, 1)
		require.True(t, metas[0].Get("doc_token").String() == token, "metadata resource mismatch")
		require.True(t, metas[0].Get("owner_id").String() == owner, "resource owner mismatch")
	}
	private("02-parent", folder, "folder")

	created := run(ctx, "03-create", "sheets", "+workbook-create", "--folder-token", folder, "--title", "next-private-sheets-"+clie2e.GenerateSuffix())
	token := gjson.Get(created.Stdout, "data.spreadsheet.spreadsheet_token").String()
	if token == "" {
		token = gjson.Get(created.Stdout, "data.spreadsheet.token").String()
	}
	if token != "" {
		// Register cleanup immediately from this create response, before asserting
		// any later operation. Never infer an ID from a title or enumerate/delete.
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
			defer cleanupCancel()
			deleted := run(cleanupCtx, "90-cleanup", "drive", "+delete", "--file-token", token, "--type", "sheet", "--yes")
			ok("cleanup", deleted)
			verified := run(cleanupCtx, "91-cleanup-readback", "sheets", "spreadsheets", "get", "--spreadsheet-token", token)
			require.NotZero(t, verified.ExitCode, "cleanup remains unknown: independent read still succeeds")
			require.True(t, gjson.Valid(verified.Stderr), "cleanup remains unknown: typed failure missing")
			problem := gjson.Get(verified.Stderr, "error")
			typedNotFound := problem.Get("subtype").String() == "not_found"
			nativeDeleted := problem.Get("type").String() == "api" && problem.Get("code").Int() == 1310249 && problem.Get("message").String() == "Path param :spreadsheet_token is deleted"
			require.True(t, typedNotFound || nativeDeleted, "cleanup remains unknown unless exact resource read proves not-found/deleted")
		})
	}
	ok("create", created)
	require.NotEmpty(t, token, "unknown create outcome: stop, retain receipt, do not retry")
	private("04-child", token, "sheet")
	info := run(ctx, "05-info", "sheets", "+workbook-info", "--spreadsheet-token", token)
	ok("workbook info", info)
	sheetID := gjson.Get(info.Stdout, "data.sheets.0.sheet_id").String()
	require.NotEmpty(t, sheetID, "new workbook must expose its default sheet")

	cells := `[[{"value":"原样 你好\n✓"},{"value":""},{"value":"0007","cell_styles":{"number_format":"@"}},{"value":0},{"value":false},{"formula":"=1+1"}]]`
	written := run(ctx, "06-write", "sheets", "+cells-set", "--spreadsheet-token", token, "--sheet-id", sheetID, "--range", "A1:F1", "--cells", cells)
	ok("typed write", written)
	read := run(ctx, "07-read", "sheets", "+cells-get", "--spreadsheet-token", token, "--sheet-id", sheetID, "--range", "A1:F1", "--include", "value,formula")
	ok("typed read", read)
	// +cells-get --include formula selects formula rendering, not raw_value.
	// Its native response may format numbers/bools as strings. The explicit
	// native export path below is the source for JSON type assertions.
	artifactPath := filepath.Join(evidence, "native-cells.json")
	exported := run(ctx, "08-native-export", "sheets", "+data-export", "--spreadsheet-token", token, "--sheet-id", sheetID, "--range", "A1:F1", "--save-result", "native-cells.json")
	ok("native export", exported)
	artifact, err := os.ReadFile(artifactPath)
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(artifact, "binding.user_id").String() == owner, "artifact account binding mismatch")
	require.True(t, gjson.GetBytes(artifact, "binding.tenant_key").String() == tenant, "artifact tenant binding mismatch")
	require.True(t, gjson.GetBytes(artifact, "binding.user_verified").Bool())
	require.True(t, gjson.GetBytes(artifact, "binding.tenant_verified").Bool())
	require.Equal(t, "feishu-cli-next/sheets-data/v1", gjson.GetBytes(artifact, "result.data.schema").String())
	require.True(t, gjson.GetBytes(artifact, "result.data.complete").Bool())
	require.Equal(t, "A1:F1", gjson.GetBytes(artifact, "result.data.range").String())
	block := gjson.Get(read.Stdout, "data.ranges.0")
	require.Equal(t, "A1:F1", block.Get("range").String(), "read response must identify the requested extent")
	// Native truncation indicators are optional; when present they must be
	// explicit false. Exact row/column dimensions below establish completeness.
	for _, source := range []gjson.Result{gjson.Get(read.Stdout, "data"), block} {
		for _, name := range []string{"has_more", "truncated", "is_truncated"} {
			if flag := source.Get(name); flag.Exists() {
				require.Equal(t, gjson.False, flag.Type, "present truncation indicator must be false")
			}
		}
	}
	rows := gjson.GetBytes(artifact, "result.data.cells").Array()
	require.Len(t, rows, 1)
	got := rows[0].Array()
	require.Len(t, got, 6)
	for column, want := range map[int]string{0: "原样 你好\n✓", 1: "", 2: "0007"} {
		require.Equal(t, gjson.String, got[column].Get("value").Type, "column %d must retain a JSON string", column)
		require.Equal(t, want, got[column].Get("value").String(), "column %d string changed", column)
	}

	require.Equal(t, gjson.Number, got[3].Get("value").Type)
	require.Zero(t, got[3].Get("value").Float())
	require.Equal(t, gjson.False, got[4].Get("value").Type)
	require.Equal(t, "=1+1", got[5].Get("formula").String(), "formula identity must survive independent read")
	require.Equal(t, gjson.Number, got[5].Get("value").Type)
	require.Equal(t, float64(2), got[5].Get("value").Float())
	t.Log("synthetic typed range verified; cleanup is registered; this is a narrow live lifecycle, not full Sheets parity")
}
