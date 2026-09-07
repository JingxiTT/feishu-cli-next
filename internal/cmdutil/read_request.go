// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"net/http"
)

type readOnlyRequestKey struct{}
type readOnlyRequest struct{ method, path string }

// WithReadOnlyRequest marks one method/path tuple as read-only according to
// domain-owned operation semantics. apiPath is the exact URL path, without a
// query. Never derive this assertion from arbitrary API flags or request bodies.
// Other method/path tuples retain normal mutation journaling, even when they
// reuse the returned context. The command-wide context and risk are unchanged.
func WithReadOnlyRequest(ctx context.Context, method, apiPath string) context.Context {
	return context.WithValue(ctx, readOnlyRequestKey{}, readOnlyRequest{method: method, path: apiPath})
}

func isDeclaredReadRequest(req *http.Request) bool {
	declared, ok := req.Context().Value(readOnlyRequestKey{}).(readOnlyRequest)
	return ok && req.URL != nil && declared.method == req.Method && declared.path == req.URL.Path
}
