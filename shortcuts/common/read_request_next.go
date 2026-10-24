// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"

	"github.com/larksuite/cli/internal/nextattempt"
)

// WithReadOnlyRequest asserts read-only semantics for an exact method and URL
// path using domain-owned operation knowledge. It does not change command risk.
// Use only for known read operations, never arbitrary user API specifications.
func WithReadOnlyRequest(ctx context.Context, method, apiPath string) context.Context {
	return nextattempt.WithReadOnlyRequest(ctx, method, apiPath)
}

// CallReadOnlyAPITyped calls a domain-known read operation, including POST reads
// inside write workflows, preserving CallAPITyped's authentication and errors.
func (ctx *RuntimeContext) CallReadOnlyAPITyped(method, apiPath string, params map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	return ctx.callAPITypedWithContext(WithReadOnlyRequest(ctx.ctx, method, apiPath), method, apiPath, params, data)
}

// WithMutationRequest identifies one domain-known lifecycle mutation inside a
// read workflow. Other method/path pairs and credential requests are unaffected.
func WithMutationRequest(ctx context.Context, method, apiPath string) context.Context {
	return nextattempt.WithMutationRequest(ctx, method, apiPath)
}

// WithStoragePutRequest marks an exact decoded URL path for a domain-known
// presigned PUT whose final HTTP 200/201/204 response confirms object storage.
func WithStoragePutRequest(ctx context.Context, apiPath string) context.Context {
	return nextattempt.WithStoragePutRequest(ctx, apiPath)
}

// CallMutationAPITyped performs a domain-known lifecycle mutation inside a read
// workflow while preserving the usual authentication and typed error behavior.
func (ctx *RuntimeContext) CallMutationAPITyped(method, apiPath string, params map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	return ctx.callAPITypedWithContext(WithMutationRequest(ctx.ctx, method, apiPath), method, apiPath, params, data)
}
