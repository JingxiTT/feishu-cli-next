// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docworkspace

import (
	"encoding/json"
	"fmt"
	"html"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/larksuite/cli/errs"
)

// RenderReadable exposes more native text and table relationships for read-only
// output. Saved editing views must keep using Render for v1 workspace compatibility.
func RenderReadable(blocks []json.RawMessage) (string, error) {
	projected, err := projectReadable(blocks)
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

func projectReadable(rawBlocks []json.RawMessage) ([]block, error) {
	// Reuse the v1 projection for its supported kinds and original validation.
	// This leaves saved views and Plan entirely independent of readable additions.
	out, err := project(rawBlocks)
	if err != nil {
		return nil, err
	}
	nodes, labels, err := readableStructure(rawBlocks)
	if err != nil {
		return nil, err
	}
	for i, node := range nodes {
		if node.Type < 2 || node.Type > 8 {
			key, prefix, suffix, _, code, err := readableText(node)
			if err != nil {
				return nil, err
			}
			if key != "" {
				var fields map[string]json.RawMessage
				_ = json.Unmarshal(node.raw, &fields)
				var text struct {
					Elements []json.RawMessage `json:"elements"`
				}
				if json.Unmarshal(fields[key], &text) != nil || text.Elements == nil {
					return nil, malformedNative("native text block %s is missing its element list", node.ID)
				}
				var body strings.Builder
				body.WriteString(prefix)
				for _, element := range text.Elements {
					content, before, after, _, readable := textRun(element)
					out[i].texts = append(out[i].texts, content)
					if !readable {
						body.WriteString("[Opaque native inline element; preserved unchanged]")
					} else if code {
						body.WriteString(escapeCode(content))
					} else {
						body.WriteString(before + escapeText(content) + after)
					}
				}
				body.WriteString(suffix + "\n\n")
				out[i].body = body.String()
			}
		}
		if node.Type == 31 || node.Type == 32 {
			out[i].body = labels[node.ID]
		} else {
			out[i].body = labels[node.ID] + out[i].body
		}
		// This projection never supplies a writable plan, including contextualized
		// paragraph bodies; mutation callers must use the original project instead.
		out[i].writable = false
	}
	return out, nil
}

// These fields/types follow oapi-sdk-go/v3 v3.7.2 service/docx/v1 Block,
// Text, TextStyle and Table. Raw blocks remain untouched in the sidecar.
type readableBlock struct {
	ID       string          `json:"block_id"`
	Type     int             `json:"block_type"`
	Parent   string          `json:"parent_id"`
	Children []string        `json:"children"`
	Table    json.RawMessage `json:"table"`
	Cell     json.RawMessage `json:"table_cell"`
	raw      json.RawMessage
}

type readableTable struct {
	Cells    []string `json:"cells"`
	Property *struct {
		Rows    *int            `json:"row_size"`
		Columns *int            `json:"column_size"`
		Merge   json.RawMessage `json:"merge_info"`
	} `json:"property"`
}

func malformedNative(format string, args ...any) error {
	return errs.NewInternalError(errs.SubtypeInvalidResponse, format, args...)
}

func readableStructure(rawBlocks []json.RawMessage) ([]readableBlock, map[string]string, error) {
	nodes := make([]readableBlock, len(rawBlocks))
	byID := make(map[string]readableBlock, len(rawBlocks))
	for i, raw := range rawBlocks {
		var node readableBlock
		if !utf8.Valid(raw) || json.Unmarshal(raw, &node) != nil || !idPattern.MatchString(node.ID) || node.Type < 1 {
			return nil, nil, malformedNative("native blocks require unique safe IDs and valid types")
		}
		if _, exists := byID[node.ID]; exists {
			return nil, nil, malformedNative("native blocks require unique safe IDs and valid types")
		}
		node.raw = raw
		nodes[i], byID[node.ID] = node, node
	}
	labels := map[string]string{}
	contexts := map[string]string{}
	cellOwners := map[string]string{}
	for _, node := range nodes {
		if node.Type == 12 || node.Type == 13 || node.Type == 15 || node.Type == 17 {
			if node.Parent != "" {
				if !idPattern.MatchString(node.Parent) {
					return nil, nil, malformedNative("native list/quote %s has an unsafe parent ID", node.ID)
				}
				labels[node.ID] = fmt.Sprintf("> Native parent: `%s`.\n\n", node.Parent)
			}
			seen := map[string]bool{}
			for _, id := range node.Children {
				child, exists := byID[id]
				if !exists || seen[id] || id == node.ID || (child.Parent != "" && child.Parent != node.ID) {
					return nil, nil, malformedNative("native list/quote %s has inconsistent children", node.ID)
				}
				seen[id] = true
			}
			if len(node.Children) > 0 {
				labels[node.ID] += "> Child blocks: " + readableIDs(node.Children) + ".\n\n"
			}
		}
		if node.Type != 31 {
			continue
		}
		var table readableTable
		if json.Unmarshal(node.Table, &table) != nil || table.Property == nil || table.Property.Rows == nil || table.Property.Columns == nil || *table.Property.Rows < 1 || *table.Property.Columns < 1 || len(table.Cells) == 0 {
			return nil, nil, malformedNative("native table %s requires dimensions and cell IDs", node.ID)
		}
		var merges []json.RawMessage
		if len(table.Property.Merge) > 0 && json.Unmarshal(table.Property.Merge, &merges) != nil {
			return nil, nil, malformedNative("native table %s has invalid merge metadata", node.ID)
		}
		if len(merges) == 0 && (len(table.Cells)%*table.Property.Columns != 0 || len(table.Cells) / *table.Property.Columns != *table.Property.Rows) {
			return nil, nil, malformedNative("native table %s dimensions do not match its unmerged cells", node.ID)
		}
		// Native cell order is displayed as supplied. Do not infer coordinates or
		// merged geometry: the pinned SDK does not specify that mapping.
		labels[node.ID] = fmt.Sprintf("> Table: %d rows × %d columns; cells in native order: %s. Structure is read-only.\n\n", *table.Property.Rows, *table.Property.Columns, readableIDs(table.Cells))
		if len(merges) > 0 {
			labels[node.ID] += "> [Native cell merging preserved; layout is not flattened]\n\n"
		}
		for index, id := range table.Cells {
			cell, exists := byID[id]
			if !exists || cell.Type != 32 || cellOwners[id] != "" || (cell.Parent != "" && cell.Parent != node.ID) {
				return nil, nil, malformedNative("native table %s has missing, duplicate or inconsistent cells", node.ID)
			}
			var cellFields map[string]json.RawMessage
			if json.Unmarshal(cell.Cell, &cellFields) != nil || cellFields == nil {
				return nil, nil, malformedNative("native table cell %s is missing its cell structure", id)
			}
			cellOwners[id] = node.ID
			context := fmt.Sprintf("> Table `%s`, cell `%s` (native position %d).\n\n", node.ID, id, index+1)
			labels[id] = context
			if len(cell.Children) == 0 {
				labels[id] += "> [Empty native cell]\n\n"
			} else {
				labels[id] += "> Content blocks: " + readableIDs(cell.Children) + ".\n\n"
			}
			seen := map[string]bool{}
			for _, childID := range cell.Children {
				child, exists := byID[childID]
				if !exists || seen[childID] || childID == id || childID == node.ID || (child.Parent != "" && child.Parent != id) {
					return nil, nil, malformedNative("native table cell %s has missing, duplicate or inconsistent content blocks", id)
				}
				seen[childID] = true
				if contexts[childID] != "" {
					return nil, nil, malformedNative("native table cell %s shares content with another cell", id)
				}
				contexts[childID] = context
			}
		}
		if node.Children != nil {
			seen := map[string]bool{}
			for _, id := range node.Children {
				if cellOwners[id] != node.ID || seen[id] {
					return nil, nil, malformedNative("native table %s has inconsistent child cells", node.ID)
				}
				seen[id] = true
			}
			if len(seen) != len(table.Cells) {
				return nil, nil, malformedNative("native table %s has incomplete child cells", node.ID)
			}
		}
	}
	for _, node := range nodes {
		if node.Type == 32 && cellOwners[node.ID] == "" {
			return nil, nil, malformedNative("native table cell %s has no containing table", node.ID)
		}
		// Forward edges alone cannot justify an empty or complete container:
		// a child may name that parent while being absent from its child list.
		parent := byID[node.Parent]
		switch parent.Type {
		case 31:
			if cellOwners[node.ID] != parent.ID {
				return nil, nil, malformedNative("native table %s does not list child %s", parent.ID, node.ID)
			}
		case 12, 13, 15, 17, 32:
			if !slices.Contains(parent.Children, node.ID) {
				return nil, nil, malformedNative("native container %s does not list child %s", parent.ID, node.ID)
			}
		}
	}
	for id, context := range contexts {
		labels[id] = context + labels[id]
	}
	return nodes, labels, nil
}

func readableIDs(ids []string) string {
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = "`" + id + "`"
	}
	return strings.Join(quoted, ", ")
}

// Only the previously supported paragraph/heading1–6 kinds are editable.
// Additional native kinds expose their text without granting mutation support.
func readableText(node readableBlock) (key, prefix, suffix string, writable, code bool, err error) {
	if node.Type == 2 {
		return "text", "", "", true, false, nil
	}
	if node.Type >= 3 && node.Type <= 11 {
		level := node.Type - 2
		key = "heading" + strconv.Itoa(level)
		if level <= 6 {
			return key, strings.Repeat("#", level) + " ", "", true, false, nil
		}
		return key, fmt.Sprintf("###### [Native heading %d] ", level), "", false, false, nil
	}
	switch node.Type {
	case 12:
		return "bullet", "- ", "", false, false, nil
	case 13:
		key, prefix = "ordered", "- [Ordered; sequence unavailable] "
	case 14:
		key, prefix, suffix, code = "code", "<pre><code>", "</code></pre>", true
	case 15:
		return "quote", "> ", "", false, false, nil
	case 17:
		key, prefix = "todo", "- [Task; state unavailable] "
	default:
		return
	}
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(node.raw, &fields)
	var text struct {
		Style *struct {
			Done     *bool   `json:"done"`
			Sequence *string `json:"sequence"`
			Language *int    `json:"language"`
		} `json:"style"`
	}
	if json.Unmarshal(fields[key], &text) != nil {
		err = malformedNative("native %s block %s has invalid text/style fields", key, node.ID)
		return
	}
	if text.Style == nil {
		return
	}
	switch node.Type {
	case 13:
		if text.Style.Sequence != nil {
			sequence := *text.Style.Sequence
			if number, parseErr := strconv.Atoi(sequence); parseErr == nil && number > 0 {
				prefix = sequence + ". "
			} else {
				prefix = "- [Ordered; native sequence: " + escapeText(sequence) + "] "
			}
		}
	case 14:
		if text.Style.Language != nil {
			prefix = fmt.Sprintf("> Code language (native ID): %d\n\n", *text.Style.Language) + prefix
		}
	case 17:
		if text.Style.Done != nil {
			prefix = "- [ ] "
			if *text.Style.Done {
				prefix = "- [x] "
			}
		}
	}
	return
}

func escapeCode(content string) string {
	// HTML code preserves physical lines without creating Markdown block markers
	// or interpreting native literal tags. Keep native CR distinct from editor CRLF.
	return strings.ReplaceAll(html.EscapeString(content), "\r", "&#13;")
}
