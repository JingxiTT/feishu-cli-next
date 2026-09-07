// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	"io/fs"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/skillcontent"
)

func TestEntrySkillBaseGuideCommandsReachEmbeddedContent(t *testing.T) {
	entry, err := fs.ReadFile(agentContent, "agent-skills/feishu-cli-next/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	tree, err := fs.Sub(embeddedContentFS, "skills")
	if err != nil {
		t.Fatal(err)
	}
	reader := skillcontent.New(tree)
	for _, fieldType := range []string{"formula", "lookup"} {
		path := "references/lark-base-field-" + fieldType + ".md"
		command := "feishu-cli-next skills read lark-base " + path
		if !strings.Contains(string(entry), "`"+command+"`") {
			t.Fatalf("entry skill cannot route to %s", command)
		}
		data, resolved, err := reader.ReadReference("lark-base", path)
		if err != nil || resolved != path {
			t.Fatalf("embedded guide unreachable: %s %v", resolved, err)
		}
		if !bytes.HasPrefix(data, []byte("# Base ")) || !bytes.Contains(data, []byte("--i-have-read-guide")) {
			t.Fatalf("missing English guide or acknowledgement contract: %s", path)
		}
	}
}
