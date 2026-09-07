// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package doc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestDocsReferenceMapPreservesRichValues(t *testing.T) {
	const raw = `{"widget":{"r1":{"id":9007199254740993,"fraction":0.1234567890123456789,"literal":"001","empty":"","nothing":null,"items":[false,"中文",9007199254740993]}}}`
	for _, operation := range []string{"create", "update"} {
		t.Run(operation, func(t *testing.T) {
			runtime := newUpdateShortcutTestRuntime(t, "", map[string]string{"content": `<p>中文</p>`, "reference-map": raw})
			var body map[string]interface{}
			var err error
			if operation == "update" {
				body, _, err = buildUpdateBodyWithPreparedInput(runtime)
			} else {
				body, _, err = buildCreateBodyWithPreparedInput(runtime)
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(body["reference_map"])
			if err != nil {
				t.Fatal(err)
			}
			for _, literal := range []string{`9007199254740993`, `0.1234567890123456789`, `"001"`, `"empty":""`, `"nothing":null`, `中文`} {
				if !strings.Contains(string(encoded), literal) {
					t.Fatalf("lost %s in %s", literal, encoded)
				}
			}
		})
	}
}

func TestDocsInputRejectsNonObjectAndTrailingJSON(t *testing.T) {
	for _, raw := range []string{"null", "[]", "{} {}", "{\"x\":1} trailing", ""} {
		_, err := parseDocsInputObject(raw, "--reference-map")
		p, ok := errs.ProblemOf(err)
		if !ok || p.Category != errs.CategoryValidation {
			t.Fatalf("input %q: expected typed validation, got %v", raw, err)
		}
	}
}

func TestDocsExplicitZeroRevisionIsPreserved(t *testing.T) {
	fetch := newFetchShortcutTestRuntime(t, "", map[string]string{"revision-id": "0"})
	if got := buildFetchBody(fetch)["revision_id"]; got != 0 {
		t.Fatalf("fetch revision = %#v", got)
	}
	update := newUpdateShortcutTestRuntime(t, "", map[string]string{"revision-id": "0"})
	if got := buildUpdateBody(update)["revision_id"]; got != 0 {
		t.Fatalf("update revision = %#v", got)
	}
}

func TestDocsFetchRejectsIgnoredFullReadBounds(t *testing.T) {
	for _, flag := range []string{"start-block-id", "end-block-id", "keyword", "context-before", "context-after", "max-depth"} {
		t.Run(flag, func(t *testing.T) {
			value := "1"
			runtime := newFetchShortcutTestRuntime(t, "", map[string]string{"scope": "full", flag: value})
			err := validateFetchV2(context.Background(), runtime)
			p, ok := errs.ProblemOf(err)
			if !ok || p.Category != errs.CategoryValidation {
				t.Fatalf("expected bounded scope validation for %s, got %v", flag, err)
			}
		})
	}
	for _, flag := range []string{"context-before", "context-after", "max-depth", "revision-id"} {
		runtime := newFetchShortcutTestRuntime(t, "", map[string]string{flag: "-2"})
		if err := validateFetchV2(context.Background(), runtime); err == nil {
			t.Fatalf("accepted %s=-2", flag)
		}
	}
}

func TestDocsSearchPreservesNumericTimeRanges(t *testing.T) {
	request, err := buildDocsSearchRequest("中文", `{"open_time":{"start":1700000000,"end":null},"create_time":{"start":"","end":"2026-01-01T00:00:00Z"},"opaque":9007199254740993}`, "continuation", "2")
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{`"start":1700000000`, `"end":null`, `"start":""`, `9007199254740993`, `"page_token":"continuation"`} {
		if !strings.Contains(string(encoded), literal) {
			t.Fatalf("lost %s in %s", literal, encoded)
		}
	}
}
