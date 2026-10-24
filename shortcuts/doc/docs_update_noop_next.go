// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

// XML is the native editing representation. Comparing a Markdown rendering
// would incorrectly suppress an intentional change to rich formatting.
func canCheckDocsOverwrite(runtime *common.RuntimeContext, resources []localDocResource) bool {
	return runtime.Str("command") == "overwrite" && runtime.Str("doc-format") == "xml" && len(resources) == 0
}

func docsOverwriteReadBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{
		"format": "xml", "extra_param": docsFetchCommentsExtraParam,
		"export_option": map[string]interface{}{"export_block_id": true, "export_style_attrs": true, "export_cite_extra_data": true},
	}
	injectDocsScene(runtime, body)
	return body
}

type docsOverwriteSnapshot struct {
	Document struct {
		DocumentID   string                 `json:"document_id"`
		Content      *string                `json:"content"`
		RevisionID   json.Number            `json:"revision_id"`
		ReferenceMap map[string]interface{} `json:"reference_map"`
		Complete     *bool                  `json:"complete"`
		HasMore      bool                   `json:"has_more"`
		Truncated    bool                   `json:"truncated"`
	} `json:"document"`
	Complete  *bool           `json:"complete"`
	Warnings  json.RawMessage `json:"warnings"`
	HasMore   bool            `json:"has_more"`
	Truncated bool            `json:"truncated"`
	Result    string          `json:"result"`
}

// docsOverwriteUnchanged compares the full native representation only. It does
// not canonicalize XML, drop IDs, or equate an omitted sidecar with a nonempty one.
// A false result preserves the explicit overwrite operation and base revision.
func docsOverwriteUnchanged(data, body map[string]interface{}, documentID string, revision int) (bool, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return false, errs.NewInternalError(errs.SubtypeInvalidResponse, "cannot inspect document snapshot").WithCause(err)
	}
	var snapshot docsOverwriteSnapshot
	// UseNumber is needed for arbitrary reference-map payloads as well as revision.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err = decoder.Decode(&snapshot); err != nil {
		return false, errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid document snapshot").WithCause(err)
	}
	doc := snapshot.Document
	if snapshot.HasMore || snapshot.Truncated || doc.HasMore || doc.Truncated {
		return false, nil
	}
	if (snapshot.Complete != nil && !*snapshot.Complete) || (doc.Complete != nil && !*doc.Complete) {
		return false, nil
	}
	if (snapshot.Result != "" && snapshot.Result != "success") || doc.Content == nil || doc.DocumentID == "" || doc.RevisionID == "" {
		return false, nil
	}
	if documentID != "" && doc.DocumentID != documentID {
		return false, nil
	}
	if len(snapshot.Warnings) > 0 && string(snapshot.Warnings) != "null" && string(snapshot.Warnings) != "[]" {
		return false, nil
	}
	if revision >= 0 && doc.RevisionID.String() != strconv.Itoa(revision) {
		return false, nil
	}
	if *doc.Content != body["content"] {
		return false, nil
	}
	wanted, _ := body["reference_map"].(map[string]interface{})
	if len(wanted) == 0 && len(doc.ReferenceMap) == 0 {
		return true, nil
	}
	return reflect.DeepEqual(wanted, doc.ReferenceMap), nil
}
