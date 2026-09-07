// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheetsbase

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const sheetToken = "sht_transfer_fixture"
const sheetID = "shtransfer"
const baseToken = "base_transfer_fixture"
const tableID = "tbl_transfer_fixture"
const sheetReadPath = "/open-apis/sheet_ai/v2/spreadsheets/" + sheetToken + "/tools/invoke_read"
const baseTablePath = "/open-apis/base/v3/bases/" + baseToken + "/tables/" + tableID
const sourceFile = "sheet-source.json"
const reportFile = "base-import.json"

var fieldIDs = []string{"fld_key", "fld_number", "fld_bool", "fld_null", "fld_omitted", "fld_empty", "fld_code", "fld_literal"}
var fieldTypes = []string{"text", "number", "checkbox", "text", "text", "text", "text", "text"}

type cell map[string]json.RawMessage
type record map[string]json.RawMessage
type storedRecord struct {
	ID     string `json:"id"`
	Fields record `json:"fields"`
}
type sourceState struct {
	Token      string   `json:"token"`
	SheetID    string   `json:"sheet_id"`
	Revision   int      `json:"revision"`
	Visibility string   `json:"visibility"`
	Cells      [][]cell `json:"cells"`
}

type transferBackend struct {
	t            *testing.T
	mu           sync.Mutex
	source       sourceState
	destination  []storedRecord
	ranges       []string
	fields       []string
	writes       int
	sourceWrites int
	mode         string
	truncatePage int
}

func raw(value any) json.RawMessage  { b, _ := json.Marshal(value); return b }
func fingerprint(value any) [32]byte { b, _ := json.Marshal(value); return sha256.Sum256(b) }

func newTransfer(t *testing.T, rows int, mode string) (*nextfixture.Fixture, *transferBackend) {
	t.Helper()
	b := &transferBackend{t: t, mode: mode, source: sourceState{Token: sheetToken, SheetID: sheetID, Revision: 23, Visibility: "self_only"}, destination: []storedRecord{{ID: "rec_unrelated", Fields: record{"fld_key": raw("unrelated fixture record"), "fld_number": raw(7)}}}}
	numbers := []json.RawMessage{raw(0), json.RawMessage("9007199254740991"), json.RawMessage("-3.5"), json.RawMessage("0.125")}
	for i := 0; i < rows; i++ {
		b.source.Cells = append(b.source.Cells, []cell{
			{"value": raw(fmt.Sprintf("源行%05d · 你好\n✓", i))},
			{"value": numbers[i%len(numbers)]},
			{"value": raw(i%2 == 1)},
			{"value": json.RawMessage("null")},
			{},
			{"value": raw("")},
			{"value": raw(fmt.Sprintf("%08d", i))},
			{"value": raw("=literal text, not a formula")},
		})
	}
	f := nextfixture.NewFixture(t, b)
	mapping := make([]map[string]any, len(fieldIDs))
	for i, id := range fieldIDs {
		mapping[i] = map[string]any{"column": i, "field_id": id, "type": fieldTypes[i]}
	}
	data, _ := json.Marshal(mapping)
	if err := os.WriteFile(filepath.Join(f.WorkDir, "mapping.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return f, b
}

func (b *transferBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "POST" && r.URL.Path == sheetReadPath {
		b.readCells(w, r)
		return
	}
	if r.Method == "GET" && strings.HasPrefix(r.URL.Path, baseTablePath+"/fields/") {
		id := strings.TrimPrefix(r.URL.Path, baseTablePath+"/fields/")
		for i, field := range fieldIDs {
			if id == field {
				b.fields = append(b.fields, id)
				reply(w, map[string]any{"id": id, "type": fieldTypes[i]})
				return
			}
		}
	}
	if r.Method == "POST" && r.URL.Path == baseTablePath+"/records/batch_create" {
		b.createRecords(w, r)
		return
	}
	if r.Method != "GET" {
		b.sourceWrites++
	}
	b.t.Errorf("unexpected cross-domain backend request: %s %s", r.Method, r.URL.Path)
	http.Error(w, "unexpected fixture request", 500)
}

func reply(w http.ResponseWriter, data any) {
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func (b *transferBackend) readCells(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Tool  string `json:"tool_name"`
		Input string `json:"input"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		b.t.Error(err)
		return
	}
	var in struct {
		Excel  string   `json:"excel_id"`
		Sheet  string   `json:"sheet_id"`
		Ranges []string `json:"ranges"`
		Limit  int      `json:"cell_limit"`
		Render string   `json:"value_render_option"`
	}
	if err := json.Unmarshal([]byte(body.Input), &in); err != nil {
		b.t.Error(err)
		return
	}
	if body.Tool != "get_cell_ranges" || in.Excel != sheetToken || in.Sheet != sheetID || len(in.Ranges) != 1 || in.Render != "raw_value" {
		b.t.Errorf("wrong source read identity/operation: %+v %+v", body, in)
		return
	}
	region := in.Ranges[0]
	var first, last int
	if _, err := fmt.Sscanf(region, "A%d:H%d", &first, &last); err != nil || first < 1 || last < first || last > len(b.source.Cells) || (last-first+1)*8 > in.Limit || in.Limit > 5000 {
		b.t.Errorf("invalid page %q limit=%d", region, in.Limit)
		return
	}
	b.ranges = append(b.ranges, region)
	indices := make([]int, last-first+1)
	for i := range indices {
		indices[i] = first + i
	}
	truncated := b.truncatePage > 0 && len(b.ranges) == b.truncatePage
	output := map[string]any{"has_more": truncated, "ranges": []any{map[string]any{"range": region, "actual_range": region, "row_indices": indices, "col_indices": []string{"A", "B", "C", "D", "E", "F", "G", "H"}, "truncated": truncated, "cells": b.source.Cells[first-1 : last]}}}
	payload, _ := json.Marshal(output)
	reply(w, map[string]any{"output": string(payload)})
}

func (b *transferBackend) createRecords(w http.ResponseWriter, r *http.Request) {
	b.writes++
	var body struct {
		Records []record `json:"create_records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		b.t.Error(err)
		return
	}
	if len(body.Records) == 0 || len(body.Records) > 200 {
		b.t.Errorf("invalid destination batch size %d", len(body.Records))
		return
	}
	if len(b.fields) != len(fieldIDs) {
		b.t.Errorf("write occurred before all destination schema checks: %v", b.fields)
	}
	if b.writes == 2 && b.mode == "rejected" {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic permission rejection"}`)
		return
	}
	apply := len(body.Records)
	if b.writes == 2 && b.mode == "partial" {
		apply = 37
	}
	ids := make([]string, apply)
	for i := 0; i < apply; i++ {
		id := fmt.Sprintf("rec_transfer_%05d", len(b.destination)-1)
		b.destination = append(b.destination, storedRecord{ID: id, Fields: body.Records[i]})
		ids[i] = id
	}
	if b.writes == 2 && b.mode == "duplicate-ids" {
		for i := range ids {
			ids[i] = fmt.Sprintf("rec_transfer_%05d", i)
		}
	}
	if b.writes == 2 && b.mode == "lost" {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			b.t.Error(err)
			return
		}
		_ = conn.Close() // request applied; response is genuinely lost after dispatch
		return
	}
	if b.writes == 2 && b.mode == "partial" {
		reply(w, map[string]any{"record_id_list": ids, "ignored_fields": []any{map[string]any{"id": "fld_key", "reason": "synthetic partial batch"}}})
		return
	}
	reply(w, map[string]any{"record_id_list": ids})
}

func exportArgs(rows int) []string {
	return []string{"sheets", "+data-export", "--as", "user", "--spreadsheet-token", sheetToken, "--sheet-id", sheetID, "--range", fmt.Sprintf("A1:H%d", rows), "--save-result", sourceFile}
}
func importArgs() []string {
	return []string{"base", "+record-import", "--as", "user", "--base-token", baseToken, "--table-id", tableID, "--artifact", "@" + sourceFile, "--mapping", "@mapping.json", "--output", reportFile}
}

func resultObject(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("command failed exit=%d stdout=%s stderr=%s", r.ExitCode, r.Stdout, r.Stderr)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(r.Stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result["ok"] != true {
		t.Fatalf("unsuccessful command result: %s", r.Stdout)
	}
	return result
}

func exportArtifact(t *testing.T, f *nextfixture.Fixture, rows int) []byte {
	t.Helper()
	r := f.Run(exportArgs(rows)...)
	obj := resultObject(t, r)
	if strings.Contains(r.Stdout, `"cells"`) || strings.Contains(r.Stdout, "源行") {
		t.Fatal("source rows spilled to stdout instead of artifact")
	}
	ref := obj["artifact"].(map[string]any)
	if ref["count"] != float64(rows) || ref["complete"] != true {
		t.Fatalf("incorrect artifact reference: %+v", ref)
	}
	data, err := os.ReadFile(filepath.Join(f.WorkDir, sourceFile))
	if err != nil {
		t.Fatal(err)
	}
	var artifact struct {
		Schema  string `json:"schema"`
		Binding struct {
			Tenant bool `json:"tenant_verified"`
			User   bool `json:"user_verified"`
		} `json:"binding"`
		Result struct {
			Data struct {
				Schema   string   `json:"schema"`
				Token    string   `json:"spreadsheet_token"`
				Sheet    string   `json:"sheet_id"`
				Range    string   `json:"range"`
				Complete bool     `json:"complete"`
				Count    int      `json:"count"`
				Cells    [][]cell `json:"cells"`
			} `json:"data"`
		} `json:"result"`
	}
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Schema != "feishu-cli-next/result/v1" || artifact.Result.Data.Schema != "feishu-cli-next/sheets-data/v1" || artifact.Result.Data.Token != sheetToken || artifact.Result.Data.Sheet != sheetID || artifact.Result.Data.Range != fmt.Sprintf("A1:H%d", rows) || !artifact.Result.Data.Complete || artifact.Result.Data.Count != rows || len(artifact.Result.Data.Cells) != rows || !artifact.Binding.Tenant || !artifact.Binding.User {
		t.Fatal("incomplete/unbound native source artifact")
	}
	return data
}

type importReport struct {
	Status    string          `json:"status"`
	Requested int             `json:"requested"`
	Created   int             `json:"created"`
	Completed int             `json:"completed_batches"`
	NextRow   int             `json:"first_unconfirmed_row"`
	IDs       []string        `json:"record_ids"`
	Response  json.RawMessage `json:"unresolved_response"`
}

func readReport(t *testing.T, f *nextfixture.Fixture) importReport {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.WorkDir, reportFile))
	if err != nil {
		t.Fatal(err)
	}
	var report importReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return report
}

// Expected destination values derive independently from immutable source cells,
// never from CLI output or the import report's claimed counts.
func verifyStoredPrefix(t *testing.T, b *transferBackend, rows int) {
	t.Helper()
	if len(b.destination) != rows+1 {
		t.Fatalf("destination count=%d, want %d plus unrelated", len(b.destination), rows)
	}
	if b.destination[0].ID != "rec_unrelated" || string(b.destination[0].Fields["fld_key"]) != `"unrelated fixture record"` || string(b.destination[0].Fields["fld_number"]) != "7" {
		t.Fatal("unrelated destination record changed")
	}
	for i := 0; i < rows; i++ {
		got := b.destination[i+1]
		if got.ID != fmt.Sprintf("rec_transfer_%05d", i) {
			t.Fatalf("wrong native record identity at %d: %s", i, got.ID)
		}
		expected := record{}
		for col, id := range fieldIDs {
			if value, present := b.source.Cells[i][col]["value"]; present {
				expected[id] = value
			}
		}
		if !reflect.DeepEqual(expected, got.Fields) {
			t.Fatalf("destination typed values/presence differ at row %d", i)
		}
	}
}

func TestTenThousandRowsThroughSheetsAndBaseCLI(t *testing.T) {
	f, b := newTransfer(t, 10000, "")
	before := fingerprint(b.source)
	artifact := exportArtifact(t, f, 10000)
	obj := resultObject(t, f.Run(importArgs()...))
	if len(b.ranges) != 16 || b.writes != 50 || len(b.fields) != 8 || b.sourceWrites != 0 {
		t.Fatalf("unexpected source pages=%d field reads=%d batches=%d source writes=%d", len(b.ranges), len(b.fields), b.writes, b.sourceWrites)
	}
	next := 1
	for _, region := range b.ranges {
		var first, last int
		_, _ = fmt.Sscanf(region, "A%d:H%d", &first, &last)
		if first != next {
			t.Fatalf("source page gap/overlap at %s", region)
		}
		next = last + 1
	}
	if next != 10001 {
		t.Fatalf("incomplete source pagination ended at %d", next)
	}
	if fingerprint(b.source) != before {
		t.Fatal("source state changed")
	}
	verifyStoredPrefix(t, b, 10000)
	report := readReport(t, f)
	if report.Status != "complete" || report.Requested != 10000 || report.Created != 10000 || report.Completed != 50 || report.NextRow != 10000 || len(report.IDs) != 10000 {
		t.Fatalf("incomplete destination report counts: status=%s created=%d batches=%d IDs=%d", report.Status, report.Created, report.Completed, len(report.IDs))
	}
	for i, id := range report.IDs {
		if id != b.destination[i+1].ID {
			t.Fatalf("report ID mismatch at row %d", i)
		}
	}
	summary, _ := json.Marshal(obj)
	if strings.Contains(string(summary), "rec_transfer_") || strings.Contains(string(summary), "源行") {
		t.Fatal("destination row data spilled to stdout")
	}
	after, err := os.ReadFile(filepath.Join(f.WorkDir, sourceFile))
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(after) != sha256.Sum256(artifact) {
		t.Fatal("import modified the exported artifact")
	}
	t.Logf("verified 10,000 rows, 80,000 cells, %d source pages, %d schema reads, %d destination batches; source artifact bytes=%d", len(b.ranges), len(b.fields), b.writes, len(artifact))
}

func TestExportedUnsupportedCellsRejectBeforeDestinationWrites(t *testing.T) {
	for _, kind := range []string{"formula", "style", "precision"} {
		t.Run(kind, func(t *testing.T) {
			f, b := newTransfer(t, 401, "")
			last := b.source.Cells[400][1]
			switch kind {
			case "formula":
				last["formula"] = raw("=1+1")
			case "style":
				last["cell_styles"] = raw(map[string]string{"number_format": "yyyy-mm-dd hh:mm:ss"})
			case "precision":
				last["value"] = json.RawMessage("9007199254740993")
			}
			before := fingerprint(b.source)
			artifact := exportArtifact(t, f, 401)
			if kind == "precision" && !strings.Contains(string(artifact), "9007199254740993") {
				t.Fatal("export rounded the unsupported number before import validation")
			}
			result := f.Run(importArgs()...)
			if result.ExitCode != 2 || result.Stdout != "" || b.writes != 0 {
				t.Fatalf("unsupported source reached write: exit=%d writes=%d stderr=%s", result.ExitCode, b.writes, result.Stderr)
			}
			var failure struct {
				Error struct {
					Type    string `json:"type"`
					Subtype string `json:"subtype"`
					Param   string `json:"param"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(result.Stderr), &failure); err != nil || failure.Error.Type != "validation" || failure.Error.Subtype != "invalid_argument" || failure.Error.Param != "--artifact" {
				t.Fatalf("missing typed artifact validation: %s", result.Stderr)
			}
			if fingerprint(b.source) != before || b.sourceWrites != 0 {
				t.Fatal("source changed during rejected handoff")
			}
			verifyStoredPrefix(t, b, 0)
			if _, err := os.Stat(filepath.Join(f.WorkDir, reportFile)); !os.IsNotExist(err) {
				t.Fatal("rejected import wrote a success report")
			}
		})
	}
}

func TestPartialExportCannotBecomeAnImportArtifact(t *testing.T) {
	f, b := newTransfer(t, 2000, "")
	b.truncatePage = 2
	before := fingerprint(b.source)
	exported := f.Run(exportArgs(2000)...)
	if exported.ExitCode == 0 || exported.Stdout != "" || len(b.ranges) != 2 {
		t.Fatalf("partial export accepted: %+v pages=%d", exported, len(b.ranges))
	}
	if _, err := os.Stat(filepath.Join(f.WorkDir, sourceFile)); !os.IsNotExist(err) {
		t.Fatal("partial source artifact was committed")
	}
	imported := f.Run(importArgs()...)
	if imported.ExitCode == 0 || len(b.fields) != 0 || b.writes != 0 {
		t.Fatal("missing/partial export triggered destination operations")
	}
	if fingerprint(b.source) != before {
		t.Fatal("source changed")
	}
	verifyStoredPrefix(t, b, 0)
}

func TestHandoffStopsAtPartialRejectedOrLostBatch(t *testing.T) {
	for _, mode := range []string{"partial", "rejected", "lost", "duplicate-ids"} {
		t.Run(mode, func(t *testing.T) {
			f, b := newTransfer(t, 401, mode)
			before := fingerprint(b.source)
			exportArtifact(t, f, 401)
			result := f.Run(importArgs()...)
			wantExit, applied := 22, 237
			if mode == "rejected" {
				wantExit, applied = 21, 200
			}
			if mode == "lost" || mode == "duplicate-ids" {
				applied = 400
			}
			if result.ExitCode != wantExit || b.writes != 2 {
				t.Fatalf("unsafe failure/replay: mode=%s exit=%d writes=%d stdout=%s stderr=%s", mode, result.ExitCode, b.writes, result.Stdout, result.Stderr)
			}
			if fingerprint(b.source) != before || b.sourceWrites != 0 {
				t.Fatal("source changed after failed handoff")
			}
			verifyStoredPrefix(t, b, applied)
			report := readReport(t, f)
			if report.Status != "unknown" || report.Created != 200 || report.Completed != 1 || report.NextRow != 200 || len(report.IDs) != 200 {
				t.Fatalf("report overstated effects: status=%s created=%d batches=%d next=%d IDs=%d", report.Status, report.Created, report.Completed, report.NextRow, len(report.IDs))
			}
			for i, id := range report.IDs {
				if id != b.destination[i+1].ID {
					t.Fatalf("confirmed prefix ID mismatch at %d", i)
				}
			}
			if !strings.Contains(result.Stdout, `"ok":false`) || !strings.Contains(result.Stdout, `"first_unconfirmed_row":200`) {
				t.Fatalf("failed workflow claimed success: %s", result.Stdout)
			}
			if mode == "partial" && !strings.Contains(string(report.Response), "rec_transfer_00236") {
				t.Fatal("partial returned IDs were discarded")
			}
			wantOutcome := "unknown"
			if mode == "rejected" {
				wantOutcome = "partial"
			}
			if !strings.Contains(result.Stderr, `"outcome":"`+wantOutcome+`"`) {
				t.Fatalf("wrong receipt outcome %s: %s", wantOutcome, result.Stderr)
			}
			var envelope struct {
				Mutation struct {
					Outcome string `json:"outcome"`
				} `json:"mutation"`
			}
			if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil || envelope.Mutation.Outcome != wantOutcome {
				t.Fatalf("stdout overstates mutation certainty: %s", result.Stdout)
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
			secondOutcome := "unknown"
			if mode == "rejected" {
				secondOutcome = "confirmed_unapplied"
			}
			if len(receipts) != 2 || outcomes["confirmed_applied"] != 1 || outcomes[secondOutcome] != 1 {
				t.Fatalf("durable receipts overstate effects: %v", outcomes)
			}
			t.Log("independent destination rows=" + strconv.Itoa(applied) + "; report confirms only first 200; no replay or third batch")
		})
	}
}
