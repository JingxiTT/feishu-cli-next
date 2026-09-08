// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/affordance"
	"github.com/larksuite/cli/internal/meta"
	docshortcuts "github.com/larksuite/cli/shortcuts/doc"
)

// The shortcut owns its concise description, while affordance/docs.md owns
// additive when-to-use guidance, tips, prerequisites, and skill references.
// Keep those roles distinct: copying the description into the lead renders two
// near-identical descriptions, while duplicating Tips in Go and Markdown lets
// the two sources drift because the Markdown overlay wins at help time.
func TestEmbeddedDocsAffordanceComplementsShortcutMetadata(t *testing.T) {
	const contentGuideTip = "XML preserves native rich structure; the XML guide describes supported blocks and reference maps when authoring unfamiliar content."

	type expectation struct {
		description   string
		useWhen       []string
		tips          []string
		prerequisites []string
		skills        []string
	}
	want := map[string]expectation{
		"+create": {
			description: "Create a Lark document",
			useWhen: []string{
				"Create a new Lark document from DocxXML or Markdown, optionally in a folder or Wiki node.",
			},
			tips: []string{
				"Match `--doc-format` to `--content`: XML is the default for rich DocxXML; use `--doc-format markdown` for Markdown input.",
				contentGuideTip,
				"For multiline `--content`, prefer `@file` or `-` (stdin) to avoid shell-escaping damage.",
			},
			skills: []string{
				"lark-doc",
				"lark-doc/references/lark-doc-create-workflow.md",
				"lark-doc/references/lark-doc-create.md",
				"lark-doc/references/lark-doc-xml.md",
				"lark-doc/references/lark-doc-md.md",
			},
		},
		"+fetch": {
			description: "Fetch Lark document content",
			useWhen: []string{
				"Read an entire Lark document, or limit the result to an outline, section, block range, or keyword match.",
			},
			tips: []string{
				"For facts and native block addresses, choose XML `with-ids`; use `full` when styles and edit/reference metadata are needed. Markdown and IM Markdown are explicit conversions.",
				"Bounds require a matching scope; `full` with range or keyword bounds fails instead of silently returning the entire document.",
				"Read small fact-lookup results directly from JSON stdout. Use `--save-result` for requested exports or large responses, preserving `document.reference_map` alongside `document.content`; the artifact is not a document import payload.",
			},
		},
		"+update": {
			description: "Update a Lark document",
			useWhen: []string{
				"Apply targeted text or block edits, append content, or deliberately replace an entire Lark document.",
			},
			tips: []string{
				"Prefer `str_replace` or `block_*` commands for targeted edits. Use `overwrite` only when replacing the entire document is intended; it can discard unrelated rich content.",
				"For one occurrence, use the smallest `--pattern` established unique across the document; leave unchanged styled text outside the match. If uniqueness is uncertain, use a native block edit addressed by ID, preserving unrelated block IDs and resources.",
				"When the match includes styled text, copy its actual XML marks and link `href` into replacement content and change only the intended characters; plain replacement can remove existing formatting.",
				"After a targeted edit, fetch the affected native XML and verify text, styles and link destinations before reporting success.",
				"Native block IDs come from `docs +fetch --detail with-ids`; structural edits can invalidate earlier addressing.",
				contentGuideTip,
				"Match `--doc-format` to `--content`; for multiline content, prefer `@file` or `-` (stdin).",
				"XML overwrite first compares a full native snapshot and skips an exact unchanged body/reference map without a write. Markdown equality cannot prove rich-content equivalence.",
				"A revision is forwarded as the base revision; local receipts and readback do not prevent concurrent edits by other clients.",
				"After partial or unknown effects, inspect the returned receipts and fetch the affected content before deciding whether another write is needed.",
			},
		},
		"+history-list": {
			description: "List Lark document history versions",
		},
		"+history-revert": {
			description: "Revert a Lark document to a historical version",
			tips: []string{
				"A returned task ID can represent pending work. Poll `+history-revert-status`; do not replay the revert after an unknown response.",
			},
			prerequisites: []string{
				"`history_version_id` from `+history-list`",
			},
		},
		"+history-revert-status": {
			description: "Get Lark document history revert task status",
			prerequisites: []string{
				"`task_id` from `+history-revert`",
			},
		},
	}

	shortcuts := make(map[string]struct {
		description string
		tips        []string
	})
	for _, shortcut := range docshortcuts.Shortcuts() {
		shortcuts[shortcut.Command] = struct {
			description string
			tips        []string
		}{description: shortcut.Description, tips: shortcut.Tips}
	}

	for command, expected := range want {
		t.Run(command, func(t *testing.T) {
			shortcut, ok := shortcuts[command]
			if !ok {
				t.Fatalf("docs shortcut %s is not registered", command)
			}
			if shortcut.description != expected.description {
				t.Fatalf("shortcut description = %q, want %q", shortcut.description, expected.description)
			}
			if len(shortcut.tips) != 0 {
				t.Fatalf("shortcut Tips = %q; docs affordance owns these tips and must remain the single source", shortcut.tips)
			}

			raw, ok := affordance.For("docs", command)
			if !ok {
				t.Fatalf("embedded affordance entry docs/%s is missing", command)
			}
			parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
			if !ok {
				t.Fatalf("embedded affordance entry docs/%s is invalid: %s", command, raw)
			}
			if !slices.Equal(parsed.UseWhen, expected.useWhen) {
				t.Errorf("use_when = %q, want %q", parsed.UseWhen, expected.useWhen)
			}
			if !slices.Equal(parsed.Tips, expected.tips) {
				t.Errorf("tips = %q, want %q", parsed.Tips, expected.tips)
			}
			if !slices.Equal(parsed.Prerequisites, expected.prerequisites) {
				t.Errorf("prerequisites = %q, want %q", parsed.Prerequisites, expected.prerequisites)
			}
			if expected.skills != nil && !slices.Equal(parsed.Skills, expected.skills) {
				t.Errorf("skills = %q, want %q", parsed.Skills, expected.skills)
			}
			for _, lead := range parsed.UseWhen {
				if normalizedCopy(lead) == normalizedCopy(shortcut.description) {
					t.Errorf("use_when repeats the shortcut description instead of adding a decision boundary: %q", lead)
				}
			}
		})
	}
}

func normalizedCopy(value string) string {
	return strings.ToLower(strings.TrimRight(strings.TrimSpace(value), "."))
}

func TestEmbeddedDocsWorkspaceReferenceIsReachable(t *testing.T) {
	raw, err := embeddedContentFS.ReadFile("skills/lark-doc/references/lark-doc-workspace.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"docs +pull", "docs +push", "docs +update"} {
		if !strings.Contains(string(raw), command) {
			t.Errorf("workspace reference missing %q", command)
		}
	}
}
