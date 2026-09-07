// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestNativeArtifactPagedPrecisionAndNoop(t *testing.T) {
	reads, writes := 0, 0
	samples := []json.RawMessage{json.RawMessage(`{"value":9007199254740993}`), json.RawMessage(`{"value":"001"}`), json.RawMessage(`{"value":false}`), json.RawMessage(`{}`), json.RawMessage(`{"formula":"=1+1"}`), json.RawMessage(`{"value":0.1234567890123456789,"cell_styles":{"number_format":"0.00"}}`)}
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "POST", r.Method)
		var body struct {
			Tool  string `json:"tool_name"`
			Input string
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if strings.HasSuffix(r.URL.Path, "invoke_write") {
			writes++
			t.Error("unchanged import must not write")
			toolReply(w, map[string]any{})
			return
		}
		reads++
		require.Equal(t, "get_cell_ranges", body.Tool)
		input := decode(t, body.Input)
		region := input["ranges"].([]any)[0].(string)
		var first, last int
		_, e := fmt.Sscanf(region, "A%d:A%d", &first, &last)
		require.NoError(t, e)
		require.LessOrEqual(t, last-first+1, 5000)
		rows := make([][]json.RawMessage, last-first+1)
		for i := range rows {
			rows[i] = []json.RawMessage{samples[(first+i-1)%len(samples)]}
		}
		toolReply(w, map[string]any{"ranges": []any{map[string]any{"range": region, "cells": rows}}, "has_more": false})
	}))
	export := f.Run("sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--save-result", "native.json")
	success(t, export)
	require.Equal(t, 2, reads)
	raw, e := os.ReadFile(filepath.Join(f.WorkDir, "native.json"))
	require.NoError(t, e)
	require.Contains(t, string(raw), "9007199254740993")
	require.Contains(t, string(raw), "0.1234567890123456789")
	require.Contains(t, string(raw), `"complete":true`)
	imported := success(t, f.Run("sheets", "+data-import", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--artifact", "@native.json"))
	data := imported["data"].(map[string]any)
	require.Equal(t, true, data["noop"])
	require.Equal(t, float64(0), data["changed_cells"])
	require.Equal(t, 4, reads)
	require.Zero(t, writes)
	receipts, e := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
	require.NoError(t, e)
	require.Empty(t, receipts, "read-only no-op must not create mutation receipts")
}

func TestNativeExportRejectsTruncation(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		toolReply(w, map[string]any{"has_more": true, "ranges": []any{map[string]any{"range": "A1:A1", "cells": []any{[]any{map[string]any{"value": "partial"}}}}}})
	}))
	r := f.Run("sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--save-result", "bad.json")
	require.NotZero(t, r.ExitCode)
	require.Empty(t, r.Stdout)
	require.Equal(t, "invalid_response", decode(t, r.Stderr)["error"].(map[string]any)["subtype"])
	_, e := os.Stat(filepath.Join(f.WorkDir, "bad.json"))
	require.True(t, os.IsNotExist(e))
}

func TestNativeChangedImportPreservesNumericWireToken(t *testing.T) {
	changed := false
	writes := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Tool  string `json:"tool_name"`
			Input string
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if strings.HasSuffix(r.URL.Path, "invoke_write") {
			writes++
			require.Equal(t, "set_cell_range", body.Tool)
			require.Contains(t, body.Input, "9007199254740993")
			require.NotContains(t, body.Input, "9007199254740992")
			toolReply(w, map[string]any{"revision": 18})
			return
		}
		value := json.RawMessage(`{"value":9007199254740993}`)
		if changed {
			value = json.RawMessage(`{"value":0}`)
		}
		toolReply(w, map[string]any{"ranges": []any{map[string]any{"range": "A1:A1", "cells": [][]json.RawMessage{{value}}}}})
	}))
	success(t, f.Run("sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--save-result", "one.json"))
	changed = true
	out := success(t, f.Run("sheets", "+data-import", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--artifact", "@one.json"))
	data := out["data"].(map[string]any)
	require.Equal(t, false, data["noop"])
	require.Equal(t, float64(1), data["changed_cells"])
	require.Equal(t, float64(1), data["acknowledged_pages"])
	require.Equal(t, 1, writes)
}
