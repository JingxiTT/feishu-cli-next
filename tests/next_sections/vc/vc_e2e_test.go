// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package vc_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const meetingID = "7651377260537433044"

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func resultData(t *testing.T, result nextfixture.Result) map[string]any {
	t.Helper()
	if result.ExitCode != 0 {
		t.Fatalf("CLI: %+v", result)
	}
	var out struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		t.Fatalf("%v: %+v", err, result)
	}
	return out.Data
}
func TestFrozenVCDiscovery(t *testing.T) {
	file, err := os.Open("../../../docs/reference/registrations.json.gz")
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
		if command != "vc" && !strings.HasPrefix(command, "vc ") {
			continue
		}
		count++
		t.Run(command, func(t *testing.T) {
			r := fixture.Run(append(strings.Fields(command), "--help")...)
			if r.ExitCode != 0 {
				t.Fatal(r)
			}
			for name, flag := range contract.Flags {
				// Authsidecar deliberately conceals profile selection; normal-build inventory owns it.
				if name != "profile" && !flag.Hidden && !strings.Contains(r.Stdout, "--"+name) {
					t.Errorf("missing --%s", name)
				}
			}
		})
	}
	if count != 16 {
		t.Fatalf("frozen count %d", count)
	}
	schema := fixture.Run("schema", "vc", "meeting", "get")
	if schema.ExitCode != 0 {
		t.Fatalf("schema %+v", schema)
	}
	for _, field := range []string{"query_mode", "with_participants", "with_meeting_ability", "user_id_type", "participants", "meeting_no"} {
		if !strings.Contains(schema.Stdout, field) {
			t.Errorf("missing schema field %s", field)
		}
	}
}

func TestVCMutatingWorkflows(t *testing.T) {
	cases := []struct {
		name, path string
		args       []string
		body       map[string]any
	}{
		{"join", "join", []string{"+meeting-join", "--meeting-number", "012345678", "--password", "secret", "--call-id", "correlation"}, map[string]any{"join_type": float64(1), "join_identify": map[string]any{"meeting_no": "012345678"}, "password": "secret", "call_id": "correlation"}},
		{"start", "join", []string{"+meeting-join", "--meeting-number", "012345678", "--action", "start"}, map[string]any{"join_type": float64(1), "join_identify": map[string]any{"meeting_no": "012345678"}, "action": float64(2)}},
		{"invite selected", "invite", []string{"+meeting-invite", "--meeting-id", meetingID, "--type", "SELECTED", "--open-ids", "ou_fixture,ou_fixture"}, map[string]any{"meeting_id": meetingID, "invite_type": float64(2), "invitees": []any{map[string]any{"id": "ou_fixture", "user_type": float64(1)}}}},
		{"invite suggested", "invite", []string{"+meeting-invite", "--meeting-id", meetingID, "--type", "ALL_SUGGESTED"}, map[string]any{"meeting_id": meetingID, "invite_type": float64(1)}},
		{"end", "end", []string{"+meeting-end", "--meeting-id", meetingID, "--yes"}, map[string]any{"meeting_id": meetingID}},
		{"leave", "leave", []string{"+meeting-leave", "--meeting-id", meetingID}, map[string]any{"meeting_id": meetingID}},
		{"text", "message", []string{"+meeting-message-send", "--meeting-id", meetingID, "--text", "  Unicode 正文\nline 2  ", "--uuid", "same-key"}, map[string]any{"meeting_id": meetingID, "msg_type": "text", "content": "Unicode 正文\nline 2", "uuid": "same-key"}},
		{"reaction", "message", []string{"+meeting-message-send", "--meeting-id", meetingID, "--emoji-type", "VC_NoSound"}, map[string]any{"meeting_id": meetingID, "msg_type": "reaction", "content": "VC_NoSound"}},
		{"countdown set", "countdown", []string{"+meeting-countdown", "--meeting-id", meetingID, "--action", "set", "--duration", "10", "--reminder-before-end", "2", "--need-play-audio-at-end=false"}, map[string]any{"meeting_id": meetingID, "action": "set", "duration": float64(10), "reminder_before_end": float64(2), "need_play_audio_at_end": false}},
		{"countdown prolong", "countdown", []string{"+meeting-countdown", "--meeting-id", meetingID, "--action", "prolong", "--duration", "3"}, map[string]any{"meeting_id": meetingID, "action": "prolong", "duration": float64(3)}},
		{"countdown end", "countdown", []string{"+meeting-countdown", "--meeting-id", meetingID, "--action", "end_in_advance"}, map[string]any{"meeting_id": meetingID, "action": "end_in_advance"}},
		{"countdown close", "countdown", []string{"+meeting-countdown", "--meeting-id", meetingID, "--action", "close_window"}, map[string]any{"meeting_id": meetingID, "action": "close_window"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			fixture := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "POST" || r.URL.Path != "/open-apis/vc/v1/bots/"+tc.path {
					t.Errorf("%s %s", r.Method, r.URL.Path)
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if !reflect.DeepEqual(body, tc.body) {
					t.Errorf("body %#v want %#v", body, tc.body)
				}
				if tc.path == "invite" && r.URL.Query().Get("user_id_type") != "open_id" {
					t.Error("invite identity type lost")
				}
				if tc.path == "invite" {
					reply(w, map[string]any{"failed_count": 0, "invited_count": 1, "has_more": false, "invite_results": []any{map[string]any{"id": "ou_fixture", "status": 1}}})
					return
				}
				reply(w, map[string]any{"meeting": map[string]any{"id": meetingID}, "request_id": "accepted"})
			}))
			args := append([]string{"vc"}, tc.args...)
			args = append(args, "--as", "bot")
			r := fixture.Run(args...)
			resultData(t, r)
			if calls.Load() != 1 || !strings.Contains(r.Stderr, "confirmed_applied") {
				t.Fatalf("calls %d %+v", calls.Load(), r)
			}
		})
	}
}

func TestVCMutationUnknownAndRejectedNeverReplay(t *testing.T) {
	for _, tc := range []struct {
		name, body, outcome string
		exit                int
	}{{"scope", `{"code":99991679,"msg":"synthetic scope rejection"}`, "confirmed_unapplied", 20}, {"lost", "", "unknown", 22}, {"malformed", "{", "unknown", 22}, {"async", `{"code":0,"data":{}}`, "unknown", 22}} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tc.name == "lost" {
					c, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error(err)
						return
					}
					_ = c.Close()
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if tc.name == "async" {
					w.WriteHeader(202)
				}
				_, _ = io.WriteString(w, tc.body)
			}))
			r := f.Run("vc", "+meeting-message-send", "--meeting-id", meetingID, "--text", "fixture", "--uuid", "stable", "--as", "user")
			if r.ExitCode != tc.exit || calls.Load() != 1 || !strings.Contains(r.Stderr, tc.outcome) {
				t.Fatalf("calls %d %+v", calls.Load(), r)
			}
			listed := f.Run("receipts", "list")
			var receipts struct {
				Data []struct {
					Outcome string `json:"outcome"`
					Binding struct {
						UserVerified   bool `json:"user_verified"`
						TenantVerified bool `json:"tenant_verified"`
					} `json:"binding"`
				} `json:"data"`
			}
			if err := json.Unmarshal([]byte(listed.Stdout), &receipts); err != nil || listed.ExitCode != 0 || len(receipts.Data) != 1 {
				t.Fatalf("receipt list %+v: %v", listed, err)
			}
			if receipts.Data[0].Outcome != tc.outcome || !receipts.Data[0].Binding.UserVerified || !receipts.Data[0].Binding.TenantVerified {
				t.Fatalf("receipt binding %#v", receipts)
			}
		})
	}
}

func TestVCValidationPreventsBusinessRequests(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	cases := [][]string{
		{"+search", "--query", "q", "--page-size", "31"}, {"+search", "--start", "2026-09-08", "--end", "2026-09-07"}, {"+search"},
		{"+detail", "--meeting-ids", strings.Repeat("1,", 50) + "1"}, {"+recording", "--meeting-ids", "1", "--calendar-event-ids", "event"}, {"+notes", "--minute-tokens", "../token"},
		{"+meeting-invite", "--meeting-id", meetingID, "--type", "SELECTED", "--open-ids", "chat_other"}, {"+meeting-invite", "--meeting-id", meetingID, "--type", "ALL_SUGGESTED", "--open-ids", "ou_fixture"},
		{"+meeting-message-send", "--meeting-id", meetingID, "--text", "x", "--emoji-type", "LOVE"}, {"+meeting-message-send", "--meeting-id", meetingID, "--text", strings.Repeat("x", 48*1024+1)},
		{"+meeting-countdown", "--meeting-id", meetingID, "--action", "set", "--duration", "2", "--reminder-before-end", "2"}, {"+meeting-countdown", "--meeting-id", meetingID, "--action", "close_window", "--duration", "1"},
		{"+meeting-events", "--meeting-id", "123456789"}, {"+meeting-events", "--meeting-id", meetingID, "--page-size", "invalid"},
		{"+meeting-list-active", "--as", "bot"}, {"+meeting-join", "--meeting-number", "012345678", "--action", "start", "--as", "user"},
		{"+meeting-invite", "--meeting-id", meetingID, "--type", "ALL_SUGGESTED", "--as", "user"}, {"+meeting-end", "--meeting-id", meetingID, "--as", "user", "--yes"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[:1], " ")+strings.Join(args[1:min(3, len(args))], " "), func(t *testing.T) {
			r := f.Run(append([]string{"vc"}, args...)...)
			if r.ExitCode == 0 || r.Stdout != "" {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestVCSearchAndGenericMeetingArtifacts(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		switch r.URL.Path {
		case "/open-apis/vc/v1/meetings/search":
			if r.Method != "POST" || r.URL.Query().Get("page_token") != "next" || r.URL.Query().Get("page_size") != "30" {
				t.Errorf("%s %s", r.Method, r.URL)
			}
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			filter, _ := body["meeting_filter"].(map[string]any)
			if filter["start_time"] == nil || !reflect.DeepEqual(filter["participant_ids"], []any{"ou_fixture"}) || filter["organizer_ids"] == nil || filter["open_room_ids"] == nil {
				t.Errorf("filter %#v", filter)
			}
			reply(w, map[string]any{"items": []any{map[string]any{"id": meetingID, "display_info": "Unicode 中文", "meta_data": map[string]any{"description": "detail", "avatar": "reference-removes-this"}}}, "has_more": true, "page_token": "continuation", "notice": "backend notice"})
		case "/open-apis/vc/v1/meetings/" + meetingID:
			if r.Method != "GET" || r.URL.Query().Get("user_id_type") != "union_id" || r.URL.Query().Get("with_participants") != "true" || r.URL.Query().Get("with_meeting_ability") != "true" || r.URL.Query().Get("query_mode") != "1" {
				t.Errorf("query %s", r.URL)
			}
			reply(w, map[string]any{"meeting": map[string]any{"id": meetingID, "topic": "Unicode 中文", "participants": []any{map[string]any{"id": "on_fixture", "is_host": false, "devices": []any{}, "user_type": 1}}, "ability": map[string]any{"screen_share": false}, "note_id": "note1"}})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
			reply(w, map[string]any{})
		}
	}))
	search := resultData(t, f.Run("vc", "+search", "--query", "review", "--participant-ids", "ou_fixture,ou_fixture", "--organizer-ids", "ou_fixture", "--room-ids", "room1", "--start", "2026-09-07", "--end", "2026-09-08", "--page-token", "next", "--page-size", "30", "--as", "user", "--save-result", "search.json"))
	if search["has_more"] != true || search["page_token"] != "continuation" {
		t.Fatal(search)
	}
	r := f.Run("vc", "meeting", "get", "--meeting-id", meetingID, "--with-participants", "--with-meeting-ability", "--user-id-type", "union_id", "--query-mode", "1", "--as", "bot", "--result-fields", "meeting.topic", "--save-result", "meeting.json")
	resultData(t, r)
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "meeting.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"devices":[]`) || !strings.Contains(string(raw), `"screen_share":false`) || !strings.Contains(string(raw), "on_fixture") {
		t.Fatalf("lost generic fields %s", raw)
	}
	if calls.Load() != 2 || strings.Contains(r.Stderr, `"mutation"`) {
		t.Fatalf("calls %d %+v", calls.Load(), r)
	}
}

func TestVCActiveAndEventsPagination(t *testing.T) {
	var pages atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/vc/v1/bots/user_active_meeting":
			if r.URL.Query().Get("user_id") != "ou_fixture" {
				t.Error("bot lookup target lost")
			}
			reply(w, map[string]any{"meetings": []any{map[string]any{"id": meetingID, "topic": "fixture"}}})
		case "/open-apis/vc/v1/bots/events":
			n := pages.Add(1)
			if r.URL.Query().Get("meeting_id") != meetingID {
				t.Errorf("%s", r.URL)
			}
			if n == 2 && r.URL.Query().Get("page_token") != "next" {
				t.Error("missing continuation")
			}
			reply(w, map[string]any{"meeting": map[string]any{"id": meetingID, "topic": "fixture", "start_time": "1700000000"}, "events": []any{map[string]any{"event_id": "e" + string(rune('0'+n)), "event_type": "future_event", "event_time": "1700000001", "payload": map[string]any{"custom": "Unicode 正文", "empty": ""}}}, "has_more": n == 1, "page_token": map[bool]string{true: "next", false: ""}[n == 1]})
		default:
			t.Errorf("%s", r.URL)
			reply(w, map[string]any{})
		}
	}))
	resultData(t, f.Run("vc", "+meeting-list-active", "--user-id", "ou_fixture", "--as", "bot"))
	events := resultData(t, f.Run("vc", "+meeting-events", "--meeting-id", meetingID, "--page-size", "20", "--page-all", "--start", "1700000000", "--end", "1700000100", "--as", "user"))
	if pages.Load() != 2 || events["has_more"] != false || len(events["events"].([]any)) != 2 {
		t.Fatalf("%#v pages %d", events, pages.Load())
	}
	if !strings.Contains(events["identity"].(map[string]any)["id"].(string), "fixture") {
		t.Fatal(events)
	}
}

func TestVCDetailRecordingAndNotesResolution(t *testing.T) {
	var recordings atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/calendars/primary"):
			reply(w, map[string]any{"calendars": []any{map[string]any{"calendar": map[string]any{"calendar_id": "private"}}}})
		case strings.HasSuffix(r.URL.Path, "/mget_instance_relation_info"):
			reply(w, map[string]any{"instance_relation_infos": []any{map[string]any{"meeting_instance_ids": []any{meetingID}, "meeting_notes": []any{"doc1"}}}})
		case strings.HasSuffix(r.URL.Path, "/recording"):
			recordings.Add(1)
			reply(w, map[string]any{"recording": map[string]any{"url": "https://example.invalid/minutes/tok1", "duration": "00123"}})
		case strings.Contains(r.URL.Path, "/meetings/"):
			reply(w, map[string]any{"meeting": map[string]any{"id": meetingID, "topic": "fixture", "start_time": "1700000000", "end_time": "1700000060", "note_id": "note1"}})
		case strings.HasSuffix(r.URL.Path, "/notes/note1"):
			reply(w, map[string]any{"note": map[string]any{"note_display_type": 2, "creator_id": "ou_fixture", "artifacts": []any{map[string]any{"artifact_type": 1, "doc_token": "doc1"}, map[string]any{"artifact_type": 2, "doc_token": "verbatim1"}}, "references": []any{map[string]any{"doc_token": "shared1"}}}})
		case strings.HasSuffix(r.URL.Path, "/minutes/tok1/artifacts"):
			reply(w, map[string]any{"summary": "摘要", "transcript": "line 1\nUnicode 中文\n", "minute_todos": []any{map[string]any{"text": "todo"}}, "minute_chapters": []any{map[string]any{"title": "chapter"}}, "keywords": []any{"关键词"}})
		case strings.HasSuffix(r.URL.Path, "/minutes/tok1"):
			reply(w, map[string]any{"minute": map[string]any{"title": "private fixture", "note_id": "note1"}})
		default:
			t.Errorf("%s %s", r.Method, r.URL)
			reply(w, map[string]any{})
		}
	}))
	for _, args := range [][]string{{"+detail", "--meeting-ids", meetingID}, {"+recording", "--meeting-ids", meetingID}, {"+recording", "--calendar-event-ids", "event1"}, {"+notes", "--meeting-ids", meetingID}, {"+notes", "--calendar-event-ids", "event1"}, {"+notes", "--minute-tokens", "tok1", "--output-dir", "artifacts"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			r := f.Run(append(append([]string{"vc"}, args...), "--as", "user")...)
			d := resultData(t, r)
			if len(d) == 0 || !strings.Contains(stringJSON(t, d), "tok1") {
				t.Fatalf("missing routed minute token: %+v", r)
			}
			if args[0] == "+notes" {
				for _, value := range []string{"note1", "doc1", "verbatim1", "shared1", "unified"} {
					if !strings.Contains(stringJSON(t, d), value) {
						t.Fatalf("missing note field %s: %+v", value, r)
					}
				}
			}
			if args[0] == "+recording" && !strings.Contains(stringJSON(t, d), "00123") {
				t.Fatalf("duration changed: %+v", r)
			}
			if strings.Contains(r.Stderr, `"mutation"`) {
				t.Fatalf("read journaled %+v", r)
			}
		})
	}
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "artifacts", "artifact-private fixture-tok1", "transcript.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "line 1\nUnicode 中文\n" {
		t.Fatalf("%q", raw)
	}
	if recordings.Load() != 5 {
		t.Fatalf("recordings %d", recordings.Load())
	}
	transcript := filepath.Join(f.WorkDir, "artifacts", "artifact-private fixture-tok1", "transcript.txt")
	if err := os.WriteFile(transcript, []byte("local existing transcript"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"vc", "+notes", "--minute-tokens", "tok1", "--output-dir", "artifacts", "--as", "user"}
	resultData(t, f.Run(args...))
	preserved, err := os.ReadFile(transcript)
	if err != nil || string(preserved) != "local existing transcript" {
		t.Fatalf("existing transcript overwritten: %q %v", preserved, err)
	}
	resultData(t, f.Run(append(args, "--overwrite")...))
	overwritten, err := os.ReadFile(transcript)
	if err != nil || !bytes.Equal(overwritten, raw) {
		t.Fatalf("overwrite %q %v", overwritten, err)
	}
}

func TestVCScreenshotBytesAndExistingOutput(t *testing.T) {
	var jpegBytes bytes.Buffer
	if err := jpeg.Encode(&jpegBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/open-apis/vc/v1/bots/screenshot" {
			t.Errorf("%s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegBytes.Bytes())
	}))
	args := []string{"vc", "+meeting-screenshot", "--meeting-id", meetingID, "--output", "capture.jpg", "--as", "user"}
	r := f.Run(args...)
	resultData(t, r)
	raw, err := os.ReadFile(filepath.Join(f.WorkDir, "capture.jpg"))
	if err != nil || !bytes.Equal(raw, jpegBytes.Bytes()) {
		t.Fatalf("image %v", err)
	}
	r = f.Run(args...)
	if r.ExitCode == 0 || calls.Load() != 1 {
		t.Fatalf("existing output %+v calls %d", r, calls.Load())
	}
	resultData(t, f.Run(append(args, "--overwrite")...))
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestVCLivePrivateLifecycle(t *testing.T) {
	t.Skip("BLOCKED: no live harness verifies current account and tenant, private Agent Calendar creation visibility, app-bot host ownership, self-only or unambiguously Ying Wu-only participants/eligible invitees, and run-owned cleanup; do not use unrelated existing meetings")
}

func TestVCReadPartialAndInProgress(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if strings.HasSuffix(r.URL.Path, "/recording") {
			if strings.Contains(r.URL.Path, "/good/") {
				reply(w, map[string]any{"recording": map[string]any{"url": "https://example.invalid/minutes/tok1"}})
			} else {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":121004,"msg":"no recording"}`)
			}
			return
		}
		reply(w, map[string]any{"meeting": map[string]any{"topic": "still running", "start_time": "1700000000", "end_time": "0"}})
	}))
	d := resultData(t, f.Run("vc", "+detail", "--meeting-ids", meetingID, "--as", "user"))
	if calls.Load() != 1 || !strings.Contains(stringJSON(t, d), "still in progress") {
		t.Fatalf("%#v calls %d", d, calls.Load())
	}
	r := f.Run("vc", "+recording", "--meeting-ids", "good,bad", "--as", "user", "--result-fields", "minute_token")
	d = resultData(t, r)
	if !strings.Contains(stringJSON(t, d), "error") || !strings.Contains(stringJSON(t, d), "tok1") {
		t.Fatal(d)
	}
	r = f.Run("vc", "+recording", "--meeting-ids", "bad", "--as", "user")
	if r.ExitCode == 0 || !strings.Contains(r.Stdout, `"ok":false`) || !strings.Contains(r.Stdout, "bad") {
		t.Fatalf("all failed %+v", r)
	}
}
func stringJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestVCInvitePartialOutcome(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		reply(w, map[string]any{"failed_count": 1, "invited_count": 1, "has_more": true, "invite_results": []any{map[string]any{"id": "ou_one", "status": 1}, map[string]any{"id": "ou_two", "status": 2}}})
	}))
	r := f.Run("vc", "+meeting-invite", "--meeting-id", meetingID, "--type", "SELECTED", "--open-ids", "ou_one,ou_two", "--as", "bot", "--result-fields", "invited_count")
	if r.ExitCode != 22 || calls.Load() != 1 || !strings.Contains(r.Stderr, "unknown") || !strings.Contains(r.Stdout, `"ok":false`) || !strings.Contains(r.Stdout, "ou_two") || !strings.Contains(r.Stdout, "notice") {
		t.Fatalf("partial invite calls %d %+v", calls.Load(), r)
	}
}

func TestVCMeetingEventsBoundedPagination(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page_size") != "100" {
			t.Errorf("page size %s", r.URL)
		}
		reply(w, map[string]any{"events": []any{}, "has_more": true, "page_token": "repeat"})
	}))
	d := resultData(t, f.Run("vc", "+meeting-events", "--meeting-id", meetingID, "--page-all", "--as", "user"))
	if calls.Load() != 200 || d["has_more"] != true || d["page_token"] != "repeat" {
		t.Fatalf("bounds %#v calls %d", d, calls.Load())
	}
}

func TestVCReadIdentityScopeFailure(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/open-apis/vc/v1/bots/user_active_meeting" || r.URL.Query().Get("user_id") != "" {
			t.Errorf("current user request %s", r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":99991679,"msg":"synthetic scope failure"}`)
	}))
	r := f.Run("vc", "+meeting-list-active", "--as", "user")
	if r.ExitCode != 3 || r.Stdout != "" || calls.Load() != 1 || !strings.Contains(r.Stderr, "vc:meeting.meetingevent:read") || !strings.Contains(r.Stderr, `"missing_scope"`) || strings.Contains(r.Stderr, `"mutation"`) {
		t.Fatalf("scope %+v calls %d", r, calls.Load())
	}
}
