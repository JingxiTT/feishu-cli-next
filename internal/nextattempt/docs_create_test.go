// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const docsPending = `{"code":0,"data":{"task":{"task_id":"private-task","status":"processing"}}}`
const docsCompleted = `{"code":0,"data":{"task":{"task_id":"private-task","status":"succeeded","result":{"create_document":"{\"document\":{\"document_id\":\"private-document\"}}"}}}}`

func pendingDocsCreate(t *testing.T) (*Transport, Store, *Journal) {
	t.Helper()
	tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		return response(200, docsPending), nil
	})
	tr.IsMutation = func(req *http.Request) bool { return req.Method == "POST" }
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents", strings.NewReader(`{"content":"private-content"}`))
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if summary := journal.Summary(); summary.Unknown != 1 || summary.ExitCode() != 22 {
		t.Fatalf("async acknowledgement is not completion: %+v", summary)
	}
	return tr, store, journal
}

func TestDocsCreateCompletionNeedsNativeTaskProof(t *testing.T) {
	for _, tt := range []struct {
		name, proof string
		status      int
		applied     bool
	}{
		{"complete", docsCompleted, 200, true},
		{"trimmed task and status", strings.ReplaceAll(strings.Replace(docsCompleted, `"private-task"`, `" private-task "`, 1), `"succeeded"`, `" SUCCEEDED "`), 200, true},
		{"processing", docsPending, 200, false},
		{"failed", strings.Replace(docsCompleted, `"succeeded"`, `"failed"`, 1), 200, false},
		{"wrong task", strings.Replace(docsCompleted, "private-task", "other-task", 1), 200, false},
		{"missing task", strings.Replace(docsCompleted, `"task_id":"private-task",`, "", 1), 200, false},
		{"missing document", strings.Replace(docsCompleted, "private-document", "", 1), 200, false},
		{"invalid result", `{"code":0,"data":{"task":{"task_id":"private-task","status":"succeeded","result":{"create_document":"invalid"}}}}`, 200, false},
		{"contradictory failure", strings.Replace(docsCompleted, `"status":"succeeded"`, `"status":"succeeded","failure":{"code":"failed"}`, 1), 200, false},
		{"wrong type", strings.Replace(docsCompleted, `"status":"succeeded"`, `"status":"succeeded","type":"delete_document"`, 1), 200, false},
		{"API failure", strings.Replace(docsCompleted, `"code":0`, `"code":1`, 1), 200, false},
		{"HTTP failure", docsCompleted, 500, false},
		{"accepted", docsCompleted, 202, false},
		{"oversized", strings.Repeat(" ", responseEvidenceLimit) + docsCompleted, 200, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tr, store, journal := pendingDocsCreate(t)
			calls := 0
			tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return response(tt.status, tt.proof), nil
			})
			req, _ := http.NewRequest("GET", "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task", nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			summary := journal.Summary()
			if err != nil || string(body) != tt.proof || calls != 1 || (summary.Applied == 1) != tt.applied || len(summary.Receipts) != 1 {
				t.Fatalf("response or evidence changed: calls=%d err=%v summary=%+v", calls, err, summary)
			}
			receipt, err := store.Inspect(summary.Receipts[0].ID)
			if err != nil || receipt.Outcome != summary.Outcome {
				t.Fatalf("durable proof missing: %+v %v", receipt, err)
			}
			raw, err := os.ReadFile(filepath.Join(store.Dir, receipt.ID+".json"))
			if err != nil || strings.Contains(string(raw), "private-") {
				t.Fatalf("task or content leaked to receipt: %s %v", raw, err)
			}
		})
	}
}

func TestDocsCreateCompletionScopeAndLocalFailure(t *testing.T) {
	for _, change := range []string{"origin", "task", "identity", "binding error", "reused task", "deleted receipt", "transport error"} {
		t.Run(change, func(t *testing.T) {
			tr, store, journal := pendingDocsCreate(t)
			endpoint := "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task"
			sentinel := errors.New("synthetic transport failure")
			var transportErr error
			switch change {
			case "origin":
				endpoint = strings.Replace(endpoint, "fixture.invalid", "other.invalid", 1)
			case "task":
				endpoint += "-other"
			case "identity":
				tr.Resolve = func(*http.Request) (Binding, error) {
					return Binding{Environment: "https://fixture.invalid", AppID: "app-test", Identity: "user"}, nil
				}
			case "binding error":
				tr.Resolve = func(*http.Request) (Binding, error) { return Binding{}, errors.New("synthetic binding failure") }
			case "reused task":
				req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents", nil)
				resp, err := tr.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
			case "deleted receipt":
				if err := store.Cleanup(journal.Summary().Receipts[0].ID); err != nil {
					t.Fatal(err)
				}
			case "transport error":
				transportErr = sentinel
			}
			tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, docsCompleted), transportErr })
			req, _ := http.NewRequest("GET", endpoint, nil)
			resp, err := tr.RoundTrip(req)
			if !errors.Is(err, transportErr) {
				t.Fatalf("read error changed: %v", err)
			}
			resp.Body.Close()
			summary := journal.Summary()
			if summary.Applied != 0 || summary.ExitCode() != 22 || summary.PersistenceFailed != (change == "deleted receipt") {
				t.Fatalf("unrelated or unverifiable task upgraded: %+v", summary)
			}
		})
	}
}

func TestStandaloneDocsTaskReadDoesNotResolveBinding(t *testing.T) {
	tr, _, journal := testTransport(t, func(*http.Request) (*http.Response, error) { return response(200, docsCompleted), nil })
	tr.IsMutation = func(*http.Request) bool { return false }
	tr.Resolve = func(*http.Request) (Binding, error) {
		t.Fatal("standalone read resolved mutation identity")
		return Binding{}, nil
	}
	req, _ := http.NewRequest("GET", "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(journal.Summary().Receipts) != 0 {
		t.Fatal("standalone read produced receipt")
	}
}

func TestDocsCreateCaptureIsBounded(t *testing.T) {
	j := &Journal{}
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents", nil)
	for i := 0; i <= 128; i++ {
		id := strconv.Itoa(i)
		captured := j.captureDocsCreate(Receipt{ID: id}, req, &docsCreateTask{TaskID: id, Status: "processing"})
		if captured != (i < 128) {
			t.Fatalf("capture %d = %v", i, captured)
		}
	}
	if len(j.docsCreates) != 128 {
		t.Fatal("capture evicted earlier tasks")
	}
	if (&Journal{}).captureDocsCreate(Receipt{ID: "large"}, req, &docsCreateTask{TaskID: strings.Repeat("x", 4097)}) {
		t.Fatal("captured oversized task ID")
	}
}

func TestDocsCreateDuplicateAtCapacityCannotComplete(t *testing.T) {
	tr, store, journal := pendingDocsCreate(t)
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents", nil)
	for i := 1; i < 128; i++ {
		id := strconv.Itoa(i)
		if !journal.captureDocsCreate(Receipt{ID: id}, req, &docsCreateTask{TaskID: id, Status: "processing"}) {
			t.Fatal("failed to fill capture capacity")
		}
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, docsCompleted), nil })
	req, _ = http.NewRequest("GET", "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task", nil)
	resp, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if summary := journal.Summary(); summary.Applied != 0 || summary.Unknown != 2 || len(journal.docsCreates) != 128 {
		t.Fatalf("capture overflow hid a reused task: %+v", summary)
	}
	receipts, err := store.List()
	if err != nil || len(receipts) != 2 {
		t.Fatalf("receipts = %+v, %v", receipts, err)
	}
	for _, receipt := range receipts {
		if receipt.Outcome != Unknown {
			t.Fatal("ambiguous completion was persisted", receipt)
		}
	}
}

func TestDocsCreateInlineCompletionCannotReuseTask(t *testing.T) {
	for _, firstCompleted := range []bool{false, true} {
		tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
			if firstCompleted {
				return response(200, docsCompleted), nil
			}
			return response(200, docsPending), nil
		})
		tr.IsMutation = func(req *http.Request) bool { return req.Method == "POST" }
		req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents", nil)
		resp, err := tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, docsCompleted), nil })
		resp, err = tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		req, _ = http.NewRequest("GET", "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task", nil)
		resp, err = tr.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		wantApplied := 0
		if firstCompleted {
			wantApplied = 1
		}
		if summary := journal.Summary(); summary.Applied != wantApplied || summary.Unknown != 2-wantApplied {
			t.Fatalf("inline response reused task completion: %+v", summary)
		}
		receipts, err := store.List()
		if err != nil || len(receipts) != 2 {
			t.Fatalf("receipts = %+v, %v", receipts, err)
		}
		applied := 0
		for _, receipt := range receipts {
			if receipt.Outcome == Applied {
				applied++
			}
		}
		if applied != wantApplied {
			t.Fatal("ambiguous completion persisted", receipts)
		}
	}
}

func TestDocsCreateConcurrentCompletionPersistsOnce(t *testing.T) {
	tr, store, journal := pendingDocsCreate(t)
	tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) { return response(200, docsCompleted), nil })
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			req, _ := http.NewRequest("GET", "https://fixture.invalid/open-apis/docs_ai/v1/async_tasks/private-task", nil)
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Error(err)
				return
			}
			resp.Body.Close()
		}()
	}
	wait.Wait()
	summary := journal.Summary()
	if summary.Applied != 1 || summary.PersistenceFailed {
		t.Fatal("concurrent completion lost evidence", summary)
	}
	receipt, err := store.Inspect(summary.Receipts[0].ID)
	if err != nil || receipt.Outcome != Applied {
		t.Fatal("completion not persisted", receipt, err)
	}
}
