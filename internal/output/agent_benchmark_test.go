// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output_test

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/output"
)

func BenchmarkAgentJSON(b *testing.B) {
	items := make([]map[string]interface{}, 1000)
	for i := range items {
		items[i] = map[string]interface{}{
			"id": i, "title": "Example", "unused": strings.Repeat("x", 1024),
			"owner": map[string]string{"name": "Example", "unused": "drop"},
		}
	}
	raw, err := json.Marshal(map[string]interface{}{"ok": true, "data": map[string]interface{}{"items": items, "has_more": false}})
	if err != nil {
		b.Fatal(err)
	}
	for _, fields := range []string{"", "title,owner.name"} {
		name := "Compact"
		if fields != "" {
			name = "Fields"
		}
		b.Run(name, func(b *testing.B) {
			opts := &output.AgentOptions{Fields: fields, ResolveMutation: func() (string, []string, bool) { return "", nil, false }}
			w := output.WithAgentOptions(io.Discard, opts)
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for i := 0; i < b.N; i++ {
				if err := output.WriteJSON(w, json.RawMessage(raw)); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
