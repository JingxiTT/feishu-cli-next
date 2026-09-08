// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package docworkspace projects native blocks into a deliberately limited,
// editable Markdown view. It never reconstructs a document from Markdown.
package docworkspace

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/larksuite/cli/errs"
)

// Edit updates one existing block's complete text element list. Callers must
// enforce identity, known-state comparisons, and post-write verification.
type Edit struct {
	BlockID  string            `json:"block_id"`
	Elements []json.RawMessage `json:"elements"`
	Text     string            `json:"text"`
}

type span struct {
	start, end, element int
}

type block struct {
	id       string
	body     string
	elements []json.RawMessage
	spans    []span
	texts    []string
	writable bool
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var markerPattern = regexp.MustCompile(`(?m)^<!-- block:([A-Za-z0-9_-]+) -->\n`)

func marker(id string) string { return "<!-- block:" + id + " -->\n" }

// Render includes stable native IDs and visibly opaque regions. The caller
// keeps all original block JSON in its sidecar, including fields not displayed.
func Render(blocks []json.RawMessage) (string, error) {
	projected, err := project(blocks)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	for _, b := range projected {
		out.WriteString(marker(b.id))
		out.WriteString(b.body)
	}
	return out.String(), nil
}

// Plan accepts only one contiguous change per block, wholly inside one native
// text run. Changes to Markdown syntax, block markers, opaque data or ambiguous
// run boundaries fail before returning any edits. No API calls are performed.
func Plan(blocks []json.RawMessage, editedMarkdown string) ([]Edit, error) {
	if !utf8.ValidString(editedMarkdown) {
		return nil, invalid("Markdown must be valid UTF-8")
	}
	projected, err := project(blocks)
	if err != nil {
		return nil, err
	}
	matches := markerPattern.FindAllStringSubmatchIndex(editedMarkdown, -1)
	if len(matches) != len(projected) || (len(matches) > 0 && matches[0][0] != 0) || (len(matches) == 0 && editedMarkdown != "") {
		return nil, invalid("block markers, order and block count must remain unchanged")
	}
	edits := []Edit{}
	for i, b := range projected {
		m := matches[i]
		if editedMarkdown[m[2]:m[3]] != b.id {
			return nil, invalid("block markers and order must remain unchanged")
		}
		end := len(editedMarkdown)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := editedMarkdown[m[1]:end]
		if body == b.body {
			continue
		}
		if !b.writable {
			return nil, invalid("block %s contains unsupported native elements or fields; preserve it unchanged", b.id)
		}
		start, oldEnd, newEnd := changeBounds(b.body, body)
		var candidates []span
		for _, s := range b.spans {
			if start >= s.start && oldEnd <= s.end {
				candidates = append(candidates, s)
			}
		}
		if len(candidates) != 1 {
			return nil, invalid("block %s edit crosses formatting or an ambiguous text boundary", b.id)
		}
		s := candidates[0]
		encoded := b.body[s.start:start] + body[start:newEnd] + b.body[oldEnd:s.end]
		content, ok := unescapeText(encoded)
		if !ok {
			return nil, invalid("block %s edit changes Markdown formatting; only text edits are supported", b.id)
		}
		elements := make([]json.RawMessage, len(b.elements))
		for j, raw := range b.elements {
			elements[j] = append(json.RawMessage(nil), raw...)
		}
		var element, run map[string]json.RawMessage
		_ = json.Unmarshal(elements[s.element], &element)
		_ = json.Unmarshal(element["text_run"], &run)
		run["content"], _ = json.Marshal(content)
		element["text_run"], _ = json.Marshal(run)
		elements[s.element], _ = json.Marshal(element)
		texts := append([]string(nil), b.texts...)
		texts[s.element] = content
		edits = append(edits, Edit{BlockID: b.id, Elements: elements, Text: strings.Join(texts, "")})
	}
	return edits, nil
}

func invalid(format string, args ...any) error {
	return errs.NewValidationError(errs.SubtypeInvalidArgument, format, args...).WithParam("--file").
		WithHint("keep block markers and formatting unchanged; use the original Docs commands for structural or style edits")
}

func project(rawBlocks []json.RawMessage) ([]block, error) {
	out := make([]block, 0, len(rawBlocks))
	seen := map[string]bool{}
	for _, raw := range rawBlocks {
		var native struct {
			ID   string `json:"block_id"`
			Type int    `json:"block_type"`
		}
		if !utf8.Valid(raw) || json.Unmarshal(raw, &native) != nil || !idPattern.MatchString(native.ID) || native.Type < 1 || seen[native.ID] {
			return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "native blocks require unique safe IDs and valid types")
		}
		seen[native.ID] = true
		b := block{id: native.ID, body: fmt.Sprintf("> [Opaque native block type %d; preserved unchanged]\n\n", native.Type)}
		key, prefix := "", ""
		if native.Type == 2 {
			key = "text"
		} else if native.Type >= 3 && native.Type <= 11 {
			level := native.Type - 2
			key = "heading" + strconv.Itoa(level)
			// Markdown has six heading levels. Keep deeper headings opaque.
			if level <= 6 {
				prefix = strings.Repeat("#", level) + " "
			} else {
				key = ""
			}
		}
		if key != "" {
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(raw, &fields)
			var text struct {
				Elements []json.RawMessage `json:"elements"`
			}
			if json.Unmarshal(fields[key], &text) != nil || text.Elements == nil {
				return nil, errs.NewInternalError(errs.SubtypeInvalidResponse, "native text block %s is missing its element list", native.ID)
			}
			b.elements = text.Elements
			b.writable = len(text.Elements) > 0
			var rendered strings.Builder
			rendered.WriteString(prefix)
			for i, rawElement := range text.Elements {
				content, before, after, writable, readable := textRun(rawElement)
				b.texts = append(b.texts, content)
				if !readable {
					b.writable = false
					rendered.WriteString("[Opaque native inline element; preserved unchanged]")
					continue
				}
				b.writable = b.writable && writable
				rendered.WriteString(before)
				start := rendered.Len()
				rendered.WriteString(escapeText(content))
				b.spans = append(b.spans, span{start, rendered.Len(), i})
				rendered.WriteString(after)
			}
			rendered.WriteString("\n\n")
			b.body = rendered.String()
		}
		out = append(out, b)
	}
	return out, nil
}

// Writable fields follow the pinned native SDK's TextRun/TextElementStyle/Link
// request types. Other elements and unknown fields are preserved in the snapshot
// but not replayed. comment_ids is deliberately excluded: it is not proof of a
// writable comment attachment. Block-level fields are never sent by this plan.
func textRun(raw json.RawMessage) (content, before, after string, writable, readable bool) {
	var element, run, style map[string]json.RawMessage
	if json.Unmarshal(raw, &element) != nil || json.Unmarshal(element["text_run"], &run) != nil {
		return
	}
	if string(run["content"]) == "null" || json.Unmarshal(run["content"], &content) != nil || !utf8.ValidString(content) {
		return
	}
	readable = true
	writable = len(element) == 1 && onlyKeys(run, "content", "text_element_style")
	if rawStyle, exists := run["text_element_style"]; exists {
		if json.Unmarshal(rawStyle, &style) != nil || style == nil {
			return content, "", "", false, true
		}
	}
	for key, value := range style {
		switch key {
		case "bold", "italic", "strikethrough", "underline", "inline_code":
			var enabled bool
			if string(value) == "null" || json.Unmarshal(value, &enabled) != nil {
				writable = false
				continue
			}
		case "background_color", "text_color":
			var color int
			if string(value) == "null" || json.Unmarshal(value, &color) != nil {
				writable = false
			}
		case "link":
			var link map[string]json.RawMessage
			var target string
			if json.Unmarshal(value, &link) != nil || !onlyKeys(link, "url") || json.Unmarshal(link["url"], &target) != nil {
				writable = false
				continue
			}
			// Angle destinations avoid bracket/parenthesis ambiguity. Whitespace,
			// controls and angle brackets cannot safely render as a Markdown URL.
			if strings.ContainsAny(target, "<>\\ ") || strings.IndexFunc(target, unicode.IsControl) >= 0 || target == "" {
				writable = false
				continue
			}
			before, after = "[", "](<"+target+">)"
		default:
			writable = false
		}
	}
	for _, mark := range []struct{ key, open, close string }{{"bold", "**", "**"}, {"italic", "_", "_"}, {"strikethrough", "~~", "~~"}, {"underline", "<u>", "</u>"}, {"inline_code", "<code>", "</code>"}} {
		if string(style[mark.key]) == "true" {
			before, after = mark.open+before, after+mark.close
		}
	}
	return
}

func onlyKeys(value map[string]json.RawMessage, names ...string) bool {
	for key := range value {
		found := false
		for _, name := range names {
			found = found || key == name
		}
		if !found {
			return false
		}
	}
	return true
}

const escapedPunctuation = "\\`*_{}[]<>()#+-.!|~&"

func escapeText(text string) string {
	var out strings.Builder
	for _, r := range text {
		switch r {
		case '\n':
			out.WriteString("<br>")
		case '\r':
			out.WriteString("&#13;")
		default:
			if strings.ContainsRune(escapedPunctuation, r) {
				out.WriteByte('\\')
			}
			out.WriteRune(r)
		}
	}
	return out.String()
}

func unescapeText(encoded string) (string, bool) {
	var out strings.Builder
	for len(encoded) > 0 {
		switch {
		case strings.HasPrefix(encoded, "<br>"):
			out.WriteByte('\n')
			encoded = encoded[4:]
		case strings.HasPrefix(encoded, "&#13;"):
			out.WriteByte('\r')
			encoded = encoded[5:]
		case encoded[0] == '\\':
			if len(encoded) < 2 || !strings.ContainsRune(escapedPunctuation, rune(encoded[1])) {
				return "", false
			}
			out.WriteByte(encoded[1])
			encoded = encoded[2:]
		default:
			r, size := utf8.DecodeRuneInString(encoded)
			if strings.ContainsRune(escapedPunctuation, r) || r == '\r' || r == '\n' {
				return "", false
			}
			out.WriteRune(r)
			encoded = encoded[size:]
		}
	}
	return out.String(), true
}

func changeBounds(old, new string) (start, oldEnd, newEnd int) {
	a, b := []rune(old), []rune(new)
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	j, k := len(a), len(b)
	for j > i && k > i && a[j-1] == b[k-1] {
		j--
		k--
	}
	return len(string(a[:i])), len(string(a[:j])), len(string(b[:k]))
}
