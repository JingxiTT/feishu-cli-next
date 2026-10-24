// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package discover provides bounded discovery from the executable command tree.
package discover

import (
	"fmt"
	"sort"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdmeta"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/meta"
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
	Example     string   `json:"example,omitempty"`
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
			if withFlags {
				for i := start; i < end; i++ {
					entries[i].Flags = describeFlags(publicFlags(entries[i].command))
					entries[i].Example = routingExample(root.Name(), f, entries[i])
				}
			}
			page := map[string]any{"query": query, "offset": offset, "limit": limit, "total": len(entries), "complete": end == len(entries)}
			if end < len(entries) {
				page["next_offset"] = end
			}
			result := entries[start:end]
			return output.WriteJSON(f.IOStreams.Out, map[string]any{"ok": true, "data": result, "meta": page})
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

// routingExample gives --flags one compact invocation route. Authored
// examples win; otherwise required flags form a placeholder skeleton. Full
// help remains the source for alternate workflows and complex semantics.
func routingExample(rootName string, f *cmdutil.Factory, result entry) string {
	if result.command == nil || (result.command.Run == nil && result.command.RunE == nil) {
		return ""
	}
	if example := firstCommandExample(rootName, guidanceExamples(f, result.command), result.command.Example); example != "" {
		return example
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", rootName, result.Command)
	required := false
	for _, flagResult := range result.Flags {
		if !flagResult.Required {
			continue
		}
		required = true
		fmt.Fprintf(&b, " --%s", flagResult.Name)
		if flagResult.Type != "bool" {
			placeholder := strings.ReplaceAll(flagResult.Name, "_", "-")
			fmt.Fprintf(&b, " <%s>", placeholder)
		}
	}
	if required {
		return b.String()
	}
	return ""
}

func guidanceExamples(f *cmdutil.Factory, cmd *cobra.Command) []string {
	service, method, ok := cmdmeta.AffordanceRef(cmd)
	if !ok || f == nil || f.Affordance == nil {
		return nil
	}
	raw, ok := f.Affordance.For(service, method)
	if !ok {
		return nil
	}
	parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
	if !ok {
		return nil
	}
	examples := make([]string, 0, len(parsed.Examples))
	for _, example := range parsed.Examples {
		examples = append(examples, example.Command)
	}
	return examples
}

func firstCommandExample(rootName string, guidance []string, helpExamples string) string {
	lines := append([]string(nil), guidance...)
	lines = append(lines, strings.Split(helpExamples, "\n")...)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		for _, binary := range []string{"lark-cli", "feishu-cli-next"} {
			if strings.HasPrefix(line, binary+" ") {
				return rootName + strings.TrimPrefix(line, binary)
			}
		}
	}
	return ""
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
			haystack := strings.ToLower(path + " " + c.Short + " " + strings.Join(c.Aliases, " "))
			match := true
			for _, term := range terms {
				if term != "*" && !strings.Contains(haystack, term) {
					match = false
					break
				}
			}
			if !match {
				var searchable strings.Builder
				for _, f := range publicFlags(c) {
					searchable.WriteString(" --" + f.Name + " " + f.Usage)
				}
				flagText := strings.ToLower(searchable.String())
				match = true
				for _, term := range terms {
					if term != "*" && !strings.Contains(haystack, term) && !strings.Contains(flagText, term) {
						match = false
						break
					}
				}
			}
			if match {
				risk, _ := cmdmeta.Risk(c)
				entries = append(entries, entry{Command: path, Description: c.Short, Risk: risk, Identities: cmdmeta.Identities(c), command: c})
			}
		}
		for _, child := range c.Commands() {
			walk(child, commandPath+" "+child.Name())
		}
	}
	walk(root, root.CommandPath())
	sort.Slice(entries, func(i, j int) bool { return entries[i].Command < entries[j].Command })
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

// publicFlags shares visibility and local-shadowing rules between matching and
// optional flag output. Hidden flag descriptions must not reveal a match.
func publicFlags(c *cobra.Command) []*pflag.Flag {
	flags := []*pflag.Flag{}
	seen := map[string]bool{}
	visit := func(f *pflag.Flag) {
		if seen[f.Name] {
			return
		}
		seen[f.Name] = true
		if !f.Hidden {
			flags = append(flags, f)
		}
	}
	c.LocalNonPersistentFlags().VisitAll(visit)
	c.PersistentFlags().VisitAll(visit)
	c.InheritedFlags().VisitAll(visit)
	return flags
}
