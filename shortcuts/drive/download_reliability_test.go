// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/internal/vfs/localfileio"
	"github.com/larksuite/cli/shortcuts/common"
)

type collisionFileIO struct{ fileio.FileIO }

func (f collisionFileIO) Save(path string, opts fileio.SaveOptions, body io.Reader) (fileio.SaveResult, error) {
	if err := os.WriteFile(path, []byte("concurrent file"), 0600); err != nil {
		return nil, err
	}
	return f.FileIO.Save(path, opts, body)
}

func (f collisionFileIO) SaveExclusive(path string, opts fileio.SaveOptions, body io.Reader) (fileio.SaveResult, error) {
	if err := os.WriteFile(path, []byte("concurrent file"), 0600); err != nil {
		return nil, err
	}
	return f.FileIO.(fileio.ExclusiveFileIO).SaveExclusive(path, opts, body)
}

type downloadFileIOProvider struct{ fio fileio.FileIO }

func (downloadFileIOProvider) Name() string                                  { return "download-test" }
func (p downloadFileIOProvider) ResolveFileIO(context.Context) fileio.FileIO { return p.fio }

func TestDriveDownloadPoliciesPreserveConcurrentFile(t *testing.T) {
	for _, tc := range []struct {
		name             string
		shortcut         common.Shortcut
		args             []string
		endpoint, target string
		pull             bool
	}{
		{"download", DriveDownload, []string{"+download", "--file-token", "file_fixture", "--output", "report.pdf"}, "/open-apis/drive/v1/files/file_fixture/download", "report.pdf", false},
		{"export", DriveExportDownload, []string{"+export-download", "--file-token", "file_fixture", "--file-name", "report.pdf"}, "/open-apis/drive/v1/export_tasks/file/file_fixture/download", "report.pdf", false},
		{"preview error", DrivePreview, []string{"+preview", "--file-token", "file_fixture", "--type", "source_file", "--output", "report.pdf"}, "/open-apis/drive/v1/medias/file_fixture/preview_download", "report.pdf", false},
		{"preview rename", DrivePreview, []string{"+preview", "--file-token", "file_fixture", "--type", "source_file", "--output", "report.pdf", "--if-exists", "rename"}, "/open-apis/drive/v1/medias/file_fixture/preview_download", "report.pdf", false},
		{"pull skip", DrivePull, []string{"+pull", "--folder-token", "folder_fixture", "--local-dir", "local", "--if-exists", "skip"}, "/open-apis/drive/v1/files/file_fixture/download", "local/report.pdf", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			withDriveWorkingDir(t, t.TempDir())
			f, stdout, _, reg := cmdutil.TestFactory(t, driveTestConfig())
			f.FileIOProvider = downloadFileIOProvider{collisionFileIO{&localfileio.LocalFileIO{}}}
			if tc.name == "download" {
				registerDriveDownloadExportAuth(reg, "file_fixture", true)
			}
			if tc.pull {
				if err := os.Mkdir("local", 0700); err != nil {
					t.Fatal(err)
				}
				reg.Register(&httpmock.Stub{Method: "GET", URL: "folder_token=folder_fixture", Body: map[string]any{
					"code": 0, "data": map[string]any{"files": []any{map[string]any{"token": "file_fixture", "name": "report.pdf", "type": "file"}}, "has_more": false},
				}})
			}
			reg.Register(&httpmock.Stub{Method: "GET", URL: tc.endpoint, RawBody: []byte("download bytes"), ContentType: "application/pdf"})
			err := mountAndRunDrive(t, tc.shortcut, append(tc.args, "--as", "bot"), f, stdout)
			if tc.pull {
				if err != nil || !strings.Contains(stdout.String(), `"skipped": 1`) || !strings.Contains(stdout.String(), `"downloaded": 0`) {
					t.Fatalf("concurrent target must be skipped: error=%v, stdout=%s", err, stdout)
				}
			} else {
				problem, ok := errs.ProblemOf(err)
				if !ok || problem.Category != errs.CategoryValidation || !errors.Is(err, fs.ErrExist) || stdout.Len() != 0 {
					t.Fatalf("collision must fail with its typed cause: problem=%#v, error=%v, stdout=%s", problem, err, stdout)
				}
			}
			data, err := os.ReadFile(tc.target)
			if err != nil || string(data) != "concurrent file" {
				t.Fatalf("concurrent file changed: %q, %v", data, err)
			}
		})
	}
}

func TestDriveDownloadsDoNotReplaceFileWithAPIError(t *testing.T) {
	for _, tc := range []struct {
		name     string
		shortcut common.Shortcut
		args     []string
		endpoint string
	}{
		{"download", DriveDownload, []string{"+download", "--file-token", "file_fixture", "--output", "report.pdf", "--overwrite"}, "/open-apis/drive/v1/files/file_fixture/download"},
		{"export", DriveExportDownload, []string{"+export-download", "--file-token", "file_fixture", "--file-name", "report.pdf", "--overwrite"}, "/open-apis/drive/v1/export_tasks/file/file_fixture/download"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			withDriveWorkingDir(t, t.TempDir())
			if err := os.WriteFile("report.pdf", []byte("good file"), 0600); err != nil {
				t.Fatal(err)
			}
			f, stdout, _, reg := cmdutil.TestFactory(t, driveTestConfig())
			if tc.name == "download" {
				registerDriveDownloadExportAuth(reg, "file_fixture", true)
			}
			reg.Register(&httpmock.Stub{Method: "GET", URL: tc.endpoint, Body: map[string]any{"code": 99991679, "msg": "scope rejected"}, Headers: http.Header{"X-Tt-Logid": []string{"fixture_log"}}})
			err := mountAndRunDrive(t, tc.shortcut, append(tc.args, "--as", "bot"), f, stdout)
			problem, ok := errs.ProblemOf(err)
			if !ok || problem.Category != errs.CategoryAuthorization || problem.Code != 99991679 || problem.LogID != "fixture_log" || stdout.Len() != 0 {
				t.Fatalf("wrong API failure: problem=%#v, error=%v, stdout=%s", problem, err, stdout)
			}
			data, err := os.ReadFile("report.pdf")
			if err != nil || string(data) != "good file" {
				t.Fatalf("good file replaced: %q, %v", data, err)
			}
		})
	}
}
