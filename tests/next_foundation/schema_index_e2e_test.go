// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"net/http"
	"testing"
)

// schema reads only the local catalog, so the fixture rejects all traffic.
func TestSchemaBareIndexIsCompactAndAllKeepsFullCatalog(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("schema sent a request: %s %s", r.Method, r.URL.Path)
	}))

	bare := f.Run("schema")
	if bare.ExitCode != 0 {
		t.Fatalf("bare schema exit %d: %s", bare.ExitCode, bare.Stderr)
	}
	if len(bare.Stdout) >= 64<<10 {
		t.Errorf("bare schema stdout = %d bytes, want < 64 KiB", len(bare.Stdout))
	}

	all := f.Run("schema", "--all")
	if all.ExitCode != 0 {
		t.Fatalf("schema --all exit %d: %s", all.ExitCode, all.Stderr)
	}
	if len(all.Stdout) <= 500<<10 {
		t.Errorf("schema --all stdout = %d bytes, want > 500 KiB", len(all.Stdout))
	}
}
