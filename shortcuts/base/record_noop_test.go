// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
)

func noopMatrix(ids []string, rows ...[]any) map[string]any {
	return map[string]any{
		"fields":         []string{"Name", "Score", "Tags", "Clear"},
		"field_id_list":  []string{"fld_name", "fld_score", "fld_tags", "fld_clear"},
		"record_id_list": ids, "data": rows,
	}
}

func TestRecordSkipUnchangedZeroWrites(t *testing.T) {
	for _, batch := range []bool{false, true} {
		name := "upsert"
		shortcut := BaseRecordUpsert
		args := []string{"+record-upsert", "--record-id", "rec_1", "--json", `{"Name":"", "Score":9007199254740993, "Tags":[], "Clear":null}`}
		if batch {
			name, shortcut = "batch", BaseRecordBatchUpdate
			args = []string{"+record-batch-update", "--json", `{"update_records":{"rec_1":{"Name":"", "Score":9007199254740993, "Tags":[], "Clear":null}}}`}
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			factory, stdout, registry := newExecuteFactory(t)
			reads := &httpmock.Stub{Method: "POST", URL: "/records/batch_get", Body: map[string]any{"code": 0, "data": noopMatrix([]string{"rec_1"}, []any{"", json.Number("9007199254740993"), []any{}, nil})}}
			registry.Register(reads)
			writes := 0
			registry.Register(&httpmock.Stub{URL: "/records", Optional: true, Reusable: true, Body: map[string]any{"code": 0, "data": map[string]any{}}, OnMatch: func(*http.Request) { writes++ }})
			args = append(args, "--base-token", "app_x", "--table-id", "tbl_x", "--skip-unchanged")
			if err := runShortcut(t, shortcut, args, factory, stdout); err != nil {
				t.Fatal(err)
			}
			if writes != 0 || len(reads.CapturedBodies) != 1 {
				t.Fatalf("reads=%d writes=%d", len(reads.CapturedBodies), writes)
			}
			if !strings.Contains(stdout.String(), `"unchanged": true`) || !strings.Contains(stdout.String(), `"updated": false`) {
				t.Fatalf("no-op output=%s", stdout.String())
			}
		})
	}
}

func TestRecordSkipUnchangedChangedRowsKeepExactSubmittedMaps(t *testing.T) {
	for _, batch := range []bool{false, true} {
		name, shortcut := "upsert", BaseRecordUpsert
		fields := `{"fld_name":"Bob","Score":9007199254740993,"Tags":[],"Clear":null}`
		args := []string{"+record-upsert", "--record-id", "rec_1", "--json", fields}
		wantBody := fields
		method, path := "PATCH", "/records/rec_1"
		ids := []string{"rec_1"}
		rows := [][]any{{"Alice", json.Number("9007199254740993"), []any{}, nil}}
		if batch {
			name, shortcut = "batch", BaseRecordBatchUpdate
			args = []string{"+record-batch-update", "--json", `{"update_records":{"rec_1":` + fields + `,"rec_2":{"Name":"Keep"}}}`}
			wantBody = `{"update_records":{"rec_1":` + fields + `}}`
			method, path = "POST", "/records/batch_update"
			ids = append(ids, "rec_2")
			rows = append(rows, []any{"Keep", nil, nil, nil})
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			factory, stdout, registry := newExecuteFactory(t)
			registry.Register(&httpmock.Stub{Method: "POST", URL: "/records/batch_get", Body: map[string]any{"code": 0, "data": noopMatrix(ids, rows...)}})
			write := &httpmock.Stub{Method: method, URL: path, Body: map[string]any{"code": 0, "data": map[string]any{}}}
			registry.Register(write)
			if err := runShortcut(t, shortcut, append(args, "--base-token", "app_x", "--table-id", "tbl_x", "--skip-unchanged"), factory, stdout); err != nil {
				t.Fatal(err)
			}
			var got, want any
			for _, item := range []struct {
				raw  string
				dest *any
			}{{string(write.CapturedBody), &got}, {wantBody, &want}} {
				decoder := json.NewDecoder(strings.NewReader(item.raw))
				decoder.UseNumber()
				if err := decoder.Decode(item.dest); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(got, want) || len(write.CapturedBodies) != 1 {
				t.Fatalf("body=%s want=%s writes=%d", write.CapturedBody, wantBody, len(write.CapturedBodies))
			}
		})
	}
}

func TestRecordComparisonPreservesJSONDistinctions(t *testing.T) {
	for _, tc := range []struct {
		name, submitted string
		current         any
		unchanged       bool
	}{
		{"null", `null`, nil, true},
		{"empty string", `""`, "", true},
		{"null is not empty", `null`, "", false},
		{"empty array is not null", `[]`, nil, false},
		{"empty object is not null", `{}`, nil, false},
		{"large number equal", `9007199254740993`, json.Number("9007199254740993"), true},
		{"large number differs", `9007199254740993`, json.Number("9007199254740992"), false},
		{"numeric representation differs", `1.0`, json.Number("1"), false},
		{"array order differs", `["b","a"]`, []any{"a", "b"}, false},
		{"object order irrelevant", `{"b":null,"a":2}`, map[string]any{"a": json.Number("2"), "b": nil}, true},
		{"omitted nested key", `{"a":2}`, map[string]any{"a": json.Number("2"), "b": nil}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var submitted any
			decoder := json.NewDecoder(strings.NewReader(tc.submitted))
			decoder.UseNumber()
			if err := decoder.Decode(&submitted); err != nil {
				t.Fatal(err)
			}
			updates := map[string]map[string]any{"rec_1": {"Name": submitted}}
			got, err := compareRecordUpdates(noopMatrix([]string{"rec_1"}, []any{tc.current, "not submitted", nil, nil}), updates)
			if err != nil || (len(got) == 1) != tc.unchanged {
				t.Fatalf("unchanged=%v err=%v", got, err)
			}
		})
	}
}

func TestRecordSkipUnchangedReadFailuresNeverWrite(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func(map[string]any)
		code     int
		category errs.Category
	}{
		{"missing row", func(m map[string]any) { m["record_id_list"] = []string{}; m["data"] = [][]any{} }, 0, errs.CategoryInternal},
		{"short row", func(m map[string]any) { m["data"] = [][]any{{"Alice"}} }, 0, errs.CategoryInternal},
		{"missing metadata", func(m map[string]any) { delete(m, "field_id_list") }, 0, errs.CategoryInternal},
		{"not found", func(m map[string]any) { m["record_not_found"] = []string{"rec_1"} }, 0, errs.CategoryInternal},
		{"ignored fields", func(m map[string]any) { m["ignored_fields"] = []any{map[string]any{"id": "fld_name"}} }, 0, errs.CategoryInternal},
		{"pagination", func(m map[string]any) { m["has_more"] = true }, 0, errs.CategoryInternal},
		{"duplicate row", func(m map[string]any) {
			m["record_id_list"] = []string{"rec_1", "rec_1"}
			m["data"] = [][]any{{"Alice", nil, nil, nil}, {"Alice", nil, nil, nil}}
		}, 0, errs.CategoryInternal},
		{"ambiguous name", func(m map[string]any) { m["fields"] = []string{"Name", "Name", "Tags", "Clear"} }, 0, errs.CategoryValidation},
		{"name ID collision", func(m map[string]any) { m["field_id_list"] = []string{"fld_name", "Name", "fld_tags", "fld_clear"} }, 0, errs.CategoryValidation},
		{"scope failure", func(map[string]any) {}, 99991679, errs.CategoryAuthorization},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			factory, stdout, registry := newExecuteFactory(t)
			matrix := noopMatrix([]string{"rec_1"}, []any{"Alice", nil, nil, nil})
			tc.change(matrix)
			registry.Register(&httpmock.Stub{Method: "POST", URL: "/records/batch_get", Body: map[string]any{"code": tc.code, "msg": "missing scope base:record:read", "data": matrix}})
			writes := 0
			registry.Register(&httpmock.Stub{URL: "/records", Optional: true, Reusable: true, Body: map[string]any{"code": 0, "data": map[string]any{}}, OnMatch: func(*http.Request) { writes++ }})
			err := runShortcut(t, BaseRecordUpsert, []string{"+record-upsert", "--base-token", "app_x", "--table-id", "tbl_x", "--record-id", "rec_1", "--json", `{"Name":"Alice"}`, "--skip-unchanged"}, factory, stdout)
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Category != tc.category || writes != 0 {
				t.Fatalf("err=%v problem=%#v writes=%d", err, problem, writes)
			}
		})
	}
}

func TestRecordSkipUnchangedCreateRejectedBeforeAPI(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	factory, stdout, _ := newExecuteFactory(t)
	err := runShortcut(t, BaseRecordUpsert, []string{"+record-upsert", "--base-token", "app_x", "--table-id", "tbl_x", "--json", `{"Name":"Alice"}`, "--skip-unchanged"}, factory, stdout)
	assertInvalidArgumentValidation(t, err, "--skip-unchanged", []string{"--skip-unchanged", "--record-id"}, "cannot be used")
}

func TestRecordSkipUnchangedKeepsLegacyScopes(t *testing.T) {
	for _, shortcut := range []common.Shortcut{BaseRecordUpsert, BaseRecordBatchUpdate} {
		for _, scope := range shortcut.Scopes {
			if scope == "base:record:read" {
				t.Fatal("optional comparison read must not become a legacy scope prerequisite")
			}
		}
		if !strings.Contains(strings.Join(shortcut.Tips, " "), "base:record:read") {
			t.Fatal("comparison read scope absent from guidance")
		}
	}
}

func TestRecordSkipUnchangedRequiresExplicitFlag(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	factory, stdout, registry := newExecuteFactory(t)
	write := &httpmock.Stub{Method: "PATCH", URL: "/records/rec_1", Body: map[string]any{"code": 0, "data": map[string]any{}}}
	registry.Register(write)
	reads := 0
	registry.Register(&httpmock.Stub{Method: "POST", URL: "/records/batch_get", Optional: true, OnMatch: func(*http.Request) { reads++ }})
	if err := runShortcut(t, BaseRecordUpsert, []string{"+record-upsert", "--base-token", "app_x", "--table-id", "tbl_x", "--record-id", "rec_1", "--json", `{"Name":"Alice"}`}, factory, stdout); err != nil {
		t.Fatal(err)
	}
	if reads != 0 || len(write.CapturedBodies) != 1 {
		t.Fatalf("reads=%d writes=%d", reads, len(write.CapturedBodies))
	}
}

func TestRecordComparisonRejectsDuplicateSubmittedAliases(t *testing.T) {
	_, err := compareRecordUpdates(noopMatrix([]string{"rec_1"}, []any{"Alice", nil, nil, nil}), map[string]map[string]any{"rec_1": {"Name": "Alice", "fld_name": "Alice"}})
	assertInvalidArgumentValidation(t, err, "--json", []string{"--json"}, "same field")
}

func TestRecordSkipUnchangedBatchMissingRowBlocksChangedRows(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	factory, stdout, registry := newExecuteFactory(t)
	registry.Register(&httpmock.Stub{Method: "POST", URL: "/records/batch_get", Body: map[string]any{"code": 0, "data": noopMatrix([]string{"rec_1"}, []any{"Alice", nil, nil, nil})}})
	writes := 0
	registry.Register(&httpmock.Stub{Method: "POST", URL: "/records/batch_update", Optional: true, OnMatch: func(*http.Request) { writes++ }})
	err := runShortcut(t, BaseRecordBatchUpdate, []string{"+record-batch-update", "--base-token", "app_x", "--table-id", "tbl_x", "--skip-unchanged", "--json", `{"update_records":{"rec_1":{"Name":"Bob"},"rec_2":{"Name":"Missing"}}}`}, factory, stdout)
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Subtype != errs.SubtypeInvalidResponse || writes != 0 {
		t.Fatalf("err=%v writes=%d", err, writes)
	}
}
