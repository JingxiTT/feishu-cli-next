// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"

	"github.com/larksuite/cli/internal/cmdutil"
)

// WithReadOnlyRequest asserts read-only semantics for an exact method and URL
// path using domain-owned operation knowledge. It does not change command risk.
// Use only for known read operations, never arbitrary user API specifications.
func WithReadOnlyRequest(ctx context.Context, method, apiPath string) context.Context {
	return cmdutil.WithReadOnlyRequest(ctx, method, apiPath)
}

// CallReadOnlyAPITyped calls a domain-known read operation, including POST reads
// inside write workflows, preserving CallAPITyped's authentication and errors.
func (ctx *RuntimeContext) CallReadOnlyAPITyped(method, apiPath string, params map[string]interface{}, data interface{}) (map[string]interface{}, error) {
	return ctx.callAPITypedWithContext(WithReadOnlyRequest(ctx.ctx, method, apiPath), method, apiPath, params, data)
}
