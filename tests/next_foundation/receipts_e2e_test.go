// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/internal/nextattempt"
)

type receiptWireSummary struct {
	Outcome           string   `json:"outcome"`
	Receipts          []string `json:"receipts"`
	Applied           int      `json:"applied"`
	Unapplied         int      `json:"unapplied"`
	Unknown           int      `json:"unknown"`
	PersistenceFailed bool     `json:"persistence_failed,omitempty"`
}

func TestRawContactBasicBatchReadHasNoMutationReceipt(t *testing.T) {
	t.Parallel()
	var reads atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.Method != "POST" || r.URL.Path != "/open-apis/contact/v3/users/basic_batch" || r.URL.Query().Get("user_id_type") != "open_id" {
			t.Errorf("unexpected Contact read: %s %s", r.Method, r.URL)
		}
		var body struct {
			UserIDs []string `json:"user_ids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.UserIDs) != 1 || body.UserIDs[0] != "ou_synthetic" {
			t.Errorf("Contact request body changed: %+v, %v", body, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"users":[{"user_id":"ou_synthetic","name":"测试"}]}}`))
	}))
	result := fixture.Run("api", "POST", "/open-apis/contact/v3/users/basic_batch", "--params", `{"user_id_type":"open_id"}`, "--data", `{"user_ids":["ou_synthetic"]}`, "--as", "user")
	if result.ExitCode != 0 || reads.Load() != 1 || !strings.Contains(result.Stdout, `"user_id":"ou_synthetic"`) || !strings.Contains(result.Stdout, "测试") || strings.Contains(result.Stdout+result.Stderr, `"mutation"`) {
		t.Fatalf("raw Contact read changed: %+v, reads=%d", result, reads.Load())
	}
	receipts, err := (nextattempt.Store{Dir: filepath.Join(fixture.ConfigDir, "next", "receipts")}).List()
	if err != nil || len(receipts) != 0 {
		t.Fatalf("read created mutation receipts: %+v, %v", receipts, err)
	}
}

func receiptSummary(t *testing.T, result Result) receiptWireSummary {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(result.Stderr))
	for decoder.More() {
		var row struct {
			Mutation *receiptWireSummary `json:"mutation"`
		}
		if err := decoder.Decode(&row); err != nil {
			t.Fatalf("stderr JSON: %v\n%s", err, result.Stderr)
		}
		if row.Mutation != nil {
			return *row.Mutation
		}
	}
	t.Fatalf("no mutation summary: %+v", result)
	return receiptWireSummary{}
}
func TestReceiptSuccessInspectAndCleanup(t *testing.T) {
	t.Parallel()
	var fixture *Fixture
	var writes atomic.Int32
	fixture = NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		store := nextattempt.Store{Dir: filepath.Join(fixture.ConfigDir, "next", "receipts")}
		records, err := store.List()
		if err != nil || len(records) != 1 || records[0].Outcome != nextattempt.Unknown {
			t.Errorf("missing pre-write evidence: %v %v", records, err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"id":"synthetic-created-id"}}`))
	}))
	result := fixture.Run("api", "POST", "/open-apis/fixture/v1/items", "--data", `{"text":"synthetic-private-body"}`, "--as", "bot")
	if result.ExitCode != 0 || writes.Load() != 1 {
		t.Fatalf("result %+v writes %d", result, writes.Load())
	}
	summary := receiptSummary(t, result)
	if summary.Applied != 1 || len(summary.Receipts) != 1 {
		t.Fatal(summary)
	}
	id := summary.Receipts[0]
	inspect := fixture.Run("receipts", "inspect", id)
	if inspect.ExitCode != 0 || !strings.Contains(inspect.Stdout, "confirmed_applied") {
		t.Fatal(inspect)
	}
	raw, err := os.ReadFile(filepath.Join(fixture.ConfigDir, "next", "receipts", id+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "synthetic-private-body") || strings.Contains(string(raw), "proxy-key") {
		t.Fatalf("unsafe journal %s", raw)
	}
	cleanup := fixture.Run("receipts", "cleanup", id)
	if cleanup.ExitCode != 0 {
		t.Fatal(cleanup)
	}
	if _, err := os.Stat(filepath.Join(fixture.ConfigDir, "next", "receipts", id+".json")); !os.IsNotExist(err) {
		t.Fatalf("receipt not removed: %v", err)
	}
	if writes.Load() != 1 {
		t.Fatal("receipt commands contacted backend")
	}
}
func TestReceiptLostResponseIsUnknownWithoutRetry(t *testing.T) {
	t.Parallel()
	var writes atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		connection, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = connection.Close()
	}))
	result := fixture.Run("api", "POST", "/open-apis/fixture/v1/items", "--data", `{}`, "--as", "bot")
	if result.ExitCode != 22 || writes.Load() != 1 {
		t.Fatalf("result %+v writes %d", result, writes.Load())
	}
	if summary := receiptSummary(t, result); summary.Unknown != 1 {
		t.Fatal(summary)
	}
}
func TestReceiptRejectedAndMalformedResponses(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name, body string
		exit       int
		outcome    string
	}{
		{"rejected", `{"code":99991679,"msg":"synthetic missing scope"}`, 20, nextattempt.Unapplied},
		{"malformed", `{"broken":`, 22, nextattempt.Unknown},
		{"batch", `{"code":0,"data":{"items":[{"code":0},{"code":42}]}}`, 22, nextattempt.Unknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			result := fixture.Run("api", "POST", "/open-apis/fixture/v1/items", "--data", `{}`, "--as", "bot")
			if result.ExitCode != tt.exit {
				t.Fatal(result)
			}
			if summary := receiptSummary(t, result); summary.Outcome != tt.outcome {
				t.Fatal(summary)
			}
			if tt.name == "batch" {
				var envelope struct {
					OK       bool `json:"ok"`
					Mutation struct {
						Outcome string `json:"outcome"`
					} `json:"mutation"`
				}
				if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil {
					t.Fatalf("invalid stdout: %v %s", err, result.Stdout)
				}
				if envelope.OK || envelope.Mutation.Outcome != nextattempt.Unknown {
					t.Fatalf("misleading batch success: %s", result.Stdout)
				}
			}
		})
	}
}
func TestReceiptStateFailurePreventsWrite(t *testing.T) {
	t.Parallel()
	var writes atomic.Int32
	fixture := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writes.Add(1)
		t.Error("write dispatched despite failed evidence persistence")
	}))
	if err := os.WriteFile(filepath.Join(fixture.ConfigDir, "next"), []byte("synthetic blocking file"), 0600); err != nil {
		t.Fatal(err)
	}
	result := fixture.Run("api", "POST", "/open-apis/fixture/v1/items", "--data", `{}`, "--as", "bot")
	if result.ExitCode == 0 || writes.Load() != 0 {
		t.Fatalf("result %+v writes %d", result, writes.Load())
	}
}
