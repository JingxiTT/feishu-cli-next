// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package apps

import (
	"context"
	"encoding/json"
	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
	"reflect"
	"strings"
	"testing"
)

func TestAutomationListAllRejectsMissingContinuation(t *testing.T) {
	rctx, stdout, reg := newOpenAPIKeyRCtx(t, automationListFlagDefs(), map[string]string{"app-id": "app_x", "all": "true"})
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/spark/v1/apps/app_x/triggers", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"items": []interface{}{}, "has_more": true}}})
	assertInternalError(t, AppsAutomationList.Execute(context.Background(), rctx), errs.SubtypeInvalidResponse)
	if stdout.Len() != 0 {
		t.Fatalf("truncated success: %s", stdout)
	}
}

func TestAppsReplacementEqualityRequiresObservedFields(t *testing.T) {
	for _, tc := range []struct {
		name          string
		current       map[string]interface{}
		requested     map[string]interface{}
		same, invalid bool
	}{
		{"equal", map[string]interface{}{"app": map[string]interface{}{"app_id": "app_x", "name": "中文", "description": "keep"}}, map[string]interface{}{"name": "中文"}, true, false},
		{"changed", map[string]interface{}{"app": map[string]interface{}{"app_id": "app_x", "name": "old"}}, map[string]interface{}{"name": "new"}, false, false},
		{"omitted", map[string]interface{}{"app": map[string]interface{}{"app_id": "app_x"}}, map[string]interface{}{"description": ""}, false, false},
		{"different target", map[string]interface{}{"app": map[string]interface{}{"app_id": "app_other", "name": "new"}}, map[string]interface{}{"name": "new"}, false, true},
		{"missing target", map[string]interface{}{}, map[string]interface{}{"name": "new"}, false, true},
		{"wrong type", map[string]interface{}{"app": map[string]interface{}{"app_id": "app_x", "name": 12}}, map[string]interface{}{"name": "new"}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			same, err := appsUpdateUnchanged("app_x", tc.current, tc.requested)
			if same != tc.same || (err != nil) != tc.invalid {
				t.Fatalf("same=%v err=%v", same, err)
			}
		})
	}
}

func TestSQLRowsPreserveDecimalIntegerAndEmptyTypes(t *testing.T) {
	stmt := map[string]interface{}{"sql_type": "SELECT", "data": `[{"id":9007199254740993,"decimal":123.1234567890123456789,"code":"001","empty":"","null":null,"enabled":false,"nested":{"n":9007199254740995}}]`}
	if err := validateSQLStatements([]map[string]interface{}{stmt}); err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(selectRows(stmt))
	if err != nil {
		t.Fatal(err)
	}
	var actual, expected interface{}
	actualDecoder := json.NewDecoder(strings.NewReader(string(got)))
	actualDecoder.UseNumber()
	_ = actualDecoder.Decode(&actual)
	expectedDecoder := json.NewDecoder(strings.NewReader(stmt["data"].(string)))
	expectedDecoder.UseNumber()
	_ = expectedDecoder.Decode(&expected)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("precision/type loss: %s", got)
	}
}

func TestSQLMalformedRowsCannotBecomeSuccessfulEmptyResults(t *testing.T) {
	for _, data := range []string{`{"error":"bad"}`, `[{bad}]`, `[] trailing`} {
		err := validateSQLStatements([]map[string]interface{}{{"sql_type": "SELECT", "data": data}})
		assertInternalError(t, err, errs.SubtypeInvalidResponse)
	}
}

func TestEnvValuesRejectArtifactsBeforeFetch(t *testing.T) {
	rctx, _, _ := newOpenAPIKeyRCtx(t, map[string]string{"app-id": "string", "environment": "string", "include-values": "bool", "save-result": "string"}, map[string]string{"app-id": "app_x", "environment": "dev", "include-values": "true", "save-result": "result.json"})
	err := AppsEnvVarList.Validate(context.Background(), rctx)
	if p, ok := errs.ProblemOf(err); !ok || p.Category != errs.CategoryValidation {
		t.Fatalf("expected typed validation, got %v", err)
	}
	_ = rctx.Cmd.Flags().Set("include-values", "false")
	if err := AppsEnvVarList.Validate(context.Background(), rctx); err != nil {
		t.Fatalf("redacted artifact rejected: %v", err)
	}
}

func TestFileDeletionMissingResultsRemainUnknown(t *testing.T) {
	results := projectDeleteResults([]interface{}{map[string]interface{}{"status": "ok"}}, []string{"/a", "/b"})
	if results[0]["status"] != "ok" || results[1]["status"] != "unknown" {
		t.Fatalf("missing result claimed success: %#v", results)
	}
	if results[1]["error"].(map[string]interface{})["code"] != "DELETE_UNKNOWN" {
		t.Fatalf("missing recovery evidence: %#v", results)
	}
}
func TestExportCountsSerializedRecords(t *testing.T) {
	for _, tc := range []struct {
		format, data string
		count        int
		known        bool
	}{
		{"csv", "id,note\n1,\"line one\nline two\"\n2,end\n", 2, true},
		{"json", `[{"id":9007199254740993},{"name":"中文"}]`, 2, true},
		{"json", `[bad]`, 0, false}, {"sql", "INSERT INTO x VALUES (1);", 0, false},
	} {
		got, known := countExportRows([]byte(tc.data), tc.format)
		if got != tc.count || known != tc.known {
			t.Fatalf("%s count=%d known=%v", tc.format, got, known)
		}
	}
}

func TestDatabaseImportRejectsGenericArtifacts(t *testing.T) {
	for _, content := range []string{`{"schema":"feishu-cli-next/result/v1","result":{"data":[]}}`, `{"schema":"feishu-cli-next/result/v2"}`} {
		if err := rejectResultArtifactImport([]byte(content)); err == nil {
			t.Fatalf("artifact accepted: %s", content)
		}
	}
	if err := rejectResultArtifactImport([]byte(`[{"id":9007199254740993}]`)); err != nil {
		t.Fatal(err)
	}
}
