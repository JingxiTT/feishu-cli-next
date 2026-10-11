// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package im

import (
	"strings"
	"testing"
)

func BenchmarkValidateMessageContentJSON(b *testing.B) {
	for _, size := range []struct {
		name  string
		bytes int
	}{{"1KiB", 1024}, {"64KiB", 64 * 1024}} {
		b.Run(size.name, func(b *testing.B) {
			content := `{"text":"` + strings.Repeat("x", size.bytes) + `"}`
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := validateMessageContentJSON(content); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
