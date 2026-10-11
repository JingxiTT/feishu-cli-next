// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail_test

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func data(t *testing.T, result nextfixture.Result) map[string]any {
	t.Helper()
	if result.ExitCode != 0 {
		t.Fatalf("CLI: %+v", result)
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &env); err != nil {
		t.Fatalf("decode %v: %+v", err, result)
	}
	return env.Data
}

// Every frozen mail command and nonhidden option must remain discoverable.
// Workflow tests below and the reference unit suite independently check behavior.
func TestFrozenMailDiscovery(t *testing.T) {
	file, err := os.Open("../../../baseline/v1.0.97/registrations.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	var snapshot map[string]struct {
		Commands map[string]struct {
			Flags map[string]struct {
				Hidden bool `json:"hidden"`
			} `json:"flags"`
		} `json:"commands"`
	}
	if err = json.NewDecoder(gz).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	fixture := nextfixture.NewFixture(t, nil)
	count := 0
	for command, contract := range snapshot["feishu"].Commands {
		if command != "mail" && !strings.HasPrefix(command, "mail ") {
			continue
		}
		count++
		t.Run(command, func(t *testing.T) {
			result := fixture.Run(append(strings.Fields(command), "--help")...)
			if result.ExitCode != 0 {
				t.Fatalf("%+v", result)
			}
			for name, flag := range contract.Flags {
				// The authsidecar build deliberately hides local profile selection.
				if name != "profile" && !flag.Hidden && !strings.Contains(result.Stdout, "--"+name) {
					t.Errorf("missing flag --%s", name)
				}
			}
		})
	}
	if count != 104 {
		t.Fatalf("frozen command count %d", count)
	}
}

func TestMailBatchReadArtifactPreservesRichContentAndMissingIDs(t *testing.T) {
	var calls atomic.Int32
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/messages/batch_get") {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			IDs []string `json:"message_ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.IDs) > 20 {
			t.Errorf("unbounded batch %d", len(body.IDs))
		}
		messages := []any{}
		for i := len(body.IDs) - 1; i >= 0; i-- {
			id := body.IDs[i]
			if id == "m21" {
				continue
			}
			messages = append(messages, map[string]any{"message_id": id, "subject": "Unicode 空值", "body_html": "<p><b>Rich</b> 正文</p>", "body_plain_text": "Rich 正文", "attachments": []any{map[string]any{"id": "a1", "filename": "附件.bin", "is_inline": false, "content_type": "application/octet-stream"}}})
		}
		reply(w, map[string]any{"messages": messages})
	}))
	ids := []string{}
	for i := 1; i <= 21; i++ {
		ids = append(ids, fmt.Sprintf("m%d", i))
	}
	result := fixture.Run("mail", "+messages", "--mailbox", "fixture@example.invalid", "--message-ids", strings.Join(ids, ","), "--as", "bot", "--result-fields", "subject", "--save-result", "mail.json")
	if result.ExitCode != 0 || calls.Load() != 2 {
		t.Fatalf("%+v calls %d", result, calls.Load())
	}
	raw, err := os.ReadFile(filepath.Join(fixture.WorkDir, "mail.json"))
	if err != nil {
		t.Fatal(err)
	}
	var artifact struct {
		Result struct {
			Data struct {
				Messages []map[string]any `json:"messages"`
				Missing  []string         `json:"unavailable_message_ids"`
			} `json:"data"`
		} `json:"result"`
	}
	if err = json.Unmarshal(raw, &artifact); err != nil {
		t.Fatal(err)
	}
	if len(artifact.Result.Data.Messages) != 20 || len(artifact.Result.Data.Missing) != 1 {
		t.Fatalf("artifact %s", raw)
	}
	for i, m := range artifact.Result.Data.Messages {
		if m["message_id"] != fmt.Sprintf("m%d", i+1) || m["body_html"] != "<p><b>Rich</b> 正文</p>" {
			t.Fatalf("lost/order content %#v", m)
		}
	}
	if strings.Contains(result.Stderr, "\"mutation\"") {
		t.Fatalf("POST read got receipt: %s", result.Stderr)
	}
}

const mimeDraft = "From: fixture@example.invalid\r\nTo: fixture@example.invalid\r\nSubject: Original\r\nX-Private: keep-exact\r\nMIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=fixture\r\n\r\n--fixture\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n<p><b>正文</b></p>\r\n--fixture\r\nContent-Type: application/octet-stream\r\nContent-Disposition: attachment; filename=fixture.bin\r\nContent-Transfer-Encoding: base64\r\n\r\nAAECA/8=\r\n--fixture--\r\n"

func TestMailDraftNoOpAndMIMEPreservingReplacement(t *testing.T) {
	var writes atomic.Int32
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/drafts/d1") {
			t.Errorf("unexpected %s", r.URL.Path)
		}
		if r.Method == "GET" {
			reply(w, map[string]any{"draft_id": "d1", "raw": base64.RawURLEncoding.EncodeToString([]byte(mimeDraft))})
			return
		}
		writes.Add(1)
		if r.Method != "PUT" {
			t.Errorf("unexpected method %s", r.Method)
		}
		var body struct {
			Raw string `json:"raw"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		decoded, err := base64.URLEncoding.DecodeString(body.Raw)
		if err != nil {
			t.Error(err)
		}
		for _, want := range []string{"Subject: Updated", "X-Private: keep-exact", "AAECA/8=", "<p><b>正文</b></p>", "filename=fixture.bin"} {
			if !strings.Contains(string(decoded), want) {
				t.Errorf("lost MIME %q in %s", want, decoded)
			}
		}
		reply(w, map[string]any{"draft_id": "d1"})
	}))
	noOp := data(t, fixture.Run("mail", "+draft-edit", "--draft-id", "d1", "--set-subject", "Original", "--as", "user"))
	if noOp["no_op"] != true || writes.Load() != 0 {
		t.Fatalf("no-op %v writes %d", noOp, writes.Load())
	}
	inspect := data(t, fixture.Run("mail", "+draft-edit", "--draft-id", "d1", "--inspect", "--as", "user"))
	if inspect["projection"] == nil || writes.Load() != 0 {
		t.Fatal(inspect)
	}
	updated := fixture.Run("mail", "+draft-edit", "--draft-id", "d1", "--set-subject", "Updated", "--as", "user")
	data(t, updated)
	if writes.Load() != 1 || !strings.Contains(updated.Stderr, "confirmed_applied") {
		t.Fatalf("%+v writes %d", updated, writes.Load())
	}
}

func TestMailDraftSendOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
		exit     int
		outcome  string
	}{
		{"accepted", `{"code":0,"data":{"message_id":"sent1","thread_id":"thread1"}}`, 0, "confirmed_applied"},
		{"scope", `{"code":99991679,"msg":"synthetic missing scope"}`, 20, "confirmed_unapplied"},
		{"malformed", `{"broken":`, 22, "unknown"},
		{"lost", "", 22, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sends atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sends.Add(1)
				if r.Method != "POST" || !strings.HasSuffix(r.URL.Path, "/drafts/d1/send") {
					t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
				}
				if tc.name == "lost" {
					connection, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = connection.Close()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.response)
			}))
			result := fixture.Run("mail", "+draft-send", "--draft-id", "d1", "--as", "user", "--yes")
			if result.ExitCode != tc.exit || sends.Load() != 1 || !strings.Contains(result.Stderr, tc.outcome) {
				t.Fatalf("%+v sends %d", result, sends.Load())
			}
		})
	}
}

func TestMailInvalidInputAndNoChangeDoNotWrite(t *testing.T) {
	fixture := nextfixture.NewFixture(t, nil)
	noOp := data(t, fixture.Run("mail", "+message-modify", "--message-ids", "0123456789abcdef", "--as", "user"))
	if len(noOp["success_message_ids"].([]any)) != 1 {
		t.Fatal(noOp)
	}
	result := fixture.Run("mail", "+message-modify", "--message-ids", "0123456789abcdef", "--add-label-ids", "UNREAD", "--remove-label-ids", "UNREAD", "--as", "user")
	if result.ExitCode == 0 || !strings.Contains(result.Stderr, "validation") {
		t.Fatal(result)
	}
	result = fixture.Run("mail", "+draft-send", "--draft-id", "d1", "--as", "bot", "--yes")
	if result.ExitCode == 0 {
		t.Fatal("bot send accepted")
	}
}

func TestMailLivePrivateMailboxLifecycle(t *testing.T) {
	t.Skip("BLOCKED: no verified live harness binds current account/tenant, run-owned private drafts and self-only recipients (including CC/BCC, reply-all, calendar attendees and receipt/share destinations); legacy credentialed mail workflows are not authorized substitutes")
}

// Exercise every shortcut's public request planning surface without credentials
// or effects. Detailed uncommon input/MIME behavior is covered by domain units.
func TestMailShortcutPlans(t *testing.T) {
	cases := []struct {
		command string
		args    []string
		want    string
	}{
		{"+message", []string{"--message-id", "m1"}, "messages"},
		{"+messages", []string{"--message-ids", "m1,m2"}, "batch_get"},
		{"+message-modify", []string{"--message-ids", "0123456789abcdef", "--add-label-ids", "UNREAD"}, "batch_modify"},
		{"+message-trash", []string{"--message-ids", "0123456789abcdef"}, "batch_trash"},
		{"+thread", []string{"--thread-id", "th1"}, "threads"},
		{"+triage", []string{"--max", "2", "--query", "预算", "--format", "json"}, "messages"},
		{"+watch", nil, "subscribe"},
		{"+rule-list", nil, "rules"},
		{"+rule-get", []string{"--rule-id", "r1"}, "rules"},
		{"+rule-create", []string{"--name", "Synthetic", "--condition", "subject:contains:Alpha", "--action", "mark_read"}, "rules"},
		{"+rule-update", []string{"--rule-id", "r1", "--name", "Changed"}, "rules"},
		{"+rule-delete", []string{"--rule-id", "r1"}, "rules"},
		{"+rule-enable", []string{"--rule-id", "r1"}, "rules"},
		{"+rule-disable", []string{"--rule-id", "r1"}, "rules"},
		{"+rule-reorder", []string{"--move-rule-id", "r1", "--to-top"}, "rules"},
		{"+reply", []string{"--message-id", "m1", "--body", "<b>正文</b>", "--no-signature"}, "draft"},
		{"+reply-all", []string{"--message-id", "m1", "--body", "Reply", "--no-signature"}, "draft"},
		{"+send", []string{"--to", "fixture@example.invalid", "--subject", "Synthetic", "--body", "Body", "--no-signature"}, "draft"},
		{"+draft-create", []string{"--to", "fixture@example.invalid", "--subject", "Synthetic", "--body", "Body", "--no-signature"}, "draft"},
		{"+draft-edit", []string{"--draft-id", "d1", "--set-subject", "New"}, "draft"},
		{"+draft-send", []string{"--draft-id", "d1,d2"}, "send"},
		{"+forward", []string{"--message-id", "m1", "--to", "fixture@example.invalid", "--body", "Forward", "--no-signature"}, "draft"},
		{"+send-receipt", []string{"--message-id", "m1"}, "messages"},
		{"+decline-receipt", []string{"--message-id", "m1"}, "messages"},
		{"+signature", nil, "signature"},
		{"+share-to-chat", []string{"--message-id", "m1", "--receive-id", "ou_synthetic", "--receive-id-type", "open_id"}, "messages"},
		{"+template-create", []string{"--name", "Synthetic", "--template-content", "<b>正文</b>"}, "templates"},
		{"+template-update", []string{"--template-id", "77", "--set-subject", "New"}, "templates"},
	}
	fixture := nextfixture.NewFixture(t, nil)
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			args := append([]string{"mail", tc.command, "--as", "user", "--dry-run"}, tc.args...)
			result := fixture.Run(args...)
			if result.ExitCode != 0 || !strings.Contains(result.Stdout, tc.want) {
				t.Fatalf("%+v", result)
			}
			var decoded any
			if err := json.Unmarshal([]byte(result.Stdout), &decoded); err != nil {
				t.Fatal(err)
			}
		})
	}
	lint := data(t, fixture.Run("mail", "+lint-html", "--body", "<b>正文</b>", "--as", "user"))
	if lint["cleaned_html"] == nil {
		t.Fatal(lint)
	}
}

func TestMailTemplateAndRuleNoOpPreserveOmittedValues(t *testing.T) {
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("no-op wrote %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected write", 500)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/templates/77"):
			reply(w, map[string]any{"template": map[string]any{"template_id": "77", "name": "Template", "subject": "Original", "template_content": "<p>正文</p>", "bccs": []any{map[string]any{"mail_address": "fixture@example.invalid"}}}})
		case strings.HasSuffix(r.URL.Path, "/rules"):
			reply(w, map[string]any{"rules": []any{map[string]any{"rule_id": "r1", "name": "Original", "is_enable": true, "condition": map[string]any{"match_type": 1, "items": []any{map[string]any{"type": 6, "operator": 1, "input": "Alpha", "vendor_extra": "keep"}}}, "action": map[string]any{"items": []any{map[string]any{"type": 3}}}}}})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	tpl := data(t, fixture.Run("mail", "+template-update", "--template-id", "77", "--set-subject", "Original", "--as", "user"))
	if tpl["no_op"] != true {
		t.Fatal(tpl)
	}
	rule := data(t, fixture.Run("mail", "+rule-update", "--rule-id", "r1", "--name", "Original", "--yes", "--as", "user"))
	if rule["no_op"] != true {
		t.Fatal(rule)
	}
}

func TestMailDraftSendPartialEffects(t *testing.T) {
	var sends atomic.Int32
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sends.Add(1)
		if strings.HasSuffix(r.URL.Path, "/d1/send") {
			reply(w, map[string]any{"message_id": "m1"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic missing scope"}`)
	}))
	result := fixture.Run("mail", "+draft-send", "--draft-id", "d1,d2,d3", "--as", "user", "--yes")
	if result.ExitCode != 21 || sends.Load() != 2 || !strings.Contains(result.Stderr, `"outcome":"partial"`) {
		t.Fatalf("%+v sends %d", result, sends.Load())
	}
	if !strings.Contains(result.Stdout, `"total":3`) || !strings.Contains(result.Stdout, "aborted") {
		t.Fatal(result)
	}
}

// All frozen generated endpoints retain real generic dispatch. Fixtures contain
// synthetic typed required input derived from the frozen v1 mail catalog.
func TestMailCatalogRequests(t *testing.T) {
	raw, err := os.ReadFile("catalog_requests.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Command []string       `json:"command"`
		Method  string         `json:"method"`
		Yes     bool           `json:"yes"`
		Path    string         `json:"path"`
		Params  map[string]any `json:"params"`
		Body    map[string]any `json:"body"`
	}
	if err = json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 57 {
		t.Fatalf("catalog endpoints %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.Command, " "), func(t *testing.T) {
			var calls atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.Method || r.URL.Path != tc.Path {
					t.Errorf("got %s %s want %s %s", r.Method, r.URL.Path, tc.Method, tc.Path)
				}
				if tc.Body != nil {
					var got any
					if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
						t.Error(err)
					}
					want, _ := json.Marshal(tc.Body)
					actual, _ := json.Marshal(got)
					if string(want) != string(actual) {
						t.Errorf("body %s want %s", actual, want)
					}
				}
				reply(w, map[string]any{"fixture_marker": "保留", "nullable": nil, "identifier": "001", "large_integer": json.Number("9007199254740993")})
			}))
			params, _ := json.Marshal(tc.Params)
			args := append(append([]string{}, tc.Command...), "--params", string(params), "--as", "user")
			if tc.Yes {
				args = append(args, "--yes")
			}
			if tc.Body != nil {
				body, _ := json.Marshal(tc.Body)
				args = append(args, "--data", string(body))
			}
			result := fixture.Run(args...)
			if result.ExitCode != 0 && result.ExitCode != 22 {
				t.Fatalf("%+v", result)
			}
			if calls.Load() != 1 || !strings.Contains(result.Stdout, "保留") || !strings.Contains(result.Stdout, "9007199254740993") {
				t.Fatalf("%+v calls %d", result, calls.Load())
			}
		})
	}
}

func TestMailTriageBoundAndContinuation(t *testing.T) {
	var lists atomic.Int32
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/messages") {
			lists.Add(1)
			if r.URL.Query().Get("page_size") != "2" || r.URL.Query().Get("folder_id") != "INBOX" {
				t.Errorf("query %s", r.URL.RawQuery)
			}
			reply(w, map[string]any{"items": []any{"m1", "m2"}, "has_more": true, "page_token": "continue-synthetic"})
			return
		}
		if r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/messages/batch_get") {
			reply(w, map[string]any{"messages": []any{map[string]any{"message_id": "m1", "subject": "One"}, map[string]any{"message_id": "m2", "subject": "Two"}}})
			return
		}
		t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
	}))
	response := fixture.Run("mail", "+triage", "--max", "2", "--mailbox", "fixture@example.invalid", "--as", "bot", "--format", "json")
	if response.ExitCode != 0 {
		t.Fatal(response)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(response.Stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result["has_more"] != true || result["page_token"] == "" || result["mailbox_id"] != "fixture@example.invalid" || result["count"] != float64(2) || lists.Load() != 1 {
		t.Fatal(result)
	}
}

func TestMailWatchLifecycleReceiptsAndCleanupOnSetupFailure(t *testing.T) {
	var subscriptions, unsubscriptions atomic.Int32
	fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/event/subscribe"):
			subscriptions.Add(1)
			reply(w, map[string]any{})
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/event/unsubscribe"):
			unsubscriptions.Add(1)
			reply(w, map[string]any{})
		case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/profile"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic profile scope rejection"}`)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	result := fixture.Run("mail", "+watch", "--as", "user")
	if result.ExitCode == 0 || subscriptions.Load() != 1 || unsubscriptions.Load() != 1 || !strings.Contains(result.Stderr, `"applied":2`) {
		t.Fatalf("%+v subscribe %d unsubscribe %d", result, subscriptions.Load(), unsubscriptions.Load())
	}
}
