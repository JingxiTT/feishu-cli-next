// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"io"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
)

// Existence checks are preflight only; the provider must enforce no-clobber at
// commit time so a concurrent writer cannot lose its destination file.
func saveRecordExportFile(fio fileio.FileIO, path string, opts fileio.SaveOptions, body io.Reader, overwrite bool) (fileio.SaveResult, error) {
	if overwrite {
		return fio.Save(path, opts, body)
	}
	exclusive, ok := fio.(fileio.ExclusiveFileIO)
	if !ok {
		return nil, errs.NewValidationError(errs.SubtypeFailedPrecondition,
			"record export without --overwrite requires an atomic exclusive FileIO provider")
	}
	return exclusive.SaveExclusive(path, opts, body)
}
