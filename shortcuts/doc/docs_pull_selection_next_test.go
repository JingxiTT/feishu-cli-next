// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"encoding/json"
	"errors"
	"os"
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

func TestDocsPushMultipleRunNativeReadback(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	s := &workspaceServer{revision: 3, blocks: workspaceBlocks()}
	s.mount(t, reg)
	md := workspaceMarkdown(t, s.blocks)
	edited := strings.Replace(strings.Replace(md, "Old 中文", "New 中文", 1), " rest", " remainder", 1)
	workspaceSave(t, f, s.blocks, edited)
	if err := workspaceRun(t, DocsPush, f, out, "+push", "--file", "note.md"); err != nil {
		t.Fatal(err)
	}
	var reply struct {
		Data struct {
			Verified bool `json:"verified"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if s.patches != 1 || !reply.Data.Verified || !strings.Contains(out.String(), "New 中文 remainder") {
		t.Fatalf("push=%s patches=%d", out.String(), s.patches)
	}
	if !strings.Contains(string(s.blocks[1]), `"bold":true`) || !strings.Contains(string(s.blocks[1]), `"url":"https://example.org"`) {
		t.Fatalf("native style/link lost: %s", s.blocks[1])
	}
	if raw, err := os.ReadFile("note.md"); err != nil || string(raw) != edited {
		t.Fatalf("local edit changed: %v", err)
	}
}

func TestDocsPullReadableStdoutKeepsSavedV1View(t *testing.T) {
	f, reg, out := workspaceFactory(t)
	blocks := workspaceBlocks()
	blocks[0] = json.RawMessage(`{"block_id":"doc1","block_type":1,"children":["p1","p2","image","bullet"],"page":{"elements":[]}}`)
	blocks = append(blocks, json.RawMessage(`{"block_id":"bullet","parent_id":"doc1","block_type":12,"bullet":{"elements":[{"text_run":{"content":"Launch Friday"}}]}}`))
	s := &workspaceServer{revision: 3, blocks: blocks}
	s.mount(t, reg)
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--scope", "keyword", "--keyword", "launch"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Launch Friday") || !strings.Contains(out.String(), "block:bullet") {
		t.Fatalf("readable text absent: %s", out)
	}
	if err := workspaceRun(t, DocsPull, f, out, "+pull", "--doc", "doc1", "--output", "legacy.md"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("legacy.md")
	if err != nil || string(raw) != workspaceMarkdown(t, blocks) || strings.Contains(string(raw), "Launch Friday") {
		t.Fatalf("saved v1 projection changed: %s %v", raw, err)
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
