// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/meta"
)

func TestAppsWorkflowGuidance(t *testing.T) {
	prev := mdSource
	t.Cleanup(func() { SetSource(prev) })
	SetSource(os.DirFS("../../affordance"))
	cases := []struct{ command, guidance, example string }{
		{"+update", "--skip-unchanged", `feishu-cli-next apps +update --app-id app_fixture --name "Journal"`},
		{"+db-data-export", "not an atomic snapshot", "feishu-cli-next apps +db-data-export --app-id app_fixture --table orders --output orders.json --limit 100"},
		{"+db-data-import", "Generic result artifacts cannot be imported", "feishu-cli-next apps +db-data-import --app-id app_fixture --file orders.json --table orders --dry-run"},
		{"+env-list", "can contain credentials", "feishu-cli-next apps +env-list --app-id app_fixture"},
		{"+file-upload", "separate effects", "feishu-cli-next apps +file-upload --app-id app_fixture --file logo.png --dry-run"},
		{"+html-publish", "does not prove deployment succeeded", "feishu-cli-next apps +html-publish --app-id app_fixture --path site --dry-run"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			raw, ok := For("apps", tc.command)
			if !ok {
				t.Fatal("missing Apps guidance")
			}
			a, ok := (meta.Method{Affordance: raw}).ParsedAffordance()
			if !ok {
				t.Fatal("unparseable Apps guidance")
			}
			if !strings.Contains(strings.Join(a.Tips, " "), tc.guidance) {
				t.Fatalf("missing workflow constraint in %v", a.Tips)
			}
			if len(a.Examples) != 1 || a.Examples[0].Command != tc.example {
				t.Fatalf("examples %#v", a.Examples)
			}
		})
	}
}
