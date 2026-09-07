// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package application

import (
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
)

func TestResolveCommandNameRequiresExactMatch(t *testing.T) {
	f, stdout, _, reg := cmdutil.TestFactory(t, appTestConfig())
	reg.Register(listStub([]interface{}{sampleItem("greet", "id1")}))
	err := mountAndRun(t, SlashCommandUpdate, []string{"+slash-command-update", "--command", "Greet", "--description", "new", "--as", "bot"}, f, stdout)
	p, ok := errs.ProblemOf(err)
	if !ok || p.Category != errs.CategoryAPI || p.Subtype != errs.SubtypeNotFound {
		t.Fatalf("exact match required: %v", err)
	}
}

func TestResolveNotFoundErrorShape(t *testing.T) {
	err := commandNotFoundError("nope")
	if err == nil {
		t.Fatalf("err = %v", err)
	}
	p, ok := errs.ProblemOf(err)
	if !ok || p.Category != errs.CategoryAPI || p.Subtype != errs.SubtypeNotFound {
		t.Fatalf("expected api/not_found, got %#v", p)
	}
}
