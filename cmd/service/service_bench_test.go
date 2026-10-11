// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/registry"
	"github.com/spf13/cobra"
)

func BenchmarkMethodCommandSpecs(b *testing.B) {
	refs := registry.RuntimeCatalog().WalkMethods(nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, ref := range refs {
			newMethodCommandSpec(ref)
		}
	}
}

func BenchmarkRegisterServiceCommands(b *testing.B) {
	catalog := registry.RuntimeCatalog()
	f := &cmdutil.Factory{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RegisterServiceCommandsFromCatalog(context.Background(), &cobra.Command{Use: "lark-cli"}, f, catalog)
	}
}
