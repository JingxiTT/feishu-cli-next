// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/tidwall/gjson"
)

// This narrow live batch consumes independently established creation visibility.
// It never creates/deletes the supplied folder or changes permissions. Every CLI
// invocation has one attempt, including cleanup; unknown writes are not replayed.
func TestDriveNextPrivateFileLifecycle(t *testing.T) {
	if os.Getenv("NEXT_DRIVE_LIVE") != "1" {
		t.Skip("requires explicitly selected private live batch")
	}
	folder := os.Getenv("NEXT_DRIVE_LIVE_FOLDER_TOKEN")
	expected := os.Getenv("NEXT_DRIVE_LIVE_BINDING_SHA256")
	state := os.Getenv("NEXT_DRIVE_LIVE_STATE_DIR")
	if folder == "" || len(expected) != 64 || state == "" || os.Getenv("NEXT_DRIVE_LIVE_CREATION_VERIFIED") != "1" {
		t.Fatal("requires supplied private folder, independent account binding, creation proof, and private local state directory")
	}
	if err := os.MkdirAll(state, 0o700); err != nil {
		t.Fatal("cannot create private evidence directory")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	work := t.TempDir()
	sequence := 0
	run := func(callCtx context.Context, label, identity string, args ...string) *clie2e.Result {
		t.Helper()
		result, err := clie2e.RunCmdWithRetry(callCtx, clie2e.Request{Args: args, DefaultAs: identity, WorkDir: work}, clie2e.RetryOptions{Attempts: 1})
		sequence++
		raw, marshalErr := json.Marshal(map[string]any{"result": result, "host_error": err != nil})
		if marshalErr != nil || os.WriteFile(filepath.Join(state, fmt.Sprintf("%02d-%s.json", sequence, label)), raw, 0o600) != nil {
			t.Fatal("cannot preserve private command evidence")
		}
		if err != nil || result == nil {
			t.Fatal("live command host failure; inspect private evidence")
		}
		return result
	}
	success := func(label string, result *clie2e.Result) string {
		t.Helper()
		if result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
			t.Fatalf("%s failed: exit=%d; inspect private evidence for typed cause (no retry)", label, result.ExitCode)
		}
		return result.Stdout
	}
	auth := run(ctx, "auth", "", "auth", "status", "--verify", "--json")
	if auth.ExitCode != 0 || !gjson.Get(auth.Stdout, "verified").Bool() || !gjson.Get(auth.Stdout, "identities.user.verified").Bool() {
		t.Fatal("user authentication is not verified")
	}
	self := success("self", run(ctx, "self", "user", "contact", "+get-user"))
	uid, tenant := gjson.Get(self, "data.user.open_id").String(), gjson.Get(self, "data.user.tenant_key").String()
	binding := sha256.Sum256([]byte("contact-current-account-v1\x00" + uid + "\x00" + tenant))
	if uid == "" || tenant == "" || hex.EncodeToString(binding[:]) != expected || gjson.Get(auth.Stdout, "identities.user.openId").String() != uid {
		t.Fatal("independent current-account/tenant binding mismatch")
	}
	metadata := func(callCtx context.Context, label, token, kind string) *clie2e.Result {
		body, _ := json.Marshal(map[string]any{"request_docs": []map[string]string{{"doc_token": token, "doc_type": kind}}, "with_url": true})
		return run(callCtx, label, "user", "drive", "metas", "batch_query", "--data", string(body))
	}
	verify := func(token, kind string) {
		t.Helper()
		meta := success("metadata", metadata(ctx, kind+"-meta", token, kind))
		settings := success("settings", run(ctx, kind+"-settings", "user", "drive", "+permission-get-setting", "--token", token, "--type", kind))
		members := success("members", run(ctx, kind+"-members", "user", "drive", "+member-list", "--token", token, "--type", kind))
		if !nextDrivePrivateTarget(meta, settings, members, token, kind, uid) {
			t.Fatal("target owner, complete membership, or closed-link privacy check failed before dependent writes")
		}
	}
	verify(folder, "folder")
	payload := make([]byte, 4096)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	before := sha256.Sum256(payload)
	source := filepath.Join(work, "source.bin")
	if err := os.WriteFile(source, payload, 0o600); err != nil {
		t.Fatal("cannot write synthetic input")
	}
	uploaded := ""
	cleanupDone := false
	cleanup := func() {
		if uploaded == "" || uploaded == folder || cleanupDone {
			return
		}
		cleanupDone = true // even ambiguous deletion must not be replayed
		cleanupCtx, stop := clie2e.CleanupContext()
		defer stop()
		deleted := run(cleanupCtx, "delete", "user", "drive", "+delete", "--file-token", uploaded, "--type", "file", "--yes")
		if deleted.ExitCode != 0 || !gjson.Get(deleted.Stdout, "ok").Bool() {
			t.Errorf("delete returned exit=%d; original outcome retained, no retry", deleted.ExitCode)
		}
		absent := false
		for i := 0; i < 10; i++ {
			// Metadata can survive deletion. Require the exact resource's native
			// deletion response, not an empty URL or a generic access failure.
			probe := run(cleanupCtx, "absence", "user", "drive", "+download", "--file-token", uploaded, "--output", filepath.Join(work, fmt.Sprintf("after-delete-%d.bin", i)))
			if nextDriveDeletedResponse(probe) {
				absent = true
				break
			}
			if probe.ExitCode != 0 {
				t.Error("absence read lacks explicit resource-deleted evidence")
				return
			}
			select {
			case <-cleanupCtx.Done():
				t.Error("absence verification timed out")
				return
			case <-time.After(time.Second):
			}
		}
		if !absent {
			t.Error("uploaded file remains downloadable after bounded deletion verification")
		}
	}
	t.Cleanup(cleanup)
	upload := run(ctx, "upload", "user", "drive", "+upload", "--file", source, "--folder-token", folder, "--name", "next-private-"+clie2e.GenerateSuffix()+".bin")
	// Preserve a returned ID for cleanup even if the receipt reports an unknown effect.
	uploaded = gjson.Get(upload.Stdout, "data.file_token").String()
	success("upload", upload)
	if uploaded == "" || uploaded == folder {
		uploaded = ""
		t.Fatal("upload returned no distinct file identity; inspect private evidence without replay")
	}
	verify(uploaded, "file")
	output := filepath.Join(work, "download.bin")
	success("download", run(ctx, "download", "user", "drive", "+download", "--file-token", uploaded, "--output", output))
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(payload, got) || sha256.Sum256(got) != before {
		t.Fatal("download bytes/hash differ from synthetic input")
	}
	original, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(original, payload) {
		t.Fatal("upload source changed")
	}
	cleanup()
	preserved := success("folder preserved", metadata(ctx, "folder-preserved", folder, "folder"))
	if !nextDriveMetaOwned(preserved, folder, "folder", uid) {
		t.Fatal("supplied folder identity/owner no longer present")
	}
	if t.Failed() {
		return
	}
	t.Logf("private file lifecycle verified: bytes=%d sha256=%x; one upload and one delete; supplied folder preserved", len(payload), before)
}

func nextDrivePrivateTarget(meta, settings, members, token, kind, uid string) bool {
	if !nextDriveMetaOwned(meta, token, kind, uid) {
		return false
	}
	if gjson.Get(settings, "data.permission_public.link_share_entity").String() != "closed" {
		return false
	}
	items := gjson.Get(members, "data.items")
	if !items.IsArray() || len(items.Array()) != 1 || gjson.Get(members, "data.has_more").Bool() || gjson.Get(members, "data.page_token").String() != "" {
		return false
	}
	member := items.Array()[0]
	return member.Get("member_type").String() == "openid" && member.Get("member_id").String() == uid && member.Get("perm").String() == "full_access"
}

func TestNextDrivePrivateTarget(t *testing.T) {
	meta := `{"data":{"metas":[{"doc_token":"fixture","doc_type":"folder","owner_id":"self"}]}}`
	settings := `{"data":{"permission_public":{"link_share_entity":"closed","external_access_entity":"open"}}}`
	members := `{"data":{"items":[{"member_type":"openid","member_id":"self","perm":"full_access"}]}}`
	if !nextDrivePrivateTarget(meta, settings, members, "fixture", "folder", "self") {
		t.Fatal("closed link with sole matching owner rejected")
	}
	if nextDriveMetaOwned(`{"data":{"metas":[]}}`, "fixture", "folder", "self") || nextDriveMetaOwned(meta, "fixture", "folder", "other") {
		t.Fatal("missing folder or wrong owner accepted")
	}
	for _, bad := range []string{`{}`, `{"data":{"items":null}}`, `{"data":{"items":[]}}`, `{"data":{"items":[{"member_type":"openid","member_id":"other","perm":"full_access"}]}}`, `{"data":{"items":[{"member_type":"chat_id","member_id":"self","perm":"full_access"}]}}`} {
		if nextDrivePrivateTarget(meta, settings, bad, "fixture", "folder", "self") {
			t.Fatal("unproved membership accepted")
		}
	}
	if nextDrivePrivateTarget(meta, `{"data":{"permission_public":{"link_share_entity":"tenant_readable"}}}`, members, "fixture", "folder", "self") || nextDrivePrivateTarget(meta, settings, members, "other", "folder", "self") {
		t.Fatal("broad access or wrong target accepted")
	}
}

// Observed on the exact uploaded resource after successful deletion. The server
// currently classifies 1063005 as "Resource is deleted"; retain both fields so
// unrelated permission failures cannot masquerade as cleanup success.
func nextDriveDeletedResponse(result *clie2e.Result) bool {
	return result != nil && result.ExitCode != 0 &&
		gjson.Get(result.Stderr, "error.type").String() == "api" &&
		gjson.Get(result.Stderr, "error.code").Int() == 1063005 &&
		gjson.Get(result.Stderr, "error.message").String() == "Resource is deleted"
}

func TestNextDriveDeletedResponse(t *testing.T) {
	for _, tc := range []struct {
		body string
		want bool
	}{
		{`{"error":{"type":"api","code":1063005,"message":"Resource is deleted"}}`, true},
		{`{"error":{"type":"authorization","code":1063004,"message":"No permission"}}`, false},
		{`{"error":{"type":"api","code":1063005,"message":"Unrecognized response"}}`, false},
		{`{"error":{"type":"api","code":404,"message":"not found or inaccessible"}}`, false},
		{`{}`, false},
	} {
		if nextDriveDeletedResponse(&clie2e.Result{ExitCode: 1, Stderr: tc.body}) != tc.want {
			t.Fatal("incorrect resource-deleted classification")
		}
	}
	if nextDriveDeletedResponse(nil) || nextDriveDeletedResponse(&clie2e.Result{ExitCode: 0}) {
		t.Fatal("successful or missing response is not deletion evidence")
	}
}

func nextDriveMetaOwned(meta, token, kind, uid string) bool {
	rows := gjson.Get(meta, "data.metas")
	if !rows.IsArray() || len(rows.Array()) != 1 {
		return false
	}
	row := rows.Array()[0]
	return row.Get("doc_token").String() == token && row.Get("doc_type").String() == kind && row.Get("owner_id").String() == uid
}
