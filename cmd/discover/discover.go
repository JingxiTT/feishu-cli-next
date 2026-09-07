// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package discover provides bounded discovery from the executable command tree.
package discover

import (
	"sort"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type entry struct {
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Risk        string   `json:"risk,omitempty"`
	Identities  []string `json:"identities,omitempty"`
	Flags       []flag   `json:"flags,omitempty"`
}

type flag struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// NewCmd uses the final tree, including host contributions and visibility rules.
// It never resolves credentials or executes the commands it describes.
func NewCmd(f *cmdutil.Factory, root *cobra.Command) *cobra.Command {
	var limit, offset int
	var withFlags bool
	cmd := &cobra.Command{
		Use:   "discover [query...]",
		Short: "Find commands by task or domain; no query lists command groups",
		Long:  "Search registered command paths and descriptions. Use '*' for all commands. Results describe the local command surface; they do not verify live permissions or backend behavior.",
		RunE: func(c *cobra.Command, args []string) error {
			if limit < 1 || limit > 1000 || offset < 0 {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "limit must be 1..1000 and offset must be nonnegative")
			}
			query := strings.Join(args, " ")
			entries := collect(root, query, withFlags)
			start := min(offset, len(entries))
			end := min(start+limit, len(entries))
			page := map[string]any{"query": query, "offset": offset, "limit": limit, "total": len(entries), "complete": end == len(entries)}
			if end < len(entries) {
				page["next_offset"] = end
			}
			return output.WriteJSON(f.IOStreams.Out, map[string]any{"ok": true, "data": entries[start:end], "meta": page})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum results (1..1000)")
	cmd.Flags().IntVar(&offset, "offset", 0, "resume this query at a result offset")
	cmd.Flags().BoolVar(&withFlags, "flags", false, "include flag types, defaults and required markers")
	cmdutil.DisableAuthCheck(cmd)
	cmdutil.SetRisk(cmd, "read")
	return cmd
}

func collect(root *cobra.Command, query string, withFlags bool) []entry {
	entries := []entry{}
	terms := strings.Fields(strings.ToLower(query))
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Hidden {
			return
		}
		path := strings.TrimSpace(strings.TrimPrefix(c.CommandPath(), root.Name()))
		if c != root && (query != "" || c.Parent() == root) {
			haystack := strings.ToLower(path + " " + c.Short + " " + strings.Join(c.Aliases, " "))
			match := true
			for _, term := range terms {
				if term != "*" && !strings.Contains(haystack, term) {
					match = false
					break
				}
			}
			if match {
				risk, _ := cmdmeta.Risk(c)
				e := entry{Command: path, Description: c.Short, Risk: risk, Identities: cmdmeta.Identities(c)}
				if withFlags {
					seen := map[string]bool{}
					visit := func(f *pflag.Flag) {
						if f.Hidden || seen[f.Name] {
							return
						}
						seen[f.Name] = true
						_, required := f.Annotations[cobra.BashCompOneRequiredFlag]
						e.Flags = append(e.Flags, flag{Name: f.Name, Type: f.Value.Type(), Default: f.DefValue, Required: required})
					}
					c.LocalNonPersistentFlags().VisitAll(visit)
					c.PersistentFlags().VisitAll(visit)
					c.InheritedFlags().VisitAll(visit)
					sort.Slice(e.Flags, func(i, j int) bool { return e.Flags[i].Name < e.Flags[j].Name })
				}
				entries = append(entries, e)
			}
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Command < entries[j].Command })
	return entries
}
