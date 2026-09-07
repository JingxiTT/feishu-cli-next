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
// invocation. User resolution reuses the credential provider's cached lookup;
// missing tenant evidence alone never triggers another network request.
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
	if identity == core.AsUser {
		binding.UserID = cfg.UserOpenId
		if f.Credential != nil {
			resolved, err := f.Credential.ResolveUserBinding(ctx)
			if err != nil {
				return output.ArtifactBinding{}, err
			}
			binding.UserID, binding.TenantKey, binding.UserVerified = resolved.OpenID, resolved.TenantKey, resolved.Verified
			binding.TenantVerified = resolved.TenantKey != ""
		}
	}
	if binding.Environment == "" || binding.AppID == "" || (identity == core.AsUser && binding.UserID == "") {
		return output.ArtifactBinding{}, errs.NewValidationError(errs.SubtypeFailedPrecondition, "artifact binding requires a resolved account and identity")
	}
	return binding, nil
}
