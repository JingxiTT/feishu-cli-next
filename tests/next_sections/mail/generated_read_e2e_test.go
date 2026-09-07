// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMailGeneratedBatchGetReadContract(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		t.Run(identity, func(t *testing.T) {
			const mailbox = "fixture+read@example.invalid"
			const messageID = "0123456789abcdef"
			var reads atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.Method != "POST" || r.URL.Path != "/open-apis/mail/v1/user_mailboxes/"+mailbox+"/messages/batch_get" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if got := r.Header.Get("X-Lark-Proxy-Identity"); got != identity {
					t.Errorf("identity %s want %s", got, identity)
				}
				var body struct {
					Format string   `json:"format"`
					IDs    []string `json:"message_ids"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Format != "full" || len(body.IDs) != 1 || body.IDs[0] != messageID {
					t.Errorf("changed read input %#v", body)
				}
				reply(w, map[string]any{"messages": []any{map[string]any{"message_id": messageID, "subject": "邮件 001", "body_html": "<p>Keep full content</p>", "nullable": nil, "numeric_id": json.Number("9007199254740993")}}})
			}))
			args := []string{"mail", "user_mailbox.messages", "batch_get", "--params", `{"user_mailbox_id":"` + mailbox + `"}`, "--data", `{"format":"full","message_ids":["` + messageID + `"]}`, "--as", identity}
			t.Run("ordinary read", func(t *testing.T) {
				result := fixture.Run(args...)
				if result.ExitCode != 0 || strings.Contains(result.Stderr, `"mutation"`) || strings.Contains(result.Stdout, `"mutation"`) {
					t.Fatalf("read misclassified: %+v", result)
				}
				if !strings.Contains(result.Stdout, "9007199254740993") || !strings.Contains(result.Stdout, "Keep full content") {
					t.Fatalf("lost read data: %+v", result)
				}
			})
			t.Run("full artifact with projection", func(t *testing.T) {
				result := fixture.Run(append(append([]string{}, args...), "--result-fields", "subject", "--save-result", "mail-read.json")...)
				if result.ExitCode != 0 || strings.Contains(result.Stderr, `"mutation"`) {
					t.Fatalf("read artifact rejected: %+v", result)
				}
				raw, err := os.ReadFile(filepath.Join(fixture.WorkDir, "mail-read.json"))
				if err != nil {
					t.Fatal(err)
				}
				var artifact struct {
					Schema string `json:"schema"`
					Result struct {
						OK   bool `json:"ok"`
						Data struct {
							Messages []struct {
								ID     string      `json:"message_id"`
								HTML   string      `json:"body_html"`
								Number json.Number `json:"numeric_id"`
							} `json:"messages"`
						} `json:"data"`
					} `json:"result"`
				}
				if err = json.Unmarshal(raw, &artifact); err != nil {
					t.Fatal(err)
				}
				if artifact.Schema != "feishu-cli-next/result/v1" || !artifact.Result.OK || len(artifact.Result.Data.Messages) != 1 {
					t.Fatalf("invalid read artifact: %s", raw)
				}
				message := artifact.Result.Data.Messages[0]
				if message.ID != messageID || message.HTML != "<p>Keep full content</p>" || message.Number.String() != "9007199254740993" {
					t.Fatalf("artifact lost full values: %s", raw)
				}
			})
			receipts, err := filepath.Glob(filepath.Join(fixture.ConfigDir, "next", "receipts", "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if reads.Load() != 2 || len(receipts) != 0 {
				t.Fatalf("reads %d, unexpected receipts %v", reads.Load(), receipts)
			}
		})
	}
}

func TestMailGeneratedReadExceptionDoesNotCoverWritesOrRawPOST(t *testing.T) {
	for _, tc := range []struct {
		name, path string
		args       []string
	}{
		{"neighboring mutation", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_modify", []string{"mail", "user_mailbox.messages", "batch_modify", "--params", `{"user_mailbox_id":"me"}`, "--data", `{"message_ids":["0123456789abcdef"],"add_label_ids":["UNREAD"]}`, "--as", "user"}},
		{"raw POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get", []string{"api", "POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get", "--data", `{"message_ids":["0123456789abcdef"]}`, "--as", "user"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != tc.path {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
				reply(w, map[string]any{"messages": []any{map[string]any{"message_id": "0123456789abcdef"}}})
			}))
			result := fixture.Run(tc.args...)
			receipts, err := filepath.Glob(filepath.Join(fixture.ConfigDir, "next", "receipts", "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if result.ExitCode != 22 || !strings.Contains(result.Stderr, `"outcome":"unknown"`) || calls.Load() != 1 || len(receipts) != 1 {
				t.Fatalf("broadened read exception: %+v calls %d receipts %v", result, calls.Load(), receipts)
			}
		})
	}
}

// These native search methods accept user tokens only; their POST bodies are
// read filters, and they share neither write receipts nor raw API waivers.
func TestMailGeneratedSearchReadContracts(t *testing.T) {
	for _, tc := range []struct {
		name, path, params, body string
		args                     []string
		data                     map[string]any
	}{
		{
			name:   "contact search",
			path:   "/open-apis/mail/v1/multi_entity/search",
			params: `{"user_id_type":"open_id"}`,
			body:   `{"query":"联系人","size":2}`,
			args:   []string{"mail", "multi_entity", "search"},
			data:   map[string]any{"items": []any{map[string]any{"id": "001", "type": "user", "name": "联系人", "email": "fixture@example.invalid"}}},
		},
		{
			name:   "message search",
			path:   "/open-apis/mail/v1/user_mailboxes/fixture+read@example.invalid/search",
			params: `{"user_mailbox_id":"fixture+read@example.invalid","page_size":2,"page_token":"continue-synthetic"}`,
			body:   `{"query":"合同","filter":{"is_unread":false,"has_attachment":true,"folder":["inbox"]}}`,
			args:   []string{"mail", "user_mailboxes", "search"},
			data:   map[string]any{"items": []any{map[string]any{"id": "001", "display_info": "<h>合同</h>", "meta_data": map[string]any{"title": "合同 001", "thread_id": "0002"}}}, "has_more": true, "page_token": "next-synthetic", "total": json.Number("9007199254740993")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != tc.path || r.Header.Get("X-Lark-Proxy-Identity") != "user" {
					t.Errorf("unexpected request %s %s identity %s", r.Method, r.URL.Path, r.Header.Get("X-Lark-Proxy-Identity"))
				}
				var params map[string]any
				if err := json.Unmarshal([]byte(tc.params), &params); err != nil {
					t.Error(err)
					return
				}
				if _, ok := params["user_mailbox_id"]; ok {
					if r.URL.Query().Get("page_size") != "2" || r.URL.Query().Get("page_token") != "continue-synthetic" {
						t.Errorf("changed pagination: %s", r.URL.RawQuery)
					}
				} else if r.URL.Query().Get("user_id_type") != "open_id" {
					t.Errorf("changed ID type: %s", r.URL.RawQuery)
				}
				var got, want any
				if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal([]byte(tc.body), &want); err != nil {
					t.Error(err)
					return
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("changed read body: %#v want %#v", got, want)
				}
				reply(w, tc.data)
			}))
			args := append(append([]string{}, tc.args...), "--params", tc.params, "--data", tc.body, "--as", "user")
			for _, save := range []bool{false, true} {
				argv := append([]string{}, args...)
				if save {
					argv = append(argv, "--save-result", "search.json")
				}
				result := fixture.Run(argv...)
				if result.ExitCode != 0 || strings.Contains(result.Stdout, `"mutation"`) || strings.Contains(result.Stderr, `"mutation"`) {
					t.Fatalf("search misclassified: %+v", result)
				}
				content := result.Stdout
				if save {
					raw, err := os.ReadFile(filepath.Join(fixture.WorkDir, "search.json"))
					if err != nil {
						t.Fatal(err)
					}
					var artifact struct {
						Schema string          `json:"schema"`
						Result json.RawMessage `json:"result"`
					}
					if err = json.Unmarshal(raw, &artifact); err != nil {
						t.Fatal(err)
					}
					if artifact.Schema != "feishu-cli-next/result/v1" {
						t.Fatalf("unexpected artifact: %s", raw)
					}
					content = string(artifact.Result)
				}
				var envelope struct {
					OK   bool           `json:"ok"`
					Data map[string]any `json:"data"`
				}
				dec := json.NewDecoder(strings.NewReader(content))
				dec.UseNumber()
				if err := dec.Decode(&envelope); err != nil {
					t.Error(err)
					return
				}
				if !envelope.OK || !reflect.DeepEqual(envelope.Data, tc.data) {
					t.Fatalf("changed search results: %s", content)
				}
			}
			receipts, err := filepath.Glob(filepath.Join(fixture.ConfigDir, "next", "receipts", "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 || len(receipts) != 0 {
				t.Fatalf("search calls %d, unexpected receipts %v", calls.Load(), receipts)
			}
		})
	}
}
