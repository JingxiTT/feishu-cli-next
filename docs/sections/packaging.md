# Packaging and host lifecycle section

Reference: unchanged sibling `feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`. Ownership: package metadata,
build/release/install scripts and their tests, distribution documentation, and
`tests/next_sections/packaging`. The coordinator authorized only lifecycle
Makefile hunks and release/preview workflow identity and publication guards.
The native updater, authentication/profile runtime, project-skill implementation,
and general CI remain their respective shared owners.

The frozen non-command entry is `coverage.json#/non_command_sections/packaging`,
pointing to `sources.json#/conditions`. All conditional native files are covered
by the seven-pair compile matrix: keychain/config downgrade, event signals/local
bus/startup, binding audit, locks, risk-control OS model, self-update, local FileIO
hardlinks/path validation, and normal/authsidecar assembly. Their runtime tests
remain with Foundation, Events and Extensions. The two server-only tags,
`authsidecar_demo` and `authsidecar_multi_tenant_demo`, belong to Extensions'
demo build/test coverage and are not native CLI release variants.

## Complete mapping and acceptance scenarios

| Reference capability | Next mapping and test owner |
| --- | --- |
| Native source build, embedded metadata and version/date | `build.sh`/Makefile produce next identity using frozen catalog; local archive builder compiles the whole package, preserving build-tag init files rather than only main.go |
| macOS amd64/arm64, Linux amd64/arm64/riscv64, Windows amd64/arm64 | Same seven pairs; impossible macOS/Windows riscv64 pairs explicitly excluded from GoReleaser. `next_package.py` builds all pairs; deterministic archive fixture test covers all formats |
| Normal and opt-in authsidecar builds | Both variants in local archive builder; tagged host integrations retained. npm wraps normal; sidecar hosts select explicit `-authsidecar` archive |
| tar.gz/ZIP, version names, SHA-256 assets | Next-named stable/beta archive names, deterministic archive metadata, normal and sidecar checksums, embedded npm checksum list; checksum mismatch must preserve installed executable |
| npm install, lazy npx binary installation, platform mapping | Private `feishu-cli-next` wrapper; no official namespace overwrite. Explicit local artifact directory reads archive bytes, never checksum authority. Unsupported OS/arch pairs reject |
| GitHub binary source, npm/corporate mirror fallback, curl Windows revocation compatibility | Resolution helpers retained under next name, regression tests retained; actual network download channel is unconfigured and fails explicitly pending P4 |
| Windows ZIP .NET/Expand-Archive/tar fallback and interrupted-update `.old` recovery | Retained installer and launcher paths under next name; requires execution on Windows, not counted from cross-compilation |
| npm setup wizard: install/upgrade binary, skills, config, auth | npm manages package install/upgrade; wrapper `install --project` maps to embedded `guide install`; config init and auth login remain explicit native commands. Setup never rewrites profiles or global upstream skills; no registry-dependent prompts dependency |
| Project skill install/update/uninstall and collision preservation | Existing guide contract, exercised via packaged executable; repeated install/update, edited content refusal, npm uninstall preserving project skill, explicit guide uninstall preserving unrelated skills |
| Prefix binary install/uninstall | `next_binary.py` and Makefile install/update-install/uninstall; ownership SHA, exclusive install, explicit update, edited/unmanaged/symlink collision refusal, executable mode, idempotency and unrelated-file preservation |
| Stable/beta version/tag validation, immutable candidate/npm publish policy | Existing preflight and publish-policy helpers/tests retained; tagging exits before git/network while private, release and preview jobs disabled, GoReleaser publication disabled |
| macOS signing/notarization, release retry and provenance workflow | Retained next-identity configuration behind publication guard; acceptance requires owned next channel, signing credentials, candidate verification on macOS and P4 approval |
| Host/plugin distribution contracts | Build variants preserve package assembly; public extension behavior and credential isolation are owned by Extensions and Foundation; no external SDK changes here |

The npm upgrade fixture changes the package version with intentionally identical
binary bytes, then verifies installed wrapper metadata and guide no-op behavior.
Changed embedded skill-version updates are covered by the foundation lifecycle
suite. The prefix lifecycle uses different executable fixture bytes to verify a
real replacement and refuses changes without explicit update.

## Evidence and remaining gates

The initial P2 results below are historical. The immutable refresh and current
acceptance scope follow them; later evidence does not rewrite earlier runs.

- `make script-test`: pass, including release/preview publication guard assertions,
  inherited shell workflow tests and all 166 JavaScript tests, zero skips.
- `python3 scripts/next_test.py --section packaging --suite e2e --binary ./feishu-cli-next --report .tmp/packaging-e2e.json`:
  pass. Go process entry runs all Python packaging tests in a credential-isolated
  HOME/config/npm cache with disabled external proxy access and npm offline mode.
- The suite covers fourteen deterministic fixture archive combinations, real
  npm pack/install/upgrade/uninstall, native executable launch and guide lifecycle,
  checksum tampering, missing release channel, symlinks, package identity/tagging
  guard, and prefix ownership lifecycle. No live Feishu resources are required.
- Full matrix: fourteen successful cross-compiles (seven pairs, normal and
  authsidecar). All fourteen SHA-256 entries independently rechecked. Both
  macOS arm64 variants passed version and embedded-guide execution; a configured
  sidecar endpoint made the normal binary fail closed while the sidecar binary
  accepted its own build mode. Linux, Windows and macOS amd64 were not executed.
- The current final builder separately rebuilt macOS arm64 normal and produced
  a real npm tarball; required shipped files, install, version, project-skill
  setup and uninstall preservation passed. Seven final Python scenarios passed,
  including incomplete ownership, malformed manifest and existing lock refusal.
- Actual Node 16 compatibility: downloaded official Node 16.20.2/npm 8.19.4
  for macOS arm64, verified SHA-256 before execution, and ran all seven packaging
  Python scenarios with that runtime first in subprocess PATH. All passed with
  zero skips; installer/release JavaScript tests also passed in all three files.
  No global runtime installation was performed. The official archive and checksum
  URLs, digest, versions and exact command are recorded in the evidence JSON.
- [Sanitized evidence](packaging-evidence.json) records checksums, test counts
  and limitations. The matrix ran against the active shared P2 checkout; it is
  not an immutable release candidate. Final documentation was repacked without
  changing compiled binary bytes after the full matrix completed.

The final artifact refresh uses immutable product `8ce0688b303eb646e1b79a329002dddcb6403be4`:
all fourteen normal/authsidecar archives compiled and passed checksum/content
verification. Both macOS arm64 variants passed startup/version, exact guide,
build-mode distinction and isolated project guide installation. The generated
npm package passed local install, launch, guide setup and uninstall preservation.
Both macOS amd64 variants subsequently passed the same binary/project guide smoke
under Rosetta on macOS arm64. This is actual translated amd64 execution; the smoke
did not exercise amd64 npm/prefix collision, upgrade or uninstall scenarios.
Existing shared lifecycle evidence must be mapped to those scenarios when
assessing coverage; physical Intel hardware is not an additional acceptance gate.

Linux arm64 now has actual source-built unit/race and packaging-suite evidence
in [linux-runtime-evidence.json](linux-runtime-evidence.json). A separate bounded
follow-up executed both delivered immutable Linux arm64 archives in a new
resource-capped Linux aarch64 container inside the existing VM. Archive/binary
checksums, startup/version, exact guide, normal/sidecar distinction, project
install/repeat, update/no-op, edited-file refusal, uninstall/repeat and unrelated
project/home/config preservation all passed. The changed update used an intact
synthetic older managed guide; it does not claim a historical binary migration.
The container used UID 10001, no mounts, network none, dropped capabilities and
no-new-privileges. The existing npm/prefix suite was not repeated. Exact hashes,
limits, commands and local evidence paths are recorded in the packaging JSON.

The completed source export was moved outside the repository after build/smoke
to prevent source guards discovering a second source tree. All 3,663 tracked file
hashes were verified before and after the move, and the full tree's bytes, modes
and links were unchanged. The source archive, fourteen binary archives, npm
package and logs remain at their original `.tmp/packaging-release-8ce0688` paths.
The exact old/new source paths and unchanged archive hash are recorded in
`.tmp/source-location.json` and [packaging-evidence.json](packaging-evidence.json).
Historical logs may refer to the old source path.

Linux amd64/riscv64 and Windows runtime behavior and the public-release/channel
checks below remain unverified. This section is locally implemented and testable; final
release acceptance remains open. No publishing, external push, global installation
or real credential use was performed. Comparative/held-out evaluation is recorded
by its own owner and is not inferred from packaging tests.

## Reference applicability of remaining release checks

The reference's `.goreleaser.yml` makes signing/notarization conditional on
`MACOS_SIGN_P12`. Neither plain source builds nor unsigned local archives require
Apple credentials. Its automated public release workflow is stricter:
`.github/workflows/release.yml` at frozen `7fd6ef3` requires six Apple inputs in
`Prepare Apple notarization key`, verifies Developer ID/team/hardened runtime,
timestamp and notarization on both macOS architectures, and makes public
publication depend on that verification. Next preserves this public-channel
capability and requirement; missing Apple credentials do not invalidate the
completed local packaging tests.

PLAN requires applicable platform, architecture, build-variant and installation
coverage; it does not require physical hardware for each architecture. The
reference release workflow uses `macos-15-intel` and `macos-15` runners for its
signed-candidate checks. Those checks remain public-release requirements; the
runner labels do not create a separate physical Intel hardware gate. Appropriate
OS runners, VMs or translation environments can provide behavioral evidence when
the execution mode and checked scenarios are stated accurately.

Cross-compilation verifies conditional source selection, not OS behavior. Linux
arm64 source-built runtime and delivered-archive lifecycle evidence are now
recorded above; Linux amd64/riscv64 execution remains unverified. Windows named pipes, LockFileEx, path semantics,
extraction fallback and `.old` recovery/rollback still need Windows execution
evidence. The remaining target/OS gaps cannot be closed by macOS Rosetta
or inferred from Linux arm64 execution. The successful Rosetta smoke establishes only its listed
macOS amd64 checks; signing/notarization and unexecuted lifecycle scenarios remain
separate coverage questions rather than hardware requirements.

There is no configured trusted remote next update source: the npm package is
private and `scripts/install.js` has `REPO = null`. Re-enabling the inherited
`update` command alone would still query/install the official product. The small
future integration, once an owned package and release URL can be verified, must
reuse the existing owners while changing the following bound product identities:

- `internal/update/update.go` registry URL and `internal/selfupdate/updater.go`
  npm/pnpm package name, package-manager invocation and binary lookup.
- `VerifyBinary` must read the actual next version token; the inherited final-token
  parser cannot parse next's trailing reference annotation.
- `cmd/update/update.go` release/changelog URLs, help and recovery text, and
  upstream global skills synchronization must be mapped to explicit project guide
  lifecycle without changing unrelated skills/profiles.
- `scripts/install.js` must use the same accepted next artifact repository and
  package-embedded checksums, followed by isolated upgrade/failure/verification
  tests before the root command's unconfigured-channel guard can be removed.

No new remote updater or channel configuration mechanism is introduced without
that source. The existing local npm upgrade and `make update-install` are tested
installation paths; they do not establish a remote automatic-update channel.

The binary/checksum ownership pair is fail-closed after interruption; only the
binary replacement is atomic. It does not promise a two-file transaction,
signing authenticity, or safety against an adversarial directory replacement.
Distribution usage and recovery live in [distribution.md](../distribution.md).
