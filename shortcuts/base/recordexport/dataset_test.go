// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recordexport

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseMatrixPreservesTypedRows(t *testing.T) {
	page, err := ParseMatrix(map[string]any{
		"rev":           json.Number("42"),
		"timezone":      "Asia/Shanghai",
		"fields":        []any{"Name", "Tags", "Owner", "Due", "Score", "Done", "Place", "Formula"},
		"field_id_list": []any{"fld_name", "fld_tags", "fld_owner", "fld_due", "fld_score", "fld_done", "fld_place", "fld_formula"},
		"field_type_list": []any{
			"text", "select", "user", "datetime", "number", "checkbox", "location", "formula",
		},
		"record_id_list": []any{"rec_1", "rec_2"},
		"data": []any{
			[]any{"Alice", nil, nil, "2026-08-04 12:30:00", 12.5, false, map[string]any{"lng": 1.0, "lat": 2.0, "full_address": "北京市"}, "ok"},
			[]any{nil, []any{"P0"}, []any{map[string]any{"id": "ou_1", "name": "Bob"}}, nil, nil, nil, nil, nil},
		},
		"has_more": true,
	})
	if err != nil {
		t.Fatalf("ParseMatrix() error = %v", err)
	}
	if !page.HasMore {
		t.Fatal("HasMore = false, want true")
	}
	if page.Rev == nil || *page.Rev != 42 {
		t.Fatalf("Rev = %#v, want 42", page.Rev)
	}
	if got := page.Dataset.Columns[0].PhysicalType(); got != "string" {
		t.Fatalf("record_id physical type = %q", got)
	}
	if got := page.Dataset.Columns[2].PhysicalType(); got != "array<string>|null" {
		t.Fatalf("Tags physical type = %q", got)
	}
	if got := page.Dataset.Columns[3].PhysicalType(); got != "array<struct<id string, name string>>|null" {
		t.Fatalf("Owner physical type = %q", got)
	}
	if got := page.Dataset.Columns[4].PhysicalType(); got != "string|null" {
		t.Fatalf("Due physical type = %q", got)
	}
	if got := page.Dataset.Columns[6].PhysicalType(); got != "boolean|null" {
		t.Fatalf("Done physical type = %q", got)
	}
	if got := page.Dataset.Columns[7].PhysicalType(); got != "struct<lng number, lat number, full_address string>|null" {
		t.Fatalf("Place physical type = %q", got)
	}
	first := page.Dataset.Records[0].Values
	if first[2] != nil {
		t.Fatalf("Tags = %#v, want null", first[2])
	}
	if first[3] != nil {
		t.Fatalf("Owner = %#v, want null", first[3])
	}
	if got := first[4]; got != "2026-08-04 12:30:00" {
		t.Fatalf("Due = %#v", got)
	}
	if got := first[6]; got != false {
		t.Fatalf("Done = %#v, want false", got)
	}
	place := first[7].(map[string]any)
	if got := place["full_address"]; got != "北京市" {
		t.Fatalf("Place.full_address = %#v", got)
	}
	second := page.Dataset.Records[1].Values
	if second[1] != nil || second[4] != nil || second[5] != nil {
		t.Fatalf("nullable scalars = %#v", second)
	}
	if got := second[6]; got != nil {
		t.Fatalf("Done = %#v, want null", got)
	}

	var output bytes.Buffer
	if err := WriteNDJSON(&output, page.Dataset); err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	var secondRow map[string]any
	if err := json.Unmarshal(lines[1], &secondRow); err != nil {
		t.Fatal(err)
	}
	if got := secondRow["Done"]; got != nil {
		t.Fatalf("NDJSON Done = %#v, want null", got)
	}
}

func TestNormalizeDateTimePreservesUpstreamOffset(t *testing.T) {
	value := "2026-11-01T01:30:00.123456-04:00"
	got, err := normalizeDateTime(value, "America/New_York")
	if err != nil {
		t.Fatalf("normalizeDateTime() error = %v", err)
	}
	if got != value {
		t.Fatalf("normalizeDateTime() = %q, want unchanged %q", got, value)
	}
}

func TestManifestDeclaresKnownObjectStructsWithoutRewritingRows(t *testing.T) {
	page, err := ParseMatrix(map[string]any{
		"timezone": "UTC",
		"fields": []any{
			"Users", "Chats", "Links", "Files", "Creator", "Updater",
		},
		"field_id_list": []any{
			"fld_users", "fld_chats", "fld_links", "fld_files", "fld_creator", "fld_updater",
		},
		"field_type_list": []any{
			"user", "group_chat", "link", "attachment", "created_by", "updated_by",
		},
		"record_id_list": []any{"rec_1"},
		"data": []any{[]any{
			[]any{map[string]any{"id": "ou_1"}},
			[]any{map[string]any{"id": "oc_1"}},
			[]any{map[string]any{"id": "rec_link"}},
			[]any{map[string]any{"file_token": "box_1"}},
			[]any{map[string]any{"id": "ou_creator"}},
			[]any{map[string]any{"id": "ou_updater"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	wantTypes := map[string]string{
		"Users":   "array<struct<id string, name string>>|null",
		"Chats":   "array<struct<id string, name string>>|null",
		"Links":   "array<struct<id string>>|null",
		"Files":   "array<struct<file_token string, size number, name string>>|null",
		"Creator": "array<struct<id string, name string>>|null",
		"Updater": "array<struct<id string, name string>>|null",
	}
	manifest := BuildManifest(page.Dataset, ManifestOptions{
		BaseToken: "base_x", TableID: "tbl_x", PageCount: 1,
		RecordFile: "/tmp/out.ndjson", ManifestFile: "/tmp/out.manifest.json",
	})
	for name, want := range wantTypes {
		if got := manifest.Columns[name].PhysicalType; got != want {
			t.Errorf("%s physical type = %q, want %q", name, got, want)
		}
	}

	users := page.Dataset.Records[0].Values[1].([]any)
	if _, exists := users[0].(map[string]any)["name"]; exists {
		t.Fatalf("runtime Users value was rewritten: %#v", users)
	}
	files := page.Dataset.Records[0].Values[4].([]any)
	if _, exists := files[0].(map[string]any)["size"]; exists {
		t.Fatalf("runtime Files value was rewritten: %#v", files)
	}
}

func TestParseMatrixRejectsColumnCollisions(t *testing.T) {
	for _, tc := range []struct {
		name       string
		names, ids []any
	}{
		{"system name", []any{"record_id", "Name"}, []any{"fld_shadow", "fld_name"}},
		{"duplicate name", []any{"Name", "Name"}, []any{"fld_one", "fld_two"}},
		{"duplicate ID", []any{"Name", "Other"}, []any{"fld_one", "fld_one"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMatrix(map[string]any{
				"timezone": "UTC", "fields": tc.names, "field_id_list": tc.ids,
				"field_type_list": []any{"text", "text"},
				"record_id_list":  []any{"rec_real"}, "data": []any{[]any{"user-value", "Alice"}},
			})
			var matrixErr *MatrixError
			if !errors.As(err, &matrixErr) {
				t.Fatalf("error = %v, want MatrixError", err)
			}
		})
	}
}

func TestDatasetAppendRejectsSchemaChange(t *testing.T) {
	first, err := ParseMatrix(matrixFixture("Name", "fld_name", "text", "rec_1", "Alice"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseMatrix(matrixFixture("Name", "fld_other", "text", "rec_2", "Bob"))
	if err != nil {
		t.Fatal(err)
	}
	dataset := first.Dataset
	if err := dataset.AppendPage(second); err == nil {
		t.Fatal("AppendPage() error = nil, want schema change")
	}
}

func TestManifestExamplesAndNDJSON(t *testing.T) {
	longText := strings.Repeat("长", 150)
	page, err := ParseMatrix(map[string]any{
		"timezone":        "UTC",
		"fields":          []any{"Long", "EmptyUsers", "Count"},
		"field_id_list":   []any{"fld_long", "fld_empty", "fld_count"},
		"field_type_list": []any{"text", "user", "number"},
		"record_id_list":  []any{"rec_1"},
		"data":            []any{[]any{longText, nil, 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := BuildManifest(page.Dataset, ManifestOptions{
		BaseToken: "base_x", TableID: "tbl_x", PageCount: 1,
		RecordFile: "/tmp/out.ndjson", ManifestFile: "/tmp/out.manifest.json",
	})
	if !manifest.Columns["Long"].ExampleTruncated {
		t.Fatal("Long.example_truncated = false")
	}
	if manifest.Columns["EmptyUsers"].Example != nil || manifest.Columns["EmptyUsers"].Hint == "" {
		t.Fatalf("EmptyUsers metadata = %#v", manifest.Columns["EmptyUsers"])
	}
	if got := manifest.Columns["EmptyUsers"].PhysicalType; got != "array<struct<id string, name string>>|null" {
		t.Fatalf("EmptyUsers physical type = %q", got)
	}
	if got := manifest.Columns["Count"].Example; got != 0 {
		t.Fatalf("Count.example = %#v, want 0", got)
	}
	if got := *manifest.Columns["Long"].Stats.MaxLength; got != 150 {
		t.Fatalf("Long.stats.max_length = %d, want 150", got)
	}
	if stats := manifest.Columns["EmptyUsers"].Stats; *stats.EmptyCount != 0 || *stats.NullCount != 1 {
		t.Fatalf("EmptyUsers.stats = %#v, want one null and zero empty arrays", stats)
	}
	if got := *manifest.Columns["Count"].Stats.Avg; got != 0 {
		t.Fatalf("Count.stats.avg = %v, want 0", got)
	}

	var output bytes.Buffer
	if err := WriteNDJSON(&output, page.Dataset); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &row); err != nil {
		t.Fatalf("unmarshal ndjson: %v\n%s", err, output.String())
	}
	if row["record_id"] != "rec_1" {
		t.Fatalf("row = %#v", row)
	}
	if row["EmptyUsers"] != nil {
		t.Fatalf("EmptyUsers = %#v", row["EmptyUsers"])
	}
}

func TestManifestColumnStatsByFieldType(t *testing.T) {
	page, err := ParseMatrix(map[string]any{
		"timezone": "UTC",
		"fields": []any{
			"Text", "Number", "When", "Done", "Place", "Tags",
		},
		"field_id_list": []any{
			"fld_text", "fld_number", "fld_when", "fld_done", "fld_place", "fld_tags",
		},
		"field_type_list": []any{
			"text", "number", "datetime", "checkbox", "location", "select",
		},
		"record_id_list": []any{"rec_1", "rec_22", "rec_333"},
		"data": []any{
			[]any{"短", 10, "2026-08-01T10:00:00+08:00", true, map[string]any{"lng": 1.0, "lat": 2.0, "full_address": "A"}, []any{"a", "b"}},
			[]any{"最长值", 20.0, "2026-08-03T10:00:00+08:00", false, nil, nil},
			[]any{nil, nil, nil, nil, nil, []any{"c"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	manifest := BuildManifest(page.Dataset, ManifestOptions{
		BaseToken: "base_x", TableID: "tbl_x", PageCount: 1,
		RecordFile: "/tmp/out.ndjson", ManifestFile: "/tmp/out.manifest.json",
	})

	if got := *manifest.Columns["record_id"].Stats.MaxLength; got != 7 {
		t.Errorf("record_id.stats.max_length = %d, want 7", got)
	}
	textStats := manifest.Columns["Text"].Stats
	if *textStats.NullCount != 1 || *textStats.MaxLength != 3 {
		t.Errorf("Text.stats = %#v", textStats)
	}
	numberStats := manifest.Columns["Number"].Stats
	if *numberStats.NullCount != 1 || numberStats.Min != 10 || numberStats.Max != 20.0 || *numberStats.Avg != 15.0 {
		t.Errorf("Number.stats = %#v", numberStats)
	}
	whenStats := manifest.Columns["When"].Stats
	if *whenStats.NullCount != 1 || whenStats.Min != "2026-08-01T10:00:00+08:00" || whenStats.Max != "2026-08-03T10:00:00+08:00" {
		t.Errorf("When.stats = %#v", whenStats)
	}
	doneStats := manifest.Columns["Done"].Stats
	if *doneStats.TrueCount != 1 || *doneStats.NullCount != 1 {
		t.Errorf("Done.stats = %#v", doneStats)
	}
	placeStats := manifest.Columns["Place"].Stats
	if *placeStats.NullCount != 2 {
		t.Errorf("Place.stats = %#v", placeStats)
	}
	tagStats := manifest.Columns["Tags"].Stats
	if *tagStats.EmptyCount != 0 || *tagStats.NullCount != 1 || *tagStats.MaxLength != 2 || *tagStats.AvgLength != 1.5 {
		t.Errorf("Tags.stats = %#v", tagStats)
	}
}

func TestParseMatrixRejectsParallelArrayMismatch(t *testing.T) {
	_, err := ParseMatrix(map[string]any{
		"timezone":        "UTC",
		"fields":          []any{"Name"},
		"field_id_list":   []any{},
		"field_type_list": []any{"text"},
		"record_id_list":  []any{},
		"data":            []any{},
	})
	if err == nil {
		t.Fatal("ParseMatrix() error = nil")
	}
}

func TestParseMatrixRejectsInvalidRev(t *testing.T) {
	fixture := matrixFixture("Name", "fld_name", "text", "rec_1", "Alice")
	fixture["rev"] = "42"
	if _, err := ParseMatrix(fixture); err == nil {
		t.Fatal("ParseMatrix() error = nil, want invalid rev")
	}
}

func matrixFixture(name, id, fieldType, recordID string, value any) map[string]any {
	return map[string]any{
		"timezone":        "UTC",
		"fields":          []any{name},
		"field_id_list":   []any{id},
		"field_type_list": []any{fieldType},
		"record_id_list":  []any{recordID},
		"data":            []any{[]any{value}},
	}
}

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
