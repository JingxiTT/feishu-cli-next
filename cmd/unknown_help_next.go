// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"slices"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/larksuite/cli/cmd/discover"
	"github.com/larksuite/cli/internal/cmdpolicy"
	"github.com/larksuite/cli/internal/suggest"
)

const unknownSubcommandSuggestionLimit = 6

// executeRejectingUnknownHelp runs root, except that `<group> <unknown> --help`
// fails as an unknown subcommand. Cobra handles --help before RunE, so it would
// print the group's help and exit 0 for a command that does not exist.
func executeRejectingUnknownHelp(root *cobra.Command, args []string) error {
	if !slices.Contains(args, "--help") && !slices.Contains(args, "-h") {
		return root.Execute()
	}
	group, rest, err := root.Find(args)
	if err != nil || group.Annotations[cmdpolicy.AnnotationPureGroup] != "true" {
		return root.Execute()
	}
	if name := firstPositional(group, rest); name != "" {
		return unknownSubcommandRunE(group, []string{name})
	}
	return root.Execute()
}

// firstPositional skips flags, consuming the value of a value-taking long flag,
// so `sheets --profile dev +read` yields "+read".
func firstPositional(group *cobra.Command, args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		case strings.HasPrefix(arg, "--") && !strings.Contains(arg, "="):
			if f := group.Flags().Lookup(strings.TrimPrefix(arg, "--")); f != nil && f.NoOptDefVal == "" {
				i++
			}
		case strings.HasPrefix(arg, "-"):
		default:
			return arg
		}
	}
	return ""
}

// unknownSubcommandSuggestions returns the group's children whose words match
// the unknown name first, then edit-distance hits, deduplicated and capped. A
// keyword hit names what the command does; a near spelling may not.
func unknownSubcommandSuggestions(group *cobra.Command, unknown string, names []string) []string {
	var suggestions []string
	seen := map[string]bool{}
	add := func(candidates []string) {
		for _, name := range candidates {
			if len(suggestions) < unknownSubcommandSuggestionLimit && !seen[name] {
				seen[name] = true
				suggestions = append(suggestions, name)
			}
		}
	}
	if terms := keywordTerms(unknown); len(terms) > 0 {
		add(discover.MatchChildren(group, terms))
	}
	add(suggest.Closest(unknown, names, unknownSubcommandSuggestionLimit))
	return suggestions
}

func keywordTerms(unknown string) []string {
	return strings.FieldsFunc(strings.ToLower(unknown), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
