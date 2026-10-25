// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package discover provides bounded discovery from the executable command tree.
package discover

import (
	"slices"
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
	rank        rank
	command     *cobra.Command
}

type flag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// NewCmd uses the final tree, including host contributions and visibility rules.
// It never resolves credentials or executes the commands it describes.
func NewCmd(f *cmdutil.Factory, root *cobra.Command) *cobra.Command {
	var limit, offset int
	var withFlags bool
	cmd := &cobra.Command{
		Use:   "discover [query...]",
		Short: "Find commands by literal keywords; no query lists command groups",
		Long:  "Match every whitespace-separated keyword as a case-insensitive substring of command paths, aliases, descriptions, or public flag names and descriptions. Use '*' for all commands. Results describe the local command surface; they do not verify live permissions or backend behavior.",
		RunE: func(c *cobra.Command, args []string) error {
			if limit < 1 || limit > 1000 || offset < 0 {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "limit must be 1..1000 and offset must be nonnegative")
			}
			query := strings.Join(args, " ")
			entries := collect(root, query)
			start := min(offset, len(entries))
			end := min(start+limit, len(entries))
			page := map[string]any{"query": query, "offset": offset, "limit": limit, "total": len(entries), "complete": end == len(entries)}
			if end < len(entries) {
				page["next_offset"] = end
			}
			if withFlags {
				if shared := describePage(entries[start:end]); len(shared) > 0 {
					page["common_flags"] = shared
				}
			}
			if len(entries) == 0 && query != "" {
				page["hint"] = "No commands matched. Matching is literal substring and every term must match; try fewer or shorter keywords."
			}
			return output.WriteJSON(f.IOStreams.Out, map[string]any{"ok": true, "data": entries[start:end], "meta": page})
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "maximum results (1..1000)")
	cmd.Flags().IntVar(&offset, "offset", 0, "resume this query at a result offset")
	cmd.Flags().BoolVar(&withFlags, "flags", false, "include flag descriptions, types, defaults and required markers")
	cmdutil.DisableAuthCheck(cmd)
	cmdutil.SetRisk(cmd, "read")
	cmdmeta.SetRequiresFullTree(cmd)
	return cmd
}

func collect(root *cobra.Command, query string) []entry {
	entries := []entry{}
	terms := strings.Fields(strings.ToLower(query))
	var walk func(*cobra.Command, string)
	walk = func(c *cobra.Command, commandPath string) {
		if c.Hidden {
			return
		}
		path := strings.TrimSpace(strings.TrimPrefix(commandPath, root.Name()))
		if c != root && (query != "" || c.Parent() == root) {
			if r, ok := matchRank(c, path, terms); ok {
				risk, _ := cmdmeta.Risk(c)
				entries = append(entries, entry{Command: path, Description: c.Short, Risk: risk, Identities: cmdmeta.Identities(c), rank: r, command: c})
			}
		}
		for _, child := range c.Commands() {
			walk(child, commandPath+" "+child.Name())
		}
	}
	walk(root, root.CommandPath())
	// Listings without a search term (empty query or `*`) stay alphabetical.
	if strings.Trim(strings.Join(terms, ""), "*") == "" {
		sort.Slice(entries, func(i, j int) bool { return entries[i].Command < entries[j].Command })
	} else {
		sort.Slice(entries, func(i, j int) bool { return entries[i].rank.before(entries[j].rank) })
	}
	return entries
}

func describeFlags(flags []*pflag.Flag) []flag {
	var result []flag
	for _, f := range flags {
		_, required := f.Annotations[cobra.BashCompOneRequiredFlag]
		result = append(result, flag{Name: f.Name, Type: f.Value.Type(), Description: f.Usage, Default: f.DefValue, Required: required})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// describePage fills each entry's flags with its local public flags and the
// inherited flags the page does not share, then returns the inherited flags
// every entry on the page shares, so a broad listing repeats none of them.
func describePage(entries []entry) []flag {
	if len(entries) == 0 {
		return nil
	}
	inherited := make([][]flag, len(entries))
	for i := range entries {
		local, shared := splitPublicFlags(entries[i].command)
		entries[i].Flags = describeFlags(local)
		inherited[i] = describeFlags(shared)
	}
	// A single entry has nothing to repeat, so it keeps its inherited flags.
	var shared []flag
	if len(entries) > 1 {
		shared = slices.Clone(inherited[0])
	}
	for _, set := range inherited[1:] {
		shared = slices.DeleteFunc(shared, func(f flag) bool { return !slices.Contains(set, f) })
	}
	for i := range entries {
		for _, f := range inherited[i] {
			if !slices.Contains(shared, f) {
				entries[i].Flags = append(entries[i].Flags, f)
			}
		}
		slices.SortFunc(entries[i].Flags, func(a, b flag) int { return strings.Compare(a.Name, b.Name) })
	}
	return shared
}

// splitPublicFlags shares visibility and local-shadowing rules between matching
// and flag output. Hidden flag descriptions must not reveal a match.
func splitPublicFlags(c *cobra.Command) (local, inherited []*pflag.Flag) {
	seen := map[string]bool{}
	visit := func(dst *[]*pflag.Flag) func(*pflag.Flag) {
		return func(f *pflag.Flag) {
			if seen[f.Name] {
				return
			}
			seen[f.Name] = true
			if !f.Hidden {
				*dst = append(*dst, f)
			}
		}
	}
	c.LocalNonPersistentFlags().VisitAll(visit(&local))
	c.PersistentFlags().VisitAll(visit(&local))
	c.InheritedFlags().VisitAll(visit(&inherited))
	return local, inherited
}
