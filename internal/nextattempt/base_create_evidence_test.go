// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestBaseCreateIDsAreUniqueWithinInvocationScope(t *testing.T) {
	const path = "/open-apis/base/v3/bases/base/tables/table/records/batch_create"
	const responseBody = `{"code":0,"data":{"record_id_list":["private-created-id"]}}`
	baseBinding := Binding{Environment: "https://fixture.invalid", Account: "selected", AppID: "app", Identity: "user", UserID: "user"}
	for _, tt := range []struct {
		name, host, path string
		binding          Binding
		unknown          int
	}{
		{"same scope", "fixture.invalid", path, baseBinding, 1},
		{"another table", "fixture.invalid", strings.Replace(path, "/tables/table/", "/tables/other/", 1), baseBinding, 0},
		{"another host", "other.invalid", path, baseBinding, 0},
		{"another account", "fixture.invalid", path, Binding{Environment: baseBinding.Environment, Account: "other", AppID: "app", Identity: "user", UserID: "user"}, 0},
		{"another identity", "fixture.invalid", path, Binding{Environment: baseBinding.Environment, Account: "selected", AppID: "app", Identity: "bot"}, 0},
		{"another user", "fixture.invalid", path, Binding{Environment: baseBinding.Environment, Account: "selected", AppID: "app", Identity: "user", UserID: "other"}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
				calls++
				return response(200, responseBody), nil
			})
			binding := baseBinding
			tr.Resolve = func(*http.Request) (Binding, error) { return binding, nil }
			for i, endpoint := range []string{"https://fixture.invalid" + path, "https://" + tt.host + tt.path} {
				if i == 1 {
					binding = tt.binding
				}
				req, err := http.NewRequest("POST", endpoint, strings.NewReader(`{"create_records":[{"field":"private-content"}]}`))
				if err != nil {
					t.Fatal(err)
				}
				resp, err := tr.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil || string(body) != responseBody {
					t.Fatal("response changed")
				}
			}
			summary := journal.Summary()
			if calls != 2 || summary.Applied != 2-tt.unknown || summary.Unknown != tt.unknown {
				t.Fatalf("scope outcome: %+v calls=%d", summary, calls)
			}
			receipts, err := store.List()
			if err != nil || len(receipts) != 2 {
				t.Fatal(receipts, err)
			}
			unknown := 0
			for _, receipt := range receipts {
				if receipt.Outcome == Unknown {
					unknown++
				}
				raw, err := os.ReadFile(filepath.Join(store.Dir, receipt.ID+".json"))
				if err != nil || strings.Contains(string(raw), "private-created-id") || strings.Contains(string(raw), path) || strings.Contains(string(raw), "private-content") {
					t.Fatal("receipt missing or persisted request evidence")
				}
			}
			if unknown != tt.unknown {
				t.Fatal("unknown outcome was not durable")
			}
		})
	}
}

func TestConcurrentBaseCreateIDOverlapIsDurablyUnknown(t *testing.T) {
	var overlap sync.WaitGroup
	overlap.Add(2)
	tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		overlap.Done()
		overlap.Wait()
		return response(200, `{"code":0,"data":{"record_id_list":["same-id"]}}`), nil
	})
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			req, err := http.NewRequest("POST", "https://fixture.invalid/open-apis/base/v3/bases/base/tables/table/records/batch_create", strings.NewReader(`{"create_records":[{}]}`))
			if err == nil {
				var resp *http.Response
				resp, err = tr.RoundTrip(req)
				if resp != nil {
					resp.Body.Close()
				}
			}
			errors <- err
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
	}
	summary := journal.Summary()
	if summary.Applied != 1 || summary.Unknown != 1 || summary.ExitCode() != 22 {
		t.Fatal(summary)
	}
	receipts, err := store.List()
	if err != nil || len(receipts) != 2 || receipts[0].Outcome == receipts[1].Outcome {
		t.Fatal("concurrent outcome not durable", receipts, err)
	}
}

func TestBaseCreateEvidenceLimitsDoNotEvictEarlierIDs(t *testing.T) {
	target, _ := url.Parse("https://fixture.invalid/table")
	journal := &Journal{}
	for i := 0; i < baseCreateEvidenceMaxIDs; i++ {
		if !journal.observeBaseCreates(Binding{}, target, []string{strconv.Itoa(i)}) {
			t.Fatal("evidence capacity unexpectedly exceeded")
		}
	}
	if journal.observeBaseCreates(Binding{}, target, []string{"new-id"}) || journal.observeBaseCreates(Binding{}, target, []string{"0"}) {
		t.Fatal("capacity exhaustion discarded prior evidence")
	}
	journal = &Journal{}
	if !journal.observeBaseCreates(Binding{}, target, []string{strings.Repeat("x", baseCreateEvidenceMaxBytes-1000)}) || journal.observeBaseCreates(Binding{}, target, []string{strings.Repeat("y", 1000)}) {
		t.Fatal("evidence byte limit was not enforced")
	}
}
