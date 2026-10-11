// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail

import (
	draftpkg "github.com/larksuite/cli/shortcuts/mail/draft"
)

// draftReplacementOnly excludes operations that can have effects beyond the
// final MIME representation, such as uploading an attachment.
func draftReplacementOnly(patch draftpkg.Patch) bool {
	for _, op := range patch.Ops {
		switch op.Op {
		case "set_subject", "set_recipients", "set_reply_to", "clear_reply_to", "set_body", "replace_body", "set_header", "remove_header":
		default:
			return false
		}
	}
	return true
}
