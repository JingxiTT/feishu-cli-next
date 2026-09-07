// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"encoding/json"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
)

func TestReplacementNoopUsesWholeProvidedObject(t *testing.T) {
	current := sampleItem("greet", "id1")
	current["description"] = map[string]interface{}{"default_value": "你好", "i18n": map[string]interface{}{"en_us": "Hi=there"}}
	if !unchangedCommand(current, buildSlashCommandBody("", "你好", map[string]string{"en_us": "Hi=there"}, "")) {
		t.Fatal("equivalent localized replacement is not a no-op")
	}
	if unchangedCommand(current, buildSlashCommandBody("", "你好", nil, "")) {
		t.Fatal("omitted i18n would be removed by whole-description replacement")
	}
	if !unchangedCommand(current, buildSlashCommandBody("", "", nil, "skill_outlined")) {
		t.Fatal("omitted description must not affect icon-only no-op")
	}
	current["description"].(map[string]interface{})["future_field"] = "preserve"
	if unchangedCommand(current, buildSlashCommandBody("", "你好", map[string]string{"en_us": "Hi=there"}, "")) {
		t.Fatal("unmodeled replacement content cannot be declared unchanged")
	}
}

func TestUpdateByNameNoopMakesNoPatch(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, appTestConfig())
	reg.Register(listStub([]interface{}{sampleItem("greet", "id1")}))
	if err := mountAndRun(t, SlashCommandUpdate, []string{"+slash-command-update", "--command", "greet", "--description", "desc of greet", "--as", "bot"}, f, stdout); err != nil {
		t.Fatal(err)
	}
	var env struct {
		Data struct {
			Action string `json:"action"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if env.Data.Action != "unchanged" {
		t.Fatalf("%s", stdout.String())
	}
}

func TestResolutionRejectsAmbiguousMalformedAndIncomplete(t *testing.T) {
	for name, data := range map[string]interface{}{
		"ambiguous":  map[string]interface{}{"items": []interface{}{sampleItem("greet", "id1"), sampleItem("greet", "id2")}},
		"incomplete": map[string]interface{}{"items": []interface{}{sampleItem("greet", "id1")}, "has_more": true},
		"malformed":  map[string]interface{}{"items": "not-an-array"},
		"missing-id": map[string]interface{}{"items": []interface{}{map[string]interface{}{"command": "greet"}}},
	} {
		t.Run(name, func(t *testing.T) {
			f, stdout, _, reg := cmdutil.TestFactory(t, appTestConfig())
			reg.Register(&httpmock.Stub{Method: "GET", URL: slashCommandBasePath, Body: map[string]interface{}{"code": 0, "data": data}})
			err := mountAndRun(t, SlashCommandUpdate, []string{"+slash-command-update", "--command", "greet", "--description", "new", "--as", "bot"}, f, stdout)
			p, ok := errs.ProblemOf(err)
			if !ok || p.Category != errs.CategoryAPI || p.Subtype != errs.SubtypeInvalidResponse {
				t.Fatalf("expected invalid_response, got %v", err)
			}
		})
	}
}
