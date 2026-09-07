// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"testing"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/assert"
)

func TestBaseShareDryRun(t *testing.T) {
	t.Run("dashboard get", func(t *testing.T) {
		result := runBaseDryRun(t, 0,
			"base", "+dashboard-share-get",
			"--base-token", "app_x",
			"--dashboard-id", "dsh_1",
		)
		assert.Equal(t, "GET", clie2e.DryRunGet(result.Stdout, "api.0.method").String(), result.Stdout)
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/bases/app_x/dashboards/dsh_1/share")
	})

	t.Run("dashboard partial update", func(t *testing.T) {
		result := runBaseDryRun(t, 0,
			"base", "+dashboard-share-update",
			"--base-token", "app_x",
			"--dashboard-id", "dsh_1",
			"--show-source=false",
		)
		assert.Equal(t, "PATCH", clie2e.DryRunGet(result.Stdout, "api.0.method").String(), result.Stdout)
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/bases/app_x/dashboards/dsh_1/share")
		assert.Equal(t, "false", clie2e.DryRunGet(result.Stdout, "api.0.body.settings.show_source").Raw, result.Stdout)
		assert.NotContains(t, result.Stdout, `"enabled":`)
		assert.NotContains(t, result.Stdout, `"enable_auto_analysis":`)
	})

	t.Run("dashboard auto analysis is not exposed", func(t *testing.T) {
		result := runBaseDryRun(t, 2,
			"base", "+dashboard-share-update",
			"--base-token", "app_x",
			"--dashboard-id", "dsh_1",
			"--enable-auto-analysis=true",
		)
		assert.Contains(t, result.Stderr, "unknown flag")
		assert.Contains(t, result.Stderr, "--enable-auto-analysis")
	})

	t.Run("form get", func(t *testing.T) {
		result := runBaseDryRun(t, 0,
			"base", "+form-share-get",
			"--base-token", "app_x",
			"--table-id", "tbl_1",
			"--form-id", "vew_1",
		)
		assert.Equal(t, "GET", clie2e.DryRunGet(result.Stdout, "api.0.method").String(), result.Stdout)
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/bases/app_x/tables/tbl_1/forms/vew_1/share")
	})

	t.Run("form settings update", func(t *testing.T) {
		result := runBaseDryRun(t, 0,
			"base", "+form-share-update",
			"--base-token", "app_x",
			"--table-id", "tbl_1",
			"--form-id", "vew_1",
			"--allow-anonymous=true",
		)
		assert.Equal(t, "PATCH", clie2e.DryRunGet(result.Stdout, "api.0.method").String(), result.Stdout)
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/bases/app_x/tables/tbl_1/forms/vew_1/share")
		assert.Equal(t, "true", clie2e.DryRunGet(result.Stdout, "api.0.body.settings.allow_anonymous").Raw, result.Stdout)
		assert.NotContains(t, result.Stdout, `"enabled":`)
		assert.NotContains(t, result.Stdout, `"require_login":`)
	})

	t.Run("form submission policy is not exposed", func(t *testing.T) {
		result := runBaseDryRun(t, 2,
			"base", "+form-share-update",
			"--base-token", "app_x",
			"--table-id", "tbl_1",
			"--form-id", "vew_1",
			"--valid-period-enabled=true",
		)
		assert.Contains(t, result.Stderr, "unknown flag")
		assert.Contains(t, result.Stderr, "--valid-period-enabled")
	})
}
