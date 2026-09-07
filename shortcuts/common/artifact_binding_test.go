// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"errors"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/spf13/cobra"
)

func TestRuntimeArtifactBindingUsesRuntimeIdentity(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	cfg := &core.CliConfig{Brand: core.BrandFeishu, AppID: "app", ProfileName: "selected", UserOpenId: "irrelevant-user"}
	f, _, _, _ := cmdutil.TestFactory(t, cfg)
	f.ResolvedIdentity = core.AsUser
	runtime := TestNewRuntimeContextForAPI(context.Background(), &cobra.Command{}, cfg, f, core.AsBot)
	binding, err := runtime.ResolveArtifactBinding()
	if err != nil || binding.Identity != "bot" || binding.UserID != "" || binding.TenantVerified || binding.Endpoint != core.ResolveEndpoints(core.BrandFeishu).Open {
		t.Fatalf("runtime identity lost: %+v %v", binding, err)
	}
	if err := ValidateArtifactBinding(binding, binding, "--input", false); err != nil {
		t.Fatal(err)
	}
	binding.AppID = "wrong-app"
	current, _ := runtime.ResolveArtifactBinding()
	err = ValidateArtifactBinding(binding, current, "--input", false)
	var validation *errs.ValidationError
	if !errors.As(err, &validation) || validation.Param != "--input" {
		t.Fatalf("mismatch not rejected: %v", err)
	}
}

func TestRuntimeArtifactBindingDryRunDoesNotResolveCredentials(t *testing.T) {
	f := &cmdutil.Factory{Config: func() (*core.CliConfig, error) { t.Fatal("dry run resolved credentials"); return nil, nil }}
	runtime := &RuntimeContext{offline: true, Factory: f}
	if _, err := runtime.ResolveArtifactBinding(); !errs.IsValidation(err) {
		t.Fatalf("dry run binding error: %v", err)
	}
}
