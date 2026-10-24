// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import (
	"errors"

	"github.com/larksuite/cli/errs"
)

// Exit codes for mutations whose effect is not a plain success.
const (
	ExitMutationUnapplied = 20 // confirmed rejection of every attempted mutation
	ExitMutationPartial   = 21 // both applied and unapplied mutations
	ExitMutationUnknown   = 22 // unknown effect or failed final evidence persistence
)

// mutationExitCode maps an *errs.MutationOutcomeError in err's chain to its
// evidence-derived exit code.
func mutationExitCode(err error) (int, bool) {
	var mutation *errs.MutationOutcomeError
	if !errors.As(err, &mutation) {
		return 0, false
	}
	if mutation.PersistenceFailed {
		return ExitMutationUnknown, true
	}
	switch mutation.Outcome {
	case errs.MutationUnapplied:
		return ExitMutationUnapplied, true
	case errs.MutationPartial:
		return ExitMutationPartial, true
	default:
		return ExitMutationUnknown, true
	}
}
