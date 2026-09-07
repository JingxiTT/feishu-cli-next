// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import (
	"testing"

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
