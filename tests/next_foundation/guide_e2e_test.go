// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package nextfoundation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This suite executes a built distribution, not an in-process Cobra command.
// The foundation runner supplies the canonical binary at section acceptance.
func TestGuideProcessLifecycle(t *testing.T) {
	binary := os.Getenv("LARK_CLI_BIN")
	if binary == "" {
		t.Skip("set LARK_CLI_BIN to the built feishu-cli-next binary")
	}
	if !filepath.IsAbs(binary) {
		t.Fatal("LARK_CLI_BIN must be absolute")
	}
	isolated := t.TempDir()
	project := filepath.Join(isolated, "project with spaces")
	home := filepath.Join(isolated, "home")
	for _, dir := range []string{project, home} {
		if err := os.Mkdir(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	canonicalProject, err := filepath.EvalSymlinks(project)
	if err != nil {
		t.Fatal(err)
	}
	env := make([]string, 0)
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if key == "HOME" || key == "USERPROFILE" || strings.HasPrefix(key, "LARK") || strings.HasPrefix(key, "FEISHU") || strings.HasPrefix(key, "XDG_") {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "HOME="+home, "USERPROFILE="+home,
		"LARKSUITE_CLI_CONFIG_DIR="+filepath.Join(isolated, "config"), "LARKSUITE_CLI_REMOTE_META=off")
	run := func(wantCode int, args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env, cmd.Dir = env, project
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("execute %v: %v", args, err)
			}
			code = exit.ExitCode()
		}
		if code != wantCode {
			t.Fatalf("%v: exit=%d want=%d stdout=%s stderr=%s", args, code, wantCode, stdout.Bytes(), stderr.Bytes())
		}
		if code != 0 {
			var failure struct {
				Error struct {
					Type    string `json:"type"`
					Subtype string `json:"subtype"`
				} `json:"error"`
			}
			if stdout.Len() != 0 || json.Unmarshal(stderr.Bytes(), &failure) != nil || failure.Error.Type != "validation" || failure.Error.Subtype != "failed_precondition" {
				t.Fatalf("invalid failure: stdout=%s stderr=%s", stdout.Bytes(), stderr.Bytes())
			}
		} else if stderr.Len() != 0 {
			t.Fatalf("unexpected diagnostic: %s", stderr.Bytes())
		}
		return stdout.Bytes()
	}
	shown := run(0, "guide", "show")
	if !bytes.HasPrefix(shown, []byte("---\nname: feishu-cli-next\n")) {
		t.Fatalf("not embedded next skill: %q", shown)
	}
	if !bytes.Equal(shown, run(0, "guide")) {
		t.Fatal("guide and guide show differ")
	}
	lifecycle := func(action string, changed bool) {
		t.Helper()
		data := run(0, "guide", action, "--project", project)
		var result struct {
			OK      bool   `json:"ok"`
			Action  string `json:"action"`
			Path    string `json:"path"`
			Changed bool   `json:"changed"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("%s: %v", data, err)
		}
		if !result.OK || result.Action != action || result.Changed != changed || result.Path != filepath.Join(canonicalProject, ".agents", "skills", "feishu-cli-next", "SKILL.md") {
			t.Fatalf("bad lifecycle result: %+v", result)
		}
	}
	lifecycle("install", true)
	target := filepath.Join(project, ".agents", "skills", "feishu-cli-next", "SKILL.md")
	original, err := os.ReadFile(target)
	if err != nil || !bytes.HasPrefix(original, shown) {
		t.Fatalf("installed content differs: %v", err)
	}
	extra := filepath.Join(filepath.Dir(target), "keep.txt")
	if err := os.WriteFile(extra, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	lifecycle("install", false)
	lifecycle("update", false)
	edited := append([]byte("local change\n"), original...)
	if err := os.WriteFile(target, edited, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"install", "update", "uninstall"} {
		run(2, "guide", action)
		after, err := os.ReadFile(target)
		if err != nil || !bytes.Equal(after, edited) {
			t.Fatalf("%s changed local edits: %v", action, err)
		}
	}
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatal(err)
	}
	lifecycle("uninstall", true)
	lifecycle("uninstall", false)
	if _, err := os.Stat(target); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("skill remains: %v", err)
	}
	other, err := os.ReadFile(extra)
	if err != nil || string(other) != "keep" {
		t.Fatalf("unrelated file modified: %v", err)
	}
}
