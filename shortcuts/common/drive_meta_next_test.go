// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"encoding/json"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestFetchDriveMetaMissingMetasIsEmpty(t *testing.T) {
	for _, data := range []map[string]interface{}{{}, {"metas": nil}} {
		runtime, reg := newDriveMetaTestRuntime(t)
		reg.Register(&httpmock.Stub{
			Method: "POST", URL: "/open-apis/drive/v1/metas/batch_query",
			Body: map[string]interface{}{"code": 0, "data": data},
		})
		meta, err := FetchDriveMeta(runtime, "target", "file", true)
		if err != nil || meta != (DriveMeta{}) {
			t.Fatalf("FetchDriveMeta() = %#v, %v, want empty metadata", meta, err)
		}
	}
}

func TestFetchDriveMetaSelectsExactDocument(t *testing.T) {
	matching := map[string]interface{}{"doc_token": "target", "doc_type": "file", "title": "Bound title", "url": "https://tenant.invalid/file/target"}
	tests := []struct {
		name    string
		metas   interface{}
		wantErr bool
	}{
		{"matching later token", []interface{}{map[string]interface{}{"doc_token": "other", "doc_type": "file", "title": "Wrong title"}, matching}, false},
		{"matching later type", []interface{}{map[string]interface{}{"doc_token": "target", "doc_type": "docx", "title": "Wrong title"}, matching}, false},
		{"wrong token", []interface{}{map[string]interface{}{"doc_token": "other", "doc_type": "file"}}, true},
		{"wrong type", []interface{}{map[string]interface{}{"doc_token": "target", "doc_type": "docx"}}, true},
		{"duplicate match", []interface{}{matching, matching}, true},
		{"missing token", []interface{}{map[string]interface{}{"doc_type": "file", "title": "Unbound title"}}, true},
		{"missing type", []interface{}{map[string]interface{}{"doc_token": "target", "title": "Unbound title"}}, true},
		{"missing identity", []interface{}{map[string]interface{}{"title": "Unbound title"}}, true},
		{"unbound row beside match", []interface{}{matching, map[string]interface{}{"title": "Unbound title"}}, true},
		{"null row", []interface{}{nil}, true},
		{"malformed list", "not a list", true},
		{"malformed token", []interface{}{map[string]interface{}{"doc_token": 42, "doc_type": "file"}}, true},
		{"malformed title", []interface{}{map[string]interface{}{"doc_token": "target", "doc_type": "file", "title": 42}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime, reg := newDriveMetaTestRuntime(t)
			stub := &httpmock.Stub{
				Method: "POST", URL: "/open-apis/drive/v1/metas/batch_query",
				Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"metas": tt.metas}},
			}
			reg.Register(stub)
			meta, err := FetchDriveMeta(runtime, "target", "file", true)
			if tt.wantErr {
				problem, ok := errs.ProblemOf(err)
				if !ok || problem.Subtype != errs.SubtypeInvalidResponse {
					t.Fatalf("error = %v, want typed invalid_response", err)
				}
				if meta != (DriveMeta{}) {
					t.Fatalf("invalid response leaked metadata: %#v", meta)
				}
				return
			}
			if err != nil || meta.Title != matching["title"] || meta.URL != matching["url"] {
				t.Fatalf("FetchDriveMeta() = %#v, %v, want exact document metadata", meta, err)
			}
			var request struct {
				Docs []struct {
					Token string `json:"doc_token"`
					Type  string `json:"doc_type"`
				} `json:"request_docs"`
				WithURL bool `json:"with_url"`
			}
			if err := json.Unmarshal(stub.CapturedBody, &request); err != nil {
				t.Fatal(err)
			}
			if len(request.Docs) != 1 || request.Docs[0].Token != "target" || request.Docs[0].Type != "file" || !request.WithURL {
				t.Fatalf("metadata request = %#v", request)
			}
		})
	}
}
