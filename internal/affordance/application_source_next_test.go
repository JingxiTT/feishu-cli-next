// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"testing"
)

func TestApplicationReplacementAndNoopGuidance(t *testing.T) {
	prev := mdSource
	t.Cleanup(func() { SetSource(prev) })
	SetSource(os.DirFS("../../affordance"))
	update := parsedApplicationAffordance(t, "+slash-command-update")
	for _, want := range []string{"whole object", "omitted languages", "unchanged replacement", "write-only path"} {
		if !containsItem(update.Tips, want) {
			t.Errorf("missing %q in %v", want, update.Tips)
		}
	}
}
