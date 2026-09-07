// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"context"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDriveVersionHistoryDryRun(t *testing.T) {
	setDriveDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"drive", "+version-history",
			"--file-token", "boxcnHistoryDryRun",
			"--limit", "5",
			"--cursor", "1777013761763",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/boxcnHistoryDryRun/history")
	assert.Equal(t, true, clie2e.DryRunGet(output, "api.0.params.only_tag").Bool())
	assert.Equal(t, int64(5), clie2e.DryRunGet(output, "api.0.params.page_size").Int())
	assert.Equal(t, "1777013761763", clie2e.DryRunGet(output, "api.0.params.last_edit_time").String())
}

func TestDriveVersionGetDryRun(t *testing.T) {
	setDriveDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"drive", "+version-get",
			"--file-token", "boxcnVersionDryRun",
			"--version", "7633658129540910621",
			"--output", "./artifact.bin",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/boxcnVersionDryRun/download")
	assert.Equal(t, "7633658129540910621", clie2e.DryRunGet(output, "api.0.params.version").String())
	assert.Equal(t, "./artifact.bin", clie2e.DryRunGet(output, "output").String())
}

func TestDriveVersionGetDryRunWithoutOutputUsesCurrentDirectory(t *testing.T) {
	setDriveDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"drive", "+version-get",
			"--file-token", "boxcnVersionDryRun",
			"--version", "7633658129540910621",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/boxcnVersionDryRun/download")
	assert.Equal(t, "7633658129540910621", clie2e.DryRunGet(output, "api.0.params.version").String())
	assert.Equal(t, ".", clie2e.DryRunGet(output, "output").String())
}

func TestDriveVersionRevertDryRun(t *testing.T) {
	setDriveDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"drive", "+version-revert",
			"--file-token", "boxcnVersionDryRun",
			"--version", "7633658129540910621",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/boxcnVersionDryRun/revert")
	assert.Equal(t, "7633658129540910621", clie2e.DryRunGet(output, "api.0.body.version").String())
}

func TestDriveVersionDeleteDryRun(t *testing.T) {
	setDriveDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"drive", "+version-delete",
			"--file-token", "boxcnVersionDryRun",
			"--version", "7633658129540910621",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/boxcnVersionDryRun/version_del")
	assert.Equal(t, "7633658129540910621", clie2e.DryRunGet(output, "api.0.body.version").String())
}

func TestDriveVersionDryRunSupportsUser(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_USER_ACCESS_TOKEN", "synthetic-drive-dryrun-user")
	setDriveDryRunConfigEnv(t)

	tests := []struct {
		name         string
		args         []string
		wantContains []string
		wantFields   map[string]string
	}{
		{
			name:       "history",
			wantFields: map[string]string{"api.0.params.only_tag": "true", "api.0.params.page_size": "5", "api.0.params.last_edit_time": "1777013761763"},
			args: []string{
				"drive", "+version-history",
				"--file-token", "boxcnHistoryDryRunUser",
				"--limit", "5",
				"--cursor", "1777013761763",
				"--dry-run",
			},
			wantContains: []string{
				"/open-apis/drive/v1/files/boxcnHistoryDryRunUser/history",
			},
		},
		{
			name:       "get",
			wantFields: map[string]string{"api.0.params.version": "7633658129540910621", "output": "./artifact-user.bin"},
			args: []string{
				"drive", "+version-get",
				"--file-token", "boxcnVersionDryRunUser",
				"--version", "7633658129540910621",
				"--output", "./artifact-user.bin",
				"--dry-run",
			},
			wantContains: []string{
				"/open-apis/drive/v1/files/boxcnVersionDryRunUser/download",
			},
		},
		{
			name:       "revert",
			wantFields: map[string]string{"api.0.body.version": "7633658129540910621"},
			args: []string{
				"drive", "+version-revert",
				"--file-token", "boxcnVersionDryRunUser",
				"--version", "7633658129540910621",
				"--dry-run",
			},
			wantContains: []string{
				"/open-apis/drive/v1/files/boxcnVersionDryRunUser/revert",
			},
		},
		{
			name:       "delete",
			wantFields: map[string]string{"api.0.body.version": "7633658129540910621"},
			args: []string{
				"drive", "+version-delete",
				"--file-token", "boxcnVersionDryRunUser",
				"--version", "7633658129540910621",
				"--dry-run",
			},
			wantContains: []string{
				"/open-apis/drive/v1/files/boxcnVersionDryRunUser/version_del",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			t.Cleanup(cancel)

			result, err := clie2e.RunCmd(ctx, clie2e.Request{
				Args:      tt.args,
				DefaultAs: "user",
			})
			require.NoError(t, err)
			result.AssertExitCode(t, 0)

			output := strings.TrimSpace(result.Stdout)
			for path, want := range tt.wantFields {
				assert.Equal(t, want, clie2e.DryRunGet(output, path).String(), path)
			}
			for _, needle := range tt.wantContains {
				assert.Contains(t, output, needle)
			}
		})
	}
}
