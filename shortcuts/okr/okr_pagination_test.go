// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr

import (
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/tidwall/gjson"
)

func TestAdvanceOKRPageRejectsIncompleteCollection(t *testing.T) {
	for _, repeated := range []bool{false, true} {
		seen := map[string]bool{}
		token := ""
		if repeated {
			token = "opaque+token="
			seen[token] = true
		}
		err := advanceOKRPage(true, token, seen)
		p, ok := errs.ProblemOf(err)
		if !ok || p.Subtype != errs.SubtypeInvalidResponse {
			t.Fatalf("expected typed invalid response, got %v", err)
		}
	}
	seen := map[string]bool{}
	if err := advanceOKRPage(true, "opaque+token=", seen); err != nil || !seen["opaque+token="] {
		t.Fatalf("opaque token: %v", err)
	}
	if err := advanceOKRPage(false, "opaque+token=", seen); err != nil {
		t.Fatal(err)
	}
}

func TestReorderNoopWithKnownPositions(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, reorderTestConfig(t))
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/okr/v2/cycles/123/objectives", Body: map[string]any{"code": 0, "data": map[string]any{"items": []any{map[string]any{"id": "1", "position": 1}}, "has_more": false}}})
	// No mutation stub: an unnecessary PUT makes this regression fail.
	err := runReorderShortcut(t, f, stdout, []string{"+reorder", "--level", "objective", "--cycle-id", "123", "--ops", `[{"id":"1","position":1}]`})
	if err != nil {
		t.Fatal(err)
	}
	if !gjson.Get(stdout.String(), "data.noop").Bool() {
		t.Fatalf("missing no-op: %s", stdout)
	}
}

func TestWeightNoopWithKnownWeights(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, weightTestConfig(t))
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/okr/v2/cycles/123/objectives", Body: map[string]any{"code": 0, "data": map[string]any{"items": []any{map[string]any{"id": "1", "weight": 1}}, "has_more": false}}})
	err := runWeightShortcut(t, f, stdout, []string{"+weight", "--level", "objective", "--cycle-id", "123", "--weights", `[{"id":"1","weight":1}]`})
	if err != nil {
		t.Fatal(err)
	}
	if !gjson.Get(stdout.String(), "data.noop").Bool() {
		t.Fatalf("missing no-op: %s", stdout)
	}
}
