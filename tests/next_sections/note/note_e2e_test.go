// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package note_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const notePath = "/open-apis/vc/v1/notes/note_fixture"

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func detail(kind int) map[string]any {
	return map[string]any{"note": map[string]any{"note_display_type": kind, "creator_id": "ou_fixture", "create_time": "1788775200", "artifacts": []any{map[string]any{"artifact_type": 1, "doc_token": "doc_main"}, map[string]any{"artifact_type": 2, "doc_token": "doc_verbatim"}}, "references": []any{map[string]any{"doc_token": "doc_共享"}, map[string]any{"doc_token": ""}}}}
}
func success(t *testing.T, r nextfixture.Result) gjson.Result {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
	require.True(t, gjson.Valid(r.Stdout), r.Stdout)
	require.True(t, gjson.Get(r.Stdout, "ok").Bool(), r.Stdout)
	require.False(t, gjson.Get(r.Stdout, "mutation").Exists(), r.Stdout)
	return gjson.Get(r.Stdout, "data")
}
func failure(t *testing.T, r nextfixture.Result, code int, kind, subtype string) {
	t.Helper()
	require.Equal(t, code, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
	require.Empty(t, r.Stdout)
	// Runtime progress may precede the pretty-printed typed envelope.
	envelope := ""
	if index := strings.Index(r.Stderr, "{"); index >= 0 {
		envelope = strings.TrimSpace(r.Stderr[index:])
	}
	require.NotEmpty(t, envelope, r.Stderr)
	require.Equal(t, kind, gjson.Get(envelope, "error.type").String(), r.Stderr)
	require.Equal(t, subtype, gjson.Get(envelope, "error.subtype").String(), r.Stderr)
}

func TestNoteDiscoveryAndFrozenFlags(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, command := range []string{"+detail", "+transcript"} {
		r := f.Run("note", command, "--help")
		require.Equal(t, 0, r.ExitCode, r.Stderr)
		for _, flag := range []string{"--note-id", "--as", "--dry-run", "--format", "--jq", "--json"} {
			require.Contains(t, r.Stdout, flag)
		}
		if command == "+transcript" {
			for _, flag := range []string{"--transcript-format", "--locale", "--output", "--overwrite"} {
				require.Contains(t, r.Stdout, flag)
			}
		}
	}
	// Public flag descriptions can match this query in other domains. Consume
	// every advertised page before asserting membership, without assuming rank.
	commands := map[string]bool{}
	for offset := 0; ; {
		r := f.Run("discover", "note", "--offset", strconv.Itoa(offset))
		data := success(t, r)
		for _, entry := range data.Array() {
			command := entry.Get("command").String()
			require.NotEmpty(t, command, r.Stdout)
			require.False(t, commands[command], "duplicate discovery entry %s", command)
			commands[command] = true
		}
		meta := gjson.Get(r.Stdout, "meta")
		require.True(t, meta.Get("complete").Exists(), r.Stdout)
		if meta.Get("complete").Bool() {
			require.Equal(t, int(meta.Get("total").Int()), len(commands), r.Stdout)
			break
		}
		require.True(t, meta.Get("next_offset").Exists(), r.Stdout)
		next := int(meta.Get("next_offset").Int())
		require.Greater(t, next, offset, r.Stdout)
		require.Less(t, next, int(meta.Get("total").Int()), r.Stdout)
		offset = next
	}
	require.True(t, commands["note +detail"], "note +detail missing from complete discovery")
	require.True(t, commands["note +transcript"], "note +transcript missing from complete discovery")
}

func TestNoteDetailIdentitiesTypesAndArtifacts(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		t.Run(identity, func(t *testing.T) {
			var reads atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, notePath, r.URL.Path)
				reads.Add(1)
				reply(w, detail(2))
			}))
			r := f.Run("note", "+detail", "--note-id", "note_fixture", "--as", identity)
			d := success(t, r)
			require.Equal(t, identity, gjson.Get(r.Stdout, "identity").String())
			require.Equal(t, "unified", d.Get("note.note_display_type").String())
			require.Equal(t, "doc_main", d.Get("note.note_doc_token").String())
			require.Equal(t, "doc_verbatim", d.Get("note.verbatim_doc_token").String())
			require.Equal(t, "doc_共享", d.Get("note.shared_doc_tokens.0").String())
			require.Equal(t, int64(1), d.Get("note.shared_doc_tokens.#").Int())
			require.Equal(t, "ou_fixture", d.Get("note.creator_id").String())
			require.NotEmpty(t, d.Get("note.create_time").String())
			path := filepath.Join(f.WorkDir, "detail.json")
			success(t, f.Run("note", "+detail", "--note-id", "note_fixture", "--as", identity, "--save-result", path, "--result-fields", "note.note_display_type"))
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "feishu-cli-next/result/v1", gjson.GetBytes(raw, "schema").String())
			require.Equal(t, "doc_main", gjson.GetBytes(raw, "result.data.note.note_doc_token").String())
			require.Equal(t, identity, gjson.GetBytes(raw, "binding.identity").String())
			require.Equal(t, int32(2), reads.Load())
		})
	}
	for _, kind := range []int{1, 99} {
		t.Run(strconv.Itoa(kind), func(t *testing.T) {
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reply(w, map[string]any{"note": map[string]any{"display_type": kind}})
			}))
			d := success(t, f.Run("note", "+detail", "--note-id", "note_fixture", "--as", "bot"))
			want := "unknown"
			if kind == 1 {
				want = "normal"
			}
			require.Equal(t, want, d.Get("note.note_display_type").String())
			require.True(t, d.Get("note.note_doc_token").Exists())
			require.Equal(t, "", d.Get("note.note_doc_token").String())
			require.False(t, d.Get("note.shared_doc_tokens").Exists())
		})
	}
}

func TestNoteTranscriptFormatsPaginationAndOverwrite(t *testing.T) {
	for _, format := range []string{"markdown", "plain_text"} {
		t.Run(format, func(t *testing.T) {
			chunks := []string{"# 标题\n\n**粗体** 👋\n", "", "speaker:  leading  spaces\n[文档](https://example.invalid/doc)\n"}
			var pages atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				if r.URL.Path == notePath {
					reply(w, detail(2))
					return
				}
				require.Equal(t, notePath+"/unified_note_transcript", r.URL.Path)
				require.Equal(t, format, r.URL.Query().Get("format"))
				require.Equal(t, "ja_jp", r.URL.Query().Get("locale"))
				require.Equal(t, "200", r.URL.Query().Get("page_size"))
				index := int(pages.Add(1)-1) % 3
				if index == 0 {
					require.Empty(t, r.URL.Query().Get("cursor_id"))
				} else {
					require.Equal(t, []string{"", "7648924766078847940", "second"}[index], r.URL.Query().Get("cursor_id"))
				}
				reply(w, map[string]any{"has_more": index < 2, "next_cursor_id": []string{"7648924766078847940", "second", ""}[index], "transcript": map[string]any{format: chunks[index]}})
			}))
			args := []string{"note", "+transcript", "--note-id", "note_fixture", "--as", "user", "--transcript-format", format, "--locale", "ja_jp", "--format", "json"}
			d := success(t, f.Run(args...))
			ext := "md"
			if format == "plain_text" {
				ext = "txt"
			}
			path := filepath.Join(f.WorkDir, "notes", "note_fixture", "unified_transcript."+ext)
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, strings.Join(chunks, ""), string(raw))
			require.Equal(t, int64(len(raw)), d.Get("size_bytes").Int())
			resolved, err := filepath.EvalSymlinks(path)
			require.NoError(t, err)
			require.Equal(t, resolved, d.Get("transcript_file").String())
			require.Equal(t, format, d.Get("transcript_format").String())
			failure(t, f.Run(args...), 2, "validation", "failed_precondition")
			require.Equal(t, int32(3), pages.Load())
			success(t, f.Run(append(args, "--overwrite")...))
			require.Equal(t, int32(6), pages.Load())
			raw2, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, raw, raw2)
		})
	}
}

func TestNoteTranscriptFailuresDoNotSavePartialContent(t *testing.T) {
	cases := []struct {
		name          string
		page          map[string]any
		code          int
		kind, subtype string
	}{
		{"missing content", map[string]any{"has_more": false}, 5, "internal", "invalid_response"},
		{"wrong content type", map[string]any{"has_more": false, "transcript": map[string]any{"markdown": 42}}, 5, "internal", "invalid_response"},
		{"wrong pagination type", map[string]any{"has_more": "true", "transcript": map[string]any{"markdown": "tail"}}, 5, "internal", "invalid_response"},
		{"repeated cursor", map[string]any{"has_more": true, "next_cursor_id": "next", "transcript": map[string]any{"markdown": "tail"}}, 5, "internal", "invalid_response"},
		{"missing cursor", map[string]any{"has_more": true, "transcript": map[string]any{"markdown": "tail"}}, 5, "internal", "invalid_response"},
		{"permission", nil, 3, "authorization", "permission_denied"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var pages atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == notePath {
					reply(w, detail(2))
					return
				}
				if pages.Add(1) == 1 {
					reply(w, map[string]any{"has_more": true, "next_cursor_id": "next", "transcript": map[string]any{"markdown": "first page"}})
					return
				}
				if tc.page == nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"code":121005,"msg":"synthetic permission denied"}`))
					return
				}
				reply(w, tc.page)
			}))
			path := filepath.Join(f.WorkDir, "out.md")
			require.NoError(t, os.WriteFile(path, []byte("preserve existing"), 0600))
			failure(t, f.Run("note", "+transcript", "--note-id", "note_fixture", "--as", "user", "--output", path, "--overwrite"), tc.code, tc.kind, tc.subtype)
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "preserve existing", string(raw))
			require.Equal(t, int32(2), pages.Load())
		})
	}
}

func TestNoteValidationAndNormalNoteRecovery(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{{"note", "+detail"}, {"note", "+detail", "--note-id", "../bad"}, {"note", "+transcript", "--note-id", "note_fixture", "--transcript-format", "html"}} {
		failure(t, f.Run(args...), 2, "validation", "invalid_argument")
	}
	r := f.Run("note", "+transcript", "--note-id", "note_fixture", "--as", "bot")
	require.NotZero(t, r.ExitCode)
	require.Empty(t, r.Stdout)
	require.Contains(t, r.Stderr, "user")
	var reads atomic.Int32
	normal := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, notePath, r.URL.Path)
		reads.Add(1)
		reply(w, detail(1))
	}))
	r = normal.Run("note", "+transcript", "--note-id", "note_fixture", "--as", "user")
	failure(t, r, 2, "validation", "failed_precondition")
	require.Contains(t, r.Stderr, "docs +fetch --doc doc_verbatim")
	require.Equal(t, int32(1), reads.Load())
}
