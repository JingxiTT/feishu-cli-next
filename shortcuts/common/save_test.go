// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"io"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/extension/fileio"
)

type saveOnlyFileIO struct {
	fileio.FileIO
	saves int
}

func (f *saveOnlyFileIO) Save(string, fileio.SaveOptions, io.Reader) (fileio.SaveResult, error) {
	f.saves++
	return nil, nil
}

func TestSaveWithOverwriteRejectsIncapableProvider(t *testing.T) {
	fio := &saveOnlyFileIO{}
	_, err := SaveWithOverwrite(fio, "result.bin", fileio.SaveOptions{}, strings.NewReader("payload"), false)
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Category != errs.CategoryValidation || problem.Subtype != errs.SubtypeFailedPrecondition || fio.saves != 0 {
		t.Fatalf("unsafe fallback: saves=%d, problem=%#v, error=%v", fio.saves, problem, err)
	}
	if _, err := SaveWithOverwrite(fio, "result.bin", fileio.SaveOptions{}, strings.NewReader("payload"), true); err != nil || fio.saves != 1 {
		t.Fatalf("explicit overwrite rejected: saves=%d, error=%v", fio.saves, err)
	}
}
