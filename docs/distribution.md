# Local distribution

The npm package and executable are `feishu-cli-next`. This derivative never
installs or publishes `@larksuite/cli`, and retains the upstream MIT license.
The npm package is private and publishing workflows are disabled pending P4.
The source repository is public at `Zhujingxi/feishu-cli-next`. npm name ownership,
release credentials, trusted publishing, and signing/notarization identities
must be accepted before enabling the reference-equivalent automated public
release channel. Apple signing is optional
for source builds and unsigned local archives.
The native `update` command currently reports an unconfigured release channel.
CI and Next offline correctness run on pull requests, main pushes, and manual
runs on Linux only. Automated CI uses offline and dry-run tests and requires no
Feishu app credentials or live tenant. Live API tests remain available for local
runs with explicitly configured test credentials; CI does not verify live API
behavior or macOS/Windows runtime behavior.
The workflow source gate checks every workflow for Feishu credential inputs and
live API entrypoints; the dry-run job uses literal placeholder app credentials.
Unrelated GitHub, model-review, coverage and release-signing credentials retain
their separate uses.

Build the current host with canonical `make build` (or `./build.sh`), using Go
1.26.10+ and the committed embedded API catalog. Neither Python nor a metadata
fetch is needed for this build. Python 3 is required for the packaging and prefix
installation tools. Go dependencies must already be cached for offline builds.
`python3 scripts/next_package.py --output .tmp/release` builds all seven OS/architecture pairs, each in normal and `authsidecar` variants,
and writes SHA-256 checksums plus a local npm wrapper. An existing output directory
is refused. `--target linux/amd64 --variant normal` selects a smaller build.

Archives contain the executable, license, changelog, README and this document.
Windows archives are ZIP; other platforms use tar.gz. Both variants retain the
same executable name and differ in the archive suffix (`-authsidecar`). Use the
sidecar archive explicitly for host-managed credential isolation. npm installs
the normal variant. The supported pairs are macOS amd64/arm64, Linux
amd64/arm64/riscv64, and Windows amd64/arm64; riscv64 is Linux-only.

For local npm setup, first pack the generated wrapper:

```sh
npm pack --ignore-scripts ./.tmp/release/npm --pack-destination .tmp/release
FEISHU_CLI_NEXT_ARTIFACT_DIR="$PWD/.tmp/release" npm install --offline --prefix ./cli-install ./.tmp/release/feishu-cli-next-0.1.0-beta.1.tgz
./cli-install/node_modules/.bin/feishu-cli-next --version
```

The installer checks the archive against checksums inside the npm package before
extracting it; supplying an artifact directory cannot override those checksums.
Public downloads are unavailable until the release channel is configured. npm
owns package binary install/upgrade/uninstall. Upgrade by installing the new
local tarball using the same prefix. `npm uninstall --offline --prefix ./cli-install
feishu-cli-next` removes that package; configuration and project skills are
separate. Avoid npm `--force`, which can override npm's existing-link checks.

For a direct archive installation, verify `checksums.txt`, extract into a fresh
version directory, and invoke that executable. Select the new version directory
to upgrade; remove only the directory you created to uninstall. Existing binaries
need not be overwritten. Alternatively use `make install PREFIX=/your/prefix`, `make update-install
PREFIX=/your/prefix`, and `make uninstall PREFIX=/your/prefix`. These targets
manage only the executable and its adjacent `.feishu-cli-next.sha256` ownership
file (with `.exe` on Windows). They refuse unrelated, symlinked or edited files.
Install repeats are no-ops; changing a managed version requires update-install.
A local exclusive lock serializes operations. A crash can leave that exact lock
or an incomplete binary/checksum pair: inspect both files before recovery and
remove the lock only when no installer is running. Incomplete or edited states
fail closed. This checksum is local ownership detection, not a signature or
protection against another process concurrently replacing directory components.

`feishu-cli-next guide install --project /existing/project` installs the one
embedded entry skill. Update with `guide update` and remove with `guide uninstall`
using the same project. The npm wrapper's `install --project` maps to `guide install`.
Installation does not configure apps or authorize accounts. Continue explicitly
with `config init --new` and `auth login` when required; their native language,
profile and identity options remain available. Existing global upstream skills
remain usable through `skills list/read`, and are never rewritten by setup.

The [skill lifecycle contract](contracts/skill.md) defines collisions, ownership,
unchanged content, symlinks and cleanup. npm uninstall deliberately preserves
project skills, profiles, configuration and unrelated files. Build success for a
foreign architecture does not establish runtime behavior. Acceptance requires
applicable OS and lifecycle evidence, not physical hardware per architecture.
Both macOS amd64 variants passed startup and project guide smoke under Rosetta on
macOS arm64. Both Linux arm64 archives also passed project guide lifecycle checks
in an isolated Linux container, and both Linux amd64 archives passed the same
checks under existing QEMU translation. Both Linux riscv64 archives passed using
a test-only QEMU package in a disposable container. Windows execution remains
unverified; broader Linux amd64/riscv64 suites were not run. The retained
automated public release channel also requires macOS signing/notarization
verification. See the [packaging evidence](sections/packaging.md) for exact scope.

## Before enabling public distribution

`npm run release:check` checks stable/beta version consistency only; success does
not certify release readiness. The publication check additionally fails closed
while the npm package is private or the installer's artifact repository is unset
or differs from `Zhujingxi/feishu-cli-next`:

```sh
npm run release:check -- --publish --tag v0.1.0-beta.1
```

The tag helper and dormant release workflow both use this publication check. A
public source repository alone does not establish npm ownership, artifact
authenticity, or a usable update channel.

Before changing the private-package, release-workflow, preview-workflow, or
GoReleaser guards, retain reviewable evidence for these prerequisites:

- Finish the outstanding domain, live, platform and comparative acceptance in
  [outstanding release evidence](#outstanding-release-evidence), tied to the
  candidate revision. Offline tests and cross-compiles do not close live or
  OS-specific checks.
- Verify ownership of the `feishu-cli-next` npm name and configure npm trusted
  publishing for this repository and workflow. Configure protected release tags
  and the `npm-production` environment; verify its signing inputs and required
  approval policy.
- Configure the installer and the existing updater owners for the same Next
  package/repository, following the identity checklist in
  [packaging.md](sections/packaging.md#reference-applicability-of-remaining-release-checks).
  The updater must never install the official upstream product or rewrite
  unrelated global skills.
- Rehearse a checksummed candidate on both macOS architectures, including signing
  and notarization verification, plus isolated remote install, upgrade, failed
  upgrade, and removal checks against the accepted channel. Retain package
  integrity/provenance and artifact-to-candidate verification.
- Run the required build, unit, vet, format, lint, license, vulnerability,
  secret and quality checks on the exact candidate; resolve blocking findings
  rather than waiving them for publication.

The publication preflight enforces wiring prerequisites. The evidence above
remains a release review responsibility; passing the preflight is not a live
acceptance result. Until those prerequisites are met, local installation and
explicit local upgrades remain the available paths.

## Outstanding release evidence

These gates were still open at the v1.0.98 sync. Offline tests, cross-compiles
and version validation do not close them.

- **Private live workflows.** Drive, explicit-XML Docs, Sheets (after its formula
  correction) and Markdown lifecycles have passed private runs with a verified
  account and private folder. Base's supported steps passed, but its selected
  suite retains a failed negative assertion. Other section live scenarios and
  broader type and concurrency guarantees remain open.
- **Contact bot tenant binding.** The requested `search:bot` grant and the selected
  read suite passed. Separate bot tenant verification still needs the
  `tenant:tenant:readonly` app scope, which was not requested.
- **OS-specific runtime.** Linux arm64 runtime suites and the platform comparison
  pass, and delivered archives pass lifecycle checks on arm64 and under
  amd64/riscv64 translation. Broader amd64/riscv64 runtime suites have not run.
  Darwin/amd64 smoke checks ran under Rosetta. Windows checks and native reference
  capture are deferred by the user's 2026-09-08 instruction, and no Windows runner
  or VM was available. Cross-built archives are not runtime verification.
- **Remote release ownership.** The public source repository is
  `Zhujingxi/feishu-cli-next`. npm package ownership, a trusted artifact and update
  channel, the updater identity integration, and remote install and update checks
  remain unverified.
- **Comparative evidence.** Development comparisons keep all costs. The
  native-fidelity audit withholds same-fidelity efficiency claims for Docs, Sheets
  and Base-size tasks, and small Base update fidelity is unvalidated. Changes made
  since then have no model measurements. Held-out tasks are reserved for the final
  release gate.

Offline source-review approvals in
[the aggregate](../evidence/reference/p3-effective-review-current.json) grant no
live or release acceptance.
