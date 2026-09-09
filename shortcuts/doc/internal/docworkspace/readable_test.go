// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package docworkspace

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func readView(t *testing.T, blocks []json.RawMessage) string {
	t.Helper()
	value, err := RenderReadable(blocks)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestReadableNativeKindsPreserveFactsAndLegacyEditingView(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"block_id":"bullet","block_type":12,"bullet":{"elements":[{"text_run":{"content":"List fact","text_element_style":{"bold":true}}}]}}`),
		json.RawMessage(`{"block_id":"ordered","block_type":13,"ordered":{"style":{"sequence":"4"},"elements":[{"text_run":{"content":"Fourth fact"}}]}}`),
		json.RawMessage(`{"block_id":"auto","block_type":13,"ordered":{"style":{"sequence":"auto"},"elements":[{"text_run":{"content":"Automatic fact"}}]}}`),
		json.RawMessage(`{"block_id":"done","block_type":17,"todo":{"style":{"done":true},"elements":[{"text_run":{"content":"Completed fact"}}]}}`),
		json.RawMessage(`{"block_id":"open","block_type":17,"todo":{"style":{"done":false},"elements":[{"text_run":{"content":"Open fact"}}]}}`),
		json.RawMessage(`{"block_id":"unknown","block_type":17,"todo":{"elements":[{"text_run":{"content":"Unknown fact"}}]}}`),
		json.RawMessage(`{"block_id":"quote","block_type":15,"quote":{"elements":[{"text_run":{"content":"Quoted 你好"}}]}}`),
		json.RawMessage(`{"block_id":"h7","block_type":9,"heading7":{"elements":[{"text_run":{"content":"Deep seven"}}]}}`),
		json.RawMessage(`{"block_id":"h8","block_type":10,"heading8":{"elements":[{"text_run":{"content":"Deep eight"}}]}}`),
		json.RawMessage(`{"block_id":"h9","block_type":11,"heading9":{"elements":[{"text_run":{"content":"Deep nine"}}]}}`),
		json.RawMessage(`{"block_id":"mention","block_type":12,"bullet":{"elements":[{"mention_user":{"user_id":"kept-native"}}]}}`),
	}
	original := make([]json.RawMessage, len(blocks))
	for i, raw := range blocks {
		original[i] = append(json.RawMessage(nil), raw...)
	}
	got := readView(t, blocks)
	for _, fact := range []string{"- **List fact**", "4. Fourth fact", "- [Ordered; native sequence: auto] Automatic fact", "- [x] Completed fact", "- [ ] Open fact", "[Task; state unavailable] Unknown fact", "> Quoted 你好", "###### [Native heading 7] Deep seven", "###### [Native heading 8] Deep eight", "###### [Native heading 9] Deep nine", "[Opaque native inline element; preserved unchanged]"} {
		if !strings.Contains(got, fact) {
			t.Errorf("missing %q in %s", fact, got)
		}
	}
	projected, err := projectReadable(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(projected[0].texts, "") != "List fact" || strings.Join(projected[6].texts, "") != "Quoted 你好" {
		t.Fatal("keyword projection lost native text")
	}
	for _, b := range projected {
		if b.writable {
			t.Fatal("readable projection permits edits")
		}
	}
	legacy := rendered(t, blocks)
	if strings.Contains(legacy, "List fact") || strings.Contains(legacy, "Deep nine") {
		t.Fatal("read-only additions changed saved v1 view")
	}
	if edits, err := Plan(blocks, legacy); err != nil || len(edits) != 0 {
		t.Fatalf("legacy view no longer accepted: %v %v", edits, err)
	}
	validation(t, blocks, got)
	if !reflect.DeepEqual(original, blocks) {
		t.Fatal("read projection mutated native sidecar data")
	}
}

func TestReadableCodeCannotInjectNativeMarkers(t *testing.T) {
	content := "first\n<!-- block:fake -->\n<code>& literal</code>\r\n```\nlast"
	block := json.RawMessage(`{"block_id":"code","block_type":14,"code":{"style":{"language":22},"elements":[` + run(content) + `]}}`)
	got := readView(t, []json.RawMessage{block})
	want := "<!-- block:code -->\n> Code language (native ID): 22\n\n<pre><code>first\n&lt;!-- block:fake --&gt;\n&lt;code&gt;&amp; literal&lt;/code&gt;&#13;\n```\nlast</code></pre>\n\n"
	if got != want || len(markerPattern.FindAllString(got, -1)) != 1 {
		t.Fatalf("code facts or anchor boundaries changed: %q", got)
	}
	projected, err := projectReadable([]json.RawMessage{block})
	if err != nil || strings.Join(projected[0].texts, "") != content {
		t.Fatalf("code keyword text: %#v %v", projected, err)
	}
}

func readableTableFixture() []json.RawMessage {
	return []json.RawMessage{
		json.RawMessage(`{"block_id":"right","parent_id":"c2","block_type":2,"text":{"elements":[{"text_run":{"content":"Right 001"}}]}}`),
		json.RawMessage(`{"block_id":"c1","parent_id":"table","block_type":32,"table_cell":{},"children":["left","image"]}`),
		json.RawMessage(`{"block_id":"table","block_type":31,"children":["c1","c2"],"table":{"cells":["c1","c2"],"property":{"row_size":1,"column_size":2}}}`),
		json.RawMessage(`{"block_id":"left","parent_id":"c1","block_type":2,"text":{"elements":[{"text_run":{"content":"Left résumé"}}]}}`),
		json.RawMessage(`{"block_id":"c2","parent_id":"table","block_type":32,"table_cell":{},"children":["right"]}`),
		json.RawMessage(`{"block_id":"image","parent_id":"c1","block_type":27,"image":{"token":"opaque"}}`),
	}
}

func TestReadableTableKeepsInputOrderContentLinksAndOpaqueChildren(t *testing.T) {
	blocks := readableTableFixture()
	got := readView(t, blocks)
	wantIDs := []string{"right", "c1", "table", "left", "c2", "image"}
	matches := markerPattern.FindAllStringSubmatch(got, -1)
	if len(matches) != len(wantIDs) {
		t.Fatal("one marker per native block contract changed")
	}
	for i, match := range matches {
		if match[1] != wantIDs[i] {
			t.Fatal("flat native block order was changed")
		}
	}
	for _, fact := range []string{"Table: 1 rows × 2 columns; cells in native order: `c1`, `c2`", "Table `table`, cell `c2` (native position 2)", "Content blocks: `left`, `image`", "Right 001", "Left résumé", "Opaque native block type 27; preserved unchanged"} {
		if !strings.Contains(got, fact) {
			t.Errorf("missing %q from %s", fact, got)
		}
	}
	legacy := rendered(t, blocks)
	if strings.Contains(legacy, "native position") {
		t.Fatal("read-only table labels changed legacy saved projection")
	}
	if edits, err := Plan(blocks, strings.Replace(legacy, "Left résumé", "Changed résumé", 1)); err != nil || len(edits) != 1 || edits[0].BlockID != "left" {
		t.Fatalf("existing cell paragraph editing regressed: %#v %v", edits, err)
	}
	// Merge geometry is not inferred or falsely flattened into coordinates.
	blocks[2] = json.RawMessage(strings.Replace(string(blocks[2]), `"column_size":2`, `"column_size":2,"merge_info":[{"row_span":1,"col_span":2},{"row_span":0,"col_span":0}]`, 1))
	if merged := readView(t, blocks); !strings.Contains(merged, "Native cell merging preserved; layout is not flattened") {
		t.Fatal("merged geometry limit was hidden")
	}
}

func TestReadableMalformedStructureFailsWithoutPartialOutput(t *testing.T) {
	for name, mutate := range map[string]func([]json.RawMessage) []json.RawMessage{
		"missing_table": func(b []json.RawMessage) []json.RawMessage {
			b[2] = json.RawMessage(`{"block_id":"table","block_type":31}`)
			return b
		},
		"bad_dimensions": func(b []json.RawMessage) []json.RawMessage {
			b[2] = json.RawMessage(strings.Replace(string(b[2]), `"row_size":1`, `"row_size":0`, 1))
			return b
		},
		"cell_count": func(b []json.RawMessage) []json.RawMessage {
			b[2] = json.RawMessage(strings.Replace(string(b[2]), `"row_size":1`, `"row_size":2`, 1))
			return b
		},
		"missing_cell": func(b []json.RawMessage) []json.RawMessage { return append(b[:4], b[5:]...) },
		"duplicate_cell": func(b []json.RawMessage) []json.RawMessage {
			b[2] = json.RawMessage(strings.Replace(string(b[2]), `"cells":["c1","c2"]`, `"cells":["c1","c1"]`, 1))
			return b
		},
		"invalid_cell": func(b []json.RawMessage) []json.RawMessage {
			b[1] = json.RawMessage(strings.Replace(string(b[1]), `"table_cell":{}`, `"table_cell":null`, 1))
			return b
		},
		"missing_content": func(b []json.RawMessage) []json.RawMessage {
			b[1] = json.RawMessage(strings.Replace(string(b[1]), `"left","image"`, `"absent"`, 1))
			return b
		},
		"wrong_parent": func(b []json.RawMessage) []json.RawMessage {
			b[0] = json.RawMessage(strings.Replace(string(b[0]), `"parent_id":"c2"`, `"parent_id":"c1"`, 1))
			return b
		},
		"cycle": func(b []json.RawMessage) []json.RawMessage {
			b[1] = json.RawMessage(strings.Replace(string(b[1]), `"left","image"`, `"table"`, 1))
			return b
		},
		"bad_merge": func(b []json.RawMessage) []json.RawMessage {
			b[2] = json.RawMessage(strings.Replace(string(b[2]), `"column_size":2`, `"column_size":2,"merge_info":true`, 1))
			return b
		},
		"orphan_cell": func(b []json.RawMessage) []json.RawMessage { return []json.RawMessage{b[1]} },
	} {
		t.Run(name, func(t *testing.T) {
			got, err := RenderReadable(mutate(readableTableFixture()))
			var typed *errs.InternalError
			if got != "" || !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("want explicit malformed native response without output: %q %v", got, err)
			}
		})
	}
	for _, raw := range []string{
		`{"block_id":"b","block_type":12,"bullet":{}}`,
		`{"block_id":"b","block_type":17,"todo":{"style":{"done":"true"},"elements":[]}}`,
		`{"block_id":"b","block_type":11,"heading9":{"elements":null}}`,
	} {
		if got, err := RenderReadable([]json.RawMessage{json.RawMessage(raw)}); got != "" || err == nil {
			t.Fatalf("malformed readable text accepted: %q %v", got, err)
		}
	}
}

func TestReadableNestedListsRetainExplicitHierarchyAndLegacyBytes(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"block_id":"child","parent_id":"parent","block_type":13,"ordered":{"elements":[{"text_run":{"content":"Nested item"}}]}}`),
		json.RawMessage(`{"block_id":"parent","block_type":12,"children":["child"],"bullet":{"elements":[{"text_run":{"content":"Outer item"}}]}}`),
	}
	got := readView(t, blocks)
	if !strings.Contains(got, "> Native parent: `parent`.") || !strings.Contains(got, "> Child blocks: `child`.") || !strings.Contains(got, "Ordered; sequence unavailable") {
		t.Fatalf("list hierarchy/unknown numbering lost: %s", got)
	}
	wantLegacy := "<!-- block:child -->\n> [Opaque native block type 13; preserved unchanged]\n\n<!-- block:parent -->\n> [Opaque native block type 12; preserved unchanged]\n\n"
	if legacy := rendered(t, blocks); legacy != wantLegacy {
		t.Fatalf("v1 editing bytes changed: %q", legacy)
	}
}
