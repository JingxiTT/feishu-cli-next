// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package markdown

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownCreateDryRun_Content(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+create",
			"--name", "README.md",
			"--content", "# hello",
			"--folder-token", "fldcnMarkdownDryRun",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_all")
	assert.Contains(t, output, "/open-apis/drive/v1/metas/batch_query")
	assert.True(t, clie2e.DryRunGet(output, "api.1.body.with_url").Bool())
	assert.Equal(t, "README.md", clie2e.DryRunGet(output, "api.0.body.file_name").String())
	assert.Equal(t, "fldcnMarkdownDryRun", clie2e.DryRunGet(output, "api.0.body.parent_node").String())
	assert.Equal(t, "explorer", clie2e.DryRunGet(output, "api.0.body.parent_type").String())
	assert.Equal(t, int64(7), clie2e.DryRunGet(output, "api.0.body.size").Int())
}

func TestMarkdownCreateDryRun_WikiTarget(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+create",
			"--name", "README.md",
			"--content", "# hello",
			"--wiki-token", "wikcnMarkdownDryRun",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_all")
	assert.Contains(t, output, "/open-apis/drive/v1/metas/batch_query")
	assert.True(t, clie2e.DryRunGet(output, "api.1.body.with_url").Bool())
	assert.Equal(t, "README.md", clie2e.DryRunGet(output, "api.0.body.file_name").String())
	assert.Equal(t, "wikcnMarkdownDryRun", clie2e.DryRunGet(output, "api.0.body.parent_node").String())
	assert.Equal(t, "wiki", clie2e.DryRunGet(output, "api.0.body.parent_type").String())
	assert.Equal(t, int64(7), clie2e.DryRunGet(output, "api.0.body.size").Int())
}

func TestMarkdownCreateDryRun_FileShowsConcreteSize(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	dir := t.TempDir()
	content := "# hi\n"
	require.NoError(t, os.WriteFile(dir+"/note.md", []byte(content), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+create",
			"--file", "note.md",
			"--dry-run",
		},
		DefaultAs: "bot",
		WorkDir:   dir,
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_all")
	assert.Contains(t, output, "/open-apis/drive/v1/metas/batch_query")
	assert.True(t, clie2e.DryRunGet(output, "api.1.body.with_url").Bool())
	assert.Equal(t, "@note.md", clie2e.DryRunGet(output, "api.0.body.file").String())
	assert.Equal(t, int64(5), clie2e.DryRunGet(output, "api.0.body.size").Int())
}

func TestMarkdownCreateDryRun_RejectsEmptyContent(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+create",
			"--name", "empty.md",
			"--content", "",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	if api := clie2e.DryRunGet(result.Stdout, "api"); api.IsArray() && len(api.Array()) > 0 {
		t.Fatalf("dry-run api list must be empty when validation fails\nstdout:\n%s", result.Stdout)
	}
	errMsg := clie2e.DryRunGet(result.Stdout, "error").String()
	assert.Contains(t, errMsg, "empty markdown content is not supported")
}

func TestMarkdownDiffDryRun_RemoteVsRemote(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+diff",
			"--file-token", "boxcnMarkdownDryRun",
			"--from-version", "7633658129540910621",
			"--to-version", "7633658129540910628",
			"--context-lines", "1",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/medias/boxcnMarkdownDryRun/preview_download")
	require.Equal(t, "remote_vs_remote", clie2e.DryRunGet(output, "mode").String(), output)
	require.Equal(t, int64(2), clie2e.DryRunGet(output, "api.#").Int(), output)
	require.Equal(t, "16", clie2e.DryRunGet(output, "api.0.params.preview_type").String(), output)
	require.Equal(t, "7633658129540910621", clie2e.DryRunGet(output, "api.0.params.version").String(), output)
	require.Equal(t, "16", clie2e.DryRunGet(output, "api.1.params.preview_type").String(), output)
	require.Equal(t, "7633658129540910628", clie2e.DryRunGet(output, "api.1.params.version").String(), output)
	require.Equal(t, int64(1), clie2e.DryRunGet(output, "context_lines").Int(), output)
}

func TestMarkdownDiffDryRun_RemoteVsLocal(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(dir+"/draft.md", []byte("# draft\n"), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+diff",
			"--file-token", "boxcnMarkdownDryRun",
			"--file", "./draft.md",
			"--dry-run",
		},
		DefaultAs: "bot",
		WorkDir:   dir,
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/medias/boxcnMarkdownDryRun/preview_download")
	assert.Equal(t, "remote_vs_local", clie2e.DryRunGet(output, "mode").String())
	assert.Equal(t, "16", clie2e.DryRunGet(output, "api.0.params.preview_type").String())
	assert.Equal(t, "./draft.md", clie2e.DryRunGet(output, "local_file").String())
}

func TestMarkdownCreateDryRun_RejectsEmptyWikiToken(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+create",
			"--name", "README.md",
			"--content", "# hello",
			"--wiki-token", "",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 2)
	assert.Contains(t, result.Stdout+result.Stderr, "--wiki-token cannot be empty")
}

func TestMarkdownFetchDryRun_OutputFile(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+fetch",
			"--file-token", "boxcnMarkdownDryRun",
			"--output", "./copy.md",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/medias/boxcnMarkdownDryRun/preview_download")
	assert.Equal(t, "16", clie2e.DryRunGet(output, "api.0.params.preview_type").String())
	assert.Equal(t, "./copy.md", clie2e.DryRunGet(output, "output").String())
}

func TestMarkdownOverwriteDryRun_ContentFile(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)
	dir := t.TempDir()
	content := "# overwrite test\n"
	require.NoError(t, os.WriteFile(dir+"/input.md", []byte(content), 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+overwrite",
			"--file-token", "boxcnMarkdownDryRun",
			"--content", "@input.md",
			"--dry-run",
		},
		DefaultAs: "bot",
		WorkDir:   dir,
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/metas/batch_query")
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_all")
	assert.Equal(t, "boxcnMarkdownDryRun", clie2e.DryRunGet(output, "api.1.body.file_token").String())
	assert.Equal(t, int64(17), clie2e.DryRunGet(output, "api.1.body.size").Int())
}

func TestMarkdownOverwriteDryRun_RejectsEmptyFile(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(dir+"/empty.md", []byte{}, 0o644))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+overwrite",
			"--file-token", "boxcnMarkdownDryRun",
			"--file", "empty.md",
			"--dry-run",
		},
		DefaultAs: "bot",
		WorkDir:   dir,
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	if api := clie2e.DryRunGet(result.Stdout, "api"); api.IsArray() && len(api.Array()) > 0 {
		t.Fatalf("dry-run api list must be empty when validation fails\nstdout:\n%s", result.Stdout)
	}
	errMsg := clie2e.DryRunGet(result.Stdout, "error").String()
	assert.Contains(t, errMsg, "empty markdown content is not supported")
}

func TestMarkdownPatchDryRun_Content(t *testing.T) {
	setMarkdownDryRunConfigEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	result, err := clie2e.RunCmd(ctx, clie2e.Request{
		Args: []string{
			"markdown", "+patch",
			"--file-token", "boxcnMarkdownDryRun",
			"--pattern", "TODO",
			"--content", "DONE",
			"--dry-run",
		},
		DefaultAs: "bot",
	})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)

	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/drive/v1/medias/boxcnMarkdownDryRun/preview_download")
	assert.Equal(t, "16", clie2e.DryRunGet(output, "api.0.params.preview_type").String())
	assert.Contains(t, output, "/open-apis/drive/v1/metas/batch_query")
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_all")
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_prepare")
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_part")
	assert.Contains(t, output, "/open-apis/drive/v1/files/upload_finish")
}

func setMarkdownDryRunConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv("LARKSUITE_CLI_APP_ID", "markdown_dryrun_test")
	t.Setenv("LARKSUITE_CLI_APP_SECRET", "markdown_dryrun_secret")
	t.Setenv("LARKSUITE_CLI_BRAND", "feishu")
}
