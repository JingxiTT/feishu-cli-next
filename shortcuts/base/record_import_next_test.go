// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func importTestArtifact(cells, region string, count int) string {
	return fmt.Sprintf(`{"schema":"feishu-cli-next/result/v1","binding":{},"result":{"ok":true,"data":{"schema":"feishu-cli-next/sheets-data/v1","spreadsheet_token":"sheet_synthetic","sheet_id":"tab_synthetic","range":%q,"complete":true,"count":%d,"cells":%s}}}`, region, count, cells)
}

func TestRecordImportPreservesScalarValuesAndPresence(t *testing.T) {
	imp, err := parseRecordImport(importTestArtifact(`[[{"value":9007199254740991},{"value":"001"},{"value":null},{},{"value":" "},{"value":false}]]`, "A1:F1", 1), `[{"column":0,"field_id":"fld_num","type":"number"},{"column":1,"field_id":"fld_code","type":"text"},{"column":2,"field_id":"fld_null","type":"text"},{"column":3,"field_id":"fld_omitted","type":"text"},{"column":4,"field_id":"fld_whitespace","type":"text"},{"column":5,"field_id":"fld_bool","type":"checkbox"}]`)
	if err != nil {
		t.Fatal(err)
	}
	row := imp.records[0]
	if row["fld_num"] != json.Number("9007199254740991") || row["fld_code"] != "001" || row["fld_whitespace"] != " " || row["fld_bool"] != false {
		t.Fatalf("changed cells: %#v", row)
	}
	if value, present := row["fld_null"]; !present || value != nil {
		t.Fatal("null lost")
	}
	if _, present := row["fld_omitted"]; present {
		t.Fatal("omission became clear")
	}
}

func TestRecordImportRejectsLossyOrIncompleteConversionBeforeWrites(t *testing.T) {
	mapping := `[{"column":0,"field_id":"fld_name","type":"text"}]`
	cases := []struct {
		name, cells, region string
		count               int
		mapping             string
	}{
		{"formula", `[[{"value":"4","formula":"=2+2"}]]`, "A1:A1", 1, mapping},
		{"date style", `[[{"value":45000,"cell_styles":{"number_format":"yyyy-mm-dd"}}]]`, "A1:A1", 1, mapping},
		{"rich", `[[{"value":{"text":"hi"}}]]`, "A1:A1", 1, mapping},
		{"null cell", `[[null]]`, "A1:A1", 1, mapping},
		{"partial rows", `[[{"value":"x"}]]`, "A1:A2", 2, mapping},
		{"ragged", `[[]]`, "A1:A1", 1, mapping},
		{"wrong type", `[[{"value":5}]]`, "A1:A1", 1, mapping},
		{"duplicate field", `[[{},{}]]`, "A1:B1", 1, `[{"column":0,"field_id":"fld_x","type":"text"},{"column":1,"field_id":"fld_x","type":"text"}]`},
		{"null column", `[[{}]]`, "A1:A1", 1, `[{"column":null,"field_id":"fld_x","type":"text"}]`},
		{"missing column", `[[{}]]`, "A1:A1", 1, `[{"field_id":"fld_x","type":"text"}]`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRecordImport(importTestArtifact(tt.cells, tt.region, tt.count), tt.mapping); err == nil {
				t.Fatal("unsafe conversion accepted")
			}
		})
	}
	for _, replacement := range []string{`"complete":false`, `"complete":true,"truncated":true`} {
		artifact := strings.Replace(importTestArtifact(`[[{}]]`, "A1:A1", 1), `"complete":true`, replacement, 1)
		if _, err := parseRecordImport(artifact, mapping); err == nil {
			t.Fatal("incomplete source accepted")
		}
	}
}

func TestRecordImportBatchIDsRequireCompleteUniqueEvidence(t *testing.T) {
	for _, data := range []map[string]interface{}{
		{"record_id_list": []string{"rec_one", "rec_two"}, "async": true},
		{"record_id_list": []string{"rec_one"}},
		{"record_id_list": []string{"rec_one", "rec_one"}},
		{"record_id_list": []string{"rec_one", "rec_two"}, "ignored_fields": []interface{}{map[string]interface{}{"id": "fld_x"}}},
	} {
		if _, err := recordImportBatchIDs(data, 2, nil); err == nil {
			t.Fatalf("ambiguous batch accepted: %#v", data)
		}
	}
	if _, err := recordImportBatchIDs(map[string]interface{}{"record_id_list": []string{"rec_one", "rec_two"}}, 2, nil); err != nil {
		t.Fatal(err)
	}
}

func TestRecordImportRejectsNumbersThatLoseDestinationPrecision(t *testing.T) {
	for _, number := range []string{"9007199254740993", "0.1234567890123456789", "1e400", "1e-400"} {
		if _, err := parseRecordImport(importTestArtifact(`[[{"value":`+number+`}]]`, "A1:A1", 1), `[{"column":0,"field_id":"fld_number","type":"number"}]`); err == nil {
			t.Fatalf("accepted lossy number %s", number)
		}
	}
	for _, number := range []json.Number{"9007199254740991", "0.1", "1.00", "1e3"} {
		if !baseDoubleRoundtrip(number) {
			t.Fatalf("rejected supported number %s", number)
		}
	}
}

func TestRecordImportBatchIDsRejectPreviouslyConfirmedIDs(t *testing.T) {
	confirmed := map[string]bool{"rec_previous": true}
	for _, ids := range [][]string{{"rec_new", "rec_previous"}, {"rec_previous", "rec_new"}} {
		if _, err := recordImportBatchIDs(map[string]interface{}{"record_id_list": ids}, 2, confirmed); err == nil {
			t.Fatal("accepted a record ID confirmed by an earlier batch")
		}
		if len(confirmed) != 1 || !confirmed["rec_previous"] {
			t.Fatal("failed batch changed confirmed evidence")
		}
	}
	if _, err := recordImportBatchIDs(map[string]interface{}{"record_id_list": []string{"rec_new", "rec_other"}}, 2, confirmed); err != nil {
		t.Fatal(err)
	}
}

func TestRecordImportRejectsEmptyTextAfterCompletePreflight(t *testing.T) {
	cells := make([][]map[string]interface{}, 401)
	for i := range cells {
		cells[i] = []map[string]interface{}{{"value": "representable"}}
	}
	cells[400][0]["value"] = ""
	raw, _ := json.Marshal(cells)
	_, err := parseRecordImport(importTestArtifact(string(raw), "A1:A401", 401), `[{"column":0,"field_id":"fld_text","type":"text"}]`)
	if err == nil || !strings.Contains(err.Error(), "empty text") {
		t.Fatalf("late empty text accepted: %v", err)
	}
}

func TestRecordImportPlainRowsNamesBatchCreate(t *testing.T) {
	_, err := parseRecordImport(`[{"Name":"A"}]`, `[{"column":0,"field_id":"fld_name","type":"text"}]`)
	var validation *errs.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("want validation error, got %T %v", err, err)
	}
	if validation.Subtype != errs.SubtypeInvalidArgument || !strings.Contains(validation.Hint, "base +record-batch-create") {
		t.Fatalf("missing batch-create fallback: subtype=%s hint=%q", validation.Subtype, validation.Hint)
	}
}
