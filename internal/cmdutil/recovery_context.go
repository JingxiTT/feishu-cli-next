// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import "github.com/larksuite/cli/internal/recovery"

// RecoveryContext shares the build-local executable identity with local auth
// workflows. Factories without a projector keep legacy command naming and the
// explicit invocation profile.
func (f *Factory) RecoveryContext() recovery.RenderContext {
	if f == nil {
		return recovery.RenderContext{}
	}
	context := f.Recovery.Context()
	if context.Profile == "" {
		context.Profile = f.Invocation.Profile
	}
	return context
}
