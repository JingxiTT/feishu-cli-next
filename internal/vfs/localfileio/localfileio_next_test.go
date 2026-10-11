// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package localfileio

import (
	"io/fs"
	"os"

	"github.com/larksuite/cli/internal/vfs"
)

// saveValidationFS has no backing filesystem: even a validation regression cannot
// write to the host. Unexpected operations panic through the nil embedded FS.
type saveValidationFS struct {
	vfs.FS
	cwd        string
	mkdirCalls int
	writeCalls int
}

func (f *saveValidationFS) Getwd() (string, error) { return f.cwd, nil }

func (f *saveValidationFS) Lstat(string) (fs.FileInfo, error) { return nil, fs.ErrNotExist }

func (f *saveValidationFS) MkdirAll(string, fs.FileMode) error {
	f.mkdirCalls++
	return fs.ErrPermission
}

func (f *saveValidationFS) CreateTemp(string, string) (*os.File, error) {
	f.writeCalls++
	return nil, fs.ErrPermission
}
