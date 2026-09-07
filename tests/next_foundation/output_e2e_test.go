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

	"github.com/larksuite/cli/internal/output"
)

const outputFixtureResult = `{"code":0,"msg":"success","data":{"files":[{"token":"file_001","name":"你好 😀","extra":"large unused content","empty":null,"identifier":"001"}],"has_more":true,"next_page_token":"next_fixture_page"}}`

func outputFixture(t *testing.T) (*Fixture, *atomic.Int32) {
	t.Helper()
	requests := &atomic.Int32{}
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != "GET" || r.URL.Path != "/open-apis/drive/v1/files" {
			t.Errorf("unexpected output fixture request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, outputFixtureResult)
	}))
	return fixture, requests
}

func runOutputRead(f *Fixture, flags ...string) Result {
	args := []string{"drive", "files", "list", "--as", "bot"}
	return f.Run(append(args, flags...)...)
}

func TestOutputCompactProjectionAndDetail(t *testing.T) {
	f, requests := outputFixture(t)
	compact := runOutputRead(f)
	if compact.ExitCode != 0 || compact.Stderr != "" || strings.Count(compact.Stdout, "\n") != 1 {
		t.Fatalf("compact read: %+v", compact)
	}
	if !strings.Contains(compact.Stdout, "large unused content") || !strings.Contains(compact.Stdout, "你好 😀") {
		t.Fatalf("default lost full data: %s", compact.Stdout)
	}
	projected := runOutputRead(f, "--result-fields", "name,empty,identifier")
	if projected.ExitCode != 0 || projected.Stderr != "" {
		t.Fatalf("projection: %+v", projected)
	}
	for _, want := range []string{`"token":"file_001"`, `"name":"你好 😀"`, `"empty":null`, `"identifier":"001"`, `"has_more":true`, `"next_page_token":"next_fixture_page"`} {
		if !strings.Contains(projected.Stdout, want) {
			t.Fatalf("projection lost %s: %s", want, projected.Stdout)
		}
	}
	if strings.Contains(projected.Stdout, "large unused content") {
		t.Fatalf("projection did not reduce data: %s", projected.Stdout)
	}
	detail := runOutputRead(f, "--result-detail")
	if detail.ExitCode != 0 || !strings.Contains(detail.Stdout, "\n  ") || !strings.Contains(detail.Stdout, "large unused content") {
		t.Fatalf("detail: %+v", detail)
	}
	if requests.Load() != 3 {
		t.Fatalf("requests = %d", requests.Load())
	}
}

func TestOutputInvalidOptionsBeforeRequests(t *testing.T) {
	f, requests := outputFixture(t)
	for _, flags := range [][]string{{"--result-fields", "name,,id"}, {"--result-fields", "name", "--result-detail"}, {"--result-fields", "name", "--jq", ".data"}, {"--result-fields", "name", "--format", "ndjson"}} {
		result := runOutputRead(f, flags...)
		if result.ExitCode != 2 || result.Stdout != "" || !strings.Contains(result.Stderr, `"type": "validation"`) {
			t.Fatalf("invalid flags %v: %+v", flags, result)
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("invalid options reached backend %d times", requests.Load())
	}
}

func TestOutputArtifactFullResultAndNoClobber(t *testing.T) {
	f, requests := outputFixture(t)
	path := filepath.Join(f.WorkDir, "saved-result.json")
	result := runOutputRead(f, "--save-result", path, "--result-fields", "name")
	if result.ExitCode != 0 || result.Stderr != "" {
		t.Fatalf("save result: %+v", result)
	}
	if strings.Contains(result.Stdout, "large unused content") || strings.Contains(result.Stdout, "你好") {
		t.Fatalf("artifact leaked record payload: %s", result.Stdout)
	}
	var reference struct {
		Artifact output.ArtifactReference `json:"artifact"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &reference); err != nil {
		t.Fatal(err)
	}
	if reference.Artifact.Schema != output.ArtifactSchema || reference.Artifact.Count == nil || *reference.Artifact.Count != 1 || reference.Artifact.Complete == nil || *reference.Artifact.Complete {
		t.Fatalf("artifact metadata: %s", result.Stdout)
	}
	if reference.Artifact.Binding.Environment != "feishu" || reference.Artifact.Binding.AppID != "synthetic-foundation-app" || reference.Artifact.Binding.Identity != "bot" {
		t.Fatalf("artifact binding: %+v", reference.Artifact.Binding)
	}
	if !strings.Contains(result.Stdout, "next_fixture_page") {
		t.Fatalf("artifact lost continuation: %s", result.Stdout)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var artifact output.ResultArtifact
	if err := json.Unmarshal(saved, &artifact); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"large unused content", "你好 😀", `"empty":null`, `"identifier":"001"`} {
		if !strings.Contains(string(artifact.Result), want) {
			t.Fatalf("file lost %s: %s", want, saved)
		}
	}
	if reference.Artifact.Bytes != int64(len(saved)) {
		t.Fatalf("byte count mismatch: %d != %d", reference.Artifact.Bytes, len(saved))
	}
	existing := runOutputRead(f, "--save-result", path)
	if existing.ExitCode != 2 || existing.Stdout != "" {
		t.Fatalf("existing target: %+v", existing)
	}
	if requests.Load() != 1 {
		t.Fatalf("existing target reached backend: %d", requests.Load())
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(saved) {
		t.Fatal("existing artifact changed")
	}
}

func TestOutputArtifactRejectsMutationBeforeRequest(t *testing.T) {
	f, requests := outputFixture(t)
	result := f.Run("api", "POST", "/open-apis/im/v1/messages", "--as", "bot", "--data", `{"receive_id":"synthetic","msg_type":"text","content":"{}"}`, "--save-result", filepath.Join(f.WorkDir, "not-created.json"))
	if result.ExitCode != 2 || result.Stdout != "" || requests.Load() != 0 {
		t.Fatalf("artifact mutation guard: %+v, requests=%d", result, requests.Load())
	}
}

func TestOutputArtifactRejectsCredentialPayload(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"code":0,"data":{"info":{"api_key":"synthetic-secret"}}}`)
	}))
	path := filepath.Join(f.WorkDir, "credential-result.json")
	result := runOutputRead(f, "--save-result", path)
	if result.ExitCode != 2 || result.Stdout != "" || !strings.Contains(result.Stderr, "credential-bearing") {
		t.Fatalf("credential artifact: %+v", result)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("credential file created: %v", err)
	}
}

func TestOutputArtifactBareMailPagination(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/messages"):
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"items":["message_fixture"],"has_more":true,"page_token":"next_fixture"}}`)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/messages/batch_get"):
			_, _ = fmt.Fprint(w, `{"code":0,"data":{"messages":[{"message_id":"message_fixture","subject":"synthetic mail subject"}]}}`)
		default:
			t.Errorf("unexpected bare mail request: %s %s", r.Method, r.URL.Path)
		}
	}))
	path := filepath.Join(f.WorkDir, "mail-result.json")
	result := f.Run("mail", "+triage", "--as", "bot", "--mailbox", "fixture@example.invalid", "--format", "json", "--max", "1", "--save-result", path)
	if result.ExitCode != 0 {
		t.Fatalf("bare mail artifact: %+v", result)
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		t.Fatal(err)
	}
	if _, exists := response["ok"]; exists {
		t.Fatalf("invented ok: %s", result.Stdout)
	}
	if string(response["mailbox_id"]) != `"fixture@example.invalid"` || string(response["has_more"]) != "true" || string(response["count"]) != "1" || len(response["page_token"]) < 3 {
		t.Fatalf("lost native pagination: %s", result.Stdout)
	}
	if _, exists := response["messages"]; exists {
		t.Fatalf("record content not handed to file: %s", result.Stdout)
	}
	var ref output.ArtifactReference
	if err := json.Unmarshal(response["artifact"], &ref); err != nil {
		t.Fatal(err)
	}
	if ref.Count == nil || *ref.Count != 1 || ref.Complete == nil || *ref.Complete {
		t.Fatalf("bare artifact extent: %+v", ref)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(saved), "synthetic mail subject") {
		t.Fatalf("mail record lost: %s", saved)
	}
}
