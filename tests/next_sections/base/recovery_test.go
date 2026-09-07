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

func TestBaseValidationStopsBeforeBusinessRequests(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"empty-schema", scoped("table-create", "--name", "Data", "--fields", "[]")},
		{"malformed-record", table("record-upsert", "--json", "{bad")},
		{"duplicate-selection", table("record-get", "--record-id", "rec_a", "--record-id", "rec_a")},
		{"conflicting-projection", table("record-list", "--field-id", "fld_a", "--fields", "fld_b")},
		{"invalid-limit", table("record-list", "--limit", "0")},
		{"missing-confirmation", table("table-delete")},
		{"partial-position", dash("dashboard-block-update", "--block-id", "blk_fixture", "--position", `{"x":1}`)},
		{"multiple-share-fields", table("form-share-update", "--form-id", "frm_fixture", "--enabled=false", "--require-login")},
		{"internal-workflow-id", table("button-rule-bind", "--field-id", "fld_button", "--workflow-id", "12345")},
		{"empty-aggregation", scoped("data-query", "--dsl", "{}")},
		{"creation-noop", table("record-upsert", "--json", `{"fields":{"Title":"A"}}`, "--skip-unchanged")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			result := f.Run(tc.args...)
			wantExit, wantType := 2, "validation"
			if tc.name == "missing-confirmation" {
				wantExit, wantType = 10, "confirmation"
			}
			if result.ExitCode != wantExit || result.Stdout != "" {
				t.Fatalf("result=%+v", result)
			}
			var envelope struct {
				Error struct{ Type, Subtype string }
			}
			if err := json.Unmarshal([]byte(result.Stderr), &envelope); err != nil {
				t.Fatalf("typed stderr: %v: %s", err, result.Stderr)
			}
			if envelope.Error.Type != wantType || envelope.Error.Subtype == "" {
				t.Fatalf("untyped error: %s", result.Stderr)
			}
		})
	}
}

func TestBaseBackendFailuresAndUnknownMutation(t *testing.T) {
	for _, tc := range []struct {
		name, body, subtype string
		exit                int
		lost                bool
	}{
		{"gateway-rejected", `{"code":99991672,"msg":"synthetic app missing scope"}`, "app_scope_not_applied", 20, false},
		{"permission", `{"code":800020304,"msg":"synthetic denied"}`, "permission_denied", 22, false},
		{"not-found", `{"code":800030110,"msg":"synthetic missing"}`, "not_found", 22, false},
		{"invalid-response", `not-json`, "invalid_response", 22, false},
		{"lost-response", "", "", 22, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != tp+"/records" {
					t.Errorf("request=%s %s", r.Method, r.URL)
				}
				raw, _ := io.ReadAll(r.Body)
				assertJSON(t, string(raw), `{"Title":"Synthetic"}`, false)
				if tc.lost {
					c, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = c.Close()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			}))
			result := f.Run(table("record-upsert", "--json", `{"Title":"Synthetic"}`)...)
			if result.ExitCode != tc.exit || result.Stdout != "" || calls.Load() != 1 {
				t.Fatalf("calls=%d result=%+v", calls.Load(), result)
			}
			if tc.subtype != "" && !strings.Contains(result.Stderr, `"subtype": "`+tc.subtype+`"`) {
				t.Fatalf("stderr=%s", result.Stderr)
			}
			if tc.name == "gateway-rejected" && !strings.Contains(result.Stderr, `"unapplied":1`) {
				t.Fatalf("missing rejected receipt: %s", result.Stderr)
			}
			if tc.lost && !strings.Contains(result.Stderr, `"unknown":1`) {
				t.Fatalf("missing unknown mutation receipt: %s", result.Stderr)
			}
		})
	}
}

func TestBaseFieldBatchPartialFailureLedger(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if r.Method != "POST" || r.URL.Path != tp+"/fields" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		raw, _ := io.ReadAll(r.Body)
		name := "First"
		if n == 2 {
			name = "Second"
		}
		assertJSON(t, string(raw), fmt.Sprintf(`{"name":%q,"type":"text"}`, name), false)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			_, _ = io.WriteString(w, `{"code":0,"data":{"id":"fld_first","name":"First","type":"text"}}`)
		} else {
			_, _ = io.WriteString(w, `{"code":800020304,"msg":"synthetic denied"}`)
		}
	}))
	result := f.Run(table("field-create", "--json", `[{"name":"First","type":"text"},{"name":"Second","type":"text"},{"name":"Third","type":"text"}]`)...)
	if result.ExitCode == 0 || calls.Load() != 2 {
		t.Fatalf("calls=%d result=%+v", calls.Load(), result)
	}
	assertJSON(t, result.Stdout, `{"summary":{"requested":3,"attempted":2,"created":1,"failed":1,"not_attempted":1},"items":[{"index":0,"status":"created"},{"index":1,"status":"failed","retryable":false,"subtype":"permission_denied"},{"index":2,"status":"not_attempted"}],"next_step":"inspect_items"}`, true)
}

func TestBaseNoopComparesMatrixWithoutWrite(t *testing.T) {
	for _, tc := range []struct {
		name   string
		args   []string
		output string
	}{
		{"upsert", table("record-upsert", "--record-id", "rec_fixture", "--json", `{"Title":"原文","Number":9007199254740993}`, "--skip-unchanged"), `{"record_id":"rec_fixture","updated":false,"unchanged":true}`},
		{"batch", table("record-batch-update", "--json", `{"update_records":{"rec_fixture":{"Title":"原文","Number":9007199254740993}}}`, "--skip-unchanged"), `{"updated":false,"unchanged":true,"unchanged_record_ids":["rec_fixture"]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runScenario(t, scenario{tc.name, tc.args, []exchange{ex("POST", tp+"/records/batch_get", "", `{"record_id_list":["rec_fixture"]}`, `{"fields":["Title","Number"],"field_id_list":["fld_title","fld_num"],"record_id_list":["rec_fixture"],"data":[["原文",9007199254740993]]}`)}, tc.output})
		})
	}
}

func TestBaseNDJSONArtifactPreservesValuesAndRejectsClobber(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" || r.URL.Path != tp+"/records" || r.URL.Query().Get("limit") != "2" || r.URL.Query().Get("offset") != "0" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"timezone":"Asia/Shanghai","rev":42,"fields":["Title","Number","Tags","Empty"],"field_id_list":["fld_title","fld_num","fld_tags","fld_empty"],"field_type_list":["text","number","select","text"],"record_id_list":["rec_a","rec_b"],"data":[["第一行\n第二行",9007199254740993,["甲","乙"],null],["",12.5,[],""]],"has_more":false}}`)
	}))
	command := table("record-list", "--limit", "2", "--output", "export.ndjson")
	result := f.Run(command...)
	if result.ExitCode != 0 {
		t.Fatal(result)
	}
	assertJSON(t, result.Stdout, `{"format":"ndjson","records_count":2,"has_more":false}`, true)
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "export.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("records=%q", raw)
	}
	assertJSON(t, lines[0], `{"record_id":"rec_a","Title":"第一行\n第二行","Number":9007199254740993,"Tags":["甲","乙"],"Empty":null}`, false)
	assertJSON(t, lines[1], `{"record_id":"rec_b","Title":"","Number":12.5,"Tags":[],"Empty":""}`, false)
	again := f.Run(command...)
	if again.ExitCode != 2 || calls.Load() != 2 {
		t.Fatalf("clobber calls=%d result=%+v", calls.Load(), again)
	}
	after, _ := os.ReadFile(filepath.Join(f.WorkDir, "export.ndjson"))
	if string(after) != string(raw) {
		t.Fatal("existing artifact changed")
	}
}

// Selection is explicit evidence of a blocked case, never passing live evidence.
// The suite performs no authentication or network access while prerequisites are absent.
func TestBaseLivePrivateLifecycle(t *testing.T) {
	t.Skip("BLOCKED: no verified pre-creation visibility fixture binds the effective account and tenant, prevents inherited workspace/group/link access, resolves Ying Wu by stable identity for any intentional share, and guarantees cleanup only of run-owned Base/Drive resources; legacy live suites are not authorized substitutes. See docs/sections/base-process.md")
}

func TestBaseReadErrorsRetainTypedMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, body, subtype, kind string
		exit                      int
	}{
		{"permission", `{"code":800020304,"msg":"synthetic denied"}`, "permission_denied", "authorization", 3},
		{"not-found", `{"code":800030110,"msg":"synthetic missing"}`, "not_found", "api", 1},
		{"malformed", `not-json`, "invalid_response", "internal", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != bp {
					t.Errorf("request %s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			}))
			result := f.Run(scoped("base-get")...)
			if result.ExitCode != tc.exit || result.Stdout != "" || calls.Load() != 1 {
				t.Fatalf("result=%+v calls=%d", result, calls.Load())
			}
			var envelope struct {
				Error struct{ Type, Subtype string }
			}
			if err := json.Unmarshal([]byte(result.Stderr), &envelope); err != nil {
				t.Fatal(err, result.Stderr)
			}
			if envelope.Error.Type != tc.kind || envelope.Error.Subtype != tc.subtype {
				t.Fatalf("error=%s", result.Stderr)
			}
		})
	}
}

func TestBaseNDJSONMultiplePagesAndSchemaDrift(t *testing.T) {
	for _, drift := range []bool{false, true} {
		name := "stable"
		if drift {
			name = "schema-drift"
		}
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				offset, count, more := 0, 500, true
				if n == 2 {
					offset, count, more = 500, 1, false
				}
				if r.Method != "GET" || r.URL.Path != tp+"/records" || r.URL.Query().Get("offset") != fmt.Sprint(offset) || r.URL.Query().Get("limit") != fmt.Sprint(count) {
					t.Errorf("request=%s %s", r.Method, r.URL)
				}
				ids := []string{}
				rows := [][]any{}
				for i := 0; i < count; i++ {
					ids = append(ids, fmt.Sprintf("rec_%03d", offset+i))
					rows = append(rows, []any{fmt.Sprintf("第 %d 行", offset+i)})
				}
				fieldName := "Title"
				if drift && n == 2 {
					fieldName = "Renamed"
				}
				payload := map[string]any{"code": 0, "data": map[string]any{"timezone": "Asia/Shanghai", "rev": 42, "fields": []string{fieldName}, "field_id_list": []string{"fld_title"}, "field_type_list": []string{"text"}, "record_id_list": ids, "data": rows, "has_more": more}}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(payload)
			}))
			result := f.Run(table("record-list", "--limit", "501", "--output", "pages.ndjson")...)
			if calls.Load() != 2 {
				t.Fatalf("calls=%d result=%+v", calls.Load(), result)
			}
			path := filepath.Join(f.WorkDir, "pages.ndjson")
			if drift {
				if result.ExitCode != 2 || result.Stdout != "" {
					t.Fatalf("drift result=%+v", result)
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("schema drift wrote artifact: %v", err)
				}
				return
			}
			if result.ExitCode != 0 {
				t.Fatal(result)
			}
			assertJSON(t, result.Stdout, `{"records_count":501,"page_count":2,"has_more":false}`, true)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n")
			if len(lines) != 501 {
				t.Fatalf("rows=%d", len(lines))
			}
			for i, line := range lines {
				assertJSON(t, line, fmt.Sprintf(`{"record_id":"rec_%03d","Title":"第 %d 行"}`, i, i), false)
			}
		})
	}
}

func TestBaseRecordMarkdownDefaultIsReadableAndKeepsIdentity(t *testing.T) {
	for _, command := range []string{"record-list", "record-get", "record-search"} {
		t.Run(command, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				path, method := tp+"/records", "GET"
				if command == "record-get" {
					path, method = tp+"/records/batch_get", "POST"
				}
				if command == "record-search" {
					path, method = tp+"/records/search", "POST"
				}
				if r.Method != method || r.URL.Path != path {
					t.Errorf("request=%s %s", r.Method, r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":{"timezone":"Asia/Shanghai","rev":42,"fields":["Title"],"field_id_list":["fld_title"],"field_type_list":["text"],"record_id_list":["rec_fixture"],"data":[["原文"]],"has_more":false}}`)
			}))
			a := table(command, "--limit", "1")
			if command == "record-search" {
				a = table(command, "--limit", "1", "--keyword", "原文", "--search-field", "Title")
			}
			if command == "record-get" {
				a = table(command, "--record-id", "rec_fixture")
			}
			result := f.Run(a...)
			if result.ExitCode != 0 || calls.Load() != 1 {
				t.Fatalf("result=%+v calls=%d", result, calls.Load())
			}
			for _, value := range []string{"_record_id", "rec_fixture", "Title", "原文", "has_more=false"} {
				if !strings.Contains(result.Stdout, value) {
					t.Errorf("missing %s in %s", value, result.Stdout)
				}
			}
		})
	}
}

func TestBaseRawAPIKeepsUnmodeledResponseFields(t *testing.T) {
	runScenario(t, scenario{"raw-base", []string{"api", "GET", bp, "--as", "user"}, []exchange{ex("GET", bp, "", "", `{"base_token":"bas_fixture","future_shape":{"nested":[null,false,9007199254740993,"原文"]}}`)}, `{"base_token":"bas_fixture","future_shape":{"nested":[null,false,9007199254740993,"原文"]}}`})
}

func TestBaseBotCreateGrantsOnlyCurrentSyntheticUserWithoutNotification(t *testing.T) {
	runScenario(t, scenario{"bot-create", []string{"base", "+base-create", "--as", "bot", "--name", "Synthetic bot Base"}, []exchange{
		ex("POST", "/open-apis/base/v3/bases", "", `{"name":"Synthetic bot Base"}`, `{"base_token":"bas_created"}`),
		ex("POST", "/open-apis/drive/v1/permissions/bas_created/members", "type=bitable&need_notification=false", `{"member_type":"openid","member_id":"ou_foundation_fixture","perm":"full_access","type":"user"}`, `{}`),
	}, `{"created":true,"base":{"base_token":"bas_created"},"permission_grant":{"status":"granted","user_open_id":"ou_foundation_fixture","perm":"full_access"}}`})
}

func TestBaseCreateReplacesOnlyRunCreatedDefaultTable(t *testing.T) {
	runScenario(t, scenario{"replace-default-schema", args("base-create", "--name", "Synthetic Base", "--table-name", "Data", "--fields", `[{"name":"Title","type":"text"}]`), []exchange{
		ex("POST", "/open-apis/base/v3/bases", "", `{"name":"Synthetic Base"}`, `{"base_token":"bas_created","default_table_id":"tbl_default"}`),
		ex("POST", "/open-apis/base/v3/bases/bas_created/tables", "", `{"name":"Data","fields":[{"name":"Title","type":"text"}]}`, `{"id":"tbl_replacement","fields":[{"id":"fld_title","name":"Title","type":"text"}]}`),
		ex("DELETE", "/open-apis/base/v3/bases/bas_created/tables/tbl_default", "", "", `{}`),
	}, `{"created":true,"base":{"base_token":"bas_created"},"table":{"id":"tbl_replacement"},"default_table_deleted":true,"deleted_default_table_id":"tbl_default"}`})
}
