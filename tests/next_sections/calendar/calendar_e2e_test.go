// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const calendarPath = "/open-apis/calendar/v4/calendars/cal_fixture"
const eventPath = calendarPath + "/events/evt_fixture_0"
const start = "2026-09-07T10:00:00Z"
const end = "2026-09-07T11:00:00Z"

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func success(t *testing.T, result nextfixture.Result) map[string]any {
	t.Helper()
	require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Stdout), &envelope), result.Stdout)
	require.NotEqual(t, false, envelope["ok"], result.Stdout)
	return envelope
}
func event() map[string]any {
	return map[string]any{"event_id": "evt_fixture_0", "summary": "日程 👋", "description_rich": "**exact**", "start_time": map[string]string{"timestamp": "1788775200", "timezone": "UTC"}, "end_time": map[string]string{"timestamp": "1788778800", "timezone": "UTC"}, "attachments": []any{map[string]string{"file_token": "file_fixture", "name": "附件", "file_size": "9007199254740993"}}, "event_check_in": map[string]any{"enable_check_in": true, "need_notify_attendees": false}}
}

func TestCalendarEventLifecycleAndNoOp(t *testing.T) {
	var mu sync.Mutex
	var stored map[string]any
	writes := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == "POST" && r.URL.Path == calendarPath+"/events":
			writes++
			require.NoError(t, json.NewDecoder(r.Body).Decode(&stored))
			stored["event_id"] = "evt_fixture_0"
			reply(w, map[string]any{"event": stored})
		case r.Method == "GET" && r.URL.Path == eventPath:
			reply(w, map[string]any{"event": stored})
		case r.Method == "PATCH" && r.URL.Path == eventPath:
			writes++
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, false, body["need_notification"])
			for k, v := range body {
				if k != "need_notification" {
					stored[k] = v
				}
			}
			reply(w, map[string]any{"event": stored})
		case r.Method == "DELETE" && r.URL.Path == eventPath:
			writes++
			require.Equal(t, "false", r.URL.Query().Get("need_notification"))
			stored = nil
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	run := func(args ...string) nextfixture.Result {
		return f.Run(append([]string{"calendar"}, append(args, "--as", "user")...)...)
	}
	success(t, run("+create", "--calendar-id", "cal_fixture", "--summary", "日程 👋", "--description", "**exact**", "--start", start, "--end", end))
	require.Equal(t, 1, writes)
	got := success(t, run("+get", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0"))
	require.Contains(t, got["data"], "summary")
	unchanged := success(t, run("+update", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--summary", "日程 👋", "--notify=false"))
	require.Equal(t, true, unchanged["data"].(map[string]any)["no_op"])
	require.Equal(t, 1, writes)
	success(t, run("+update", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--description", "", "--summary", "Changed", "--notify=false"))
	require.Equal(t, "", stored["description_rich"])
	require.Equal(t, "Changed", stored["summary"])
	require.Equal(t, 2, writes)
	success(t, run("+delete", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--notify=false", "--yes"))
	require.Nil(t, stored)
	require.Equal(t, 3, writes)
}

func TestCalendarReadsAndPlanning(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		args               []string
		data               any
		want               string
	}{
		{"get", "GET", eventPath, []string{"+get", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0"}, map[string]any{"event": event()}, "9007199254740993"},
		{"agenda", "GET", calendarPath + "/events/instance_view", []string{"+agenda", "--calendar-id", "cal_fixture", "--start", start, "--end", end}, map[string]any{"items": []any{event()}, "has_more": false}, "日程"},
		{"search", "POST", calendarPath + "/events/search_event", []string{"+search-event", "--calendar-id", "cal_fixture", "--query", "日程", "--page-size", "2", "--page-token", "previous"}, map[string]any{"items": []any{map[string]any{"meta_data": event()}}, "has_more": true, "page_token": "next_fixture"}, "next_fixture"},
		{"attendees", "GET", eventPath + "/attendees", []string{"+list-attendees", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--type", "third_party", "--page-size", "2", "--page-token", "previous"}, map[string]any{"items": []any{map[string]any{"type": "third_party", "third_party_email": "synthetic@example.invalid", "attendee_id": "att_fixture"}}, "has_more": true, "page_token": "next_fixture"}, "synthetic@example.invalid"},
		{"meeting", "POST", calendarPath + "/events/mget_instance_relation_info", []string{"+meeting", "--calendar-id", "cal_fixture", "--event-ids", "evt_fixture_0"}, map[string]any{"instance_relation_infos": []any{map[string]any{"meeting_instance_ids": []string{"meeting_fixture"}, "meeting_notes": []string{"note_fixture"}}}}, "meeting_fixture"},
		{"suggestion", "POST", "/open-apis/calendar/v4/freebusy/suggestion", []string{"+suggestion", "--start", start, "--end", end, "--duration-minutes", "30", "--timezone", "UTC", "--attendee-ids", "ou_fixture"}, map[string]any{"suggestions": []any{map[string]string{"event_start_time": start, "event_end_time": end, "recommend_reason": "fixture"}}}, "fixture"},
		{"rooms", "POST", "/open-apis/calendar/v4/freebusy/room_find", []string{"+room-find", "--slot", start + "~" + end, "--city", "Synthetic", "--building", "A", "--floor", "F2", "--room-name", "A,B", "--min-capacity", "2", "--max-capacity", "6", "--timezone", "UTC"}, map[string]any{"available_rooms": []any{map[string]any{"room_id": "omm_fixture", "room_name": "Fixture room", "capacity": 4}}}, "omm_fixture"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count++
				require.Equal(t, tc.method, r.Method)
				require.Equal(t, tc.path, r.URL.Path)
				if tc.name == "search" || tc.name == "attendees" {
					require.Equal(t, "previous", r.URL.Query().Get("page_token"))
				}
				reply(w, tc.data)
			}))
			args := append([]string{"calendar"}, tc.args...)
			args = append(args, "--as", "user")
			result := f.Run(args...)
			success(t, result)
			require.Contains(t, result.Stdout, tc.want)
			require.Equal(t, 1, count)
			require.NotContains(t, result.Stderr, `"receipts"`)
		})
	}
	for _, view := range []string{"busy", "raw_busy", "free", "common_free"} {
		t.Run(view, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/open-apis/calendar/v4/freebusy/batch", r.URL.Path)
				require.Equal(t, "POST", r.Method)
				reply(w, map[string]any{"user_freebusy_list": []any{}})
			}))
			result := f.Run("calendar", "+freebusy", "--as", "user", "--user-id", "ou_fixture", "--start", start, "--end", end, "--type", view, "--min-duration", "30m")
			success(t, result)
			require.NotContains(t, result.Stderr, `"receipts"`)
		})
	}
}

func TestCalendarArtifactPreservesFullEvent(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, eventPath, r.URL.Path)
		reply(w, map[string]any{"event": event()})
	}))
	path := filepath.Join(f.WorkDir, "event.json")
	success(t, f.Run("calendar", "+get", "--as", "user", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--result-fields", "summary", "--save-result", path))
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), "9007199254740993")
	require.Contains(t, string(raw), "file_fixture")
	require.Contains(t, string(raw), "event_check_in")
	var artifact map[string]any
	require.NoError(t, json.Unmarshal(raw, &artifact))
	require.Equal(t, "feishu-cli-next/result/v1", artifact["schema"])
	binding := artifact["binding"].(map[string]any)
	require.Equal(t, true, binding["user_verified"])
	require.Equal(t, "foundation-fixture-tenant", binding["tenant_key"])
}

func TestCalendarActionsAndAttendees(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		path string
		want map[string]any
	}{
		{"join", []string{"+join-event", "--share-token", "token_fixture"}, "/open-apis/calendar/v4/calendars/join_event", map[string]any{"share_token": "token_fixture"}},
		{"rsvp", []string{"+rsvp", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--rsvp-status", "tentative"}, eventPath + "/reply", map[string]any{"rsvp_status": "tentative"}},
		{"transfer", []string{"+transfer", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--to-user-id", "ou_fixture", "--yes"}, eventPath + "/transfer", map[string]any{"to_user_id": "ou_fixture"}},
		{"attendee add", []string{"+update", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--add-attendee-ids", "ou_fixture,oc_fixture,omm_fixture", "--skip-room-check", "--notify=false"}, eventPath + "/attendees", map[string]any{"need_notification": false}},
		{"attendee remove", []string{"+update", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--remove-attendee-ids", "ou_fixture,oc_fixture,omm_fixture", "--notify=false"}, eventPath + "/attendees/batch_delete", map[string]any{"need_notification": false}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			count := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					require.Equal(t, eventPath, r.URL.Path)
					reply(w, map[string]any{"event": event()})
					return
				}
				count++
				require.Equal(t, "POST", r.Method)
				require.Equal(t, tc.path, r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				for k, v := range tc.want {
					require.Equal(t, v, body[k], "body=%v", body)
				}
				reply(w, map[string]any{})
			}))
			args := append([]string{"calendar"}, tc.args...)
			args = append(args, "--as", "user")
			result := f.Run(args...)
			if tc.name == "attendee remove" {
				require.Equal(t, 22, result.ExitCode)
				require.Contains(t, result.Stderr, `"outcome":"unknown"`)
			} else {
				success(t, result)
			}
			require.Equal(t, 1, count)
		})
	}
}

func TestCalendarValidationAndScopeFailures(t *testing.T) {
	for _, args := range [][]string{
		{"+update", "--event-id", "evt_fixture_0", "--start", start},
		{"+update", "--event-id", "evt_fixture_0", "--add-attendee-ids", "ou_fixture", "--remove-attendee-ids", "ou_fixture"},
		{"+get", "--event-id", "ambiguous"},
		{"+create", "--start", end, "--end", start},
		{"+room-find", "--slot", start + "~" + end, "--min-capacity", "10", "--max-capacity", "1"},
		{"+suggestion", "--duration-minutes", "1441"},
		{"+list-attendees", "--event-id", "evt_fixture_0", "--type", "unknown"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			result := f.Run(append(append([]string{"calendar"}, args...), "--as", "user")...)
			require.NotEqual(t, 0, result.ExitCode)
			require.Equal(t, "validation", errorType(t, result.Stderr))
			require.Empty(t, result.Stdout)
		})
	}
	t.Run("permission", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(403)
			_, _ = io.WriteString(w, `{"code":99991672,"msg":"scope forbidden"}`)
		}))
		result := f.Run("calendar", "+get", "--as", "user", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0")
		require.NotEqual(t, 0, result.ExitCode)
		require.Contains(t, result.Stderr, `"error"`)
		require.Empty(t, result.Stdout)
	})
}

func TestCalendarPartialAndUnknownMutation(t *testing.T) {
	for _, mode := range []string{"partial", "unknown"} {
		t.Run(mode, func(t *testing.T) {
			writes := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					reply(w, map[string]any{"event": event()})
					return
				}
				writes++
				if r.Method == "PATCH" {
					reply(w, map[string]any{"event": event()})
					return
				}
				if mode == "unknown" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"code":`)
					return
				}
				w.WriteHeader(403)
				_, _ = io.WriteString(w, `{"code":99991672,"msg":"scope forbidden"}`)
			}))
			result := f.Run("calendar", "+update", "--as", "user", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--summary", "changed", "--add-attendee-ids", "ou_fixture", "--notify=false")
			if mode == "partial" {
				require.Equal(t, 21, result.ExitCode)
				require.Contains(t, result.Stderr, `"outcome":"partial"`)
			} else {
				require.Equal(t, 22, result.ExitCode)
				require.Contains(t, result.Stderr, `"outcome":"unknown"`)
			}
			require.Equal(t, 2, writes, "unknown/partial writes must never replay")
			receipts := success(t, f.Run("receipts", "list"))
			require.NotEmpty(t, receipts)
		})
	}
}

func TestCalendarLivePrivateLifecycle(t *testing.T) {
	t.Skip("BLOCKED: no verified current-account/tenant/private calendar creation fixture; live invitations and notifications also require verified self or unambiguous Ying Wu. Legacy personal-state tests are excluded.")
}

func errorType(t *testing.T, raw string) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))
	return envelope.Error.Type
}
