// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package okr

import (
	"errors"
	"strconv"
	"strings"
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

func TestReorderRejectsDuplicateNativeIDs(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, reorderTestConfig(t))
	reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/okr/v2/cycles/123/objectives", Body: map[string]any{"code": 0, "data": map[string]any{"items": []any{map[string]any{"id": "1", "position": 1}, map[string]any{"id": "1", "position": 2}}, "has_more": false}}})
	err := runReorderShortcut(t, f, stdout, []string{"+reorder", "--level", "objective", "--cycle-id", "123", "--ops", `[{"id":"1","position":1}]`})
	p, ok := errs.ProblemOf(err)
	if !ok || p.Subtype != errs.SubtypeInvalidResponse {
		t.Fatalf("expected typed invalid response, got %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected success output %s", stdout)
	}
}

func TestMutationSnapshotRejectsInvalidAndDuplicateIDs(t *testing.T) {
	for _, ids := range [][]string{{"1", "1"}, {""}, {"not-an-id"}, {"0"}, {"-1"}, {"9223372036854775808"}} {
		t.Run(strings.Join(ids, "_"), func(t *testing.T) {
			items := make([]Objective, len(ids))
			for i, id := range ids {
				items[i].ID = id
			}
			_, err := buildReorderedIDs(items, []reorderOp{{ID: "1", Position: 1}}, len(items))
			p, ok := errs.ProblemOf(err)
			if !ok || p.Subtype != errs.SubtypeInvalidResponse || p.Category != errs.CategoryInternal {
				t.Fatalf("reorder should reject native snapshot: %v", err)
			}
			err = validateOKRMutationIDs(items)
			p, ok = errs.ProblemOf(err)
			if !ok || p.Subtype != errs.SubtypeInvalidResponse || p.Category != errs.CategoryInternal {
				t.Fatalf("validator should reject native snapshot: %v", err)
			}
		})
	}
	items := []Objective{{ID: "001"}, {ID: "9223372036854775807"}}
	if err := validateOKRMutationIDs(items); err != nil {
		t.Fatal(err)
	}
	if items[0].ID != "001" {
		t.Fatal("native ID was normalized")
	}
	err := validateOKRMutationIDs([]Objective{{ID: "invalid"}})
	var cause *strconv.NumError
	if !errors.As(err, &cause) {
		t.Fatalf("missing parse cause: %v", err)
	}
}
