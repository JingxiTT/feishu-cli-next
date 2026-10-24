// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr

import (
	"strconv"

	"github.com/larksuite/cli/errs"
)

// advanceOKRPage refuses incomplete pagination before consumers can replace a
// whole collection or claim a complete cycle. Tokens are opaque and verbatim.
func advanceOKRPage(hasMore bool, token string, seen map[string]bool) error {
	if !hasMore {
		return nil
	}
	if token == "" || seen[token] {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "OKR pagination returned a missing or repeated continuation token").WithHint("Read the collection again before making changes; this result is incomplete.")
	}
	seen[token] = true
	return nil
}

// validateOKRMutationIDs checks the complete native collection before algorithms
// key it by ID. A duplicate would collapse a reorder or misassign weights; an
// invalid ID cannot safely identify a replacement target. Keep valid IDs verbatim.
func validateOKRMutationIDs[T reorderItem](items []T) error {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		id := item.GetID()
		value, err := strconv.ParseInt(id, 10, 64)
		if err != nil || value <= 0 {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "invalid native OKR collection ID %q", id).WithCause(err).WithHint("Read the complete collection again before replacing its order or weights; no replacement was sent.")
		}
		if seen[id] {
			return errs.NewInternalError(errs.SubtypeInvalidResponse, "duplicate native OKR collection ID %q", id).WithHint("Read the complete collection again before replacing its order or weights; no replacement was sent.")
		}
		seen[id] = true
	}
	return nil
}
