// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package minutes_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func decoded(t *testing.T, raw string) map[string]any {
	t.Helper()
	var v map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &v), raw)
	return v
}
func success(t *testing.T, r nextfixture.Result) map[string]any {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "stdout=%s stderr=%s", r.Stdout, r.Stderr)
	v := decoded(t, r.Stdout)
	require.Equal(t, true, v["ok"])
	return v
}
func noReceipts(t *testing.T, f *nextfixture.Fixture) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
	require.NoError(t, err)
	require.Empty(t, files)
}

func TestSearchBoundedPageAndArtifact(t *testing.T) {
	for _, identity := range []string{"bot", "user"} {
		t.Run(identity, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/open-apis/minutes/v1/minutes/search", r.URL.Path)
				require.Equal(t, "30", r.URL.Query().Get("page_size"))
				require.Equal(t, "cursor001", r.URL.Query().Get("page_token"))
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "会议", body["query"])
				filter := body["filter"].(map[string]any)
				require.Equal(t, []any{"ou_owner"}, filter["owner_ids"])
				require.Equal(t, []any{"ou_participant"}, filter["participant_ids"])
				require.Contains(t, filter, "create_time")
				reply(w, map[string]any{"items": []any{map[string]any{"minute_token": "tok001", "title": "会议", "description": "  完整\n内容  ", "optional": nil, "duration": "001"}}, "has_more": true, "page_token": "cursor002", "notice": "bounded page"})
			}))
			result := success(t, f.Run("minutes", "+search", "--as", identity, "--query", "会议", "--owner-ids", "ou_owner", "--participant-ids", "ou_participant", "--start", "2026-01-01", "--end", "2026-01-02", "--page-size", "30", "--page-token", "cursor001", "--result-fields", "title", "--save-result", "search.json"))
			require.EqualValues(t, 1, calls.Load())
			require.NotContains(t, fmt.Sprint(result), "完整")
			artifact, err := os.ReadFile(filepath.Join(f.WorkDir, "search.json"))
			require.NoError(t, err)
			v := decoded(t, string(artifact))
			require.Equal(t, "feishu-cli-next/result/v1", v["schema"])
			data := v["result"].(map[string]any)["data"].(map[string]any)
			require.Equal(t, true, data["has_more"])
			require.Equal(t, "cursor002", data["page_token"])
			item := data["items"].([]any)[0].(map[string]any)
			require.Equal(t, "  完整\n内容  ", item["description"])
			require.Equal(t, "001", item["duration"])
			require.Contains(t, item, "optional")
			info, err := os.Stat(filepath.Join(f.WorkDir, "search.json"))
			require.NoError(t, err)
			require.EqualValues(t, 0600, info.Mode().Perm())
			noReceipts(t, f)
		})
	}
}

func TestDetailArtifactsFilesAndFailures(t *testing.T) {
	const transcript = "  发言人 001\nUnicode 中文 🎉\n\t最后一行  \n"
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "GET", r.Method)
		switch r.URL.Path {
		case "/open-apis/minutes/v1/minutes/tok001":
			reply(w, map[string]any{"minute": map[string]any{"title": "会议", "note_id": "note001"}})
		case "/open-apis/minutes/v1/minutes/tok001/artifacts":
			reply(w, map[string]any{"summary": "  总结\n", "transcript": transcript, "minute_todos": []any{map[string]any{"todo_id": "001", "content": "  todo  ", "is_done": false}}, "minute_chapters": []any{map[string]any{"start_time": "0001", "title": "章节"}}, "keywords": []any{"词语"}})
		case "/open-apis/minutes/v1/minutes/tok002":
			_, _ = io.WriteString(w, `{"code":2091005,"msg":"permission denied"}`)
		case "/open-apis/minutes/v1/minutes/tok003":
			_, _ = io.WriteString(w, `{"code":2091003,"msg":"processing"}`)
		default:
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	args := []string{"minutes", "+detail", "--as", "bot", "--minute-tokens", "tok001", "--summary", "--todo", "--chapter", "--keyword", "--transcript", "--output-dir", "artifacts"}
	result := success(t, f.Run(args...))
	require.EqualValues(t, 2, calls.Load())
	item := result["data"].(map[string]any)["minutes"].([]any)[0].(map[string]any)
	artifacts := item["artifacts"].(map[string]any)
	require.Equal(t, "  总结\n", artifacts["summary"])
	require.Len(t, artifacts["todos"], 1)
	require.Len(t, artifacts["chapters"], 1)
	require.Len(t, artifacts["keywords"], 1)
	path := filepath.Join(f.WorkDir, artifacts["transcript_file"].(string))
	bytes, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, transcript, string(bytes))
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0600))
	success(t, f.Run(args...))
	bytes, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "existing", string(bytes))
	success(t, f.Run(append(args, "--overwrite")...))
	bytes, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, transcript, string(bytes))
	mixed := f.Run("minutes", "+detail", "--as", "bot", "--minute-tokens", "tok001,tok002", "--result-fields", "title")
	require.Equal(t, 1, mixed.ExitCode)
	v := decoded(t, mixed.Stdout)
	require.Equal(t, false, v["ok"])
	require.Contains(t, mixed.Stdout, "No read permission")
	require.Len(t, v["data"].(map[string]any)["minutes"], 2)
	before := calls.Load()
	pending := f.Run("minutes", "+detail", "--as", "bot", "--minute-tokens", "tok003", "--wait-ready", "--wait-timeout-seconds", "1", "--wait-interval-seconds", "2")
	require.Equal(t, 1, pending.ExitCode)
	require.Contains(t, pending.Stdout, `"status":"processing"`)
	require.Contains(t, pending.Stdout, "next_command")
	require.Equal(t, before+1, calls.Load())
	require.NoError(t, os.WriteFile(filepath.Join(f.WorkDir, "not-directory"), []byte("keep"), 0600))
	failed := f.Run("minutes", "+detail", "--as", "bot", "--minute-tokens", "tok001", "--transcript", "--output-dir", "not-directory")
	require.Equal(t, 1, failed.ExitCode)
	require.Contains(t, failed.Stdout, "failed to save transcript")
	require.Contains(t, failed.Stdout, `"ok":false`)
	noReceipts(t, f)
}

func TestDownloadMediaAndBatch(t *testing.T) {
	const media = "\x00synthetic audio/video 中文\xff"
	var downloads atomic.Int32
	// This server is an HTTP proxy sink. It never forwards a request off host.
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		require.Equal(t, "203.0.113.10", r.URL.Host)
		require.Equal(t, "/media", r.URL.Path)
		w.Header().Set("Content-Type", "video/mp4")
		w.Header().Set("Content-Disposition", `attachment; filename="clip.mp4"`)
		_, _ = io.WriteString(w, media)
	}))
	defer proxy.Close()
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		if strings.HasSuffix(r.URL.Path, "tok002/media") {
			_, _ = io.WriteString(w, `{"code":2091005,"msg":"permission denied"}`)
			return
		}
		require.Equal(t, "/open-apis/minutes/v1/minutes/tok001/media", r.URL.Path)
		reply(w, map[string]any{"download_url": "http://203.0.113.10/media"})
	}))
	f.Env["HTTP_PROXY"] = proxy.URL
	f.Env["http_proxy"] = proxy.URL
	success(t, f.Run("minutes", "+download", "--as", "bot", "--minute-tokens", "tok001", "--url-only"))
	require.Zero(t, downloads.Load())
	result := success(t, f.Run("minutes", "+download", "--as", "user", "--minute-tokens", "tok001", "--output", "clip.mp4"))
	require.EqualValues(t, len(media), result["data"].(map[string]any)["size_bytes"])
	saved, err := os.ReadFile(filepath.Join(f.WorkDir, "clip.mp4"))
	require.NoError(t, err)
	require.Equal(t, media, string(saved))
	collision := f.Run("minutes", "+download", "--as", "bot", "--minute-tokens", "tok001", "--output", "clip.mp4")
	require.Equal(t, 2, collision.ExitCode)
	require.Contains(t, collision.Stderr, "failed_precondition")
	success(t, f.Run("minutes", "+download", "--as", "bot", "--minute-tokens", "tok001", "--output", "clip.mp4", "--overwrite"))
	mixed := f.Run("minutes", "+download", "--as", "bot", "--minute-tokens", "tok001,tok002", "--url-only")
	require.Equal(t, 1, mixed.ExitCode)
	require.Contains(t, mixed.Stdout, `"ok":false`)
	require.Contains(t, mixed.Stdout, "tok001")
	require.Contains(t, mixed.Stdout, "tok002")
	noReceipts(t, f)
}

func TestMutationBodiesAndNoop(t *testing.T) {
	const content = "  中文\n\tvalue  "
	for _, tc := range []struct {
		name, method, suffix string
		args                 []string
		expected             map[string]any
		current              map[string]any
		readSuffix           string
	}{
		{"update", "PATCH", "", []string{"--topic", content}, map[string]any{"topic": content}, map[string]any{"minute": map[string]any{"title": content}}, ""},
		{"summary", "PUT", "/summary", []string{"--summary", "@content.txt"}, map[string]any{"summary": content}, map[string]any{"summary": content}, "/artifacts"},
		{"todo", "POST", "/todo", []string{"--operation", "update", "--todo-id", "001", "--todo", "@content.txt", "--is-done=false"}, map[string]any{"todo_items": []any{map[string]any{"operation": "update", "todo_id": "001", "content": content, "is_done": false}}}, map[string]any{"minute_todos": []any{map[string]any{"todo_id": "001", "content": content, "is_done": false}}}, "/artifacts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reads, writes atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == "GET" {
					reads.Add(1)
					require.Equal(t, "/open-apis/minutes/v1/minutes/tok001"+tc.readSuffix, r.URL.Path)
					reply(w, tc.current)
					return
				}
				writes.Add(1)
				require.Equal(t, tc.method, r.Method)
				require.Equal(t, "/open-apis/minutes/v1/minutes/tok001"+tc.suffix, r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, tc.expected, body)
				reply(w, map[string]any{})
			}))
			require.NoError(t, os.WriteFile(filepath.Join(f.WorkDir, "content.txt"), []byte(content), 0600))
			args := append([]string{"minutes", "+" + tc.name, "--as", "user", "--minute-token", "tok001"}, tc.args...)
			preview := success(t, f.Run(append(args, "--skip-unchanged", "--dry-run")...))
			require.Contains(t, fmt.Sprint(preview), "GET")
			require.EqualValues(t, 0, reads.Load())
			require.EqualValues(t, 0, writes.Load())
			same := success(t, f.Run(append(args, "--skip-unchanged")...))
			require.Equal(t, true, same["data"].(map[string]any)["unchanged"])
			require.EqualValues(t, 1, reads.Load())
			require.Zero(t, writes.Load())
			noReceipts(t, f)
			success(t, f.Run(args...))
			require.EqualValues(t, 1, writes.Load())
			require.EqualValues(t, 1, reads.Load())
			files, err := filepath.Glob(filepath.Join(f.ConfigDir, "next", "receipts", "*.json"))
			require.NoError(t, err)
			require.Len(t, files, 1)
		})
	}
}

func TestTodoBatchSpeakerUploadAndPermission(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		args               []string
		body               map[string]any
		data               map[string]any
	}{
		{"todo", "POST", "/tok001/todo", []string{"+todo", "--minute-token", "tok001", "--todos", `[{"operation":"add","content":" 新 ","is_done":false},{"operation":"update","todo_id":"001","content":" 完成 ","is_done":true},{"operation":"delete","todo_id":"002"}]`}, map[string]any{"todo_items": []any{map[string]any{"operation": "add", "content": " 新 ", "is_done": false}, map[string]any{"operation": "update", "todo_id": "001", "content": " 完成 ", "is_done": true}, map[string]any{"operation": "delete", "todo_id": "002"}}}, map[string]any{}},
		{"speaker", "PUT", "/tok001/transcript/speaker", []string{"+speaker-replace", "--minute-token", "tok001", "--from-speaker-id", "speaker001", "--to-user-id", "ou_target"}, map[string]any{"from_speaker_id": "speaker001", "to_user_id": "ou_target"}, map[string]any{}},
		{"legacy-speaker", "PUT", "/tok001/transcript/speaker", []string{"+speaker-replace", "--minute-token", "tok001", "--from-user-id", "ou_source", "--to-user-id", "ou_target"}, map[string]any{"from_user_id": "ou_source", "to_user_id": "ou_target"}, map[string]any{}},
		{"upload", "POST", "/upload", []string{"+upload", "--file-token", "file001"}, map[string]any{"file_token": "file001"}, map[string]any{"minute_url": "https://minutes.example.test/minutes/tok001"}},
		{"permission-view", "POST", "/tok001/permissions/apply", []string{"+apply-permission", "--minute-token", "tok001", "--perm", "view"}, map[string]any{"perm": "view"}, map[string]any{}},
		{"permission-edit", "POST", "/tok001/permissions/apply", []string{"+apply-permission", "--minute-token", "tok001", "--perm", "edit"}, map[string]any{"perm": "edit"}, map[string]any{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, tc.method, r.Method)
				require.Equal(t, "/open-apis/minutes/v1/minutes"+tc.path, r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, tc.body, body)
				if strings.Contains(tc.name, "speaker") {
					require.Equal(t, "open_id", r.URL.Query().Get("user_id_type"))
				}
				reply(w, tc.data)
			}))
			args := append([]string{"minutes"}, tc.args...)
			args = append(args, "--as", "user")
			result := success(t, f.Run(args...))
			require.EqualValues(t, 1, calls.Load())
			if tc.name == "upload" {
				require.Equal(t, "tok001", result["data"].(map[string]any)["minute_token"])
			}
		})
	}
}

func TestWordReplaceOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		exit           int
		contains       string
	}{
		{"applied", `{"replace_word_counts":[{"source_word":" 词 ","replace_count":"2"},{"source_word":"x","replace_count":1}]}`, 0, "Succeeded"},
		{"mixed", `{"replace_word_counts":[{"source_word":" 词 ","replace_count":"2"},{"source_word":"x","replace_count":"0"}]}`, 22, "Do not reprocess"},
		{"missing", `{}`, 22, "replace_word_counts"},
		{"incomplete", `{"replace_word_counts":[{"source_word":" 词 ","replace_count":"2"}]}`, 22, "omitted outcomes"},
		{"zero", `{"replace_word_counts":[{"source_word":" 词 ","replace_count":"0"},{"source_word":"x","replace_count":"0"}]}`, 22, "nothing was replaced"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "PUT", r.Method)
				require.Equal(t, "/open-apis/minutes/v1/minutes/tok001/transcript/word", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, " 词 ", body["replace_words"].([]any)[0].(map[string]any)["source_word"])
				require.Equal(t, "", body["replace_words"].([]any)[0].(map[string]any)["target_word"])
				_, _ = fmt.Fprintf(w, `{"code":0,"data":%s}`, tc.response)
			}))
			result := f.Run("minutes", "+word-replace", "--as", "user", "--minute-token", "tok001", "--replace-words", `[{"source_word":" 词 ","target_word":""},{"source_word":"x","target_word":" y "}]`)
			require.Equal(t, tc.exit, result.ExitCode, "stdout=%s stderr=%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout+result.Stderr, tc.contains)
			require.EqualValues(t, 1, calls.Load())
			if tc.exit != 0 {
				require.Contains(t, result.Stderr, `"outcome":"unknown"`)
			}
		})
	}
}

func TestValidationPermissionsAndUnknownMutation(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	for _, args := range [][]string{
		{"+detail", "--minute-tokens", ", ,"}, {"+detail", "--minute-tokens", strings.Repeat("tok001,", 51)},
		{"+search", "--query", "x", "--page-size", "31"}, {"+search", "--owner-ids", "ambiguous name"},
		{"+speaker-replace", "--minute-token", "tok001", "--from-speaker-id", "speaker001", "--to-user-id", "display name"},
		{"+todo", "--minute-token", "tok001", "--operation", "add", "--todo", "x", "--is-done=false", "--skip-unchanged"},
		{"+word-replace", "--minute-token", "tok001", "--replace-words", `[{"source_word":"x"}]`},
	} {
		r := f.Run(append([]string{"minutes", "--as", "user"}, args...)...)
		require.Equal(t, 2, r.ExitCode, "%v: %s %s", args, r.Stdout, r.Stderr)
		require.Empty(t, r.Stdout)
		require.Equal(t, "validation", decoded(t, r.Stderr)["error"].(map[string]any)["type"])
	}
	noReceipts(t, f)
	for _, mode := range []string{"scope", "disconnect"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if mode == "scope" {
					w.WriteHeader(403)
					_, _ = io.WriteString(w, `{"code":99991679,"msg":"missing scope"}`)
					return
				}
				connection, _, err := w.(http.Hijacker).Hijack()
				require.NoError(t, err)
				_ = connection.Close()
			}))
			r := f.Run("minutes", "+summary", "--as", "user", "--minute-token", "tok001", "--summary", "synthetic")
			expected := 22
			if mode == "scope" {
				expected = 20
			}
			require.Equal(t, expected, r.ExitCode, r.Stderr)
			require.EqualValues(t, 1, calls.Load())
			require.Contains(t, r.Stderr, "mutation_outcome")
		})
	}
}

func TestGenericMetadata(t *testing.T) {
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/open-apis/minutes/v1/minutes/tok001", r.URL.Path)
		require.Equal(t, "union_id", r.URL.Query().Get("user_id_type"))
		reply(w, map[string]any{"minute": map[string]any{"token": "tok001", "title": "会议", "owner_id": "on_owner", "create_time": "0001", "duration": "9007199254740993", "url": "https://minutes.example.test/minutes/tok001"}})
	}))
	result := success(t, f.Run("minutes", "minutes", "get", "--minute-token", "tok001", "--user-id-type", "union_id", "--as", "bot"))
	require.Contains(t, fmt.Sprint(result), "9007199254740993")
	require.Contains(t, fmt.Sprint(result), "0001")
	noReceipts(t, f)
}

func TestLiveMinutesPrivateLifecycle(t *testing.T) {
	t.Skip("blocked: no verified run-owned private Drive media plus Minutes creation visibility/current-account tenant fixture; permission requests and speaker rebinding may notify unrelated users. No live requests are dispatched. See docs/sections/minutes.md")
}

func TestSkipUnchangedFailsClosedOnUnreadableContent(t *testing.T) {
	for _, body := range []string{`{"code":99991679,"msg":"missing scope"}`, `{"code":0,"data":{}}`} {
		t.Run(body, func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, "GET", r.Method)
				_, _ = io.WriteString(w, body)
			}))
			r := f.Run("minutes", "+summary", "--as", "user", "--minute-token", "tok001", "--summary", "new", "--skip-unchanged")
			require.NotZero(t, r.ExitCode)
			require.Empty(t, r.Stdout)
			require.EqualValues(t, 1, calls.Load())
			noReceipts(t, f)
		})
	}
}

func TestReplacementReadback(t *testing.T) {
	const target = "  新的 标题\n"
	var writes atomic.Int32
	current := atomic.Value{}
	current.Store("old")
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/open-apis/minutes/v1/minutes/tok001", r.URL.Path)
		switch r.Method {
		case "GET":
			reply(w, map[string]any{"minute": map[string]any{"title": current.Load().(string)}})
		case "PATCH":
			var body struct {
				Topic string `json:"topic"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			current.Store(body.Topic)
			writes.Add(1)
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s", r.Method)
		}
	}))
	args := []string{"minutes", "+update", "--as", "user", "--minute-token", "tok001", "--topic", target, "--skip-unchanged"}
	success(t, f.Run(args...))
	require.EqualValues(t, 1, writes.Load())
	read := success(t, f.Run("minutes", "minutes", "get", "--as", "user", "--minute-token", "tok001"))
	require.Equal(t, target, read["data"].(map[string]any)["minute"].(map[string]any)["title"])
	same := success(t, f.Run(args...))
	require.Equal(t, true, same["data"].(map[string]any)["unchanged"])
	require.EqualValues(t, 1, writes.Load())
}
