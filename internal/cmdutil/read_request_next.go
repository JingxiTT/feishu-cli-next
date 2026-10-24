// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"net/http"

	"github.com/larksuite/cli/internal/nextattempt"
)

type requestEffectKey struct{}
type requestEffect struct {
	method, path string
	mutation     bool
}

// WithReadOnlyRequest marks one method/path tuple as read-only according to
// domain-owned operation semantics. apiPath is the exact URL path, without a
// query. Never derive this assertion from arbitrary API flags or request bodies.
// Other method/path tuples retain normal mutation journaling, even when they
// reuse the returned context. The command-wide context and risk are unchanged.
func WithReadOnlyRequest(ctx context.Context, method, apiPath string) context.Context {
	return context.WithValue(ctx, requestEffectKey{}, requestEffect{method: method, path: apiPath})
}

// WithMutationRequest identifies an exact domain-known lifecycle mutation
// performed by a command otherwise classified as read-only. Credential requests
// remain excluded by the transport. Never infer this from arbitrary user input.
func WithMutationRequest(ctx context.Context, method, apiPath string) context.Context {
	return context.WithValue(ctx, requestEffectKey{}, requestEffect{method: method, path: apiPath, mutation: true})
}

// WithStoragePutRequest identifies a domain-known presigned object upload whose
// final HTTP 200/201/204 response confirms storage. apiPath is the exact decoded
// URL path without its query. Use only with an authoritative storage protocol.
func WithStoragePutRequest(ctx context.Context, apiPath string) context.Context {
	return nextattempt.WithStoragePutRequest(ctx, apiPath)
}

func declaredRequestEffect(req *http.Request) (mutation, known bool) {
	declared, ok := req.Context().Value(requestEffectKey{}).(requestEffect)
	if !ok || req.URL == nil || declared.method != req.Method || declared.path != req.URL.Path {
		return false, false
	}
	return declared.mutation, true
}
