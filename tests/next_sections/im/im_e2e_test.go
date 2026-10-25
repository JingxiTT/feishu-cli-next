// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package im_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }
func respond(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func success(t *testing.T, r nextfixture.Result) {
	t.Helper()
	if r.ExitCode != 0 || !gjson.Get(r.Stdout, "ok").Bool() {
		t.Fatalf("%+v", r)
	}
}

func TestMessageContentDeliveryAndRepeatedSend(t *testing.T) {
	cases := []struct {
		name          string
		flags         []string
		kind, content string
	}{
		{"text", []string{"--text", "你好 <at user_id=\"ou_fixture\">Fixture</at>"}, "text", `{"text":"你好 <at user_id=\"ou_fixture\">Fixture</at>"}`},
		{"post", []string{"--msg-type", "post", "--content", `{"en_us":{"title":"Rich","content":[[{"tag":"text","text":"α","style":["bold","italic"]},{"tag":"a","text":"link","href":"https://example.com"}]],"files":[{"file_key":"file_fixture"}]}}`}, "post", ""},
		{"image", []string{"--image", "img_fixture"}, "image", `{"image_key":"img_fixture"}`},
		{"file", []string{"--file", "file_fixture"}, "file", `{"file_key":"file_fixture"}`},
		{"audio", []string{"--audio", "file_fixture"}, "audio", `{"file_key":"file_fixture"}`},
		{"video", []string{"--video", "file_fixture", "--video-cover", "img_fixture"}, "media", `{"file_key":"file_fixture","image_key":"img_fixture"}`},
		{"card", []string{"--msg-type", "interactive", "--content", `{"config":{"wide_screen_mode":true},"elements":[{"tag":"markdown","content":"hello"}]}`}, "interactive", ""},
		{"share chat", []string{"--msg-type", "share_chat", "--content", `{"chat_id":"oc_shared_fixture"}`}, "share_chat", ""},
		{"share user", []string{"--msg-type", "share_user", "--content", `{"user_id":"ou_shared_fixture"}`}, "share_user", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writes.Add(1)
				if r.Method != "POST" || r.URL.Path != "/open-apis/im/v1/messages" || r.URL.Query().Get("receive_id_type") != "open_id" {
					t.Errorf("request %s %s", r.Method, r.URL)
				}
				var body struct {
					ReceiveID string `json:"receive_id"`
					Kind      string `json:"msg_type"`
					Content   string `json:"content"`
					UUID      string `json:"uuid"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				expected := tc.content
				if expected == "" {
					expected = tc.flags[3]
				}
				var got, want any
				_ = json.Unmarshal([]byte(body.Content), &got)
				_ = json.Unmarshal([]byte(expected), &want)
				if body.ReceiveID != "ou_fixture" || body.Kind != tc.kind || !reflect.DeepEqual(got, want) || body.UUID != "" {
					t.Errorf("delivery changed: %+v expected %s", body, expected)
				}
				respond(w, map[string]any{"message_id": fmt.Sprintf("om_fixture_%d", writes.Load()), "chat_id": "oc_fixture", "create_time": "1700000000000"})
			}))
			args := append([]string{"im", "+messages-send", "--user-id", "ou_fixture"}, tc.flags...)
			success(t, f.Run(args...))
			success(t, f.Run(args...))
			if writes.Load() != 2 {
				t.Fatal("legitimate repeated send deduplicated")
			}
		})
	}
}

func TestEditPreflightPreservesRichReplacementAndClear(t *testing.T) {
	for _, tc := range []struct {
		name, kind, current string
		flags               []string
		writes              int
	}{
		{"text unchanged", "text", `{"text":"same"}`, []string{"--text", "same"}, 0},
		{"post unchanged", "post", `{"en_us":{"title":"","content":[[{"tag":"text","text":"same","style":["bold"]}]],"files":[]}}`, []string{"--msg-type", "post", "--content", `{"en_us":{"files":[],"content":[[{"text":"same","tag":"text","style":["bold"]}]],"title":""}}`}, 0},
		{"text changed", "text", `{"text":"before"}`, []string{"--text", "after"}, 1},
		{"explicit clear", "post", `{"zh_cn":{"content":[],"files":[{"file_key":"file_old"}]}}`, []string{"--msg-type", "post", "--clear-attachments"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reads, writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/open-apis/im/v1/messages/om_fixture" {
					t.Errorf("target %s", r.URL)
				}
				if r.Method == "GET" {
					reads.Add(1)
					respond(w, map[string]any{"items": []any{map[string]any{"message_id": "om_fixture", "chat_id": "oc_fixture", "msg_type": tc.kind, "body": map[string]string{"content": tc.current}}}})
					return
				}
				writes.Add(1)
				if r.Method != "PUT" {
					t.Errorf("method %s", r.Method)
				}
				var body struct {
					Content string `json:"content"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				if tc.name == "explicit clear" && !gjson.Get(body.Content, "files").IsArray() {
					t.Errorf("clear lost: %s", body.Content)
				}
				if tc.name == "text changed" && gjson.Get(body.Content, "text").String() != "after" {
					t.Errorf("replacement %s", body.Content)
				}
				respond(w, map[string]any{"message_id": "om_fixture", "chat_id": "oc_fixture"})
			}))
			args := append([]string{"im", "+messages-edit", "--message-id", "om_fixture", "--skip-unchanged", "--result-fields", "nothing"}, tc.flags...)
			r := f.Run(args...)
			success(t, r)
			if reads.Load() != 1 || int(writes.Load()) != tc.writes {
				t.Fatalf("reads %d writes %d", reads.Load(), writes.Load())
			}
			if tc.writes == 0 && (!gjson.Get(r.Stdout, "data.unchanged").Bool() || strings.Contains(r.Stderr, "confirmed_applied")) {
				t.Fatalf("no-op evidence %+v", r)
			}
		})
	}
}

func TestEditPreflightFailureDoesNotWrite(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writes.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic missing read permission"}`)
	}))
	r := f.Run("im", "+messages-edit", "--message-id", "om_fixture", "--text", "same", "--skip-unchanged")
	if r.ExitCode == 0 || writes.Load() != 0 || !strings.Contains(r.Stderr, "99991679") {
		t.Fatalf("%+v writes %d", r, writes.Load())
	}
}

func TestRecipientsAndConflictsRejectBeforeDispatch(t *testing.T) {
	for _, args := range [][]string{
		{"+messages-send", "--chat-id", "oc_fixture", "--user-id", "ou_fixture", "--text", "hello"},
		{"+messages-send", "--user-id", "ambiguous name", "--text", "hello"},
		{"+messages-send", "--user-id", "ou_fixture", "--text", "hello", "--markdown", "other"},
		{"+messages-reply", "--message-id", "invalid", "--text", "hello"},
		{"+messages-edit", "--message-id", "om_fixture", "--msg-type", "post", "--clear-attachments", "--set-attachments", "file_fixture"},
		{"+chat-create", "--users", "ou_fixture,wrong"},
		{"+chat-create", "--owner", "ambiguous"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			r := f.Run(append([]string{"im"}, args...)...)
			if r.ExitCode == 0 || gjson.Get(r.Stderr, "error.type").String() != "validation" {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestUnknownSendIsNotRetried(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		c, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = c.Close()
	}))
	r := f.Run("im", "+messages-send", "--user-id", "ou_fixture", "--text", "once", "--idempotency-key", "synthetic-explicit-key")
	if r.ExitCode != 22 || writes.Load() != 1 || !strings.Contains(r.Stderr, "unknown") {
		t.Fatalf("%+v writes %d", r, writes.Load())
	}
}

func TestBoundedChatReadArtifact(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" || r.URL.Path != "/open-apis/im/v1/chats" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		respond(w, map[string]any{"items": []any{map[string]any{"chat_id": "oc_fixture", "name": "private synthetic", "description": "full preserved"}}, "has_more": true, "page_token": "next_fixture"})
	}))
	r := f.Run("im", "+chat-list", "--as", "bot", "--page-all", "--page-limit", "1", "--save-result", "chats.json", "--result-fields", "name")
	success(t, r)
	if calls.Load() != 1 || !strings.Contains(r.Stdout, "next_fixture") {
		t.Fatalf("bounded result %+v calls %d", r, calls.Load())
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "chats.json"))
	if err != nil || !strings.Contains(string(raw), "full preserved") {
		t.Fatalf("artifact %s %v", raw, err)
	}
}

func TestLiveIMBoundary(t *testing.T) {
	t.Skip("blocked live: current account/tenant, private destination and delivery restricted to the verified configured recipient have not been verified; no live content dispatched")
}

func TestChatUpdateNoopAndExplicitEmptyDescription(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			respond(w, map[string]any{"name": "same", "description": "old"})
			return
		}
		writes.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if value, ok := body["description"]; !ok || value != "" || len(body) != 1 {
			t.Errorf("clear request %#v", body)
		}
		respond(w, map[string]any{})
	}))
	r := f.Run("im", "+chat-update", "--chat-id", "oc_fixture", "--name", "same", "--skip-unchanged")
	success(t, r)
	if !gjson.Get(r.Stdout, "data.unchanged").Bool() || writes.Load() != 0 {
		t.Fatalf("%+v", r)
	}
	success(t, f.Run("im", "+chat-update", "--chat-id", "oc_fixture", "--description", "", "--skip-unchanged"))
	if writes.Load() != 1 {
		t.Fatal("explicit clear not dispatched")
	}
}

func TestReadShortcutRoutesAndEmptyResults(t *testing.T) {
	for _, tc := range []struct {
		command, method, path string
		flags                 []string
	}{
		{"+chat-members-list", "GET", "/open-apis/im/v1/chats/oc_fixture/members/list", []string{"--chat-id", "oc_fixture", "--member-types", "user,bot", "--page-size", "2"}},
		{"+chat-messages-list", "GET", "/open-apis/im/v1/messages", []string{"--chat-id", "oc_fixture", "--no-reactions"}},
		{"+threads-messages-list", "GET", "/open-apis/im/v1/messages", []string{"--thread", "omt_fixture", "--no-reactions"}},
		{"+messages-mget", "GET", "/open-apis/im/v1/messages/mget", []string{"--message-ids", "om_fixture", "--no-reactions"}},
		{"+messages-read-status", "POST", "/open-apis/im/v1/messages/read_status", []string{"--message-ids", "om_fixture"}},
		{"+message-read-users", "GET", "/open-apis/im/v1/messages/om_fixture/read_users", []string{"--message-id", "om_fixture"}},
		{"+chat-search", "POST", "/open-apis/im/v2/chats/search", []string{"--query", "synthetic", "--chat-modes", "topic", "--member-ids", "ou_fixture"}},
		{"+messages-search", "POST", "/open-apis/im/v1/messages/search", []string{"--query", "synthetic", "--chat-id", "oc_fixture", "--sender", "ou_fixture", "--no-reactions"}},
		{"+flag-list", "GET", "/open-apis/im/v1/flags", []string{"--enrich-feed-thread=false"}},
		{"+feed-shortcut-list", "GET", "/open-apis/im/v2/feed_shortcuts", []string{"--no-detail"}},
		{"+feed-group-list", "GET", "/open-apis/im/v1/groups", nil},
		{"+feed-group-list-item", "GET", "/open-apis/im/v1/groups/ofg_fixture/list_item", []string{"--feed-group-id", "ofg_fixture"}},
		{"+feed-group-query-item", "POST", "/open-apis/im/v1/groups/ofg_fixture/batch_query_item", []string{"--feed-group-id", "ofg_fixture", "--feed-id", "oc_fixture"}},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.method || r.URL.Path != tc.path {
					t.Errorf("%s %s", r.Method, r.URL)
				}
				respond(w, map[string]any{"items": []any{}, "has_more": false})
			}))
			args := append([]string{"im", tc.command, "--as", "user"}, tc.flags...)
			r := f.Run(args...)
			success(t, r)
			if calls.Load() != 1 || strings.Contains(r.Stderr, "\"mutation\"") {
				t.Fatalf("read was journaled or repeated %+v calls %d", r, calls.Load())
			}
		})
	}
}

func TestReplyAndPrivateChatEffects(t *testing.T) {
	t.Run("thread reply", func(t *testing.T) {
		var calls atomic.Int32
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method != "POST" || r.URL.Path != "/open-apis/im/v1/messages/om_fixture/reply" {
				t.Errorf("%s %s", r.Method, r.URL)
			}
			raw, _ := io.ReadAll(r.Body)
			if !gjson.GetBytes(raw, "reply_in_thread").Bool() || gjson.GetBytes(raw, "uuid").String() != "explicit_fixture" || gjson.GetBytes(raw, "content").String() != `{"text":"reply α"}` {
				t.Errorf("reply %s", raw)
			}
			respond(w, map[string]any{"message_id": "om_reply", "chat_id": "oc_fixture"})
		}))
		success(t, f.Run("im", "+messages-reply", "--message-id", "om_fixture", "--text", "reply α", "--reply-in-thread", "--idempotency-key", "explicit_fixture"))
		if calls.Load() != 1 {
			t.Fatal(calls.Load())
		}
	})
	t.Run("chat creation invites and share link", func(t *testing.T) {
		var calls atomic.Int32
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path == "/open-apis/im/v1/chats/oc_fixture/link" {
				respond(w, map[string]any{"share_link": "https://example.com/synthetic"})
				return
			}
			raw, _ := io.ReadAll(r.Body)
			if r.URL.Path != "/open-apis/im/v1/chats" || r.Method != "POST" || gjson.GetBytes(raw, "chat_type").String() != "private" || gjson.GetBytes(raw, "chat_mode").String() != "topic" || gjson.GetBytes(raw, "user_id_list.0").String() != "ou_fixture" || gjson.GetBytes(raw, "owner_id").String() != "ou_fixture" {
				t.Errorf("create %s %s %s", r.Method, r.URL, raw)
			}
			respond(w, map[string]any{"chat_id": "oc_fixture", "name": "synthetic", "chat_type": "private"})
		}))
		success(t, f.Run("im", "+chat-create", "--name", "synthetic", "--chat-mode", "topic", "--type", "private", "--users", "ou_fixture", "--owner", "ou_fixture"))
		if calls.Load() != 2 {
			t.Fatal(calls.Load())
		}
	})
}

func TestResourceDownloadWritesExactBytes(t *testing.T) {
	payload := strings.Repeat("synthetic resource α\n", 100)
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/open-apis/im/v1/messages/om_fixture/resources/file_fixture" || r.URL.Query().Get("type") != "file" || r.Header.Get("Range") == "" {
			t.Errorf("%s %s range %s", r.Method, r.URL, r.Header.Get("Range"))
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="resource.bin"`)
		_, _ = io.WriteString(w, payload)
	}))
	success(t, f.Run("im", "+messages-resources-download", "--message-id", "om_fixture", "--file-key", "file_fixture", "--type", "file", "--output", "resource.bin"))
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "resource.bin"))
	if err != nil || string(raw) != payload {
		t.Fatalf("bytes %d %v", len(raw), err)
	}
}

func TestFlagAndFeedShortcutWrites(t *testing.T) {
	for _, tc := range []struct {
		command, path string
		flags         []string
		field         string
	}{
		{"+flag-create", "/open-apis/im/v1/flags", []string{"--message-id", "om_fixture"}, "flag_items.0.item_id"},
		{"+flag-cancel", "/open-apis/im/v1/flags/cancel", []string{"--message-id", "om_fixture", "--flag-type", "message"}, "flag_items.0.item_id"},
		{"+feed-shortcut-create", "/open-apis/im/v2/feed_shortcuts", []string{"--chat-id", "oc_fixture", "--tail"}, "shortcuts.0.feed_card_id"},
		{"+feed-shortcut-remove", "/open-apis/im/v2/feed_shortcuts/remove", []string{"--chat-id", "oc_fixture"}, "shortcuts.0.feed_card_id"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				raw, _ := io.ReadAll(r.Body)
				if r.Method != "POST" || r.URL.Path != tc.path {
					t.Errorf("%s %s", r.Method, r.URL)
				}
				value := gjson.GetBytes(raw, tc.field).String()
				if value != "oc_fixture" && value != "om_fixture" {
					t.Errorf("target lost: %s", raw)
				}
				if tc.command == "+feed-shortcut-create" && gjson.GetBytes(raw, "is_header").Bool() {
					t.Errorf("tail changed: %s", raw)
				}
				respond(w, map[string]any{})
			}))
			r := f.Run(append([]string{"im", tc.command, "--as", "user"}, tc.flags...)...)
			success(t, r)
			if calls.Load() != 1 {
				t.Fatal(calls.Load())
			}
		})
	}
}

func TestImageAndFileUploadBytes(t *testing.T) {
	for _, tc := range []struct{ resource, field, data string }{
		{"images", "image", `{"image_type":"message"}`},
		{"files", "file", `{"file_type":"stream","file_name":"fixture.bin"}`},
	} {
		t.Run(tc.resource, func(t *testing.T) {
			payload := []byte("synthetic upload\x00α")
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/open-apis/im/v1/"+tc.resource {
					t.Errorf("%s %s", r.Method, r.URL)
				}
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				file, _, err := r.FormFile(tc.field)
				if err != nil {
					t.Error(err)
					return
				}
				defer file.Close()
				got, _ := io.ReadAll(file)
				if string(got) != string(payload) {
					t.Errorf("upload corrupted %q", got)
				}
				if tc.resource == "images" && r.FormValue("image_type") != "message" {
					t.Error("image usage changed")
				}
				respond(w, map[string]any{tc.field + "_key": tc.field + "_fixture"})
			}))
			if err := os.WriteFile(filepath.Join(f.WorkDir, "fixture.bin"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			success(t, f.Run("im", tc.resource, "create", "--file", tc.field+"=fixture.bin", "--data", tc.data))
		})
	}
}

func TestPartialRecipientAndBatchOutcomesRemainVisible(t *testing.T) {
	for _, tc := range []struct{ resource, method, path, params, data, response string }{
		{"messages", "urgent_app", "/open-apis/im/v1/messages/om_fixture/urgent_app", `{"message_id":"om_fixture","user_id_type":"open_id"}`, `{"user_id_list":["ou_fixture","ou_second_fixture"]}`, `{"invalid_user_id_list":["ou_second_fixture"]}`},
		{"chat.members", "create", "/open-apis/im/v1/chats/oc_fixture/members", `{"chat_id":"oc_fixture","member_id_type":"open_id"}`, `{"id_list":["ou_fixture","ou_second_fixture"]}`, `{"pending_approval_id_list":["ou_second_fixture"]}`},
		{"chat.user_setting", "batch_update", "/open-apis/im/v1/chat_user_setting/batch_update", `{}`, `{"chat_settings":[{"chat_id":"oc_fixture","is_muted":false}]}`, `{"invalid_ids":[{"id":"oc_fixture","msg":"not_a_member"}]}`},
	} {
		t.Run(tc.resource+tc.method, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != tc.path {
					t.Errorf("target %s", r.URL)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":`+tc.response+`}`)
			}))
			identity := "bot"
			if tc.resource == "chat.user_setting" {
				identity = "user"
			}
			r := f.Run("im", tc.resource, tc.method, "--params", tc.params, "--data", tc.data, "--as", identity, "--result-fields", "nothing")
			if r.ExitCode != 22 || calls.Load() != 1 || gjson.Get(r.Stdout, "ok").Bool() || !strings.Contains(r.Stdout, "ou_second_fixture") && !strings.Contains(r.Stdout, "not_a_member") {
				t.Fatalf("partial results hidden: %+v calls %d", r, calls.Load())
			}
		})
	}
}

func TestFlagDoubleCancelPartialEffects(t *testing.T) {
	var writes atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/im/v1/messages/om_fixture":
			respond(w, map[string]any{"items": []any{map[string]any{"message_id": "om_fixture", "chat_id": "oc_fixture"}}})
		case "/open-apis/im/v1/chats/oc_fixture":
			respond(w, map[string]any{"chat_mode": "topic"})
		case "/open-apis/im/v1/flags/cancel":
			if writes.Add(1) == 1 {
				respond(w, map[string]any{})
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic rejected second layer"}`)
		default:
			t.Errorf("unexpected %s", r.URL)
		}
	}))
	r := f.Run("im", "+flag-cancel", "--message-id", "om_fixture", "--as", "user", "--result-fields", "nothing")
	if r.ExitCode != 21 || writes.Load() != 2 || gjson.Get(r.Stdout, "ok").Bool() || !strings.Contains(r.Stdout, "failed") || !strings.Contains(r.Stderr, "partial") {
		t.Fatalf("%+v writes %d", r, writes.Load())
	}
}

func TestOptInPreflightRejectsUnusableTarget(t *testing.T) {
	for _, data := range []any{
		map[string]any{"items": []any{}},
		map[string]any{"items": []any{map[string]any{"message_id": "om_other", "msg_type": "text", "body": map[string]string{"content": `{"text":"same"}`}}}},
		map[string]any{"items": []any{map[string]any{"message_id": "om_fixture", "deleted": true, "msg_type": "text", "body": map[string]string{"content": `{"text":"same"}`}}}},
	} {
		t.Run(fmt.Sprint(data), func(t *testing.T) {
			var writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					writes.Add(1)
				}
				respond(w, data)
			}))
			r := f.Run("im", "+messages-edit", "--message-id", "om_fixture", "--text", "same", "--skip-unchanged")
			if r.ExitCode == 0 || writes.Load() != 0 {
				t.Fatalf("unusable target edited %+v writes %d", r, writes.Load())
			}
		})
	}
}

func TestEditDryRunPreflightAndWriteOnlyCompatibility(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	r := f.Run("im", "+messages-edit", "--message-id", "om_fixture", "--text", "same", "--skip-unchanged", "--dry-run")
	success(t, r)
	if !strings.Contains(r.Stdout, "GET") || !strings.Contains(r.Stdout, "PUT") {
		t.Fatalf("missing preview: %+v", r)
	}
	f = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("unsolicited read %s", r.Method)
		}
		respond(w, map[string]any{"message_id": "om_fixture"})
	}))
	success(t, f.Run("im", "+messages-edit", "--message-id", "om_fixture", "--text", "same"))
}
