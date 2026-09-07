// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/httpmock"
	"github.com/larksuite/cli/shortcuts/common"
)

func TestMinutesReplacementPreservesContent(t *testing.T) {
	const content = "  中文\n\tvalue  "
	for _, spec := range []minuteTodoSpec{
		{Operation: "add", Content: content, IsDone: boolPointer(false)},
		{Operation: "update", TodoID: "001", Content: content, IsDone: boolPointer(true)},
	} {
		item, err := buildMinuteTodoItem(spec)
		if err != nil || item["content"] != content {
			t.Fatalf("todo content = %#v, %v", item, err)
		}
	}
	words, err := parseReplaceWords(`[{"source_word":"  中文  ","target_word":""},{"source_word":"中文","target_word":" 值 "}]`)
	if err != nil || words[0]["source_word"] != "  中文  " || words[0]["target_word"] != "" || words[1]["target_word"] != " 值 " {
		t.Fatalf("words = %#v, %v", words, err)
	}
	for _, raw := range []string{`[{"source_word":"x"}]`, `[{"source_word":"x","target_word":null}]`} {
		_, err := parseReplaceWords(raw)
		var p *errs.ValidationError
		if !errors.As(err, &p) || p.Param != "--replace-words" {
			t.Fatalf("missing deletion intent accepted: %v", err)
		}
	}
}

func boolPointer(value bool) *bool { return &value }

func TestMinutesWordCountsRejectAmbiguousEvidence(t *testing.T) {
	for _, raw := range []string{
		`{}`, `{"replace_word_counts":null}`, `{"replace_word_counts":[{"source_word":"x","replace_count":null}]}`,
		`{"replace_word_counts":[{"source_word":"x","replace_count":-1}]}`,
		`{"replace_word_counts":[{"source_word":"x","replace_count":1.5}]}`,
		`{"replace_word_counts":[{"source_word":"x","replace_count":"1"},{"source_word":"x","replace_count":"0"}]}`,
	} {
		var data map[string]any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			t.Fatal(err)
		}
		if _, ok := parseReplaceWordCounts(data); ok {
			t.Fatalf("accepted ambiguous counts %s", raw)
		}
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(`{"replace_word_counts":[{"source_word":" x ","replace_count":"9007199254740993"}]}`), &data)
	got, ok := parseReplaceWordCounts(data)
	if !ok || got[" x "] != 9007199254740993 {
		t.Fatalf("lost exact count/source: %#v", got)
	}
}

func TestMinutesSkipUnchanged(t *testing.T) {
	const content = "  中文\n\tvalue  "
	for _, tc := range []struct {
		name     string
		shortcut common.Shortcut
		args     []string
		data     map[string]any
		suffix   string
	}{
		{"title", MinutesUpdate, []string{"--topic", content}, map[string]any{"minute": map[string]any{"title": content}}, ""},
		{"summary", MinutesSummary, []string{"--summary", content}, map[string]any{"summary": content}, "/artifacts"},
		{"todo", MinutesTodo, []string{"--operation", "update", "--todo-id", "001", "--todo", content, "--is-done=false"}, map[string]any{"minute_todos": []any{map[string]any{"todo_id": "001", "content": content, "is_done": false}}}, "/artifacts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			f, out, _, reg := cmdutil.TestFactory(t, defaultConfig())
			warmTokenCache(t)
			reg.Register(&httpmock.Stub{Method: "GET", URL: "/open-apis/minutes/v1/minutes/tok001" + tc.suffix, Body: map[string]any{"code": 0, "data": tc.data}})
			args := []string{tc.shortcut.Command, "--minute-token", "tok001", "--as", "user", "--skip-unchanged"}
			args = append(args, tc.args...)
			if err := mountAndRun(t, tc.shortcut, args, f, out); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), `"unchanged": true`) {
				t.Fatal(out.String())
			}
		})
	}
}

func TestMinutesTranscriptDirectoryPreservesUTF8(t *testing.T) {
	got := sanitizeDetailDirName(strings.Repeat("中", 100), "tok001")
	if !utf8.ValidString(got) || !strings.HasSuffix(got, "-tok001") {
		t.Fatalf("invalid transcript directory: %q", got)
	}
}
