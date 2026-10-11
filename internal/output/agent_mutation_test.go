// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestAgentMutationEnvelopeDetection(t *testing.T) {
	for _, raw := range []string{
		`{"ok":true,"data":{"title":"你好","number":900719925474099312345}}`,
		`{"data":{"ok":true},"ok":null}`,
		`{"ok":false,"ok":"last"}`,
		`{"ok":[],"data":null}`,
		`{"ok":{},"data":null}`,
		`{"\u006f\u006b":true,"data":null}`,
		`{"data":{"ok":true},"OK":true}`,
		`{"items":[{"ok":true}]}`,
		`[{"ok":true}]`,
		`null`,
	} {
		t.Run(raw, func(t *testing.T) {
			var original map[string]json.RawMessage
			_ = json.Unmarshal([]byte(raw), &original)
			_, envelope := original["ok"]
			calls := 0
			opts := &AgentOptions{ResolveMutation: func() (string, []string, bool) {
				calls++
				return "confirmed_applied", []string{"attempt_fixture"}, false
			}}
			got, err := opts.annotateMutation(json.RawMessage(raw))
			if err != nil {
				t.Fatal(err)
			}
			if !envelope {
				if calls != 0 || string(got) != raw {
					t.Fatalf("bare result resolved a mutation or changed: calls=%d result=%s", calls, got)
				}
				return
			}
			var annotated map[string]json.RawMessage
			if err := json.Unmarshal(got, &annotated); err != nil {
				t.Fatal(err)
			}
			if calls != 1 || !bytes.Equal(annotated["ok"], original["ok"]) || !bytes.Equal(annotated["data"], original["data"]) || !bytes.Contains(annotated["mutation"], []byte("attempt_fixture")) {
				t.Fatalf("mutation annotation changed envelope values: calls=%d result=%s", calls, got)
			}
		})
	}
}

func TestAgentReadMutationAnnotationAvoidsPayloadDecoding(t *testing.T) {
	raw := json.RawMessage(`{"ok":true,"data":{"content":"` + strings.Repeat("x", 1024*1024) + `"}}`)
	opts := &AgentOptions{ResolveMutation: func() (string, []string, bool) { return "", nil, false }}
	allocs := testing.AllocsPerRun(10, func() {
		got, err := opts.annotateMutation(raw)
		if err != nil || !bytes.Equal(got, raw) {
			t.Fatalf("read result changed: %v", err)
		}
	})
	// Only the envelope marker may allocate; the business payload stays untouched.
	if allocs > 2 {
		t.Fatalf("read annotation allocated %.0f objects; expected at most 2", allocs)
	}
}
