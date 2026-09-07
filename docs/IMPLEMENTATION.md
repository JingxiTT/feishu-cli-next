# Implementation ledger

The release criteria remain those in [PLAN.md](PLAN.md). Imported source is implementation reuse, not evidence of live parity. Scoped foundation measurements are available; no full-product efficiency or release claim is established.

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

P0 inventory and harness checks pass. The reference remains clean and unchanged;
the actual next tree preserves all 938 reference command nodes and adds ten
foundation nodes. Source changes remain explicitly tracked for final review.

[P1 shared interfaces are ready for P2](sections/foundation.md): the full
foundation unit/process suite, plugin and sidecar checks, static checks and
quality gate pass, with explicit skips and revision boundaries. Read-only live
checks verified the current user; private content fixtures are not yet verified.
The [foundation model report](../evaluation/evidence/foundation/REPORT.md)
records 18/18 compatible trials, actual token usage, a task-level regression,
earlier unknown-cost failures and trace limitations. Held-out tasks are untouched.

P2 assigns every independent inventory section concurrently under
[section ownership](sections/OWNERSHIP.md), preserving imported implementations
and adding complete behavioral evidence and useful domain improvements.

## Outstanding release evidence

P2 domain acceptance, verified private live fixtures, final source reconciliation,
10,000-row typed transfer, cross-section workflows, platform/lifecycle packaging,
full release checks and held-out evaluation remain open. Blocked or unverified
checks remain outstanding; offline or scoped foundation success does not count
as full live parity or satisfy the complete goal.
