// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/internal/vfs"
)

const importRequest = `{"file_token":"private-source","file_extension":"xlsx","file_name":"private-name","type":"bitable","token":"private-target","point":{"mount_type":1,"mount_key":"private-folder"}}`
const importProof = `{"result":{"job_status":0,"type":"bitable","token":"private-target"}}`

func pendingImport(t *testing.T, requestBody string) (*Transport, Store, *Journal, Receipt) {
	t.Helper()
	tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		return response(200, `{"code":0,"data":{"ticket":"private-ticket"}}`), nil
	})
	req, err := http.NewRequest("POST", "https://fixture.invalid/open-apis/drive/v1/import_tasks", strings.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	receipt := journal.Summary().Receipts[0]
	if receipt.Outcome != Unknown {
		t.Fatal("create acknowledgement confirmed completed import")
	}
	return tr, store, journal, receipt
}

func TestDriveImportCompletionRequiresNativeProofAndImmutableScope(t *testing.T) {
	for _, tt := range []struct {
		name, proof string
		applied     bool
	}{
		{"complete optional echoes absent", importProof, true},
		{"complete matching echoes", `{"result":{"job_status":0,"type":"bitable","token":"private-target","ticket":"private-ticket","file_token":"private-source","file_extension":"xlsx","file_name":"private-name","point":{"mount_type":1,"mount_key":"private-folder"},"job_error_msg":""}}`, true},
		{"missing status", `{"result":{"type":"bitable","token":"private-target"}}`, false},
		{"null status", `{"result":{"job_status":null,"type":"bitable","token":"private-target"}}`, false},
		{"string status", `{"result":{"job_status":"0","type":"bitable","token":"private-target"}}`, false},
		{"fractional status", `{"result":{"job_status":0.5,"type":"bitable","token":"private-target"}}`, false},
		{"pending", `{"result":{"job_status":1,"type":"bitable","token":"private-target"}}`, false},
		{"missing type", `{"result":{"job_status":0,"token":"private-target"}}`, false},
		{"wrong type", `{"result":{"job_status":0,"type":"sheet","token":"private-target"}}`, false},
		{"wrong target", `{"result":{"job_status":0,"type":"bitable","token":"other"}}`, false},
		{"missing token", `{"result":{"job_status":0,"type":"bitable"}}`, false},
		{"wrong ticket echo", `{"result":{"job_status":0,"type":"bitable","token":"private-target","ticket":"other"}}`, false},
		{"wrong source echo", `{"result":{"job_status":0,"type":"bitable","token":"private-target","file_token":"other"}}`, false},
		{"null source echo", `{"result":{"job_status":0,"type":"bitable","token":"private-target","file_token":null}}`, false},
		{"malformed source echo", `{"result":{"job_status":0,"type":"bitable","token":"private-target","file_token":42}}`, false},
		{"wrong folder", `{"result":{"job_status":0,"type":"bitable","token":"private-target","point":{"mount_key":"other"}}}`, false},
		{"null folder", `{"result":{"job_status":0,"type":"bitable","token":"private-target","point":{"mount_key":null}}}`, false},
		{"null point", `{"result":{"job_status":0,"type":"bitable","token":"private-target","point":null}}`, false},
		{"wrong mount type", `{"result":{"job_status":0,"type":"bitable","token":"private-target","point":{"mount_type":2}}}`, false},
		{"contradictory error", `{"result":{"job_status":0,"type":"bitable","token":"private-target","job_error_msg":"failed"}}`, false},
		{"unwrapped shape", `{"job_status":0,"type":"bitable","token":"private-target"}`, false},
		{"oversized proof", strings.Repeat(" ", responseEvidenceLimit) + importProof, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if DriveImportCompletionMatches("private-ticket", []byte(importRequest), []byte(tt.proof)) != tt.applied {
				t.Fatal("pure completion validation differs from receipt validation")
			}
			_, store, journal, receipt := pendingImport(t, importRequest)
			journal.ReconcileDriveImport(store, receipt.Binding, "https://fixture.invalid", "private-ticket", []byte(tt.proof))
			summary := journal.Summary()
			if (summary.Applied == 1) != tt.applied || summary.PersistenceFailed {
				t.Fatal(summary)
			}
			saved, err := store.Inspect(receipt.ID)
			if err != nil || saved.Outcome != summary.Outcome {
				t.Fatal("completion was not durably stored", saved, err)
			}
			raw, err := os.ReadFile(filepath.Join(store.Dir, receipt.ID+".json"))
			if err != nil || strings.Contains(string(raw), "private-") {
				t.Fatal("receipt persisted source, destination or ticket")
			}
		})
	}
	for _, changed := range []string{"ticket", "origin", "account", "identity", "tenant", "app", "user"} {
		t.Run(changed+" cannot reconcile", func(t *testing.T) {
			_, store, journal, receipt := pendingImport(t, importRequest)
			binding, origin, ticket := receipt.Binding, "https://fixture.invalid", "private-ticket"
			switch changed {
			case "ticket":
				ticket = "other"
			case "origin":
				origin = "https://other.invalid"
			case "account":
				binding.Account = "other"
			case "identity":
				binding.Identity = "user"
			case "tenant":
				binding.TenantKey = "other"
			case "app":
				binding.AppID = "other"
			case "user":
				binding.UserID = "other"
			}
			journal.ReconcileDriveImport(store, binding, origin, ticket, []byte(importProof))
			if journal.Summary().Unknown != 1 {
				t.Fatal("changed immutable import scope")
			}
		})
	}
}

func TestDriveImportDoesNotUpgradeOtherUnknownsOrAmbiguousTickets(t *testing.T) {
	for _, mode := range []string{"untracked", "unrelated reason", "reused ticket"} {
		t.Run(mode, func(t *testing.T) {
			body := importRequest
			if mode == "untracked" {
				body = `{}`
			}
			tr, store, journal, receipt := pendingImport(t, body)
			if mode == "unrelated reason" {
				journal.receipts[0].Evidence = "transport_error"
			}
			if mode == "reused ticket" {
				req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/drive/v1/import_tasks", strings.NewReader(importRequest))
				resp, err := tr.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
			}
			journal.ReconcileDriveImport(store, receipt.Binding, "https://fixture.invalid", "private-ticket", []byte(importProof))
			if journal.Summary().Applied != 0 {
				t.Fatal("unrelated or ambiguous evidence upgraded")
			}
		})
	}
}

type failDirectoryOpenFS struct{ vfs.FS }

func (failDirectoryOpenFS) Open(string) (*os.File, error) {
	return nil, errors.New("synthetic directory sync open failure")
}

func TestDriveImportCompletionPersistenceFailureRemainsUnknown(t *testing.T) {
	for _, fault := range []string{"rename", "directory sync", "deleted", "corrupt", "locked"} {
		t.Run(fault, func(t *testing.T) {
			if fault == "directory sync" && runtime.GOOS == "windows" {
				t.Skip("Windows does not perform directory fsync")
			}
			_, store, journal, receipt := pendingImport(t, importRequest)
			switch fault {
			case "rename":
				previous := vfs.DefaultFS
				vfs.DefaultFS = failRenameFS{previous}
				t.Cleanup(func() { vfs.DefaultFS = previous })
			case "directory sync":
				previous := vfs.DefaultFS
				vfs.DefaultFS = failDirectoryOpenFS{previous}
				t.Cleanup(func() { vfs.DefaultFS = previous })
			case "deleted":
				if err := store.Cleanup(receipt.ID); err != nil {
					t.Fatal(err)
				}
			case "corrupt":
				if err := os.WriteFile(filepath.Join(store.Dir, receipt.ID+".json"), []byte("corrupt"), 0600); err != nil {
					t.Fatal(err)
				}
			case "locked":
				lock, err := store.lock(receipt.ID)
				if err != nil {
					t.Fatal(err)
				}
				defer lock.Unlock()
			}
			journal.ReconcileDriveImport(store, receipt.Binding, "https://fixture.invalid", "private-ticket", []byte(importProof))
			if summary := journal.Summary(); summary.Unknown != 1 || !summary.PersistenceFailed || summary.ExitCode() != 22 {
				t.Fatal("local failure allowed success", summary)
			}
			if fault == "directory sync" {
				if visible, err := store.Inspect(receipt.ID); err != nil || visible.Outcome != Applied {
					t.Fatal("post-rename failure must acknowledge visible but not durably confirmed replacement", visible, err)
				}
			}
		})
	}
}

func TestConcurrentDriveImportCompletionKeepsOtherAttempts(t *testing.T) {
	tr, store, journal, receipt := pendingImport(t, importRequest)
	tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, `{"code":0,"data":{"task_id":"unrelated"}}`), nil
	})
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/docs_ai/v1/documents/doc/history/revert", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	var wait sync.WaitGroup
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			journal.ReconcileDriveImport(store, receipt.Binding, "https://fixture.invalid", "private-ticket", []byte(importProof))
		}()
	}
	wait.Wait()
	if summary := journal.Summary(); summary.Applied != 1 || summary.Unknown != 1 || summary.PersistenceFailed {
		t.Fatal(summary)
	}
	if saved, err := store.Inspect(receipt.ID); err != nil || saved.Outcome != Applied {
		t.Fatal("completion not durable", saved, err)
	}
}

func TestDriveImportCaptureIsBoundedAndDoesNotEvict(t *testing.T) {
	for _, large := range []bool{false, true} {
		journal := &Journal{}
		body := importRequest
		limit := 128
		if large {
			body = strings.Replace(body, "private-name", strings.Repeat("n", 800<<10), 1)
			limit = 5
		}
		for i := 0; i <= limit; i++ {
			req, err := http.NewRequest("POST", "https://fixture.invalid/open-apis/drive/v1/import_tasks", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			captured := journal.captureDriveImport(Receipt{ID: strconv.Itoa(i)}, req, []byte(`{"ticket":"ticket-`+strconv.Itoa(i)+`"}`))
			if captured != (i < limit) {
				t.Fatalf("large=%v index=%d captured=%v", large, i, captured)
			}
		}
		if !journal.HasDriveImport("ticket-0") || journal.HasDriveImport("ticket-"+strconv.Itoa(limit)) || journal.driveImportSize > 4<<20 {
			t.Fatal("capacity exhaustion evicted proof or retained oversized evidence")
		}
	}
}

func TestDriveImportAcceptedResponseAndRawStatusDoNotReconcile(t *testing.T) {
	tr, store, journal := testTransport(t, func(*http.Request) (*http.Response, error) {
		return response(202, `{"code":0,"data":{"ticket":"private-ticket"}}`), nil
	})
	req, _ := http.NewRequest("POST", "https://fixture.invalid/open-apis/drive/v1/import_tasks", strings.NewReader(importRequest))
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if journal.HasDriveImport("private-ticket") {
		t.Fatal("unclassified HTTP202 evidence was captured as native import acknowledgement")
	}
	receipt := journal.Summary().Receipts[0]
	journal.ReconcileDriveImport(store, receipt.Binding, "https://fixture.invalid", "private-ticket", []byte(importProof))
	if journal.Summary().Unknown != 1 {
		t.Fatal("non-native acknowledgement upgraded")
	}
	tr, _, journal, _ = pendingImport(t, importRequest)
	tr.IsMutation = func(req *http.Request) bool { return req.Method == "POST" }
	tr.Base = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response(200, `{"code":0,"data":`+importProof+`}`), nil
	})
	req, _ = http.NewRequest("GET", "https://fixture.invalid/open-apis/drive/v1/import_tasks/private-ticket", nil)
	resp, err = tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if journal.Summary().Unknown != 1 || len(journal.Summary().Receipts) != 1 {
		t.Fatal("raw GET auto-reconciled or created a receipt")
	}
}
