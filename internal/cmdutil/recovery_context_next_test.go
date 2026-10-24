// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"testing"

	"github.com/larksuite/cli/internal/recovery"
)

func TestRecoveryContextUsesBuildIdentityAndInvocationFallback(t *testing.T) {
	for _, tc := range []struct {
		name    string
		factory *Factory
		want    string
	}{
		{"nil", nil, "lark-cli auth login"},
		{"legacy", &Factory{Invocation: InvocationContext{Profile: "legacy profile"}}, "lark-cli auth login --profile='legacy profile'"},
		{"next", &Factory{Invocation: InvocationContext{Profile: "next profile"}, Recovery: recovery.NewProjectorWithContext(nil, recovery.RenderContext{CommandName: "feishu-cli-next"})}, "feishu-cli-next auth login --profile='next profile'"},
		{"build snapshot", &Factory{Invocation: InvocationContext{Profile: "other"}, Recovery: recovery.NewProjectorWithContext(nil, recovery.RenderContext{CommandName: "feishu-cli-next", Profile: "snapshot"})}, "feishu-cli-next auth login --profile='snapshot'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			context := tc.factory.RecoveryContext()
			if got := context.AuthLoginCommand(""); got != tc.want {
				t.Fatalf("recovery command = %q, want %q", got, tc.want)
			}
			context.CommandName = "changed"
			if got := tc.factory.RecoveryContext().AuthLoginCommand(""); got != tc.want {
				t.Fatal("mutated build-local context")
			}
		})
	}
}
