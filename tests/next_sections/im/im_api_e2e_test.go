// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package im_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

// Static synthetic cases independently pin every generated IM JSON endpoint.
// These are transport/contract evidence; the backend does not prove live access
// control, notification delivery, indexing, or server-side idempotency.
func TestGeneratedIMContracts(t *testing.T) {
	raw, err := os.ReadFile("api_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, Method, Path, Identity, Risk string
		Params                             map[string]any
		Body                               any
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.Method || r.URL.Path != tc.Path {
					t.Errorf("wire got %s %s want %s %s", r.Method, r.URL, tc.Method, tc.Path)
				}
				for key, value := range tc.Params {
					if strings.HasSuffix(key, "_id_type") || key == "receive_id_type" {
						if r.URL.Query().Get(key) != value {
							t.Errorf("query %s=%s want %v", key, r.URL.Query().Get(key), value)
						}
					}
				}
				body, _ := io.ReadAll(r.Body)
				if tc.Body != nil {
					var got any
					if err := json.Unmarshal(body, &got); err != nil {
						t.Errorf("body %s: %v", body, err)
					}
					if !reflect.DeepEqual(got, tc.Body) {
						t.Errorf("body %s want %#v", body, tc.Body)
					}
				}
				respond(w, map[string]any{"fixture_marker": "preserved α", "items": []any{}, "has_more": false})
			}))
			args := append([]string{"im"}, strings.Split(tc.Name, " ")...)
			args = append(args, "--as", tc.Identity)
			if len(tc.Params) > 0 {
				data, _ := json.Marshal(tc.Params)
				args = append(args, "--params", string(data))
			}
			if tc.Body != nil {
				data, _ := json.Marshal(tc.Body)
				args = append(args, "--data", string(data))
			}
			if tc.Risk == "destructive" || tc.Risk == "high-risk-write" {
				args = append(args, "--yes")
			}
			r := f.Run(args...)
			if tc.Method != "GET" && (strings.Contains(tc.Name, "batch_update") || strings.Contains(tc.Name, "batch_add_item") || strings.Contains(tc.Name, "batch_remove_item")) {
				if r.ExitCode != 22 || !strings.Contains(r.Stderr, "unknown") {
					t.Fatalf("batch must stay unknown: %+v", r)
				}
			} else {
				success(t, r)
			}
			if calls.Load() != 1 || !strings.Contains(r.Stdout, "preserved α") {
				t.Fatalf("%+v calls %d", r, calls.Load())
			}
		})
	}
}
