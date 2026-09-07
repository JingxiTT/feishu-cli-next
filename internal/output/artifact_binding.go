// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output

import "github.com/larksuite/cli/errs"

// ValidateArtifactBinding compares source and effective destination identity.
// Profile is a local label and does not participate. Strict verification means
// observed tenant evidence on both sides and, for user identity, observed user
// evidence on both sides. Even non-strict matching cannot equate a known tenant
// with an absent one. This does not validate artifact schema, extent, or content.
func ValidateArtifactBinding(saved, current ArtifactBinding, param string, requireVerifiedIdentity bool) error {
	reject := func(message string) error {
		return errs.NewValidationError(errs.SubtypeFailedPrecondition, message).WithParam(param)
	}
	for _, binding := range []ArtifactBinding{saved, current} {
		if binding.Environment == "" || binding.Endpoint == "" || binding.AppID == "" || (binding.Identity != "user" && binding.Identity != "bot") || (binding.Identity == "user" && binding.UserID == "") {
			return reject("artifact account binding is incomplete")
		}
		if binding.TenantVerified && binding.TenantKey == "" {
			return reject("artifact tenant verification has no tenant identity")
		}
	}
	if saved.Environment != current.Environment || saved.Endpoint != current.Endpoint {
		return reject("artifact environment does not match the current environment")
	}
	if saved.AppID != current.AppID || saved.Identity != current.Identity {
		return reject("artifact account or effective identity does not match the current account")
	}
	if saved.Identity == "user" && saved.UserID != current.UserID {
		return reject("artifact user does not match the current user")
	}
	if saved.TenantKey != current.TenantKey {
		return reject("artifact tenant cannot be matched to the current tenant")
	}
	if requireVerifiedIdentity {
		if !saved.TenantVerified || !current.TenantVerified {
			return reject("artifact reuse requires verified tenant identity on both sides")
		}
		if saved.Identity == "user" && (!saved.UserVerified || !current.UserVerified) {
			return reject("artifact reuse requires verified user identity on both sides")
		}
	}
	return nil
}
