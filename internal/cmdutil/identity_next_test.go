// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"testing"

	"github.com/larksuite/cli/internal/core"
)

func resolveIdentityForCommand(t *testing.T, f *Factory, asFlag string, changed bool, supported []string) (core.Identity, error) {
	t.Helper()
	cmd := newCmdWithAsFlag(asFlag, changed)
	as := f.ResolveAs(context.Background(), cmd, core.Identity(asFlag))
	as = f.ResolveSupportedIdentity(as, supported)
	return as, f.CheckIdentity(as, supported)
}

func TestResolveSupportedIdentity_AutoDetectedBotUsesOnlySupportedUser(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s"})

	as, err := resolveIdentityForCommand(t, f, "", false, []string{"user"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if as != core.AsUser {
		t.Errorf("identity = %s, want user", as)
	}
	if f.ResolvedIdentity != core.AsUser {
		t.Errorf("ResolvedIdentity = %s, want user", f.ResolvedIdentity)
	}
}

func TestResolveSupportedIdentity_ExplicitBotStillRejected(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s"})

	as, err := resolveIdentityForCommand(t, f, "bot", true, []string{"user"})
	if err == nil {
		t.Fatal("expected explicit --as bot to be rejected")
	}
	if as != core.AsBot {
		t.Errorf("identity = %s, want bot", as)
	}
	want := "--as bot is not supported, this command only supports: user"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestResolveSupportedIdentity_DefaultAsBotStillRejected(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s", DefaultAs: "bot"})

	as, err := resolveIdentityForCommand(t, f, "auto", false, []string{"user"})
	if err == nil {
		t.Fatal("expected default-as bot to be rejected")
	}
	if as != core.AsBot {
		t.Errorf("identity = %s, want bot", as)
	}
	want := "--as bot is not supported, this command only supports: user"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestResolveSupportedIdentity_StrictModeForcedBotStillRejected(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s", SupportedIdentities: 2})

	as, err := resolveIdentityForCommand(t, f, "", false, []string{"user"})
	if err == nil {
		t.Fatal("expected strict-mode bot to be rejected")
	}
	if as != core.AsBot {
		t.Errorf("identity = %s, want bot", as)
	}
	if f.IdentityAutoDetected {
		t.Error("strict-mode forced identity must not count as auto-detected")
	}
}

func TestResolveSupportedIdentity_BothIdentitiesKeepsAutoDetect(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s"})

	as, err := resolveIdentityForCommand(t, f, "", false, []string{"user", "bot"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if as != core.AsBot {
		t.Errorf("identity = %s, want bot", as)
	}
}

func TestResolveSupportedIdentity_MultipleSupportedIdentitiesIgnored(t *testing.T) {
	f, _, _, _ := TestFactory(t, &core.CliConfig{AppID: "a", AppSecret: "s"})
	f.IdentityAutoDetected = true

	if got := f.ResolveSupportedIdentity(core.AsBot, []string{"user", "bot"}); got != core.AsBot {
		t.Errorf("identity = %s, want bot", got)
	}
	if got := f.ResolveSupportedIdentity(core.AsBot, nil); got != core.AsBot {
		t.Errorf("identity with no supported list = %s, want bot", got)
	}
}
