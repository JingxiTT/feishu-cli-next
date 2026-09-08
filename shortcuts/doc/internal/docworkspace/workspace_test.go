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

func paragraph(id string, elements ...string) json.RawMessage {
	return json.RawMessage(`{"block_id":"` + id + `","block_type":2,"text":{"elements":[` + strings.Join(elements, ",") + `]}}`)
}
func run(content string) string {
	b, _ := json.Marshal(content)
	return `{"text_run":{"content":` + string(b) + `}}`
}
func rendered(t *testing.T, blocks []json.RawMessage) string {
	t.Helper()
	s, err := Render(blocks)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func validation(t *testing.T, blocks []json.RawMessage, edited string) {
	t.Helper()
	result, err := Plan(blocks, edited)
	var typed *errs.ValidationError
	if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidArgument || typed.Param != "--file" || typed.Hint == "" || result != nil {
		t.Fatalf("want typed validation and no partial plan, got %#v, %v", result, err)
	}
}
func TestRenderNativeStylesAndOpaqueRegions(t *testing.T) {
	blocks := []json.RawMessage{
		json.RawMessage(`{"block_id":"h","block_type":3,"heading1":{"elements":[{"text_run":{"content":"Title"}}]}}`),
		paragraph("p", `{"text_run":{"content":"bold","text_element_style":{"bold":true}}}`, `{"text_run":{"content":"斜体","text_element_style":{"italic":true}}}`, `{"text_run":{"content":"site","text_element_style":{"link":{"url":"https://example.org/a(b)?x=1&y=2"}}}}`),
		json.RawMessage(`{"block_id":"image","block_type":27,"image":{"token":"opaque","future":9007199254740993}}`),
		paragraph("mention", `{"mention_user":{"user_id":"opaque"}}`),
	}
	got := rendered(t, blocks)
	want := "<!-- block:h -->\n# Title\n\n<!-- block:p -->\n**bold**_斜体_[site](<https://example.org/a(b)?x=1&y=2>)\n\n<!-- block:image -->\n> [Opaque native block type 27; preserved unchanged]\n\n<!-- block:mention -->\n[Opaque native inline element; preserved unchanged]\n\n"
	if got != want {
		t.Fatalf("render:\n%q\nwant:\n%q", got, want)
	}
	if edits, err := Plan(blocks, got); err != nil || len(edits) != 0 {
		t.Fatalf("no-op: %v %v", edits, err)
	}
	validation(t, blocks, strings.Replace(got, "Opaque native block type 27", "image removed", 1))
}
func TestPlanPreservesSupportedNativeStylesAndUntouchedRawElements(t *testing.T) {
	first := `{"text_run":{"content":"target word","text_element_style":{"bold":true,"italic":true,"strikethrough":true,"underline":true,"inline_code":true,"background_color":4,"text_color":2,"link":{"url":"https://example.org/path"}}}}`
	other := `{ "text_run": { "content": " rest", "text_element_style": {"bold":false} } }`
	blocks := []json.RawMessage{paragraph("p", first, other), json.RawMessage(`{"block_id":"opaque","block_type":999,"future":9007199254740993}`)}
	original := append([]byte(nil), blocks[0]...)
	edits, err := Plan(blocks, strings.Replace(rendered(t, blocks), "target word", "changed word", 1))
	if err != nil || len(edits) != 1 {
		t.Fatalf("plan: %v %v", edits, err)
	}
	if edits[0].BlockID != "p" || edits[0].Text != "changed word rest" || string(edits[0].Elements[1]) != other {
		t.Fatalf("lost data: %#v", edits[0])
	}
	var before, after map[string]any
	_ = json.Unmarshal([]byte(first), &before)
	_ = json.Unmarshal(edits[0].Elements[0], &after)
	before["text_run"].(map[string]any)["content"] = "changed word"
	if !reflect.DeepEqual(before, after) || string(blocks[0]) != string(original) {
		t.Fatalf("mutated native data: %s", edits[0].Elements[0])
	}
	// Returned storage must not alias the snapshot.
	edits[0].Elements[1][0] = '!'
	if string(blocks[0]) != string(original) {
		t.Fatal("plan aliases original snapshot")
	}
}
func TestPlanUnicodeEscapesAndEmptyText(t *testing.T) {
	for _, tc := range []struct{ old, new string }{
		{"你好猫", "你好🐈"}, {"abc", ""}, {"", "新增"}, {"abc", "xabc"}, {"abc", "abcx"},
		{"literal &#13; &amp; <br> and \r\n end", "literal &#13; &amp; <br> and \r\n changed"},
		{"escape \\ * _ [ ] < > ( ) # + - . ! | ~ & ` { } end", "escape \\ * _ [ ] < > ( ) # + - . ! | ~ & ` { } changed"},
	} {
		t.Run(tc.old, func(t *testing.T) {
			blocks := []json.RawMessage{paragraph("p", run(tc.old))}
			replacement := rendered(t, []json.RawMessage{paragraph("p", run(tc.new))})
			edits, err := Plan(blocks, replacement)
			if err != nil || len(edits) != 1 || edits[0].Text != tc.new {
				t.Fatalf("roundtrip %q -> %q: %#v %v", tc.old, tc.new, edits, err)
			}
		})
	}
	if got := rendered(t, []json.RawMessage{paragraph("p", run("&#13;\r\n"))}); got != "<!-- block:p -->\n\\&\\#13;&#13;<br>\n\n" {
		t.Fatalf("entity ambiguity: %q", got)
	}
	for _, blocks := range [][]json.RawMessage{nil, {}, {paragraph("empty")}} {
		edits, err := Plan(blocks, rendered(t, blocks))
		if err != nil || len(edits) != 0 {
			t.Fatalf("empty no-op: %v %v", edits, err)
		}
	}
	validation(t, nil, "text")
}
func TestRejectUnsupportedWritableFields(t *testing.T) {
	for name, element := range map[string]string{
		"element":      `{"text_run":{"content":"word"},"future":9007199254740993}`,
		"run":          `{"text_run":{"content":"word","future":9007199254740993}}`,
		"style":        `{"text_run":{"content":"word","text_element_style":{"future":9007199254740993}}}`,
		"comments":     `{"text_run":{"content":"word","text_element_style":{"comment_ids":["c"]}}}`,
		"bad_bool":     `{"text_run":{"content":"word","text_element_style":{"bold":"true"}}}`,
		"null_bool":    `{"text_run":{"content":"word","text_element_style":{"bold":null}}}`,
		"bad_color":    `{"text_run":{"content":"word","text_element_style":{"text_color":1.5}}}`,
		"link_unknown": `{"text_run":{"content":"word","text_element_style":{"link":{"url":"https://example.org","future":true}}}}`,
		"link_control": `{"text_run":{"content":"word","text_element_style":{"link":{"url":"https://example.org/\u0000"}}}}`,
		"null_style":   `{"text_run":{"content":"word","text_element_style":null}}`,
	} {
		t.Run(name, func(t *testing.T) {
			blocks := []json.RawMessage{paragraph("p", element)}
			s := rendered(t, blocks)
			if edits, err := Plan(blocks, s); err != nil || len(edits) != 0 {
				t.Fatalf("unknown unchanged: %v %v", edits, err)
			}
			validation(t, blocks, strings.Replace(s, "word", "changed", 1))
		})
	}
	// Response-only block metadata never enters the PATCH payload.
	blocks := []json.RawMessage{json.RawMessage(`{"block_id":"p","block_type":2,"future":9007199254740993,"text":{"style":{"align":2},"future":{},"elements":[{"text_run":{"content":"word"}}]}}`)}
	edits, err := Plan(blocks, strings.Replace(rendered(t, blocks), "word", "changed", 1))
	if err != nil || len(edits) != 1 || strings.Contains(string(edits[0].Elements[0]), "future") {
		t.Fatalf("block metadata: %v %v", edits, err)
	}
}
func TestRejectMarkersFormattingAndCrossRunEdits(t *testing.T) {
	blocks := []json.RawMessage{paragraph("p", run("plain"), `{"text_run":{"content":"bold","text_element_style":{"bold":true}}}`), paragraph("q", run("next"))}
	s := rendered(t, blocks)
	cases := map[string]string{
		"delete_marker": strings.Replace(s, "<!-- block:p -->\n", "", 1),
		"change_id":     strings.Replace(s, "block:p", "block:other", 1),
		"duplicate_id":  strings.Replace(s, "block:q", "block:p", 1),
		"extra_marker":  s + "<!-- block:evil -->\n", "prefix": "prefix" + s,
		"style":           strings.Replace(s, "**bold**", "_bold_", 1),
		"cross_run":       strings.Replace(s, "plain**bold**", "short", 1),
		"injected_markup": strings.Replace(s, "plain", "<img src=x>", 1),
		"injected_line":   strings.Replace(s, "plain", "plain\n# heading", 1),
		"invalid_utf8":    s + string([]byte{255}),
		"late_invalid":    strings.Replace(strings.Replace(s, "plain", "changed", 1), "next", "[link](x)", 1),
	}
	for name, edited := range cases {
		t.Run(name, func(t *testing.T) { validation(t, blocks, edited) })
	}
	adjacent := []json.RawMessage{paragraph("p", run("left"), run("right"))}
	validation(t, adjacent, strings.Replace(rendered(t, adjacent), "leftright", "leftXright", 1))
	// A native string resembling a marker cannot create a new block boundary.
	literal := []json.RawMessage{paragraph("p", run("<!-- block:evil -->"))}
	if got := rendered(t, literal); len(markerPattern.FindAllString(got, -1)) != 1 {
		t.Fatalf("unsafe content marker: %q", got)
	}
}
func TestMalformedNativeSnapshots(t *testing.T) {
	for name, blocks := range map[string][]json.RawMessage{
		"json": {json.RawMessage(`{`)}, "no_id": {json.RawMessage(`{"block_type":2}`)},
		"unsafe_id":        {json.RawMessage(`{"block_id":"p\n<!-- block:evil -->","block_type":2}`)},
		"duplicate":        {paragraph("p", run("a")), paragraph("p", run("b"))},
		"elements_missing": {json.RawMessage(`{"block_id":"p","block_type":2,"text":{}}`)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Render(blocks)
			var typed *errs.InternalError
			if !errors.As(err, &typed) || typed.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("untyped malformed native: %v", err)
			}
		})
	}
	blocks := []json.RawMessage{paragraph("p", `{"text_run":{"content":null}}`)}
	s := rendered(t, blocks)
	if !strings.Contains(s, "Opaque native inline") {
		t.Fatalf("null content interpreted as empty editable text: %q", s)
	}
	validation(t, blocks, strings.Replace(s, "Opaque", "changed", 1))
}

func TestPlanMultipleBlocksAndRuns(t *testing.T) {
	blocks := []json.RawMessage{paragraph("p", run("left"), `{"text_run":{"content":"target","text_element_style":{"bold":true}}}`, run("right")), paragraph("q", run("other"))}
	s := strings.Replace(strings.Replace(rendered(t, blocks), "target", "changed", 1), "other", "also changed", 1)
	edits, err := Plan(blocks, s)
	if err != nil || len(edits) != 2 || edits[0].Text != "leftchangedright" || edits[1].Text != "also changed" {
		t.Fatalf("multi-block plan: %v %v", edits, err)
	}
	// Edits crossing two native runs are refused even when neither has style.
	plain := []json.RawMessage{paragraph("p", run("abc"), run("def"))}
	validation(t, plain, strings.Replace(rendered(t, plain), "abcdef", "abXef", 1))
}

func TestPlanEditorNewlinesPreserveNativeContent(t *testing.T) {
	blocks := []json.RawMessage{paragraph("p", run("Old \r\n literal &#13; <br>  ")), paragraph("empty"), json.RawMessage(`{"block_id":"resource","block_type":27,"image":{"token":"opaque"}}`)}
	original := rendered(t, blocks)
	for name, reformat := range map[string]func(string) string{
		"single_separator":  func(s string) string { return strings.ReplaceAll(s, "\n\n", "\n") },
		"extra_separator":   func(s string) string { return strings.ReplaceAll(s, "\n\n", "\n\n\n\n") },
		"no_eof_newline":    func(s string) string { return strings.TrimRight(s, "\n") },
		"extra_eof_newline": func(s string) string { return s + "\n" },
		"crlf_bom":          func(s string) string { return "\ufeff" + strings.ReplaceAll(s, "\n", "\r\n") },
	} {
		t.Run(name, func(t *testing.T) {
			edits, err := Plan(blocks, reformat(original))
			if err != nil || len(edits) != 0 {
				t.Fatalf("format-only change planned mutations: %v %v", edits, err)
			}
			edits, err = Plan(blocks, reformat(strings.Replace(original, "Old ", "New ", 1)))
			if err != nil || len(edits) != 1 || edits[0].Text != "New \r\n literal &#13; <br>  " {
				t.Fatalf("native text changed: %#v %v", edits, err)
			}
		})
	}
	empty := []json.RawMessage{paragraph("empty")}
	if edits, err := Plan(empty, "<!-- block:empty -->"); err != nil || len(edits) != 0 {
		t.Fatalf("empty block EOF: %v %v", edits, err)
	}
	// Native line breaks remain encoded; deleting one is a real text edit.
	edits, err := Plan(blocks, strings.Replace(original, "<br>", "", 1))
	if err != nil || len(edits) != 1 || edits[0].Text != "Old \r literal &#13; <br>  " {
		t.Fatalf("lost native newline edit: %v %v", edits, err)
	}
	validation(t, blocks, strings.Replace(original, "Old ", "Old\n", 1))
	validation(t, blocks, strings.Replace(original, "block:resource", "block:other", 1))
	validation(t, blocks, strings.Replace(original, "Opaque native block type 27", "Removed", 1)+"\n")
}

func TestPlanVisiblePhraseAcrossMarksIsNotFlattened(t *testing.T) {
	blocks := []json.RawMessage{paragraph("p", run("Old "), `{"text_run":{"content":"status","text_element_style":{"bold":true,"link":{"url":"https://example.org/status"}}}}`, run(" — 你好"))}
	original := rendered(t, blocks)
	unchanged := strings.Replace(original, "Old status", "New status", 1)
	if edits, err := Plan(blocks, unchanged); err != nil || len(edits) != 0 {
		t.Fatalf("unmatched local replacement should be no-op: %v %v", edits, err)
	}
	edits, err := Plan(blocks, strings.Replace(original, "Old ", "New ", 1))
	if err != nil || len(edits) != 1 || edits[0].Text != "New status — 你好" {
		t.Fatalf("smallest edit/complete paragraph: %v %v", edits, err)
	}
	validation(t, blocks, "<!-- block:p -->\nNew status — 你好\n")
}
