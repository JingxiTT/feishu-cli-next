// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/recovery"
)

func TestAgentBuildScopesAuthorizationCommandIdentity(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	options := []BuildOption{WithoutPlugins(), WithoutStrictMode(), WithoutServiceCommands(), WithIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})}
	invocation := cmdutil.InvocationContext{Profile: "team'next"}
	legacy, _, _ := buildInternal(context.Background(), invocation, options...)
	next, _, _ := buildInternal(context.Background(), invocation, append(options, WithAgentMode())...)
	for _, tc := range []struct {
		runtime *buildRuntime
		name    string
	}{{next, "feishu-cli-next"}, {legacy, "lark-cli"}} {
		command := tc.name + " auth login --profile='team'\"'\"'next'"
		if got := tc.runtime.Factory.RecoveryContext().AuthLoginCommand(""); got != command {
			t.Fatalf("local auth context: %q, want %q", got, command)
		}
		hint := tc.runtime.recovery.RenderHint(recovery.UserAuthorization())
		if strings.Count(hint, command) != 2 {
			t.Fatalf("structured recovery context mismatch: %s", hint)
		}
	}
}
