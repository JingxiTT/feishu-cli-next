// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package shortcuts

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/internal/core"
)

func TestRegisterBrandGuardDoesNotResolveCredentialConfig(t *testing.T) {
	for _, brand := range []core.LarkBrand{core.BrandFeishu, core.BrandLark} {
		t.Run(string(brand), func(t *testing.T) {
			factory := newFactoryWithBrand(brand)
			factory.Config = func() (*core.CliConfig, error) { t.Fatal("registration resolved credential config"); return nil, nil }
			program := &cobra.Command{Use: "root"}
			RegisterShortcuts(program, factory)
			apps := findChild(program, "apps")
			if apps == nil || apps.Hidden != (brand == core.BrandLark) {
				t.Fatalf("startup brand restriction not preserved for %s", brand)
			}
		})
	}
}
