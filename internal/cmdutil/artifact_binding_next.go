// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/output"
)

// ResolveArtifactBinding returns effective, non-secret account evidence for this
// requested artifact operation. Read-only identity verification is cached by
// the credential provider; ordinary reads do not call this method.
func (f *Factory) ResolveArtifactBinding(ctx context.Context, identity core.Identity) (output.ArtifactBinding, error) {
	if f == nil || f.Config == nil {
		return output.ArtifactBinding{}, errs.NewInternalError(errs.SubtypeUnknown, "artifact binding has no configuration resolver")
	}
	if identity != core.AsUser && identity != core.AsBot {
		return output.ArtifactBinding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact identity has not been resolved")
	}
	cfg, err := f.Config()
	if err != nil {
		return output.ArtifactBinding{}, err
	}
	if cfg == nil {
		return output.ArtifactBinding{}, errs.NewInternalError(errs.SubtypeUnknown, "artifact configuration resolver returned no account")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	binding := output.ArtifactBinding{Environment: string(cfg.Brand), Endpoint: core.ResolveEndpoints(cfg.Brand).Open, AppID: cfg.AppID, Identity: string(identity), Profile: cfg.ProfileName}
	if f.Credential == nil {
		return output.ArtifactBinding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact binding requires a credential resolver for identity verification")
	}
	resolved, err := f.Credential.ResolveArtifactIdentity(ctx, identity)
	if err != nil {
		return output.ArtifactBinding{}, err
	}
	binding.TenantKey, binding.TenantVerified = resolved.TenantKey, true
	if identity == core.AsUser {
		binding.UserID, binding.UserVerified = resolved.OpenID, true
	}
	if binding.Environment == "" || binding.AppID == "" || (identity == core.AsUser && binding.UserID == "") {
		return output.ArtifactBinding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact binding requires a resolved account and identity")
	}
	return binding, nil
}
