// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/require"
)

func TestBaseRecordSkipUnchangedDryRun(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		args                   []string
		method, path, bodyPath string
	}{
		{"upsert", []string{"+record-upsert", "--record-id", "recA", "--json", `{"Score":9007199254740993,"Clear":null}`}, "PATCH", "/records/recA", "Score"},
		{"batch", []string{"+record-batch-update", "--json", `{"update_records":{"recB":{"Name":""},"recA":{"Score":9007199254740993,"Clear":null}}}`}, "POST", "/records/batch_update", "update_records.recA.Score"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setBaseDryRunConfigEnv(t)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)
			args := append([]string{"base"}, tc.args...)
			args = append(args, "--base-token", "app_x", "--table-id", "tbl_x", "--skip-unchanged", "--dry-run")
			result, err := clie2e.RunCmd(ctx, clie2e.Request{Args: args, DefaultAs: "bot"})
			require.NoError(t, err)
			result.AssertExitCode(t, 0)
			out := result.Stdout
			require.Equal(t, "POST", clie2e.DryRunGet(out, "api.0.method").String(), out)
			require.Equal(t, "/open-apis/base/v3/bases/app_x/tables/tbl_x/records/batch_get", clie2e.DryRunGet(out, "api.0.url").String(), out)
			require.Equal(t, "recA", clie2e.DryRunGet(out, "api.0.body.record_id_list.0").String(), out)
			require.Equal(t, tc.method, clie2e.DryRunGet(out, "api.1.method").String(), out)
			require.Equal(t, "/open-apis/base/v3/bases/app_x/tables/tbl_x"+tc.path, clie2e.DryRunGet(out, "api.1.url").String(), out)
			require.Equal(t, "9007199254740993", clie2e.DryRunGet(out, "api.1.body."+tc.bodyPath).Raw, out)
			require.True(t, clie2e.DryRunGet(out, "comparison.skip_unchanged").Bool(), out)
			require.False(t, clie2e.DryRunGet(out, "comparison.performed").Bool(), out)
			require.False(t, clie2e.DryRunGet(out, "comparison.atomic").Bool(), out)
			require.Equal(t, "base:record:read", clie2e.DryRunGet(out, "comparison.requires_scope").String(), out)
		})
	}
}

func TestBaseRecordSkipUnchangedCreateDryRunRejects(t *testing.T) {
	setBaseDryRunConfigEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	result, err := clie2e.RunCmd(ctx, clie2e.Request{Args: []string{"base", "+record-upsert", "--base-token", "app_x", "--table-id", "tbl_x", "--json", `{"Name":"Alice"}`, "--skip-unchanged", "--dry-run"}, DefaultAs: "bot"})
	require.NoError(t, err)
	result.AssertExitCode(t, 2)
	require.Contains(t, result.Stderr, "--skip-unchanged")
}
