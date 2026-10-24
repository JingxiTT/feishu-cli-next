// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/spf13/cobra"
)

func TestConfigReadsDoNotCreateWriterLock(t *testing.T) {
	for _, command := range []func(*cmdutil.Factory) *cobra.Command{NewCmdConfigDefaultAs, NewCmdConfigRiskControl, NewCmdConfigStrictMode} {
		t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
		if err := core.SaveMultiAppConfig(&core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "test-app"}}}); err != nil {
			t.Fatal(err)
		}
		f, _, _, _ := cmdutil.TestFactory(t, nil)
		cmd := command(f)
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(core.GetConfigDir(), "config.lock")); !os.IsNotExist(err) {
			t.Fatalf("read created writer lock: %v", err)
		}
	}
}

func TestRiskControlMutationWaitsForConfigTransaction(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	initial := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "test-app"}}}
	if err := core.SaveMultiAppConfig(initial); err != nil {
		t.Fatal(err)
	}
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigRiskControl(f)
	cmd.SetArgs([]string{"off"})
	done := make(chan error, 1)
	var early bool
	if err := core.WithConfigLock(func() error {
		go func() { done <- cmd.Execute() }()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
			early = true
		case <-time.After(100 * time.Millisecond):
		}
		initial.StrictMode = core.StrictModeBot
		return core.SaveMultiAppConfig(initial)
	}); err != nil {
		t.Fatal(err)
	}
	if !early {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	saved, err := core.LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if early || saved.StrictMode != core.StrictModeBot || saved.RiskControl == nil || *saved.RiskControl {
		t.Fatalf("lost concurrent config update: %+v", saved)
	}
}

func TestDPoPMutationWaitsForConfigTransaction(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	initial := &core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "test-app"}}}
	if err := core.SaveMultiAppConfig(initial); err != nil {
		t.Fatal(err)
	}
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigSet(f)
	cmd.SetArgs([]string{"dpop", "disabled"})
	done := make(chan error, 1)
	var early bool
	if err := core.WithConfigLock(func() error {
		go func() { done <- cmd.Execute() }()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
			early = true
		case <-time.After(100 * time.Millisecond):
		}
		initial.StrictMode = core.StrictModeBot
		return core.SaveMultiAppConfig(initial)
	}); err != nil {
		t.Fatal(err)
	}
	if !early {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	saved, err := core.LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if early || saved.StrictMode != core.StrictModeBot || saved.Apps[0].DPoPMode != core.DPoPModeDisabled {
		t.Fatalf("lost concurrent config update: %+v", saved)
	}
}
