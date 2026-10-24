// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recovery

import (
	"regexp"
	"strings"

	"github.com/larksuite/cli/errs"
)

const upstreamCommandName = "lark-cli"

var upstreamCommandNamePattern = regexp.MustCompile(`\b` + upstreamCommandName + `\b`)

// RewriteCommandName replaces the upstream executable name where it names a
// command, so runtime text names the executable this build installs. Paths such
// as ~/.lark-cli, file names such as lark-cli.yml, and identifiers such as
// LARKSUITE_CLI_* are left alone.
func RewriteCommandName(text, name string) string {
	if name == "" || name == upstreamCommandName {
		return text
	}
	var b strings.Builder
	last := 0
	for _, match := range upstreamCommandNamePattern.FindAllStringIndex(text, -1) {
		if !isCommandNameToken(text, match[0], match[1]) {
			continue
		}
		b.WriteString(text[last:match[0]])
		b.WriteString(name)
		last = match[1]
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

func isCommandNameToken(text string, start, end int) bool {
	if start > 0 && strings.ContainsRune("-./~", rune(text[start-1])) {
		return false
	}
	if end < len(text) {
		next := text[end]
		if next == '-' || next == '/' {
			return false
		}
		if next == '.' && end+1 < len(text) && isASCIIAlnum(text[end+1]) {
			return false
		}
	}
	return true
}

func isASCIIAlnum(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// LocalizeError returns a clone of a typed error whose message and hint name
// this build's command. Producers and untyped errors are returned unchanged.
func LocalizeError(err error, name string) error {
	if name == "" || name == upstreamCommandName {
		return err
	}
	original, ok := errs.ProblemOf(err)
	if !ok {
		return err
	}
	message := RewriteCommandName(original.Message, name)
	hint := RewriteCommandName(original.Hint, name)
	if message == original.Message && hint == original.Hint {
		return err
	}
	localized, ok := CloneTyped(err)
	if !ok {
		return err
	}
	problem, ok := errs.ProblemOf(localized)
	if !ok {
		return err
	}
	problem.Message = message
	problem.Hint = hint
	return localized
}
