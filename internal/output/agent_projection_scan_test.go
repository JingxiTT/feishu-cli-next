// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestUnsafeProjectionScalarsDoNotAllocate(t *testing.T) {
	for _, scalar := range []string{`"mail subject"`, `900719925474099312345`, " true\n", "false", "null", ""} {
		t.Run(scalar, func(t *testing.T) {
			raw := json.RawMessage(scalar)
			if allocations := testing.AllocsPerRun(100, func() {
				if unsafeProjection(raw) {
					t.Fatal("scalar cannot contain failure metadata")
				}
			}); allocations != 0 {
				t.Fatalf("scalar failure scan allocated %g times", allocations)
			}
		})
	}
}

func TestUnsafeProjectionUpstreamFailureMarkers(t *testing.T) {
	for _, field := range []string{"failed_thread_ids", "error_logs"} {
		for _, value := range []string{"null", "[]", "{}", "false"} {
			if unsafeProjection(json.RawMessage(`{"` + field + `":` + value + `}`)) {
				t.Fatalf("empty %s=%s marked unsafe", field, value)
			}
		}
		for _, value := range []string{`[{"reason":"failure"}]`, `{"detail":"failure"}`, `"failure"`, "true"} {
			if !unsafeProjection(json.RawMessage(` {"nested":[{"` + field + `":` + value + `}]} `)) {
				t.Fatalf("nested %s=%s not marked unsafe", field, value)
			}
		}
	}
}

func BenchmarkAgentProjectionFailureScan(b *testing.B) {
	for _, count := range []int{20, 200} {
		b.Run(fmt.Sprintf("messages_%d", count), func(b *testing.B) {
			messages := make([]map[string]any, count)
			for i := range messages {
				messages[i] = map[string]any{"message_id": fmt.Sprint(i), "subject": "subject", "body": "message text", "timestamp": json.Number("9007199254740993"), "unread": true, "labels": []string{"INBOX", "IMPORTANT"}}
			}
			raw, err := json.Marshal(map[string]any{"ok": true, "data": map[string]any{"messages": messages, "has_more": false}})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if unsafeProjection(raw) {
					b.Fatal("successful fixture marked unsafe")
				}
			}
		})
	}
}
