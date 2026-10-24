// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package affordance

import (
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/apicatalog"
)

func TestMarkdownAffordancePreservesSourceAndWriteSemantics(t *testing.T) {
	previous := mdSource
	t.Cleanup(func() { SetSource(previous) })
	SetSource(os.DirFS("../../affordance"))
	for command, guidance := range map[string]string{
		"+create":    "native document blocks",
		"+fetch":     "downloaded bytes",
		"+diff":      "missing final newline",
		"+patch":     "including when matches exist",
		"+overwrite": "including identical content",
	} {
		raw, ok := For(apicatalog.Catalog{}, "markdown", command)
		if !ok || !strings.Contains(string(raw), guidance) {
			t.Errorf("%s missing guidance %q: %s", command, guidance, raw)
		}
	}
}
