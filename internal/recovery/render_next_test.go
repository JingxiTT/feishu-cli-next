// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recovery

import (
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestCloneMutationOutcomeCopiesDiagnosticSlices(t *testing.T) {
	original := errs.NewMutationOutcomeError("unknown", []string{"id"}, false, errs.NewPermissionError(errs.SubtypeMissingScope, "scope"))
	value, ok := CloneTyped(original)
	if !ok {
		t.Fatal("not cloned")
	}
	clone := value.(*errs.MutationOutcomeError)
	clone.Receipts[0] = "changed"
	clone.CauseDetail[0] = 'x'
	if original.Receipts[0] != "id" || original.CauseDetail[0] != '{' {
		t.Fatal("clone mutated source")
	}
}
