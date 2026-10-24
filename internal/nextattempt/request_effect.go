// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextattempt

import (
	"context"
	"net/http"
	"strings"

	"github.com/larksuite/cli/internal/core"
)

// Request-effect declarations. Domain code that knows an exact request's effect
// marks it on the request context; the journal reads the marks. Never derive a
// mark from arbitrary API flags, user input, or request bodies.

type requestEffectKey struct{}
type requestEffect struct {
	method, path string
	mutation     bool
}

// WithReadOnlyRequest marks one method/path tuple as read-only according to
// domain-owned operation semantics. apiPath is the exact URL path, without a
// query. Other method/path tuples retain normal mutation journaling, even when
// they reuse the returned context. The command-wide context and risk are unchanged.
func WithReadOnlyRequest(ctx context.Context, method, apiPath string) context.Context {
	return context.WithValue(ctx, requestEffectKey{}, requestEffect{method: method, path: apiPath})
}

// WithMutationRequest identifies an exact domain-known lifecycle mutation
// performed by a command otherwise classified as read-only. Credential requests
// remain excluded by the transport.
func WithMutationRequest(ctx context.Context, method, apiPath string) context.Context {
	return context.WithValue(ctx, requestEffectKey{}, requestEffect{method: method, path: apiPath, mutation: true})
}

// DeclaredRequestEffect reports whether req matches a declared tuple and, if so,
// whether it was declared a mutation.
func DeclaredRequestEffect(req *http.Request) (mutation, known bool) {
	declared, ok := req.Context().Value(requestEffectKey{}).(requestEffect)
	if !ok || req.URL == nil || declared.method != req.Method || declared.path != req.URL.Path {
		return false, false
	}
	return declared.mutation, true
}

type driveImportStatusKey struct{}

// WithDriveImportStatusRequest requires the native import status GET to finish
// at its requested endpoint. Redirects cannot supply completion or grant proof.
func WithDriveImportStatusRequest(ctx context.Context, apiPath string) context.Context {
	return context.WithValue(ctx, driveImportStatusKey{}, apiPath)
}

// IsDriveImportStatusRequest recognizes only the declared status GET against a
// Feishu or Lark open-platform origin.
func IsDriveImportStatusRequest(req *http.Request) bool {
	if req == nil || req.URL == nil {
		return false
	}
	path, ok := req.Context().Value(driveImportStatusKey{}).(string)
	const prefix = "/open-apis/drive/v1/import_tasks/"
	if !ok || req.Method != "GET" || !core.IsPlatformEndpointURL(req.URL) || path != req.URL.EscapedPath() || !strings.HasPrefix(req.URL.Path, prefix) {
		return false
	}
	ticket := strings.TrimPrefix(req.URL.EscapedPath(), prefix)
	origin := "https://" + strings.ToLower(req.URL.Hostname())
	return ticket != "" && !strings.Contains(ticket, "/") && (origin == core.ResolveEndpoints(core.BrandFeishu).Open || origin == core.ResolveEndpoints(core.BrandLark).Open)
}
