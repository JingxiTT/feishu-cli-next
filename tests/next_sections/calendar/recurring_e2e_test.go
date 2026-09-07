// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestCalendarRecurringScopes(t *testing.T) {
	for _, tc := range []struct {
		name, command, scope string
		timeChange           bool
	}{
		{"update all", "+update", "all", false},
		{"move all", "+update", "all", true},
		{"delete all", "+delete", "all", false},
		{"update following", "+update", "this-and-following", false},
		{"delete following", "+delete", "this-and-following", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			master := event()
			master["recurrence"] = "FREQ=DAILY;COUNT=3"
			pivotID := "evt_fixture_1788861600"
			pivot := event()
			pivot["event_id"] = pivotID
			pivot["start_time"] = map[string]string{"timestamp": "1788861600", "timezone": "UTC"}
			pivot["end_time"] = map[string]string{"timestamp": "1788865200", "timezone": "UTC"}
			exceptionPath := calendarPath + "/events/evt_fixture_1788948000"
			var writes []string
			var newSeries map[string]any
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					switch r.URL.Path {
					case eventPath:
						reply(w, map[string]any{"event": master})
					case calendarPath + "/events/" + pivotID:
						reply(w, map[string]any{"event": pivot})
					case eventPath + "/instances":
						reply(w, map[string]any{"items": []any{map[string]any{"event_id": "evt_fixture_1788948000", "is_exception": true, "start_time": map[string]string{"timestamp": "1788948000"}}}, "has_more": false})
					case eventPath + "/attendees":
						reply(w, map[string]any{"items": []any{map[string]any{"type": "third_party", "third_party_email": "synthetic@example.invalid"}}, "has_more": false})
					default:
						t.Errorf("unexpected GET %s", r.URL)
						http.Error(w, "unexpected", 500)
					}
					return
				}
				writes = append(writes, r.Method+" "+r.URL.Path)
				switch r.Method {
				case "DELETE":
					require.Equal(t, "false", r.URL.Query().Get("need_notification"))
					if r.URL.Path == exceptionPath {
						require.Equal(t, "true", r.URL.Query().Get("delete_exception"))
					}
					reply(w, map[string]any{})
				case "PATCH":
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, false, body["need_notification"])
					if tc.scope == "this-and-following" {
						require.Contains(t, body["recurrence"], "UNTIL=")
					} else {
						require.Equal(t, "Changed", body["summary"])
					}
					reply(w, map[string]any{"event": master})
				case "POST":
					if r.URL.Path == calendarPath+"/events" {
						require.NoError(t, json.NewDecoder(r.Body).Decode(&newSeries))
						require.Equal(t, "Changed", newSeries["summary"])
						a, _ := json.Marshal(master["attachments"])
						b, _ := json.Marshal(newSeries["attachments"])
						require.JSONEq(t, string(a), string(b))
						require.Equal(t, master["event_check_in"], newSeries["event_check_in"])
						newSeries["event_id"] = "following_fixture_0"
						reply(w, map[string]any{"event": newSeries})
					} else {
						require.Equal(t, calendarPath+"/events/following_fixture_0/attendees", r.URL.Path)
						var body map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, false, body["need_notification"])
						require.Equal(t, []any{map[string]any{"type": "third_party", "third_party_email": "synthetic@example.invalid"}}, body["attendees"])
						reply(w, map[string]any{})
					}
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
				}
			}))
			id := "evt_fixture_0"
			if tc.scope == "this-and-following" {
				id = pivotID
			}
			args := []string{"calendar", tc.command, "--as", "user", "--calendar-id", "cal_fixture", "--event-id", id, "--apply-to", tc.scope, "--notify=false"}
			if tc.command == "+delete" {
				args = append(args, "--yes")
			}
			if tc.command == "+update" {
				args = append(args, "--summary", "Changed", "--skip-room-check")
				if tc.timeChange {
					args = append(args, "--start", "2026-09-07T12:00:00Z", "--end", "2026-09-07T13:00:00Z")
				}
			}
			result := f.Run(args...)
			success(t, result)
			require.Contains(t, result.Stdout, tc.scope)
			require.GreaterOrEqual(t, len(writes), 2)
			first := "PATCH "
			if tc.command == "+delete" || tc.timeChange || tc.scope == "this-and-following" {
				first = "DELETE "
			}
			require.Equal(t, first+exceptionPath, writes[0])
			if tc.scope == "this-and-following" {
				require.Equal(t, "PATCH "+eventPath, writes[1])
			} else {
				require.True(t, strings.HasSuffix(writes[len(writes)-1], eventPath))
			}
			if tc.command == "+update" && tc.scope == "this-and-following" {
				require.Equal(t, 4, len(writes))
				require.NotNil(t, newSeries)
			}
		})
	}
}

func TestCalendarRecurringAmbiguityDoesNotWrite(t *testing.T) {
	for _, command := range []string{"+update", "+delete", "+transfer"} {
		t.Run(command, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "GET", r.Method)
				e := event()
				e["recurrence"] = "FREQ=DAILY;COUNT=3"
				reply(w, map[string]any{"event": e})
			}))
			args := []string{"calendar", command, "--as", "user", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0"}
			if command != "+update" {
				args = append(args, "--yes")
			}
			if command == "+update" {
				args = append(args, "--summary", "Changed", "--description", "![fixture](missing-image.png)")
			}
			if command == "+transfer" {
				args = append(args, "--to-user-id", "ou_fixture")
			}
			result := f.Run(args...)
			require.NotEqual(t, 0, result.ExitCode)
			require.Equal(t, "validation", errorType(t, result.Stderr))
			require.NotContains(t, result.Stderr, `"receipts"`)
		})
	}
}
