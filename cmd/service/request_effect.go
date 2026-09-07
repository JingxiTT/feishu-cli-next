// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package service

import (
	"net/url"
	"strings"

	"github.com/larksuite/cli/internal/client"
	"github.com/larksuite/cli/internal/meta"
)

// knownGeneratedRead corrects frozen POST risk annotations using their
// native query contracts. All coordinates are exact; absolute URLs, substitute
// paths and methods with only a matching name retain their declared risk.
func knownGeneratedRead(servicePath string, method meta.Method) bool {
	if method.HTTPMethod != "POST" || method.Risk != "write" {
		return false
	}
	return (servicePath == "/open-apis/attendance/v1" && method.ID == "user_task.query" && method.Path == "user_tasks/query") ||
		(servicePath == "/open-apis/approval/v4" && method.ID == "approvals.search" && method.Path == "approvals/search_launchable") ||
		(servicePath == "/open-apis/mail/v1" && ((method.ID == "user_mailbox.message.batch_get" && method.Path == "user_mailboxes/{user_mailbox_id}/messages/batch_get") ||
			(method.ID == "multi_entity.search" && method.Path == "multi_entity/search") ||
			(method.ID == "user_mailbox.search" && method.Path == "user_mailboxes/{user_mailbox_id}/search")))
}

// knownGeneratedReadPath binds a recognized native read to its concrete built
// path. Path parameters have already passed buildServiceRequest validation;
// compare their decoded path to the HTTP transport's URL.Path representation.
// Do not apply this metadata assertion to arbitrary raw API requests.
func knownGeneratedReadPath(servicePath string, method meta.Method, request client.RawApiRequest) (string, bool) {
	if !knownGeneratedRead(servicePath, method) || request.Method != method.HTTPMethod {
		return "", false
	}
	parsed, err := url.Parse(request.URL)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	actual := strings.Split(parsed.Path, "/")
	template := strings.Split(servicePath+"/"+method.Path, "/")
	if len(actual) != len(template) {
		return "", false
	}
	for i, part := range template {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			if actual[i] == "" {
				return "", false
			}
		} else if actual[i] != part {
			return "", false
		}
	}
	return parsed.Path, true
}
