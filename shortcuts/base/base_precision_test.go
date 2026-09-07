// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBaseJSONPreservesDecimalTokensAndFieldPresence(t *testing.T) {
	raw := `{"number":9007199254740993,"decimal":0.1234567890123456789,"empty":"","null":null,"list":[],"formula":"=A1","date":"2026-01-01 00:00:00","identity":[{"id":"ou_synthetic"}]}`
	body, err := parseJSONObject(&parseCtx{}, raw, "json")
	if err != nil {
		t.Fatal(err)
	}
	if body["number"] != json.Number("9007199254740993") || body["decimal"] != json.Number("0.1234567890123456789") {
		t.Fatalf("numeric precision lost: %#v", body)
	}
	if _, ok := body["omitted"]; ok {
		t.Fatal("omitted value manufactured")
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"empty":""`, `"null":null`, `"list":[]`, `"date":"2026-01-01 00:00:00"`, `"id":"ou_synthetic"`} {
		if !strings.Contains(string(encoded), fragment) {
			t.Fatalf("missing %s in %s", fragment, encoded)
		}
	}
}

func TestBaseJSONRejectsTrailingValueAndRetainsSyntaxCause(t *testing.T) {
	for _, raw := range []string{`{} {}`, `{"unfinished":`} {
		if _, err := parseJSONObject(&parseCtx{}, raw, "json"); err == nil {
			t.Fatalf("accepted invalid JSON %s", raw)
		}
	}
}

func TestBaseRoleResponsePreservesNestedLargeNumbers(t *testing.T) {
	factory, stdout, _ := newExecuteFactory(t)
	runtime := newBaseTestRuntime(nil, nil, nil)
	runtime.Factory = factory
	raw := []byte(`{"code":0,"data":{"code":0,"data":{"quota":9007199254740993}}}`)
	if err := handleRoleResponse(runtime, raw, "read role"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "9007199254740993") {
		t.Fatalf("precision lost: %s", stdout.String())
	}
}
