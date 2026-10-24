// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package discover

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Match locations, weakest evidence last. A command's tier is the weakest
// location any of its terms needed, so a term found only in an inherited flag
// demotes the command below every command that matched without it.
const (
	matchPath = iota
	matchSummary
	matchSubstring
	matchLocalFlag
	matchInheritedFlag
)

type rank struct {
	tier     int
	exact    int
	shortcut bool
	segments int
	length   int
	command  string
}

// before orders results deterministically: tier, exact segment matches,
// shortcuts, shallower paths, shorter paths, then command path.
func (r rank) before(o rank) bool {
	if r.tier != o.tier {
		return r.tier < o.tier
	}
	if r.exact != o.exact {
		return r.exact > o.exact
	}
	if r.shortcut != o.shortcut {
		return r.shortcut
	}
	if r.segments != o.segments {
		return r.segments < o.segments
	}
	if r.length != o.length {
		return r.length < o.length
	}
	return r.command < o.command
}

// matchRank reports whether every term appears somewhere in the command's
// searchable text and returns where it appeared. The match set is the same as
// a per-term substring test over path, summary, and public flag text.
func matchRank(c *cobra.Command, command string, terms []string) (rank, bool) {
	path := strings.ToLower(command)
	summary := path + " " + strings.ToLower(c.Short+" "+strings.Join(c.Aliases, " "))
	local, inherited := splitPublicFlags(c)
	localText := flagSearchText(local)
	inheritedText := flagSearchText(inherited)
	segments := strings.Fields(path)
	leaf := segments[len(segments)-1]
	r := rank{shortcut: strings.HasPrefix(leaf, "+"), segments: len(segments), length: len(command), command: command}
	for _, term := range terms {
		if term == "*" {
			continue
		}
		switch {
		case startsWord(path, term):
		case startsWord(summary, term):
			r.tier = max(r.tier, matchSummary)
		case strings.Contains(summary, term):
			r.tier = max(r.tier, matchSubstring)
		case strings.Contains(localText, term):
			r.tier = max(r.tier, matchLocalFlag)
		case strings.Contains(inheritedText, term):
			r.tier = max(r.tier, matchInheritedFlag)
		default:
			return rank{}, false
		}
		if hasExactSegment(segments, term) {
			r.exact++
		}
	}
	return r, true
}

// startsWord reports whether term begins a word, so "read" does not count as a
// word match inside "spreadsheet". Substring-only matches still count.
func startsWord(text, term string) bool {
	for i := range text {
		if (i == 0 || strings.ContainsRune(" +-_./", rune(text[i-1]))) && strings.HasPrefix(text[i:], term) {
			return true
		}
	}
	return false
}

func hasExactSegment(segments []string, term string) bool {
	for _, segment := range segments {
		if strings.TrimPrefix(segment, "+") == term {
			return true
		}
	}
	return false
}

func flagSearchText(flags []*pflag.Flag) string {
	var searchable strings.Builder
	for _, f := range flags {
		searchable.WriteString(" --" + f.Name + " " + f.Usage)
	}
	return strings.ToLower(searchable.String())
}
