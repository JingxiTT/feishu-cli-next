// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"embed"
	"io/fs"

	"github.com/larksuite/cli/cmd"
)

//go:embed agent-skills/feishu-cli-next
var agentContent embed.FS

func init() {
	content, err := fs.Sub(agentContent, "agent-skills")
	if err == nil {
		cmd.SetAgentSkillContent(content)
	}
}
