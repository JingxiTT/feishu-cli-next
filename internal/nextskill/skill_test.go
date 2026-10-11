// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextskill

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/vfs"
)

func TestEntryRoutesUpstreamWorkflowsWithoutRepeatedDiscovery(t *testing.T) {
	content, err := os.ReadFile("../../agent-skills/feishu-cli-next/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, decision := range []string{
		"apps +export --app-id", "--meta-token", "mail +thread-modify --thread-ids", "mail +thread-trash --thread-ids <id,id> --as user --dry-run",
		"success_thread_ids", "failed_thread_ids", "authorized whole-thread deletion", "--concise", "--no-reactions",
		"Creation waits for async completion", "inspect the receipt and destination before retrying",
		"--print-schema --flag-name", "Reuse guidance already read for this version",
	} {
		if !strings.Contains(string(content), decision) {
			t.Errorf("entry guidance missing %q", decision)
		}
	}
}

func fixture(body string) fstest.MapFS {
	return fstest.MapFS{contentPath: &fstest.MapFile{Data: []byte(body)}}
}

func requireConflict(t *testing.T, err error) {
	t.Helper()
	p, ok := errs.ProblemOf(err)
	if !ok || p.Category != errs.CategoryValidation || p.Subtype != errs.SubtypeFailedPrecondition {
		t.Fatalf("expected typed conflict, got %v", err)
	}
}

func TestLifecyclePreservesOtherFilesAndDetectsEdits(t *testing.T) {
	project := t.TempDir()
	first, err := Apply(fixture("version one\n"), "install", project)
	if err != nil || !first.Changed {
		t.Fatalf("install: %+v %v", first, err)
	}
	data, err := os.ReadFile(first.Path)
	if err != nil || !isManaged(data) || !bytes.HasPrefix(data, []byte("version one\n")) {
		t.Fatalf("installed bytes: %q %v", data, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(first.Path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("permissions: %v %v", info, err)
		}
	}
	extra := filepath.Join(filepath.Dir(first.Path), "notes.md")
	if err := os.WriteFile(extra, []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	same, err := Apply(fixture("version one\n"), "install", project)
	if err != nil || same.Changed {
		t.Fatalf("idempotent install: %+v %v", same, err)
	}
	_, err = Apply(fixture("version two\n"), "install", project)
	requireConflict(t, err)
	updated, err := Apply(fixture("version two\n"), "update", project)
	if err != nil || !updated.Changed {
		t.Fatalf("update: %+v %v", updated, err)
	}
	data, _ = os.ReadFile(first.Path)
	if !bytes.HasPrefix(data, []byte("version two\n")) || !isManaged(data) {
		t.Fatalf("update content: %q", data)
	}
	edited := bytes.Replace(data, []byte("version two"), []byte("local edits"), 1)
	if err := os.WriteFile(first.Path, edited, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"install", "update", "uninstall"} {
		_, err := Apply(fixture("version three\n"), action, project)
		requireConflict(t, err)
		after, _ := os.ReadFile(first.Path)
		if !bytes.Equal(after, edited) {
			t.Fatalf("%s overwrote edits", action)
		}
	}
	if err := os.WriteFile(first.Path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	removed, err := Apply(nil, "uninstall", project)
	if err != nil || !removed.Changed {
		t.Fatalf("uninstall: %+v %v", removed, err)
	}
	if _, err := os.Stat(first.Path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("skill remains: %v", err)
	}
	other, err := os.ReadFile(extra)
	if err != nil || string(other) != "unrelated" {
		t.Fatalf("unrelated file changed: %q %v", other, err)
	}
	absent, err := Apply(nil, "uninstall", project)
	if err != nil || absent.Changed {
		t.Fatalf("repeat uninstall: %+v %v", absent, err)
	}
}

func TestAbsentLifecycleDoesNotCreateFiles(t *testing.T) {
	project := t.TempDir()
	absent, err := Apply(nil, "uninstall", project)
	if err != nil || absent.Changed {
		t.Fatalf("uninstall absent: %+v %v", absent, err)
	}
	_, err = Apply(fixture("x"), "update", project)
	requireConflict(t, err)
	entries, err := os.ReadDir(project)
	if err != nil || len(entries) != 0 {
		t.Fatalf("created files: %v %v", entries, err)
	}
}

func TestCollisionAndSymlinks(t *testing.T) {
	for _, kind := range []string{"file", "directory", "symlink", "ancestor-symlink", "lock"} {
		t.Run(kind, func(t *testing.T) {
			project := t.TempDir()
			dir := filepath.Join(project, ".agents", "skills", Name)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(dir, "SKILL.md")
			switch kind {
			case "file":
				if err := os.WriteFile(target, []byte("user skill"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(target, 0o700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), target); err != nil {
					t.Skip(err)
				}
			case "ancestor-symlink":
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
				outside := t.TempDir()
				if err := os.Symlink(outside, dir); err != nil {
					t.Skip(err)
				}
			case "lock":
				if err := os.WriteFile(filepath.Join(dir, ".install.lock"), []byte("existing"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			_, err := Apply(fixture("new"), "install", project)
			requireConflict(t, err)
		})
	}
}

type failedRenameFS struct {
	vfs.FS
	failure error
}

func (f failedRenameFS) Rename(oldpath, newpath string) error { return f.failure }

func TestFailedAtomicUpdatePreservesOldVersion(t *testing.T) {
	project := t.TempDir()
	initial, err := Apply(fixture("old"), "install", project)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(initial.Path)
	original := vfs.DefaultFS
	cause := errors.New("rename unavailable")
	vfs.DefaultFS = failedRenameFS{FS: original, failure: cause}
	t.Cleanup(func() { vfs.DefaultFS = original })
	_, err = Apply(fixture("new"), "update", project)
	if !errors.Is(err, cause) {
		t.Fatalf("missing cause: %v", err)
	}
	p, ok := errs.ProblemOf(err)
	if !ok || p.Subtype != errs.SubtypeFileIO {
		t.Fatalf("not file I/O: %v", err)
	}
	after, _ := os.ReadFile(initial.Path)
	if !bytes.Equal(after, before) {
		t.Fatal("failed update changed installed skill")
	}
	entries, _ := os.ReadDir(filepath.Dir(initial.Path))
	if len(entries) != 1 || entries[0].Name() != "SKILL.md" {
		t.Fatalf("temporary files leaked: %v", entries)
	}
}

func TestReadMissingContent(t *testing.T) {
	for _, content := range []fs.FS{nil, fstest.MapFS{}} {
		_, err := Read(content)
		p, ok := errs.ProblemOf(err)
		if !ok || p.Subtype != errs.SubtypeFileIO {
			t.Fatalf("not typed missing content: %v", err)
		}
	}
}
