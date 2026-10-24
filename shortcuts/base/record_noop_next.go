// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

const recordComparisonTip = "--skip-unchanged requires the additional base:record:read scope. It compares submitted cells before writing; the read and write are not atomic and concurrent edits can still be overwritten."

func recordSkipUnchangedFlag() common.Flag {
	return common.Flag{Name: "skip-unchanged", Type: "bool", Desc: "read records first and skip updates whose submitted cells exactly match (requires base:record:read)"}
}

func validateRecordUpsert(runtime *common.RuntimeContext) error {
	if runtime.Bool("skip-unchanged") && runtime.Str("record-id") == "" {
		return baseFlagErrorf("--skip-unchanged requires --record-id; it cannot be used when creating a record")
	}
	return validateRecordJSON(runtime)
}

func validateRecordBatchUpdate(runtime *common.RuntimeContext) error {
	if !runtime.Bool("skip-unchanged") {
		return validateRecordJSON(runtime)
	}
	body, err := parseJSONObject(newParseCtx(runtime), runtime.Str("json"), "json")
	if err != nil {
		return err
	}
	_, err = recordUpdateMaps(body)
	return err
}

func recordUpdateMaps(body map[string]interface{}) (map[string]map[string]interface{}, error) {
	updates, ok := body["update_records"].(map[string]interface{})
	if !ok || len(updates) == 0 || len(updates) > maxRecordSelectionCount {
		return nil, baseFlagErrorf("--json update_records must be an object containing 1 to %d record field maps", maxRecordSelectionCount)
	}
	result := make(map[string]map[string]interface{}, len(updates))
	for id, value := range updates {
		fields, ok := value.(map[string]interface{})
		if strings.TrimSpace(id) == "" || !ok || fields == nil {
			return nil, baseFlagErrorf("--json update_records keys must be non-empty record IDs and values must be field map objects")
		}
		result[id] = fields
	}
	return result, nil
}

// recordComparisonMatrix is a projection of the backend read shape. Cells stay
// raw until decoded with UseNumber; null is a present value, not an omitted cell.
type recordComparisonMatrix struct {
	Fields         []string            `json:"fields"`
	FieldIDs       []string            `json:"field_id_list"`
	RecordIDs      []string            `json:"record_id_list"`
	Rows           [][]json.RawMessage `json:"data"`
	RecordNotFound []string            `json:"record_not_found"`
	IgnoredFields  []json.RawMessage   `json:"ignored_fields"`
	HasMore        bool                `json:"has_more"`
}

func recordComparisonError(reason string) *errs.InternalError {
	return errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot safely compare records: %s", reason).
		WithHint("Read the requested records and fields again; no update was sent.")
}

func recordComparisonIDs(updates map[string]map[string]interface{}) []string {
	ids := make([]string, 0, len(updates))
	for id := range updates {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func unchangedRecordIDs(runtime *common.RuntimeContext, updates map[string]map[string]interface{}) ([]string, error) {
	data, err := baseV3ReadCall(runtime, "POST", baseV3Path("bases", runtime.Str("base-token"), "tables", baseTableID(runtime), "records", "batch_get"), nil,
		map[string]interface{}{"record_id_list": recordComparisonIDs(updates)})
	if err != nil {
		return nil, err
	}
	return compareRecordUpdates(data, updates)
}

func compareRecordUpdates(data map[string]interface{}, updates map[string]map[string]interface{}) ([]string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, recordComparisonError("unencodable record matrix").WithCause(err)
	}
	var matrix recordComparisonMatrix
	if err := json.Unmarshal(raw, &matrix); err != nil {
		return nil, recordComparisonError("invalid record matrix types").WithCause(err)
	}
	if matrix.Fields == nil || matrix.FieldIDs == nil || len(matrix.Fields) != len(matrix.FieldIDs) ||
		matrix.RecordIDs == nil || matrix.Rows == nil || len(matrix.Rows) != len(matrix.RecordIDs) ||
		len(matrix.RecordNotFound) > 0 || len(matrix.IgnoredFields) > 0 || matrix.HasMore {
		return nil, recordComparisonError("missing or incomplete fields or records")
	}
	columns := make(map[string]int, len(matrix.Fields)*2)
	fieldIDs := make(map[string]bool, len(matrix.Fields))
	for i, name := range matrix.Fields {
		id := matrix.FieldIDs[i]
		if name == "" || id == "" || fieldIDs[id] {
			return nil, recordComparisonError("missing or duplicate field identifiers")
		}
		fieldIDs[id] = true
		for _, key := range []string{name, id} {
			if previous, exists := columns[key]; exists && previous != i {
				columns[key] = -1 // A name can collide with another name or ID.
			} else {
				columns[key] = i
			}
		}
	}
	rows := make(map[string][]json.RawMessage, len(matrix.Rows))
	for i, id := range matrix.RecordIDs {
		if _, duplicate := rows[id]; id == "" || duplicate || len(matrix.Rows[i]) != len(matrix.Fields) {
			return nil, recordComparisonError("missing, duplicate, or incomplete record rows")
		}
		if _, requested := updates[id]; !requested {
			return nil, recordComparisonError("unexpected record in comparison response")
		}
		rows[id] = matrix.Rows[i]
	}
	unchanged := make([]string, 0, len(updates))
	for _, id := range recordComparisonIDs(updates) {
		row, exists := rows[id]
		if !exists {
			return nil, recordComparisonError("requested record is missing")
		}
		equal := true
		seenColumns := make(map[int]bool)
		for key, submitted := range updates[id] {
			column, exists := columns[key]
			if !exists || column < 0 {
				return nil, baseFlagErrorf("--json field %q cannot be resolved unambiguously from the comparison read", key)
			}
			if seenColumns[column] {
				return nil, baseFlagErrorf("--json refers to the same field by both name and ID")
			}
			seenColumns[column] = true
			var current interface{}
			decoder := json.NewDecoder(bytes.NewReader(row[column]))
			decoder.UseNumber()
			if err := decoder.Decode(&current); err != nil {
				return nil, recordComparisonError("invalid cell value").WithCause(err)
			}
			// No coercion, null/empty normalization, array reordering, or field
			// deletion inference: a conservative false negative just writes again.
			if !reflect.DeepEqual(current, submitted) {
				equal = false
			}
		}
		if equal {
			unchanged = append(unchanged, id)
		}
	}
	return unchanged, nil
}

func recordComparisonDryRun(dry *common.DryRunAPI, ids []string) *common.DryRunAPI {
	return dry.POST("/open-apis/base/v3/bases/:base_token/tables/:table_id/records/batch_get").
		Body(map[string]interface{}{"record_id_list": ids}).
		Desc("Read submitted records for exact comparison before any conditional update").
		Set("comparison", map[string]interface{}{
			"skip_unchanged": true, "performed": false, "requires_scope": "base:record:read",
			"mode": "exact_submitted_cells", "atomic": false,
			"write_condition": "only_records_with_changed_submitted_cells",
		})
}
