// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package note

import (
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestParseTranscriptPagePreservesContentAndRejectsMalformedFields(t *testing.T) {
	for _, format := range []string{transcriptFormatMarkdown, transcriptFormatPlainText} {
		t.Run(format, func(t *testing.T) {
			for _, content := range []string{"", "  中文 👋\n**bold**\n"} {
				got, err := parseTranscriptPage(map[string]any{"has_more": true, "next_cursor_id": "7648924766078847940", "transcript": map[string]any{format: content}}, format)
				if err != nil || got.Content != content || !got.HasMore || got.NextCursor != "7648924766078847940" {
					t.Fatalf("page=%#v err=%v", got, err)
				}
			}
			terminal, err := parseTranscriptPage(map[string]any{"transcript": map[string]any{format: "tail"}}, format)
			if err != nil || terminal.HasMore {
				t.Fatalf("legacy terminal page=%#v err=%v", terminal, err)
			}
			for _, data := range []map[string]any{
				{}, {"transcript": nil}, {"transcript": map[string]any{}},
				{"transcript": map[string]any{format: nil}}, {"transcript": map[string]any{format: 42}},
				{"transcript": map[string]any{format: "tail"}, "has_more": "true"},
				{"transcript": map[string]any{format: "tail"}, "has_more": nil},
			} {
				_, err := parseTranscriptPage(data, format)
				problem, ok := errs.ProblemOf(err)
				if !ok || problem.Category != errs.CategoryInternal || problem.Subtype != errs.SubtypeInvalidResponse {
					t.Fatalf("malformed page=%#v err=%v", data, err)
				}
			}
		})
	}
}
