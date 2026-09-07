// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package nextskill owns the CLI-managed, host-local project skill lifecycle.
package nextskill

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/vfs"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

const Name = "feishu-cli-next"
const contentPath = Name + "/SKILL.md"
const marker = "\n<!-- feishu-cli-next managed v1 sha256:"

// Result describes a local lifecycle effect, never an authentication change.
type Result struct {
	OK      bool   `json:"ok"`
	Action  string `json:"action"`
	Path    string `json:"path"`
	Changed bool   `json:"changed"`
}

// Read returns the single entry skill from the distribution's skill root.
func Read(content fs.FS) ([]byte, error) {
	if content == nil {
		return nil, errs.NewInternalError(errs.SubtypeFileIO, "entry skill is not embedded in this build")
	}
	data, err := fs.ReadFile(content, contentPath)
	if err != nil {
		return nil, fileError("read embedded entry skill", err)
	}
	return data, nil
}

// Apply manages only SKILL.md under an existing project. This is deliberately
// host-local CLI configuration, not a portable user-document FileIO operation.
func Apply(content fs.FS, action, project string) (*Result, error) {
	if action != "install" && action != "update" && action != "uninstall" {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "unknown skill action %q", action)
	}
	dir, err := projectDir(project)
	if err != nil {
		return nil, err
	}
	target := filepath.Join(dir, "SKILL.md")
	result := &Result{OK: true, Action: action, Path: target}
	var desired []byte
	if action != "uninstall" {
		data, err := Read(content)
		if err != nil {
			return nil, err
		}
		desired = managed(data)
	}
	if action != "install" {
		if _, err := vfs.Lstat(dir); errors.Is(err, fs.ErrNotExist) {
			if action == "uninstall" {
				return result, nil
			}
			return nil, conflict("skill is not installed; use guide install", target)
		} else if err != nil {
			return nil, fileError("inspect project skill directory", err)
		}
	}
	if err := vfs.MkdirAll(dir, 0o700); err != nil {
		return nil, fileError("create project skill directory", err)
	}
	// Exclusive creation serializes CLI lifecycle operations for this skill.
	// A crash leaves an explicit lock rather than silently stealing ownership.
	lockPath := filepath.Join(dir, ".install.lock")
	lock, err := vfs.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil, conflict("skill lifecycle is locked", lockPath)
		}
		return nil, fileError("lock project skill", err)
	}
	if err := lock.Close(); err != nil {
		_ = vfs.Remove(lockPath)
		return nil, fileError("close project skill lock", err)
	}
	defer func() { _ = vfs.Remove(lockPath) }()
	current, exists, err := readTarget(target)
	if err != nil {
		return nil, err
	}
	if exists && !isManaged(current) {
		return nil, conflict("skill exists without an intact installation checksum", target)
	}
	switch action {
	case "install":
		if exists {
			if bytes.Equal(current, desired) {
				return result, nil
			}
			return nil, conflict("a different skill version is installed; use guide update", target)
		}
		if _, err := localfileio.ExclusiveWriteFromReader(target, bytes.NewReader(desired), 0o600); err != nil {
			return nil, fileError("install entry skill", err)
		}
	case "update":
		if !exists {
			return nil, conflict("skill is not installed; use guide install", target)
		}
		if bytes.Equal(current, desired) {
			return result, nil
		}
		if err := localfileio.AtomicWrite(target, desired, 0o600); err != nil {
			return nil, fileError("update entry skill", err)
		}
	case "uninstall":
		if !exists {
			return result, nil
		}
		if err := vfs.Remove(target); err != nil {
			return nil, fileError("uninstall entry skill", err)
		}
	}
	result.Changed = true
	return result, nil
}

func managed(content []byte) []byte {
	return []byte(fmt.Sprintf("%s%s%x -->\n", content, marker, sha256.Sum256(content)))
}

func isManaged(content []byte) bool {
	i := bytes.LastIndex(content, []byte(marker))
	return i >= 0 && bytes.Equal(content, managed(content[:i]))
}

func readTarget(path string) ([]byte, bool, error) {
	info, err := vfs.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fileError("inspect installed skill", err)
	}
	if !info.Mode().IsRegular() {
		return nil, false, conflict("skill target is not a regular file", path)
	}
	data, err := vfs.ReadFile(path)
	if err != nil {
		return nil, false, fileError("read installed skill", err)
	}
	return data, true, nil
}

func projectDir(project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "project directory must not be empty").WithParam("--project")
	}
	if _, err := localfileio.LocalInputPath(project); err != nil {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid project directory").WithParam("--project").WithCause(err)
	}
	if !filepath.IsAbs(project) {
		cwd, err := vfs.Getwd()
		if err != nil {
			return "", fileError("resolve working directory", err)
		}
		project = filepath.Join(cwd, project)
	}
	project, err := vfs.EvalSymlinks(project)
	if err != nil {
		return "", fileError("resolve existing project directory", err)
	}
	info, err := vfs.Stat(project)
	if err != nil {
		return "", fileError("inspect project directory", err)
	}
	if !info.IsDir() {
		return "", conflict("project is not a directory", project)
	}
	// Fixed child names bound every write to this project. Reject links in the
	// managed subtree rather than following them into another skill location.
	dir := project
	for _, child := range []string{".agents", "skills", Name} {
		dir = filepath.Join(dir, child)
		info, err := vfs.Lstat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", fileError("inspect project skill directory", err)
		}
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return "", conflict("skill directory component is not a plain directory", dir)
		}
	}
	return dir, nil
}

func conflict(message, path string) error {
	return errs.NewValidationError(errs.SubtypeFailedPrecondition, "%s: %s", message, path).
		WithHint("preserve local edits; resolve the named collision or stale lock before retrying")
}

func fileError(operation string, cause error) error {
	return errs.NewInternalError(errs.SubtypeFileIO, "%s failed", operation).WithCause(cause)
}
