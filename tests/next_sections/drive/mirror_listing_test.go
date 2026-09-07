// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestDeletingMirrorRequiresExplicitListingBeforeChanges(t *testing.T) {
	for _, command := range []string{"+pull", "+push"} {
		for _, mode := range []string{"missing_both", "missing_files", "null_files", "missing_pagination", "null_pagination", "second_page", "child_folder"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				var reads, changes atomic.Int32
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" || r.URL.Path != "/open-apis/drive/v1/files" {
						changes.Add(1)
						http.Error(w, "no transfer or mutation expected", 500)
						return
					}
					n := reads.Add(1)
					data := map[string]any{"files": []any{}, "has_more": false}
					switch mode {
					case "missing_both":
						data = map[string]any{}
					case "missing_files":
						delete(data, "files")
					case "null_files":
						data["files"] = nil
					case "missing_pagination":
						delete(data, "has_more")
					case "null_pagination":
						data["has_more"] = nil
					case "second_page":
						if n == 1 {
							data["files"] = []any{map[string]any{"token": "remote1", "name": "remote.bin", "type": "file"}}
							data["has_more"] = true
							data["next_page_token"] = "next"
						} else {
							data = map[string]any{}
						}
					case "child_folder":
						if n == 1 {
							data["files"] = []any{map[string]any{"token": "child1", "name": "child", "type": "folder"}}
						} else {
							data = map[string]any{}
						}
					}
					reply(w, data)
				}))
				local := filepath.Join(f.WorkDir, "local.bin")
				original := []byte("keep local bytes\x00\xff\n")
				if err := os.WriteFile(local, original, 0600); err != nil {
					t.Fatal(err)
				}
				flag := "--delete-local"
				if command == "+push" {
					flag = "--delete-remote"
				}
				got := f.Run("drive", command, "--local-dir", ".", "--folder-token", "root", flag, "--yes", "--as", "user")
				if got.ExitCode == 0 {
					t.Fatalf("incomplete mirror succeeded: %s", got.Stdout)
				}
				var failure struct {
					Error struct {
						Subtype string `json:"subtype"`
					} `json:"error"`
				}
				if err := json.Unmarshal([]byte(got.Stderr), &failure); err != nil {
					t.Fatalf("error envelope: %v %s", err, got.Stderr)
				}
				want := "failed_precondition"
				if mode == "null_pagination" {
					want = "invalid_response"
				}
				if failure.Error.Subtype != want {
					t.Fatalf("want %s, got %s", want, got.Stderr)
				}
				saved, err := os.ReadFile(local)
				if err != nil || !bytes.Equal(saved, original) {
					t.Fatalf("local bytes changed: %v", err)
				}
				wantReads := int32(1)
				if mode == "second_page" || mode == "child_folder" {
					wantReads = 2
				}
				if changes.Load() != 0 || reads.Load() != wantReads {
					t.Fatalf("reads=%d changes=%d", reads.Load(), changes.Load())
				}
			})
		}
	}
}

func TestOptionalEmptyListingCompatibilityAndExplicitMirrorDeletion(t *testing.T) {
	for _, mode := range []string{"missing", "null_files", "explicit"} {
		t.Run(mode, func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/open-apis/drive/v1/files" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.Error(w, "unexpected", 500)
					return
				}
				data := map[string]any{}
				if mode == "null_files" {
					data["files"] = nil
				}
				if mode == "explicit" {
					data = map[string]any{"files": []any{}, "has_more": false}
				}
				reply(w, data)
			}))
			local := filepath.Join(f.WorkDir, "local.bin")
			original := []byte("preserve unless explicitly mirrored\n")
			if err := os.WriteFile(local, original, 0600); err != nil {
				t.Fatal(err)
			}
			successful(t, f.Run("drive", "+status", "--local-dir", ".", "--folder-token", "root", "--as", "user"))
			args := []string{"drive", "+pull", "--local-dir", ".", "--folder-token", "root", "--as", "user"}
			if mode == "explicit" {
				args = append(args, "--delete-local", "--yes")
			}
			successful(t, f.Run(args...))
			saved, err := os.ReadFile(local)
			if mode == "explicit" {
				if !os.IsNotExist(err) {
					t.Fatalf("explicit empty mirror did not delete file: %v", err)
				}
			} else if err != nil || !bytes.Equal(saved, original) {
				t.Fatalf("optional empty read changed local bytes: %v", err)
			}
		})
	}
}
