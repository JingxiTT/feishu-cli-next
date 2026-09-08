// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/meta"
)

func TestContactBasicProfileGuidanceAllowsUserIdentity(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	raw, ok := For("contact", "+get-user")
	if !ok {
		t.Fatal("missing get-user overlay")
	}
	parsed, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
	if !ok {
		t.Fatal("invalid get-user overlay")
	}
	if !strings.Contains(strings.Join(parsed.UseWhen, "\n"), "user or bot identity") {
		t.Fatal("basic-profile route does not support both identities in help/schema")
	}
	avoid := strings.Join(parsed.AvoidWhen, "\n")
	for _, want := range []string{"email or departments", "several known open_ids", "+search-user"} {
		if !strings.Contains(avoid, want) {
			t.Errorf("missing conditional richer/batch route %q", want)
		}
	}
	if strings.Contains(avoid, "Running as a user") {
		t.Fatal("user identity incorrectly excludes basic-profile lookup")
	}
	tips := strings.Join(parsed.Tips, "\n")
	for _, want := range []string{"a bot must pass --user-id", "--user-id-type"} {
		if !strings.Contains(tips, want) {
			t.Errorf("lost identity/ID constraint %q", want)
		}
	}
	if !containsExact(parsed.Skills, "lark-contact/references/lark-contact-get-user.md") {
		t.Fatal("full reference is no longer discoverable")
	}
}

func TestContactGuidesRetainBasicAndRichUserRoutes(t *testing.T) {
	for _, path := range []string{"../../skills/lark-contact/SKILL.md", "../../skills/lark-contact/references/lark-contact-get-user.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		guide := string(raw)
		for _, want := range []string{"+get-user --user-id", "基本资料", "部门 / 邮箱", "+search-user --user-ids"} {
			if !strings.Contains(guide, want) {
				t.Errorf("%s missing %q", path, want)
			}
		}
		if strings.Contains(guide, "user 身份按 ID 取他人请用") {
			t.Errorf("%s wrongly requires richer search for basic profile", path)
		}
	}
}
