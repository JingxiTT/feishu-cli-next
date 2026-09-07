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
| Windows ZIP .NET/Expand-Archive/tar fallback and interrupted-update `.old` recovery | Retained installer and launcher paths under next name; requires native Windows execution, not counted from cross-compilation |
| npm setup wizard: install/upgrade binary, skills, config, auth | npm manages package install/upgrade; wrapper `install --project` maps to embedded `guide install`; config init and auth login remain explicit native commands. Setup never rewrites profiles or global upstream skills; no registry-dependent prompts dependency |
| Project skill install/update/uninstall and collision preservation | Existing guide contract, exercised via packaged executable; repeated install/update, edited content refusal, npm uninstall preserving project skill, explicit guide uninstall preserving unrelated skills |
| Prefix binary install/uninstall | `next_binary.py` and Makefile install/update-install/uninstall; ownership SHA, exclusive install, explicit update, edited/unmanaged/symlink collision refusal, executable mode, idempotency and unrelated-file preservation |
| Stable/beta version/tag validation, immutable candidate/npm publish policy | Existing preflight and publish-policy helpers/tests retained; tagging exits before git/network while private, release and preview jobs disabled, GoReleaser publication disabled |
| macOS signing/notarization, release retry and provenance workflow | Retained next-identity configuration behind publication guard; acceptance requires owned next channel, signing credentials, native verification and P4 approval |
| Host/plugin distribution contracts | Build variants preserve package assembly; public extension behavior and credential isolation are owned by Extensions and Foundation; no external SDK changes here |

The npm upgrade fixture changes the package version with intentionally identical
binary bytes, then verifies installed wrapper metadata and guide no-op behavior.
Changed embedded skill-version updates are covered by the foundation lifecycle
suite. The prefix lifecycle uses different executable fixture bytes to verify a
real replacement and refuses changes without explicit update.

## Evidence and remaining gates

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
- [Sanitized evidence](packaging-evidence.json) records checksums, test counts
  and limitations. The matrix ran against the active shared P2 checkout; it is
  not an immutable release candidate. Final documentation was repacked without
  changing compiled binary bytes after the full matrix completed.

Native Windows runtime (ZIP fallbacks and recovery), native Linux runtime,
macOS Intel runtime, signing/notarization, remote-channel install/update and
Node 16 runtime, and comparative/held-out evaluation remain unverified. This section is locally
implemented and testable; these required release checks keep final acceptance
open. Public repository/npm ownership, matching update/download URLs, publisher
credentials, Apple identity and release approval are prerequisites. No publishing,
external push, global installation or real credential use was performed.

The binary/checksum ownership pair is fail-closed after interruption; only the
binary replacement is atomic. It does not promise a two-file transaction,
signing authenticity, or safety against an adversarial directory replacement.
Distribution usage and recovery live in [distribution.md](../distribution.md).
