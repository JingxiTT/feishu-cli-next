// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestPullRejectsRemotePathTraversal(t *testing.T) {
	names := []string{"../sibling.txt", ".", "..", "/sibling.txt", `..\sibling.txt`, "sub/file.txt", "bad\x00name"}
	if runtime.GOOS == "windows" {
		names = append(names, "C:sibling.txt", "NUL")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			f, out, _, reg := cmdutil.TestFactory(t, driveTestConfig())
			withDriveWorkingDir(t, t.TempDir())
			if err := os.Mkdir("local", 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("sibling.txt", []byte("original outside mirror"), 0600); err != nil {
				t.Fatal(err)
			}
			reg.Register(&httpmock.Stub{Method: "GET", URL: "folder_token=folder_root", Body: map[string]any{"code": 0, "data": map[string]any{"has_more": false, "files": []any{map[string]any{"token": "tok_a", "name": name, "type": "file"}}}}})
			download := &httpmock.Stub{Optional: true, Method: "GET", URL: "/files/tok_a/download", Body: []byte("remote replacement")}
			reg.Register(download)
			err := mountAndRunDrive(t, DrivePull, []string{"+pull", "--local-dir", "local", "--folder-token", "folder_root", "--as", "bot"}, f, out)
			var typed *errs.InternalError
			if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("unsafe name accepted: %v", err)
			}
			if len(download.CapturedBodies) != 0 {
				t.Fatal("downloaded before validating listing")
			}
			saved, err := os.ReadFile("sibling.txt")
			if err != nil || string(saved) != "original outside mirror" {
				t.Fatalf("sibling changed: %q %v", saved, err)
			}
		})
	}
}

func TestPullRejectsSymlinkOutsideMirror(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	f, out, _, reg := cmdutil.TestFactory(t, driveTestConfig())
	withDriveWorkingDir(t, t.TempDir())
	if err := os.Mkdir("local", 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sibling.txt", []byte("original outside mirror"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../sibling.txt", "local/file.txt"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	reg.Register(&httpmock.Stub{Method: "GET", URL: "folder_token=folder_root", Body: map[string]any{"code": 0, "data": map[string]any{"has_more": false, "files": []any{map[string]any{"token": "tok_a", "name": "file.txt", "type": "file"}}}}})
	download := &httpmock.Stub{Optional: true, Method: "GET", URL: "/files/tok_a/download", Body: []byte("remote replacement")}
	reg.Register(download)
	err := mountAndRunDrive(t, DrivePull, []string{"+pull", "--local-dir", "local", "--folder-token", "folder_root", "--as", "bot"}, f, out)
	if err == nil || len(download.CapturedBodies) != 0 {
		t.Fatalf("must reject escaped target before download: %v %s", err, out.String())
	}
	saved, err := os.ReadFile("sibling.txt")
	if err != nil || string(saved) != "original outside mirror" {
		t.Fatalf("sibling changed: %q %v", saved, err)
	}
}
