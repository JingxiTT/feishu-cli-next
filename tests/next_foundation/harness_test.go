// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"net/http"
	"os"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) {
	os.Exit(nextfixture.RunMain(m, "../.."))
}

type Fixture = nextfixture.Fixture
type Result = nextfixture.Result

func NewFixture(t *testing.T, handler http.Handler) *Fixture {
	t.Helper()
	return nextfixture.NewFixture(t, handler)
}
