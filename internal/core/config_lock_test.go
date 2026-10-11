// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package core

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/larksuite/cli/errs"
)

func TestConfigLockAcrossProcesses(t *testing.T) {
	if os.Getenv("LARK_CLI_TEST_CONFIG_LOCK_CHILD") == "1" {
		fmt.Println("ready")
		err := WithConfigLock(func() error {
			config, err := LoadMultiAppConfig()
			if err != nil {
				return err
			}
			if config.CurrentApp != "parent-update" {
				return fmt.Errorf("read stale config: %q", config.CurrentApp)
			}
			config.Apps[0].Users = []AppUser{{UserOpenId: "child-update"}}
			return SaveMultiAppConfig(config)
		})
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	if err := SaveMultiAppConfig(&MultiAppConfig{Apps: []AppConfig{{Name: "profile", AppId: "test-app"}}}); err != nil {
		t.Fatal(err)
	}
	child := exec.Command(os.Args[0], "-test.run=^TestConfigLockAcrossProcesses$")
	child.Env = append(os.Environ(), "LARK_CLI_TEST_CONFIG_LOCK_CHILD=1")
	stdout, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	child.Stderr = os.Stderr
	done := make(chan error, 1)
	err = WithConfigLock(func() error {
		if err := child.Start(); err != nil {
			return err
		}
		go func() { done <- child.Wait() }()
		scanner := bufio.NewScanner(stdout)
		if !scanner.Scan() || scanner.Text() != "ready" {
			return fmt.Errorf("child did not start")
		}
		select {
		case err := <-done:
			return fmt.Errorf("child bypassed held config lock: %v", err)
		case <-time.After(100 * time.Millisecond):
		}
		config, err := LoadMultiAppConfig()
		if err != nil {
			return err
		}
		config.CurrentApp = "parent-update"
		return SaveMultiAppConfig(config)
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		child.Process.Kill()
		t.Fatal("child did not acquire released config lock")
	}
	config, err := LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.CurrentApp != "parent-update" || len(config.Apps[0].Users) != 1 || config.Apps[0].Users[0].UserOpenId != "child-update" {
		t.Fatalf("lost updates: %+v", config)
	}
}

func TestConfigLockPreservesCallbackErrorAndReleases(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	cause := errs.NewValidationError(errs.SubtypeInvalidArgument, "synthetic failure")
	if got := WithConfigLock(func() error { return cause }); !errors.Is(got, cause) {
		t.Fatalf("lost callback error: %v", got)
	}
	if err := WithConfigLock(func() error { return nil }); err != nil {
		t.Fatalf("lock not released: %v", err)
	}
}
