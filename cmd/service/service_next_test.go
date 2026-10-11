// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import (
	"testing"

	"github.com/larksuite/cli/internal/meta"
)

func TestMethodPaginates(t *testing.T) {
	for _, tt := range []struct {
		name   string
		params map[string]meta.Field
		want   bool
	}{
		{name: "no parameters"},
		{name: "query token", params: map[string]meta.Field{"page_token": {Location: "query"}}, want: true},
		{name: "path token", params: map[string]meta.Field{"page_token": {Location: "path"}}, want: true},
		{name: "body token", params: map[string]meta.Field{"page_token": {Location: "body"}}},
		{name: "header token", params: map[string]meta.Field{"page_token": {Location: "header"}}},
		{name: "unnamed token", params: map[string]meta.Field{"page_token": {}}},
		{name: "other parameter", params: map[string]meta.Field{"other": {Name: "page_token", Location: "query"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := methodPaginates(meta.Method{Parameters: tt.params}); got != tt.want {
				t.Fatalf("methodPaginates() = %v, want %v", got, tt.want)
			}
		})
	}
}
