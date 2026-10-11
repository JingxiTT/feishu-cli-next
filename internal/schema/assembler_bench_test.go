// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package schema

import (
	"testing"

	"github.com/larksuite/cli/internal/registry"
)

func BenchmarkInputEnvelopes(b *testing.B) {
	refs := registry.EmbeddedCatalog().WalkMethods(nil)
	if len(refs) == 0 {
		b.Skip("restore the frozen catalog with make fetch_meta")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InputEnvelopes(refs)
	}
}
