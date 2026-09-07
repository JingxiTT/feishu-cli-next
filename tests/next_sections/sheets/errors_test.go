// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestValidationHasNoBusinessRequests(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		param string
	}{
		{"ambiguous-sheet", []string{"+cells-get", "--sheet-id", "sh1", "--sheet-name", "Sheet1", "--range", "A1"}, ""},
		{"conflicting-locator", []string{"+cells-get", "--url", "https://example.invalid/sheets/other", "--sheet-id", "sh1", "--range", "A1"}, ""},
		{"invalid-sort-type", []string{"+range-sort", "--sheet-id", "sh1", "--range", "A1:B2", "--sort-keys", `[{"column":"B","ascending":"false"}]`}, ""},
		{"invalid-cell-json", []string{"+cells-set", "--sheet-id", "sh1", "--range", "A1", "--cells", "[["}, ""},
		{"bad-native-extent", []string{"+data-export", "--sheet-id", "sh1", "--range", "A:A"}, "--range"},
		{"unbound-native-artifact", []string{"+data-import", "--sheet-id", "sh1", "--range", "A1", "--artifact", `{"cells":[[{"value":1}]]}`}, "--artifact"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := nextfixture.NewFixture(t, nil)
			args := append([]string{"sheets"}, tc.args...)
			args = append(args, "--spreadsheet-token", "shtFixture")
			r := f.Run(args...)
			require.NotZero(t, r.ExitCode)
			require.Empty(t, r.Stdout)
			p := errorProblem(t, r.Stderr)
			require.Equal(t, "validation", p["type"])
			require.NotEmpty(t, p["subtype"])
			if tc.param != "" {
				require.Equal(t, tc.param, p["param"])
			}
		})
	}
}

func TestAPIErrorPreservesMetadataAndDoesNotReplayMutation(t *testing.T) {
	calls := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1310246, "msg": "synthetic invalid range", "log_id": "synthetic-sheets-log", "data": map[string]any{}})
	}))
	r := f.Run("sheets", "+cells-clear", "--token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:B2", "--yes")
	require.NotZero(t, r.ExitCode)
	require.Empty(t, r.Stdout)
	p := errorProblem(t, r.Stderr)
	require.Equal(t, "api", p["type"])
	require.Equal(t, "mutation_outcome", p["subtype"])
	require.Equal(t, "unknown", p["outcome"])
	p = p["cause"].(map[string]any)
	require.Equal(t, "invalid_parameters", p["subtype"])
	require.Equal(t, float64(1310246), p["code"])
	require.Equal(t, "synthetic-sheets-log", p["log_id"])
	require.Contains(t, p["message"], "clear_cell_range")
	require.Equal(t, 1, calls)
}

func TestMalformedBackendOutputIsFailure(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reply(w, map[string]any{"output": "{invalid"}) }))
	r := f.Run("sheets", "+workbook-info", "--token", "shtFixture")
	require.NotZero(t, r.ExitCode)
	require.Empty(t, r.Stdout)
	p := errorProblem(t, r.Stderr)
	require.Equal(t, "internal", p["type"])
	require.Equal(t, "invalid_response", p["subtype"])
}

func TestFormulaFailureRetainsPredicateResult(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		toolReply(w, map[string]any{"status": "errors_found", "total_errors": 1, "error_summary": []any{map[string]any{"error": "#DIV/0!", "locations": []string{"sh1!A1"}}}})
	}))
	r := f.Run("sheets", "+formula-verify", "--token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--exit-on-error")
	require.NotZero(t, r.ExitCode)
	out := decode(t, r.Stdout)
	require.Equal(t, "errors_found", out["data"].(map[string]any)["status"])
	require.Contains(t, r.Stdout, "#DIV/0!")
	require.Equal(t, "failed_precondition", errorProblem(t, r.Stderr)["subtype"])
}

func TestPartialBatchFailureDoesNotClaimRollbackOrReplay(t *testing.T) {
	calls := 0
	detail, _ := json.Marshal(map[string]any{"message": "batch_update: 1 succeeded, 1 failed", "failures": []any{map[string]any{"index": 1, "tool_name": "clear_cell_range", "error": "synthetic failure"}}})
	outer, _ := json.Marshal(map[string]any{"error": string(detail)})
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 1310201, "msg": string(outer), "log_id": "batch-log", "data": map[string]any{}})
	}))
	r := f.Run("sheets", "+batch-update", "--token", "shtFixture", "--operations", `[{"shortcut":"+cells-clear","input":{"sheet_id":"sh1","range":"A1"}},{"shortcut":"+cells-clear","input":{"sheet_id":"sh1","range":"B1"}}]`, "--continue-on-error", "--yes")
	require.NotZero(t, r.ExitCode)
	require.Empty(t, r.Stdout)
	p := errorProblem(t, r.Stderr)
	require.Equal(t, "unknown", p["outcome"])
	require.Equal(t, "mutation_outcome", p["subtype"])
	p = p["cause"].(map[string]any)
	require.Equal(t, "batch-log", p["log_id"])
	require.Contains(t, p["message"], "operations[1]")
	require.Contains(t, p["message"], "no rollback")
	require.Contains(t, p["message"], "only the failed operations")
	require.Equal(t, 1, calls)
}

// These are selectable, explicitly blocked acceptance scenarios. The current
// harness cannot prove account-private creation before a write. A skip is not
// successful live evidence and cannot close the section.
func TestLivePrivateSheetsAcceptance(t *testing.T) {
	for _, scenario := range []string{"private-workbook-create-use-cleanup", "typed-cells-formulas-native-roundtrip", "sheet-dimension-range-lifecycle", "chart-pivot-condition-filter-view-sparkline-image-lifecycle", "import-export-large-file-cleanup", "history-revert-status-and-changesets", "generated-filter-and-find-lifecycle"} {
		t.Run(scenario, func(t *testing.T) {
			t.Skip("BLOCKED: effective account and tenant plus account-only creation visibility have not been proven; no live writes, delivery, sharing, or cleanup of pre-existing resources permitted")
		})
	}
}

func errorProblem(t *testing.T, raw string) map[string]any {
	t.Helper()
	d := json.NewDecoder(strings.NewReader(raw))
	var first map[string]any
	require.NoError(t, d.Decode(&first))
	p, ok := first["error"].(map[string]any)
	require.True(t, ok)
	for {
		var extra map[string]any
		err := d.Decode(&extra)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		require.Contains(t, extra, "mutation")
	}
	return p
}

// An API code of zero is not evidence that every member of a batch applied.
// Per-member failures inside Sheet-AI's JSON-string output remain unknown and
// must not be labeled confirmed_applied or automatically replayed.
func TestBatchMemberFailureInsideSuccessfulEnvelope(t *testing.T) {
	calls := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		toolReply(w, map[string]any{"results": []any{map[string]any{"success": false, "error": "synthetic member failed"}}})
	}))
	r := f.Run("sheets", "+batch-update", "--token", "shtFixture", "--operations", `[{"shortcut":"+cells-clear","input":{"sheet_id":"sh1","range":"A1"}}]`, "--yes")
	require.Equal(t, 22, r.ExitCode, r.Stdout+"\n"+r.Stderr)
	p := errorProblem(t, r.Stderr)
	require.Equal(t, "mutation_outcome", p["subtype"])
	require.Equal(t, "unknown", p["outcome"])
	require.Equal(t, 1, calls)
}
