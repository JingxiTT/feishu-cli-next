// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package guide

import (
	"encoding/json"
	"io"
	"testing"
	"testing/fstest"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/nextskill"
)

func TestGuideCommandLifecycle(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	project := t.TempDir()
	content := fstest.MapFS{"feishu-cli-next/SKILL.md": &fstest.MapFile{Data: []byte("# entry skill\n")}}
	for _, args := range [][]string{nil, {"show"}, {"install", "--project", project}, {"update", "--project", project}, {"uninstall", "--project", project}} {
		f, out, errOut, _ := cmdutil.TestFactory(t, nil)
		cmd := NewCmd(f, content)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr %v: %s", args, errOut)
		}
		if len(args) < 2 {
			if out.String() != "# entry skill\n" {
				t.Fatalf("show changed embedded content: %q", out.String())
			}
			continue
		}
		var result nextskill.Result
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !result.OK || result.Action != args[0] || result.Path == "" {
			t.Fatalf("invalid result: %+v", result)
		}
	}
}
