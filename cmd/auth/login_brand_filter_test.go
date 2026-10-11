// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package auth

import (
	"slices"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/registry"
)

func TestBrandFilter_AppsExcludedOnLark(t *testing.T) {
	feishuDomains := builtinResolver(t).allKnown(core.BrandFeishu)
	if !feishuDomains["apps"] {
		t.Errorf("expected apps domain to be known on Feishu brand")
	}

	larkDomains := builtinResolver(t).allKnown(core.BrandLark)
	if larkDomains["apps"] {
		t.Errorf("expected apps domain to be EXCLUDED on Lark brand")
	}

	feishuScopes := builtinResolver(t).scopesFor([]string{"apps"}, "user", core.BrandFeishu)
	if len(feishuScopes) == 0 {
		t.Errorf("expected non-empty scopes for apps on Feishu brand, got %d", len(feishuScopes))
	}

	larkScopes := builtinResolver(t).scopesFor([]string{"apps"}, "user", core.BrandLark)
	if len(larkScopes) != 0 {
		t.Errorf("expected empty scopes for apps on Lark brand, got %d: %v", len(larkScopes), larkScopes)
	}
}
func TestLoginHelpBrandDoesNotResolveCredentialConfig(t *testing.T) {
	priorCompletion := cmdutil.FlagCompletionsEnabled()
	cmdutil.SetFlagCompletionsEnabled(true)
	t.Cleanup(func() { cmdutil.SetFlagCompletionsEnabled(priorCompletion) })
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for _, brand := range []core.LarkBrand{core.BrandFeishu, core.BrandLark} {
		t.Run(string(brand), func(t *testing.T) {
			factory, _, _, _ := cmdutil.TestFactory(t, &core.CliConfig{Brand: brand})
			snapshot, err := registry.OpenSnapshot()
			if err != nil {
				t.Fatal(err)
			}
			factory.APICatalog = snapshot.Catalog()
			factory.Config = func() (*core.CliConfig, error) {
				t.Fatal("constructor/help resolved credential config")
				return nil, nil
			}
			login := NewCmdAuthLogin(factory, nil)
			want := "available: " + strings.Join(builtinResolver(t).sorted(brand), ", ") + ", all"
			if usage := login.Flag("domain").Usage; !strings.Contains(usage, want) {
				t.Fatalf("%s help: %s", brand, usage)
			}
			completion, ok := login.GetFlagCompletionFunc("domain")
			if !ok {
				t.Fatal("missing domain completion")
			}
			domains, _ := completion(login, nil, "apps")
			if slices.Contains(domains, "apps") != (brand == core.BrandFeishu) {
				t.Fatalf("%s domain completion: %v", brand, domains)
			}
		})
	}
}
