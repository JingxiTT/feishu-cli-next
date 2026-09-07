// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

// These 25 explicit fixtures were projected from the frozen Drive catalog.
// Each exercises the supported service command through actual transport and
// checks every supplied query/body field, including nested schema values.
func TestDriveCatalogMethods(t *testing.T) {
	raw, err := os.ReadFile("catalog_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Command             []string
		Method, Path, Risk  string
		Params, Body, Query map[string]any
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 25 {
		t.Fatalf("frozen Drive methods=%d, want 25", len(rows))
	}
	for _, row := range rows {
		t.Run(strings.Join(row.Command[1:], "/"), func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != row.Method || r.URL.Path != row.Path {
					t.Errorf("request %s %s, want %s %s", r.Method, r.URL.Path, row.Method, row.Path)
				}
				for k, v := range row.Query {
					if r.URL.Query().Get(k) != fmt.Sprint(v) {
						t.Errorf("query %s=%q, want %v", k, r.URL.Query().Get(k), v)
					}
				}
				if len(row.Body) > 0 {
					got := body(t, r)
					if !reflect.DeepEqual(got, row.Body) {
						t.Errorf("body changed: got %#v want %#v", got, row.Body)
					}
				}
				reply(w, map[string]any{"fixture_id": "retained", "unicode": "原文 Ω", "empty": "", "null": nil, "leading": "001", "has_more": true, "page_token": "continue"})
			}))
			params, _ := json.Marshal(row.Params)
			data, _ := json.Marshal(row.Body)
			args := append(append([]string{}, row.Command...), "--as", "user", "--params", string(params))
			if len(row.Body) > 0 {
				args = append(args, "--data", string(data))
			}
			if row.Risk == "high-risk-write" {
				args = append(args, "--yes")
			}
			got := f.Run(args...)
			successful(t, got)
			for _, want := range []string{`"fixture_id":"retained"`, `"unicode":"原文 Ω"`, `"empty":""`, `"null":null`, `"leading":"001"`, `"page_token":"continue"`} {
				if !strings.Contains(got.Stdout, want) {
					t.Fatalf("lost result %s: %s", want, got.Stdout)
				}
			}
			if calls.Load() != 1 {
				t.Fatalf("unbounded or missing request: %d", calls.Load())
			}
		})
	}
}
