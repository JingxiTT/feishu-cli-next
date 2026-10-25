// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

func nativeCells(t *testing.T, raw string) [][]nativeCell {
	t.Helper()
	var cells [][]nativeCell
	if err := json.Unmarshal([]byte(raw), &cells); err != nil {
		t.Fatal(err)
	}
	return cells
}

func TestNativeDataPreservesExactCellTypes(t *testing.T) {
	raw := `{"ranges":[{"range":"A1:J1","cells":[[{},{"value":null},{"value":""},{"value":"001"},{"value":9007199254740993},{"value":1.234567890123456789},{"formula":"=1+1","value":2},{"value":"=literal"},{"value":45600.123456789,"cell_styles":{"number_format":"yyyy-mm-dd hh:mm:ss"}},{"value":0.5,"cell_styles":{"number_format":"hh:mm"}}]]}]}`
	extent, _ := dataRange("A1:J1")
	cells, err := decodeNativePage(raw, extent)
	if err != nil {
		t.Fatal(err)
	}
	checks := map[int]string{1: `null`, 2: `""`, 3: `"001"`, 4: `9007199254740993`, 5: `1.234567890123456789`, 7: `"=literal"`, 8: `45600.123456789`, 9: `0.5`}
	for col, want := range checks {
		if string(cells[0][col]["value"]) != want {
			t.Errorf("column %d value = %s, want %s", col, cells[0][col]["value"], want)
		}
	}
	if _, present := cells[0][0]["value"]; present {
		t.Fatal("omitted value became present")
	}
	if string(cells[0][6]["formula"]) != `"=1+1"` {
		t.Fatal("lost formula")
	}
}

func TestNativeDataRejectsIncompletePages(t *testing.T) {
	extent, _ := dataRange("A1:B1")
	for _, raw := range []string{
		`{"ranges":[]}`,
		`{"ranges":[{"cells":[[{},{}]]}]}`,
		`{"ranges":[{"range":"invalid","cells":[[{},{}]]}]}`,
		`{"has_more":true,"ranges":[{"cells":[[{},{}]]}]}`,
		`{"ranges":[{"range":"A2:B2","cells":[[{},{}]]}]}`,
		`{"ranges":[{"actual_range":"C1:D1","cells":[[{},{}]]}]}`,
		`{"ranges":[{"row_indices":[2],"cells":[[{},{}]]}]}`,
		`{"ranges":[{"col_indices":["A","C"],"cells":[[{},{}]]}]}`,
		`{"ranges":[{"cells":[[{}]]}]}`,
		`{"ranges":[{"cells":[[{},null]]}]}`,
		`{"ranges":[{"cells":[[{"isRowTruncated":true},{}]]}]}`,
		`{"ranges":[{"is_truncated":true,"cells":[[{},{}]]}]}`,
	} {
		_, err := decodeNativePage(raw, extent)
		requireProblem(t, err, errs.CategoryInternal, errs.SubtypeInvalidResponse, "")
	}
}

func TestNativePaginationCoversTenThousandRows(t *testing.T) {
	extent, err := dataRange("C2:E10001")
	if err != nil {
		t.Fatal(err)
	}
	pages := dataRanges(extent)
	next := 1
	total := 0
	for _, region := range pages {
		page, err := parseCellRange(region)
		if err != nil {
			t.Fatal(err)
		}
		if page.row != next || page.col != 2 || page.cols != 3 || page.rows*page.cols > dataPageCells {
			t.Fatalf("unexpected page %s", region)
		}
		next += page.rows
		total += page.rows
	}
	if total != 10000 || next != 10001 {
		t.Fatalf("incomplete coverage %d", total)
	}
}

func TestNativeNoopPreservesOmittedEmptyNullAndPrecision(t *testing.T) {
	cells := nativeCells(t, `[[{}, {"value":null}, {"value":""}, {"value":"001"}, {"value":9007199254740993}, {"formula":"=1+1"}, {"value":0.5,"cell_styles":{"number_format":"hh:mm"}}]]`)
	writes, changed, err := nativeWritePlan(cells, cells)
	if err != nil || changed != 0 || nativePageChanged(writes) {
		t.Fatalf("unchanged data writes: %d %v %v", changed, writes, err)
	}
	want := nativeCells(t, `[[{"value":9007199254740993}]]`)
	current := nativeCells(t, `[[{"value":9007199254740992}]]`)
	_, changed, err = nativeWritePlan(want, current)
	if err != nil || changed != 1 {
		t.Fatalf("precision change ignored: %d %v", changed, err)
	}
}

func TestNativeWriteRejectsUnsupportedBeforeAnyPlan(t *testing.T) {
	for _, bad := range []string{`{"value":null}`, `{"unsupported":{"x":1}}`, `{"formula":true}`, `{"value":{"nested":"unsupported"}}`, `{"cell_styles":{"font_size":"large"}}`, `{"value":"keep","rich_text":[]}`, `{"value":"keep","multiple_values":null}`} {
		want := nativeCells(t, `[[{"value":"valid change"},`+bad+`]]`)
		current := nativeCells(t, `[[{},{}]]`)
		writes, _, err := nativeWritePlan(want, current)
		requireProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument, "")
		if writes != nil {
			t.Fatal("failed validation leaked partial write plan")
		}
	}
}

func TestNativeFormulaAndLiteralWriteAreDistinct(t *testing.T) {
	cells := nativeCells(t, `[[{"value":"=literal"},{"formula":"=1+1","value":2}]]`)
	literal, err := nativeWriteCell(cells[0][0])
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := literal["value"]; ok {
		t.Fatal("literal routed through auto-formula value")
	}
	if string(literal["rich_text"]) != `[{"text":"=literal","type":"text"}]` {
		t.Fatalf("literal rich text %s", literal["rich_text"])
	}
	formula, err := nativeWriteCell(cells[0][1])
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := formula["value"]; ok {
		t.Fatal("cached formula value became a write")
	}
	if string(formula["formula"]) != `"=1+1"` {
		t.Fatal("lost formula")
	}
	if nativeCellUnchanged(nativeCell{"value": json.RawMessage(`2`)}, cells[0][1]) {
		t.Fatal("cached formula value equated to static replacement")
	}
}

func TestNativeExportPrecisionAtToolBoundary(t *testing.T) {
	page := `{"revision":0,"ranges":[{"range":"A1:B1","cells":[[{"value":9007199254740993},{"value":1.234567890123456789}]]}]}`
	stdout, err := runShortcutWithStubs(t, DataExport, []string{"--spreadsheet-token", testToken, "--sheet-id", testSheetID, "--range", "A1:B1"}, toolOutputStub(testToken, "read", page), toolOutputStub(testToken, "read", page))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"9007199254740993", "1.234567890123456789", sheetDataSchema} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output lost %s: %s", want, stdout)
		}
	}
}

func TestSheetArtifactRejectsSchemaExtentAndFailure(t *testing.T) {
	cells := nativeCells(t, `[[{"value":"001"}]]`)
	data := sheetData{Schema: sheetDataSchema, SpreadsheetToken: testToken, SheetID: testSheetID, Range: "A1", Complete: true, Count: 1, Cells: cells}
	artifact := func(d sheetData, ok bool, schema string) string {
		result, _ := json.Marshal(map[string]interface{}{"ok": ok, "data": d})
		raw, _ := json.Marshal(output.ResultArtifact{Schema: schema, Result: result})
		return string(raw)
	}
	valid := artifact(data, true, output.ArtifactSchema)
	if _, _, err := parseSheetArtifact(valid, "C3"); err != nil {
		t.Fatal(err)
	}
	incomplete := data
	incomplete.Complete = false
	for _, raw := range []string{artifact(incomplete, true, output.ArtifactSchema), artifact(data, false, output.ArtifactSchema), artifact(data, true, "wrong"), valid + " trailing"} {
		_, _, err := parseSheetArtifact(raw, "C3")
		requireProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument, "")
	}
	_, _, err := parseSheetArtifact(valid, "A1:B1")
	requireProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument, "")
}

func TestNativeFormulaNoopIgnoresRecomputedCache(t *testing.T) {
	want := nativeCells(t, `[[{"formula":"=NOW()","value":1}]]`)
	current := nativeCells(t, `[[{"formula":"=NOW()","value":2}]]`)
	writes, changed, err := nativeWritePlan(want, current)
	if err != nil || changed != 0 || nativePageChanged(writes) {
		t.Fatalf("formula cache caused write: %d %v", changed, err)
	}
}

func TestNativeEmptyFormulaMetadataDoesNotHideStaticChange(t *testing.T) {
	want := nativeCells(t, `[[{"formula":"","value":"new"}]]`)
	current := nativeCells(t, `[[{"formula":"","value":"old"}]]`)
	writes, changed, err := nativeWritePlan(want, current)
	if err != nil || changed != 1 {
		t.Fatalf("static change missed: %d %v", changed, err)
	}
	if string(writes[0][0]["value"]) != `"new"` {
		t.Fatalf("value lost: %v", writes)
	}
	if _, ok := writes[0][0]["formula"]; ok {
		t.Fatal("empty read metadata became a formula write")
	}
}

func TestNativeRichContentCacheDoesNotReplaceIdentity(t *testing.T) {
	cells := nativeCells(t, `[[{"value":"=literal","rich_text":[{"type":"link","text":"=literal","link":"https://example.test"}]}]]`)
	write, err := nativeWriteCell(cells[0][0])
	if err != nil {
		t.Fatal(err)
	}
	if _, present := write["value"]; present {
		t.Fatal("cached text became a second content write")
	}
	if !equalNativeJSON(write["rich_text"], cells[0][0]["rich_text"]) {
		t.Fatal("native rich link was replaced by plain literal text")
	}
	cells = nativeCells(t, `[[{"formula":"=1+1","rich_text":[{"type":"text","text":"2"}]}]]`)
	_, err = nativeWriteCell(cells[0][0])
	requireProblem(t, err, errs.CategoryValidation, errs.SubtypeInvalidArgument, "")
}

func TestNativePlainValueReplacesStructuredContent(t *testing.T) {
	want := nativeCells(t, `[[{"value":"hello"}]]`)
	for key, content := range map[string]string{
		"rich_text":       `[{"type":"link","text":"hello","link":"https://example.test"}]`,
		"multiple_values": `[{"value":"hello"}]`,
	} {
		t.Run(key, func(t *testing.T) {
			current := nativeCells(t, `[[{"value":"hello","`+key+`":`+content+`}]]`)
			writes, changed, err := nativeWritePlan(want, current)
			if err != nil || changed != 1 || len(writes[0][0]) != 1 || string(writes[0][0]["value"]) != `"hello"` {
				t.Fatalf("cached structured text hid plain replacement: changed=%d writes=%v err=%v", changed, writes, err)
			}
			writes, changed, err = nativeWritePlan(current, current)
			if err != nil || changed != 0 || nativePageChanged(writes) {
				t.Fatalf("unchanged structured identity caused write: changed=%d err=%v", changed, err)
			}
		})
	}
}

func TestNativeExportCombinesRawTypesAndExplicitFormula(t *testing.T) {
	raw := toolOutputStub(testToken, "read", `{"revision":1,"ranges":[{"range":"A1:H1","cells":[[{"value":0},{"value":false},{"value":""},{},{"value":null},{"value":"=literal"},{"value":"001"},{"value":2,"cell_styles":{"number_format":"0.00"}}]]}]}`)
	formula := toolOutputStub(testToken, "read", `{"revision":1,"ranges":[{"range":"A1:H1","cells":[[{"value":"0"},{"value":"FALSE"},{},{},{},{"value":"=literal"},{"value":"001"},{"formula":"=1+1","value":2}]]}]}`)
	out, err := runShortcutWithStubs(t, DataExport, []string{"--spreadsheet-token", testToken, "--sheet-id", testSheetID, "--range", "A1:H1"}, raw, formula)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Data sheetData `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatal(err)
	}
	want := nativeCells(t, `[[{"value":0},{"value":false},{"value":""},{},{"value":null},{"value":"=literal"},{"value":"001"},{"formula":"=1+1","value":2,"cell_styles":{"number_format":"0.00"}}]]`)
	gotJSON, _ := json.Marshal(envelope.Data.Cells)
	wantJSON, _ := json.Marshal(want)
	if !equalNativeJSON(gotJSON, wantJSON) {
		t.Fatalf("native identity or typed value lost: %s", gotJSON)
	}
	if !strings.Contains(string(raw.CapturedBody), "raw_value") || !strings.Contains(string(formula.CapturedBody), "formula") || len(raw.CapturedBodies) != 1 || len(formula.CapturedBodies) != 1 {
		t.Fatal("export must issue exactly one read of each rendering")
	}
	calls := parseDryRunAPI(t, DataExport, []string{"--spreadsheet-token", testToken, "--sheet-id", testSheetID, "--range", "A1:H1"})
	if len(calls) != 2 {
		t.Fatalf("dry-run omitted paired read: %d calls", len(calls))
	}
}

func TestNativePageRevisionRequiresIntegerEvidence(t *testing.T) {
	for _, good := range []string{"0", "1", "9007199254740993"} {
		got, err := nativePageRevision(`{"revision":` + good + `}`)
		if err != nil || got != good {
			t.Fatalf("revision %s: %s %v", good, got, err)
		}
	}
	for _, bad := range []string{`{}`, `{"revision":null}`, `{"revision":"1"}`, `{"revision":true}`, `{"revision":1.5}`, `{"revision":1e2}`, `{"revision":-1}`, `{"revision":[]}`} {
		_, err := nativePageRevision(bad)
		requireProblem(t, err, errs.CategoryInternal, errs.SubtypeInvalidResponse, "")
	}
}

func TestNativeFormulaMergeRejectsConflictingEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, raw, formula string
		category           errs.Category
		subtype            errs.Subtype
	}{
		{"changed cache", `{"value":2}`, `{"formula":"=1+1","value":3}`, errs.CategoryAPI, errs.SubtypeConflict},
		{"changed formula", `{"formula":"=1+1","value":2}`, `{"formula":"=2","value":2}`, errs.CategoryAPI, errs.SubtypeConflict},
		{"missing formula", `{"formula":"=1+1","value":2}`, `{"value":2}`, errs.CategoryAPI, errs.SubtypeConflict},
		{"null formula", `{"value":2}`, `{"formula":null}`, errs.CategoryInternal, errs.SubtypeInvalidResponse},
		{"numeric formula", `{"value":2}`, `{"formula":2}`, errs.CategoryInternal, errs.SubtypeInvalidResponse},
		{"invalid syntax", `{"value":2}`, `{"formula":"SUM(A1)"}`, errs.CategoryInternal, errs.SubtypeInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := mergeNativeFormulas(nativeCells(t, `[[`+tc.raw+`]]`), nativeCells(t, `[[`+tc.formula+`]]`))
			requireProblem(t, err, tc.category, tc.subtype, "")
		})
	}
}

func TestNativeIncompleteCellResponseNamesCellsFallback(t *testing.T) {
	extent, _ := dataRange("A1:B1")
	_, err := decodeNativePage(`{"ranges":[{"range":"A1:B1","cells":[[{}]]}]}`, extent)
	var internal *errs.InternalError
	if !errors.As(err, &internal) {
		t.Fatalf("want internal error, got %T %v", err, err)
	}
	if internal.Subtype != errs.SubtypeInvalidResponse || !strings.Contains(internal.Hint, "sheets +cells-get") {
		t.Fatalf("missing fallback: subtype=%s hint=%q", internal.Subtype, internal.Hint)
	}
}
