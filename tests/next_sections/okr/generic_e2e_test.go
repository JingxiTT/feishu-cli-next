// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr_test

import (
	"encoding/json"
	"github.com/larksuite/cli/tests/nextfixture"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// Frozen catalog operations retain their native requests and response fields.
func TestOKRGenericOperations(t *testing.T) {
	cases := []struct {
		command, method, path, body string
		args                        []string
	}{
		{"alignments delete", "DELETE", "/open-apis/okr/v2/alignments/201", "", []string{"--alignment-id", "201"}},
		{"alignments get", "GET", "/open-apis/okr/v2/alignments/201", "", []string{"--alignment-id", "201"}},
		{"categories list", "GET", "/open-apis/okr/v2/categories", "", []string{}},
		{"cycle.objectives create", "POST", "/open-apis/okr/v2/cycles/201/objectives", "{}", []string{"--cycle-id", "201"}},
		{"cycle.objectives list", "GET", "/open-apis/okr/v2/cycles/201/objectives", "", []string{"--cycle-id", "201"}},
		{"cycles list", "GET", "/open-apis/okr/v2/cycles", "", []string{"--user-id", "201"}},
		{"cycles objectives_position", "PUT", "/open-apis/okr/v2/cycles/201/objectives_position", "{\"objective_ids\":[\"201\"]}", []string{"--cycle-id", "201"}},
		{"cycles objectives_weight", "PUT", "/open-apis/okr/v2/cycles/201/objectives_weight", "{\"objective_weights\":[{\"objective_id\":\"201\",\"weight\":1}]}", []string{"--cycle-id", "201"}},
		{"indicators patch", "PATCH", "/open-apis/okr/v2/indicators/201", "{}", []string{"--indicator-id", "201"}},
		{"key_result.indicators list", "GET", "/open-apis/okr/v2/key_results/201/indicators", "", []string{"--key-result-id", "201"}},
		{"key_results delete", "DELETE", "/open-apis/okr/v2/key_results/201", "", []string{"--key-result-id", "201", "--yes"}},
		{"key_results get", "GET", "/open-apis/okr/v2/key_results/201", "", []string{"--key-result-id", "201"}},
		{"key_results patch", "PATCH", "/open-apis/okr/v2/key_results/201", "{}", []string{"--key-result-id", "201"}},
		{"objective.alignments create", "POST", "/open-apis/okr/v2/objectives/201/alignments", "{\"to_entity_type\":1,\"to_entity_id\":\"201\"}", []string{"--objective-id", "201"}},
		{"objective.alignments list", "GET", "/open-apis/okr/v2/objectives/201/alignments", "", []string{"--objective-id", "201"}},
		{"objective.indicators list", "GET", "/open-apis/okr/v2/objectives/201/indicators", "", []string{"--objective-id", "201"}},
		{"objective.key_results create", "POST", "/open-apis/okr/v2/objectives/201/key_results", "{}", []string{"--objective-id", "201"}},
		{"objective.key_results list", "GET", "/open-apis/okr/v2/objectives/201/key_results", "", []string{"--objective-id", "201"}},
		{"objectives delete", "DELETE", "/open-apis/okr/v2/objectives/201", "", []string{"--objective-id", "201", "--yes"}},
		{"objectives get", "GET", "/open-apis/okr/v2/objectives/201", "", []string{"--objective-id", "201"}},
		{"objectives key_results_position", "PUT", "/open-apis/okr/v2/objectives/201/key_results_position", "{\"key_result_ids\":[\"201\"]}", []string{"--objective-id", "201"}},
		{"objectives key_results_weight", "PUT", "/open-apis/okr/v2/objectives/201/key_results_weight", "{\"key_result_weights\":[{\"key_result_id\":\"201\",\"weight\":1}]}", []string{"--objective-id", "201"}},
		{"objectives patch", "PATCH", "/open-apis/okr/v2/objectives/201", "{}", []string{"--objective-id", "201"}},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("dispatch %s %s", r.Method, r.URL)
				}
				if tc.body != "" {
					b, _ := io.ReadAll(r.Body)
					var got, want any
					_ = json.Unmarshal(b, &got)
					_ = json.Unmarshal([]byte(tc.body), &want)
					if !reflect.DeepEqual(got, want) {
						t.Errorf("body %s want %s", b, tc.body)
					}
				}
				reply(w, map[string]any{"id": "9223372036854775807", "items": []any{map[string]any{"id": "201", "empty": "", "nullable": nil, "label": "中文"}}, "has_more": false})
			}))
			args := append([]string{"okr"}, strings.Fields(tc.command)...)
			args = append(args, tc.args...)
			if tc.body != "" {
				args = append(args, "--data", tc.body)
			}
			success(t, f.Run(append(args, "--dry-run")...), tc.path)
			if calls.Load() != 0 {
				t.Fatal("dry-run dispatched")
			}
			success(t, f.Run(args...), `"9223372036854775807"`)
			if calls.Load() != 1 {
				t.Fatalf("calls %d", calls.Load())
			}
		})
	}
}
