// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package contact

import (
	"errors"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestContactFanoutAllFailedError_PreservesIdentityFailures(t *testing.T) {
	cases := []error{
		errs.NewPermissionError(errs.SubtypeMissingScope, "missing scope").WithCode(99991679).WithLogID("synthetic-log").WithMissingScopes("contact:user:search"),
		errs.NewAuthenticationError(errs.SubtypeTokenExpired, "expired token").WithCode(99991663),
	}
	for _, original := range cases {
		t.Run(original.Error(), func(t *testing.T) {
			before, _ := errs.ProblemOf(original)
			originalMessage := before.Message
			aggregate := contactFanoutAllFailedError(original, "all 2 queries failed")
			after, ok := errs.ProblemOf(aggregate)
			if !ok || after.Category != before.Category || after.Subtype != before.Subtype || after.Code != before.Code || after.LogID != before.LogID {
				t.Fatalf("identity metadata lost: before=%+v after=%+v", before, after)
			}
			if after.Message != "all 2 queries failed" || before.Message != originalMessage || !errors.Is(aggregate, original) {
				t.Fatalf("aggregate lost cause or mutated original: %v", aggregate)
			}
			var permission *errs.PermissionError
			if errors.As(aggregate, &permission) && (len(permission.MissingScopes) != 1 || permission.MissingScopes[0] != "contact:user:search") {
				t.Fatalf("missing scopes lost: %+v", permission)
			}
		})
	}
}
