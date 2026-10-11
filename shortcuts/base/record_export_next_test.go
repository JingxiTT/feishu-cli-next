// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestRecordListNDJSONRejectsCollisionsWithoutPublishingFiles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		fields, ids []any
	}{
		{"system name", []any{"record_id", "Name"}, []any{"fld_one", "fld_two"}},
		{"duplicate name", []any{"Name", "Name"}, []any{"fld_one", "fld_two"}},
		{"duplicate ID", []any{"Name", "Other"}, []any{"fld_one", "fld_one"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			withBaseWorkingDir(t, dir)
			factory, stdout, registry := newExecuteFactory(t)
			registry.Register(&httpmock.Stub{Method: "GET", URL: "limit=1&offset=0", Body: map[string]any{"code": 0, "data": map[string]any{
				"timezone": "UTC", "fields": tc.fields, "field_id_list": tc.ids,
				"field_type_list": []any{"text", "text"}, "record_id_list": []any{"rec_1"},
				"data": []any{[]any{"first", "second"}}, "has_more": false,
			}}})
			err := runShortcut(t, BaseRecordList, []string{"+record-list", "--base-token", "app_x", "--table-id", "tbl_x", "--limit", "1", "--output", "collision.ndjson"}, factory, stdout)
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Subtype != errs.SubtypeInvalidResponse || errors.Unwrap(err) == nil {
				t.Fatalf("problem = %#v, error = %v", problem, err)
			}
			for _, filename := range []string{"collision.ndjson", "collision.manifest.json"} {
				if _, statErr := os.Stat(filepath.Join(dir, filename)); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("%s exists: %v", filename, statErr)
				}
			}
		})
	}
}
