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
the immutable P3 registration checkpoint preserves all 938 reference command
nodes and adds 13 nodes, including three data-transfer shortcuts. Source changes
remain explicitly tracked for semantic review and subsequent fix reconciliation.

[P1 shared interfaces are ready for P2](sections/foundation.md): the full
foundation unit/process suite, plugin and sidecar checks, static checks and
quality gate pass, with explicit skips and revision boundaries. Read-only live
checks verified the current user; private content fixtures are not yet verified.
The [foundation model report](../evaluation/evidence/foundation/REPORT.md)
records 18/18 compatible trials, actual token usage, a task-level regression,
earlier unknown-cost failures and trace limitations. Held-out trials have not run.

P2 developed every independent inventory section concurrently under
[section ownership](sections/OWNERSHIP.md). All original domain offline suites
have recorded passes; their reports retain explicit live blockers and conditional
skips. The additional [generated business section](sections/business.md) covers
17 methods found outside the domain assignments. Its complete offline boundary
passes after fixing later-page errors that previously became successful partial
aggregates. Commit `6056d73` also rejects missing/repeated continuation tokens;
its eight unit regressions fail against the prior implementation, and public raw
API JSON/NDJSON process tests preserve typed failure evidence.

Shared follow-ups verify normal-account artifact identity (`eafd052`), bind
receipts to each request's actual user/bot identity (`f1bb815`), preserve uncertain
batch outcomes, and journal both Apps storage PUT paths (`7550245`). Focused
independent review closed the earlier artifact and IM/VC evidence findings.
These changes require affected-suite checks at the next integration boundary.

The [Contact comparison](../evaluation/evidence/contact/REPORT.md) records nine
successful trials and complete usage accounting, but a regression: Next used
66,811 tokens per success versus 59,485 for the official arm and 41,233 for its
concise-guidance arm. The small sample and sanitized queries limit causal claims.
Commit `ced4030` makes public flag descriptions searchable and clarifies literal
keyword discovery; it has no measured improvement claim yet. A frozen 81-trial
development batch across nine additional read-only sections
is running separately; remaining section adapters are undergoing controlled
verification. Its candidate is recorded independently of newer product fixes.
Held-out trials have not run. A coordinator file search exposed fragments of the
Base and Sheets held-out prompts and one size field; the evaluation owner must
record that exposure and arrange independent replacements before final testing.

[Live Contact evidence](sections/contact-live-evidence.json) verifies the current
account and the sequential user reads. The full test failed on bot search; a
single bounded diagnosis identified missing `search:bot` user scope. No permission
was granted and no content was written. This remains an unmet live prerequisite.

The [P3 Sheets-to-Base process suite](integration/sheets-base.md) passes all 11
test/subtest events without skips. Its actual 10,000-row transfer preserves
80,000 cells through 16 source pages and 50 write batches. Failure injection
exposed cross-batch duplicate IDs being accepted by the importer; `f080e43` and
`d91e695` now stop the operation with consistent unknown report and receipt
evidence. This is separate from Base's earlier import-only test and does not
establish live conversion behavior.

The [complete isolated race gate](sections/p3-unit-evidence.json) at `32183054`
passes 18,499 tests with no failures or races and 12 explained skips. Later fixes
have their own focused unit/process checks. Vet, formatting, dependency/license
checks, quality gate, source guards and pinned Go lint have passed at their
recorded development boundaries; final release verification still needs an
immutable candidate after remaining changes settle.

[Packaging evidence](sections/packaging-evidence.json) covers all 14 cross-built
platform/variant archives, local installation/update/removal, and actual Node 16
runtime tests. Native Linux/Windows/macOS Intel execution and a trusted remote
Next release channel remain unverified or unavailable. Local build evidence does
not establish public release readiness.

## Outstanding release evidence

P2 domain acceptance, verified private live fixtures, final source reconciliation,
remaining cross-section workflows, native platform and remote lifecycle checks,
full release checks and held-out evaluation remain open. The completed offline
10,000-row transfer and local packaging checks above are recorded milestones.
Blocked or unverified
checks remain outstanding; offline or scoped foundation success does not count
as full live parity or satisfy the complete goal.
