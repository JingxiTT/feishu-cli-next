// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestKnownEndpointMutationEvidence(t *testing.T) {
	words := `{"replace_words":[{"source_word":"first","target_word":"target"},{"source_word":"second","target_word":"other"}]}`
	records := `{"create_records":[{"f":"private-value"},{"f":"other"}]}`
	nodes := `{"nodes":[{"texts":[]},{"finish":false}]}`
	tests := []struct{ name, method, path, body, data, outcome string }{
		{"mindnote created IDs complete", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one","two"]}`, Applied},
		{"mindnote missing IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{}`, Unknown},
		{"mindnote short IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one"]}`, Unknown},
		{"mindnote duplicate IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one","one"]}`, Unknown},
		{"mindnote empty IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one",""]}`, Unknown},
		{"mindnote extra IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one","two","three"]}`, Unknown},
		{"mindnote malformed IDs", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":[1,2]}`, Unknown},
		{"mindnote unknown async field", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", nodes, `{"ids":["one","two"],"task_id":"pending"}`, Unknown},
		{"mindnote null node", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", `{"nodes":[null]}`, `{"ids":["one"]}`, Unknown},
		{"mindnote empty request", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", `{}`, `{"ids":[]}`, Unknown},
		{"mindnote update IDs not coverage proof", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", `{"nodes":[{"node_id":"one","finish":true}]}`, `{"ids":["one"]}`, Unknown},
		{"mindnote mixed IDs not coverage proof", "POST", "/open-apis/mindnote/v1/mindnotes/note/nodes", `{"nodes":[{}, {"node_id":"one","finish":true}]}`, `{"ids":["two","one"]}`, Unknown},
		{"docs create failed", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"result":"failed"}`, Unknown},
		{"docs update failed", "PUT", "/open-apis/docs_ai/v1/documents/doc", `{}`, `{"result":" FAILED "}`, Unknown},
		{"docs create success", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"result":"success"}`, Applied},
		{"docs synchronous document", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"document":{"document_id":"doc"},"task":null}`, Applied},
		{"docs pending task", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"task":{"task_id":"task","status":"processing"}}`, Unknown},
		{"docs failed task", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"task":{"task_id":"task","status":"failed"}}`, Unknown},
		{"docs task without proof", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"task":{"task_id":"task","status":"succeeded"}}`, Unknown},
		{"docs completed task", "POST", "/open-apis/docs_ai/v1/documents", `{}`, `{"task":{"task_id":"task","status":"succeeded","result":{"create_document":"{\"document\":{\"document_id\":\"doc\"}}"}}}`, Applied},
		{"docs history pending", "POST", "/open-apis/docs_ai/v1/documents/doc/history/revert", `{}`, `{"status":"pending","task_id":"task"}`, Unknown},
		{"docs history running", "POST", "/open-apis/docs_ai/v1/documents/doc/history/revert", `{}`, `{"status":"running"}`, Unknown},
		{"docs history no guessed completion", "POST", "/open-apis/docs_ai/v1/documents/doc/history/revert", `{}`, `{"status":"completed"}`, Unknown},
		{"urgent app invalid users", "PATCH", "/open-apis/im/v1/messages/msg/urgent_app", `{}`, `{"invalid_user_id_list":["user"]}`, Unknown},
		{"urgent phone invalid users", "PATCH", "/open-apis/im/v1/messages/msg/urgent_phone", `{}`, `{"invalid_user_id_list":["user"]}`, Unknown},
		{"urgent sms empty invalid users", "PATCH", "/open-apis/im/v1/messages/msg/urgent_sms", `{}`, `{"invalid_user_id_list":[]}`, Applied},
		{"members invalid", "DELETE", "/open-apis/im/v1/chats/chat/members", `{}`, `{"invalid_id_list":["user"]}`, Unknown},
		{"members missing", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `{"not_existed_id_list":["user"]}`, Unknown},
		{"members pending", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `{"pending_approval_id_list":["user"]}`, Unknown},
		{"members success", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `{"invalid_id_list":[],"pending_approval_id_list":[]}`, Applied},
		{"settings invalid", "POST", "/open-apis/im/v1/chats/chat/chat_user_setting/batch_update", `{}`, `{"invalid_ids":[{"id":"user","msg":"denied"}]}`, Unknown},
		{"settings success", "POST", "/open-apis/im/v1/chats/chat/chat_user_setting/batch_update", `{}`, `{"invalid_ids":[]}`, Applied},
		{"membership failure", "POST", "/open-apis/im/v1/chats/chat/membership_applications/handle", `{}`, `{"results":[{"result":"success"},{"result":"failed"}]}`, Unknown},
		{"minutes complete", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{"replace_word_counts":[{"source_word":"first","replace_count":"2"},{"source_word":"second","replace_count":1}]}`, Applied},
		{"minutes mixed", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{"replace_word_counts":[{"source_word":"first","replace_count":"2"},{"source_word":"second","replace_count":"0"}]}`, Unknown},
		{"minutes zeros", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{"replace_word_counts":[{"source_word":"first","replace_count":0},{"source_word":"second","replace_count":0}]}`, Unknown},
		{"minutes absent", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{}`, Unknown},
		{"minutes partial coverage", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{"replace_word_counts":[{"source_word":"first","replace_count":2}]}`, Unknown},
		{"minutes duplicate", "PUT", "/open-apis/minutes/v1/minutes/minute/transcript/word", words, `{"replace_word_counts":[{"source_word":"first","replace_count":2},{"source_word":"first","replace_count":1}]}`, Unknown},
		{"base complete IDs", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one","two"]}`, Applied},
		{"base ignored fields", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one","two"],"ignored_fields":["field"]}`, Unknown},
		{"base duplicate IDs", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one","one"]}`, Unknown},
		{"base missing IDs", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one"]}`, Unknown},
		{"base extra async state", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one","two"],"task_id":"task"}`, Unknown},
		{"base empty ignored fields", "POST", "/open-apis/base/v3/bases/base/tables/table/records/batch_create", records, `{"record_id_list":["one","two"],"ignored_fields":[]}`, Applied},
		{"sql mixed failure", "POST", "/open-apis/spark/v1/apps/app/sql_commands", `{}`, `{"result":"[{\"sql_type\":\"INSERT\",\"affected_rows\":1},{\"sql_type\":\"ERROR\",\"data\":\"failure\"}]"}`, Unknown},
		{"sql malformed output", "POST", "/open-apis/spark/v1/apps/app/sql_commands", `{}`, `{"result":"bad"}`, Unknown},
		{"sql absent output", "POST", "/open-apis/spark/v1/apps/app/sql_commands", `{}`, `{}`, Unknown},
		{"sql complete statements", "POST", "/open-apis/spark/v1/apps/app/sql_commands", `{}`, `{"result":"[{\"sql_type\":\"INSERT\",\"affected_rows\":1}]"}`, Applied},
		{"vc mixed invitation", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1}]}`, `{"failed_count":1,"invited_count":1,"invite_results":[{"id":"one","status":1},{"id":"two","status":2}]}`, Unknown},
		{"vc complete invitation", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1}]}`, `{"failed_count":0,"invited_count":1,"invite_results":[{"id":"one","status":1}],"has_more":false}`, Applied},
		{"vc incomplete invitation", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1}]}`, `{"failed_count":0,"invited_count":2,"invite_results":[{"id":"one","status":1}]}`, Unknown},
		{"vc truncated invitation", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1}]}`, `{"failed_count":0,"invited_count":1,"invite_results":[{"id":"one","status":1}],"has_more":true}`, Unknown},
		{"wiki move docs task", "POST", "/open-apis/wiki/v2/spaces/space/nodes/move_docs_to_wiki", `{}`, `{"task_id":"task"}`, Unknown},
		{"wiki move node task", "POST", "/open-apis/wiki/v2/nodes/node/move_wiki_to_docs", `{}`, `{"task_id":"task"}`, Unknown},
		{"wiki delete node task", "DELETE", "/open-apis/wiki/v2/spaces/space/nodes/node", `{}`, `{"task_id":"task"}`, Unknown},
		{"wiki delete space task", "DELETE", "/open-apis/wiki/v2/spaces/space", `{}`, `{"task_id":"task"}`, Unknown},
		{"wiki synchronous response without task ID", "POST", "/open-apis/wiki/v2/spaces/space/nodes/move_docs_to_wiki", `{}`, `{}`, Applied},
		{"wiki synchronous response empty task ID", "DELETE", "/open-apis/wiki/v2/spaces/space/nodes/node", `{}`, `{"task_id":""}`, Applied},
		{"slides failed index zero", "POST", "/open-apis/slides_ai/v1/xml_presentations/presentation/slide/replace", `{}`, `{"revision_id":3,"failed_part_index":0}`, Unknown},
		{"slides failed reason", "POST", "/open-apis/slides_ai/v1/xml_presentations/presentation/slide/replace", `{}`, `{"failed_reason":"failure"}`, Unknown},
		{"slides advisory success", "POST", "/open-apis/slides_ai/v1/xml_presentations/presentation/slide/replace", `{}`, `{"revision_id":3,"issues":["advisory"]}`, Applied},
		{"slides native history task", "POST", "/open-apis/slides_ai/v1/xml_presentations/presentation/history/revert", `{}`, `{"task_id":"task"}`, Unknown},
		{"drive import ticket", "POST", "/open-apis/drive/v1/import_tasks", `{}`, `{"ticket":"ticket"}`, Unknown},
		{"sheets nested batch failure", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"batch_update"}`, `{"output":"{\"results\":[{\"success\":false,\"error\":\"failed\"}]}"}`, Unknown},
		{"sheets batch incomplete positive", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"batch_update"}`, `{"output":"{\"results\":[{\"success\":true}]}"}`, Unknown},
		{"sheets tool complete response", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"set_range"}`, `{"output":"{\"updated_cells\":1}"}`, Applied},
		{"sheets tool malformed output", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"set_range"}`, `{"output":"malformed"}`, Unknown},
		{"sheets tool empty output", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"set_range"}`, `{"output":"{}"}`, Unknown},
		{"base create ignored fields", "POST", "/open-apis/base/v3/bases/base/tables/table/records", `{}`, `{"record_id":"one","ignored_fields":["computed"]}`, Unknown},
		{"base update ignored fields", "PATCH", "/open-apis/base/v3/bases/base/tables/table/records/record", `{}`, `{"record_id":"one","ignored_fields":["computed"]}`, Unknown},
		{"base update malformed ignored fields", "PATCH", "/open-apis/base/v3/bases/base/tables/table/records/record", `{}`, `{"record_id":"one","ignored_fields":"computed"}`, Unknown},
		{"base update empty ignored fields", "PATCH", "/open-apis/base/v3/bases/base/tables/table/records/record", `{}`, `{"record_id":"one","ignored_fields":[]}`, Applied},
		{"IM member missing evidence", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `{}`, Unknown},
		{"IM member null data", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `null`, Unknown},
		{"IM member null arrays", "POST", "/open-apis/im/v1/chats/chat/members", `{}`, `{"invalid_id_list":null}`, Unknown},
		{"IM urgency unrelated array", "PATCH", "/open-apis/im/v1/messages/msg/urgent_app", `{}`, `{"invalid_id_list":[]}`, Unknown},
		{"VC omitted target", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1},{"id":"two","user_type":1}]}`, `{"failed_count":0,"invited_count":1,"invite_results":[{"id":"one","status":1}],"has_more":false}`, Unknown},
		{"VC mismatched target", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":2,"invitees":[{"id":"one","user_type":1}]}`, `{"failed_count":0,"invited_count":1,"invite_results":[{"id":"unrelated","status":1}],"has_more":false}`, Unknown},
		{"VC full target selection unproven", "POST", "/open-apis/vc/v1/bots/invite", `{"invite_type":1}`, `{"failed_count":0,"invited_count":1,"invite_results":[{"id":"one","status":1}],"has_more":false}`, Unknown},
		{"sheets history task acknowledgement", "POST", "/open-apis/sheet_ai/v2/spreadsheets/sheet/tools/invoke_write", `{"tool_name":"history_revert"}`, `{"output":"{\"transaction_id\":\"task\"}"}`, Unknown},
		{"endpoint scope", "POST", "/open-apis/fixture/create", `{}`, `{"result":"failed","invalid_user_id_list":["user"]}`, Applied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := `{"code":0,"data":` + tt.data + `}`
			calls := 0
			tr, store, journal := testTransport(t, func(req *http.Request) (*http.Response, error) {
				calls++
				body, err := io.ReadAll(req.Body)
				if err != nil || string(body) != tt.body {
					t.Fatalf("request changed %s %v", body, err)
				}
				return response(200, original), nil
			})
			req, err := http.NewRequest(tt.method, "https://fixture.invalid"+tt.path, strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := tr.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || string(body) != original || calls != 1 {
				t.Fatalf("response changed/retried: %s %v %d", body, err, calls)
			}
			receipts, err := store.List()
			if err != nil || len(receipts) != 1 || receipts[0].Outcome != tt.outcome || journal.Summary().Outcome != tt.outcome {
				t.Fatalf("%+v %v", receipts, err)
			}
		})
	}
}

func TestBatchEvidenceCannotConsumeUnclonableRequest(t *testing.T) {
	tr, _, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		return response(200, `{"code":0,"data":{"record_id_list":["one"]}}`), nil
	})
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/base/v3/bases/base/tables/table/records/batch_create", strings.NewReader(`{"create_records":[{"f":"private"}]}`))
	req.GetBody = nil
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if journal.Summary().Outcome != Unknown {
		t.Fatal(journal.Summary())
	}
	body, err := io.ReadAll(req.Body)
	if err != nil || string(body) != `{"create_records":[{"f":"private"}]}` {
		t.Fatalf("consumed request: %s %v", body, err)
	}
}
