// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
)

func TestBuildAuthHelpUsesStartupBrandSnapshot(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for _, brand := range []core.LarkBrand{core.BrandFeishu, core.BrandLark} {
		runtime, root, _ := buildInternal(context.Background(), cmdutil.InvocationContext{},
			WithStartupBrand(brand), WithoutPlugins(), WithoutStrictMode(), WithoutServiceCommands(),
			WithIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}))
		login, _, err := root.Find([]string{"auth", "login"})
		if err != nil {
			t.Fatal(err)
		}
		if runtime.Factory.StartupBrand != brand {
			t.Fatalf("build brand lost: %s", runtime.Factory.StartupBrand)
		}
		if strings.Contains(login.Flag("domain").Usage, "apps,") != (brand == core.BrandFeishu) {
			t.Fatalf("%s auth domain help: %s", brand, login.Flag("domain").Usage)
		}
	}
}
