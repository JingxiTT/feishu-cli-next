// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package common

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/credential"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/spf13/cobra"
)

func TestLegacyShortcutDryRunDoesNotResolveTokensOrClients(t *testing.T) {
	for _, identity := range []string{"user", "bot"} {
		t.Run(identity, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			f, stdout, stderr, _ := cmdutil.TestFactory(t, &core.CliConfig{})
			resolver := &scopeCheckTokenResolver{err: context.Canceled}
			f.Credential = credential.NewCredentialProvider(nil, nil, resolver, nil)
			f.LarkClient = func() (*lark.Client, error) {
				t.Fatal("dry-run initialized an API client")
				return nil, nil
			}
			s := Shortcut{
				Service: "fixture", Command: "+preview", Risk: "read",
				AuthTypes: []string{"user", "bot"}, Scopes: []string{"fixture:read"},
				Validate: func(_ context.Context, runtime *RuntimeContext) error {
					if err := runtime.EnsureScopes([]string{"fixture:conditional"}); err != nil {
						return err
					}
					if _, err := runtime.CallAPITyped("GET", "/open-apis/fixture/v1/lookup", nil, nil); !errs.IsValidation(err) {
						t.Fatalf("dry-run API guard = %v", err)
					}
					return nil
				},
				DryRun: func(_ context.Context, _ *RuntimeContext) *DryRunAPI {
					return NewDryRunAPI().GET("/open-apis/fixture/v1/preview")
				},
				Execute: func(_ context.Context, _ *RuntimeContext) error {
					t.Fatal("dry-run executed the shortcut")
					return nil
				},
			}
			root := &cobra.Command{Use: "fixture"}
			s.Mount(root, f)
			root.SetArgs([]string{"+preview", "--as", identity, "--dry-run"})
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			var result struct {
				OK       bool   `json:"ok"`
				Identity string `json:"identity"`
				DryRun   bool   `json:"dry_run"`
			}
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.OK || !result.DryRun || result.Identity != identity || resolver.calls != 0 || stderr.Len() != 0 {
				t.Fatalf("result = %+v, token calls = %d, stderr = %s", result, resolver.calls, stderr)
			}
		})
	}
}
