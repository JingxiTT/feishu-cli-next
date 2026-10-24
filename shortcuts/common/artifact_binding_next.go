// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

// ArtifactBinding is the shared account identity recorded in domain artifacts.
// Consumers also validate their native artifact schema, extent, and data types.
type ArtifactBinding = output.ArtifactBinding

// ResolveArtifactBinding resolves this runtime's effective identity using the
// same invocation-scoped credential cache as --save-result and mutation evidence.
func (ctx *RuntimeContext) ResolveArtifactBinding() (ArtifactBinding, error) {
	if ctx == nil {
		return ArtifactBinding{}, errs.NewInternalError(errs.SubtypeUnknown, "artifact binding has no runtime")
	}
	if ctx.offline {
		return ArtifactBinding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact identity resolution is unavailable during dry-run")
	}
	return ctx.Factory.ResolveArtifactBinding(ctx.Ctx(), ctx.As())
}

// ValidateArtifactBinding checks source against current identity before reuse.
// requireVerifiedIdentity=true requires observed tenant evidence on both sides
// and, for user operations, observed user evidence on both sides. False allows
// explicitly unverified identities but still rejects all mismatching known IDs.
// param is the caller's artifact input flag, for example --input.
func ValidateArtifactBinding(saved, current ArtifactBinding, param string, requireVerifiedIdentity bool) error {
	return output.ValidateArtifactBinding(saved, current, param, requireVerifiedIdentity)
}
