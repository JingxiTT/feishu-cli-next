// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docworkspace

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func selectionFixture() []json.RawMessage {
	// Deliberately shuffled; child links define document order.
	return []json.RawMessage{
		json.RawMessage(`{"block_id":"last","parent_id":"doc","block_type":2,"text":{"elements":[{"text_run":{"content":"Other section"}}]}}`),
		json.RawMessage(`{"block_id":"doc","block_type":1,"children":["h1","p","h2","last"]}`),
		json.RawMessage(`{"block_id":"h2","parent_id":"doc","block_type":3,"heading1":{"elements":[{"text_run":{"content":"Next"}}]}}`),
		json.RawMessage(`{"block_id":"p","parent_id":"doc","block_type":2,"text":{"elements":[{"text_run":{"content":"Budget 中文 42"}}]}}`),
		json.RawMessage(`{"block_id":"h1","parent_id":"doc","block_type":3,"heading1":{"elements":[{"text_run":{"content":"Plan"}}]}}`),
	}
}

func TestSelectionUsesNativeHierarchyAndExplicitBounds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		selection Selection
		ids       []string
	}{
		{"outline", Selection{Scope: "outline"}, []string{"h1", "h2"}},
		{"section", Selection{Scope: "section", StartID: "h1"}, []string{"h1", "p"}},
		{"range", Selection{Scope: "range", StartID: "p", EndID: "h2"}, []string{"p", "h2"}},
		{"keyword", Selection{Scope: "keyword", Keyword: "budget 中文"}, []string{"p"}},
		{"no_match", Selection{Scope: "keyword", Keyword: "absent"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md, n, err := Select(selectionFixture(), tc.selection)
			if err != nil || n != len(tc.ids) {
				t.Fatalf("n=%d err=%v", n, err)
			}
			matches := markerPattern.FindAllStringSubmatch(md, -1)
			if len(matches) != len(tc.ids) {
				t.Fatalf("anchors: %q", md)
			}
			for i, id := range tc.ids {
				if matches[i][1] != id {
					t.Fatalf("order: %q", md)
				}
			}
		})
	}
	md, n, err := Select(selectionFixture(), Selection{Scope: "full"})
	if err != nil || n != 5 || !strings.HasPrefix(md, marker("last")) {
		t.Fatalf("full view contract changed: %q %d %v", md, n, err)
	}
}

func TestSelectionRejectsInvalidRequestsAndHierarchy(t *testing.T) {
	for _, s := range []Selection{
		{Scope: "full", Keyword: "ignored"}, {Scope: "outline", StartID: "p"},
		{Scope: "range"}, {Scope: "range", StartID: "last", EndID: "p"},
		{Scope: "section", StartID: "p"}, {Scope: "section", StartID: "missing"},
		{Scope: "keyword", Keyword: " "}, {Scope: "invalid"},
	} {
		_, _, err := Select(selectionFixture(), s)
		var typed *errs.ValidationError
		if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument {
			t.Fatalf("%+v: %v", s, err)
		}
	}
	for _, root := range []string{
		`{"block_id":"doc","block_type":1,"children":["h1","p","h2","missing"]}`,
		`{"block_id":"doc","block_type":1,"children":["h1","p","h2","last","p"]}`,
		`{"block_id":"doc","block_type":1,"children":["h1","p","h2"]}`,
	} {
		blocks := selectionFixture()
		blocks[1] = json.RawMessage(root)
		_, _, err := Select(blocks, Selection{Scope: "outline"})
		var typed *errs.InternalError
		if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidResponse {
			t.Fatalf("hierarchy accepted: %v", err)
		}
	}
}
