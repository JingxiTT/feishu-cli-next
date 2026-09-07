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
Subsequent fixes preserve HTTP failures during pagination (`94e4afc`), exclude
trusted OAuth acquisition from content receipts (`a2f2626`), and keep local
command discovery free of identity requests. Three exact native Mail POST reads
now preserve read/artifact behavior (`1058824`) while raw and neighboring writes
retain conservative mutation handling. Independent source reviews and affected
section checks record their specific revision boundaries.

Drive import now reconciles a pending receipt only from an exact, successful
native task response for the captured invocation (`032b922`, `68126ee`).
Contradictory, redirected, non-200 or unpersisted completion evidence cannot
authorize the dependent bot grant. The independent
[Drive receipt review](reference/reviews/p3-drive-import-receipt-review.json)
closed all three findings. Full affected suites recorded 1,748 Drive and 1,903
Sheets pass events; seven Sheets live blockers remain. The fresh combined gate
at immutable `8d92e52` now covers these changes, with the results below.

The [Contact comparison](../evaluation/evidence/contact/REPORT.md) records nine
successful trials and complete usage accounting, but a regression: Next used
66,811 tokens per success versus 59,485 for the official arm and 41,233 for its
concise-guidance arm. The small sample and sanitized queries limit causal claims.
Commit `ced4030` makes public flag descriptions searchable and clarifies literal
keyword discovery; it has no measured improvement claim yet. The frozen
[nine-section comparison](../evaluation/evidence/read-312af93/REPORT.md)
completed 75/81 verified successes with 8,419,135 provider tokens, retaining all
failed costs. Among its eight fully successful sections, Next used fewer tokens
than the official arm in seven, but fewer than concise official guidance only in
IM. A separately versioned
[corrected Wiki comparison](../evaluation/evidence/wiki-native-74/REPORT.md)
passed 9/9 after a demonstrated fixture gap was fixed; historical failures and
diagnostic costs remain separate. The guide clarification in `43bdff2` has no
measured improvement claim. Remaining development comparisons and controlled
100/1,000/10,000-row scaling experiments use separately frozen candidates.
The [Calendar, Mail and OKR comparison](../evaluation/evidence/read-8d92e5/REPORT.md)
at `8d92e52` completed 27/27 successful trials and 3,647,948 provider tokens.
Each task has three fresh trials per arm; Next used fewer tokens per success
than both baselines on these workloads. Independent recount verifies the
balanced counts, input-plus-output totals, cached-input accounting and API sums.
This small development sample does not establish a full-product improvement.
The following scenario queue stopped on a provider-rejected Base answer schema,
before any Base CLI calls. Its nine starts retain unknown usage; the one completed
Docs trial and its cost remain separately recorded in the
[stopped-series report](../evaluation/evidence/scenarios-8d92e5/REPORT.md).
Held-out trials have not run. A coordinator search exposed fragments of the Base
and Sheets held-out prompts and one size field. The
[exposure record](../evaluation/evidence/held-out-exposure.json) quarantines those
examples for independent replacement after product freeze; full coordinator
blinding is not claimed.

[Live Contact evidence](sections/contact-live-evidence.json) verifies the current
account and the sequential user reads. The full test failed on bot search; a
single bounded diagnosis identified missing `search:bot` user scope. No permission
was granted and no content was written. This remains an unmet live prerequisite.

The [P3 Sheets-to-Base process suite](integration/sheets-base.md) passes all 11
initial test/subtest events without skips; the expanded suite at `1f1ce4e`
passes 18 events including real-export binding switches. Its actual 10,000-row transfer preserves
80,000 cells through 16 source pages and 50 write batches. Failure injection
exposed cross-batch duplicate IDs being accepted by the importer; `f080e43` and
`d91e695` now stop the operation with consistent unknown report and receipt
evidence. This is separate from Base's earlier import-only test and does not
establish live conversion behavior.

The [complete isolated race gate](sections/p3-unit-evidence.json) at `8d92e52`
passes 18,656 tests with no failures or races and 12 explained skips, including
the root package. Earlier failed checkpoints remain in the report. The
[static gate](sections/p3-static-evidence.json) passed nine checks and initially
failed pinned Go lint on four intentionally suppressed proof-assembly errors.
The comment-only clarification in `8ce0688` has focused regression coverage,
independent review and a passing full pinned-lint follow-up. The original failed
run remains visible; no runtime statements changed.

The [complete offline E2E checkpoint](sections/p3-e2e-evidence.json) at `8d92e52`
passes 3,525 test events with zero failures and 55 explicit skips. All 23 section
suites, the 18-event transfer suite, scripts and controlled evaluator ran.
Optional native evaluator checks have their own separately recorded execution;
the default harness skips are not counted as passes.

[Packaging evidence](sections/packaging-evidence.json) covers all 14 cross-built
platform/variant archives, local installation/update/removal, and actual Node 16
runtime tests. Native Linux/Windows/macOS Intel execution and a trusted remote
Next release channel remain unverified or unavailable. Local build evidence does
not establish public release readiness.

## Outstanding release evidence

P2 domain acceptance, verified private live fixtures,
remaining development comparisons and provider-token scaling measurements,
native platform and remote lifecycle checks, full release checks and held-out
evaluation remain open. The [source review aggregate](reference/p3-effective-review-final.json)
records exact-hash offline approvals for all runtime/build paths; it grants no
live or release acceptance. The concrete cross-section workflow audit and 18-event
transfer/binding follow-up close the previously identified offline composition
gap; they do not establish live guarantees. Completed offline transfer and local
packaging checks remain milestones. Blocked or unverified checks remain
outstanding; offline success does not establish full live parity or satisfy the
complete goal.

| Remaining prerequisite | Evidence needed to close it |
| --- | --- |
| Private live fixtures | Verify current-account and tenant binding plus account-only creation visibility before writing; run the owning live suites and clean up only their resources. |
| Contact bot search | Current-user authorization for `search:bot`, followed by the failed live scenario and bot binding checks. |
| Native runtime coverage | Execute the retained platform-specific installation, recovery and runtime suites on native Linux, Windows and macOS Intel; cross-built archives do not substitute. |
| Remote release ownership | An owned Next repository/download destination and package identity, then the documented updater identity integration and remote install/update checks. Public signing/notarization and publication retain their separate prerequisites. |
| Comparative release evidence | Finish the serialized development/scaling comparisons, review their tradeoffs, and perform independently replaced held-out tasks only at the final release gate. |
