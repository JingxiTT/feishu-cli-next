// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

// Frozen v4 catalog methods stay on the generic API surface. These concrete
// request fixtures exercise every method and preserve optional query/body types.
func TestCalendarGeneratedMethods(t *testing.T) {
	for _, tc := range []struct{ resource, method, verb, path, params, query, body string }{
		{"calendars", "create", "POST", "/open-apis/calendar/v4/calendars", `{}`, `{}`, `{"summary":"Fixture","description":"","permissions":"private","color":0,"summary_alias":"别名"}`},
		{"calendars", "delete", "DELETE", "/open-apis/calendar/v4/calendars/cal_fixture", `{"calendar_id":"cal_fixture"}`, `{}`, `{}`},
		{"calendars", "get", "GET", "/open-apis/calendar/v4/calendars/cal_fixture", `{"calendar_id":"cal_fixture"}`, `{}`, `{}`},
		{"calendars", "list", "GET", "/open-apis/calendar/v4/calendars", `{"page_size":2,"page_token":"next_fixture","sync_token":"sync_fixture"}`, `{"page_size":"2","page_token":"next_fixture","sync_token":"sync_fixture"}`, `{}`},
		{"calendars", "patch", "PATCH", "/open-apis/calendar/v4/calendars/cal_fixture", `{"calendar_id":"cal_fixture"}`, `{}`, `{"summary":"Fixture","description":"","permissions":"private","color":0,"summary_alias":"别名"}`},
		{"calendars", "primary", "POST", "/open-apis/calendar/v4/calendars/primary", `{"user_id_type":"open_id"}`, `{"user_id_type":"open_id"}`, `{}`},
		{"calendars", "search", "POST", "/open-apis/calendar/v4/calendars/search", `{"page_token":"next_fixture","page_size":2}`, `{"page_token":"next_fixture","page_size":"2"}`, `{"query":"日程"}`},
		{"event.attendees", "batch_delete", "POST", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0/attendees/batch_delete", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","user_id_type":"open_id"}`, `{"user_id_type":"open_id"}`, `{"delete_ids":[{"type":"user","user_id":"ou_fixture"}],"need_notification":false}`},
		{"event.attendees", "create", "POST", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0/attendees", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","user_id_type":"open_id"}`, `{"user_id_type":"open_id"}`, `{"attendees":[{"type":"user","user_id":"ou_fixture"},{"type":"chat","chat_id":"oc_fixture"},{"type":"resource","room_id":"omm_fixture"},{"type":"third_party","third_party_email":"synthetic@example.invalid"}],"need_notification":false,"add_operator_to_attendee":false}`},
		{"event.attendees", "list", "GET", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0/attendees", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","user_id_type":"open_id","need_resource_customization":false,"page_token":"next_fixture","page_size":2}`, `{"user_id_type":"open_id","need_resource_customization":"false","page_token":"next_fixture","page_size":"2"}`, `{}`},
		{"events", "create", "POST", "/open-apis/calendar/v4/calendars/cal_fixture/events", `{"calendar_id":"cal_fixture","idempotency_key":"fixture_idempotency","user_id_type":"open_id"}`, `{"idempotency_key":"fixture_idempotency","user_id_type":"open_id"}`, `{"summary":"日程 👋","description":"","description_rich":"**rich**","start_time":{"date":"2026-09-07","timezone":"UTC"},"end_time":{"date":"2026-09-08","timezone":"UTC"},"recurrence":"FREQ=DAILY;COUNT=2","need_notification":false,"visibility":"private","attendee_ability":"none","free_busy_status":"busy","color":0,"reminders":[{"minutes":0}],"vchat":{"vc_type":"no_meeting"},"location":{"name":"Fixture","address":"","latitude":0,"longitude":0},"attachments":[{"file_token":"file_fixture","name":"附件"}],"event_check_in":{"enable_check_in":false,"need_notify_attendees":false}}`},
		{"events", "delete", "DELETE", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","need_notification":"fixture"}`, `{"need_notification":"fixture"}`, `{}`},
		{"events", "get", "GET", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","need_meeting_settings":false,"need_attendee":false,"max_attendee_num":2,"user_id_type":"open_id"}`, `{"need_meeting_settings":"false","need_attendee":"false","max_attendee_num":"2","user_id_type":"open_id"}`, `{}`},
		{"events", "instance_view", "GET", "/open-apis/calendar/v4/calendars/cal_fixture/events/instance_view", `{"calendar_id":"cal_fixture","start_time":"1788775200","end_time":"1788778800","user_id_type":"open_id"}`, `{"start_time":"1788775200","end_time":"1788778800","user_id_type":"open_id"}`, `{}`},
		{"events", "patch", "PATCH", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0","user_id_type":"open_id"}`, `{"user_id_type":"open_id"}`, `{"summary":"日程 👋","description":"","description_rich":"**rich**","start_time":{"date":"2026-09-07","timezone":"UTC"},"end_time":{"date":"2026-09-08","timezone":"UTC"},"recurrence":"FREQ=DAILY;COUNT=2","need_notification":false,"visibility":"private","attendee_ability":"none","free_busy_status":"busy","color":0,"reminders":[{"minutes":0}],"vchat":{"vc_type":"no_meeting"},"location":{"name":"Fixture","address":"","latitude":0,"longitude":0},"attachments":[{"file_token":"file_fixture","name":"附件"}],"event_check_in":{"enable_check_in":false,"need_notify_attendees":false}}`},
		{"events", "search_event", "POST", "/open-apis/calendar/v4/calendars/cal_fixture/events/search_event", `{"calendar_id":"cal_fixture","page_token":"next_fixture","page_size":2,"user_id_type":"open_id"}`, `{"page_token":"next_fixture","page_size":"2","user_id_type":"open_id"}`, `{"query":"日程"}`},
		{"events", "share_info", "POST", "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0/share_info", `{"calendar_id":"cal_fixture","event_id":"evt_fixture_0"}`, `{}`, `{}`},
		{"freebusys", "list", "POST", "/open-apis/calendar/v4/freebusy/list", `{"user_id_type":"open_id"}`, `{"user_id_type":"open_id"}`, `{"time_min":"2026-09-07T10:00:00Z","time_max":"2026-09-07T11:00:00Z","user_id":"ou_fixture","include_external_calendar":false,"need_rsvp_status":true,"only_busy":false}`},
	} {
		t.Run(tc.resource+" "+tc.method, func(t *testing.T) {
			calls := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, tc.verb, r.Method)
				require.Equal(t, tc.path, r.URL.Path)
				var query map[string]string
				require.NoError(t, json.Unmarshal([]byte(tc.query), &query))
				for k, v := range query {
					require.Equal(t, v, r.URL.Query().Get(k), k)
				}
				if tc.body != "{}" {
					raw, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.JSONEq(t, tc.body, string(raw))
				}
				reply(w, map[string]any{"event": event(), "calendar": map[string]any{"calendar_id": "cal_fixture", "permissions": "private"}, "items": []any{}, "has_more": false, "fixture_marker": "preserved"})
			}))
			args := []string{"calendar", tc.resource, tc.method, "--as", "user", "--params", tc.params}
			if tc.resource == "calendars" && tc.method == "delete" {
				args = append(args, "--yes")
			}
			if tc.body != "{}" {
				args = append(args, "--data", tc.body)
			}
			result := f.Run(args...)
			if tc.method == "batch_delete" {
				require.Equal(t, 22, result.ExitCode)
				require.Contains(t, result.Stderr, `"outcome":"unknown"`)
			} else {
				success(t, result)
			}
			require.Contains(t, result.Stdout, "preserved")
			require.Equal(t, 1, calls)
		})
	}
}
