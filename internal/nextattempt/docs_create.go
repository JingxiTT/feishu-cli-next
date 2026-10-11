// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type docsCreateTask struct {
	TaskID  string          `json:"task_id"`
	Type    string          `json:"type"`
	Status  string          `json:"status"`
	Failure json.RawMessage `json:"failure"`
	Result  struct {
		CreateDocument string `json:"create_document"`
	} `json:"result"`
}

type docsCreateAttempt struct {
	Binding   Binding
	Origin    string
	TaskID    string
	Ambiguous bool
}

func (j *Journal) captureDocsCreate(receipt Receipt, req *http.Request, task *docsCreateTask) bool {
	taskID := strings.TrimSpace(task.TaskID)
	if taskID == "" || len(taskID) > 4096 || task.Type != "" && task.Type != "create_document" {
		return false
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	origin := req.URL.Scheme + "://" + req.URL.Host
	duplicate := false
	for id, previous := range j.docsCreates {
		if previous.Origin == origin && previous.TaskID == taskID {
			previous.Ambiguous = true
			j.docsCreates[id] = previous
			duplicate = true
		}
	}
	// shortcut: retain at most 128 task IDs per invocation; expand only for a workflow that creates more documents.
	if duplicate || len(j.docsCreates) >= 128 {
		return false
	}
	if j.docsCreates == nil {
		j.docsCreates = make(map[string]docsCreateAttempt)
	}
	j.docsCreates[receipt.ID] = docsCreateAttempt{Binding: receipt.Binding, Origin: origin, TaskID: taskID}
	return true
}

func completedDocsCreate(task *docsCreateTask, taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if task == nil || taskID == "" || strings.TrimSpace(task.TaskID) != taskID || task.Type != "" && task.Type != "create_document" || !strings.EqualFold(strings.TrimSpace(task.Status), "succeeded") || len(task.Failure) != 0 && string(task.Failure) != "null" {
		return false
	}
	var result struct {
		Result   string `json:"result"`
		Document struct {
			ID string `json:"document_id"`
		} `json:"document"`
	}
	raw := []byte(task.Result.CreateDocument)
	return json.Unmarshal(raw, &result) == nil && strings.TrimSpace(result.Document.ID) != "" && !strings.EqualFold(strings.TrimSpace(result.Result), "failed") && !hasItemFailure(raw)
}

// docsCreateForRequest selects only the exact native GET for a captured task.
// Observing at RoundTrip keeps redirect responses from impersonating that GET.
func (j *Journal) docsCreateForRequest(req *http.Request) (string, docsCreateAttempt) {
	if j == nil || req.Method != "GET" || req.URL == nil {
		return "", docsCreateAttempt{}
	}
	path := req.URL.EscapedPath()
	if !strings.HasPrefix(path, "/open-apis/docs_ai/v1/async_tasks/") {
		return "", docsCreateAttempt{}
	}
	origin := req.URL.Scheme + "://" + req.URL.Host
	j.mu.Lock()
	defer j.mu.Unlock()
	id := ""
	var attempt docsCreateAttempt
	for candidate, pending := range j.docsCreates {
		if origin != pending.Origin || path != "/open-apis/docs_ai/v1/async_tasks/"+url.PathEscape(pending.TaskID) {
			continue
		}
		if id != "" || pending.Ambiguous {
			return "", docsCreateAttempt{} // Reused task IDs cannot identify one create.
		}
		id, attempt = candidate, pending
	}
	return id, attempt
}

func (t *Transport) observeDocsCreate(req *http.Request, resp *http.Response) {
	id, attempt := t.Journal.docsCreateForRequest(req)
	if id == "" || resp == nil || resp.Body == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || resp.StatusCode == http.StatusAccepted {
		return
	}
	data, err := readResponseEvidence(resp)
	if err != nil || len(data) > responseEvidenceLimit {
		return
	}
	var envelope struct {
		Code *int `json:"code"`
		Data struct {
			Task *docsCreateTask `json:"task"`
		} `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.Code == nil || *envelope.Code != 0 || !completedDocsCreate(envelope.Data.Task, attempt.TaskID) || hasItemFailure(data) {
		return
	}
	binding, err := t.Resolve(req)
	if err != nil || binding != attempt.Binding {
		return
	}
	j := t.Journal
	j.mu.Lock()
	defer j.mu.Unlock()
	for candidate, pending := range j.docsCreates {
		if (candidate != id || pending.Ambiguous) && pending.Origin == attempt.Origin && pending.TaskID == attempt.TaskID {
			return // A concurrent create may have reused the task after selection.
		}
	}
	for i, receipt := range j.receipts {
		if receipt.ID != id || receipt.Outcome != Unknown || receipt.Evidence != "docs_create_task_pending" || receipt.PersistenceFailed {
			continue
		}
		completed := receipt
		completed.Outcome, completed.Evidence = Applied, "docs_create_task_completed"
		completed.UpdatedAt = time.Now().UTC()
		if persistPendingCompletion(t.Store(), receipt, completed) == nil {
			j.receipts[i] = completed
		} else {
			j.receipts[i].PersistenceFailed = true
		}
		return
	}
}
