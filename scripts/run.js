#!/usr/bin/env node
// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

const { execFileSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const ext = process.platform === "win32" ? ".exe" : "";
const bin = path.join(__dirname, "..", "bin", "feishu-cli-next" + ext);

// On Windows, a crashed self-update may have left the binary renamed to .old.
// Recover it before proceeding so the CLI remains functional.
const oldBin = bin + ".old";
function restoreOldBinary() {
  try {
    if (fs.existsSync(bin)) {
      fs.rmSync(bin, { force: true });
    }
    fs.renameSync(oldBin, bin);
    return true;
  } catch (_) {
    return false;
  }
}

if (process.platform === "win32" && fs.existsSync(oldBin)) {
  if (!fs.existsSync(bin)) {
    restoreOldBinary();
  } else {
    try {
      execFileSync(bin, ["--version"], { stdio: "ignore", timeout: 10000 });
      try {
        fs.rmSync(oldBin, { force: true });
      } catch (_) {
        // Best-effort cleanup; keep running the healthy binary.
      }
    } catch (_) {
      restoreOldBinary();
    }
  }
}

// The npm setup alias installs the embedded skill in the selected project.
const args = process.argv.slice(2);
{
  // Auto-download binary if missing (e.g. npx skipped postinstall).
  if (!fs.existsSync(bin)) {
    try {
      execFileSync(process.execPath, [path.join(__dirname, "install.js")], {
        stdio: "inherit",
        env: { ...process.env, FEISHU_CLI_NEXT_RUN: "true" },
      });
    } catch (_) {
      console.error(
        `\nFailed to auto-install feishu-cli-next binary.\n` +
        `To fix, run the install script manually:\n` +
        `  node "${path.join(__dirname, "install.js")}"\n`
      );
      process.exit(1);
    }
  }

  try {
    execFileSync(bin, args[0] === "install" ? ["guide", "install", ...args.slice(1)] : args, { stdio: "inherit" });
  } catch (e) {
    process.exit(e.status || 1);
  }
}
