// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import "github.com/larksuite/cli/internal/meta"

// knownGeneratedRead corrects two frozen POST risk annotations using their
// native query contracts. All coordinates are exact; absolute URLs, substitute
// paths and methods with only a matching name retain their declared risk.
func knownGeneratedRead(servicePath string, method meta.Method) bool {
	if method.HTTPMethod != "POST" || method.Risk != "write" {
		return false
	}
	return (servicePath == "/open-apis/attendance/v1" && method.ID == "user_task.query" && method.Path == "user_tasks/query") ||
		(servicePath == "/open-apis/approval/v4" && method.ID == "approvals.search" && method.Path == "approvals/search_launchable")
}
