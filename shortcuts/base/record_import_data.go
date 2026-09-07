// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"encoding/json"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

// This is the Sheets domain contract inside the existing result artifact, not a
// second generic file format. Unknown native cell content is never discarded.
type importSheetData struct {
	Schema           string                         `json:"schema"`
	SpreadsheetToken string                         `json:"spreadsheet_token"`
	SheetID          string                         `json:"sheet_id"`
	Range            string                         `json:"range"`
	Complete         bool                           `json:"complete"`
	Count            int                            `json:"count"`
	Cells            [][]map[string]json.RawMessage `json:"cells"`
}

type importColumn struct {
	Column  int    `json:"column"`
	FieldID string `json:"field_id"`
	Type    string `json:"type"`
}

type recordImport struct {
	artifact output.ResultArtifact
	source   importSheetData
	columns  []importColumn
	records  []map[string]interface{}
}

func importInvalid(param, reason string) *errs.ValidationError {
	return errs.NewValidationError(errs.SubtypeInvalidArgument, "cannot import records: %s", reason).WithParam(param)
}

var importRangePattern = regexp.MustCompile(`^([A-Z]+)([1-9][0-9]*):([A-Z]+)([1-9][0-9]*)$`)

func importExtent(region string) (int, int, error) {
	parts := importRangePattern.FindStringSubmatch(region)
	if parts == nil {
		return 0, 0, importInvalid("--artifact", "source range must be an explicit rectangular A1 extent")
	}
	column := func(value string) int {
		n := 0
		for _, r := range value {
			if n > 1000000 {
				return -1
			}
			n = n*26 + int(r-'A'+1)
		}
		return n
	}
	first, last := column(parts[1]), column(parts[3])
	start, e1 := strconv.Atoi(parts[2])
	end, e2 := strconv.Atoi(parts[4])
	if e1 != nil || e2 != nil || first < 1 || last < first || end < start || end-start >= 10000 || last-first >= 5000 {
		return 0, 0, importInvalid("--artifact", "source extent is invalid or exceeds 10,000 rows / 5,000 columns")
	}
	return end - start + 1, last - first + 1, nil
}

func parseRecordImport(artifactRaw, mappingRaw string) (recordImport, error) {
	var imp recordImport
	if err := parseBaseJSON([]byte(artifactRaw), &imp.artifact); err != nil {
		return imp, importInvalid("--artifact", "invalid result artifact JSON").WithCause(err)
	}
	if imp.artifact.Schema != output.ArtifactSchema {
		return imp, importInvalid("--artifact", "expected feishu-cli-next/result/v1")
	}
	var result struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := parseBaseJSON(imp.artifact.Result, &result); err != nil || !result.OK {
		return imp, importInvalid("--artifact", "source result must be successful")
	}
	sourceDecoder := json.NewDecoder(strings.NewReader(string(result.Data)))
	sourceDecoder.DisallowUnknownFields()
	if err := sourceDecoder.Decode(&imp.source); err != nil {
		return imp, importInvalid("--artifact", "invalid typed Sheets payload")
	}
	source := imp.source
	if source.Schema != "feishu-cli-next/sheets-data/v1" || !source.Complete || source.SpreadsheetToken == "" || source.SheetID == "" {
		return imp, importInvalid("--artifact", "expected complete typed Sheets data with source identifiers")
	}
	rows, width, err := importExtent(source.Range)
	if err != nil {
		return imp, err
	}
	if rows > 1000000/width || len(source.Cells) != rows || source.Count != rows {
		return imp, importInvalid("--artifact", "row count does not match source extent")
	}
	// Reject mapping typos rather than silently ignoring user policy.
	decoder := json.NewDecoder(strings.NewReader(mappingRaw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&imp.columns); err != nil {
		return imp, importInvalid("--mapping", "expected [{column:0,field_id:\"fld...\",type:\"text\"}] with no unknown properties")
	}
	if !json.Valid([]byte(mappingRaw)) || len(imp.columns) != width {
		return imp, importInvalid("--mapping", "map every source column exactly once")
	}
	var mappingFields []map[string]json.RawMessage
	_ = json.Unmarshal([]byte(mappingRaw), &mappingFields)
	for _, field := range mappingFields {
		raw, present := field["column"]
		var index *int
		if !present || json.Unmarshal(raw, &index) != nil || index == nil {
			return imp, importInvalid("--mapping", "every mapping needs an explicit integer column index")
		}
	}
	seenColumns := make(map[int]bool, width)
	seenFields := make(map[string]bool, width)
	for _, col := range imp.columns {
		if col.Column < 0 || col.Column >= width || seenColumns[col.Column] || seenFields[col.FieldID] || !strings.HasPrefix(col.FieldID, "fld") {
			return imp, importInvalid("--mapping", "column indices and destination field IDs must be unique and valid")
		}
		if col.Type != "text" && col.Type != "number" && col.Type != "checkbox" {
			return imp, importInvalid("--mapping", "only lossless text, number and checkbox conversions are supported; native dates, formulas, identities and rich cells require an explicit separate conversion")
		}
		seenColumns[col.Column] = true
		seenFields[col.FieldID] = true
	}
	for _, row := range source.Cells {
		if len(row) != width {
			return imp, importInvalid("--artifact", "ragged row cannot establish complete cell extent")
		}
		record := make(map[string]interface{}, width)
		for _, col := range imp.columns {
			cell := row[col.Column]
			if cell == nil {
				return imp, importInvalid("--artifact", "native cells must be objects; null is not an omitted cell")
			}
			for key := range cell {
				if key != "value" {
					return imp, importInvalid("--artifact", "native cell contains "+key+"; conversion would discard formatting, formula or identity information")
				}
			}
			raw, present := cell["value"]
			if !present {
				continue
			}
			var value interface{}
			if err := parseBaseJSON(raw, &value); err != nil {
				return imp, importInvalid("--artifact", "invalid native cell value")
			}
			if value != nil {
				valid := false
				switch col.Type {
				case "text":
					_, valid = value.(string)
				case "number":
					_, valid = value.(json.Number)
				case "checkbox":
					_, valid = value.(bool)
				}
				if !valid {
					return imp, importInvalid("--mapping", "native cell value does not match destination type "+col.Type)
				}
			}
			if number, ok := value.(json.Number); ok && !baseDoubleRoundtrip(number) {
				return imp, importInvalid("--artifact", "number cannot round-trip through Base double storage without decimal precision loss")
			}
			record[col.FieldID] = value
		}
		imp.records = append(imp.records, record)
	}
	return imp, nil
}

// The native Base CellValue contract stores number fields as doubles. A typed
// conversion may not promise precision that the destination cannot represent.
func baseDoubleRoundtrip(number json.Number) bool {
	value, err := strconv.ParseFloat(string(number), 64)
	if err != nil || math.IsInf(value, 0) || math.IsNaN(value) {
		return false
	}
	if value == 0 {
		mantissa := strings.Split(strings.ToLower(string(number)), "e")[0]
		return strings.Trim(mantissa, "-+.0") == ""
	}
	original, ok := new(big.Rat).SetString(string(number))
	if !ok {
		return false
	}
	roundtrip, ok := new(big.Rat).SetString(strconv.FormatFloat(value, 'g', -1, 64))
	return ok && original.Cmp(roundtrip) == 0
}
