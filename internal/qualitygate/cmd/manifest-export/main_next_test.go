// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"testing"

	"github.com/larksuite/cli/internal/qualitygate/manifest"
)

func TestCollectIncludesNextDistributionControls(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	for name, collect := range map[string]func(context.Context) (manifest.Manifest, error){
		"hand-authored": collectHandAuthored,
		"full-index":    collectCommandIndex,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := collect(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			for _, path := range []string{"discover", "guide install", "receipts inspect", "docs +fetch"} {
				command := findManifestCommand(&got, path)
				if command == nil || !command.Runnable {
					t.Fatalf("next command missing or not runnable: %s", path)
				}
				for _, flag := range []string{"result-fields", "result-detail", "save-result"} {
					if findManifestFlag(command, flag) == nil {
						t.Fatalf("%s missing --%s", path, flag)
					}
				}
			}
		})
	}
}
