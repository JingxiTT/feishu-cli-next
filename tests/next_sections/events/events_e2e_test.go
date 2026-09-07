// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package events_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larksuite/cli/events"
	"github.com/larksuite/cli/internal/event/adapter/localbus/protocol"
	"github.com/larksuite/cli/internal/event/adapter/localbus/transport"
	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const appID = "cli-baseline-app"

// These are frozen synthetic reference inputs, not captured tenant content.
type replayFixtures struct {
	Events    map[string]json.RawMessage `json:"events"`
	Responses map[string]json.RawMessage `json:"responses"`
}

func readJSON(t *testing.T, path string, value any) {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, value))
}
func assertSuccess(t *testing.T, r nextfixture.Result) {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "stdout=%s\nstderr=%s", r.Stdout, r.Stderr)
}

// busFixture exercises the real process's production IPC address and protocol.
// It never starts the remote WebSocket daemon or forwards synthetic HTTP traffic.
type busFixture struct {
	mu      sync.Mutex
	hello   *protocol.Hello
	cleanup *protocol.PreShutdownCheck
}

func startBus(t *testing.T, f *nextfixture.Fixture, first, legacy bool, frames ...any) *busFixture {
	t.Helper()
	// Unix socket paths are short; the default test directory embeds the test name.
	temporaryRoot := os.TempDir()
	if runtime.GOOS != "windows" {
		temporaryRoot = "/tmp"
	}
	directory, err := os.MkdirTemp(temporaryRoot, "next-event-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(directory) })
	f.ConfigDir = directory
	f.Env["LARKSUITE_CLI_CONFIG_DIR"] = directory
	f.Env["LARKSUITE_CLI_APP_ID"] = appID
	f.Env["TZ"] = "UTC"
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", directory)
	tr := transport.New()
	address := tr.Address(appID)
	listener, err := tr.Listen(address)
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close(); tr.Cleanup(address) })
	bus := &busFixture{}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
				reader := bufio.NewReader(conn)
				for {
					line, err := protocol.ReadFrame(reader)
					if err != nil {
						return
					}
					message, err := protocol.Decode(bytes.TrimSpace(line))
					if err != nil {
						return
					}
					switch message := message.(type) {
					case *protocol.StatusQuery:
						_ = protocol.Encode(conn, protocol.NewStatusResponse(12345, 1, 1, []protocol.ConsumerInfo{{PID: 12346, EventKey: "im.chat.updated_v1"}}))
						return
					case *protocol.Hello:
						bus.mu.Lock()
						bus.hello = message
						bus.mu.Unlock()
						ack := protocol.NewHelloAck("fixture", first, protocol.CapabilityCanonicalMetadataV1)
						if legacy {
							ack.Capabilities = nil
						}
						if err := protocol.Encode(conn, ack); err != nil {
							return
						}
						for _, frame := range frames {
							if err := protocol.Encode(conn, frame); err != nil {
								return
							}
						}
					case *protocol.PreShutdownCheck:
						bus.mu.Lock()
						bus.cleanup = message
						bus.mu.Unlock()
						_ = protocol.Encode(conn, protocol.NewPreShutdownAck(true))
					case *protocol.Bye:
						return
					}
				}
			}()
		}
	}()
	return bus
}
func (b *busFixture) assertHandshake(t *testing.T, key string) {
	t.Helper()
	b.mu.Lock()
	defer b.mu.Unlock()
	require.NotNil(t, b.hello)
	require.Equal(t, key, b.hello.EventKey)
	require.Contains(t, b.hello.EventTypes, key)
}

func envelope(key string, body json.RawMessage) json.RawMessage {
	header := map[string]string{"event_id": "evt-baseline-001", "event_type": key, "create_time": "1700000000000", "app_id": appID, "tenant_key": "tenant-baseline"}
	b, _ := json.Marshal(map[string]any{"schema": "2.0", "header": header, "event": body})
	return b
}
func frame(key string, payload json.RawMessage, seq uint64) *protocol.Event {
	return &protocol.Event{Type: "event", EventType: key, EventID: "evt-baseline-001", SourceTime: "1700000000000", AppID: appID, TenantKey: "tenant-baseline", ObservedAt: "2023-11-14T22:13:20Z", Seq: seq, Payload: payload}
}

func TestAllEventKeysReplayThroughCLI(t *testing.T) {
	var fixtures replayFixtures
	readJSON(t, "testdata/replay.json", &fixtures)
	var expected map[string]json.RawMessage
	readJSON(t, "../../../events/testdata/output_baseline.json", &expected)
	definitions := events.All()
	require.Len(t, definitions, 25)
	require.Len(t, expected, 13)
	for _, definition := range definitions {
		t.Run(definition.Key, func(t *testing.T) {
			var mu sync.Mutex
			var writes []string
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == "POST" {
					mu.Lock()
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					writes = append(writes, r.URL.RequestURI()+" "+strings.TrimSpace(string(body)))
					mu.Unlock()
					_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
					return
				}
				if response, ok := fixtures.Responses[r.URL.RequestURI()]; ok {
					_, _ = w.Write(response)
					return
				}
				if r.URL.Path == "/open-apis/application/v6/applications/"+appID {
					_, _ = io.WriteString(w, `{"code":0,"data":{"app":{"callback_info":{"subscribed_callbacks":["card.action.trigger"]}}}}`)
					return
				}
				if strings.Contains(r.URL.Path, "/applications/") || r.URL.Path == "/open-apis/event/v1/connection" {
					_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
					return
				}
				t.Errorf("unexpected read: %s %s", r.Method, r.URL.RequestURI())
				http.Error(w, "unexpected", 500)
			}))
			body, processed := fixtures.Events[definition.Key]
			if !processed {
				// Native events preserve future/unknown fields and numeric tokens verbatim.
				body = json.RawMessage(`{"whiteboard_id":"board-fixture","chat_id":"oc-fixture","nullable":null,"empty":"","large":9007199254740993,"code":"0001","text":"中文 👋"}`)
			}
			payload := envelope(definition.Key, body)
			bus := startBus(t, f, true, false, frame(definition.Key, payload, 1))
			args := []string{"event", "consume", definition.Key, "--max-events", "1", "--timeout", "3s", "--as", definition.AuthTypes[0]}
			if definition.Key == "board.whiteboard.updated_v1" {
				args = append(args, "--param", "whiteboard_id=board-fixture")
			}
			r := f.Run(args...)
			assertSuccess(t, r)
			if processed {
				require.JSONEq(t, string(expected[definition.Key]), strings.TrimSpace(r.Stdout), r.Stderr)
			} else {
				require.JSONEq(t, string(payload), strings.TrimSpace(r.Stdout), r.Stderr)
				require.Contains(t, r.Stdout, `9007199254740993`)
			}
			bus.assertHandshake(t, definition.Key)
			require.Contains(t, r.Stderr, "reason: limit")
			mu.Lock()
			observed := append([]string(nil), writes...)
			mu.Unlock()
			if definition.PreConsume == nil {
				require.Empty(t, observed)
				require.NotContains(t, r.Stderr, `"mutation"`)
			} else {
				require.Equal(t, expectedSubscriptionWrites(definition.Key), observed)
				require.Contains(t, r.Stderr, `"outcome":"confirmed_applied"`)
				var receipts struct {
					Data []map[string]any `json:"data"`
				}
				listed := f.Run("receipts", "list")
				assertSuccess(t, listed)
				require.NoError(t, json.Unmarshal([]byte(listed.Stdout), &receipts))
				require.Len(t, receipts.Data, len(observed))
			}
		})
	}
}
func expectedSubscriptionWrites(key string) []string {
	if strings.HasPrefix(key, "task.") {
		return []string{"/open-apis/task/v2/task_v2/task_subscription?user_id_type=open_id "}
	}
	if strings.HasPrefix(key, "approval.") {
		entity := "instances"
		if strings.HasPrefix(key, "approval.task.") {
			entity = "tasks"
		}
		path := "/open-apis/approval/v4/" + entity + "/subscription "
		return []string{path + `{"subscription_type":"INVOLVED_APPROVAL"}`, path + `{"subscription_type":"MANAGED_APPROVAL"}`}
	}
	base := "/open-apis/minutes/v1/minutes/"
	subscribe, unsubscribe := "subscription", "unsubscription"
	switch {
	case strings.HasPrefix(key, "vc.meeting."):
		base = "/open-apis/vc/v1/meetings/"
	case strings.HasPrefix(key, "vc.note."):
		base = "/open-apis/vc/v1/notes/"
	case strings.HasPrefix(key, "vc.recording."):
		base = "/open-apis/vc/v1/recordings/"
	case strings.HasPrefix(key, "board."):
		base = "/open-apis/board/v1/whiteboards/board-fixture/"
		subscribe = "subscribe"
		unsubscribe = "unsubscribe"
	}
	body := ` {"event_type":"` + key + `"}`
	return []string{base + subscribe + body, base + unsubscribe + body}
}

func TestEventDiscoveryAndDryRun(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method, "dry-run must not mutate")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	r := f.Run("event", "list", "--json")
	assertSuccess(t, r)
	var keys []map[string]any
	require.NoError(t, json.Unmarshal([]byte(r.Stdout), &keys))
	require.Len(t, keys, 25)
	for _, def := range events.All() {
		t.Run(def.Key, func(t *testing.T) {
			r := f.Run("event", "schema", def.Key, "--json")
			assertSuccess(t, r)
			var schema map[string]any
			require.NoError(t, json.Unmarshal([]byte(r.Stdout), &schema))
			require.Equal(t, def.Key, schema["key"])
			require.NotEmpty(t, schema["resolved_output_schema"])
			args := []string{"event", "consume", def.Key, "--dry-run", "--as", def.AuthTypes[0]}
			if def.Key == "board.whiteboard.updated_v1" {
				args = append(args, "--param", "whiteboard_id=board-fixture")
			}
			r = f.Run(args...)
			assertSuccess(t, r)
			require.Contains(t, r.Stdout, `"dry_run":true`)
			require.NotContains(t, r.Stderr, `"mutation"`)
		})
	}
	for _, args := range [][]string{
		{"event", "consume", "bogus.key"},
		{"event", "consume", "im.message.receive_v1", "--param", "broken"},
		{"event", "consume", "approval.instance.status_changed_v4", "--as", "bot"},
		{"event", "consume", "im.message.receive_v1", "--result-fields", "content"},
		{"event", "consume", "im.message.receive_v1", "--save-result", "events.json"},
	} {
		r := f.Run(args...)
		require.NotZero(t, r.ExitCode, fmt.Sprint(args))
		require.Empty(t, r.Stdout)
		require.Contains(t, r.Stderr, `"error"`)
	}
	_, err := os.Stat(filepath.Join(f.WorkDir, "events.json"))
	require.True(t, os.IsNotExist(err))
}

func emptyBackend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
}

func TestStreamControlsAndDiagnostics(t *testing.T) {
	const key = "im.chat.updated_v1"
	payload := envelope(key, json.RawMessage(`{"chat_id":"oc-fixture","name":"中文 👋"}`))
	t.Run("jq and gap and reconnect status", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(emptyBackend))
		startBus(t, f, false, false,
			&protocol.SourceStatus{Type: "source_status", Source: "feishu", State: "reconnecting"},
			frame(key, payload, 1), frame(key, payload, 3))
		r := f.Run("event", "consume", key, "--as", "bot", "--max-events", "2", "--timeout", "3s", "--jq", ".event.chat_id")
		assertSuccess(t, r)
		require.Equal(t, "\"oc-fixture\"\n\"oc-fixture\"\n", r.Stdout)
		require.Contains(t, r.Stderr, "reconnecting")
		require.Contains(t, r.Stderr, "missed 1 events")
	})
	t.Run("file sink", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(emptyBackend))
		startBus(t, f, false, false, frame(key, payload, 1))
		r := f.Run("event", "consume", key, "--as", "bot", "--max-events", "1", "--timeout", "3s", "--output-dir", "events-out")
		assertSuccess(t, r)
		require.Empty(t, r.Stdout)
		files, err := filepath.Glob(filepath.Join(f.WorkDir, "events-out", "*.json"))
		require.NoError(t, err)
		require.Len(t, files, 1)
		b, err := os.ReadFile(files[0])
		require.NoError(t, err)
		require.JSONEq(t, string(payload), string(b))
	})
	t.Run("quiet timeout", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(emptyBackend))
		startBus(t, f, false, false)
		r := f.Run("event", "consume", key, "--as", "bot", "--timeout", "100ms", "--quiet")
		assertSuccess(t, r)
		require.Empty(t, r.Stdout)
		require.Empty(t, r.Stderr)
	})
	t.Run("stdin EOF", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(emptyBackend))
		startBus(t, f, false, false)
		r := f.Run("event", "consume", key, "--as", "bot")
		assertSuccess(t, r)
		require.Empty(t, r.Stdout)
		require.Contains(t, r.Stderr, "stdin closed")
	})
	t.Run("legacy native replay", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(emptyBackend))
		legacy := frame(key, payload, 1)
		legacy.AppID = ""
		legacy.TenantKey = ""
		legacy.ObservedAt = ""
		startBus(t, f, false, true, legacy)
		r := f.Run("event", "consume", key, "--as", "bot", "--max-events", "1", "--timeout", "3s")
		assertSuccess(t, r)
		require.JSONEq(t, string(payload), r.Stdout)
	})
	t.Run("legacy board cannot weaken subscription scope", func(t *testing.T) {
		f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "GET", r.Method, "must refuse before subscription")
			emptyBackend(w, r)
		}))
		startBus(t, f, true, true)
		r := f.Run("event", "consume", "board.whiteboard.updated_v1", "--as", "bot", "--param", "whiteboard_id=board-fixture", "--timeout", "100ms")
		require.Equal(t, 2, r.ExitCode, r.Stderr)
		require.Empty(t, r.Stdout)
		require.Contains(t, r.Stderr, "failed_precondition")
		require.NotContains(t, r.Stderr, `"mutation"`)
	})
}

func TestLifecycleMutationFailureEvidence(t *testing.T) {
	const key = "board.whiteboard.updated_v1"
	payload := envelope(key, json.RawMessage(`{"whiteboard_id":"board-fixture"}`))
	for _, scenario := range []struct {
		name                           string
		failCall, status, exit, writes int
		output                         bool
	}{
		{"subscription scope rejection", 1, 403, 20, 1, false},
		{"subscription unknown", 1, 503, 22, 1, false},
		{"cleanup unknown after emitted event", 2, 503, 22, 2, true},
		{"cleanup rejected after applied registration", 2, 403, 21, 2, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var mu sync.Mutex
			var writes int
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					mu.Lock()
					writes++
					current := writes
					mu.Unlock()
					if current == scenario.failCall {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(scenario.status)
						code := 999999
						if scenario.status == 403 {
							code = 99991679
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"code": code, "msg": "synthetic failure"})
						return
					}
				}
				emptyBackend(w, r)
			}))
			startBus(t, f, true, false, frame(key, payload, 1))
			r := f.Run("event", "consume", key, "--as", "bot", "--param", "whiteboard_id=board-fixture", "--max-events", "1", "--timeout", "3s")
			require.Equal(t, scenario.exit, r.ExitCode, r.Stderr)
			if scenario.output {
				require.JSONEq(t, string(payload), r.Stdout)
			} else {
				require.Empty(t, r.Stdout)
			}
			mu.Lock()
			observed := writes
			mu.Unlock()
			require.Equal(t, scenario.writes, observed, "a failed lifecycle request must not be replayed")
			require.Contains(t, r.Stderr, `"mutation"`)
			listed := f.Run("receipts", "list")
			assertSuccess(t, listed)
			var receipts struct {
				Data []map[string]any `json:"data"`
			}
			require.NoError(t, json.Unmarshal([]byte(listed.Stdout), &receipts))
			require.Len(t, receipts.Data, scenario.writes)
		})
	}
}

func TestEventBusStatusAndSafeStop(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	r := f.Run("event", "stop", "--json")
	assertSuccess(t, r)
	require.Contains(t, r.Stdout, `"no_bus"`)
	startBus(t, f, false, false)
	r = f.Run("event", "status", "--current", "--json")
	assertSuccess(t, r)
	require.Contains(t, r.Stdout, `"running"`)
	r = f.Run("event", "stop", "--json")
	require.Equal(t, 2, r.ExitCode, r.Stderr)
	require.Contains(t, r.Stdout, `"refused"`)
	require.Contains(t, r.Stdout, `consumer`)
}

func TestStreamInterruptPerformsCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Interrupt delivery is a Unix process contract; Windows cancellation is covered by context units")
	}
	var mu sync.Mutex
	var writes []string
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			mu.Lock()
			writes = append(writes, r.URL.Path)
			mu.Unlock()
		}
		emptyBackend(w, r)
	}))
	payload := envelope("board.whiteboard.updated_v1", json.RawMessage(`{"whiteboard_id":"board-fixture"}`))
	bus := startBus(t, f, true, false, frame("board.whiteboard.updated_v1", payload, 1))
	interrupted := make(chan error, 1)
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-timer.C:
				interrupted <- fmt.Errorf("CLI never attached")
				return
			case <-ticker.C:
				bus.mu.Lock()
				hello := bus.hello
				bus.mu.Unlock()
				files, _ := filepath.Glob(filepath.Join(f.WorkDir, "interrupt-events", "*.json"))
				if hello != nil && len(files) == 1 {
					process, err := os.FindProcess(hello.PID)
					if err == nil {
						err = process.Signal(os.Interrupt)
					}
					interrupted <- err
					return
				}
			}
		}
	}()
	// The timeout keeps stdin EOF from ending this process before the interrupt.
	r := f.Run("event", "consume", "board.whiteboard.updated_v1", "--as", "bot", "--param", "whiteboard_id=board-fixture", "--timeout", "10s", "--output-dir", "interrupt-events")
	require.NoError(t, <-interrupted)
	assertSuccess(t, r)
	require.Empty(t, r.Stdout)
	require.Contains(t, r.Stderr, "reason: signal")
	mu.Lock()
	observed := append([]string(nil), writes...)
	mu.Unlock()
	require.Equal(t, []string{"/open-apis/board/v1/whiteboards/board-fixture/subscribe", "/open-apis/board/v1/whiteboards/board-fixture/unsubscribe"}, observed)
}

func TestEscapedBoardSubscriptionReceipts(t *testing.T) {
	var mu sync.Mutex
	var writes []string
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			mu.Lock()
			writes = append(writes, r.URL.EscapedPath())
			mu.Unlock()
		}
		emptyBackend(w, r)
	}))
	startBus(t, f, true, false)
	r := f.Run("event", "consume", "board.whiteboard.updated_v1", "--as", "bot", "--param", "whiteboard_id=board/fixture", "--timeout", "100ms")
	assertSuccess(t, r)
	mu.Lock()
	observed := append([]string(nil), writes...)
	mu.Unlock()
	require.Equal(t, []string{"/open-apis/board/v1/whiteboards/board%2Ffixture/subscribe", "/open-apis/board/v1/whiteboards/board%2Ffixture/unsubscribe"}, observed)
	listed := f.Run("receipts", "list")
	assertSuccess(t, listed)
	var receipts struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(listed.Stdout), &receipts))
	require.Len(t, receipts.Data, 2)
}
