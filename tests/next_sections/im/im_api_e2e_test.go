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
	"github.com/tidwall/gjson"
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
		Response                           map[string]any
		ExpectedUnknown                    bool `json:"expected_unknown"`
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
				data := map[string]any{"fixture_marker": "preserved α", "items": []any{}, "has_more": false}
				if tc.Response != nil {
					// Endpoint-native evidence replaces the generic list-shaped stub.
					data = map[string]any{"fixture_marker": "preserved α"}
					for key, value := range tc.Response {
						data[key] = value
					}
				}
				respond(w, data)
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
			if tc.ExpectedUnknown || tc.Method != "GET" && (strings.Contains(tc.Name, "batch_update") || strings.Contains(tc.Name, "batch_add_item") || strings.Contains(tc.Name, "batch_remove_item")) {
				if r.ExitCode != 22 || !strings.Contains(r.Stderr, "unknown") {
					t.Fatalf("batch must stay unknown: %+v", r)
				}
			} else {
				success(t, r)
			}
			if calls.Load() != 1 || !strings.Contains(r.Stdout, "preserved α") {
				t.Fatalf("%+v calls %d", r, calls.Load())
			}
			for key, want := range tc.Response {
				if got := gjson.Get(r.Stdout, "data."+key); !got.Exists() || !reflect.DeepEqual(got.Value(), want) {
					t.Fatalf("native outcome %s lost: %+v", key, r)
				}
			}
			if tc.Response == nil {
				return
			}
			nullFields := map[string]any{}
			for key := range tc.Response {
				nullFields[key] = nil
			}
			for _, missing := range []struct {
				name string
				data any
			}{
				{"empty_data", map[string]any{}},
				{"null_data", nil},
				{"unrelated_array", map[string]any{"items": []any{}}},
				{"null_native_fields", nullFields},
			} {
				t.Run(missing.name, func(t *testing.T) {
					var attempts atomic.Int32
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
						attempts.Add(1)
						if req.Method != tc.Method || req.URL.Path != tc.Path {
							t.Errorf("unexpected %s %s", req.Method, req.URL)
						}
						respond(w, missing.data)
					}))
					r := f.Run(args...)
					if r.ExitCode != 22 || gjson.Get(r.Stdout, "ok").Bool() || !strings.Contains(r.Stderr, "unknown") || attempts.Load() != 1 {
						t.Fatalf("missing native evidence must stay unknown without replay: %+v calls %d", r, attempts.Load())
					}
				})
			}
		})
	}
}
