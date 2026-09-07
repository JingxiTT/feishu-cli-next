// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT
package im

import "testing"

func TestEqualMessageJSONPreservesReplacementContent(t *testing.T) {
	for _, tt := range []struct {
		name, left, right string
		equal             bool
	}{
		{"text", `{"text":"你好"}`, ` { "text": "你好" } `, true},
		{"localized post", `{"en_us":{"title":"","content":[[{"tag":"text","text":"x","style":["bold"]}]],"files":[]}}`, `{"en_us":{"files":[],"content":[[{"style":["bold"],"text":"x","tag":"text"}]],"title":""}}`, true},
		{"attachment removal", `{"files":[{"file_key":"file_a"}]}`, `{"files":[]}`, false},
		{"omitted is not clear", `{}`, `{"files":[]}`, false},
		{"null is not empty", `{"text":null}`, `{"text":""}`, false},
		{"styles", `{"style":["bold"]}`, `{"style":["italic"]}`, false},
		{"array order", `["a","b"]`, `["b","a"]`, false},
		{"large number", `{"n":9007199254740992}`, `{"n":9007199254740993}`, false},
		{"number type", `{"n":1}`, `{"n":"1"}`, false},
		{"invalid", `broken`, `broken`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := equalMessageJSON(tt.left, tt.right); got != tt.equal {
				t.Fatalf("equality %v, want %v", got, tt.equal)
			}
		})
	}
}

func TestChatDescriptionExplicitEmptyIsClear(t *testing.T) {
	rt := newTestRuntimeContext(t, map[string]string{"chat-id": "oc_fixture", "description": ""}, nil)
	body := buildUpdateChatBody(rt)
	if value, ok := body["description"]; !ok || value != "" {
		t.Fatalf("explicit clear lost: %#v", body)
	}
	if _, ok := body["name"]; ok {
		t.Fatalf("omitted name changed: %#v", body)
	}
}
