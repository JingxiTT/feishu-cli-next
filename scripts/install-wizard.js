#!/usr/bin/env node
// Next setup maps installation to the embedded project skill. Auth/config remain
// explicit native commands so installation never changes accounts or profiles.
const { execFileSync } = require("node:child_process");
const path = require("node:path");
try {
  execFileSync(process.execPath, [path.join(__dirname, "run.js"), "install", ...process.argv.slice(2)], { stdio: "inherit" });
} catch (error) { process.exit(error.status || 1); }
