// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/larksuite/cli/internal/vfs"
)

func TestInstallWizardRoutesOnlyProjectGuideAndPreservesExplicitSetup(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatalf("Node is required to verify the shipped npm setup wrappers: %v", err)
	}
	// Execute the actual shipped JavaScript in a restricted VM. The only
	// permitted effect is one native CLI invocation; unexpected filesystem or
	// package-manager operations fail instead of reaching profiles/global skills.
	const probe = `
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const scripts = process.argv[1];
const sources = Object.fromEntries(["install-wizard.js", "run.js"].map(name =>
  [name, fs.readFileSync(path.join(scripts, name), "utf8")]));
const project = path.join(process.argv[2], "project with spaces 世界");
for (const platform of ["darwin", "linux", "win32"]) {
  const binary = path.join(scripts, "..", "bin", "feishu-cli-next" + (platform === "win32" ? ".exe" : ""));
  const cases = [
    ["install-wizard.js", ["--project", project], ["guide", "install", "--project", project]],
    ["run.js", ["install", "--project", project], ["guide", "install", "--project", project]],
    ["run.js", ["guide", "update", "--project", project], ["guide", "update", "--project", project]],
    ["run.js", ["config", "init", "--new", "--lang", "en"], ["config", "init", "--new", "--lang", "en"]],
    ["run.js", ["auth", "login", "--domain", "docs", "--no-wait", "--json"], ["auth", "login", "--domain", "docs", "--no-wait", "--json"]],
  ];
  for (const [entry, args, expected] of cases) {
    for (const failure of [false, true]) {
      const calls = [];
      const probes = [];
      function execute(name, argv) {
        const modules = {
          path,
          fs: { existsSync(candidate) {
            assert.ok(candidate === binary || candidate === binary + ".old", "unexpected filesystem probe: " + candidate);
            probes.push(candidate);
            return candidate === binary;
          } },
          child_process: { execFileSync(command, commandArgs) {
            const nativeArgs = Array.from(commandArgs);
            if (command === process.execPath) {
              assert.equal(nativeArgs[0], path.join(scripts, "run.js"));
              return execute("run.js", nativeArgs.slice(1));
            }
            assert.equal(command, binary, "setup must not invoke npm/npx, global skills, or another executable");
            calls.push(nativeArgs);
            if (failure) throw Object.assign(new Error("synthetic native failure"), {status: 23});
            return Buffer.from("{}");
          } },
        };
        const context = {
          __dirname: scripts,
          require(module) {
            const value = modules[module.replace(/^node:/, "")];
            assert.ok(value, "unexpected module access: " + module);
            return value;
          },
          process: {platform, execPath: process.execPath, argv: [process.execPath, path.join(scripts, name), ...argv], env: {},
            exit(status) { throw Object.assign(new Error("process exit"), {status, testExit: true}); }},
          console: { error() {} },
        };
        return vm.runInNewContext(sources[name], context, {filename: name, timeout: 1000});
      }
      let exit = 0;
      try { execute(entry, args); }
      catch (error) { if (!error.testExit) throw error; exit = error.status; }
      assert.deepEqual(calls, [expected], "exactly one invocation with unchanged arguments is required");
      assert.equal(exit, failure ? 23 : 0, "native exit status must survive both wrappers");
      assert.ok(probes.includes(binary));
    }
  }
}
`
	isolated := t.TempDir()
	command := exec.Command(node, "-e", probe, filepath.Join(filepath.Dir(currentFile), "scripts"), isolated)
	command.Env = []string{"HOME=" + isolated, "USERPROFILE=" + isolated, "TMPDIR=" + isolated, "LARKSUITE_CLI_CONFIG_DIR=" + isolated}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("verify npm setup routing without global/profile effects: %v\n%s", err, output)
	}
}

func TestLarkSuiteArchivePathsFitSkillsInstallerTarLimit(t *testing.T) {
	// npx skills v1.5.21 did not honor PAX long-path metadata while extracting
	// Agent Skills v0.2 archives and instead used tar's 100-byte name field. A
	// 101-byte suite entry ending in .md was consequently installed as .m. Keep
	// generated suite paths within this compatibility limit until the upstream
	// extractor reliably supports long archive paths.
	const maxTarPathBytes = 100

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	skillsRoot := filepath.Join(filepath.Dir(currentFile), "skills")

	var walk func(string, string)
	walk = func(dir, relativeDir string) {
		t.Helper()
		entries, err := vfs.ReadDir(dir)
		if err != nil {
			t.Fatalf("read skills directory %s: %v", dir, err)
		}
		for _, entry := range entries {
			relativePath := filepath.Join(relativeDir, entry.Name())
			archivePath := filepath.ToSlash(filepath.Join("references", relativePath))
			if entry.IsDir() {
				archivePath += "/"
			}
			if pathBytes := len([]byte(archivePath)); pathBytes > maxTarPathBytes {
				t.Errorf("suite archive path %q is %d bytes; npx skills compatibility limit is %d bytes", archivePath, pathBytes, maxTarPathBytes)
			}
			if entry.IsDir() {
				walk(filepath.Join(dir, entry.Name()), relativePath)
			}
		}
	}

	walk(skillsRoot, "")
}
