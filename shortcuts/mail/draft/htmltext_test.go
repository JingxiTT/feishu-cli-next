// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package draft

import (
	"strings"
	"testing"

	xhtml "golang.org/x/net/html"
)

func TestPlainTextFromHTML(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{
			name: "strips inline style tag",
			html: `<html><head><style>body{color:red}</style></head><body><p>Hello</p></body></html>`,
			want: "Hello",
		},
		{
			name: "strips script tag",
			html: `<html><body><p>Before</p><script>alert("xss")</script><p>After</p></body></html>`,
			want: "Before\nAfter",
		},
		{
			name: "strips noscript tag",
			html: `<html><body><p>Visible</p><noscript>Fallback text</noscript></body></html>`,
			want: "Visible",
		},
		{
			name: "strips head and title",
			html: `<html><head><title>Page Title</title></head><body>Content</body></html>`,
			want: "Content",
		},
		{
			name: "plain text passthrough",
			html: `<div>Line one</div><div>Line two</div>`,
			want: "Line one\nLine two",
		},
		{
			name: "mixed non-text and text tags",
			html: `<html><head><style>.a{}</style><link rel="stylesheet"/><meta charset="utf-8"/></head><body><div>Only this</div></body></html>`,
			want: "Only this",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plainTextFromHTML(tt.html)
			if got != tt.want {
				t.Errorf("plainTextFromHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlainTextFromHTMLDeepNesting(t *testing.T) {
	// Build HTML with 10000 levels of nesting — would overflow the stack
	// with the old recursive implementation.
	const depth = 10_000
	var b strings.Builder
	for i := 0; i < depth; i++ {
		b.WriteString("<div>")
	}
	b.WriteString("deep")
	for i := 0; i < depth; i++ {
		b.WriteString("</div>")
	}
	got := plainTextFromHTML(b.String())
	if got != "deep" {
		t.Errorf("deep nesting: got %q, want %q", got, "deep")
	}
}

func TestIsHTMLNonTextTag(t *testing.T) {
	tests := []struct {
		tag  string
		want bool
	}{
		{"script", true},
		{"style", true},
		{"head", true},
		{"meta", true},
		{"noscript", true},
		{"link", true},
		{"title", true},
		{"div", false},
		{"p", false},
		{"span", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			n := &xhtml.Node{Type: xhtml.ElementNode, Data: tt.tag}
			if got := isHTMLNonTextTag(n); got != tt.want {
				t.Errorf("isHTMLNonTextTag(%q) = %v, want %v", tt.tag, got, tt.want)
			}
		})
	}
}

func TestPlainTextFromHTMLExported(t *testing.T) {
	got := PlainTextFromHTML("<p>Hello world</p>")
	if !strings.Contains(got, "Hello world") {
		t.Fatalf("PlainTextFromHTML: expected to contain \"Hello world\", got %q", got)
	}
}

func TestPlainTextFromHTMLDeepNestingExcludesHiddenContent(t *testing.T) {
	html := strings.Repeat("<div>", 600) + `<head><title>secret</title></head><script>secret()</script><style>.secret{}</style><noscript>secret</noscript><p>Visible &amp; safe</p><div>Next</div>` + strings.Repeat("</div>", 600)
	if got := plainTextFromHTML(html); got != "Visible & safe\nNext" {
		t.Fatalf("unexpected plain-text fallback: %q", got)
	}
}

func TestPlainTextFromHTMLDeepNestingMalformedHiddenTags(t *testing.T) {
	cases := []struct{ html, want string }{
		{`<script/>hidden()</script><p>Visible</p>`, "Visible"},
		{`<style/>hidden{}</style><p>Visible</p>`, "Visible"},
		{`<title/>Hidden</title><p>Visible</p>`, "Visible"},
		{`<head><title>Hidden</title><body><p>Visible</p>`, "Visible"},
		{`<head><title>Hidden</title><p>Visible</p>`, "Visible"},
	}
	for _, test := range cases {
		html := strings.Repeat("<div>", 600) + test.html + strings.Repeat("</div>", 600)
		if got := plainTextFromHTML(html); got != test.want {
			t.Errorf("fallback for %q: got %q, want %q", test.html, got, test.want)
		}
	}
}
