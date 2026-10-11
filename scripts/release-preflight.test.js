// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

const assert = require("node:assert/strict");
const { describe, it } = require("node:test");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const { spawnSync } = require("node:child_process");

const { validateReleasePreflight, validateReleaseSourcePolicy, validateReleasePublication } = require("./release-preflight");

function metadata(version = "1.2.3") {
  return {
    packageJson: { version },
    packageLockJson: {
      version,
      packages: { "": { version } },
    },
  };
}

function assertRejected(result) {
  assert.equal(result.ok, false);
  assert.equal(result.error.type, "release_preflight");
  assert.equal(typeof result.error.message, "string");
}

describe("validateReleasePreflight", () => {
  it("accepts matching stable package, lock, and tag versions", () => {
    const { packageJson, packageLockJson } = metadata();

    assert.deepEqual(
      validateReleasePreflight(packageJson, packageLockJson, "v1.2.3"),
      {
        ok: true,
        data: {
          packageVersion: "1.2.3",
          lockVersion: "1.2.3",
          lockRootVersion: "1.2.3",
          tagVersion: "1.2.3",
          releaseChannel: "stable",
        },
      },
    );
  });

  it("accepts matching beta package, lock, and tag versions", () => {
    const { packageJson, packageLockJson } = metadata("1.2.3-beta.4");

    assert.deepEqual(
      validateReleasePreflight(packageJson, packageLockJson, "v1.2.3-beta.4"),
      {
        ok: true,
        data: {
          packageVersion: "1.2.3-beta.4",
          lockVersion: "1.2.3-beta.4",
          lockRootVersion: "1.2.3-beta.4",
          tagVersion: "1.2.3-beta.4",
          releaseChannel: "beta",
        },
      },
    );
  });

  it("derives the release channel when no tag is provided", () => {
    const { packageJson, packageLockJson } = metadata("1.2.3-beta.0");

    assert.deepEqual(
      validateReleasePreflight(packageJson, packageLockJson),
      {
        ok: true,
        data: {
          packageVersion: "1.2.3-beta.0",
          lockVersion: "1.2.3-beta.0",
          lockRootVersion: "1.2.3-beta.0",
          tagVersion: null,
          releaseChannel: "beta",
        },
      },
    );
  });

  it("rejects unsupported or invalid package versions with an actionable hint", () => {
    for (const version of [
      "1.2.3-alpha.1",
      "1.2.3-rc.1",
      "1.2.3-beta",
      "1.2.3-beta.01",
      "1.2.3+build.1",
      "1.2.3-beta.1+build.1",
      "01.2.3",
      "1.02.3",
      "1.2.03",
    ]) {
      const { packageJson, packageLockJson } = metadata(version);
      const result = validateReleasePreflight(packageJson, packageLockJson);

      assertRejected(result);
      assert.match(result.error.hint, /stable X\.Y\.Z or beta X\.Y\.Z-beta\.N/i);
    }
  });

  it("rejects inconsistent package metadata", () => {
    const topLevelMismatch = metadata();
    topLevelMismatch.packageLockJson.version = "1.2.4";
    const rootMismatch = metadata();
    rootMismatch.packageLockJson.packages[""].version = "1.2.4";
    const channelMismatch = metadata("1.2.3-beta.1");
    channelMismatch.packageLockJson.version = "1.2.3";

    for (const { packageJson, packageLockJson } of [
      topLevelMismatch,
      rootMismatch,
      channelMismatch,
    ]) {
      assertRejected(validateReleasePreflight(packageJson, packageLockJson));
    }
  });

  it("rejects an invalid or mismatched release tag", () => {
    const { packageJson, packageLockJson } = metadata();

    for (const tag of [
      "1.2.3",
      "v1.2.3-alpha.1",
      "v1.2.3-beta.01",
      "v1.2.3+build.1",
      "v1.2.3-beta.1",
      "v1.2.4",
    ]) {
      assertRejected(validateReleasePreflight(packageJson, packageLockJson, tag));
    }
  });

  it("rejects a beta tag that does not match beta package metadata", () => {
    const { packageJson, packageLockJson } = metadata("1.2.3-beta.2");

    for (const tag of ["v1.2.3-beta.1", "v1.2.3"]) {
      assertRejected(validateReleasePreflight(packageJson, packageLockJson, tag));
    }
  });
});

describe("validateReleaseSourcePolicy", () => {
  const mainSha = "a".repeat(40);
  const betaSha = "b".repeat(40);

  it("accepts a stable release commit contained in main", () => {
    assert.deepEqual(
      validateReleaseSourcePolicy("stable", betaSha, mainSha, true),
      { ok: true, data: { warning: null } },
    );
  });

  it("rejects a stable release commit outside main", () => {
    const result = validateReleaseSourcePolicy("stable", betaSha, mainSha, false);

    assertRejected(result);
    assert.match(result.error.message, /contained in origin\/main/);
  });

  it("accepts a beta release from a separate branch without a warning", () => {
    assert.deepEqual(
      validateReleaseSourcePolicy("beta", betaSha, mainSha, false),
      { ok: true, data: { warning: null } },
    );
  });

  it("warns but accepts a beta release tag pointing to a commit already in main", () => {
    const result = validateReleaseSourcePolicy("beta", betaSha, mainSha, true);

    assert.equal(result.ok, true);
    assert.match(result.data.warning, /unintended beta version was merged into main/);
  });
});

describe("publication wiring preflight", () => {
  const packageJson = { name: "feishu-cli-next", private: false };
  const repository = "Zhujingxi/feishu-cli-next";

  it("requires an explicit public-package decision", () => {
    for (const privateValue of [true, undefined, null, "false"]) {
      assertRejected(validateReleasePublication({ ...packageJson, private: privateValue }, repository));
    }
  });

  it("rejects an absent or upstream installer channel even after private is removed", () => {
    for (const artifactRepository of [null, undefined, "", "larksuite/cli"]) {
      assertRejected(validateReleasePublication(packageJson, artifactRepository));
    }
    assertRejected(validateReleasePublication({ ...packageJson, name: "@larksuite/cli" }, repository));
  });

  it("accepts only the matching Next package and artifact identity", () => {
    assert.equal(validateReleasePublication(packageJson, repository).ok, true);
  });

  it("keeps version checks available while publication rejects an unconfigured installer", (t) => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), "next-release-preflight-"));
    t.after(() => fs.rmSync(root, { recursive: true, force: true }));
    fs.mkdirSync(path.join(root, "scripts"));
    fs.copyFileSync(path.join(__dirname, "release-preflight.js"), path.join(root, "scripts/release-preflight.js"));
    fs.writeFileSync(path.join(root, "scripts/install.js"), "module.exports = { REPO: null };\n");
    const fixture = metadata();
    fs.writeFileSync(path.join(root, "package.json"), JSON.stringify({ ...fixture.packageJson, ...packageJson }));
    fs.writeFileSync(path.join(root, "package-lock.json"), JSON.stringify(fixture.packageLockJson));
    const script = path.join(root, "scripts/release-preflight.js");
    const versionCheck = spawnSync(process.execPath, [script, "--tag", "v1.2.3"], { encoding: "utf8" });
    assert.equal(versionCheck.status, 0, versionCheck.stderr);
    assert.equal(JSON.parse(versionCheck.stdout).ok, true);
    const publicationCheck = spawnSync(process.execPath, [script, "--publish", "--tag", "v1.2.3"], { encoding: "utf8" });
    assert.equal(publicationCheck.status, 1);
    assert.equal(publicationCheck.stdout, "");
    const failure = JSON.parse(publicationCheck.stderr);
    assertRejected(failure);
    assert.equal(failure.error.observed.artifactRepository, null);
    if (process.platform !== "win32") {
      fs.copyFileSync(path.join(__dirname, "tag-release.sh"), path.join(root, "scripts/tag-release.sh"));
      const sentinel = path.join(root, "git-called");
      fs.mkdirSync(path.join(root, "fake-bin"));
      fs.writeFileSync(path.join(root, "fake-bin/git"), '#!/bin/sh\nprintf invoked > "$GIT_SENTINEL"\nexit 1\n', { mode: 0o755 });
      const tagCheck = spawnSync("bash", [path.join(root, "scripts/tag-release.sh")], {
        encoding: "utf8",
        env: { ...process.env, GIT_SENTINEL: sentinel, PATH: `${path.join(root, "fake-bin")}${path.delimiter}${process.env.PATH}` },
      });
      assert.equal(tagCheck.status, 1);
      assertRejected(JSON.parse(tagCheck.stderr));
      assert.equal(fs.existsSync(sentinel), false, "publication must reject before any git/network operation");
    }
  });
});
