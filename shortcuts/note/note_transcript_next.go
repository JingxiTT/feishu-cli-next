// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package note

import (
	"github.com/larksuite/cli/errs"
)

// transcriptPage projects the loose response before pagination consumes it.
// Empty chunks are valid; absent or non-string chunks cannot prove completeness.
type transcriptPage struct {
	Content    string
	HasMore    bool
	NextCursor any
}

func parseTranscriptPage(data map[string]any, format string) (transcriptPage, error) {
	var page transcriptPage
	transcript, ok := data["transcript"].(map[string]any)
	if !ok {
		return page, errs.NewInternalError(errs.SubtypeInvalidResponse, "transcript page is missing its content object")
	}
	page.Content, ok = transcript[format].(string)
	if !ok {
		return page, errs.NewInternalError(errs.SubtypeInvalidResponse, "transcript page is missing string content for %s", format)
	}
	// Preserve the legacy omitted has_more terminal-page behavior, but never
	// interpret a malformed value (for example the string "true") as false.
	if value, present := data["has_more"]; present {
		page.HasMore, ok = value.(bool)
		if !ok {
			return page, errs.NewInternalError(errs.SubtypeInvalidResponse, "transcript page has_more must be boolean")
		}
	}
	page.NextCursor = data["next_cursor_id"]
	return page, nil
}
