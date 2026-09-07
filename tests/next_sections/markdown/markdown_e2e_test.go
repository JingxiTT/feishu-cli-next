// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package markdowne2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }

const token = "box_md_next_fixture"
const richSource = "# 中文 😀\r\n\r\n**bold** _italic_ ~~strike~~ `a<b && c`\n\\*literal\\* | &amp;\n\n| 列 | value |\n| --- | --- |\n| 001 | null |\n\n- [x] done\n> quote\n\n![图](asset.png) [link](https://example.com/a?q=%E4%B8%AD)\n```go\nx := \"$1\\n\"\n```\n尾行"

func reply(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": data})
}
func success(t *testing.T, r nextfixture.Result) {
	t.Helper()
	require.Equal(t, 0, r.ExitCode, "%+v", r)
}
func data(r nextfixture.Result) gjson.Result {
	v := gjson.Parse(r.Stdout)
	if v.Get("data").Exists() {
		return v.Get("data")
	}
	return v
}

// Source files are opaque bytes; this server stores each uploaded revision and
// exposes the same preview/version and metadata contracts as the reference.
type store struct {
	mu       sync.Mutex
	name     string
	versions [][]byte
	uploads  int
	targets  []string
}

func (s *store) serve(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		switch {
		case r.Method == "POST" && r.URL.Path == "/open-apis/drive/v1/files/upload_all":
			require.NoError(t, r.ParseMultipartForm(32<<20))
			defer r.MultipartForm.RemoveAll()
			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			content, err := io.ReadAll(file)
			require.NoError(t, err)
			require.Equal(t, strconv.Itoa(len(content)), r.FormValue("size"))
			if len(s.versions) > 0 {
				require.Equal(t, token, r.FormValue("file_token"))
			}
			s.name = r.FormValue("file_name")
			s.versions = append(s.versions, content)
			s.uploads++
			s.targets = append(s.targets, r.FormValue("parent_type")+":"+r.FormValue("parent_node"))
			reply(w, map[string]any{"file_token": token, "version": strconv.Itoa(len(s.versions))})
		case r.Method == "POST" && r.URL.Path == "/open-apis/drive/v1/metas/batch_query":
			var body struct {
				RequestDocs []struct {
					Token string `json:"doc_token"`
					Type  string `json:"doc_type"`
				} `json:"request_docs"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body.RequestDocs, 1)
			require.Equal(t, token, body.RequestDocs[0].Token)
			require.Equal(t, "file", body.RequestDocs[0].Type)
			reply(w, map[string]any{"metas": []any{map[string]any{"doc_token": token, "doc_type": "file", "title": s.name, "url": "https://fixture.example/file/" + token}}})
		case r.Method == "GET" && r.URL.Path == "/open-apis/drive/v1/medias/"+token+"/preview_download":
			require.Equal(t, "16", r.URL.Query().Get("preview_type"))
			version := len(s.versions)
			if v := r.URL.Query().Get("version"); v != "" {
				var err error
				version, err = strconv.Atoi(v)
				require.NoError(t, err)
			}
			require.Greater(t, version, 0)
			require.LessOrEqual(t, version, len(s.versions))
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="`+s.name+`"`)
			_, _ = w.Write(s.versions[version-1])
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	})
}

func TestMarkdownCompleteSourceLifecycle(t *testing.T) {
	s := &store{}
	f := nextfixture.NewFixture(t, s.serve(t))
	f.Env["LARKSUITE_CLI_DEFAULT_AS"] = "user"
	input := filepath.Join(f.WorkDir, "source.md")
	require.NoError(t, os.WriteFile(input, []byte(richSource), 0600))
	created := f.Run("markdown", "+create", "--name", "source.md", "--content", "@"+input)
	success(t, created)
	require.Equal(t, token, data(created).Get("file_token").String())
	require.Equal(t, int64(len([]byte(richSource))), data(created).Get("size_bytes").Int())
	require.Equal(t, []string{"explorer:"}, s.targets)
	// Metadata POST reads must not generate mutation receipts.
	receipts := f.Run("receipts", "list")
	success(t, receipts)
	require.Equal(t, int64(1), gjson.Get(receipts.Stdout, "data.#").Int(), receipts.Stdout)
	fetched := f.Run("markdown", "+fetch", "--file-token", token)
	success(t, fetched)
	require.Equal(t, richSource, data(fetched).Get("content").String())
	pretty := f.Run("markdown", "+fetch", "--file-token", token, "--format", "pretty")
	success(t, pretty)
	require.Equal(t, richSource, pretty.Stdout)
	saved := filepath.Join(f.WorkDir, "saved.md")
	success(t, f.Run("markdown", "+fetch", "--file-token", token, "--output", saved))
	actual, err := os.ReadFile(saved)
	require.NoError(t, err)
	require.Equal(t, []byte(richSource), actual)
	exists := f.Run("markdown", "+fetch", "--file-token", token, "--output", saved)
	require.Equal(t, 2, exists.ExitCode)
	require.Contains(t, exists.Stderr, "--output")
	success(t, f.Run("markdown", "+fetch", "--file-token", token, "--output", saved, "--overwrite"))
	unchanged := f.Run("markdown", "+patch", "--file-token", token, "--pattern", "(中文 😀)", "--content", "$1", "--regex")
	success(t, unchanged)
	require.False(t, data(unchanged).Get("updated").Bool())
	require.Equal(t, int64(1), data(unchanged).Get("match_count").Int())
	require.Equal(t, 1, s.uploads)
	patched := f.Run("markdown", "+patch", "--file-token", token, "--pattern", "尾行", "--content", "结束 $1")
	success(t, patched)
	require.True(t, data(patched).Get("updated").Bool())
	require.Equal(t, strings.ReplaceAll(richSource, "尾行", "结束 $1"), string(s.versions[1]))
	diff := f.Run("markdown", "+diff", "--file-token", token, "--from-version", "01", "--to-version", "2", "--context-lines", "0")
	success(t, diff)
	require.True(t, data(diff).Get("changed").Bool())
	require.Equal(t, "01", data(diff).Get("from_version").String())
	require.Contains(t, data(diff).Get("diff").String(), "No newline at end of file")
	localDiff := f.Run("markdown", "+diff", "--file-token", token, "--from-version", "1", "--file", input, "--format", "pretty")
	success(t, localDiff)
	require.Equal(t, "No differences.\n", localDiff.Stdout)
	// Explicit overwrite retains its upload/version semantics even for equal bytes.
	overwritten := f.Run("markdown", "+overwrite", "--file-token", token, "--file", input, "--name", "renamed.md")
	success(t, overwritten)
	require.Equal(t, "3", data(overwritten).Get("version").String())
	require.Equal(t, "renamed.md", s.name)
	require.Equal(t, []byte(richSource), s.versions[2])
	success(t, f.Run("markdown", "+overwrite", "--file-token", token, "--content", richSource))
	require.Equal(t, 4, s.uploads)
	require.Equal(t, "renamed.md", s.name)
}

func TestMarkdownCreateTargets(t *testing.T) {
	for _, tc := range []struct{ name, flag, value, target string }{
		{"folder", "--folder-token", "https://fixture.example/drive/folder/fldSynthetic", "explorer:fldSynthetic"},
		{"wiki", "--wiki-token", "https://fixture.example/wiki/wikSynthetic", "wiki:wikSynthetic"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &store{}
			f := nextfixture.NewFixture(t, s.serve(t))
			f.Env["LARKSUITE_CLI_DEFAULT_AS"] = "user"
			path := filepath.Join(f.WorkDir, "输入.md")
			require.NoError(t, os.WriteFile(path, []byte(richSource), 0600))
			r := f.Run("markdown", "+create", tc.flag, tc.value, "--file", path)
			success(t, r)
			require.Equal(t, tc.target, s.targets[0])
			require.Equal(t, []byte(richSource), s.versions[0])
		})
	}
}

func TestMarkdownMultipartPreservesAllBytes(t *testing.T) {
	payload := bytes.Repeat([]byte("中文 😀\\*\r\n"), 2*1024*1024)
	var uploaded bytes.Buffer
	parts := 0
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/files/upload_prepare":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, float64(len(payload)), body["size"])
			require.Equal(t, "wiki", body["parent_type"])
			require.Equal(t, "wikSynthetic", body["parent_node"])
			reply(w, map[string]any{"upload_id": "uploadSynthetic", "block_size": 8 * 1024 * 1024, "block_num": (len(payload) + 8*1024*1024 - 1) / (8 * 1024 * 1024)})
		case "/open-apis/drive/v1/files/upload_part":
			require.NoError(t, r.ParseMultipartForm(10<<20))
			defer r.MultipartForm.RemoveAll()
			require.Equal(t, "uploadSynthetic", r.FormValue("upload_id"))
			require.Equal(t, strconv.Itoa(parts), r.FormValue("seq"))
			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			n, err := io.Copy(&uploaded, file)
			require.NoError(t, err)
			require.Equal(t, strconv.FormatInt(n, 10), r.FormValue("size"))
			parts++
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/files/upload_finish":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, float64(parts), body["block_num"])
			reply(w, map[string]any{"file_token": token, "version": "9007199254740993"})
		case "/open-apis/drive/v1/metas/batch_query":
			reply(w, map[string]any{"metas": []any{}})
		default:
			t.Errorf("unexpected %s", r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	f.Env["LARKSUITE_CLI_DEFAULT_AS"] = "user"
	path := filepath.Join(f.WorkDir, "large.md")
	require.NoError(t, os.WriteFile(path, payload, 0600))
	r := f.Run("markdown", "+create", "--file", path, "--wiki-token", "wikSynthetic")
	success(t, r)
	require.Equal(t, payload, uploaded.Bytes())
	require.Greater(t, parts, 1)
}

func TestMarkdownUnknownWriteIsNotReplayed(t *testing.T) {
	for _, status := range []int{0, http.StatusBadGateway, http.StatusServiceUnavailable} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			var calls atomic.Int32
			f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/open-apis/drive/v1/files/upload_all", r.URL.Path)
				calls.Add(1)
				if status == 0 {
					_, _ = io.Copy(io.Discard, r.Body)
					connection, _, err := w.(http.Hijacker).Hijack()
					require.NoError(t, err)
					_ = connection.Close()
					return
				}
				http.Error(w, "lost upstream reply", status)
			}))
			f.Env["LARKSUITE_CLI_DEFAULT_AS"] = "user"
			r := f.Run("markdown", "+create", "--name", "unknown.md", "--content", "# test")
			require.Equal(t, 22, r.ExitCode, "%+v", r)
			require.Equal(t, int32(1), calls.Load())
			require.Contains(t, r.Stderr, `"outcome":"unknown"`)
		})
	}
}

func TestMarkdownValidationAndDiscovery(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	commands := map[string]bool{}
	for offset := 0; ; {
		discovery := f.Run("discover", "markdown", "--offset", strconv.Itoa(offset))
		success(t, discovery)
		for _, entry := range gjson.Get(discovery.Stdout, "data").Array() {
			commands[entry.Get("command").String()] = true
		}
		if gjson.Get(discovery.Stdout, "meta.complete").Bool() {
			break
		}
		next := gjson.Get(discovery.Stdout, "meta.next_offset")
		require.True(t, next.Exists(), discovery.Stdout)
		require.Greater(t, int(next.Int()), offset, discovery.Stdout)
		offset = int(next.Int())
	}
	for _, command := range []string{"+create", "+fetch", "+diff", "+patch", "+overwrite"} {
		require.Contains(t, commands, "markdown "+command)
		success(t, f.Run("markdown", command, "--help"))
	}
	for i, args := range [][]string{
		{"+create", "--name", "a.md", "--content", ""},
		{"+create", "--name", "a.md", "--content", "x", "--folder-token", "fldX", "--wiki-token", "wikX"},
		{"+patch", "--file-token", token, "--regex", "--pattern", "[", "--content", "x"},
		{"+diff", "--file-token", token, "--from-version", "x"},
		{"+diff", "--file-token", token, "--from-version", "1", "--context-lines", "-1"},
	} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			r := f.Run(append([]string{"markdown"}, args...)...)
			require.Equal(t, 2, r.ExitCode, "%+v", r)
			require.Contains(t, r.Stderr, "invalid_argument")
		})
	}
}
