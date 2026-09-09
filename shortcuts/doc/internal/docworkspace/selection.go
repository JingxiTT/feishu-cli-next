// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docworkspace

import (
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/errs"
)

// Selection narrows a read-only view. It never makes an editable partial workspace.
type Selection struct {
	Scope   string `json:"scope"`
	StartID string `json:"start_block_id,omitempty"`
	EndID   string `json:"end_block_id,omitempty"`
	Keyword string `json:"keyword,omitempty"`
}

func (s Selection) Validate() error {
	bad := func(param, message string) error {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, message).WithParam(param)
	}
	switch s.Scope {
	case "", "full", "outline":
		if s.StartID != "" || s.EndID != "" || s.Keyword != "" {
			return bad("--scope", "selection bounds require their matching range, section or keyword scope")
		}
	case "range":
		if s.Keyword != "" || (s.StartID == "" && s.EndID == "") {
			return bad("--scope", "range requires a start or end block ID and no keyword")
		}
	case "section":
		if s.StartID == "" || s.EndID != "" || s.Keyword != "" {
			return bad("--start-block-id", "section requires a heading block ID and no end block or keyword")
		}
	case "keyword":
		if strings.TrimSpace(s.Keyword) == "" || s.StartID != "" || s.EndID != "" {
			return bad("--keyword", "keyword requires nonblank literal text and no block bounds")
		}
	default:
		return bad("--scope", "unsupported document selection scope")
	}
	return nil
}

// Select renders against the complete native snapshot before selecting blocks,
// retaining every selected native anchor. Sections/ranges follow native block
// order; keyword matching is literal, case-insensitive, and block-local.
func Select(blocks []json.RawMessage, s Selection) (string, int, error) {
	if err := s.Validate(); err != nil {
		return "", 0, err
	}
	projected, err := projectReadable(blocks)
	if err != nil {
		return "", 0, err
	}
	if s.Scope != "" && s.Scope != "full" {
		order, orderErr := selectionOrder(blocks)
		if orderErr != nil {
			return "", 0, orderErr
		}
		orderedBlocks, orderedProjection := make([]json.RawMessage, len(blocks)), make([]block, len(blocks))
		for i, index := range order {
			orderedBlocks[i], orderedProjection[i] = blocks[index], projected[index]
		}
		blocks, projected = orderedBlocks, orderedProjection
	}
	levels := make([]int, len(blocks))
	ids := make(map[string]int, len(blocks))
	for i, raw := range blocks {
		var native struct {
			Type int `json:"block_type"`
		}
		_ = json.Unmarshal(raw, &native) // project has validated every native block.
		if native.Type >= 3 && native.Type <= 11 {
			levels[i] = native.Type - 2
		}
		ids[projected[i].id] = i
	}
	start, end := 0, len(blocks)
	for _, bound := range []struct {
		id, param string
		start     bool
	}{{s.StartID, "--start-block-id", true}, {s.EndID, "--end-block-id", false}} {
		if bound.id == "" {
			continue
		}
		i, ok := ids[bound.id]
		if !ok {
			return "", 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "block ID is absent from this snapshot").WithParam(bound.param)
		}
		if bound.start {
			start = i
		} else {
			end = i + 1
		}
	}
	if s.Scope == "section" {
		if levels[start] == 0 {
			return "", 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "section anchor must be a heading block").WithParam("--start-block-id")
		}
		for i := start + 1; i < end; i++ {
			if levels[i] > 0 && levels[i] <= levels[start] {
				end = i
				break
			}
		}
	}
	if start >= end && len(blocks) > 0 {
		return "", 0, errs.NewValidationError(errs.SubtypeInvalidArgument, "range start follows its end").WithParam("--end-block-id")
	}
	var out strings.Builder
	count := 0
	for i := start; i < end; i++ {
		b := projected[i]
		if s.Scope == "outline" && levels[i] == 0 {
			continue
		}
		if s.Scope == "keyword" && !strings.Contains(strings.ToLower(strings.Join(b.texts, "")), strings.ToLower(s.Keyword)) {
			continue
		}
		out.WriteString(marker(b.id))
		out.WriteString(b.body)
		count++
	}
	return out.String(), count, nil
}

// Native list responses need not be in document order. Bounded reads follow the
// page's ordered child links and reject incomplete or ambiguous hierarchies.
func selectionOrder(blocks []json.RawMessage) ([]int, error) {
	type node struct {
		ID       string   `json:"block_id"`
		Parent   string   `json:"parent_id"`
		Type     int      `json:"block_type"`
		Children []string `json:"children"`
	}
	nodes := make([]node, len(blocks))
	ids := make(map[string]int, len(blocks))
	root := -1
	bad := func() error {
		return errs.NewInternalError(errs.SubtypeInvalidResponse, "bounded Markdown read requires a complete, unambiguous native block hierarchy").WithHint("Use docs +fetch for a server-scoped read; no content was changed.")
	}
	for i, raw := range blocks {
		if json.Unmarshal(raw, &nodes[i]) != nil {
			return nil, bad()
		}
		ids[nodes[i].ID] = i
		if nodes[i].Type == 1 {
			if root != -1 {
				return nil, bad()
			}
			root = i
		}
	}
	if root == -1 {
		return nil, bad()
	}
	order, stack := make([]int, 0, len(nodes)), []int{root}
	seen := make([]bool, len(nodes))
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[i] {
			return nil, bad()
		}
		seen[i] = true
		order = append(order, i)
		for j := len(nodes[i].Children) - 1; j >= 0; j-- {
			child, ok := ids[nodes[i].Children[j]]
			if !ok || nodes[child].Parent != nodes[i].ID {
				return nil, bad()
			}
			stack = append(stack, child)
			if len(stack) > len(nodes) {
				return nil, bad()
			}
		}
	}
	if len(order) != len(nodes) {
		return nil, bad()
	}
	return order, nil
}
