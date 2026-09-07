// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/meta"
)

func TestBaseNextGuidancePreservesMutationAndConversionDecisions(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	for command, phrase := range map[string]string{"+field-create": "skills read lark-base references/lark-base-field-formula.md", "+field-update": "installed legacy skill", "+record-upsert": "does not prevent concurrent edits", "+record-batch-update": "every record exists", "+record-import": "does not authorize replay", "+record-list": "not a snapshot"} {
		raw, ok := For("base", command)
		if !ok || !strings.Contains(string(raw), phrase) {
			t.Errorf("%s missing %q", command, phrase)
		}
		if _, ok := (meta.Method{Affordance: raw}).ParsedAffordance(); !ok {
			t.Errorf("%s did not parse", command)
		}
	}
}
