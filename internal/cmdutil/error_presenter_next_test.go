// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/recovery"
)

func TestFactoryPresentErrorUsesNextAuthorizationCommand(t *testing.T) {
	source := errs.NewPermissionError(errs.SubtypeMissingScope, "user text lark-cli auth login").WithIdentity("user")
	f := &Factory{ResolvedIdentity: core.AsUser, Recovery: recovery.NewProjectorWithContext(nil,
		recovery.RenderContext{CommandName: "feishu-cli-next", Profile: "next profile"})}
	rendered := f.PresentError(source, ErrorPresentationOptions{
		DeclaredScopes: func() []string { return []string{"calendar:calendar.event:read"} },
	})
	problem, ok := errs.ProblemOf(rendered)
	if !ok {
		t.Fatalf("not a typed error: %v", rendered)
	}
	if strings.Count(problem.Hint, "feishu-cli-next auth login --profile='next profile'") != 2 {
		t.Fatalf("incorrect invocation recovery: %s", problem.Hint)
	}
	if problem.Message != source.Message || source.Hint != "" {
		t.Fatal("rewrote unrelated message or mutated producer")
	}
}
