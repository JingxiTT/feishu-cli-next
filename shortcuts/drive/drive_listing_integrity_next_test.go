// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"net/http"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestDriveMirrorRejectsIncompleteListing(t *testing.T) {
	for _, scenario := range []string{"missing_cursor", "repeated_cursor", "folder_cycle", "malformed_entry", "malformed_collection", "malformed_pagination"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			withDriveWorkingDir(t, dir)
			f, stdout, _, reg := cmdutil.TestFactory(t, driveTestConfig())
			calls := 0
			data := map[string]interface{}{"files": []interface{}{}, "has_more": true}
			if scenario == "repeated_cursor" {
				data["next_page_token"] = "same"
			}
			if scenario == "malformed_pagination" {
				data = map[string]interface{}{"files": []interface{}{}, "has_more": "true"}
			}
			if scenario == "malformed_collection" {
				data = map[string]interface{}{"files": "invalid", "has_more": false}
			}
			if scenario == "malformed_entry" {
				data = map[string]interface{}{"files": []interface{}{map[string]interface{}{"token": "broken"}}, "has_more": false}
			}
			if scenario == "folder_cycle" {
				data = map[string]interface{}{"files": []interface{}{map[string]interface{}{"token": "root", "name": "loop", "type": "folder"}}, "has_more": false}
			}
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/drive/v1/files", Body: map[string]interface{}{"code": 0, "data": data}, OnMatch: func(*http.Request) { calls++ }})
			if scenario == "repeated_cursor" {
				reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/drive/v1/files", Body: map[string]interface{}{"code": 0, "data": data}, OnMatch: func(*http.Request) { calls++ }})
			}
			// Overwrite plus deletion makes silently accepting a partial list destructive.
			err := mountAndRunDrive(t, DrivePush, []string{"+push", "--local-dir", dir, "--folder-token", "root", "--if-exists", "overwrite", "--delete-remote", "--yes", "--as", "bot"}, f, stdout)
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("want typed invalid_response, got %v", err)
			}
			if calls < 1 || calls > 2 {
				t.Fatalf("unbounded listing: %d calls", calls)
			}
			if stdout.Len() != 0 {
				t.Fatalf("incomplete mirror must not report success: %s", stdout.String())
			}
		})
	}
}
