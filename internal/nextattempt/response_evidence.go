// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// classifyKnownResponse projects only endpoint shapes whose existing domain
// owners establish stronger evidence than code:0. It never stores payloads or
// invents applied item counts. Unrecognized endpoint semantics remain generic.
func classifyKnownResponse(r *Receipt, req *http.Request, data json.RawMessage, journal *Journal) bool {
	unknown := func(evidence string) bool { r.Outcome, r.Evidence = Unknown, evidence; return true }
	applied := func(evidence string) bool { r.Outcome, r.Evidence = Applied, evidence; return true }
	if requestMatches(req, "POST", "/open-apis/mindnote/v1/mindnotes/{mindnote}/nodes") {
		if completeMindnoteCreate(req, data) {
			return applied("mindnote_created_node_ids_complete")
		}
		return unknown("mindnote_node_effects_require_reconciliation")
	}
	if requestMatches(req, "POST", "/open-apis/drive/v1/import_tasks") {
		if journal.captureDriveImport(*r, req, data) {
			return unknown("drive_import_task_pending")
		}
		return unknown("async_task_requires_reconciliation")
	}
	if requestMatches(req, "POST", "/open-apis/slides_ai/v1/xml_presentations/{presentation}/history/revert") {
		return unknown("async_task_requires_reconciliation")
	}
	if requestMatches(req, "POST", "/open-apis/sheet_ai/v2/spreadsheets/{spreadsheet}/tools/invoke_write") {
		var request struct {
			ToolName string `json:"tool_name"`
		}
		var result struct {
			Output string `json:"output"`
		}
		if !decodeRequestEvidence(req, &request) || request.ToolName == "" || json.Unmarshal(data, &result) != nil || strings.TrimSpace(result.Output) == "" || !json.Valid([]byte(result.Output)) {
			return unknown("sheets_tool_output_unverifiable")
		}
		complete, evidence := SheetsToolOutputComplete(request.ToolName, result.Output)
		if !complete {
			return unknown(evidence)
		}
		return applied(evidence)
	}
	if requestMatches(req, "POST", "/open-apis/base/v3/bases/{base}/tables/{table}/records") || requestMatches(req, "PATCH", "/open-apis/base/v3/bases/{base}/tables/{table}/records/{record}") {
		var result struct {
			Ignored []json.RawMessage `json:"ignored_fields"`
		}
		if json.Unmarshal(data, &result) != nil || len(result.Ignored) > 0 {
			return unknown("base_record_fields_ignored")
		}
		return false
	}
	if requestMatches(req, "POST", "/open-apis/docs_ai/v1/documents/{document}/history/revert") {
		return unknown("docs_history_requires_reconciliation")
	}
	if requestMatches(req, "POST", "/open-apis/docs_ai/v1/documents") || requestMatches(req, "PUT", "/open-apis/docs_ai/v1/documents/{document}") {
		var result struct {
			Result string          `json:"result"`
			Task   *docsCreateTask `json:"task"`
		}
		if json.Unmarshal(data, &result) != nil || strings.EqualFold(strings.TrimSpace(result.Result), "failed") {
			return unknown("docs_result_failed_or_invalid")
		}
		if result.Task != nil {
			if req.Method == "POST" && journal.captureDocsCreate(*r, req, result.Task) {
				if completedDocsCreate(result.Task, result.Task.TaskID) {
					return applied("docs_create_task_completed")
				}
				if status := strings.ToLower(strings.TrimSpace(result.Task.Status)); status == "" || status == "processing" {
					return unknown("docs_create_task_pending")
				}
			}
			return unknown("async_task_requires_reconciliation")
		}
		return false
	}
	if requestMatches(req, "POST", "/open-apis/slides_ai/v1/xml_presentations/{presentation}/slide/replace") {
		var result struct {
			Index  *int   `json:"failed_part_index"`
			Reason string `json:"failed_reason"`
		}
		if json.Unmarshal(data, &result) != nil || strings.TrimSpace(result.Reason) != "" || result.Index != nil && *result.Index >= 0 {
			return unknown("slides_replace_parts_failed")
		}
		return false
	}
	if isIMRecipientMutation(req) {
		var result struct {
			InvalidUsers    *[]json.RawMessage `json:"invalid_user_id_list"`
			InvalidIDs      *[]json.RawMessage `json:"invalid_id_list"`
			NotExisted      *[]json.RawMessage `json:"not_existed_id_list"`
			PendingApproval *[]json.RawMessage `json:"pending_approval_id_list"`
			Invalid         *[]json.RawMessage `json:"invalid_ids"`
		}
		if json.Unmarshal(data, &result) != nil {
			return unknown("im_recipient_outcomes_incomplete")
		}
		for _, items := range []*[]json.RawMessage{result.InvalidUsers, result.InvalidIDs, result.NotExisted, result.PendingApproval, result.Invalid} {
			if items != nil && len(*items) > 0 {
				return unknown("im_recipient_outcomes_incomplete")
			}
		}
		// Failure arrays imply success for recipients outside those arrays. They
		// must actually be present; an empty data object proves nothing. Explicit
		// per-member result lists instead need request/result reconciliation.
		present := false
		switch {
		case requestMatches(req, "PATCH", "/open-apis/im/v1/messages/{message}/urgent_app"), requestMatches(req, "PATCH", "/open-apis/im/v1/messages/{message}/urgent_phone"), requestMatches(req, "PATCH", "/open-apis/im/v1/messages/{message}/urgent_sms"):
			present = result.InvalidUsers != nil
		case requestMatches(req, "POST", "/open-apis/im/v1/chats/{chat}/members"):
			present = result.InvalidIDs != nil || result.NotExisted != nil || result.PendingApproval != nil
		case requestMatches(req, "DELETE", "/open-apis/im/v1/chats/{chat}/members"):
			present = result.InvalidIDs != nil
		case requestMatches(req, "POST", "/open-apis/im/v1/chats/{chat}/chat_user_setting/batch_update"):
			present = result.Invalid != nil
		}
		if !present {
			return unknown("im_recipient_outcomes_incomplete")
		}
		return applied("im_recipient_response_complete")
	}

	if requestMatches(req, "PUT", "/open-apis/minutes/v1/minutes/{minute}/transcript/word") {
		if completeWordReplacement(req, data) {
			return applied("minutes_replacement_counts_complete")
		}
		return unknown("minutes_replacement_counts_incomplete")
	}
	if requestMatches(req, "POST", "/open-apis/base/v3/bases/{base}/tables/{table}/records/batch_create") {
		if ids := completeBaseCreate(req, data); ids != nil && journal.observeBaseCreates(r.Binding, req.URL, ids) {
			return applied("base_created_record_ids_complete")
		}
		return unknown("base_created_record_ids_incomplete")
	}
	if requestMatches(req, "POST", "/open-apis/spark/v1/apps/{app}/sql_commands") {
		var result struct {
			Result string `json:"result"`
		}
		var statements []struct {
			SQLType string `json:"sql_type"`
		}
		if json.Unmarshal(data, &result) != nil || json.Unmarshal([]byte(result.Result), &statements) != nil || len(statements) == 0 {
			return unknown("sql_statement_results_incomplete")
		}
		for _, statement := range statements {
			if strings.TrimSpace(statement.SQLType) == "" || strings.EqualFold(statement.SQLType, "ERROR") {
				return unknown("sql_statement_results_incomplete")
			}
		}
		return applied("sql_statement_responses_complete")
	}
	if requestMatches(req, "POST", "/open-apis/vc/v1/bots/invite") {
		var request struct {
			Type     int `json:"invite_type"`
			Invitees []struct {
				ID       string `json:"id"`
				UserType int    `json:"user_type"`
			} `json:"invitees"`
		}
		var result struct {
			Failed  *int `json:"failed_count"`
			Invited *int `json:"invited_count"`
			HasMore bool `json:"has_more"`
			Results []struct {
				ID     string `json:"id"`
				Status int    `json:"status"`
			} `json:"invite_results"`
		}
		if !decodeRequestEvidence(req, &request) || request.Type != 2 || len(request.Invitees) == 0 || json.Unmarshal(data, &result) != nil || result.Failed == nil || *result.Failed != 0 || result.Invited == nil || *result.Invited != len(request.Invitees) || len(request.Invitees) != len(result.Results) || result.HasMore {
			return unknown("vc_invitation_results_incomplete")
		}
		remaining := map[string]bool{}
		for _, item := range request.Invitees {
			if item.ID == "" || item.UserType != 1 || remaining[item.ID] {
				return unknown("vc_invitation_results_incomplete")
			}
			remaining[item.ID] = true
		}
		for _, item := range result.Results {
			if !remaining[item.ID] || item.Status != 1 {
				return unknown("vc_invitation_results_incomplete")
			}
			delete(remaining, item.ID)
		}
		if len(remaining) > 0 {
			return unknown("vc_invitation_results_incomplete")
		}
		return applied("vc_invitation_results_complete")
	}

	if requestMatches(req, "POST", "/open-apis/wiki/v2/spaces/{space}/nodes/move_docs_to_wiki") ||
		requestMatches(req, "POST", "/open-apis/wiki/v2/nodes/{node}/move_wiki_to_docs") ||
		requestMatches(req, "DELETE", "/open-apis/wiki/v2/spaces/{space}/nodes/{node}") ||
		requestMatches(req, "DELETE", "/open-apis/wiki/v2/spaces/{space}") {
		var result struct {
			TaskID string `json:"task_id"`
		}
		if json.Unmarshal(data, &result) != nil || result.TaskID != "" {
			return unknown("wiki_task_requires_reconciliation")
		}
	}
	return false
}

func isIMRecipientMutation(req *http.Request) bool {
	for _, suffix := range []string{"urgent_app", "urgent_phone", "urgent_sms"} {
		if requestMatches(req, "PATCH", "/open-apis/im/v1/messages/{message}/"+suffix) {
			return true
		}
	}
	return requestMatches(req, "POST", "/open-apis/im/v1/chats/{chat}/members") ||
		requestMatches(req, "DELETE", "/open-apis/im/v1/chats/{chat}/members") ||
		requestMatches(req, "POST", "/open-apis/im/v1/chats/{chat}/chat_user_setting/batch_update") ||
		requestMatches(req, "POST", "/open-apis/im/v1/chats/{chat}/membership_applications/handle")
}

func completeWordReplacement(req *http.Request, data json.RawMessage) bool {
	var request struct {
		Words []struct {
			Source string `json:"source_word"`
		} `json:"replace_words"`
	}
	var result struct {
		Counts []struct {
			Source string          `json:"source_word"`
			Count  json.RawMessage `json:"replace_count"`
		} `json:"replace_word_counts"`
	}
	if !decodeRequestEvidence(req, &request) || json.Unmarshal(data, &result) != nil || len(request.Words) == 0 || len(request.Words) != len(result.Counts) {
		return false
	}
	remaining := map[string]bool{}
	for _, item := range request.Words {
		if item.Source == "" || remaining[item.Source] {
			return false
		}
		remaining[item.Source] = true
	}
	for _, item := range result.Counts {
		if !remaining[item.Source] {
			return false
		}
		raw := string(item.Count)
		if strings.HasPrefix(raw, "\"") {
			if json.Unmarshal(item.Count, &raw) != nil {
				return false
			}
		}
		count, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || count <= 0 {
			return false
		}
		delete(remaining, item.Source)
	}
	return len(remaining) == 0
}

func completeBaseCreate(req *http.Request, data json.RawMessage) []string {
	var request struct {
		Records []json.RawMessage `json:"create_records"`
	}
	var result struct {
		IDs     []string          `json:"record_id_list"`
		Ignored []json.RawMessage `json:"ignored_fields"`
		Rev     *int64            `json:"rev"`
	}
	// This narrower success proof intentionally accepts only the established
	// batch-create response schema; extra async/failure fields remain unknown.
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if !decodeRequestEvidence(req, &request) || decoder.Decode(&result) != nil || len(request.Records) == 0 || len(request.Records) > 200 || len(request.Records) != len(result.IDs) || len(result.Ignored) > 0 || result.Rev != nil && *result.Rev < 0 {
		return nil
	}
	seen := map[string]bool{}
	for _, id := range result.IDs {
		if id == "" || seen[id] {
			return nil
		}
		seen[id] = true
	}
	return result.IDs
}

func completeMindnoteCreate(req *http.Request, data json.RawMessage) bool {
	var request struct {
		Nodes []*struct {
			ID string `json:"node_id"`
		} `json:"nodes"`
	}
	var result struct {
		ClientToken string   `json:"client_token"`
		IDs         []string `json:"ids"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if !decodeRequestEvidence(req, &request) || decoder.Decode(&result) != nil || len(request.Nodes) == 0 || len(request.Nodes) != len(result.IDs) {
		return false
	}
	for _, node := range request.Nodes {
		// This endpoint also updates nodes, but the frozen response contract
		// defines ids as created nodes only. It cannot establish update coverage.
		if node == nil || node.ID != "" {
			return false
		}
	}
	seen := map[string]bool{}
	for _, id := range result.IDs {
		if strings.TrimSpace(id) == "" || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

// The SDK's bytes.Buffer requests expose GetBody. Inspect only a bounded copy;
// unsupported streaming bodies remain unknown and are never consumed or saved.
func decodeRequestEvidence(req *http.Request, target any) bool {
	if req.GetBody == nil {
		return false
	}
	body, err := req.GetBody()
	if err != nil {
		return false
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, responseEvidenceLimit+1))
	return err == nil && len(data) <= responseEvidenceLimit && json.Unmarshal(data, target) == nil
}

func requestMatches(req *http.Request, method, pattern string) bool {
	if req.Method != method || req.URL == nil {
		return false
	}
	path := strings.Split(req.URL.Path, "/")
	parts := strings.Split(pattern, "/")
	if len(path) != len(parts) {
		return false
	}
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if path[i] == "" {
				return false
			}
			continue
		}
		if path[i] != part {
			return false
		}
	}
	return true
}

// SheetsToolOutputComplete applies the receipt layer's established nested-output
// evidence rules. Envelope success alone does not prove a write page completed.
func SheetsToolOutputComplete(toolName, raw string) (bool, string) {
	if toolName == "batch_update" || toolName == "history_revert" || hasItemFailure([]byte(raw)) {
		return false, "sheets_tool_items_require_reconciliation"
	}
	var object map[string]json.RawMessage
	var array []json.RawMessage
	if (json.Unmarshal([]byte(raw), &object) == nil && len(object) > 0) ||
		(json.Unmarshal([]byte(raw), &array) == nil && len(array) > 0) {
		return true, "sheets_tool_response_decoded"
	}
	return false, "sheets_tool_output_unverifiable"
}
