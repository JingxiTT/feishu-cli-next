// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar

import (
	"net/http"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/stretchr/testify/require"
)

func TestCalendarUpdateExactReplacementNoOp(t *testing.T) {
	for _, tc := range []struct {
		name      string
		args      []string
		wantPatch bool
	}{
		{"summary", []string{"--summary", "日程"}, false},
		{"description", []string{"--description", "**exact**"}, false},
		{"timestamps", []string{"--start", "2026-09-07T10:00:00Z", "--end", "2026-09-07T11:00:00Z"}, false},
		{"different", []string{"--summary", "different"}, true},
		{"clear", []string{"--description", ""}, true},
		{"explicit notification", []string{"--summary", "日程", "--notify=true"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			f, stdout, _, reg := cmdutil.TestFactory(t, defaultConfig())
			event := map[string]interface{}{"event_id": "evt_fixture_0", "summary": "日程", "description_rich": "**exact**", "start_time": map[string]string{"timestamp": "1788775200"}, "end_time": map[string]string{"timestamp": "1788778800"}}
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"event": event}}})
			patched := false
			reg.Register(&httpmock.Stub{Method: "PATCH", URL: "/open-apis/calendar/v4/calendars/cal_fixture/events/evt_fixture_0", Body: map[string]interface{}{"code": 0, "data": map[string]interface{}{"event": event}}, Optional: true, OnMatch: func(r *http.Request) { patched = true }})
			args := append([]string{"+update", "--as", "bot", "--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--skip-room-check"}, tc.args...)
			require.NoError(t, mountAndRun(t, CalendarUpdate, args, f, stdout))
			require.Equal(t, tc.wantPatch, patched, stdout.String())
			if !tc.wantPatch {
				require.Contains(t, stdout.String(), `"no_op": true`)
			}
		})
	}
}
