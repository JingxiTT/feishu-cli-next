// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr

import "github.com/larksuite/cli/errs"

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
