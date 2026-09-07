// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestPaginationHTTPFailureDoesNotPublishArtifact(t *testing.T) {
	for _, status := range []int{403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var pages atomic.Int32
			f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if pages.Add(1) == 1 {
					_, _ = fmt.Fprint(w, `{"code":0,"data":{"files":[{"token":"first"}],"has_more":true,"page_token":"next"}}`)
				} else {
					w.WriteHeader(status)
					_, _ = fmt.Fprint(w, `{"code":0,"data":{"files":[],"has_more":false}}`)
				}
			}))
			artifact := filepath.Join(f.WorkDir, "incomplete.json")
			r := f.Run("drive", "files", "list", "--as", "bot", "--page-all", "--save-result", artifact)
			if r.ExitCode == 0 || r.Stdout != "" || pages.Load() != 2 {
				t.Fatalf("HTTP failure published success: %+v pages=%d", r, pages.Load())
			}
			if _, err := os.Stat(artifact); !os.IsNotExist(err) {
				t.Fatalf("failed aggregate published artifact: %v", err)
			}
			var envelope struct {
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			start := strings.Index(r.Stderr, "{")
			if start < 0 || json.Unmarshal([]byte(r.Stderr[start:]), &envelope) != nil || envelope.Error.Code != status {
				t.Fatalf("HTTP status lost: %s", r.Stderr)
			}
		})
	}
}

func TestRawPaginationLaterErrorIsNotSuccess(t *testing.T) {
	for _, format := range []string{"json", "ndjson"} {
		t.Run(format, func(t *testing.T) {
			var pages atomic.Int32
			f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if pages.Add(1) == 1 {
					_, _ = fmt.Fprint(w, `{"code":0,"data":{"items":[{"id":"first"}],"has_more":true,"page_token":"next"}}`)
				} else {
					_, _ = fmt.Fprint(w, `{"code":99991679,"msg":"synthetic permission failure","log_id":"fixture-page-log"}`)
				}
			}))
			r := f.Run("api", "GET", "/open-apis/fixture/items", "--as", "user", "--page-all", "--format", format)
			var envelope struct {
				Error struct {
					Code  int    `json:"code"`
					LogID string `json:"log_id"`
				} `json:"error"`
			}
			start := strings.Index(r.Stderr, "{")
			if start < 0 {
				t.Fatalf("missing error envelope: %+v", r)
			}
			decodeErr := json.Unmarshal([]byte(r.Stderr[start:]), &envelope)
			if decodeErr != nil || r.ExitCode != 3 || pages.Load() != 2 || envelope.Error.Code != 99991679 || envelope.Error.LogID != "fixture-page-log" {
				t.Fatalf("lost terminal error evidence: %+v pages=%d", r, pages.Load())
			}
			if format == "json" && r.Stdout != "" {
				t.Fatalf("published failed aggregate: %s", r.Stdout)
			}
			if format == "ndjson" && !strings.Contains(r.Stdout, `"id":"first"`) {
				t.Fatalf("lost already streamed page: %s", r.Stdout)
			}
		})
	}
}
