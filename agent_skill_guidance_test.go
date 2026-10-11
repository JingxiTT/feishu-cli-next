// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"io/fs"
	"path"
	"strings"
	"testing"
)

// Compact entry guides must keep their current workflow destinations available
// in the shipped binary; source-only files cannot satisfy these links.
func TestCompactSkillRoutesRemainEmbedded(t *testing.T) {
	routes := map[string][]string{
		"lark-im": {
			"../lark-shared/SKILL.md",
			"references/card/lark-im-card-create.md",
			"references/lark-im-card-action-reply.md",
			"references/lark-im-chat-create.md",
			"references/lark-im-chat-list.md",
			"references/lark-im-chat-members-list.md",
			"references/lark-im-chat-messages-list.md",
			"references/lark-im-chat-search.md",
			"references/lark-im-chat-update.md",
			"references/lark-im-feed-group-list-item.md",
			"references/lark-im-feed-group-list.md",
			"references/lark-im-feed-group-query-item.md",
			"references/lark-im-feed-groups.md",
			"references/lark-im-feed-shortcut-create.md",
			"references/lark-im-feed-shortcut-list.md",
			"references/lark-im-feed-shortcut-remove.md",
			"references/lark-im-flag-cancel.md",
			"references/lark-im-flag-create.md",
			"references/lark-im-flag-list.md",
			"references/lark-im-message-enrichment.md",
			"references/lark-im-message-read-status.md",
			"references/lark-im-messages-edit.md",
			"references/lark-im-messages-mget.md",
			"references/lark-im-messages-reply.md",
			"references/lark-im-messages-resources-download.md",
			"references/lark-im-messages-search.md",
			"references/lark-im-messages-send.md",
			"references/lark-im-reactions.md",
			"references/lark-im-threads-messages-list.md",
		},
		"lark-sheets": {
			"../lark-shared/SKILL.md",
			"references/lark-sheets-batch-update.md",
			"references/lark-sheets-changeset.md",
			"references/lark-sheets-chart.md",
			"references/lark-sheets-conditional-format.md",
			"references/lark-sheets-filter-view.md",
			"references/lark-sheets-filter.md",
			"references/lark-sheets-float-image.md",
			"references/lark-sheets-formula-translation.md",
			"references/lark-sheets-formula-verify.md",
			"references/lark-sheets-history.md",
			"references/lark-sheets-pivot-table.md",
			"references/lark-sheets-range-operations.md",
			"references/lark-sheets-read-data.md",
			"references/lark-sheets-search-replace.md",
			"references/lark-sheets-sheet-structure.md",
			"references/lark-sheets-sparkline.md",
			"references/lark-sheets-styles-put.md",
			"references/lark-sheets-visual-standards.md",
			"references/lark-sheets-workbook.md",
			"references/lark-sheets-write-cells.md",
		},
		"lark-openapi-explorer": {
			"../lark-shared/SKILL.md",
			"../lark-skill-maker/SKILL.md",
		},
	}
	for name, references := range routes {
		t.Run(name, func(t *testing.T) {
			entryPath := "skills/" + name + "/SKILL.md"
			raw, err := fs.ReadFile(embeddedContentFS, entryPath)
			if err != nil {
				t.Fatal(err)
			}
			for _, reference := range references {
				if !strings.Contains(string(raw), "]("+reference+")") {
					t.Errorf("entry no longer routes to %s", reference)
				}
				target := path.Clean(path.Join(path.Dir(entryPath), reference))
				if _, err := fs.ReadFile(embeddedContentFS, target); err != nil {
					t.Errorf("embedded workflow %s: %v", target, err)
				}
			}
		})
	}
}

func TestCompactSkillDecisionBoundaries(t *testing.T) {
	// Check operational anchors rather than exact prose: these rules protect
	// identities, approval, coordinates, types, verification and bounded reads.
	boundaries := map[string][]string{
		"lark-im": {
			"user_access_token", "tenant_access_token", "明确授权",
			"interactive", "必须读", "不能手写", "Opus", "cite type=",
			"--no-reactions", "--download-resources", "sender_i18n_names",
			"has_more", "page_token", "逐项检查", "7 天", "14 天", "30 KB",
			"--input-only", "bot-only", "--text", "--as user",
		},
		"lark-sheets": {
			"只写空格", "读全再写", "row_indices", "col_indices", "[row=N]",
			"--include formula", "number_format", "--inherit-style", "行高不继承",
			"失败 3 次", "静态", "来源", "不能编造", "未完成",
			"+workbook-export", "+revision-get", "--output-path", "专有 --sheet-id",
			"飞书表格编辑准则", "公共 flag 速查", "Agent 使用提示",
			"--writes", "--print-schema", "--flag-name", "properties.snapshot.plotArea.axes",
			"stdin", "UTF-8", "@./file.json", "scripts/", "不是前置依赖",
			"has_more", "complete", "unread_sheets", "exit 10", "明确同意", "--yes",
		},
		"lark-openapi-explorer": {
			"feishu-cli-next discover 'im members' --limit 3", "--flags", "lark-cli <service> --help",
			"schema im.chat.members.create --input-only", "im chat.members create",
			"https://open.feishu.cn/llms.txt", "https://open.larksuite.com/llms.txt",
			"用户意图", "高风险", "不要猜路径",
		},
	}
	for name, required := range boundaries {
		raw, err := fs.ReadFile(embeddedContentFS, "skills/"+name+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		for _, anchor := range required {
			if !strings.Contains(string(raw), anchor) {
				t.Errorf("%s lost decision boundary %q", name, anchor)
			}
		}
	}
	explorer, err := fs.ReadFile(embeddedContentFS, "skills/lark-openapi-explorer/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(explorer), "没有 chat_members") ||
		strings.Contains(string(explorer), "没有 announcement") {
		t.Fatal("explorer treats registered methods as absent before discovery")
	}
}
