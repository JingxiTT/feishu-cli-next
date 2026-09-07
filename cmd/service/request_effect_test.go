// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import (
	"testing"

	"github.com/larksuite/cli/internal/client"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/meta"
)

func TestGeneratedReadRiskUsesExactCoordinatesAndPreservesLegacy(t *testing.T) {
	for _, tt := range []struct {
		name, servicePath, id, path, method string
		read                                bool
	}{
		{"attendance", "/open-apis/attendance/v1", "user_task.query", "user_tasks/query", "POST", true},
		{"approval", "/open-apis/approval/v4", "approvals.search", "approvals/search_launchable", "POST", true},
		{"mail batch get", "/open-apis/mail/v1", "user_mailbox.message.batch_get", "user_mailboxes/{user_mailbox_id}/messages/batch_get", "POST", true},
		{"mail entity search", "/open-apis/mail/v1", "multi_entity.search", "multi_entity/search", "POST", true},
		{"mail message search", "/open-apis/mail/v1", "user_mailbox.search", "user_mailboxes/{user_mailbox_id}/search", "POST", true},
		{"mail batch modify", "/open-apis/mail/v1", "user_mailbox.message.batch_modify", "user_mailboxes/{user_mailbox_id}/messages/batch_modify", "POST", false},
		{"mail copied read ID on modify", "/open-apis/mail/v1", "user_mailbox.message.batch_get", "user_mailboxes/{user_mailbox_id}/messages/batch_modify", "POST", false},
		{"wrong ID", "/open-apis/attendance/v1", "user_task.create", "user_tasks/query", "POST", false},
		{"wrong path", "/open-apis/attendance/v1", "user_task.query", "user_tasks/create", "POST", false},
		{"wrong method", "/open-apis/attendance/v1", "user_task.query", "user_tasks/query", "PUT", false},
		{"wrong service", "/open-apis/other/v1", "user_task.query", "user_tasks/query", "POST", false},
		{"absolute service URL", "https://fixture.invalid/open-apis/attendance/v1", "user_task.query", "user_tasks/query", "POST", false},
		{"absolute method URL", "/open-apis/attendance/v1", "user_task.query", "https://fixture.invalid/user_tasks/query", "POST", false},
		{"path prefix", "/open-apis/attendance/v1", "user_task.query", "user_tasks/query/other", "POST", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, enabled := range []bool{false, true} {
				f, _, _, _ := cmdutil.TestFactory(t, testConfig)
				f.EnableAgentAttempts = enabled
				service := meta.ServiceFromMap(map[string]interface{}{"name": "fixture", "servicePath": tt.servicePath})
				method := meta.FromMap(map[string]interface{}{"id": tt.id, "path": tt.path, "httpMethod": tt.method, "risk": "write", "scopes": []interface{}{"synthetic:read"}})
				command := NewCmdServiceMethod(f, service, method, "query", "records", nil)
				want := "write"
				if enabled && tt.read {
					want = "read"
				}
				if risk, _ := cmdutil.GetRisk(command); risk != want {
					t.Fatalf("enabled=%v risk=%s want=%s", enabled, risk, want)
				}
				if method.Risk != "write" || len(method.Scopes) != 1 || method.Scopes[0] != "synthetic:read" || command.Flags().Lookup("yes") != nil {
					t.Fatal("changed frozen method scopes, risk or confirmation flags")
				}
			}
		})
	}
}

func TestKnownGeneratedReadPathBindsConcreteValidatedPath(t *testing.T) {
	method := meta.FromMap(map[string]interface{}{"id": "user_mailbox.message.batch_get", "path": "user_mailboxes/{user_mailbox_id}/messages/batch_get", "httpMethod": "POST", "risk": "write"})
	for _, tt := range []struct{ name, method, path, want string }{
		{"encoded address", "POST", "/open-apis/mail/v1/user_mailboxes/fixture%2Bread%40example.invalid/messages/batch_get", "/open-apis/mail/v1/user_mailboxes/fixture+read@example.invalid/messages/batch_get"},
		{"mailbox alias", "POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get"},
		{"empty mailbox", "POST", "/open-apis/mail/v1/user_mailboxes//messages/batch_get", ""},
		{"neighbor mutation", "POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_modify", ""},
		{"method mismatch", "PUT", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get", ""},
		{"absolute URL", "POST", "https://example.invalid/open-apis/mail/v1/user_mailboxes/me/messages/batch_get", ""},
		{"extra suffix", "POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get/other", ""},
		{"embedded query", "POST", "/open-apis/mail/v1/user_mailboxes/me/messages/batch_get?other=true", ""},
		{"encoded extra segment", "POST", "/open-apis/mail/v1/user_mailboxes/me%2Fother/messages/batch_get", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, known := knownGeneratedReadPath("/open-apis/mail/v1", method, client.RawApiRequest{Method: tt.method, URL: tt.path})
			if got != tt.want || known != (tt.want != "") {
				t.Fatalf("path=%q known=%v want=%q", got, known, tt.want)
			}
		})
	}
}
