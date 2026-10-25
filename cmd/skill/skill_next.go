// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package skill

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/larksuite/cli/errs"
	"github.com/spf13/cobra"
)

const (
	largeReadBytes       = 8 * 1024
	maxSectionCandidates = 20
)

type heading struct {
	level int
	text  string
	slug  string
	start int
	end   int
}

type sectionOptions struct {
	outline  bool
	sections []string
}

func addSectionFlags(cmd *cobra.Command, o *sectionOptions) {
	cmd.Flags().BoolVar(&o.outline, "outline", false, "print the heading tree (indent, text, [slug], section bytes) instead of the content")
	cmd.Flags().StringSliceVar(&o.sections, "section", nil, "print only these sections by heading slug or text (repeatable or comma-separated)")
}

func (o sectionOptions) apply(content []byte, name, pathOut string) ([]byte, error) {
	if !o.outline && len(o.sections) == 0 {
		return content, nil
	}
	if o.outline && len(o.sections) > 0 {
		return nil, errs.NewValidationError(errs.SubtypeInvalidArgument,
			"--outline and --section cannot be combined").
			WithHint("run --outline first to find slugs, then --section <slug>")
	}
	heads := parseHeadings(content)
	if o.outline {
		return renderOutline(heads), nil
	}
	return renderSections(content, heads, o.sections, name, pathOut)
}

// largeReadTip is stderr-only, so stdout bytes stay identical to the file.
// Reference-file reads stay silent on stderr by contract.
func largeReadTip(w io.Writer, name, pathOut string, content []byte, o sectionOptions) {
	if pathOut != "SKILL.md" || o.outline || len(o.sections) > 0 || len(content) <= largeReadBytes {
		return
	}
	fmt.Fprintf(w, "> Tip: this file is %d KB; list its headings with `feishu-cli-next skills read %s --outline` and print one part with `--section <slug>`.\n",
		len(content)/1024, name)
}

func renderOutline(heads []heading) []byte {
	var b bytes.Buffer
	for _, h := range heads {
		fmt.Fprintf(&b, "%s%s  [%s]  %d B\n", strings.Repeat("  ", h.level-1), h.text, h.slug, h.end-h.start)
	}
	return b.Bytes()
}

func renderSections(content []byte, heads []heading, queries []string, name, pathOut string) ([]byte, error) {
	idx := make([]int, 0, len(queries))
	for _, q := range queries {
		i, err := resolveSection(heads, q, name, pathOut)
		if err != nil {
			return nil, err
		}
		idx = append(idx, i)
	}
	sort.Ints(idx)
	var b bytes.Buffer
	emittedEnd := -1
	for _, i := range idx {
		h := heads[i]
		if h.start < emittedEnd {
			continue
		}
		b.Write(content[h.start:h.end])
		emittedEnd = h.end
	}
	return b.Bytes(), nil
}

func resolveSection(heads []heading, query, name, pathOut string) (int, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "--section must not be empty").
			WithHint("run 'feishu-cli-next skills read %s %s --outline' to list sections", name, pathOut)
	}
	if i := slices.IndexFunc(heads, func(h heading) bool { return h.slug == q }); i >= 0 {
		return i, nil
	}
	found := indexesWhere(heads, func(h heading) bool { return strings.EqualFold(h.text, q) })
	if len(found) == 0 {
		lower := strings.ToLower(q)
		found = indexesWhere(heads, func(h heading) bool { return strings.Contains(strings.ToLower(h.text), lower) })
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		all := indexesWhere(heads, func(heading) bool { return true })
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "section %q not found in %s", q, pathOut).
			WithHint("%s", candidateHint(heads, all, name, pathOut))
	default:
		return 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "section %q is ambiguous in %s", q, pathOut).
			WithHint("%s", candidateHint(heads, found, name, pathOut))
	}
}

func candidateHint(heads []heading, idx []int, name, pathOut string) string {
	if len(heads) == 0 {
		return "the file has no headings; read it without --section"
	}
	var parts []string
	for _, i := range idx[:min(len(idx), maxSectionCandidates)] {
		parts = append(parts, fmt.Sprintf("%s (%s)", heads[i].slug, heads[i].text))
	}
	hint := fmt.Sprintf("use a slug from 'feishu-cli-next skills read %s %s --outline'; candidates: %s",
		name, pathOut, strings.Join(parts, "; "))
	if extra := len(idx) - maxSectionCandidates; extra > 0 {
		hint += fmt.Sprintf("; and %d more", extra)
	}
	return hint
}

func indexesWhere(heads []heading, match func(heading) bool) []int {
	var idx []int
	for i, h := range heads {
		if match(h) {
			idx = append(idx, i)
		}
	}
	return idx
}

// parseHeadings returns ATX headings in document order, skipping YAML
// frontmatter and fenced code blocks. Each section runs to the next heading of
// equal or higher level, so a section includes its subsections.
func parseHeadings(content []byte) []heading {
	var heads []heading
	var fenceChar byte
	fenceLen := 0
	for pos := frontmatterEnd(content); pos < len(content); {
		line, next := nextLine(content, pos)
		if fenceLen > 0 {
			if isFenceClose(line, fenceChar, fenceLen) {
				fenceLen = 0
			}
		} else if c, n := fenceRun(line); n >= 3 {
			fenceChar, fenceLen = c, n
		} else if level, text, ok := parseATX(line); ok {
			heads = append(heads, heading{level: level, text: text, start: pos})
		}
		pos = next
	}
	for i := range heads {
		heads[i].end = len(content)
		for j := i + 1; j < len(heads); j++ {
			if heads[j].level <= heads[i].level {
				heads[i].end = heads[j].start
				break
			}
		}
	}
	used := map[string]bool{}
	for i := range heads {
		base := slugify(heads[i].text)
		s := base
		for n := 2; used[s]; n++ {
			s = base + "-" + strconv.Itoa(n)
		}
		used[s] = true
		heads[i].slug = s
	}
	return heads
}

func frontmatterEnd(content []byte) int {
	first, pos := nextLine(content, 0)
	if first != "---" {
		return 0
	}
	for pos < len(content) {
		line, next := nextLine(content, pos)
		if line == "---" {
			return next
		}
		pos = next
	}
	return 0
}

// nextLine returns the line starting at pos without its line ending, and the
// offset of the following line.
func nextLine(content []byte, pos int) (string, int) {
	end := bytes.IndexByte(content[pos:], '\n')
	if end < 0 {
		return strings.TrimRight(string(content[pos:]), "\r"), len(content)
	}
	return strings.TrimRight(string(content[pos:pos+end]), "\r"), pos + end + 1
}

// fenceRun reports the fence character and run length when line opens a
// backtick or tilde fence (up to three spaces of indentation).
func fenceRun(line string) (byte, int) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 || t == "" || (t[0] != '`' && t[0] != '~') {
		return 0, 0
	}
	return t[0], len(t) - len(strings.TrimLeft(t, t[:1]))
}

func isFenceClose(line string, c byte, n int) bool {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return false
	}
	run := len(t) - len(strings.TrimLeft(t, string(c)))
	return run >= n && strings.TrimSpace(t[run:]) == ""
}

func parseATX(line string) (int, string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return 0, "", false
	}
	level := len(t) - len(strings.TrimLeft(t, "#"))
	if level < 1 || level > 6 {
		return 0, "", false
	}
	rest := t[level:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	text := strings.TrimSpace(rest)
	if strings.HasSuffix(text, "#") {
		if stripped := strings.TrimRight(text, "#"); stripped == "" || strings.HasSuffix(stripped, " ") || strings.HasSuffix(stripped, "\t") {
			text = strings.TrimSpace(stripped)
		}
	}
	return level, text, true
}

// slugify keeps letters and digits (including non-ASCII, such as CJK) and
// turns every other run of characters into a single hyphen.
func slugify(text string) string {
	var b strings.Builder
	dash := true
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.TrimSuffix(b.String(), "-")
	if s == "" {
		return "section"
	}
	return s
}
