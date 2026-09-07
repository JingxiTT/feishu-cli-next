// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/nextattempt"
	"github.com/larksuite/cli/tests/nextfixture"
)

func TestImportTaskResultRejectsMalformedCompletionStatus(t *testing.T) {
	for _, mode := range []string{"missing", "null", "string", "fractional", "wrong_ticket", "accepted", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			var polls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/open-apis/drive/v1/import_tasks/import1" {
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
					return
				}
				polls.Add(1)
				data := map[string]any{"token": "doc1", "type": "docx"}
				switch mode {
				case "null":
					data["job_status"] = nil
				case "string":
					data["job_status"] = "0"
				case "fractional":
					data["job_status"] = 0.5
				case "wrong_ticket":
					data["job_status"] = 0
					data["ticket"] = "other"
				}
				if mode == "accepted" || mode == "redirect" {
					data["job_status"] = 0
					w.Header().Set("Content-Type", "application/json")
					code := http.StatusAccepted
					if mode == "redirect" {
						code = http.StatusFound
					}
					w.WriteHeader(code)
				}
				reply(w, map[string]any{"result": data})
			}))
			got := f.Run("drive", "+task_result", "--scenario", "import", "--ticket", "import1", "--as", "user")
			var out struct {
				Error struct {
					Subtype string `json:"subtype"`
				} `json:"error"`
			}
			if got.ExitCode == 0 || json.Unmarshal([]byte(got.Stderr), &out) != nil || out.Error.Subtype != "invalid_response" || polls.Load() != 1 {
				t.Fatalf("malformed completion accepted: %+v polls=%d", got, polls.Load())
			}
		})
	}
}

func TestImportInterruptedPollLeavesDurableUnknownWithoutReplay(t *testing.T) {
	for _, mode := range []string{"pending", "missing_status", "poll_failure"} {
		t.Run(mode, func(t *testing.T) {
			var creates, polls atomic.Int32
			observed := make(chan struct{})
			var once sync.Once
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/open-apis/drive/v1/medias/upload_all":
					reply(w, map[string]any{"file_token": "media1"})
				case "/open-apis/drive/v1/lark_cli_file_event/report":
					reply(w, map[string]any{})
				case "/open-apis/drive/v1/import_tasks":
					creates.Add(1)
					reply(w, map[string]any{"ticket": "import1"})
				case "/open-apis/drive/v1/import_tasks/import1":
					polls.Add(1)
					if mode == "poll_failure" {
						http.Error(w, "unavailable", 503)
					} else {
						data := map[string]any{"token": "doc1", "type": "docx"}
						if mode == "pending" {
							data["job_status"] = 2
						}
						reply(w, map[string]any{"result": data})
					}
					once.Do(func() { close(observed) })
				default:
					t.Errorf("unexpected %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
				}
			}))
			if err := os.WriteFile(filepath.Join(f.WorkDir, "source.md"), []byte("# Preserve\n"), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			result := make(chan nextfixture.Result, 1)
			go func() {
				result <- f.RunContext(ctx, "drive", "+import", "--file", "source.md", "--type", "docx", "--as", "user")
			}()
			select {
			case <-observed:
				cancel()
			case <-ctx.Done():
				t.Fatal("poll was not reached")
			}
			got := <-result
			if got.ExitCode == 0 {
				t.Fatalf("interrupted import succeeded: %+v", got)
			}
			receipts, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}).List()
			if err != nil {
				t.Fatal(err)
			}
			unknown := 0
			for _, receipt := range receipts {
				if receipt.Outcome == nextattempt.Unknown {
					unknown++
					if receipt.Evidence != "drive_import_task_pending" {
						t.Fatalf("unexpected unknown receipt: %#v", receipt)
					}
				}
			}
			if unknown != 1 || creates.Load() != 1 || polls.Load() != 1 {
				t.Fatalf("lost receipt or replay: unknown=%d creates=%d polls=%d", unknown, creates.Load(), polls.Load())
			}
			// Read-only recovery in a later invocation cannot erase the original unknown.
			recovery := f.Run("drive", "+task_result", "--scenario", "import", "--ticket", "import1", "--as", "user")
			if mode == "pending" && (recovery.ExitCode != 0 || !strings.Contains(recovery.Stdout, `"ready":false`)) {
				t.Fatalf("pending recovery changed: %+v", recovery)
			}
			after, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}).List()
			if err != nil || len(after) != len(receipts) {
				t.Fatalf("recovery changed journal: %v", err)
			}
			for i := range receipts {
				if after[i].Outcome != receipts[i].Outcome {
					t.Fatal("cross-invocation recovery rewrote receipt")
				}
			}
		})
	}
}

func TestImportCompletionEvidenceBindsRequestedResource(t *testing.T) {
	for _, mode := range []string{"complete", "wrong_type", "missing_type", "wrong_source", "contradictory_error", "wrong_target", "matching_target"} {
		for _, identity := range []string{"user", "bot"} {
			t.Run(identity+"/"+mode, func(t *testing.T) {
				var creates, polls, grants atomic.Int32
				docType, fileName := "docx", "source.md"
				target := ""
				if mode == "wrong_target" || mode == "matching_target" {
					docType, fileName, target = "bitable", "source.csv", "base1"
				}
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/open-apis/drive/v1/medias/upload_all":
						reply(w, map[string]any{"file_token": "media1"})
					case "/open-apis/drive/v1/lark_cli_file_event/report":
						reply(w, map[string]any{})
					case "/open-apis/drive/v1/import_tasks":
						creates.Add(1)
						b := body(t, r)
						if b["type"] != docType || b["file_token"] != "media1" || target != "" && b["token"] != target {
							t.Errorf("wrong import binding: %#v", b)
						}
						reply(w, map[string]any{"ticket": "import1"})
					case "/open-apis/drive/v1/import_tasks/import1":
						polls.Add(1)
						data := map[string]any{"type": docType, "token": "doc1", "job_status": 0}
						switch mode {
						case "wrong_type":
							data["type"] = "sheet"
						case "missing_type":
							delete(data, "type")
						case "wrong_source":
							data["file_token"] = "foreign"
						case "contradictory_error":
							data["job_error_msg"] = "failed to import"
						case "wrong_target":
							data["token"] = "otherbase"
						case "matching_target":
							data["token"] = "base1"
						}
						reply(w, map[string]any{"result": data})
					case "/open-apis/drive/v1/permissions/doc1/members", "/open-apis/drive/v1/permissions/base1/members":
						grants.Add(1)
						b := body(t, r)
						if b["member_id"] != "ou_foundation_fixture" || b["member_type"] != "openid" || b["perm"] != "full_access" || r.URL.Query().Get("type") != docType {
							t.Errorf("wrong dependent grant: %#v %s", b, r.URL)
						}
						reply(w, map[string]any{"member": b})
					default:
						t.Errorf("unexpected %s %s", r.Method, r.URL)
						http.Error(w, "unexpected", 500)
					}
				}))
				if err := os.WriteFile(filepath.Join(f.WorkDir, fileName), []byte("Name,Value\nOriginal,001\n"), 0600); err != nil {
					t.Fatal(err)
				}
				args := []string{"drive", "+import", "--file", fileName, "--type", docType, "--as", identity}
				if target != "" {
					args = append(args, "--target-token", target)
				}
				got := f.Run(args...)
				complete := mode == "complete" || mode == "matching_target"
				if complete {
					successful(t, got)
				} else if got.ExitCode != 22 || !strings.Contains(got.Stdout, `"outcome":"unknown"`) {
					t.Fatalf("unbound completion must remain unknown: %+v", got)
				}
				receipts, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}).List()
				if err != nil {
					t.Fatal(err)
				}
				unknown := 0
				for _, receipt := range receipts {
					if receipt.Outcome == nextattempt.Unknown {
						unknown++
					}
				}
				wantUnknown := 0
				if !complete {
					wantUnknown = 1
				}
				wantGrants := int32(0)
				if complete && identity == "bot" {
					wantGrants = 1
				}
				if grants.Load() != wantGrants {
					t.Fatalf("unverified dependent grant: grants=%d want=%d", grants.Load(), wantGrants)
				}
				if creates.Load() != 1 || polls.Load() != 1 || unknown != wantUnknown {
					t.Fatalf("receipt or replay: creates=%d polls=%d unknown=%d", creates.Load(), polls.Load(), unknown)
				}
			})
		}
	}
}

func TestImportRequiresFinalUnredirectedPollBeforeReconciliationOrGrant(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		for _, mode := range []string{"accepted", "unresolved_redirect", "different_ticket_redirect", "cross_origin_redirect"} {
			t.Run(identity+"/"+mode, func(t *testing.T) {
				var creates, polls, redirects, grants atomic.Int32
				final := map[string]any{"result": map[string]any{"type": "docx", "token": "doc1", "job_status": 0}}
				redirectURL, certificatePath := "", ""
				if mode == "cross_origin_redirect" {
					target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						redirects.Add(1)
						reply(w, final)
					}))
					t.Cleanup(target.Close)
					redirectURL = target.URL + "/open-apis/drive/v1/import_tasks/import1"
					certificatePath = filepath.Join(t.TempDir(), "fixture-ca.pem")
					certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: target.Certificate().Raw})
					if err := os.WriteFile(certificatePath, certificate, 0600); err != nil {
						t.Fatal(err)
					}
				}
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					switch r.URL.Path {
					case "/open-apis/drive/v1/medias/upload_all":
						reply(w, map[string]any{"file_token": "media1"})
					case "/open-apis/drive/v1/lark_cli_file_event/report":
						reply(w, map[string]any{})
					case "/open-apis/drive/v1/import_tasks":
						creates.Add(1)
						reply(w, map[string]any{"ticket": "import1"})
					case "/open-apis/drive/v1/import_tasks/import1":
						polls.Add(1)
						w.Header().Set("Content-Type", "application/json")
						code := http.StatusAccepted
						if mode != "accepted" {
							code = http.StatusFound
						}
						if mode == "different_ticket_redirect" {
							w.Header().Set("Location", "/open-apis/drive/v1/import_tasks/foreign")
						}
						if mode == "cross_origin_redirect" {
							w.Header().Set("Location", redirectURL)
						}
						w.WriteHeader(code)
						reply(w, final)
					case "/open-apis/drive/v1/import_tasks/foreign":
						redirects.Add(1)
						reply(w, final)
					case "/open-apis/drive/v1/permissions/doc1/members":
						grants.Add(1)
						reply(w, map[string]any{})
					default:
						t.Errorf("unexpected %s %s", r.Method, r.URL)
						http.Error(w, "unexpected", 500)
					}
				}))
				if certificatePath != "" {
					f.Env["SSL_CERT_FILE"] = certificatePath
				}
				if err := os.WriteFile(filepath.Join(f.WorkDir, "source.md"), []byte("# Source\n"), 0600); err != nil {
					t.Fatal(err)
				}
				got := f.Run("drive", "+import", "--file", "source.md", "--type", "docx", "--as", identity)
				if got.ExitCode != 22 || !strings.Contains(got.Stderr, `"unknown"`) {
					t.Fatalf("non-final poll reconciled: %+v", got)
				}
				receipts, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}).List()
				if err != nil {
					t.Fatal(err)
				}
				pending := 0
				for _, r := range receipts {
					if r.Outcome == nextattempt.Unknown && r.Evidence == "drive_import_task_pending" {
						pending++
					}
				}
				if pending != 1 || creates.Load() != 1 || polls.Load() != 1 || redirects.Load() != 0 || grants.Load() != 0 {
					t.Fatalf("invalid proof caused effects: pending=%d creates=%d polls=%d redirects=%d grants=%d", pending, creates.Load(), polls.Load(), redirects.Load(), grants.Load())
				}
			})
		}
	}
}

func TestImportRecoveryBotSkipsContradictoryPermissionTarget(t *testing.T) {
	for _, mode := range []string{"error", "missing_type", "unknown_type"} {
		t.Run(mode, func(t *testing.T) {
			var grants atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "POST" {
					grants.Add(1)
					reply(w, map[string]any{})
					return
				}
				if r.URL.Path != "/open-apis/drive/v1/import_tasks/import1" {
					t.Errorf("unexpected %s", r.URL)
					http.Error(w, "unexpected", 500)
					return
				}
				data := map[string]any{"token": "doc1", "type": "docx", "job_status": 0}
				switch mode {
				case "error":
					data["job_error_msg"] = "failed"
				case "missing_type":
					delete(data, "type")
				case "unknown_type":
					data["type"] = "unknown"
				}
				reply(w, map[string]any{"result": data})
			}))
			got := f.Run("drive", "+task_result", "--scenario", "import", "--ticket", "import1", "--as", "bot")
			successful(t, got)
			if grants.Load() != 0 || !strings.Contains(got.Stdout, `"status":"skipped"`) {
				t.Fatalf("contradictory recovery target granted: %+v grants=%d", got, grants.Load())
			}
		})
	}
}

func TestImportReceiptSaveFailurePreventsDependentBotGrant(t *testing.T) {
	var grants, creates, polls atomic.Int32
	var f *nextfixture.Fixture
	f = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/medias/upload_all":
			reply(w, map[string]any{"file_token": "media1"})
		case "/open-apis/drive/v1/lark_cli_file_event/report":
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/import_tasks":
			creates.Add(1)
			reply(w, map[string]any{"ticket": "import1"})
		case "/open-apis/drive/v1/import_tasks/import1":
			polls.Add(1)
			root := filepath.Join(f.ConfigDir, "next")
			if err := os.Rename(filepath.Join(root, "receipts"), filepath.Join(root, "held-receipts")); err != nil {
				t.Error(err)
			}
			if err := os.WriteFile(filepath.Join(root, "receipts"), []byte("fixture storage failure"), 0600); err != nil {
				t.Error(err)
			}
			reply(w, map[string]any{"result": map[string]any{"token": "doc1", "type": "docx", "job_status": 0}})
		case "/open-apis/drive/v1/permissions/doc1/members":
			grants.Add(1)
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	if err := os.WriteFile(filepath.Join(f.WorkDir, "source.md"), []byte("# Source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got := f.Run("drive", "+import", "--file", "source.md", "--type", "docx", "--as", "bot")
	if got.ExitCode != 22 || !strings.Contains(got.Stdout, `"ready":true`) || !strings.Contains(got.Stdout, `"persistence_failed":true`) || !strings.Contains(got.Stdout, `"status":"skipped"`) {
		t.Fatalf("save failure hid completion or allowed grant: %+v", got)
	}
	receipts, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "held-receipts")}).List()
	if err != nil {
		t.Fatal(err)
	}
	unknown := 0
	for _, r := range receipts {
		if r.Outcome == nextattempt.Unknown {
			unknown++
		}
	}
	if creates.Load() != 1 || polls.Load() != 1 || grants.Load() != 0 || unknown != 1 {
		t.Fatalf("save failure replay/grant: creates=%d polls=%d grants=%d unknown=%d", creates.Load(), polls.Load(), grants.Load(), unknown)
	}
}
