// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recovery

import (
	"strings"
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestUserAuthorizationUsesBuildLocalCommandName(t *testing.T) {
	hint := UserAuthorization("docx:document")
	legacy := hint.String()
	projector := NewProjectorWithContext(nil, RenderContext{CommandName: "feishu-cli-next", Profile: "team'$(touch /tmp/should-not-run)`"})
	wantCommand := "feishu-cli-next auth login --profile='team'\"'\"'$(touch /tmp/should-not-run)`'"
	want := "run `` " + wantCommand + " --scope \"docx:document\" --no-wait --json `` to get device_code and verification_url; " +
		"present verification_url to the user exactly and end this turn; after the user confirms authorization, " +
		"run `` " + wantCommand + " --device-code <device_code> `` in a later turn to finish login"
	if got := projector.RenderHint(hint); got != want {
		t.Fatalf("next recovery = %q, want %q", got, want)
	}
	if got := NewProjector(nil).RenderHint(hint); got != legacy {
		t.Fatalf("legacy hint changed: %q", got)
	}
	if got := hint.String(); got != legacy {
		t.Fatalf("producer hint mutated: %q", got)
	}
}

func TestNextContextOnlyRendersOwnedAuthorizationFragment(t *testing.T) {
	hint := UserAuthorization("docx:document")
	custom := "user text: lark-cli auth login; "
	source := errs.NewAuthenticationError(errs.SubtypeTokenMissing, custom).
		WithHint("%s%s", custom, hint.String())
	annotated := Annotate(source, hint)
	projector := NewProjectorWithContext(nil, RenderContext{CommandName: "feishu-cli-next"})
	got := projector.Render(annotated)
	problem, ok := errs.ProblemOf(got)
	if !ok {
		t.Fatalf("not typed: %v", got)
	}
	want := custom + strings.ReplaceAll(hint.String(), "lark-cli auth login", "feishu-cli-next auth login")
	if problem.Hint != want || problem.Message != custom {
		t.Fatalf("unexpected presentation: %+v", problem)
	}
	if problem == &source.Problem || source.Hint != custom+hint.String() {
		t.Fatal("presentation mutated producer")
	}
	raw := errs.MarkRaw(annotated)
	if projector.Render(raw) != raw {
		t.Fatal("raw error was rewritten")
	}
	unannotated := errs.NewAuthenticationError(errs.SubtypeTokenMissing, custom).WithHint("%s", custom)
	untouched, _ := errs.ProblemOf(projector.Render(unannotated))
	if untouched.Hint != custom || untouched.Message != custom {
		t.Fatal("unannotated text was rewritten")
	}
}
