// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import "github.com/larksuite/cli/internal/core"

// ResolveSupportedIdentity returns the command's only supported identity when
// the current identity was auto-detected, so a single-identity command does not
// fail on the auto-detected fallback. An explicit --as or a configured default-as
// keeps the chosen identity, and CheckIdentity still rejects it.
func (f *Factory) ResolveSupportedIdentity(as core.Identity, supported []string) core.Identity {
	if !f.IdentityAutoDetected || len(supported) != 1 {
		return as
	}
	return core.Identity(supported[0])
}
