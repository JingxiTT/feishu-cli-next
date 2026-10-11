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

func TestTaskAffordanceMutationSemantics(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	for command, phrase := range map[string]string{
		"+update":            "does not prevent concurrent edits",
		"+tasklist-create":   "failed_tasks",
		"+assign":            "notify members",
		"+followers":         "not rolled back",
		"+tasklist-members":  "not a clear",
		"+reminder":          "may leave no reminder",
		"+reopen":            "completed_at",
		"+set-ancestor":      "deliberately detaches",
		"+upload-attachment": "append effect",
	} {
		raw, ok := For(apicatalog.Catalog{}, "task", command)
		if !ok || !strings.Contains(string(raw), phrase) {
			t.Errorf("%s missing %q: %s", command, phrase, raw)
		}
		if _, ok := (meta.Method{Affordance: raw}).ParsedAffordance(); !ok {
			t.Errorf("%s did not parse", command)
		}
	}
}
