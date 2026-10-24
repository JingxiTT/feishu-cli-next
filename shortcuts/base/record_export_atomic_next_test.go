// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

type atomicExportProvider struct{ fio fileio.FileIO }

func (atomicExportProvider) Name() string                                  { return "atomic-export-test" }
func (p atomicExportProvider) ResolveFileIO(context.Context) fileio.FileIO { return p.fio }

type atomicExportFileIO struct {
	fileio.FileIO
	stats, saves, exclusiveSaves []string
	beforeCommit                 func(string) error
}

func (f *atomicExportFileIO) Stat(path string) (fileio.FileInfo, error) {
	f.stats = append(f.stats, path)
	return f.FileIO.Stat(path)
}

func (f *atomicExportFileIO) Save(path string, opts fileio.SaveOptions, body io.Reader) (fileio.SaveResult, error) {
	f.saves = append(f.saves, path)
	return f.FileIO.Save(path, opts, body)
}

func (f *atomicExportFileIO) SaveExclusive(path string, opts fileio.SaveOptions, body io.Reader) (fileio.SaveResult, error) {
	f.exclusiveSaves = append(f.exclusiveSaves, path)
	if f.beforeCommit != nil {
		if err := f.beforeCommit(path); err != nil {
			return nil, err
		}
	}
	return f.FileIO.(fileio.ExclusiveFileIO).SaveExclusive(path, opts, body)
}

func atomicExportPath(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, "records.ndjson")
}

func runAtomicExport(t *testing.T, fio fileio.FileIO, path string, overwrite bool) (error, string) {
	t.Helper()
	withBaseWorkingDir(t, filepath.Dir(path))
	factory, stdout, registry := newExecuteFactory(t)
	factory.FileIOProvider = atomicExportProvider{fio}
	registry.Register(&httpmock.Stub{Method: "GET", URL: "limit=1&offset=0",
		Body: map[string]any{"code": 0, "data": recordMatrixPage(0, 1, false, "fld_name")},
	})
	args := []string{"+record-list", "--base-token", "app_x", "--table-id", "tbl_x", "--limit", "1", "--output", path}
	if overwrite {
		args = append(args, "--overwrite")
	}
	err := runShortcut(t, BaseRecordList, args, factory, stdout)
	return err, stdout.String()
}

func TestRecordExportAtomicRecordCollisionPreservesConcurrentWriter(t *testing.T) {
	path := atomicExportPath(t)
	manifestPath := strings.TrimSuffix(path, ".ndjson") + ".manifest.json"
	fio := &atomicExportFileIO{FileIO: &localfileio.LocalFileIO{}}
	fio.beforeCommit = func(commitPath string) error {
		if !reflect.DeepEqual(fio.stats, []string{path, manifestPath}) {
			t.Fatalf("both destinations must be preflighted before publication: %#v", fio.stats)
		}
		// A competing writer wins after both Stat calls but before the atomic
		// commit. Save would replace these bytes; SaveExclusive must refuse.
		written := make(chan error, 1)
		go func() { written <- os.WriteFile(commitPath, []byte("concurrent records\n"), 0o600) }()
		return <-written
	}
	err, stdout := runAtomicExport(t, fio, path, false)
	if err == nil || !errors.Is(err, fs.ErrExist) {
		t.Fatalf("error = %v, want existing-target cause", err)
	}
	if _, ok := errs.ProblemOf(err); !ok {
		t.Fatalf("error is not typed: %v", err)
	}
	content, readErr := os.ReadFile(path)
	if readErr != nil || string(content) != "concurrent records\n" {
		t.Fatalf("concurrent file changed: %q, %v", content, readErr)
	}
	if _, statErr := os.Stat(manifestPath); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("manifest unexpectedly published: %v", statErr)
	}
	if len(fio.saves) != 0 || len(fio.exclusiveSaves) != 1 || stdout != "" {
		t.Fatalf("calls/output = %#v / %#v / %q", fio.saves, fio.exclusiveSaves, stdout)
	}
}

func TestRecordExportAtomicManifestCollisionReportsPartialPublication(t *testing.T) {
	path := atomicExportPath(t)
	manifestPath := strings.TrimSuffix(path, ".ndjson") + ".manifest.json"
	fio := &atomicExportFileIO{FileIO: &localfileio.LocalFileIO{}}
	fio.beforeCommit = func(commitPath string) error {
		if !reflect.DeepEqual(fio.stats, []string{path, manifestPath}) {
			t.Fatalf("preflight = %#v", fio.stats)
		}
		if commitPath == manifestPath {
			return os.WriteFile(commitPath, []byte("concurrent manifest\n"), 0o600)
		}
		return nil
	}
	err, stdout := runAtomicExport(t, fio, path, false)
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Subtype != errs.SubtypeFileIO || !errors.Is(err, fs.ErrExist) {
		t.Fatalf("problem/error = %#v / %v", problem, err)
	}
	if !strings.Contains(err.Error(), "partially published locally") || !strings.Contains(err.Error(), path) || strings.Contains(problem.Hint, "overwrite") || strings.Contains(problem.Hint, "retry") {
		t.Fatalf("missing truthful recovery evidence: %#v", problem)
	}
	records, readErr := os.ReadFile(path)
	if readErr != nil || !strings.Contains(string(records), `"record_id":"rec_0000"`) {
		t.Fatalf("records = %q, %v", records, readErr)
	}
	manifest, readErr := os.ReadFile(manifestPath)
	if readErr != nil || string(manifest) != "concurrent manifest\n" {
		t.Fatalf("concurrent manifest changed: %q, %v", manifest, readErr)
	}
	if len(fio.saves) != 0 || len(fio.exclusiveSaves) != 2 || stdout != "" {
		t.Fatalf("calls/output = %#v / %#v / %q", fio.saves, fio.exclusiveSaves, stdout)
	}
}

func TestRecordExportAtomicPreflightsExistingManifestBeforeRecords(t *testing.T) {
	path := atomicExportPath(t)
	manifestPath := strings.TrimSuffix(path, ".ndjson") + ".manifest.json"
	if err := os.WriteFile(manifestPath, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fio := &atomicExportFileIO{FileIO: &localfileio.LocalFileIO{}}
	err, _ := runAtomicExport(t, fio, path, false)
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Subtype != errs.SubtypeFailedPrecondition {
		t.Fatalf("error = %v", err)
	}
	if len(fio.saves)+len(fio.exclusiveSaves) != 0 {
		t.Fatalf("published before complete preflight: %#v", fio)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("records unexpectedly published: %v", statErr)
	}
}

func TestRecordExportAtomicRequiresProviderCapabilityUnlessOverwrite(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(map[bool]string{false: "exclusive required", true: "explicit overwrite"}[overwrite], func(t *testing.T) {
			path := atomicExportPath(t)
			fio := &atomicExportFileIO{FileIO: &localfileio.LocalFileIO{}}
			// Expose only the base interface to model an older host provider.
			legacy := struct{ fileio.FileIO }{fio}
			err, _ := runAtomicExport(t, legacy, path, overwrite)
			if overwrite {
				if err != nil || len(fio.saves) != 2 || len(fio.exclusiveSaves) != 0 {
					t.Fatalf("overwrite error/calls = %v / %#v", err, fio)
				}
			} else {
				problem, ok := errs.ProblemOf(err)
				if !ok || problem.Subtype != errs.SubtypeFailedPrecondition || len(fio.saves)+len(fio.exclusiveSaves) != 0 {
					t.Fatalf("error/calls = %v / %#v", err, fio)
				}
			}
		})
	}
}
