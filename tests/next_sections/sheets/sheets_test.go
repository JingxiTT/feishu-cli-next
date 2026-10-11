// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets_test

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

type commandContract struct {
	Flags    map[string]struct{ Hidden bool }
	Risk     string
	Source   string
	Runnable bool
}
type shortcutContract struct {
	FlagSchemas map[string]any `json:"flag_schemas"`
}
type inventory struct {
	Commands  map[string]commandContract
	Shortcuts map[string]shortcutContract
}

func frozen(t *testing.T) inventory {
	t.Helper()
	file, err := os.Open("../../../docs/reference/v1.0.97/registrations.json.gz")
	require.NoError(t, err)
	defer file.Close()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer reader.Close()
	var brands map[string]inventory
	require.NoError(t, json.NewDecoder(reader).Decode(&brands))
	return brands["feishu"]
}
func decode(t *testing.T, raw string) map[string]any {
	t.Helper()
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &out), raw)
	return out
}
func success(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, r.Stderr)
	return decode(t, r.Stdout)
}
func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func toolReply(w http.ResponseWriter, data any) {
	raw, _ := json.Marshal(data)
	reply(w, map[string]any{"output": string(raw)})
}

// Exercise actual CLI help and every published structured JSON flag schema.
// Frozen registration comparison elsewhere additionally pins flag types/defaults,
// aliases, identities, scopes, and hidden flag metadata; help alone is not parity.
func TestFrozenDiscovery(t *testing.T) {
	inv := frozen(t)
	f := nextfixture.NewFixture(t, nil)
	var commands []string
	for name := range inv.Commands {
		if strings.HasPrefix(name, "sheets ") {
			commands = append(commands, name)
		}
	}
	sort.Strings(commands)
	for _, name := range commands {
		t.Run(name, func(t *testing.T) {
			r := f.Run(append(strings.Fields(name), "--help")...)
			require.Equal(t, 0, r.ExitCode, r.Stderr)
			for flag, contract := range inv.Commands[name].Flags {
				if !contract.Hidden && flag != "profile" {
					require.Contains(t, r.Stdout, "--"+flag, name+" flag "+flag)
				}
			}
			for flag, schema := range inv.Shortcuts[name].FlagSchemas {
				r := f.Run(append(strings.Fields(name), "--print-schema", "--flag-name", flag)...)
				require.Equal(t, schema, success(t, r), name+" schema "+flag)
			}
		})
	}
}

type scenario struct {
	Command string
	Tool    string
	Args    []string
	Execute bool
}

func cases(t *testing.T) []scenario {
	t.Helper()
	b, e := os.ReadFile("cases.json")
	require.NoError(t, e)
	var out []scenario
	require.NoError(t, json.Unmarshal(b, &out))
	return out
}

// Every frozen shortcut has an explicit scenario. The runtime checks below use
// the production authsidecar process and a backend that never forwards requests.
func TestShortcutRequests(t *testing.T) {
	inv := frozen(t)
	scenarios := cases(t)
	covered := map[string]bool{}
	for _, tc := range scenarios {
		require.False(t, covered[tc.Command], "duplicate scenario")
		covered[tc.Command] = true
		t.Run(tc.Command, func(t *testing.T) {
			var mu sync.Mutex
			var requests []map[string]any
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.True(t, strings.HasPrefix(r.URL.Path, "/open-apis/sheet_ai/v2/spreadsheets/shtFixture/tools/invoke_"), r.URL.Path)
				b, e := io.ReadAll(r.Body)
				require.NoError(t, e)
				body := decode(t, string(b))
				body["path"] = r.URL.Path
				mu.Lock()
				requests = append(requests, body)
				mu.Unlock()
				toolReply(w, map[string]any{"fixture_marker": "preserve 原样", "revision": 17, "sheets": []any{map[string]any{"sheet_id": "sh1", "title": "Sheet1", "index": 0, "charts": []any{map[string]any{"chart_id": "chart1", "details": map[string]any{"snapshot": map[string]any{"plotArea": map[string]any{"plot": map[string]any{"type": "line"}}, "data": map[string]any{"type": "range", "direction": "column"}}}}}}}, "ranges": []any{}, "csv": "name,value\n原样,001\n", "has_more": false, "status": "success", "transaction_id": "transaction1"})
			}))
			require.NoError(t, os.WriteFile(filepath.Join(f.WorkDir, "input.csv"), []byte("name,value\n原样,001\n"), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(f.WorkDir, "image.png"), []byte("synthetic upload bytes"), 0600))
			args := append([]string{}, tc.Args...)
			if inv.Commands["sheets "+tc.Command].Risk == "high-risk-write" {
				args = append(args, "--yes")
			}
			data := success(t, f.Run(append(args, "--dry-run")...))["data"].(map[string]any)
			mu.Lock()
			require.Empty(t, requests, "dry-run must not reach business backend")
			mu.Unlock()
			if tc.Tool == "" {
				expected := map[string]string{
					"+dim-move":        "/open-apis/sheets/v3/spreadsheets/shtFixture/sheets/sh1/move_dimension",
					"+table-get":       "/open-apis/sheet_ai/v2/spreadsheets/shtFixture/tools/invoke_read",
					"+table-put":       "/open-apis/sheet_ai/v2/spreadsheets/shtFixture/tools/invoke_write",
					"+workbook-create": "/open-apis/sheets/v3/spreadsheets",
					"+workbook-export": "/open-apis/drive/v1/export_tasks",
					"+workbook-import": "/open-apis/drive/v1/medias/upload_all",
					"+cells-set-image": "/open-apis/drive/v1/medias/upload_all",
				}
				api := data["api"].([]any)
				require.NotEmpty(t, api)
				first := api[0].(map[string]any)
				require.Equal(t, "POST", first["method"])
				require.Equal(t, expected[tc.Command], first["url"])
				if tc.Command == "+workbook-import" {
					require.Len(t, api, 4)
					require.Equal(t, "GET", api[3].(map[string]any)["method"])
					require.Equal(t, "sheet", api[2].(map[string]any)["body"].(map[string]any)["type"])
				}
				if tc.Command == "+cells-set-image" {
					require.Equal(t, "set_cell_range", api[len(api)-1].(map[string]any)["body"].(map[string]any)["tool_name"])
				}
			}
			if tc.Tool != "" {
				require.Equal(t, tc.Tool, data["tool_name"])
				input := data["tool_input"].(map[string]any)
				require.Equal(t, "shtFixture", input["excel_id"])
				api := data["api"].([]any)[0].(map[string]any)
				require.Equal(t, "POST", api["method"])
				kind := "write"
				if inv.Commands["sheets "+tc.Command].Risk == "read" {
					kind = "read"
				}
				require.Equal(t, "/open-apis/sheet_ai/v2/spreadsheets/shtFixture/tools/invoke_"+kind, api["url"])
				wire := api["body"].(map[string]any)
				require.Equal(t, input, decode(t, wire["input"].(string)))
				if tc.Execute {
					result := f.Run(args...)
					if tc.Tool == "batch_update" || tc.Tool == "history_revert" {
						require.Equal(t, 22, result.ExitCode, result.Stdout+result.Stderr)
						out := decode(t, result.Stdout)
						require.Equal(t, false, out["ok"])
						require.Equal(t, "unknown", out["mutation"].(map[string]any)["outcome"], "batch and async acknowledgements cannot prove complete application")
					} else {
						success(t, result)
					}
					mu.Lock()
					defer mu.Unlock()
					require.NotEmpty(t, requests)
					last := requests[len(requests)-1]
					require.Equal(t, tc.Tool, last["tool_name"])
					require.Equal(t, api["url"], last["path"])
					if tc.Command == "+chart-config-update" || tc.Command == "+chart-data-update" || tc.Command == "+batch-chart-update" {
						require.Contains(t, last["input"], "chart1")
					} else {
						require.Equal(t, input, decode(t, last["input"].(string)))
					}
				}
			}
		})
	}
	for name := range inv.Shortcuts {
		if strings.HasPrefix(name, "sheets ") {
			require.True(t, covered[strings.TrimPrefix(name, "sheets ")], "missing behavioral scenario: "+name)
		}
	}
	require.Len(t, covered, 91)
}
