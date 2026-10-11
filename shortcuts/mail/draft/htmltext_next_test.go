// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package draft

import (
	"strings"
	"testing"
)

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
