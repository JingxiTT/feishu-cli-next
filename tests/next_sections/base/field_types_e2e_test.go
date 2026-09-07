// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package baseprocess

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

// This covers the native field schema, beyond convenience type aliases. Bodies
// follow the pinned field-schema, formula and lookup reference contracts.
func TestEveryNativeFieldTypePreservesWritableConfiguration(t *testing.T) {
	definitions := []string{
		`{"type":"text","name":"Text","description":"说明 [link](https://example.invalid)","style":{"type":"plain"},"default_value":""}`,
		`{"type":"text","name":"Phone","style":{"type":"phone"},"default_value":"+8613800000000"}`,
		`{"type":"text","name":"URL","style":{"type":"url"},"default_value":"https://example.invalid"}`,
		`{"type":"text","name":"Email","style":{"type":"email"},"default_value":"synthetic@example.invalid"}`,
		`{"type":"text","name":"Barcode","style":{"type":"barcode"}}`,
		`{"type":"number","name":"Number","style":{"type":"plain","precision":4,"percentage":false,"thousands_separator":true},"default_value":0.1}`,
		`{"type":"number","name":"Currency","style":{"type":"currency","precision":2,"currency_code":"USD"},"default_value":null}`,
		`{"type":"number","name":"Progress","style":{"type":"progress","percentage":true,"color":"Blue"}}`,
		`{"type":"number","name":"Rating","style":{"type":"rating","icon":"heart","min":1,"max":5}}`,
		`{"type":"select","name":"Select","multiple":false,"options":[{"name":"Todo","hue":"Blue","lightness":"Light"}],"default_value":["Todo"]}`,
		`{"type":"select","name":"Multi","multiple":true,"options":[{"name":"A"},{"name":"B"}],"default_value":[]}`,
		`{"type":"select","name":"Dynamic","multiple":true,"dynamic_options_source":{"table_id":"tbl_source","field_id":"fld_source"}}`,
		`{"type":"datetime","name":"Date","style":{"format":"yyyy/MM/dd HH:mm"},"default_value":"2026-01-01T12:30:00+08:00"}`,
		`{"type":"created_at","name":"Created","style":{"format":"yyyy/MM/dd"}}`,
		`{"type":"updated_at","name":"Updated","style":{"format":"yyyy/MM/dd HH:mm"}}`,
		`{"type":"user","name":"Users","multiple":true,"default_value":[{"id":"ou_synthetic"}]}`,
		`{"type":"group_chat","name":"Chats","multiple":false}`,
		`{"type":"created_by","name":"Creator"}`,
		`{"type":"updated_by","name":"Editor"}`,
		`{"type":"link","name":"Link","link_table":"tbl_source","bidirectional":true,"bidirectional_link_field_name":"Back"}`,
		`{"type":"formula","name":"Formula","expression":"SUM([Number])"}`,
		`{"type":"lookup","name":"Lookup","from":"Source","select":"Name","where":{"conjunction":"and","conditions":[{"field_name":"Name","operator":"is","value":["A"]}]},"aggregate":"raw_value"}`,
		`{"type":"auto_number","name":"Serial","style":{"rules":[{"type":"text","text":"TASK-"},{"type":"created_time","date_format":"yyyyMMdd"},{"type":"incremental_number","length":4}]}}`,
		`{"type":"attachment","name":"Files"}`,
		`{"type":"location","name":"Location"}`,
		`{"type":"checkbox","name":"Done"}`,
		`{"type":"button","name":"Button","button_config":{"title":"Run"}}`,
	}
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := int(calls.Add(1)) - 1
		if i >= len(definitions) {
			t.Errorf("unexpected extra field write")
			http.Error(w, "unexpected", 500)
			return
		}
		if r.Method != "POST" || r.URL.Path != tp+"/fields" {
			t.Errorf("wrong request %s %s", r.Method, r.URL.Path)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		assertJSON(t, string(raw), definitions[i], false)
		var field map[string]json.RawMessage
		if err := json.Unmarshal(raw, &field); err != nil {
			t.Error(err)
		}
		field["id"] = json.RawMessage(fmt.Sprintf(`"fld_%d"`, i))
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": field})
	}))
	raw := []byte("[")
	for i, definition := range definitions {
		if i > 0 {
			raw = append(raw, ',')
		}
		raw = append(raw, definition...)
	}
	raw = append(raw, ']')
	if err := os.WriteFile(filepath.Join(f.WorkDir, "fields.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	result := f.Run(table("field-create", "--json", "@fields.json", "--i-have-read-guide")...)
	if result.ExitCode != 0 || calls.Load() != int32(len(definitions)) {
		t.Fatalf("exit=%d calls=%d stdout=%s stderr=%s", result.ExitCode, calls.Load(), result.Stdout, result.Stderr)
	}
}

func TestFieldGuideRecoveryNamesInstalledExecutableReads(t *testing.T) {
	for _, kind := range []string{"formula", "lookup"} {
		for _, command := range []string{"field-create", "field-update"} {
			t.Run(command+"/"+kind, func(t *testing.T) {
				f := nextfixture.NewFixture(t, nil)
				flags := []string{"--json", fmt.Sprintf(`{"type":%q,"name":"Derived"}`, kind), "--dry-run"}
				if command == "field-update" {
					flags = append(flags, "--field-id", "fld_derived", "--yes")
				}
				result := f.Run(table(command, flags...)...)
				var envelope struct {
					Error struct {
						Param string `json:"param"`
						Hint  string `json:"hint"`
					}
				}
				if err := json.Unmarshal([]byte(result.Stderr), &envelope); err != nil {
					t.Fatalf("error is not JSON: %+v", result)
				}
				expected := "feishu-cli-next skills read lark-base references/lark-base-field-" + kind + ".md"
				if result.ExitCode != 2 || result.Stdout != "" || envelope.Error.Param != "--i-have-read-guide" || !strings.Contains(envelope.Error.Hint, expected) {
					t.Fatalf("unusable recovery: %+v", result)
				}
			})
		}
	}
}
