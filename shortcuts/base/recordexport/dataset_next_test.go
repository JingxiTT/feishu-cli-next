// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recordexport

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestParseMatrixPreservesSourceJSONValues(t *testing.T) {
	for _, tc := range []struct {
		fieldType string
		values    []any
	}{
		{"text", []any{nil, "", "00123", "名字😀"}},
		{"select", []any{nil, []any{}, []any{"", "00123", "名字😀"}}},
		{"user", []any{nil, []any{}, []any{map[string]any{"id": "00123", "extra": json.Number("9007199254740993")}}}},
		{"checkbox", []any{nil, false, true}},
		{"datetime", []any{nil, "", "2026-08-04 12:30:00", "2026-11-01T01:30:00.123456-04:00", "not a date"}},
		{"created_at", []any{"2026-08-04 12:30:00", ""}},
		{"updated_at", []any{"2026-08-04 12:30:00", ""}},
		{"number", []any{nil, json.Number("9007199254740993"), json.Number("1.2300000000000000001"), json.Number("1e+30")}},
	} {
		t.Run(tc.fieldType, func(t *testing.T) {
			for _, value := range tc.values {
				page, err := ParseMatrix(matrixFixture("Value", "fld_value", tc.fieldType, "rec_1", value))
				if err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				if err := WriteNDJSON(&output, page.Dataset); err != nil {
					t.Fatal(err)
				}
				var row map[string]any
				decoder := json.NewDecoder(&output)
				decoder.UseNumber()
				if err := decoder.Decode(&row); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(row["Value"], value) {
					t.Fatalf("round trip = %#v, want %#v", row["Value"], value)
				}
			}
		})
	}
}

func TestManifestPreservesExactNumericBounds(t *testing.T) {
	page, err := ParseMatrix(map[string]any{
		"timezone": "UTC", "fields": []any{"Number"}, "field_id_list": []any{"fld_number"},
		"field_type_list": []any{"number"}, "record_id_list": []any{"rec_1", "rec_2"},
		"data": []any{[]any{json.Number("9007199254740993")}, []any{json.Number("9007199254740992")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	stats := BuildManifest(page.Dataset, ManifestOptions{}).Columns["Number"].Stats
	if stats.Min != json.Number("9007199254740992") || stats.Max != json.Number("9007199254740993") || stats.Avg == nil || !stats.AvgApproximate {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestManifestNullEmptyAndUnparsedDates(t *testing.T) {
	page, err := ParseMatrix(map[string]any{
		"timezone": "UTC", "fields": []any{"Tags", "When"}, "field_id_list": []any{"fld_tags", "fld_when"},
		"field_type_list": []any{"select", "datetime"}, "record_id_list": []any{"rec_1", "rec_2", "rec_3"},
		"data": []any{[]any{nil, nil}, []any{[]any{}, ""}, []any{[]any{"a", "b"}, "2026-08-04 12:30:00"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := BuildManifest(page.Dataset, ManifestOptions{})
	tags := manifest.Columns["Tags"].Stats
	if *tags.NullCount != 1 || *tags.EmptyCount != 1 || *tags.AvgLength != 1 {
		t.Fatalf("tags = %#v", tags)
	}
	when := manifest.Columns["When"].Stats
	if *when.NullCount != 1 || *when.UnparsedCount != 1 || when.Min != "2026-08-04 12:30:00" || when.Max != when.Min {
		t.Fatalf("when = %#v", when)
	}
}

func TestParseMatrixTracksPaginationPresence(t *testing.T) {
	fixture := matrixFixture("Name", "fld_name", "text", "rec_1", "Alice")
	page, err := ParseMatrix(fixture)
	if err != nil || page.HasMoreKnown {
		t.Fatalf("absent has_more = %#v, %v", page, err)
	}
	fixture["has_more"] = false
	page, err = ParseMatrix(fixture)
	if err != nil || !page.HasMoreKnown || page.HasMore {
		t.Fatalf("explicit has_more = %#v, %v", page, err)
	}
}
