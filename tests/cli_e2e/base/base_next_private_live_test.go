// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/tidwall/gjson"
)

// This dedicated opt-in probe uses real user credentials only after a separately
// verified private-at-creation destination and account proof are supplied. It
// never selects a bot, grants access, or retries a mutation. The shared offline
// authsidecar suite remains separate and does not run this live workflow.
func TestBaseNextPrivateLiveLifecycle(t *testing.T) {
	if os.Getenv("NEXT_BASE_PRIVATE_LIVE") != "1" {
		t.Skip("requires explicit private live selection and coordinator-verified fixture")
	}
	fixturePath, proofPath, statePath := os.Getenv("NEXT_BASE_PRIVATE_FIXTURE"), os.Getenv("NEXT_BASE_PRIVATE_BINDING"), os.Getenv("NEXT_BASE_PRIVATE_STATE")
	if fixturePath == "" || proofPath == "" || statePath == "" {
		t.Fatal("private live fixture, binding proof and new recovery-state path are required")
	}
	readProtected := func(path string) []byte {
		t.Helper()
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			t.Fatal("live manifest must be a private regular file")
		}
		raw, err := os.ReadFile(path)
		if err != nil || !json.Valid(raw) {
			t.Fatal("live manifest could not be read as JSON")
		}
		return raw
	}
	fixture, proof := string(readProtected(fixturePath)), string(readProtected(proofPath))
	folder := gjson.Get(fixture, "folder_token").String()
	if folder == "" || !gjson.Get(fixture, "created_by_this_run").Bool() || gjson.Get(fixture, "default_access_observed_before_creation").String() != "Only invited members can access" || !gjson.Get(fixture, "invited_collaborators").Exists() || gjson.Get(fixture, "invited_collaborators").Int() != 0 || !gjson.Get(fixture, "child_writes_require_cli_account_and_acl_preflight").Bool() {
		t.Fatal("private-at-creation fixture evidence is incomplete")
	}
	var components []string
	if json.Unmarshal([]byte(gjson.Get(proof, "utf8_components_nul_separated").Raw), &components) != nil || !reflect.DeepEqual(components, []string{"contact-current-account-v1", "open_id", "tenant_key"}) || gjson.Get(proof, "algorithm").String() != "sha256" {
		t.Fatal("unsupported binding proof format")
	}
	stateFile, err := os.OpenFile(statePath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal("cannot reserve private recovery state before any request")
	}
	state := map[string]interface{}{"status": "preflight", "cleanup": "not_needed", "steps": []string{}}
	persist := func() {
		t.Helper()
		if err := stateFile.Truncate(0); err != nil {
			t.Fatal("private recovery state truncate failed")
		}
		if _, err := stateFile.Seek(0, 0); err != nil {
			t.Fatal("private recovery state seek failed")
		}
		if err := json.NewEncoder(stateFile).Encode(state); err != nil {
			t.Fatal("private recovery state write failed")
		}
		if err := stateFile.Sync(); err != nil {
			t.Fatal("private recovery state sync failed")
		}
	}
	persist()
	t.Cleanup(func() { _ = stateFile.Close() })
	t.Cleanup(func() {
		if t.Failed() {
			state["status"] = "failed"
			persist()
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(cancel)
	once := func(callCtx context.Context, args ...string) (*clie2e.Result, error) {
		return clie2e.RunCmdWithRetry(callCtx, clie2e.Request{Args: args, DefaultAs: "user", Format: "json"}, clie2e.RetryOptions{Attempts: 1})
	}
	requireOK := func(label string, result *clie2e.Result, callErr error) string {
		t.Helper()
		if callErr != nil || result == nil || result.ExitCode != 0 || !gjson.Get(result.Stdout, "ok").Bool() {
			state["status"] = "failed"
			state["failed_step"] = label
			if result != nil {
				state["exit_code"] = result.ExitCode
				state["error_type"] = gjson.Get(result.Stderr, "error.type").String()
				state["error_subtype"] = gjson.Get(result.Stderr, "error.subtype").String()
				state["api_code"] = gjson.Get(result.Stderr, "error.code").Value()
			}
			persist()
			t.Fatalf("private live step %s failed; sanitized details and run-owned recovery coordinates are in the private state file; no mutation replay", label)
		}
		state["steps"] = append(state["steps"].([]string), label)
		persist()
		return result.Stdout
	}
	run := func(label string, args ...string) string {
		t.Helper()
		result, callErr := once(ctx, args...)
		return requireOK(label, result, callErr)
	}
	// auth status is identity discovery and has no --as flag. All business and
	// cleanup calls below carry explicit --as user through Request.DefaultAs.
	auth, authErr := clie2e.RunCmdWithRetry(ctx, clie2e.Request{Args: []string{"auth", "status", "--verify", "--json"}}, clie2e.RetryOptions{Attempts: 1})
	if authErr != nil || auth == nil || auth.ExitCode != 0 || !gjson.Get(auth.Stdout, "verified").Bool() || !gjson.Get(auth.Stdout, "identities.user.verified").Bool() {
		t.Fatal("fresh verified user authentication is required; no content writes performed")
	}
	self := run("current_account", "contact", "+get-user")
	openID, tenant := gjson.Get(self, "data.user.open_id").String(), gjson.Get(self, "data.user.tenant_key").String()
	digest := sha256.Sum256([]byte("contact-current-account-v1\x00" + openID + "\x00" + tenant))
	if openID == "" || tenant == "" || gjson.Get(auth.Stdout, "identities.user.openId").String() != openID || hex.EncodeToString(digest[:]) != gjson.Get(proof, "digest").String() {
		t.Fatal("current user or tenant differs from independent binding proof; no content writes performed")
	}
	privateResource := func(token, kind, label string) {
		t.Helper()
		settings := run(label+"_public", "drive", "+permission-get-setting", "--token", token, "--type", kind)
		members := run(label+"_members", "drive", "+member-list", "--token", token, "--type", kind)
		query, _ := json.Marshal(map[string]interface{}{"request_docs": []map[string]string{{"doc_token": token, "doc_type": kind}}, "with_url": false})
		meta := run(label+"_owner", "drive", "metas", "batch_query", "--data", string(query))
		if !baseLivePrivateACL(settings, members, meta, token, kind, openID) {
			t.Fatal("private live ACL/owner precondition failed; no further content writes")
		}
	}
	privateResource(folder, "folder", "parent")
	state["status"] = "creating"
	persist()
	created, createErr := once(ctx, "base", "+base-create", "--name", "next-private-base-"+clie2e.GenerateSuffix(), "--folder-token", folder)
	baseToken := ""
	if created != nil && gjson.Get(created.Stdout, "data.created").Bool() {
		baseToken = gjson.Get(created.Stdout, "data.base.base_token").String()
		if baseToken == "" {
			baseToken = gjson.Get(created.Stdout, "data.base.app_token").String()
		}
	}
	if baseToken != "" && baseToken != folder {
		// Register cleanup as soon as an authoritative returned resource ID is
		// available, before any child schema or record mutations.
		state["base_token"], state["cleanup"] = baseToken, "pending"
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cleanupCancel()
			deleted, deleteErr := once(cleanupCtx, "drive", "+delete", "--file-token", baseToken, "--type", "bitable", "--yes")
			if deleteErr != nil || deleted == nil || deleted.ExitCode != 0 || !gjson.Get(deleted.Stdout, "ok").Bool() || !gjson.Get(deleted.Stdout, "data.deleted").Bool() {
				state["cleanup"] = "unconfirmed_no_replay"
				persist()
				t.Error("run-owned Base cleanup is unconfirmed; inspect protected recovery state, do not replay blindly")
				return
			}
			state["cleanup"] = "delete_acknowledged_absence_unconfirmed"
			// A deletion acknowledgement is separate from independent absence.
			// Retain the exact native read evidence privately; never infer absence
			// from a missing URL, generic authorization failure, or transport error.
			absent, absentErr := once(cleanupCtx, "base", "+base-get", "--base-token", baseToken)
			if absent != nil {
				state["cleanup_read_exit"] = absent.ExitCode
				state["cleanup_read_stdout"] = json.RawMessage(validBaseLiveJSON(absent.Stdout))
				state["cleanup_read_stderr"] = json.RawMessage(validBaseLiveJSON(absent.Stderr))
			}
			persist()
			if absentErr != nil || absent == nil || !baseLiveDeletedProof(absent.ExitCode, absent.Stderr) {
				t.Error("delete acknowledged but independent Base absence is unconfirmed; protected native read evidence retained, no delete replay")
				return
			}
			state["cleanup"] = "confirmed"
			persist()
		})
		persist()
	}
	_ = requireOK("base_create", created, createErr)
	if baseToken == "" || baseToken == folder {
		t.Fatal("Base create returned no unique run-owned Base token; no replay or inferred cleanup")
	}
	privateResource(baseToken, "bitable", "child")
	const fields = `[{"name":"Title","type":"text"},{"name":"Amount","type":"number"},{"name":"Checked","type":"checkbox"},{"name":"Note","type":"text"}]`
	table := run("table_create", "base", "+table-create", "--base-token", baseToken, "--name", "Typed probe", "--fields", fields)
	tableID := gjson.Get(table, "data.table.id").String()
	if tableID == "" {
		tableID = gjson.Get(table, "data.table.table_id").String()
	}
	if tableID == "" {
		t.Fatal("table create returned no exact table ID; whole run-owned Base cleanup remains registered")
	}
	state["table_id"] = tableID
	persist()
	const rows = `{"create_records":[{"Title":"probe-A","Amount":1.25,"Checked":true,"Note":"保留 Café 🧪"},{"Title":"probe-B","Amount":0,"Checked":false,"Note":""}]}`
	records := run("records_create", "base", "+record-batch-create", "--base-token", baseToken, "--table-id", tableID, "--json", rows)
	ids := gjson.Get(records, "data.record_id_list").Array()
	if len(ids) != 2 || ids[0].String() == "" || ids[1].String() == "" || ids[0].String() == ids[1].String() {
		t.Fatal("record creation lacks complete distinct IDs; no replay")
	}
	readArgs := []string{"base", "+record-get", "--base-token", baseToken, "--table-id", tableID, "--record-id", ids[0].String(), "--record-id", ids[1].String()}
	checkRows := func(label string, expected map[string][]interface{}) {
		t.Helper()
		for attempt := 0; attempt < 5; attempt++ {
			result := run(label, readArgs...)
			state["last_record_read"] = json.RawMessage(validBaseLiveJSON(result))
			persist()
			if baseLiveRowsMatch(result, expected) {
				return
			}
			if attempt < 4 {
				time.Sleep(time.Second) // Read-only propagation wait; never repeats a mutation.
			}
		}
		t.Fatalf("%s did not preserve all requested types/values within the bounded read window", label)
	}
	checkRows("records_read", map[string][]interface{}{ids[0].String(): {"probe-A", 1.25, true, "保留 Café 🧪"}, ids[1].String(): {"probe-B", float64(0), false, ""}})
	run("record_update", "base", "+record-upsert", "--base-token", baseToken, "--table-id", tableID, "--record-id", ids[0].String(), "--json", `{"Amount":0,"Note":""}`)
	checkRows("update_readback", map[string][]interface{}{ids[0].String(): {"probe-A", float64(0), true, ""}, ids[1].String(): {"probe-B", float64(0), false, ""}})
	unchanged := run("unchanged_probe", "base", "+record-upsert", "--base-token", baseToken, "--table-id", tableID, "--record-id", ids[0].String(), "--json", `{"Amount":0,"Note":""}`, "--skip-unchanged")
	if !gjson.Get(unchanged, "data.unchanged").Bool() || gjson.Get(unchanged, "data.updated").Bool() || gjson.Get(unchanged, "mutation").Exists() {
		t.Fatal("unchanged update did not report a mutation-free no-op")
	}
	state["status"] = "passed"
	persist()
	t.Log("narrow private Base lifecycle passed; run-owned cleanup follows; full Base capability and live parity remain separate")
}

// Exact Base v3 response observed after acknowledged deletion of this
// workflow's run-owned Base; generic permission or transport failures do not prove absence.
func baseLiveDeletedProof(exit int, stderr string) bool {
	return exit != 0 && gjson.Get(stderr, "error.type").String() == "api" &&
		gjson.Get(stderr, "error.code").Int() == 1002 &&
		gjson.Get(stderr, "error.message").String() == "note has been deleted"
}

func validBaseLiveJSON(raw string) []byte {
	if json.Valid([]byte(raw)) {
		return []byte(raw)
	}
	return []byte(`null`)
}

func baseLivePrivateACL(settings, members, meta, token, kind, openID string) bool {
	if gjson.Get(settings, "data.permission_public.link_share_entity").String() != "closed" {
		return false
	}
	items := gjson.Get(members, "data.items")
	if !items.IsArray() || len(items.Array()) != 1 {
		return false
	}
	member := items.Array()[0]
	if member.Get("member_type").String() != "openid" || member.Get("member_id").String() != openID || member.Get("perm").String() != "full_access" || member.Get("perm_type").String() != "container" {
		return false
	}
	metas := gjson.Get(meta, "data.metas")
	if !metas.IsArray() || len(metas.Array()) != 1 {
		return false
	}
	item := metas.Array()[0]
	return item.Get("doc_token").String() == token && item.Get("doc_type").String() == kind && item.Get("owner_id").String() == openID
}

func baseLiveRowsMatch(raw string, expected map[string][]interface{}) bool {
	data := gjson.Get(raw, "data")
	ids, rows, fields := data.Get("record_id_list").Array(), data.Get("data").Array(), data.Get("fields").Array()
	if len(ids) != len(expected) || len(rows) != len(ids) || len(fields) != 4 || data.Get("has_more").Bool() || len(data.Get("record_not_found").Array()) != 0 {
		return false
	}
	positions := map[string]int{}
	fieldIDs, types := data.Get("field_id_list").Array(), data.Get("field_type_list").Array()
	if len(fieldIDs) != 4 || len(types) != 4 {
		return false
	}
	seenFields := map[string]bool{}
	wantTypes := map[string]string{"Title": "text", "Amount": "number", "Checked": "checkbox", "Note": "text"}
	for i, field := range fields {
		if fieldIDs[i].String() == "" || seenFields[fieldIDs[i].String()] || types[i].String() != wantTypes[field.String()] {
			return false
		}
		seenFields[fieldIDs[i].String()] = true
		positions[field.String()] = i
	}
	if len(positions) != 4 {
		return false
	}
	seen := map[string]bool{}
	for i, id := range ids {
		want, exists := expected[id.String()]
		if !exists || seen[id.String()] || len(rows[i].Array()) != 4 {
			return false
		}
		seen[id.String()] = true
		for j, name := range []string{"Title", "Amount", "Checked", "Note"} {
			position, found := positions[name]
			if !found || !reflect.DeepEqual(rows[i].Array()[position].Value(), want[j]) {
				return false
			}
		}
	}
	return true
}

func TestBaseNextPrivateLiveGuards(t *testing.T) {
	deleted := `{"error":{"type":"api","subtype":"unknown","code":1002,"message":"note has been deleted"}}`
	if !baseLiveDeletedProof(1, deleted) {
		t.Fatal("observed explicit deletion rejected")
	}
	for _, bad := range []string{`{}`, strings.Replace(deleted, "1002", "403", 1), strings.Replace(deleted, "note has been deleted", "permission denied", 1)} {
		if baseLiveDeletedProof(1, bad) {
			t.Fatal("unproven absence accepted")
		}
	}
	settings := `{"data":{"permission_public":{"link_share_entity":"closed"}}}`
	members := `{"data":{"items":[{"member_type":"openid","member_id":"self","perm":"full_access","perm_type":"container"}]}}`
	meta := `{"data":{"metas":[{"doc_token":"resource","doc_type":"folder","owner_id":"self"}]}}`
	if !baseLivePrivateACL(settings, members, meta, "resource", "folder", "self") {
		t.Fatal("complete private synthetic ACL rejected")
	}
	for _, changed := range []string{strings.Replace(settings, "closed", "tenant_readable", 1), `{}`} {
		if baseLivePrivateACL(changed, members, meta, "resource", "folder", "self") {
			t.Fatal("unverified or organization-visible sharing accepted")
		}
	}
	if baseLivePrivateACL(settings, strings.Replace(members, `}]`, `},{"member_id":"other"}]`, 1), meta, "resource", "folder", "self") || baseLivePrivateACL(settings, members, strings.Replace(meta, "self", "other", 1), "resource", "folder", "self") {
		t.Fatal("additional member or foreign owner accepted")
	}
	matrix := `{"data":{"fields":["Title","Amount","Checked","Note"],"field_id_list":["fld1","fld2","fld3","fld4"],"field_type_list":["text","number","checkbox","text"],"record_id_list":["record"],"data":[["Unicode 🧪",0,false,""]]}}`
	want := map[string][]interface{}{"record": {"Unicode 🧪", float64(0), false, ""}}
	if !baseLiveRowsMatch(matrix, want) {
		t.Fatal("supported native values rejected")
	}
	for _, bad := range []string{strings.Replace(matrix, `"number"`, `"text"`, 1), strings.Replace(matrix, `"fld2"`, `"fld1"`, 1), strings.Replace(matrix, `,0,`, `,"0",`, 1), strings.Replace(matrix, `false`, `null`, 1), strings.Replace(matrix, `false,""`, `false,null`, 1)} {
		if baseLiveRowsMatch(bad, want) {
			t.Fatal("type or empty/null corruption accepted")
		}
	}
}
