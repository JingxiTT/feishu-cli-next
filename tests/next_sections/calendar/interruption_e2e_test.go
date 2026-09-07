// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package calendar_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/nextattempt"
	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestCalendarRecurringProcessInterruptionKeepsUnknownReceipt(t *testing.T) {
	const exceptionID = "evt_fixture_1788948000"
	const changedSummary = "Interrupted fixture replacement"
	exceptionPath := calendarPath + "/events/" + exceptionID
	master := event()
	master["recurrence"] = "FREQ=DAILY;COUNT=3"
	exception := event()
	exception["event_id"] = exceptionID
	exception["is_exception"] = true
	exception["start_time"] = map[string]string{"timestamp": "1788948000", "timezone": "UTC"}
	exception["end_time"] = map[string]string{"timestamp": "1788951600", "timezone": "UTC"}

	var stateMu sync.Mutex
	var writes, attendeeRequests, masterWrites atomic.Int32
	type dispatchEvidence struct {
		receipts []nextattempt.Receipt
		err      error
	}
	dispatched := make(chan dispatchEvidence, 1)
	release := make(chan struct{})
	defer close(release)
	var fixture *nextfixture.Fixture
	fixture = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			stateMu.Lock()
			defer stateMu.Unlock()
			switch r.URL.Path {
			case eventPath:
				reply(w, map[string]any{"event": master})
			case eventPath + "/instances":
				reply(w, map[string]any{"items": []any{exception}, "has_more": false})
			case exceptionPath:
				reply(w, map[string]any{"event": exception})
			default:
				t.Errorf("unexpected read %s", r.URL)
				http.Error(w, "unexpected fixture read", http.StatusBadRequest)
			}
			return
		}
		writes.Add(1)
		if r.URL.Path == exceptionPath+"/attendees" || r.URL.Path == eventPath+"/attendees" {
			attendeeRequests.Add(1)
		}
		if r.URL.Path == eventPath {
			masterWrites.Add(1)
		}
		if r.Method != http.MethodPatch || r.URL.Path != exceptionPath {
			t.Errorf("unexpected mutation %s %s", r.Method, r.URL)
			http.Error(w, "unexpected fixture mutation", http.StatusBadRequest)
			return
		}
		var body struct {
			Summary          string `json:"summary"`
			NeedNotification bool   `json:"need_notification"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			dispatched <- dispatchEvidence{err: err}
			return
		}
		if body.Summary != changedSummary || body.NeedNotification {
			t.Errorf("unexpected exception patch: %+v", body)
		}

		// Inspect the durable journal before the synthetic backend applies the
		// PATCH. Never acknowledge it: the process cannot know whether it applied.
		store := nextattempt.Store{Dir: filepath.Join(fixture.ConfigDir, "next", "receipts")}
		receipts, err := store.List()
		stateMu.Lock()
		exception["summary"] = body.Summary
		stateMu.Unlock()
		dispatched <- dispatchEvidence{receipts: receipts, err: err}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	finished := make(chan nextfixture.Result, 1)
	go func() {
		finished <- fixture.RunContext(ctx, "calendar", "+update", "--as", "user",
			"--calendar-id", "cal_fixture", "--event-id", "evt_fixture_0", "--apply-to", "all",
			"--summary", changedSummary, "--add-attendee-ids", "ou_foundation_fixture",
			"--notify=false", "--skip-room-check")
	}()

	var before dispatchEvidence
	select {
	case before = <-dispatched:
		cancel()
	case <-ctx.Done():
		t.Fatal("recurring update did not reach the hanging exception PATCH")
	}
	select {
	case result := <-finished:
		require.NotZero(t, result.ExitCode, "killed process must not report success: %+v", result)
		require.Empty(t, result.Stdout, "interrupted update must not emit a successful result")
	case <-time.After(5 * time.Second):
		t.Fatal("calendar process did not terminate after cancellation")
	}

	require.NoError(t, before.err)
	require.Len(t, before.receipts, 1, "exactly one durable attempt must precede the backend effect")
	receipt := before.receipts[0]
	require.Equal(t, nextattempt.Unknown, receipt.Outcome)
	require.Equal(t, "dispatch_started", receipt.Evidence)
	require.Equal(t, http.MethodPatch, receipt.Method)
	require.Equal(t, "user", receipt.Binding.Identity)
	require.Equal(t, "ou_foundation_fixture", receipt.Binding.UserID)
	require.Equal(t, "foundation-fixture-tenant", receipt.Binding.TenantKey)
	require.True(t, receipt.Binding.UserVerified)
	require.True(t, receipt.Binding.TenantVerified)
	require.Zero(t, receipt.HTTPStatus)
	require.Nil(t, receipt.APICode)

	store := nextattempt.Store{Dir: filepath.Join(fixture.ConfigDir, "next", "receipts")}
	after, err := store.List()
	require.NoError(t, err)
	require.Equal(t, before.receipts, after, "abrupt termination must leave the durable attempt unknown")
	inspect := fixture.Run("receipts", "inspect", receipt.ID)
	success(t, inspect)
	require.Contains(t, inspect.Stdout, `"outcome":"unknown"`)
	require.Contains(t, inspect.Stdout, `"evidence":"dispatch_started"`)

	// Independent readback proves that unknown is not equivalent to unapplied.
	readback := fixture.Run("calendar", "+get", "--as", "user", "--calendar-id", "cal_fixture", "--event-id", exceptionID)
	success(t, readback)
	require.Contains(t, readback.Stdout, changedSummary)
	stateMu.Lock()
	unchangedMasterSummary := master["summary"]
	stateMu.Unlock()
	require.Equal(t, "日程 👋", unchangedMasterSummary)
	require.EqualValues(t, 1, writes.Load(), "no retry or continuation after process interruption")
	require.Zero(t, attendeeRequests.Load(), "the planned attendee step must not run")
	require.Zero(t, masterWrites.Load(), "the master update must not run after the interrupted exception")
}
