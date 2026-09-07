// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestPermissionsReadGrantRemoveAndPartial(t *testing.T) {
	var grants, removed, batches atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v2/permissions/doc1/public":
			reply(w, map[string]any{"permission_public": map[string]any{"link_share_entity": "closed", "external_access": false, "unknown_setting": "preserved"}})
		case "/open-apis/drive/v1/permissions/doc1/members":
			if r.Method == "GET" {
				reply(w, map[string]any{"members": []any{map[string]any{"member_id": "synthetic1", "member_type": "openid", "perm": "view"}}})
				return
			}
			b := body(t, r)
			if b["member_id"] != "synthetic1" || b["perm"] != "view" || r.URL.Query().Get("need_notification") == "true" {
				t.Errorf("wrong grant: %#v %s", b, r.URL)
			}
			grants.Add(1)
			reply(w, map[string]any{"member": b})
		case "/open-apis/drive/v1/permissions/doc1/members/synthetic1":
			if r.Method != "DELETE" {
				t.Errorf("wrong method %s", r.Method)
			}
			removed.Add(1)
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/permissions/doc1/members/batch_create":
			batches.Add(1)
			b := body(t, r)
			if len(b["members"].([]any)) != 2 {
				t.Error("batch lost recipients")
			}
			reply(w, map[string]any{"members": []any{map[string]any{"member_id": "synthetic1", "member_type": "openid", "perm": "view"}}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	out := successful(t, f.Run("drive", "+permission-get-setting", "--token", "doc1", "--type", "docx", "--as", "user"))
	if out["data"].(map[string]any)["permission_public"].(map[string]any)["unknown_setting"] != "preserved" {
		t.Fatal("lost permission detail")
	}
	successful(t, f.Run("drive", "+member-add", "--token", "doc1", "--type", "docx", "--member-type", "openid", "--member-id", "synthetic1", "--perm", "view", "--yes", "--as", "user"))
	successful(t, f.Run("drive", "+member-list", "--token", "doc1", "--type", "docx", "--as", "user"))
	successful(t, f.Run("drive", "+member-remove", "--token", "doc1", "--type", "docx", "--member-type", "openid", "--member-id", "synthetic1", "--yes", "--as", "user"))
	partial := f.Run("drive", "+member-add", "--token", "doc1", "--type", "docx", "--member-type", "openid", "--member-id", "synthetic1,synthetic2", "--perm", "view", "--yes", "--as", "user")
	if partial.ExitCode == 0 || !strings.Contains(partial.Stdout, `"partial":true`) || !strings.Contains(partial.Stdout, `"missing_member_ids":["synthetic2"]`) {
		t.Fatalf("hidden batch failure: %+v", partial)
	}
	if grants.Load() != 1 || removed.Load() != 1 || batches.Load() != 1 {
		t.Fatal("unexpected permission replay")
	}
}

func TestCommentReplyLifecyclePreservesContent(t *testing.T) {
	var created atomic.Int32
	var content map[string]any
	var solved bool
	var deleted bool
	var reactions atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/drive/v1/files/doc1/new_comments":
			b := body(t, r)
			if b["file_type"] != "docx" || len(b["reply_elements"].([]any)) != 1 {
				t.Errorf("comment content lost: %#v", b)
			}
			created.Add(1)
			reply(w, map[string]any{"comment_id": "c1", "is_whole": true})
		case "/open-apis/drive/v1/files/doc1/comments/c1/replies":
			if r.Method == "POST" {
				content = body(t, r)["content"].(map[string]any)
				reply(w, map[string]any{"reply_id": "r1"})
				return
			}
			reply(w, map[string]any{"items": []any{map[string]any{"reply_id": "r1", "content": content}}, "has_more": true, "page_token": "next"})
		case "/open-apis/drive/v1/files/doc1/comments/c1/replies/r1":
			if r.Method == "DELETE" {
				deleted = true
			} else {
				content = body(t, r)["content"].(map[string]any)
			}
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/files/doc1/comments/c1":
			solved = body(t, r)["is_solved"].(bool)
			reply(w, map[string]any{})
		case "/open-apis/drive/v1/files/doc1/comments":
			reply(w, map[string]any{"items": []any{map[string]any{"comment_id": "c1", "is_solved": solved}}, "has_more": true, "page_token": "next"})
		case "/open-apis/drive/v1/files/doc1/comments/batch_query":
			b := body(t, r)
			if len(b["comment_ids"].([]any)) != 1 {
				t.Error("wrong comment IDs")
			}
			reply(w, map[string]any{"items": []any{map[string]any{"comment_id": "c1", "is_solved": solved}}})
		case "/open-apis/drive/v2/files/doc1/comments/reaction":
			b := body(t, r)
			if b["reply_id"] != "r1" || b["reaction_type"] != "THUMBSUP" {
				t.Errorf("wrong reaction %#v", b)
			}
			reactions.Add(1)
			reply(w, map[string]any{})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL)
			http.Error(w, "unexpected", 500)
		}
	}))
	run := func(command string, flags ...string) nextfixture.Result {
		args := append([]string{"drive", command, "--token", "doc1", "--type", "docx", "--as", "user"}, flags...)
		return f.Run(args...)
	}
	successful(t, f.Run("drive", "+add-comment", "--doc", "https://example.feishu.cn/docx/doc1", "--full-comment", "--content", `[{"type":"text","text":"原文"}]`, "--as", "user"))
	if created.Load() != 1 {
		t.Fatal("comment create was not observed")
	}
	successful(t, run("+add-reply", "--comment-id", "c1", "--content", `[{"type":"text","text":"原文 & Ω"},{"type":"link","url":"https://example.com","text":"源"}]`))
	listed := run("+list-replies", "--comment-id", "c1", "--page-size", "1", "--need-reaction")
	successful(t, listed)
	if !strings.Contains(listed.Stdout, "next") || !strings.Contains(listed.Stdout, "原文") {
		t.Fatal("reply content or continuation lost")
	}
	successful(t, run("+update-reply", "--comment-id", "c1", "--reply-id", "r1", "--content", `[{"type":"text","text":"更改 Ω"}]`))
	if content["elements"].([]any)[0].(map[string]any)["text_run"].(map[string]any)["text"] != "更改 Ω" {
		t.Fatalf("content changed: %#v", content)
	}
	successful(t, run("+resolve-comment", "--comment-id", "c1"))
	if !solved {
		t.Fatal("not solved")
	}
	successful(t, run("+list-comments", "--page-size", "1", "--page-token", "next"))
	successful(t, run("+batch-query-comments", "--comment-ids", "c1"))
	successful(t, run("+restore-comment", "--comment-id", "c1"))
	if solved {
		t.Fatal("not restored")
	}
	for _, action := range []string{"add", "delete"} {
		successful(t, run("+react-reply", "--reply-id", "r1", "--emoji", "THUMBSUP", "--action", action))
	}
	successful(t, run("+delete-reply", "--comment-id", "c1", "--reply-id", "r1", "--yes"))
	if !deleted || reactions.Load() != 2 {
		t.Fatal("missing effects")
	}
}

func TestScopeFailureRetainsEvidence(t *testing.T) {
	var calls atomic.Int32
	f := nextfixture.NewFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":99991672,"msg":"scope missing","error":{"permission_violations":[{"subject":"docs:permission.member:read"}]}}`))
	}))
	got := f.Run("drive", "+member-list", "--token", "doc1", "--type", "docx", "--as", "user")
	if got.ExitCode == 0 || got.Stdout != "" || !strings.Contains(got.Stderr, "99991672") || calls.Load() != 1 {
		t.Fatalf("wrong failure: %+v", got)
	}
}
