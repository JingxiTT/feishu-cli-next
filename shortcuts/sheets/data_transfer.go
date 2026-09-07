// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
	"github.com/larksuite/cli/shortcuts/common"
)

const sheetDataSchema = "feishu-cli-next/sheets-data/v1"
const dataPageCells = 5000
const dataMaxCells = 1000000

// Native cell JSON retains field presence and decimal tokens. In particular,
// {} is not {"value":null}, and a formula is not a value starting with '='.
type nativeCell map[string]json.RawMessage

type sheetData struct {
	Schema           string         `json:"schema"`
	SpreadsheetToken string         `json:"spreadsheet_token"`
	SheetID          string         `json:"sheet_id"`
	Range            string         `json:"range"`
	Complete         bool           `json:"complete"`
	Count            int            `json:"count"`
	Cells            [][]nativeCell `json:"cells"`
}

// DataExport reads every cell in an explicit extent through bounded row pages.
// It does not infer a used range, invent missing cells, or claim a snapshot lock.
var DataExport = common.Shortcut{
	Service: "sheets", Command: "+data-export", Description: "Export a complete explicit range as native typed cells, preserving formulas, numeric precision, and field presence.",
	Risk: "read", Scopes: []string{"sheets:spreadsheet:read"}, AuthTypes: []string{"user", "bot"}, HasFormat: true,
	Flags: flagsFor("+data-export"),
	Validate: func(_ context.Context, r *common.RuntimeContext) error {
		_, err := dataRange(r.Str("range"))
		return err
	},
	DryRun: func(_ context.Context, r *common.RuntimeContext) *common.DryRunAPI {
		return dataReadDryRun(r)
	},
	Execute: func(ctx context.Context, r *common.RuntimeContext) error {
		extent, err := dataRange(r.Str("range"))
		if err != nil {
			return err
		}
		cells, err := readNativeCells(ctx, r, r.Str("spreadsheet-token"), r.Str("sheet-id"), extent)
		if err != nil {
			return err
		}
		r.Out(sheetData{Schema: sheetDataSchema, SpreadsheetToken: r.Str("spreadsheet-token"), SheetID: r.Str("sheet-id"), Range: extent.sized(extent.rows, extent.cols), Complete: true, Count: len(cells), Cells: cells}, nil)
		return nil
	},
	Tips: []string{"Use --save-result PATH to keep cells outside model context with the shared identity binding. Completeness covers the explicit range only; reads are not a snapshot. Date/time serials retain number_format; no timezone is inferred."},
}

func dataRange(value string) (cellRange, error) {
	extent, err := parseCellRange(value)
	if err != nil {
		return cellRange{}, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid rectangular range: %v", err).WithParam("--range").WithCause(err)
	}
	if extent.sheetQualifier != "" {
		return cellRange{}, errs.NewValidationError(errs.SubtypeInvalidArgument, "use --sheet-id separately from --range").WithParam("--range")
	}
	if extent.cols > dataPageCells || extent.rows > dataMaxCells/extent.cols {
		return cellRange{}, errs.NewValidationError(errs.SubtypeInvalidArgument, "range exceeds the supported export/import extent (1,000,000 cells; 5,000 columns)").WithParam("--range")
	}
	return extent, nil
}

func dataRanges(extent cellRange) []string {
	rowsPerPage := dataPageCells / extent.cols
	var pages []string
	for row := 0; row < extent.rows; row += rowsPerPage {
		rows := min(rowsPerPage, extent.rows-row)
		pages = append(pages, fmt.Sprintf("%s%d:%s%d", columnIndexToLetter(extent.col), extent.row+row+1, columnIndexToLetter(extent.col+extent.cols-1), extent.row+row+rows))
	}
	return pages
}

func dataReadInput(token, sheetID, region string) map[string]interface{} {
	return map[string]interface{}{"excel_id": token, "sheet_id": sheetID, "ranges": []string{region}, "include_styles": true, "value_render_option": "raw_value", "cell_limit": dataPageCells, "max_chars": outputPathReadLimit, "include_truncation_info": true}
}

type nativeRangeResponse struct {
	Truncated   bool `json:"truncated"`
	IsTruncated bool `json:"is_truncated"`
	HasMore     bool `json:"has_more"`
	Ranges      []struct {
		Range       string         `json:"range"`
		ActualRange string         `json:"actual_range"`
		RowIndices  []int          `json:"row_indices"`
		ColIndices  []string       `json:"col_indices"`
		Truncated   bool           `json:"truncated"`
		IsTruncated bool           `json:"is_truncated"`
		HasMore     bool           `json:"has_more"`
		Cells       [][]nativeCell `json:"cells"`
	} `json:"ranges"`
}

func decodeNativePage(raw string, extent cellRange) ([][]nativeCell, error) {
	var page nativeRangeResponse
	if err := json.Unmarshal([]byte(raw), &page); err != nil {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid native cell response: %v", err).WithCause(err)
	}
	if page.Truncated || page.IsTruncated || page.HasMore || len(page.Ranges) != 1 {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cell response is incomplete or has an unexpected range count")
	}
	block := page.Ranges[0]
	if block.Truncated || block.IsTruncated || block.HasMore || len(block.Cells) != extent.rows {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cell response does not contain the complete requested rows")
	}
	for _, reported := range []string{block.Range, block.ActualRange} {
		if reported == "" {
			continue
		}
		actual, err := parseCellRange(reported)
		if err != nil || actual.col != extent.col || actual.row != extent.row || actual.rows != extent.rows || actual.cols != extent.cols {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned a different cell extent")
		}
	}
	if len(block.RowIndices) > 0 {
		if len(block.RowIndices) != extent.rows {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned incomplete row indices")
		}
		for i, row := range block.RowIndices {
			if row != extent.row+i+1 {
				return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned noncontiguous row indices")
			}
		}
	}
	if len(block.ColIndices) > 0 {
		if len(block.ColIndices) != extent.cols {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned incomplete column indices")
		}
		for i, col := range block.ColIndices {
			if col != columnIndexToLetter(extent.col+i) {
				return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned noncontiguous column indices")
			}
		}
	}
	for _, row := range block.Cells {
		if len(row) != extent.cols {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "cell response omitted columns; missing cells cannot be inferred")
		}
		for _, cell := range row {
			if cell == nil {
				return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned a null cell object; field presence is unknown")
			}
			for _, key := range []string{"isRowTruncated", "isColTruncated", "truncated", "is_truncated"} {
				if bytes.Equal(bytes.TrimSpace(cell[key]), []byte("true")) {
					return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "backend returned a truncated cell")
				}
			}
		}
	}
	return block.Cells, nil
}

func readNativeCells(ctx context.Context, r *common.RuntimeContext, token, sheetID string, extent cellRange) ([][]nativeCell, error) {
	cells := make([][]nativeCell, 0, extent.rows)
	for _, region := range dataRanges(extent) {
		raw, err := callToolOutput(ctx, r, token, ToolKindRead, "get_cell_ranges", dataReadInput(token, sheetID, region))
		if err != nil {
			return nil, err
		}
		part, _ := parseCellRange(region)
		rows, err := decodeNativePage(raw, part)
		if err != nil {
			return nil, err
		}
		cells = append(cells, rows...)
	}
	return cells, nil
}

// equalNativeJSON compares decoded decimal tokens without converting numbers
// to float64. Object ordering is irrelevant; omitted fields remain distinct.
func equalNativeJSON(a, b json.RawMessage) bool {
	var x, y interface{}
	da := json.NewDecoder(bytes.NewReader(a))
	da.UseNumber()
	db := json.NewDecoder(bytes.NewReader(b))
	db.UseNumber()
	if da.Decode(&x) != nil || db.Decode(&y) != nil {
		return false
	}
	ax, _ := json.Marshal(x)
	by, _ := json.Marshal(y)
	return bytes.Equal(ax, by)
}

func nativeCellUnchanged(want, current nativeCell) bool {
	for key, value := range want {
		if key == "value" {
			if nativeFormulaPresent(want) {
				continue
			}
		}
		actual, ok := current[key]
		if !ok || !equalNativeJSON(value, actual) {
			return false
		}
	}
	// A static value must not be mistaken for an equal cached formula value.
	if _, ok := want["value"]; ok && !nativeFormulaPresent(want) {
		if raw, ok := current["formula"]; ok && string(raw) != "null" && string(raw) != `""` {
			return false
		}
	}
	return true
}

func nativeWriteCell(cell nativeCell) (nativeCell, error) {
	out := nativeCell{}
	for key, value := range cell {
		switch key {
		case "value", "formula", "cell_styles", "border_styles", "note", "data_validation", "rich_text", "multiple_values":
			out[key] = value
		case "isRowTruncated", "isColTruncated": // read evidence, validated before import
		default:
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "cell field %q has no verified write conversion", key).WithParam("--artifact")
		}
	}
	if raw, ok := out["formula"]; ok {
		var formula string
		if json.Unmarshal(raw, &formula) != nil || (formula != "" && !strings.HasPrefix(formula, "=")) {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "formula must be a string starting with =").WithParam("--artifact")
		}
		if formula == "" {
			delete(out, "formula") // read metadata saying this is a static cell
		} else {
			delete(out, "value") // backend cached value is not a second content write
		}
	}
	if raw, ok := out["value"]; ok {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "null cell values have no verified write conversion; use an explicit clear operation").WithParam("--artifact")
		}
		var value string
		if json.Unmarshal(raw, &value) == nil && strings.HasPrefix(value, "=") {
			// set_cell_range auto-evaluates value strings beginning '='. Native text
			// segments explicitly preserve literal text instead.
			text, _ := json.Marshal([]map[string]string{{"type": "text", "text": value}})
			out["rich_text"] = text
			delete(out, "value")
		}
	}

	raw, err := json.Marshal([][]nativeCell{{out}})
	if err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid cell JSON: %v", err).WithParam("--artifact").WithCause(err)
	}
	var value interface{}
	// The existing schema checker expects float64 numbers. This decoded copy
	// is validation-only; the returned write payload remains the original raw
	// decimal JSON and never forwards the checker's floating-point values.
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid cell JSON: %v", err).WithParam("--artifact").WithCause(err)
	}
	if err := validateValueAgainstSchema(newMapFlagViewForCommand("+cells-set", nil), "cells", value); err != nil {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "unsupported native write cell: %v", err).WithParam("--artifact").WithCause(err)
	}
	return out, nil
}

// DataImport reuses a bound complete export; target placement remains explicit.
// The backend patch contract retains omitted fields and does not support CAS.
var DataImport = common.Shortcut{
	Service: "sheets", Command: "+data-import", Description: "Apply a bound native-cell artifact to an explicit range, skipping unchanged cells before writes.",
	Risk: "write", Scopes: []string{"sheets:spreadsheet:read", "sheets:spreadsheet:write_only"}, AuthTypes: []string{"user", "bot"}, HasFormat: true,
	Flags: flagsFor("+data-import"),
	Validate: func(_ context.Context, r *common.RuntimeContext) error {
		_, _, err := parseSheetArtifact(r.Str("artifact"), r.Str("range"))
		return err
	},
	DryRun: func(_ context.Context, r *common.RuntimeContext) *common.DryRunAPI {
		return dataReadDryRun(r).Set("mutation_plan", map[string]interface{}{
			"method": "POST", "path": toolInvokePath(r.Str("spreadsheet-token"), ToolKindWrite),
			"tool_name": "set_cell_range", "condition": "after verifying artifact binding and reading every target page, validate every changed cell then write only changed pages", "unchanged_cells": "omitted from writes", "rollback": false,
		})
	},
	Execute: func(ctx context.Context, r *common.RuntimeContext) error {
		artifact, data, err := parseSheetArtifact(r.Str("artifact"), r.Str("range"))
		if err != nil {
			return err
		}
		binding, err := r.ResolveArtifactBinding()
		if err != nil {
			return err
		}
		if err := common.ValidateArtifactBinding(artifact.Binding, binding, "--artifact", true); err != nil {
			return err
		}
		extent, _ := dataRange(r.Str("range"))
		current, err := readNativeCells(ctx, r, r.Str("spreadsheet-token"), r.Str("sheet-id"), extent)
		if err != nil {
			return err
		}
		writes, changed, err := nativeWritePlan(data.Cells, current)
		if err != nil {
			return err
		}
		pages := dataRanges(extent)
		row := 0
		acknowledged := 0
		for _, region := range pages {
			part, _ := parseCellRange(region)
			page := writes[row : row+part.rows]
			row += part.rows
			if !nativePageChanged(page) {
				continue
			}
			_, err := callTool(ctx, r, r.Str("spreadsheet-token"), ToolKindWrite, "set_cell_range", map[string]interface{}{"excel_id": r.Str("spreadsheet-token"), "sheet_id": r.Str("sheet-id"), "range": region, "cells": page})
			if err != nil {
				r.Out(map[string]interface{}{"status": "incomplete", "acknowledged_pages": acknowledged, "failed_range": region, "retry": "inspect mutation receipts and read target cells before deciding whether to retry"}, nil)
				return err
			}
			acknowledged++
		}
		r.Out(map[string]interface{}{"spreadsheet_token": r.Str("spreadsheet-token"), "sheet_id": r.Str("sheet-id"), "range": r.Str("range"), "noop": changed == 0, "changed_cells": changed, "acknowledged_pages": acknowledged, "concurrency": "unconditional writes; no snapshot or compare-and-swap protection"}, nil)
		return nil
	},
	Tips: []string{"Requires a complete +data-export --save-result artifact with verified current account and tenant binding. Omitted fields are patches, not deletion. Unsupported changed cells fail before any write. Pages are sequential with no rollback; reads and writes are not isolated from other editors."},
}

func parseSheetArtifact(raw, targetRange string) (output.ResultArtifact, sheetData, error) {
	var artifact output.ResultArtifact
	var data sheetData
	bad := func(message string) (output.ResultArtifact, sheetData, error) {
		return artifact, data, errs.NewValidationError(errs.SubtypeInvalidArgument, "%s", message).WithParam("--artifact")
	}
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&artifact); err != nil {
		return artifact, data, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid result artifact: %v", err).WithParam("--artifact").WithCause(err)
	}
	if err := decoderExpectEOF(dec); err != nil {
		return artifact, data, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid trailing artifact data: %v", err).WithParam("--artifact").WithCause(err)
	}
	if artifact.Schema != output.ArtifactSchema {
		return bad("unsupported artifact container schema")
	}
	var result struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(artifact.Result, &result); err != nil {
		return artifact, data, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid artifact result: %v", err).WithParam("--artifact").WithCause(err)
	}
	if !result.OK {
		return bad("artifact result is not successful")
	}
	dec = json.NewDecoder(bytes.NewReader(result.Data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&data); err != nil {
		return artifact, data, errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid native cell payload: %v", err).WithParam("--artifact").WithCause(err)
	}
	if data.Schema != sheetDataSchema || !data.Complete || data.SpreadsheetToken == "" || data.SheetID == "" {
		return bad("artifact must be a complete bound native sheet-data export")
	}
	source, err := dataRange(data.Range)
	if err != nil {
		return artifact, data, err
	}
	target, err := dataRange(targetRange)
	if err != nil {
		return artifact, data, err
	}
	if source.rows != target.rows || source.cols != target.cols || len(data.Cells) != source.rows || data.Count != source.rows {
		return bad("artifact and target extents do not match")
	}
	for _, row := range data.Cells {
		if len(row) != source.cols {
			return bad("artifact omitted cell columns")
		}
		for _, cell := range row {
			if cell == nil {
				return bad("artifact contains a null cell object")
			}
			for _, key := range []string{"isRowTruncated", "isColTruncated", "truncated", "is_truncated"} {
				if bytes.Equal(bytes.TrimSpace(cell[key]), []byte("true")) {
					return bad("artifact contains a truncated cell")
				}
			}
		}
	}
	return artifact, data, nil
}

func nativeWritePlan(want, current [][]nativeCell) ([][]nativeCell, int, error) {
	writes := make([][]nativeCell, len(want))
	changed := 0
	for row := range want {
		writes[row] = make([]nativeCell, len(want[row]))
		for col, cell := range want[row] {
			writes[row][col] = nativeCell{}
			if nativeCellUnchanged(cell, current[row][col]) {
				continue
			}
			write, err := nativeWriteCell(cell)
			if err != nil {
				return nil, 0, err
			}
			writes[row][col] = write
			changed++
		}
	}
	return writes, changed, nil
}

func nativePageChanged(page [][]nativeCell) bool {
	for _, row := range page {
		for _, cell := range row {
			if len(cell) > 0 {
				return true
			}
		}
	}
	return false
}

func dataReadDryRun(r *common.RuntimeContext) *common.DryRunAPI {
	dry := common.NewDryRunAPI()
	extent, err := dataRange(r.Str("range"))
	if err != nil {
		return dry
	}
	for _, region := range dataRanges(extent) {
		body, _ := buildToolBody("get_cell_ranges", dataReadInput(r.Str("spreadsheet-token"), r.Str("sheet-id"), region))
		dry.POST(toolInvokePath(r.Str("spreadsheet-token"), ToolKindRead)).Desc("read complete native cells in " + region).Body(body)
	}
	return dry.Set("complete_extent", r.Str("range"))
}

func nativeFormulaPresent(cell nativeCell) bool {
	var formula string
	return json.Unmarshal(cell["formula"], &formula) == nil && formula != ""
}
