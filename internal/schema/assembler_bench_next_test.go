// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package schema

import (
	"testing"

	"github.com/larksuite/cli/internal/registry"
)

func BenchmarkInputEnvelopes(b *testing.B) {
	snapshot, err := registry.OpenSnapshot()
	if err != nil {
		b.Fatal(err)
	}
	refs := snapshot.Catalog().WalkMethods(nil)
	if len(refs) == 0 {
		b.Skip("embedded catalog contains no methods")
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		InputEnvelopes(nil, refs)
	}
}
