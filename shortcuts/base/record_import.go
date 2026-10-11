// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/shortcuts/common"
)

var BaseRecordImport = common.Shortcut{
	Service: "base", Command: "+record-import", Description: "Append records from a complete typed Sheets artifact with explicit lossless field mapping",
	Risk: "write", Scopes: []string{"base:field:read", "base:record:create"}, AuthTypes: authTypes(),
	Flags: []common.Flag{baseTokenFlag(true), tableRefFlag(true),
		{Name: "artifact", Required: true, Input: []string{common.File, common.Stdin}, Desc: "complete Sheets --save-result artifact JSON, @file or -"},
		{Name: "mapping", Required: true, Input: []string{common.File}, Desc: `all-column mapping array: [{"column":0,"field_id":"fld_name","type":"text"}]`},
		{Name: "output", Desc: "exclusive JSON report file for created IDs; stdout keeps counts and reference"},
	},
	Tips: []string{"This appends new records every invocation; it does not match business keys or delete destination data. Batch size is 200 and source maximum is 10,000 rows.", "Only unformatted native text, number and checkbox values convert without loss. Formula, rich, date-format and identity cells are rejected before any write; explicit empty text is rejected because Base stores it as null; null and omitted cells remain distinct. Number decimals must round-trip through native Base double storage.", "The artifact must match the currently verified user and tenant. Destination schemas are checked first but concurrent changes are not locked. Unknown batch outcomes stop the import; inspect receipts and read records before deciding recovery."},
	Validate: func(_ context.Context, r *common.RuntimeContext) error {
		_, err := parseRecordImport(r.Str("artifact"), r.Str("mapping"))
		if err != nil {
			return err
		}
		_, err = recordImportOutputPath(r)
		return err
	},
	DryRun: func(_ context.Context, r *common.RuntimeContext) *common.DryRunAPI {
		imp, _ := parseRecordImport(r.Str("artifact"), r.Str("mapping"))
		return common.NewDryRunAPI().POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/batch_create").Set("base_token", r.Str("base-token")).Set("table_id", baseTableID(r)).Set("records_count", len(imp.records)).Set("batch_size", 200).Set("batches", (len(imp.records)+199)/200).Set("preflight", "verify artifact identity and destination field schemas before all writes")
	},
	Execute: func(_ context.Context, r *common.RuntimeContext) error { return executeRecordImport(r) },
}

type recordImportReport struct {
	Status           string                 `json:"status"`
	BaseToken        string                 `json:"base_token"`
	TableID          string                 `json:"table_id"`
	Requested        int                    `json:"requested"`
	Created          int                    `json:"created"`
	CompletedBatches int                    `json:"completed_batches"`
	NextRow          int                    `json:"first_unconfirmed_row"`
	RecordIDs        []string               `json:"record_ids"`
	Binding          common.ArtifactBinding `json:"binding"`
	Response         json.RawMessage        `json:"unresolved_response,omitempty"`
	OperationError   json.RawMessage        `json:"operation_error,omitempty"`
}

func recordImportOutputPath(r *common.RuntimeContext) (string, error) {
	path := strings.TrimSpace(r.Str("output"))
	if path == "" {
		return "", nil
	}
	fio := r.FileIO()
	if _, ok := fio.(fileio.ExclusiveFileIO); !ok {
		return "", errs.NewValidationError(errs.SubtypeFailedPrecondition, "--output requires atomic exclusive FileIO").WithParam("--output")
	}
	resolved, err := fio.ResolvePath(path)
	if err != nil {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid output path").WithParam("--output").WithCause(err)
	}
	if _, err = fio.Stat(resolved); err == nil {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "output already exists").WithParam("--output")
	} else if !errors.Is(err, fs.ErrNotExist) {
		return "", errs.NewInternalError(errs.SubtypeFileIO, "cannot inspect import output").WithCause(err)
	}
	return resolved, nil
}

func executeRecordImport(r *common.RuntimeContext) error {
	imp, err := parseRecordImport(r.Str("artifact"), r.Str("mapping"))
	if err != nil {
		return err
	}
	current, err := r.ResolveArtifactBinding()
	if err != nil {
		return err
	}
	if err = common.ValidateArtifactBinding(imp.artifact.Binding, current, "--artifact", true); err != nil {
		return err
	}
	outputPath, err := recordImportOutputPath(r)
	if err != nil {
		return err
	}
	for _, column := range imp.columns {
		data, err := baseV3Call(r, "GET", baseV3Path("bases", r.Str("base-token"), "tables", baseTableID(r), "fields", column.FieldID), nil, nil)
		if err != nil {
			return err
		}
		// Field configuration is projected at the boundary; formula/style details
		// cannot accidentally be interpreted as a writable scalar type.
		var field struct {
			ID           string          `json:"id"`
			Type         string          `json:"type"`
			DefaultValue json.RawMessage `json:"default_value"`
		}
		raw, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid field configuration").WithCause(marshalErr)
		}
		if json.Unmarshal(raw, &field) != nil || field.ID != column.FieldID || field.Type != column.Type {
			return importInvalid("--mapping", "destination field ID/type does not match the explicit mapping")
		}
		if len(field.DefaultValue) > 0 && string(bytes.TrimSpace(field.DefaultValue)) != "null" {
			for _, record := range imp.records {
				if _, present := record[column.FieldID]; !present {
					return importInvalid("--mapping", "destination default would replace an omitted source cell")
				}
			}
		}
	}
	report := recordImportReport{Status: "complete", BaseToken: r.Str("base-token"), TableID: baseTableID(r), Requested: len(imp.records), RecordIDs: []string{}, Binding: current}
	var operationErr error
	confirmedIDs := make(map[string]bool, len(imp.records))
	for start := 0; start < len(imp.records); start += 200 {
		end := min(start+200, len(imp.records))
		data, err := baseV3Call(r, "POST", baseV3Path("bases", r.Str("base-token"), "tables", baseTableID(r), "records", "batch_create"), nil, map[string]interface{}{"create_records": imp.records[start:end]})
		if err != nil {
			report.Status = "unknown"
			operationErr = err
			break
		}
		ids, err := recordImportBatchIDs(data, end-start, confirmedIDs)
		if err != nil {
			report.Status = "unknown"
			report.Response, _ = json.Marshal(data)
			operationErr = err
			break
		}
		for _, id := range ids {
			confirmedIDs[id] = true
		}
		report.RecordIDs = append(report.RecordIDs, ids...)
		report.Created += len(ids)
		report.CompletedBatches++
		report.NextRow = end
		if summary := r.Factory.AgentAttemptSummary(); summary.Unknown > 0 || summary.Unapplied > 0 || summary.PersistenceFailed {
			report.Status = "unknown"
			operationErr = errs.NewInternalError(errs.SubtypeStorage, "import mutation evidence is incomplete").WithHint("Inspect mutation receipts and read destination records before deciding whether to retry")
			break
		}
	}
	if operationErr != nil {
		if typed, ok := errs.UnwrapTypedError(operationErr); ok {
			report.OperationError, _ = json.Marshal(typed)
		}
	}
	if err := writeRecordImportReport(r, outputPath, report); err != nil {
		if operationErr != nil {
			return errs.NewInternalError(errs.SubtypeFileIO, "batch failed and import report could not be saved; original operation evidence remains on stdout").WithCause(errors.Join(operationErr, err))
		}
		return err
	}
	return operationErr
}

func recordImportBatchIDs(data map[string]interface{}, count int, confirmedIDs map[string]bool) ([]string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid import batch response").WithCause(err)
	}
	var response struct {
		IDs     []string          `json:"record_id_list"`
		Ignored []json.RawMessage `json:"ignored_fields"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&response) != nil || len(response.IDs) != count || len(response.Ignored) > 0 {
		return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "batch did not confirm all requested record IDs and fields; do not replay without reconciliation")
	}
	seen := map[string]bool{}
	for _, id := range response.IDs {
		if id == "" || seen[id] || confirmedIDs[id] {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "batch returned missing record IDs or IDs duplicated within this import; do not replay without reconciliation")
		}
		seen[id] = true
	}
	return response.IDs, nil
}

func writeRecordImportReport(r *common.RuntimeContext, path string, report recordImportReport) error {
	if path == "" {
		if report.Status != "complete" {
			_ = r.OutPartialFailure(report, nil)
		} else {
			r.Out(report, nil)
		}
		return nil
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot encode import report").WithCause(err)
	}
	saved, err := r.FileIO().(fileio.ExclusiveFileIO).SaveExclusive(path, fileio.SaveOptions{ContentType: "application/json", ContentLength: int64(len(raw))}, bytes.NewReader(raw))
	if err != nil {
		_ = r.OutPartialFailure(report, nil)
		return errs.NewInternalError(errs.SubtypeFileIO, "import effects may have occurred, but report could not be saved; full observed report is on stdout").WithCause(err)
	}
	if saved == nil {
		_ = r.OutPartialFailure(report, nil)
		return errs.NewInternalError(errs.SubtypeFileIO, "import report save returned no file information; inspect observed effects on stdout")
	}
	result := map[string]interface{}{"status": report.Status, "base_token": report.BaseToken, "table_id": report.TableID, "requested": report.Requested, "created": report.Created, "completed_batches": report.CompletedBatches, "first_unconfirmed_row": report.NextRow, "report": map[string]interface{}{"ref": path, "bytes": saved.Size()}, "binding": report.Binding}
	if report.Status != "complete" {
		result["unresolved_response"] = report.Response
		result["operation_error"] = report.OperationError
		_ = r.OutPartialFailure(result, nil)
	} else {
		r.Out(result, nil)
	}
	return nil
}
