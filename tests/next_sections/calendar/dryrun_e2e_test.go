// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar_test

import (
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestCalendarEveryShortcutDryRun(t *testing.T) {
	for _, tc := range []struct {
		command string
		args    []string
		path    string
	}{
		{"+agenda", []string{"--start", start, "--end", end}, "/events/instance_view"},
		{"+create", []string{"--start", start, "--end", end, "--summary", "日程", "--description", "**rich**", "--rrule", "FREQ=DAILY;COUNT=3", "--attendee-ids", "ou_fixture,oc_fixture,omm_fixture"}, "/attendees"},
		{"+delete", []string{"--event-id", "evt_fixture_0", "--apply-to", "all", "--notify=false"}, "/events/evt_fixture_0"},
		{"+update", []string{"--event-id", "evt_fixture_0", "--summary", "changed", "--description", "", "--start", start, "--end", end, "--rrule", "FREQ=WEEKLY;COUNT=2", "--add-attendee-ids", "ou_fixture", "--remove-attendee-ids", "omm_fixture", "--notify=false"}, "/attendees/batch_delete"},
		{"+get", []string{"--event-id", "evt_fixture_0"}, "/events/evt_fixture_0"},
		{"+freebusy", []string{"--start", start, "--end", end, "--user-id", "ou_fixture", "--type", "common_free", "--min-duration", "30m"}, "/freebusy/batch"},
		{"+room-find", []string{"--slot", start + "~" + end, "--city", "Synthetic", "--building", "A", "--floor", "F2", "--room-name", "A,B", "--min-capacity", "2", "--max-capacity", "6", "--timezone", "UTC", "--attendee-ids", "ou_fixture,oc_fixture", "--event-rrule", "FREQ=DAILY;COUNT=2"}, "/freebusy/room_find"},
		{"+rsvp", []string{"--event-id", "evt_fixture_0", "--rsvp-status", "accept"}, "/reply"},
		{"+suggestion", []string{"--start", start, "--end", end, "--duration-minutes", "30", "--timezone", "UTC", "--attendee-ids", "ou_fixture,oc_fixture", "--event-rrule", "FREQ=WEEKLY;COUNT=2", "--exclude", start + "~" + end}, "/freebusy/suggestion"},
		{"+meeting", []string{"--event-ids", "evt_fixture_0,evt_other_0"}, "/mget_instance_relation_info"},
		{"+search-event", []string{"--query", "日程", "--attendee-ids", "ou_fixture,oc_fixture,omm_fixture", "--start", start, "--end", end, "--page-token", "next_fixture", "--page-size", "2"}, "/search_event"},
		{"+transfer", []string{"--event-id", "evt_fixture_0", "--to-user-id", "ou_fixture", "--transfer-series", "--remove-original-organizer"}, "/transfer"},
		{"+join-event", []string{"--share-token", "share_fixture"}, "/join_event"},
		{"+list-attendees", []string{"--event-id", "evt_fixture_0", "--type", "user,resource,chat,third_party", "--page-size", "2", "--page-token", "next_fixture"}, "/attendees"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			args := append([]string{"calendar", tc.command}, tc.args...)
			if tc.command != "+join-event" && tc.command != "+freebusy" && tc.command != "+room-find" && tc.command != "+suggestion" {
				args = append(args, "--calendar-id", "cal_fixture")
			}
			args = append(args, "--as", "user", "--dry-run")
			result := f.Run(args...)
			require.Equal(t, 0, result.ExitCode, "%s\n%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout, tc.path)
			require.Contains(t, strings.ToLower(result.Stdout), "method")
			require.NotContains(t, result.Stderr, `"receipts"`)
		})
	}
}
