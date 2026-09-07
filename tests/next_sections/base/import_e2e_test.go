// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func saveImportSource(t *testing.T, f *nextfixture.Fixture, count int, change func(map[string]interface{})) {
	t.Helper()
	rows := make([]interface{}, count)
	for i := range rows {
		rows[i] = []interface{}{map[string]interface{}{"value": fmt.Sprintf("行%05d", i)}, map[string]interface{}{"value": json.Number("9007199254740991")}, map[string]interface{}{"value": nil}, map[string]interface{}{}, map[string]interface{}{"value": ""}, map[string]interface{}{"value": i%2 == 0}}
	}
	artifact := map[string]interface{}{
		"schema":  "feishu-cli-next/result/v1",
		"binding": map[string]interface{}{"environment": "feishu", "endpoint": "https://open.feishu.cn", "app_id": "synthetic-foundation-app", "identity": "user", "user_id": "ou_foundation_fixture", "user_verified": true, "tenant_key": "foundation-fixture-tenant", "tenant_verified": true},
		"result":  map[string]interface{}{"ok": true, "identity": "user", "data": map[string]interface{}{"schema": "feishu-cli-next/sheets-data/v1", "spreadsheet_token": "sheet_synthetic", "sheet_id": "sheet_one", "range": fmt.Sprintf("A1:F%d", count), "complete": true, "count": count, "cells": rows}},
	}
	if change != nil {
		change(artifact)
	}
	raw, err := json.Marshal(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.WorkDir, "source.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	mapping := `[{"column":0,"field_id":"fld_text","type":"text"},{"column":1,"field_id":"fld_number","type":"number"},{"column":2,"field_id":"fld_null","type":"text"},{"column":3,"field_id":"fld_omitted","type":"text"},{"column":4,"field_id":"fld_empty","type":"text"},{"column":5,"field_id":"fld_checkbox","type":"checkbox"}]`
	if err := os.WriteFile(filepath.Join(f.WorkDir, "mapping.json"), []byte(mapping), 0600); err != nil {
		t.Fatal(err)
	}
}

func importArguments() []string {
	return []string{"base", "+record-import", "--base-token", "base_synthetic", "--table-id", "tbl_synthetic", "--artifact", "@source.json", "--mapping", "@mapping.json", "--output", "import-report.json", "--as", "user"}
}

func importFieldResponse(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != "GET" || !strings.Contains(r.URL.Path, "/fields/") {
		return false
	}
	id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	kind := "text"
	if id == "fld_number" {
		kind = "number"
	}
	if id == "fld_checkbox" {
		kind = "checkbox"
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"id": id, "type": kind}})
	return true
}

func TestRecordImportTenThousandRowsPreservesTypedValuesOutsideStdout(t *testing.T) {
	var writes atomic.Int32
	var observed atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if importFieldResponse(w, r) {
			return
		}
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/records/batch_create") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", 500)
			return
		}
		writes.Add(1)
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		var body struct {
			Records []map[string]interface{} `json:"create_records"`
		}
		if err := decoder.Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body.Records) != 200 {
			t.Errorf("batch len=%d", len(body.Records))
		}
		ids := make([]string, len(body.Records))
		for i, row := range body.Records {
			index := int(observed.Add(1)) - 1
			if row["fld_text"] != fmt.Sprintf("行%05d", index) || row["fld_number"] != json.Number("9007199254740991") || row["fld_empty"] != "" || row["fld_checkbox"] != (index%2 == 0) {
				t.Errorf("corrupted row %d: %#v", index, row)
			}
			if v, ok := row["fld_null"]; !ok || v != nil {
				t.Errorf("null changed at %d", index)
			}
			if _, ok := row["fld_omitted"]; ok {
				t.Errorf("omission became write at %d", index)
			}
			ids[i] = fmt.Sprintf("rec_%05d", index)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"record_id_list": ids}})
	}))
	saveImportSource(t, f, 10000, nil)
	result := f.Run(importArguments()...)
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d stdout=%s stderr=%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	if writes.Load() != 50 || observed.Load() != 10000 {
		t.Fatalf("writes=%d rows=%d", writes.Load(), observed.Load())
	}
	if len(result.Stdout) > 3000 || strings.Contains(result.Stdout, "rec_09999") {
		t.Fatalf("row IDs spilled to stdout (%d bytes)", len(result.Stdout))
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "import-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Status  string   `json:"status"`
		Created int      `json:"created"`
		IDs     []string `json:"record_ids"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "complete" || report.Created != 10000 || len(report.IDs) != 10000 || report.IDs[9999] != "rec_09999" {
		t.Fatalf("incomplete report: %+v", report)
	}
}

func TestRecordImportRejectsUnsafeInputsBeforeMutation(t *testing.T) {
	for _, kind := range []string{"tenant", "incomplete", "formula", "collision", "field-type", "precision", "default"} {
		t.Run(kind, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					if kind == "default" && strings.HasSuffix(r.URL.Path, "/fields/fld_omitted") {
						_, _ = io.WriteString(w, `{"code":0,"data":{"id":"fld_omitted","type":"text","default_value":"manufactured"}}`)
						return
					}
					if kind == "field-type" {
						_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"id": "fld_text", "type": "formula"}})
						return
					}
					if importFieldResponse(w, r) {
						return
					}
				}
				writes.Add(1)
				http.Error(w, "unexpected write", 500)
			}))
			saveImportSource(t, f, 1, func(a map[string]interface{}) {
				data := a["result"].(map[string]interface{})["data"].(map[string]interface{})
				switch kind {
				case "tenant":
					a["binding"].(map[string]interface{})["tenant_key"] = "other-tenant"
				case "incomplete":
					data["complete"] = false
				case "precision":
					data["cells"].([]interface{})[0].([]interface{})[1].(map[string]interface{})["value"] = json.Number("9007199254740993")
				case "formula":
					data["cells"].([]interface{})[0].([]interface{})[0].(map[string]interface{})["formula"] = "=A1"
				}
			})
			if kind == "collision" {
				if err := os.WriteFile(filepath.Join(f.WorkDir, "import-report.json"), []byte("unrelated"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			result := f.Run(importArguments()...)
			if result.ExitCode == 0 || writes.Load() != 0 {
				t.Fatalf("unsafe accepted: %+v writes=%d", result, writes.Load())
			}
			if result.Stdout != "" {
				t.Fatalf("validation wrote stdout %s", result.Stdout)
			}
		})
	}
}

func TestRecordImportStopsAfterUnknownBatchAndWritesRecoveryReport(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if importFieldResponse(w, r) {
			return
		}
		count := writes.Add(1)
		if count == 1 {
			ids := make([]string, 200)
			for i := range ids {
				ids[i] = fmt.Sprintf("rec_%d", i)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"record_id_list": ids}})
			return
		}
		_, _ = io.WriteString(w, `{"code":0,"data":{"record_id_list":["rec_maybe"],"ignored_fields":[{"id":"fld_text"}]}}`)
	}))
	saveImportSource(t, f, 401, nil)
	result := f.Run(importArguments()...)
	if result.ExitCode != 22 || writes.Load() != 2 {
		t.Fatalf("exit=%d writes=%d stdout=%s stderr=%s", result.ExitCode, writes.Load(), result.Stdout, result.Stderr)
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "import-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"created":200`) || !strings.Contains(string(raw), `"first_unconfirmed_row":200`) || !strings.Contains(string(raw), "rec_maybe") {
		t.Fatalf("lost recovery evidence %s", raw)
	}
}

func TestRecordImportReportRacePreservesObservedEffectsOnStdout(t *testing.T) {
	var f *nextfixture.Fixture
	var writes atomic.Int32
	f = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if importFieldResponse(w, r) {
			return
		}
		writes.Add(1)
		if err := os.WriteFile(filepath.Join(f.WorkDir, "import-report.json"), []byte("concurrent unrelated file"), 0600); err != nil {
			t.Error(err)
		}
		_, _ = io.WriteString(w, `{"code":0,"data":{"record_id_list":["rec_created"]}}`)
	}))
	saveImportSource(t, f, 1, nil)
	result := f.Run(importArguments()...)
	if result.ExitCode == 0 || writes.Load() != 1 {
		t.Fatalf("exit=%d writes=%d stdout=%s stderr=%s", result.ExitCode, writes.Load(), result.Stdout, result.Stderr)
	}
	if !strings.Contains(result.Stdout, "rec_created") || !strings.Contains(result.Stdout, `"created":1`) || !strings.Contains(result.Stderr, "effects may have occurred") {
		t.Fatalf("lost applied evidence: %+v", result)
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "import-report.json"))
	if err != nil || string(raw) != "concurrent unrelated file" {
		t.Fatalf("overwrote concurrent file: %q %v", raw, err)
	}
}

func TestRecordImportDryRunShowsBoundedAppendPlanWithoutRequests(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	saveImportSource(t, f, 401, nil)
	result := f.Run(append(importArguments(), "--dry-run")...)
	if result.ExitCode != 0 || !strings.Contains(result.Stdout, `"batches":3`) || !strings.Contains(result.Stdout, `"records_count":401`) || !strings.Contains(result.Stdout, "batch_create") {
		t.Fatalf("dry-run %+v", result)
	}
	if _, err := os.Stat(filepath.Join(f.WorkDir, "import-report.json")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote report: %v", err)
	}
}

func TestRecordUpsertIgnoredFieldsKeepsPartialEvidence(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		_, _ = io.WriteString(w, `{"code":0,"data":{"record_id":"rec_partial","ignored_fields":[{"id":"fld_formula","name":"Formula","reason":"read_only"}]}}`)
	}))
	result := f.Run("base", "+record-upsert", "--base-token", "base_synthetic", "--table-id", "tbl_synthetic", "--record-id", "rec_partial", "--json", `{"Text":"changed","Formula":"ignored"}`, "--as", "user")
	if result.ExitCode != 22 || writes.Load() != 1 || !strings.Contains(result.Stdout, "ignored_fields") || !strings.Contains(result.Stdout, `"ok":false`) {
		t.Fatalf("partial write lost: %+v writes=%d", result, writes.Load())
	}
}

func TestRecordImportBatchAndReportFailureRetainBothErrors(t *testing.T) {
	var f *nextfixture.Fixture
	f = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if importFieldResponse(w, r) {
			return
		}
		if err := os.WriteFile(filepath.Join(f.WorkDir, "import-report.json"), []byte("concurrent"), 0600); err != nil {
			t.Error(err)
		}
		w.Header().Set("x-tt-logid", "synthetic-log-id")
		_, _ = io.WriteString(w, `{"code":1254291,"msg":"synthetic write conflict"}`)
	}))
	saveImportSource(t, f, 1, nil)
	result := f.Run(importArguments()...)
	if result.ExitCode != 22 || !strings.Contains(result.Stdout, `"ok":false`) || !strings.Contains(result.Stdout, `"code":1254291`) || !strings.Contains(result.Stdout, "synthetic write conflict") || !strings.Contains(result.Stderr, "report could not be saved") {
		t.Fatalf("lost one failure: %+v", result)
	}
}

func TestRecordImportStopsOnIDsRepeatedAcrossBatches(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if importFieldResponse(w, r) {
			return
		}
		count := writes.Add(1)
		ids := make([]string, 200)
		for i := range ids {
			ids[i] = fmt.Sprintf("rec_%d", (int(count)-1)*200+i)
		}
		if count == 2 {
			ids[199] = "rec_0"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"record_id_list": ids}})
	}))
	saveImportSource(t, f, 401, nil)
	result := f.Run(importArguments()...)
	if result.ExitCode != 22 || writes.Load() != 2 || !strings.Contains(result.Stderr, `"outcome":"unknown"`) || !strings.Contains(result.Stdout, `"ok":false`) {
		t.Fatalf("exit=%d writes=%d stdout=%s stderr=%s", result.ExitCode, writes.Load(), result.Stdout, result.Stderr)
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "import-report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Status    string          `json:"status"`
		Created   int             `json:"created"`
		Completed int             `json:"completed_batches"`
		Next      int             `json:"first_unconfirmed_row"`
		IDs       []string        `json:"record_ids"`
		Response  json.RawMessage `json:"unresolved_response"`
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "unknown" || report.Created != 200 || report.Completed != 1 || report.Next != 200 || len(report.IDs) != 200 || !strings.Contains(string(report.Response), "rec_398") {
		t.Fatalf("lost confirmed prefix or unresolved evidence: %s", raw)
	}
	for i, id := range report.IDs {
		if id != fmt.Sprintf("rec_%d", i) {
			t.Fatalf("wrong confirmed ID %d: %s", i, id)
		}
	}
	var envelope struct {
		Mutation struct {
			Outcome string `json:"outcome"`
		} `json:"mutation"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil || envelope.Mutation.Outcome != "unknown" {
		t.Fatalf("stdout overstates certainty: %s", result.Stdout)
	}
	receipts, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	outcomes := map[string]int{}
	for _, path := range receipts {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var receipt struct {
			Outcome string `json:"outcome"`
		}
		if err := json.Unmarshal(data, &receipt); err != nil {
			t.Fatal(err)
		}
		outcomes[receipt.Outcome]++
	}
	if len(receipts) != 2 || outcomes["confirmed_applied"] != 1 || outcomes["unknown"] != 1 {
		t.Fatalf("durable receipts overstate effects: %v", outcomes)
	}
}
