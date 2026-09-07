// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive_test

import (
	"os"
	"testing"

	"github.com/larksuite/cli/tests/nextfixture"
)

func TestMain(m *testing.M) { os.Exit(nextfixture.RunMain(m, "../../..")) }
