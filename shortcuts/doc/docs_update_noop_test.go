// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/json"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
)

func TestDocsOverwriteNoopRequiresExactNativeSnapshot(t *testing.T) {
	const content = `<title>中文</title><p id="block1"><b>Rich</b></p>`
	sidecar := map[string]interface{}{"widget": map[string]interface{}{"large": json.Number("9007199254740993")}}
	body := map[string]interface{}{"content": content, "reference_map": sidecar}
	tests := []struct {
		name      string
		data      map[string]interface{}
		revision  int
		unchanged bool
	}{
		{"exact", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": json.Number("3"), "content": content, "reference_map": sidecar}}, -1, true},
		{"same revision", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": content, "reference_map": sidecar}}, 3, true},
		{"different revision", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 4, "content": content, "reference_map": sidecar}}, 3, false},
		{"wrong document", map[string]interface{}{"document": map[string]interface{}{"document_id": "otherDocument", "revision_id": 3, "content": content, "reference_map": sidecar}}, -1, false},
		{"explicit incomplete", map[string]interface{}{"complete": false, "document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": content, "reference_map": sidecar}}, -1, false},
		{"no content", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3}}, -1, false},
		{"lossy text", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": "Rich", "reference_map": sidecar}}, -1, false},
		{"missing sidecar", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": content}}, -1, false},
		{"truncated", map[string]interface{}{"truncated": true, "document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": content, "reference_map": sidecar}}, -1, false},
		{"warnings", map[string]interface{}{"warnings": []string{"partial content"}, "document": map[string]interface{}{"document_id": "doxcnTest", "revision_id": 3, "content": content, "reference_map": sidecar}}, -1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := docsOverwriteUnchanged(tt.data, body, "doxcnTest", tt.revision)
			if err != nil || got != tt.unchanged {
				t.Fatalf("unchanged=%v err=%v, want %v", got, err, tt.unchanged)
			}
		})
	}
}

func TestDocsOverwriteIdenticalXMLDoesNotMutate(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, stdout, _, reg := cmdutil.TestFactory(t, docsTestConfigWithAppID("docs-unchanged"))
	const content = `<p id="native"><b>中文</b></p>`
	read := registerDocsAIStub(reg, "POST", "/open-apis/docs_ai/v1/documents/doxcnNoop/fetch", map[string]interface{}{"document": map[string]interface{}{"document_id": "doxcnNoop", "revision_id": 3, "content": content}})
	// No PUT stub: a backend mutation fails this test independently of stdout.
	err := mountAndRunDocs(t, DocsUpdate, []string{"+update", "--as", "user", "--doc", "doxcnNoop", "--command", "overwrite", "--content", content}, f, stdout)
	if err != nil {
		t.Fatal(err)
	}
	body := decodeRequestBody(t, read.CapturedBody)
	if body["format"] != "xml" || body["read_option"] != nil {
		t.Fatalf("not a full XML snapshot: %#v", body)
	}
	var result struct {
		Data struct {
			Unchanged bool `json:"unchanged"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || !result.Data.Unchanged {
		t.Fatalf("output %s, err %v", stdout, err)
	}
}
