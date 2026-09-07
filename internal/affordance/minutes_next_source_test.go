// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"
)

func TestMinutesNextRecoveryAndEqualityGuidance(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	for method, fragments := range map[string][]string{
		"+update":       {"--skip-unchanged", "not concurrency protection"},
		"+summary":      {"--skip-unchanged", "preserves whitespace"},
		"+todo":         {"distinct todo IDs", "missing list never implies deletion"},
		"+word-replace": {"never repeat successful replacements", "Missing or unknown outcomes", "read-only transcript check"},
	} {
		raw, ok := For("minutes", method)
		if !ok {
			t.Fatalf("missing minutes %s guidance", method)
		}
		for _, fragment := range fragments {
			if !strings.Contains(string(raw), fragment) {
				t.Errorf("%s missing %q: %s", method, fragment, raw)
			}
		}
	}
}
