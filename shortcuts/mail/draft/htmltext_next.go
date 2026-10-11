// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package draft

import (
	"strings"

	xhtml "golang.org/x/net/html"
)

func plainTextHTMLTokens(raw string) *xhtml.Node {
	doc := &xhtml.Node{Type: xhtml.DocumentNode}
	tokens := xhtml.NewTokenizer(strings.NewReader(raw))
	var skipped string
	var skipDepth int
	for {
		kind := tokens.Next()
		if kind == xhtml.ErrorToken {
			return doc
		}
		token := tokens.Token()
		if skipped != "" {
			if token.Data == skipped {
				switch kind {
				case xhtml.StartTagToken:
					skipDepth++
				case xhtml.EndTagToken:
					skipDepth--
					if skipDepth == 0 {
						skipped = ""
					}
				}
			}
			continue
		}
		node := &xhtml.Node{Type: xhtml.ElementNode, Data: token.Data}
		switch kind {
		case xhtml.TextToken:
			node.Type = xhtml.TextNode
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			if isHTMLNonTextTag(node) {
				// HTML ignores the slash on non-void elements. Head is only
				// structural: visible content implicitly closes it.
				if node.Data != "meta" && node.Data != "link" && node.Data != "head" {
					skipped, skipDepth = node.Data, 1
				}
				continue
			}
		case xhtml.EndTagToken:
			if !isHTMLBlockBoundary(node) {
				continue
			}
		default:
			continue
		}
		doc.AppendChild(node)
	}
}
