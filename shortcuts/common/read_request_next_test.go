// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestReadOnlyAPITypedPreservesRequestContextAndResponse(t *testing.T) {
	runtime, registry := newCallAPITypedRuntime(t)
	type key struct{}
	original, cancel := context.WithTimeout(context.WithValue(runtime.ctx, key{}, "preserved"), time.Minute)
	defer cancel()
	runtime.ctx = original
	registry.Register(&httpmock.Stub{Method: "POST", URL: "/open-apis/fixture/fetch", Body: map[string]any{"code": 0, "data": map[string]any{"id": "synthetic"}}, OnMatch: func(req *http.Request) {
		if req.Context().Value(key{}) != "preserved" {
			t.Fatal("request context lost")
		}
		expected, _ := original.Deadline()
		actual, ok := req.Context().Deadline()
		if !ok || !actual.Equal(expected) {
			t.Fatal("deadline lost")
		}
	}})
	result, err := runtime.CallReadOnlyAPITyped("POST", "/open-apis/fixture/fetch", nil, map[string]any{"format": "xml"})
	if err != nil || result["id"] != "synthetic" {
		t.Fatalf("%v %v", result, err)
	}
	if runtime.ctx != original {
		t.Fatal("runtime context changed")
	}
}
func TestReadOnlyAPITypedPreservesTypedFailure(t *testing.T) {
	runtime, registry := newCallAPITypedRuntime(t)
	registry.Register(&httpmock.Stub{Method: "POST", URL: "/open-apis/fixture/fetch", Headers: http.Header{"X-Tt-Logid": []string{"synthetic-log"}}, Body: map[string]any{"code": 1061044, "msg": "synthetic failure"}})
	_, err := runtime.CallReadOnlyAPITyped("POST", "/open-apis/fixture/fetch", nil, nil)
	problem, ok := errs.ProblemOf(err)
	if !ok || problem.Code != 1061044 || problem.LogID != "synthetic-log" {
		t.Fatalf("lost typed response evidence: %v", err)
	}
}
