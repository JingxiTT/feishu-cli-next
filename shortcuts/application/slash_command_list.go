// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"context"
	"fmt"
	"io"

	"github.com/larksuite/cli/shortcuts/common"
)

// SlashCommandList lists all slash commands of the current bound app.
var SlashCommandList = common.Shortcut{
	Service:     "application",
	Command:     "+slash-command-list",
	Description: "List all slash commands (/ commands) registered on the currently bound Open Platform app; source of command_id for update/delete",
	Risk:        "read",
	Scopes:      []string{"application:app_slash_command:read"},
	AuthTypes:   []string{"bot", "user"},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		return common.NewDryRunAPI().
			Desc("List all slash commands of the current bound app (read-only)").
			GET(slashCommandBasePath)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		data, items, err := commandList(runtime)
		if err != nil {
			return err
		}
		data["items"] = items
		data["count"] = len(items)
		out := data
		runtime.OutFormat(out, nil, func(w io.Writer) {
			fmt.Fprintf(w, "%d slash command(s)\n", len(items))
			for _, it := range items {
				state, _ := decodeCommandState(it) // validated before emitting
				fmt.Fprintf(w, "  /%s\t%s\t%s\n", state.Command, state.ID, state.Description.DefaultValue)
			}
		})
		return nil
	},
}
