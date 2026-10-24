// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"net/http"
	"strings"
	"testing"
)

// Usage errors are raised before any request, so the fixture rejects all traffic.
func TestUsageErrorsNameTheNextInvocation(t *testing.T) {
	f := NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("usage error sent a request: %s %s", r.Method, r.URL.Path)
	}))
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"doc"}, []string{`"subtype": "invalid_argument"`, `"unknown subcommand \"doc\"`, `"docs"`}},
		{[]string{"docs", "+pull"}, []string{`"name": "--doc"`, `required; string: `, "retry with `", "docs +pull --doc <doc>`"}},
		{[]string{"discover", "x", "--limit", "abc"}, []string{`invalid value \"abc\" for --limit: expected an integer`}},
		{[]string{"base", "+record-list", "--base-token", "b", "--table-id", "t", "--limit", "abc"}, []string{`invalid value \"abc\" for --limit: expected an integer`}},
		{[]string{"sheets", "+cells-get", "--spreadsheet-token", "s", "--sheet-id", "x", "--range", "A1", "--max-chars", "abc"}, []string{`invalid value \"abc\" for --max-chars: expected an integer`}},
	} {
		got := f.Run(tc.args...)
		if got.ExitCode != 2 {
			t.Fatalf("%v: exit %d, stderr %s", tc.args, got.ExitCode, got.Stderr)
		}
		for _, want := range tc.want {
			if !strings.Contains(got.Stderr, want) {
				t.Errorf("%v: stderr missing %s: %s", tc.args, want, got.Stderr)
			}
		}
		if strings.Contains(got.Stderr, "strconv.") || strings.Contains(got.Stderr, "Did you mean this?") {
			t.Errorf("%v: stderr leaks raw parser text: %s", tc.args, got.Stderr)
		}
	}
}
