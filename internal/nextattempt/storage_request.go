// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"context"
	"net/http"
)

type storagePutKey struct{}

// WithStoragePutRequest declares a domain-known presigned object PUT whose final
// HTTP 200, 201 or 204 response confirms storage. The exact decoded URL path is
// invocation-only context; it is never persisted in the receipt. This assertion
// must not be inferred from arbitrary API requests or user flags.
func WithStoragePutRequest(ctx context.Context, apiPath string) context.Context {
	return context.WithValue(ctx, storagePutKey{}, apiPath)
}

// IsStoragePutRequest recognizes only the explicitly declared PUT/path pair.
func IsStoragePutRequest(req *http.Request) bool {
	path, ok := req.Context().Value(storagePutKey{}).(string)
	return ok && req.Method == http.MethodPut && req.URL != nil && path != "" && path == req.URL.Path
}
