// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

// Every shortcut executes validation and request planning in a real CLI process.
// The handler rejects business requests: dry-run must never dispatch a mutation.
func TestEveryDriveShortcutDryRun(t *testing.T) {
	f := nextfixture.NewFixture(t, nil)
	if err := os.WriteFile(filepath.Join(f.WorkDir, "source.md"), []byte("# 测试\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rows := []struct{ command, flags, want string }{
		{"+upload", "--file source.md --folder-token root", "upload_all"},
		{"+create-folder", "--name 测试 --folder-token root", "create_folder"},
		{"+create-shortcut", "--file-token source --type docx --folder-token root", "create_shortcut"},
		{"+copy", "--token source --type base --name 测试 --folder-token my_space --extra target_type=docx", "root_folder/meta"},
		{"+download", "--file-token source --output saved.bin", "download"},
		{"+preview", "--file-token source --list-only", "preview"},
		{"+cover", "--file-token source --spec square --output cover.png", "cover"},
		{"+add-comment", "--doc source --type docx", "comments"},
		{"+list-comments", "--token source --type docx --page-size 2 --page-token next --need-relation --need-reaction", "comments"},
		{"+batch-query-comments", "--token source --type docx --comment-ids c1,c2", "batch_query"},
		{"+resolve-comment", "--token source --type docx --comment-id c1", "is_solved"},
		{"+restore-comment", "--token source --type docx --comment-id c1", "is_solved"},
		{"+add-reply", "--token source --type docx --comment-id c1", "replies"},
		{"+list-replies", "--token source --type docx --comment-id c1 --page-size 2 --page-token next --need-reaction", "replies"},
		{"+update-reply", "--token source --type base --comment-id c1 --reply-id r1", "bitable"},
		{"+delete-reply", "--token source --type docx --comment-id c1 --reply-id r1", "DELETE"},
		{"+react-reply", "--token source --type docx --reply-id r1 --emoji THUMBSUP --action add", "reaction"},
		{"+export", "--token source --doc-type docx --file-extension pdf", "export_tasks"},
		{"+export-download", "--file-token artifact --file-name out.pdf", "export_tasks/file/artifact/download"},
		{"+import", "--file source.md --type docx --folder-token root", "import_tasks"},
		{"+version-history", "--file-token source --limit 2 --cursor 7", "version"},
		{"+version-get", "--file-token source --version 1 --output old.bin", "version"},
		{"+version-revert", "--file-token source --version 1", "version"},
		{"+version-delete", "--file-token source --version 1", "version"},
		{"+move", "--file-token source --type file --folder-token root", "move"},
		{"+update-title", "--token source --type file --title 测试.xlsx", "batch_query"},
		{"+delete", "--file-token source --type file", "async"},
		{"+status", "--local-dir . --folder-token root", "files"},
		{"+push", "--local-dir . --folder-token root --if-exists overwrite --delete-remote --yes", "files"},
		{"+pull", "--local-dir . --folder-token root --on-duplicate-remote rename --delete-local --yes", "files"},
		{"+sync", "--local-dir . --folder-token root --on-conflict keep-both", "files"},
		{"+task_result", "--scenario task_check --task-id task1", "task_check"},
		{"+apply-permission", "--token source --type docx --perm edit --remark test", "permission"},
		{"+member-add", "--token source --type docx --member-id synthetic --member-type openid --perm view", "members"},
		{"+member-list", "--token source --type docx", "members"},
		{"+member-remove", "--token source --type docx --member-id synthetic --member-type openid", "members"},
		{"+permission-get-setting", "--token source --type folder", "public"},
		{"+secure-label-list", "--page-size 2 --page-token next --lang en", "label"},
		{"+secure-label-update", "--token source --type docx --label-id 7", "label"},
		{"+search", "--query 测试 --doc-types docx,sheet --page-size 2 --page-token next --only-title", "doc_wiki/search"},
		{"+inspect", "--url https://example.feishu.cn/wiki/wiki1", "node_by_token"},
	}
	if len(rows) != 41 {
		t.Fatal("frozen Drive shortcut mapping must cover all 41 commands")
	}
	for _, row := range rows {
		t.Run(row.command, func(t *testing.T) {
			args := append([]string{"drive", row.command}, strings.Fields(row.flags)...)
			if row.command == "+add-comment" || row.command == "+add-reply" || row.command == "+update-reply" {
				args = append(args, "--content", `[{"type":"text","text":"原文 Ω"}]`)
			}
			args = append(args, "--as", "user", "--dry-run")
			got := f.Run(args...)
			if got.ExitCode != 0 || !json.Valid([]byte(got.Stdout)) || !strings.Contains(got.Stdout, row.want) {
				t.Fatalf("dry-run %s: exit=%d stdout=%s stderr=%s", row.command, got.ExitCode, got.Stdout, got.Stderr)
			}
		})
	}
}
