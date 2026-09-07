// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/nextattempt"
	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestWorkbookImportNativeCompletionAndRecovery(t *testing.T) {
	for _, mode := range []string{"complete", "pending_complete", "wrong_type", "missing_status", "failed", "pending_interrupt"} {
		t.Run(mode, func(t *testing.T) {
			var uploads, creates, polls atomic.Int32
			var pendingObserved atomic.Bool
			observed := make(chan struct{}, 1)
			var f *nextfixture.Fixture
			f = nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				respond := func(data any) { require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})) }
				switch r.URL.Path {
				case "/open-apis/drive/v1/medias/upload_all":
					require.Equal(t, "POST", r.Method)
					uploads.Add(1)
					respond(map[string]any{"file_token": "mediaSheetImport"})
				case "/open-apis/drive/v1/lark_cli_file_event/report":
					respond(map[string]any{})
				case "/open-apis/drive/v1/import_tasks":
					require.Equal(t, "POST", r.Method)
					creates.Add(1)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "sheet", body["type"], "Sheets wrapper must pin the native target type")
					require.Equal(t, "csv", body["file_extension"])
					require.Equal(t, "mediaSheetImport", body["file_token"])
					respond(map[string]any{"ticket": "ticketSheetImport"})
				case "/open-apis/drive/v1/import_tasks/ticketSheetImport":
					require.Equal(t, "GET", r.Method)
					poll := polls.Add(1)
					data := map[string]any{"type": "sheet", "token": "shtImportedFixture", "job_status": 0}
					if mode == "pending_interrupt" || mode == "pending_complete" && poll == 1 {
						data["job_status"] = 2
						receipts, err := (nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}).List()
						require.NoError(t, err)
						for _, receipt := range receipts {
							if receipt.Evidence == "drive_import_task_pending" && receipt.Outcome == nextattempt.Unknown {
								pendingObserved.Store(true)
							}
						}
					}
					switch mode {
					case "wrong_type":
						data["type"] = "docx"
					case "missing_status":
						delete(data, "job_status")
					case "failed":
						data["job_status"], data["job_error_msg"] = 3, "synthetic sheet conversion failed"
					}
					respond(map[string]any{"result": data})
					select {
					case observed <- struct{}{}:
					default:
					}
				default:
					t.Errorf("unexpected workbook-import request %s %s", r.Method, r.URL)
					http.Error(w, "unexpected fixture request", 500)
				}
			}))
			source := []byte("Name,Code\n你好,001\n")
			require.NoError(t, os.WriteFile(filepath.Join(f.WorkDir, "source.csv"), source, 0600))
			args := []string{"sheets", "+workbook-import", "--file", "source.csv", "--as", "user"}
			var result nextfixture.Result
			if mode == "pending_interrupt" {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				completed := make(chan nextfixture.Result, 1)
				go func() { completed <- f.RunContext(ctx, args...) }()
				select {
				case <-observed:
					cancel()
				case <-ctx.Done():
					t.Fatal("import did not reach its pending poll")
				}
				result = <-completed
				require.NotZero(t, result.ExitCode)
			} else {
				result = f.Run(args...)
			}
			complete := mode == "complete" || mode == "pending_complete"
			if complete {
				out := success(t, result)
				require.Equal(t, "confirmed_applied", out["mutation"].(map[string]any)["outcome"])
				data := out["data"].(map[string]any)
				require.Equal(t, true, data["ready"])
				require.Equal(t, "sheet", data["type"])
				require.Equal(t, "shtImportedFixture", data["token"])
				require.Contains(t, data["url"], "/sheets/shtImportedFixture")
			} else if mode != "pending_interrupt" {
				require.Equal(t, 22, result.ExitCode, result.Stdout+result.Stderr)
				require.Contains(t, result.Stderr, `"outcome":"unknown"`)
				if mode == "missing_status" {
					require.Contains(t, result.Stderr, "invalid_response")
				}
			}
			wantPolls := int32(1)
			if mode == "pending_complete" {
				wantPolls = 2
			}
			require.Equal(t, int32(1), uploads.Load())
			require.Equal(t, int32(1), creates.Load(), "polling must not create another workbook")
			require.Equal(t, wantPolls, polls.Load())
			if strings.HasPrefix(mode, "pending_") {
				require.True(t, pendingObserved.Load(), "the create acknowledgement must remain unknown while pending")
			}
			store := nextattempt.Store{Dir: filepath.Join(f.ConfigDir, "next", "receipts")}
			receipts, err := store.List()
			require.NoError(t, err)
			imports := 0
			for _, receipt := range receipts {
				if strings.HasPrefix(receipt.Evidence, "drive_import_task_") {
					imports++
					if complete {
						require.Equal(t, nextattempt.Applied, receipt.Outcome)
						require.Equal(t, "drive_import_task_completed", receipt.Evidence)
					} else {
						require.Equal(t, nextattempt.Unknown, receipt.Outcome)
						require.Equal(t, "drive_import_task_pending", receipt.Evidence)
					}
				}
			}
			require.Equal(t, 1, imports)
			if mode == "pending_interrupt" {
				recovered := success(t, f.Run("drive", "+task_result", "--scenario", "import", "--ticket", "ticketSheetImport", "--as", "user"))
				require.Equal(t, false, recovered["data"].(map[string]any)["ready"])
				after, err := store.List()
				require.NoError(t, err)
				require.Equal(t, receipts, after, "later read-only recovery must not rewrite the original receipt")
				require.Equal(t, int32(1), creates.Load())
			}
			unchanged, err := os.ReadFile(filepath.Join(f.WorkDir, "source.csv"))
			require.NoError(t, err)
			require.Equal(t, source, unchanged)
		})
	}
}
