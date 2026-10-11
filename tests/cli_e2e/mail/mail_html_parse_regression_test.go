// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package mail

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	clie2e "github.com/larksuite/cli/tests/cli_e2e"
	"github.com/stretchr/testify/require"
)

func TestMail_HTMLParseRegression(t *testing.T) {
	setMailTriageDryRunEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	body := strings.Repeat("<div>", 600) + `<script>alert(1)</script>` + strings.Repeat("</div>", 600)
	result, err := clie2e.RunCmd(ctx, clie2e.Request{Args: []string{"mail", "+lint-html", "--body", body}, DefaultAs: "user"})
	require.NoError(t, err)
	result.AssertExitCode(t, 2)
	require.Empty(t, result.Stdout)
	var failure struct {
		Error struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(result.Stderr), &failure))
	require.Equal(t, "validation", failure.Error.Type)
	require.Equal(t, "invalid_argument", failure.Error.Subtype)
}

func TestMail_HTMLParseDraftCreateDryRun(t *testing.T) {
	setMailTriageDryRunEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	body := strings.Repeat("<div>", 600) + "text" + strings.Repeat("</div>", 600)
	result, err := clie2e.RunCmd(ctx, clie2e.Request{Args: []string{"mail", "+draft-create", "--mailbox", "alias@example.com", "--to", "recipient@example.com", "--subject", "Parse preview", "--body", body, "--dry-run"}, DefaultAs: "user"})
	require.NoError(t, err)
	result.AssertExitCode(t, 0)
	require.Equal(t, "GET", clie2e.DryRunGet(result.Stdout, "api.0.method").String())
	require.Equal(t, "/open-apis/mail/v1/user_mailboxes/alias@example.com/profile", clie2e.DryRunGet(result.Stdout, "api.0.url").String())
	require.Equal(t, "POST", clie2e.DryRunGet(result.Stdout, "api.1.method").String())
	require.Equal(t, "/open-apis/mail/v1/user_mailboxes/alias@example.com/drafts", clie2e.DryRunGet(result.Stdout, "api.1.url").String())
	require.Equal(t, "<base64url-EML>", clie2e.DryRunGet(result.Stdout, "api.1.body.raw").String())
	require.Equal(t, "Parse preview", clie2e.DryRunGet(result.Stdout, "api.1.body._preview.subject").String())
	require.Equal(t, "recipient@example.com", clie2e.DryRunGet(result.Stdout, "api.1.body._preview.to").String())
}
