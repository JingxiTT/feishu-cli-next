// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package packaging_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPackagingOfflineLifecycle(t *testing.T) {
	if os.Getenv("LARK_CLI_BIN") == "" {
		t.Fatal("use scripts/next_test.py --section packaging --suite e2e --binary /path/to/feishu-cli-next")
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	cmd := exec.Command(python, "-m", "unittest", "discover", "-s", "tests/next_sections/packaging", "-p", "*_test.py", "-v")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatal(err)
	}
}
