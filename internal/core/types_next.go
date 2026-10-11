// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package core

import (
	"context"
)

type requestIdentityContextKey struct{}

// WithRequestIdentity binds the identity selected for an outbound token to that
// request flow. It does not change the invocation's default identity.
func WithRequestIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, requestIdentityContextKey{}, identity)
}

// RequestIdentityFromContext returns an explicit user or bot request identity.
func RequestIdentityFromContext(ctx context.Context) (Identity, bool) {
	if ctx == nil {
		return "", false
	}
	identity, ok := ctx.Value(requestIdentityContextKey{}).(Identity)
	return identity, ok && (identity == AsUser || identity == AsBot)
}
