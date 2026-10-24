// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package draft

import "testing"

func TestCoupledBodyReplacementPreservesUnchangedMIMEParts(t *testing.T) {
	for _, tc := range []struct {
		name, html, text     string
		htmlDirty, textDirty bool
	}{
		{"unchanged", "<p>same</p>", "same", false, false},
		{"repair_summary", "<p>same</p>", "stale", false, true},
		{"changed", "<p>old</p>", "old", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := &Part{PartID: "text", MediaType: "text/plain", Body: []byte(tc.text)}
			html := &Part{PartID: "html", MediaType: "text/html", Body: []byte(tc.html)}
			snapshot := &DraftSnapshot{Body: &Part{MediaType: "multipart/alternative", Children: []*Part{text, html}}, PrimaryTextPartID: "text", PrimaryHTMLPartID: "html"}
			if !tryApplyCoupledBodySetBody(snapshot, "<p>same</p>") {
				t.Fatal("coupled replacement rejected")
			}
			if text.Dirty != tc.textDirty || html.Dirty != tc.htmlDirty || string(text.Body) != "same" || string(html.Body) != "<p>same</p>" {
				t.Fatalf("text %#v html %#v", text, html)
			}
		})
	}
}

func TestSubjectReplacementPreservesMatchingEncodedWord(t *testing.T) {
	const encoded = "=?UTF-8?B?5Lit5paH?="
	snapshot := &DraftSnapshot{Headers: []Header{{Name: "Subject", Value: encoded}}}
	if err := applyOp(nil, snapshot, PatchOp{Op: "set_subject", Value: "中文"}, PatchOptions{}); err != nil {
		t.Fatal(err)
	}
	if snapshot.Headers[0].Value != encoded {
		t.Fatal("unchanged encoded subject was rewritten")
	}
	snapshot.Headers = append(snapshot.Headers, Header{Name: "Subject", Value: "other"})
	if err := applyOp(nil, snapshot, PatchOp{Op: "set_subject", Value: "中文"}, PatchOptions{}); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Headers) != 1 || snapshot.Headers[0].Value != "中文" {
		t.Fatalf("ambiguous subjects retained: %#v", snapshot.Headers)
	}
}
