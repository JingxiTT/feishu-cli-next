// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/httpmock"
)

var driveMetaTestSeq atomic.Int64

func TestFetchDriveMetaTitle(t *testing.T) {
	t.Run("returns title from batch_query response", func(t *testing.T) {
		runtime, reg := newDriveMetaTestRuntime(t)
		reg.Register(&httpmock.Stub{
			Method: "POST",
			URL:    "/open-apis/drive/v1/metas/batch_query",
			Body: map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"metas": []map[string]interface{}{
						{"doc_token": "doxcnABC", "doc_type": "docx", "title": "My Document"},
					},
				},
			},
		})

		title, err := FetchDriveMetaTitle(runtime, "doxcnABC", "docx")
		if err != nil {
			t.Fatalf("FetchDriveMetaTitle() error: %v", err)
		}
		if title != "My Document" {
			t.Errorf("title = %q, want %q", title, "My Document")
		}
	})

	t.Run("returns empty string when metas is empty", func(t *testing.T) {
		runtime, reg := newDriveMetaTestRuntime(t)
		reg.Register(&httpmock.Stub{
			Method: "POST",
			URL:    "/open-apis/drive/v1/metas/batch_query",
			Body: map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"metas": []map[string]interface{}{},
				},
			},
		})

		title, err := FetchDriveMetaTitle(runtime, "doxcnABC", "docx")
		if err != nil {
			t.Fatalf("FetchDriveMetaTitle() error: %v", err)
		}
		if title != "" {
			t.Errorf("title = %q, want empty string", title)
		}
	})

	t.Run("returns empty string when meta has no title", func(t *testing.T) {
		runtime, reg := newDriveMetaTestRuntime(t)
		reg.Register(&httpmock.Stub{
			Method: "POST",
			URL:    "/open-apis/drive/v1/metas/batch_query",
			Body: map[string]interface{}{
				"code": 0,
				"data": map[string]interface{}{
					"metas": []map[string]interface{}{
						{"doc_token": "doxcnABC", "doc_type": "docx"},
					},
				},
			},
		})

		title, err := FetchDriveMetaTitle(runtime, "doxcnABC", "docx")
		if err != nil {
			t.Fatalf("FetchDriveMetaTitle() error: %v", err)
		}
		if title != "" {
			t.Errorf("title = %q, want empty string", title)
		}
	})

	t.Run("propagates API error", func(t *testing.T) {
		runtime, reg := newDriveMetaTestRuntime(t)
		reg.Register(&httpmock.Stub{
			Method: "POST",
			URL:    "/open-apis/drive/v1/metas/batch_query",
			Body: map[string]interface{}{
				"code": 99991668,
				"msg":  "permission denied",
			},
		})

		_, err := FetchDriveMetaTitle(runtime, "doxcnABC", "docx")
		if err == nil {
			t.Fatal("FetchDriveMetaTitle() expected error, got nil")
		}
		p, ok := errs.ProblemOf(err)
		if !ok {
			t.Fatalf("expected typed error, got %T", err)
		}
		if p.Code != 99991668 {
			t.Fatalf("code = %d, want 99991668", p.Code)
		}
	})
}

func TestFetchDriveMetaURL(t *testing.T) {
	runtime, reg := newDriveMetaTestRuntime(t)
	stub := &httpmock.Stub{
		Method: "POST",
		URL:    "/open-apis/drive/v1/metas/batch_query",
		Body: map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"metas": []map[string]interface{}{
					{
						"doc_token": "boxcnABC",
						"doc_type":  "file",
						"title":     "report.pdf",
						"url":       "https://tenant.example.com/file/boxcnABC",
					},
				},
			},
		},
	}
	reg.Register(stub)

	got, err := FetchDriveMetaURL(runtime, "boxcnABC", "file")
	if err != nil {
		t.Fatalf("FetchDriveMetaURL() error: %v", err)
	}
	if got != "https://tenant.example.com/file/boxcnABC" {
		t.Fatalf("url = %q, want tenant URL", got)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(stub.CapturedBody, &body); err != nil {
		t.Fatalf("decode captured body: %v", err)
	}
	if body["with_url"] != true {
		t.Fatalf("with_url = %#v, want true", body["with_url"])
	}
}

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

func newDriveMetaTestRuntime(t *testing.T) (*RuntimeContext, *httpmock.Registry) {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())

	cfg := &core.CliConfig{
		AppID: fmt.Sprintf("drive-meta-test-%d", driveMetaTestSeq.Add(1)), AppSecret: "test-secret", Brand: core.BrandFeishu,
	}
	f, _, _, reg := cmdutil.TestFactory(t, cfg)
	runtime := &RuntimeContext{
		ctx:        context.Background(),
		Config:     cfg,
		Factory:    f,
		resolvedAs: core.AsBot,
	}
	return runtime, reg
}
