// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"errors"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
)

func TestResolveArtifactBindingUsesRequestedEffectiveIdentity(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, _, _, _ := TestFactory(t, &core.CliConfig{Brand: core.BrandLark, AppID: "app", ProfileName: "selected", UserOpenId: "selected-user"})
	f.Credential = credential.NewCredentialProvider(nil, artifactVerifiedAccount{}, nil, nil)
	f.ResolvedIdentity = core.AsBot
	binding, err := f.ResolveArtifactBinding(context.Background(), core.AsUser)
	if err != nil || binding.Identity != "user" || binding.UserID != "selected-user" || binding.Profile != "selected" || binding.Endpoint != core.ResolveEndpoints(core.BrandLark).Open {
		t.Fatalf("wrong runtime binding: %+v %v", binding, err)
	}
}

func TestResolveArtifactBindingPreservesConfigError(t *testing.T) {
	cause := errors.New("synthetic cause")
	expected := errs.NewConfigError(errs.SubtypeNotConfigured, "fixture missing configuration").WithCause(cause)
	f := &Factory{Config: func() (*core.CliConfig, error) { return nil, expected }}
	_, err := f.ResolveArtifactBinding(context.Background(), core.AsUser)
	if err != expected || !errors.Is(err, cause) {
		t.Fatalf("configuration error changed: %v", err)
	}
}

func TestResolveArtifactBindingRejectsIncompleteIdentity(t *testing.T) {
	for _, identity := range []core.Identity{"", core.AsAuto, core.AsUser} {
		f := &Factory{Config: func() (*core.CliConfig, error) { return &core.CliConfig{Brand: core.BrandFeishu, AppID: "app"}, nil }}
		if _, err := f.ResolveArtifactBinding(context.Background(), identity); !errs.IsValidation(err) {
			t.Fatalf("incomplete identity %q accepted: %v", identity, err)
		}
	}
}

type artifactVerifiedAccount struct{}

func (artifactVerifiedAccount) ResolveAccount(context.Context) (*credential.Account, error) {
	return &credential.Account{AppID: "app", Brand: core.BrandLark, UserOpenId: "selected-user", TenantKey: "tenant", UserInfoVerified: true}, nil
}
