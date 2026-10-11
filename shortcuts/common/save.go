// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"io"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
)

// SaveWithOverwrite enforces no-overwrite at commit time, including concurrent writers.
func SaveWithOverwrite(fio fileio.FileIO, path string, opts fileio.SaveOptions, body io.Reader, overwrite bool) (fileio.SaveResult, error) {
	if overwrite {
		return fio.Save(path, opts, body)
	}
	exclusive, ok := fio.(fileio.ExclusiveFileIO)
	if !ok {
		return nil, errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"saving without overwrite requires an atomic exclusive FileIO provider")
	}
	return exclusive.SaveExclusive(path, opts, body)
}
