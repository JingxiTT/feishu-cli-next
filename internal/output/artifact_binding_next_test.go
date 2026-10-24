// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package output_test

import (
	"errors"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/output"
)

func TestValidateArtifactBinding(t *testing.T) {
	verified := output.ArtifactBinding{Environment: "feishu", Endpoint: "https://fixture.invalid", AppID: "app", Identity: "user", UserID: "user", TenantKey: "tenant", UserVerified: true, TenantVerified: true, Profile: "original"}
	for _, tc := range []struct {
		name      string
		change    func(*output.ArtifactBinding)
		strict    bool
		wantError bool
	}{
		{"same", func(*output.ArtifactBinding) {}, true, false},
		{"renamed profile", func(b *output.ArtifactBinding) { b.Profile = "renamed" }, true, false},
		{"environment", func(b *output.ArtifactBinding) { b.Environment = "lark" }, false, true},
		{"endpoint", func(b *output.ArtifactBinding) { b.Endpoint = "https://other.invalid" }, false, true},
		{"missing endpoint", func(b *output.ArtifactBinding) { b.Endpoint = "" }, false, true},
		{"app", func(b *output.ArtifactBinding) { b.AppID = "other" }, false, true},
		{"identity", func(b *output.ArtifactBinding) { b.Identity = "bot" }, false, true},
		{"unresolved identity", func(b *output.ArtifactBinding) { b.Identity = "auto" }, false, true},
		{"user", func(b *output.ArtifactBinding) { b.UserID = "other" }, false, true},
		{"tenant", func(b *output.ArtifactBinding) { b.TenantKey = "other" }, false, true},
		{"known source unknown current tenant", func(b *output.ArtifactBinding) { b.TenantKey = ""; b.TenantVerified = false }, false, true},
		{"unverified tenant strict", func(b *output.ArtifactBinding) { b.TenantVerified = false }, true, true},
		{"unverified user strict", func(b *output.ArtifactBinding) { b.UserVerified = false }, true, true},
		{"unverified user nonstrict", func(b *output.ArtifactBinding) { b.UserVerified = false }, false, false},
		{"fabricated tenant proof", func(b *output.ArtifactBinding) { b.TenantKey = "" }, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			current := verified
			tc.change(&current)
			err := output.ValidateArtifactBinding(verified, current, "--artifact", tc.strict)
			if !tc.wantError {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var validation *errs.ValidationError
			if !errors.As(err, &validation) || validation.Param != "--artifact" || validation.Subtype != errs.SubtypeFailedPrecondition {
				t.Fatalf("wrong mismatch error: %v", err)
			}
		})
	}
}

func TestValidateArtifactBindingRequiresBothSidesEvidence(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		binding := output.ArtifactBinding{Environment: "feishu", Endpoint: "https://fixture.invalid", AppID: "app", Identity: identity, UserID: "user"}
		if err := output.ValidateArtifactBinding(binding, binding, "--input", false); err != nil {
			t.Fatal(err)
		}
		if err := output.ValidateArtifactBinding(binding, binding, "--input", true); !errs.IsValidation(err) {
			t.Fatalf("unknown tenant claimed verified: %v", err)
		}
	}
	verified := output.ArtifactBinding{Environment: "feishu", Endpoint: "https://fixture.invalid", AppID: "app", Identity: "user", UserID: "user", TenantKey: "tenant", TenantVerified: true, UserVerified: true}
	for _, drop := range []func(*output.ArtifactBinding){func(b *output.ArtifactBinding) { b.UserVerified = false }, func(b *output.ArtifactBinding) { b.TenantVerified = false }} {
		source := verified
		drop(&source)
		if err := output.ValidateArtifactBinding(source, verified, "--input", true); !errs.IsValidation(err) {
			t.Fatalf("unverified source accepted: %v", err)
		}
	}
}
