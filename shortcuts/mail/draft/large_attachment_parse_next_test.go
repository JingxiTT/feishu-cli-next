// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package draft

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestRemoveLargeAttachmentParseFailurePreservesSnapshot(t *testing.T) {
	body := strings.Repeat("<div>", 600) + `<div id="large-file-area-1"><div id="large-file-item"><a data-mail-token="tokA">file.pdf</a></div></div>` + strings.Repeat("</div>", 600)
	header := base64.StdEncoding.EncodeToString([]byte(`[{"id":"tokA"}]`))
	snapshot := &DraftSnapshot{Headers: []Header{{Name: LargeAttachmentIDsHeader, Value: header}}, Body: &Part{MediaType: "text/html", Body: []byte(body)}}
	before, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	err = removeLargeAttachment(snapshot, "tokA")
	if err == nil || errors.Unwrap(err) == nil {
		t.Fatalf("expected parser error with cause, got %v", err)
	}
	after, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed removal changed attachment headers or body")
	}
	updated, changed, err := RemoveLargeFileItemFromHTMLChecked(body, "tokA")
	if err == nil || changed || updated != body {
		t.Fatal("checked removal must preserve HTML on parser failure")
	}
	legacy, changed := RemoveLargeFileItemFromHTML(body, "tokA")
	if changed || legacy != body {
		t.Fatal("legacy wrapper must retain its failure contract")
	}
}
