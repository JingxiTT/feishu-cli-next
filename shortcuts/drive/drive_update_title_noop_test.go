// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
)

func TestDriveUpdateTitleUnchangedGuardSkipsMutation(t *testing.T) {
	for _, title := range []string{"季度.xlsx", "季度"} {
		t.Run(title, func(t *testing.T) {
			f, stdout, _, reg := cmdutil.TestFactory(t, driveTestConfig())
			reg.Register(driveUpdateTitleMetaStub("fileRenameTarget", "季度.xlsx"))
			err := mountAndRunDrive(t, DriveUpdateTitle, []string{"+update-title", "--token", "fileRenameTarget", "--type", "file", "--title", title, "--as", "user"}, f, stdout)
			if err != nil {
				t.Fatalf("unchanged title: %v", err)
			}
			data := mustMapValue(t, decodeJSONMap(t, stdout.String())["data"], "data")
			if data["updated"] != false || data["unchanged"] != true || data["title"] != "季度.xlsx" {
				t.Fatalf("wrong no-op: %#v", data)
			}
			reg.Verify(t) // No PATCH handler: any mutation fails the test.
		})
	}
}
