// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBaseDashboardBlockGetDataDryRun(t *testing.T) {
	setBaseDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"base", "+dashboard-block-get-data",
			"--base-token", "app_x",
			"--block-id", "blk_chart",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/base/v3/bases/app_x/dashboards/blocks/blk_chart/data")
	assert.Equal(t, "GET", clie2e.DryRunGet(output, "api.0.method").String(), output)
	assert.Equal(t, "blk_chart", clie2e.DryRunGet(output, "block_id").String(), output)
	assert.Equal(t, "app_x", clie2e.DryRunGet(output, "base_token").String(), output)
}

func TestBaseDashboardBlockGetDataDryRun_IgnoresDashboardID(t *testing.T) {
	setBaseDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"base", "+dashboard-block-get-data",
			"--base-token", "app_x",
			"--dashboard-id", "dsh_ignored",
			"--block-id", "blk_chart",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/base/v3/bases/app_x/dashboards/blocks/blk_chart/data")
	assert.Equal(t, "blk_chart", clie2e.DryRunGet(output, "block_id").String(), output)
	assert.NotContains(t, output, "dsh_ignored")
}

func TestBaseDashboardBlockGetDataDryRun_MissingRequiredFlags(t *testing.T) {
	setBaseDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"base", "+dashboard-block-get-data",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	assert.NotEqual(t, 0, result.ExitCode)
	assert.Contains(t, result.Stderr, "base-token")
	assert.Contains(t, result.Stderr, "block-id")
}
