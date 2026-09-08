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

// The native raw renderer preserves typed values but does not expose formulas.
// Only the formula renderer supplies formula identity; its static display values
// and formatting must never replace the raw representation in an artifact.
func renderedNativeRows(t *testing.T, rows [][]json.RawMessage, render string) [][]json.RawMessage {
	t.Helper()
	require.Contains(t, []string{"raw_value", "formula"}, render)
	out := make([][]json.RawMessage, len(rows))
	for i, row := range rows {
		out[i] = make([]json.RawMessage, len(row))
		for j, raw := range row {
			var c map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &c))
			if render == "raw_value" {
				delete(c, "formula")
			} else if len(c["formula"]) == 0 && len(c["value"]) > 0 {
				c["value"] = json.RawMessage(`"native display text"`)
			}
			var err error
			out[i][j], err = json.Marshal(c)
			require.NoError(t, err)
		}
	}
	return out
}

func nativePageReply(t *testing.T, w http.ResponseWriter, region, render string, rows [][]json.RawMessage, revision any) {
	t.Helper()
	output := map[string]any{"ranges": []any{map[string]any{"range": region, "actual_range": region, "cells": renderedNativeRows(t, rows, render)}}, "has_more": false}
	if revision != nil {
		output["revision"] = revision
	}
	toolReply(w, output)
}

func TestNativeArtifactPagedPrecisionAndNoop(t *testing.T) {
	reads, writes := 0, 0
	samples := []json.RawMessage{json.RawMessage(`{"value":9007199254740993}`), json.RawMessage(`{"value":"001"}`), json.RawMessage(`{"value":false}`), json.RawMessage(`{}`), json.RawMessage(`{"value":2,"formula":"=1+1"}`), json.RawMessage(`{"value":0.1234567890123456789,"cell_styles":{"number_format":"0.00"}}`), json.RawMessage(`{"value":"=literal text"}`)}
	renders, regions := []string{}, []string{}
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
		render := input["value_render_option"].(string)
		renders, regions = append(renders, render), append(regions, region)
		var first, last int
		_, e := fmt.Sscanf(region, "A%d:A%d", &first, &last)
		require.NoError(t, e)
		require.LessOrEqual(t, last-first+1, 5000)
		rows := make([][]json.RawMessage, last-first+1)
		for i := range rows {
			rows[i] = []json.RawMessage{samples[(first+i-1)%len(samples)]}
		}
		nativePageReply(t, w, region, render, rows, 17)
	}))
	export := f.Run("sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--save-result", "native.json")
	exported := success(t, export)
	require.NotContains(t, export.Stdout, `"cells"`, "saved export must keep the native matrix outside process stdout")
	reference := exported["artifact"].(map[string]any)
	require.Equal(t, float64(5001), reference["count"])
	require.Equal(t, true, reference["complete"])
	require.Equal(t, 4, reads)
	require.Equal(t, []string{"raw_value", "formula", "raw_value", "formula"}, renders)
	require.Equal(t, []string{"A1:A5000", "A1:A5000", "A5001:A5001", "A5001:A5001"}, regions)
	raw, e := os.ReadFile(filepath.Join(f.WorkDir, "native.json"))
	require.NoError(t, e)
	require.Contains(t, string(raw), "9007199254740993")
	require.Contains(t, string(raw), "0.1234567890123456789")
	require.Contains(t, string(raw), `"complete":true`)
	var artifact struct {
		Result struct {
			Data struct {
				Cells [][]json.RawMessage `json:"cells"`
			} `json:"data"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(raw, &artifact))
	for i, row := range artifact.Result.Data.Cells {
		require.JSONEq(t, string(samples[i%len(samples)]), string(row[0]), "raw types, explicit formulas and static literal equals text must survive both views")
	}
	require.NotContains(t, string(raw), "native display text")
	imported := success(t, f.Run("sheets", "+data-import", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--artifact", "@native.json"))
	data := imported["data"].(map[string]any)
	require.Equal(t, true, data["noop"])
	require.Equal(t, float64(0), data["changed_cells"])
	require.Equal(t, 8, reads)
	require.Zero(t, writes)
	receipts, e := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
	require.NoError(t, e)
	require.Empty(t, receipts, "read-only no-op must not create mutation receipts")
}

func TestNativeExportRejectsTruncation(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		toolReply(w, map[string]any{"revision": 17, "has_more": true, "ranges": []any{map[string]any{"range": "A1:A1", "cells": []any{[]any{map[string]any{"value": "partial"}}}}}})
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
		input := decode(t, body.Input)
		nativePageReply(t, w, "A1:A1", input["value_render_option"].(string), [][]json.RawMessage{{value}}, 17)
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

func TestNativePlainImportReplacesStructuredIdentityThenNoops(t *testing.T) {
	for key, content := range map[string]string{
		"rich_text":       `[{"type":"link","text":"hello","link":"https://example.test"}]`,
		"multiple_values": `[{"value":"hello"}]`,
	} {
		t.Run(key, func(t *testing.T) {
			current := json.RawMessage(`{"value":"hello"}`)
			writes := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				var body struct {
					Tool  string `json:"tool_name"`
					Input string
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				if strings.HasSuffix(r.URL.Path, "invoke_write") {
					writes++
					require.Equal(t, "set_cell_range", body.Tool)
					var input struct {
						Cells [][]json.RawMessage `json:"cells"`
					}
					require.NoError(t, json.Unmarshal([]byte(body.Input), &input))
					require.Len(t, input.Cells, 1)
					require.Len(t, input.Cells[0], 1)
					require.JSONEq(t, `{"value":"hello"}`, string(input.Cells[0][0]), "plain replacement must not retain structured content")
					current = append(json.RawMessage(nil), input.Cells[0][0]...)
					toolReply(w, map[string]any{"revision": 19})
					return
				}
				require.Equal(t, "get_cell_ranges", body.Tool)
				input := decode(t, body.Input)
				nativePageReply(t, w, "A1:A1", input["value_render_option"].(string), [][]json.RawMessage{{current}}, 17)
			}))
			success(t, f.Run("sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--save-result", "plain.json"))
			current = json.RawMessage(`{"value":"hello","` + key + `":` + content + `}`)
			args := []string{"sheets", "+data-import", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A1", "--artifact", "@plain.json"}
			out := success(t, f.Run(args...))
			data := out["data"].(map[string]any)
			require.Equal(t, false, data["noop"])
			require.Equal(t, float64(1), data["changed_cells"])
			require.Equal(t, float64(1), data["acknowledged_pages"])
			require.Equal(t, "confirmed_applied", out["mutation"].(map[string]any)["outcome"])
			require.JSONEq(t, `{"value":"hello"}`, string(current))
			out = success(t, f.Run(args...))
			data = out["data"].(map[string]any)
			require.Equal(t, true, data["noop"])
			require.Equal(t, float64(0), data["changed_cells"])
			require.Equal(t, 1, writes, "second import must not repeat the replacement")
			receipts, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
			require.NoError(t, err)
			require.Len(t, receipts, 1, "only the actual replacement may create a receipt")
			receipt, err := os.ReadFile(receipts[0])
			require.NoError(t, err)
			require.Equal(t, "confirmed_applied", decode(t, string(receipt))["outcome"])
		})
	}
}

func TestNativePairedReadsRejectUnverifiedRevisionBeforeEffects(t *testing.T) {
	for _, operation := range []string{"export", "import"} {
		for _, mode := range []string{"missing", "string", "fractional", "paired-mismatch", "later-page-mismatch", "paired-extent"} {
			t.Run(operation+"/"+mode, func(t *testing.T) {
				reject := operation == "export"
				reads, writes := 0, 0
				f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.HasSuffix(r.URL.Path, "invoke_write") {
						writes++
						toolReply(w, map[string]any{"revision": 19})
						return
					}
					var body struct{ Tool, Input string }
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					input := decode(t, body.Input)
					region := input["ranges"].([]any)[0].(string)
					render := input["value_render_option"].(string)
					var first, last int
					_, err := fmt.Sscanf(region, "A%d:A%d", &first, &last)
					require.NoError(t, err)
					rows := make([][]json.RawMessage, last-first+1)
					for i := range rows {
						value := json.RawMessage(`{"value":0}`)
						if reject && operation == "import" {
							value = json.RawMessage(`{"value":1}`)
						}
						rows[i] = []json.RawMessage{value}
					}
					var revision any = 17
					if reject {
						reads++
						switch mode {
						case "missing":
							revision = nil
						case "string":
							revision = "17"
						case "fractional":
							revision = 17.5
						case "paired-mismatch":
							if render == "formula" {
								revision = 18
							}
						case "later-page-mismatch":
							if first > 1 {
								revision = 18
							}
						case "paired-extent":
							if render == "formula" {
								region = "A2:A5001"
							}
						}
					}
					nativePageReply(t, w, region, render, rows, revision)
				}))
				exportArgs := []string{"sheets", "+data-export", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--save-result", "revision.json"}
				var before []byte
				args := exportArgs
				if operation == "import" {
					success(t, f.Run(exportArgs...))
					var err error
					before, err = os.ReadFile(filepath.Join(f.WorkDir, "revision.json"))
					require.NoError(t, err)
					reject = true
					args = []string{"sheets", "+data-import", "--as", "user", "--spreadsheet-token", "shtFixture", "--sheet-id", "sh1", "--range", "A1:A5001", "--artifact", "@revision.json"}
				}
				result := f.Run(args...)
				require.NotZero(t, result.ExitCode)
				require.Empty(t, result.Stdout)
				failure := decode(t, result.Stderr)["error"].(map[string]any)
				want := "invalid_response"
				if mode == "paired-mismatch" || mode == "later-page-mismatch" {
					want = "conflict"
				}
				require.Equal(t, want, failure["subtype"])
				require.Greater(t, reads, 0)
				require.Zero(t, writes, "failed complete read must finish before import dispatch")
				after, err := os.ReadFile(filepath.Join(f.WorkDir, "revision.json"))
				if operation == "export" {
					require.True(t, os.IsNotExist(err), "failed export must not publish an artifact")
				} else {
					require.NoError(t, err)
					require.Equal(t, before, after, "failed import must preserve the source artifact")
				}
				receipts, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
				require.NoError(t, err)
				require.Empty(t, receipts, "read failure must create no mutation receipt")
			})
		}
	}
}
