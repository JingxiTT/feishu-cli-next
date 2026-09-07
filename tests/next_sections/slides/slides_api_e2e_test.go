// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package slides_test

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

// Independent static cases cover all frozen generated Slides endpoints. Generic
// content retains SML bytes and endpoints retain uncommon revision/idempotency
// parameters; workflow semantics are exercised separately in the shortcut suite.
func TestGeneratedSlidesEndpoints(t *testing.T) {
	cases := []struct {
		name, method, path string
		params, body       map[string]any
	}{
		{"xml_presentations create", "POST", "/xml_presentations", nil, map[string]any{"xml_presentation": map[string]any{"content": `<presentation><slide><note>你好</note></slide></presentation>`}}},
		{"xml_presentations get", "GET", "/xml_presentations/pres1", map[string]any{"xml_presentation_id": "pres1", "revision_id": 7, "remove_attr_id": true}, nil},
		{"xml_presentation.slide create", "POST", "/xml_presentations/pres1/slide", map[string]any{"xml_presentation_id": "pres1", "revision_id": 7, "idempotency_key": "unique1"}, map[string]any{"slide": map[string]any{"content": `<slide><data/><note>你好</note></slide>`}, "before_slide_id": "p1"}},
		{"xml_presentation.slide get", "GET", "/xml_presentations/pres1/slide", map[string]any{"xml_presentation_id": "pres1", "slide_id": "p1", "revision_id": 7}, nil},
		{"xml_presentation.slide delete", "DELETE", "/xml_presentations/pres1/slide", map[string]any{"xml_presentation_id": "pres1", "slide_id": "p1", "revision_id": 7, "idempotency_key": "unique1"}, nil},
		{"xml_presentation.slide replace", "POST", "/xml_presentations/pres1/slide/replace", map[string]any{"xml_presentation_id": "pres1", "slide_id": "p1", "revision_id": 7}, map[string]any{"idempotency_key": "unique1", "parts": []any{map[string]any{"action": "str_replace", "old_str": "你好", "new_str": "世界"}}}},
		{"xml_presentation.slide_image render", "POST", "/slide_image/render", nil, map[string]any{"content": "<slide><data/></slide>"}},
		{"xml_presentation.slide_image list", "POST", "/xml_presentations/pres1/slide_images", map[string]any{"xml_presentation_id": "pres1"}, map[string]any{"slide_ids": []any{"p1"}, "slide_numbers": []any{2}}},
		{"xml_presentation.history list", "GET", "/xml_presentations/pres1/histories", map[string]any{"xml_presentation_id": "pres1", "page_size": 1, "page_token": "next"}, nil},
		{"xml_presentation.history revert", "POST", "/xml_presentations/pres1/history/revert", map[string]any{"xml_presentation_id": "pres1"}, map[string]any{"history_version_id": "9007199254740993"}},
		{"xml_presentation.history revert_status", "GET", "/xml_presentations/pres1/history/revert_status", map[string]any{"xml_presentation_id": "pres1", "task_id": "task1"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != tc.method || r.URL.Path != "/open-apis/slides_ai/v1"+tc.path {
					t.Error(r.Method, r.URL)
				}
				for key, value := range tc.params {
					if key == "xml_presentation_id" {
						continue
					}
					want, _ := json.Marshal(value)
					if s, ok := value.(string); ok {
						want = []byte(s)
					}
					if r.URL.Query().Get(key) != string(want) {
						t.Error(key, r.URL)
					}
				}
				raw, _ := io.ReadAll(r.Body)
				if tc.body != nil {
					var body map[string]any
					_ = json.Unmarshal(raw, &body)
					want, _ := json.Marshal(tc.body)
					var expected map[string]any
					_ = json.Unmarshal(want, &expected)
					if !reflect.DeepEqual(body, expected) {
						t.Error(string(raw), string(want))
					}
				}
				response := map[string]any{"xml_presentation_id": "pres1", "slide_id": "p1", "revision_id": 8, "items": []any{}, "has_more": false, "content": "你好", "empty": "", "nullable": nil}
				if tc.name == "xml_presentation.history revert" {
					response["task_id"] = "task1"
				}
				reply(w, response)
			}))
			args := strings.Fields(tc.name)
			if tc.params != nil {
				b, _ := json.Marshal(tc.params)
				args = append(args, "--params", string(b))
			}
			if tc.body != nil {
				b, _ := json.Marshal(tc.body)
				args = append(args, "--data", string(b))
			}
			r := run(f, args...)
			if tc.name == "xml_presentation.history revert" {
				if r.ExitCode != 22 || !strings.Contains(r.Stdout, "task1") {
					t.Fatal(r)
				}
			} else {
				success(t, r)
			}
			if calls != 1 || !strings.Contains(r.Stdout, `"nullable":null`) || !strings.Contains(r.Stdout, `"empty":""`) {
				t.Fatal(r, calls)
			}
		})
	}
}

func TestGeneratedHistoryBoundedPagination(t *testing.T) {
	calls := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		reply(w, map[string]any{"items": []any{map[string]any{"history_version_id": "9007199254740993"}}, "has_more": true, "page_token": "next"})
	}))
	r := run(f, "xml_presentation.history", "list", "--params", `{"xml_presentation_id":"pres1","page_size":1}`, "--page-all", "--page-limit", "1", "--page-delay", "0", "--save-result", "history.json")
	success(t, r)
	if calls != 1 || !strings.Contains(r.Stdout, "next") || !strings.Contains(r.Stdout, `"has_more":true`) {
		t.Fatal(r, calls)
	}
}
