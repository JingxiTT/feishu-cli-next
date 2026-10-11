// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets_test

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

// Generated schemas retain every nested catalog request/response field and
// type. Descriptions may be normalized by the shared schema renderer.
func assertFields(t *testing.T, fields, schema map[string]any) {
	t.Helper()
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range fields {
		t.Run(name, func(t *testing.T) {
			expected := raw.(map[string]any)
			actual, ok := props[name].(map[string]any)
			require.True(t, ok, "missing catalog field")
			if typ, ok := expected["type"]; ok {
				require.Equal(t, typ, actual["type"])
			}
			if nested, ok := expected["properties"].(map[string]any); ok {
				if expected["type"] == "array" {
					actual = actual["items"].(map[string]any)
				}
				assertFields(t, nested, actual)
			}
			if item, ok := expected["items"].(map[string]any); ok {
				a, ok := actual["items"].(map[string]any)
				require.True(t, ok)
				if typ, ok := item["type"]; ok {
					require.Equal(t, typ, a["type"])
				}
				if p, ok := item["properties"].(map[string]any); ok {
					assertFields(t, p, a)
				}
			}
		})
	}
}
func TestGeneratedSchemasAndRequests(t *testing.T) {
	file, e := os.Open("../../../baseline/v1.0.97/meta_data.json.gz")
	require.NoError(t, e)
	defer file.Close()
	z, e := gzip.NewReader(file)
	require.NoError(t, e)
	defer z.Close()
	var catalog struct {
		Services []struct {
			Name      string
			Resources map[string]struct{ Methods map[string]map[string]any }
		}
	}
	require.NoError(t, json.NewDecoder(z).Decode(&catalog))
	bodies := map[string]string{
		"spreadsheets create":              `{"title":"Synthetic 原样","without_mount":true}`,
		"spreadsheets patch":               `{"title":"Changed 原样"}`,
		"spreadsheet.sheet.filters create": `{"range":"sh1!A1:C10","col":"B","condition":{"filter_type":"multiValue","expected":["001","原样"]}}`,
		"spreadsheet.sheet.filters update": `{"col":"B","condition":{"filter_type":"multiValue","expected":["002"]}}`,
		"spreadsheet.sheets find":          `{"find":"原样","find_condition":{"range":"sh1!A1:C10","match_case":true,"match_entire_cell":false,"search_by_regex":false,"include_formulas":true}}`,
	}
	count := 0
	for _, service := range catalog.Services {
		if service.Name != "sheets" {
			continue
		}
		for resource, r := range service.Resources {
			for method, contract := range r.Methods {
				count++
				t.Run(resource+" "+method, func(t *testing.T) {
					calls := 0
					bodyText := bodies[resource+" "+method]
					path := contract["path"].(string)
					path = strings.ReplaceAll(strings.ReplaceAll(path, "{spreadsheet_token}", "shtFixture"), "{sheet_id}", "sh1")
					// Frozen resources include v2 filter/find and v3 workbook endpoints.
					version := "v3"
					if resource == "spreadsheet.sheet.filters" {
						version = "v3"
					}
					f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls++
						require.Equal(t, contract["httpMethod"], r.Method)
						require.Equal(t, "/open-apis/sheets/"+version+"/"+path, r.URL.Path)
						raw, e := io.ReadAll(r.Body)
						require.NoError(t, e)
						if bodyText != "" {
							require.JSONEq(t, bodyText, string(raw))
						} else {
							require.Empty(t, raw)
						}
						if resource == "spreadsheets" && method == "get" {
							require.Equal(t, "open_id", r.URL.Query().Get("user_id_type"))
						}
						reply(w, map[string]any{"fixture_marker": "原样", "spreadsheet": map[string]any{"token": "shtFixture", "title": "Synthetic 原样"}})
					}))
					schema := success(t, f.Run("schema", "sheets", resource, method))
					require.Equal(t, "sheets "+resource+" "+method, schema["name"])
					input := schema["inputSchema"].(map[string]any)
					props := input["properties"].(map[string]any)
					if fields, ok := contract["parameters"].(map[string]any); ok {
						assertFields(t, fields, props["params"].(map[string]any))
					}
					if fields, ok := contract["requestBody"].(map[string]any); ok {
						assertFields(t, fields, props["data"].(map[string]any))
					}
					if fields, ok := contract["responseBody"].(map[string]any); ok {
						assertFields(t, fields, schema["outputSchema"].(map[string]any))
					}
					params := map[string]string{}
					if strings.Contains(contract["path"].(string), "{spreadsheet_token}") {
						params["spreadsheet_token"] = "shtFixture"
					}
					if strings.Contains(contract["path"].(string), "{sheet_id}") {
						params["sheet_id"] = "sh1"
					}
					if resource == "spreadsheets" && method == "get" {
						params["user_id_type"] = "open_id"
					}
					p, _ := json.Marshal(params)
					args := []string{"sheets", resource, method, "--params", string(p)}
					if bodyText != "" {
						args = append(args, "--data", bodyText)
					}
					if contract["risk"] == "high-risk-write" {
						args = append(args, "--yes")
					}
					success(t, f.Run(append(args, "--dry-run")...))
					require.Zero(t, calls)
					out := success(t, f.Run(args...))
					require.Equal(t, "原样", out["data"].(map[string]any)["fixture_marker"])
					require.Equal(t, 1, calls)
				})
			}
		}
	}
	require.Equal(t, 8, count)
}
