# Implementation ledger

The release criteria remain those in [PLAN.md](PLAN.md). Imported source is implementation reuse, not evidence of live parity. No efficiency claims have been established.

## Reference and baseline

- Reference: `../feishu-cli-review`, clean commit `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; kept unchanged.
- Source reuse: commit `0c1ac75`, full tracked source, tests, MIT notices, extension SDK, platform variants and installation tools.
- Upstream Go module paths remain stable for internal and public SDK source compatibility. New executable/distribution identity is separate.
- Build-time metadata: fetched in the target checkout; frozen inventory records the exact catalog and hashes. Remote runtime refresh is disabled during comparison.

## Section ownership and gates

| Package | Scope | Owner | Acceptance suite | Dependencies |
| --- | --- | --- | --- | --- |
| P0 inventory | Actual command registrations, generated catalog, schemas, events, extension and build contracts | inventory agent | inventory unit and process reconciliation tests | pinned reference |
| P0 evaluation | Three-arm task loading, process runner/verifier, honest token accounting and reports | evaluation agent | unit and synthetic process E2E | pinned reference |
| P0 test isolation | Clean environment, isolated state, selectable suites and machine-readable results | offline runner agent | runner unit and CLI process E2E | imported source |
| P0 baseline repair | Remove live DNS dependence from mock download tests | baseline test agent | affected test packages | imported tests |
| P1 foundation | Authentication/configuration reuse; shared discovery, output, artifacts, binding, receipts and skill framework | coordinator, shared-module owners assigned after P0 | complete foundation unit and process E2E; inherited auth/config/host tests; development evaluation | P0 |
| P2 domains | Every inventory section, using shared runtime | assigned after P1 passes | section unit, integration, executable offline and authorized live E2E; development evaluation | P1 |
| P3 integration | Full capability reconciliation, cross-domain typed transfer and recovery | coordinator | cross-domain process tests, 10,000-row preservation, inventory reconciliation | participating P2 sections |
| P4 release | Platform/install packaging, CI, held-out evaluation and release evidence | coordinator | all required suites, packaging matrix and held-out trials | P3 and all sections |

## Current evidence

Canonical imported build succeeds with the generated catalog. Initial race-enabled upstream unit suite passed shared/authentication packages, but download tests in IM and Minutes depend on external DNS resolving `example.com` as public. These fixture failures are being repaired without changing URL safety checks. Test evidence is retained in `.tmp/next-evidence/` during development and summarized in committed section reports.

## Outstanding release evidence

Real comparative agent trials require an external agent runner with actual model usage and independent backend verification. Controlled harness tests do not provide comparative product measurements. Live tests require verified current-account identity and private fixture visibility, following PLAN.md authorization boundaries. Offline success never counts as live verification. Required blocked/unverified checks keep their sections open.
