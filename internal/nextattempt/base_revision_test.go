// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestBaseCreateRevisionMetadataDoesNotObscureCompleteIDs(t *testing.T) {
	for _, test := range []struct {
		name, data string
		complete   bool
	}{
		{"revision", `{"record_id_list":["one","two"],"rev":42}`, true},
		{"zero revision", `{"record_id_list":["one","two"],"rev":0}`, true},
		{"precise revision", `{"record_id_list":["one","two"],"rev":9007199254740993}`, true},
		{"negative revision", `{"record_id_list":["one","two"],"rev":-1}`, false},
		{"string revision", `{"record_id_list":["one","two"],"rev":"42"}`, false},
		{"fractional revision", `{"record_id_list":["one","two"],"rev":1.5}`, false},
		{"short IDs", `{"record_id_list":["one"],"rev":42}`, false},
		{"duplicate IDs", `{"record_id_list":["one","one"],"rev":42}`, false},
		{"ignored fields", `{"record_id_list":["one","two"],"rev":42,"ignored_fields":["computed"]}`, false},
		{"async metadata", `{"record_id_list":["one","two"],"rev":42,"task_id":"pending"}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest("POST", "https://fixture.invalid/records", strings.NewReader(`{"create_records":[{"Name":"first"},{"Name":"second"}]}`))
			if err != nil {
				t.Fatal(err)
			}
			ids := completeBaseCreate(req, []byte(test.data))
			if (ids != nil) != test.complete || test.complete && !reflect.DeepEqual(ids, []string{"one", "two"}) {
				t.Fatalf("complete IDs = %v, want complete=%v", ids, test.complete)
			}
		})
	}
}
