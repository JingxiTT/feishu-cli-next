// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/apicatalog"
	"github.com/larksuite/cli/internal/meta"
)

func TestUpstreamWorkflowGuidanceReachesCommandTips(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	cases := []struct {
		domain, command, reference string
		decisions                  []string
	}{
		{"apps", "+export", "lark-apps/references/lark-apps-export.md", []string{"last commit", "download permission", "native ZIP", "Generic saved results"}},
		{"apps", "+release-create", "lark-apps/references/lark-apps-release-create.md", []string{"html", "frontend", "full_stack", "confirmed reason", "same release_id"}},
		{"apps", "+release-get", "lark-apps/references/lark-apps-release-get.md", []string{"Terminal status wins", "stop polling", "PENDING", "approval owner", "finished", "online_url"}},
		{"mail", "+thread-modify", "lark-mail/references/lark-mail-thread-modify.md", []string{"one call", "explicit mailbox address", "success_thread_ids", "failed_thread_ids", "before retrying"}},
		{"mail", "+thread-trash", "lark-mail/references/lark-mail-thread-trash.md", []string{"one call", "explicit mailbox address", "success_thread_ids", "failed_thread_ids", "before retrying", "authorized deletion scope"}},
		{"docs", "+create", "lark-doc/references/lark-doc-create.md", []string{"waits for async completion", "receipt and destination", "before retrying"}},
		{"im", "+chat-messages-list", "lark-im/references/lark-im-chat-messages-list.md", []string{"user-requested Markdown", "--no-reactions", "--result-fields", "--save-result", "pagination"}},
		{"im", "+threads-messages-list", "lark-im/references/lark-im-threads-messages-list.md", []string{"user-requested Markdown", "--no-reactions", "--result-fields", "--save-result", "pagination"}},
	}
	for _, test := range cases {
		t.Run(test.domain+"/"+test.command, func(t *testing.T) {
			raw, ok := For(apicatalog.Catalog{}, test.domain, test.command)
			if !ok {
				t.Fatal("missing command guidance")
			}
			parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
			if !ok {
				t.Fatal("invalid command guidance")
			}
			tips := strings.Join(parsed.Tips, "\n")
			for _, decision := range test.decisions {
				if !strings.Contains(tips, decision) {
					t.Errorf("missing decision %q in %s", decision, tips)
				}
			}
			if !containsExact(parsed.Skills, test.reference) {
				t.Errorf("missing reference %q", test.reference)
			}
			if _, err := os.Stat("../../skills/" + test.reference); err != nil {
				t.Fatalf("unreachable workflow reference: %v", err)
			}
		})
	}
}
