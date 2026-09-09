// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestDocsPullSelectionCoverageAndNoWrites(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	s := &workspaceServer{revision: 3, blocks: workspaceBlocks()}
	s.mount(t, reg)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--scope", "keyword", "--keyword", "second"); err != nil {
		t.Fatal(err)
	}
	var reply struct {
		Data struct {
			Complete bool   `json:"complete"`
			Returned int    `json:"returned_blocks"`
			Total    int    `json:"total_blocks"`
			Markdown string `json:"markdown"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Data.Complete || reply.Data.Returned != 1 || reply.Data.Total != 4 || !strings.Contains(reply.Data.Markdown, "block:p2") || strings.Contains(reply.Data.Markdown, "block:p1") || s.patches != 0 {
		t.Fatalf("bad selection: %s writes=%d", out.String(), s.patches)
	}
}

func TestDocsPullRejectsPartialWorkspaceBeforeRead(t *testing.T) {
	for _, args := range [][]string{
		{"--scope", "keyword", "--keyword", "Second", "--output", "note.md"},
		{"--scope", "full", "--keyword", "Second"},
		{"--scope", "section"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			f, reg, out := workspaceFactory(t)
			s := &workspaceServer{revision: 3, blocks: workspaceBlocks()}
			s.mount(t, reg)
			err := workspaceRun(t, DocsPull, f, out, append([]string{"+pull", "--doc", "doc1", "--dry-run"}, args...)...)
			var typed *errs.ValidationError
			if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || s.metadataCalls+s.blockCalls+s.patches != 0 {
				t.Fatalf("err=%v reads=%d", err, s.metadataCalls+s.blockCalls)
			}
		})
	}
}
