// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package recovery

import (
	"testing"

	"github.com/larksuite/cli/errs"
)

func TestRewriteCommandNameRewritesInvocations(t *testing.T) {
	cases := map[string]string{
		"Domain guide: lark-cli skills read lark-apps":       "Domain guide: feishu-cli-next skills read lark-apps",
		"run `lark-cli config init --new` in the background": "run `feishu-cli-next config init --new` in the background",
		"lark-cli auth login\nlark-cli schema x":             "feishu-cli-next auth login\nfeishu-cli-next schema x",
		"lark-cli is not bound to it":                        "feishu-cli-next is not bound to it",
		"(run lark-cli.)":                                    "(run feishu-cli-next.)",
	}
	for in, want := range cases {
		if got := RewriteCommandName(in, "feishu-cli-next"); got != want {
			t.Fatalf("RewriteCommandName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRewriteCommandNameLeavesNonCommandText(t *testing.T) {
	cases := []string{
		"see github.com/larksuite/cli for source",
		"set LARKSUITE_CLI_APP_ID before running",
		"config at ~/.lark-cli/config.json and .lark-cli/policy.yml",
		"use lark-cli.yml or my_lark-cli or lark-cli-old",
		"path lark-cli/bin",
	}
	for _, in := range cases {
		if got := RewriteCommandName(in, "feishu-cli-next"); got != in {
			t.Fatalf("RewriteCommandName(%q) = %q, want unchanged", in, got)
		}
	}
}

func TestRewriteCommandNameIdentityForUpstreamOrEmptyName(t *testing.T) {
	in := "run `lark-cli config init --new` in ~/.lark-cli"
	for _, name := range []string{"", "lark-cli"} {
		if got := RewriteCommandName(in, name); got != in {
			t.Fatalf("RewriteCommandName(name=%q) = %q, want identity", name, got)
		}
	}
}

func TestLocalizeErrorRewritesClonedProblemOnly(t *testing.T) {
	source := errs.NewConfigError(errs.SubtypeNotConfigured, "lark-cli is not bound").
		WithHint("run `lark-cli config init --new`")
	got := LocalizeError(source, "feishu-cli-next")
	problem, ok := errs.ProblemOf(got)
	if !ok {
		t.Fatalf("localized error lost its problem: %v", got)
	}
	if problem.Message != "feishu-cli-next is not bound" || problem.Hint != "run `feishu-cli-next config init --new`" {
		t.Fatalf("unexpected localized problem: %+v", problem)
	}
	if producer, _ := errs.ProblemOf(source); producer.Message != "lark-cli is not bound" {
		t.Fatalf("producer problem mutated: %+v", producer)
	}
	if LocalizeError(source, "lark-cli") != source {
		t.Fatal("upstream command name must return the error unchanged")
	}
}

func TestLocalizeErrorReturnsUnrelatedErrorUnchanged(t *testing.T) {
	source := errs.NewValidationError(errs.SubtypeInvalidArgument, "bad --limit").WithHint("use a number")
	if LocalizeError(source, "feishu-cli-next") != source {
		t.Fatal("an error without the upstream command name must not be cloned")
	}
}
