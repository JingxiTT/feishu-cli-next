# Implementation ledger

The release criteria remain those in [PLAN.md](PLAN.md), with the user-directed
scope adjustment below. Imported source is implementation reuse, not evidence of
live parity. Scoped foundation measurements are available; no full-product
efficiency or release claim is established.

On 2026-09-08 the user asked to ignore Windows and proceed with private live tests
and `search:bot`. Windows execution is deferred by that instruction and is no
longer a blocker for this requested continuation. Windows artifacts and coverage
records remain available; deferred checks are not reported as passes. Private
creation visibility and recipient boundaries from PLAN still apply.

The [2026-09-08 native-fidelity audit](../evaluation/evidence/native-fidelity-validity-audit-20260908.md)
qualifies the historical comparisons below: Docs plain replacement, Sheets native
renderings and Base create-time empty-text behavior differ from the development
fixtures. Preserve all costs and synthetic verdicts, but withhold affected live
same-fidelity efficiency conclusions. Small Base update fidelity remains
unvalidated; native create behavior alone does not establish update behavior.
Sheets/Base correctness changes and the Docs help tip were not covered by those
historical batches. The subsequent [Luna/medium paired pilot](../evaluation/evidence/paired-luna-medium-01/REPORT.md)
ran 72 model trials against corrected synthetic fixtures: 54 verified successes,
seven inconclusive outcomes and 11 failed verifications, with complete token
accounting. Its per-function results do not close outstanding live or release
gates; Markdown and some Sheets strategies remain limited by fixture coverage.

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
checks verified the current user; the later private fixture evidence is recorded below.
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
The following [81-trial scenario series](../evaluation/evidence/scenarios-8d-schema02/REPORT.md)
at `8d92e52` records 63 verified outcomes and 18 inconclusive outcomes after
explicit interpretation reviews; frozen raw verdicts remain intact. Known usage
is a lower bound of 12,201,326 tokens, including 2,282,405 from non-verified
attempts; one attempt has incomplete accounting. Independent recount confirms
three trials per arm/task, all 27 group token/API totals and full cached-input
accounting. Of the five fully verified tasks, Next improves over both baselines
on Drive, Sheets and Tasks under their original synthetic verifiers, regresses
against both on Base, and regresses against
concise guidance on Markdown. Fixture corrections for Docs, Extensions and
Packaging received separate balanced comparisons below. Events retains an explicit
missing-exit-proof limitation. Data-size and generated-business comparisons
remain outstanding; no full-product efficiency claim follows from these results.
The [corrected Docs, Extensions and Packaging series](../evaluation/evidence/scenarios-8d-corrected03/REPORT.md)
(`193c424`) verifies all 27 outcomes with complete usage totaling 3,702,291 tokens.
Independent recount confirms the three trials per arm/task, all group totals,
API sums, distributions and matched-success pairs. Next regresses against both
baselines on Docs, improves against both on Extensions, and uses slightly more
than official guidance but less than concise official guidance on Packaging.
The [Docs trace review](../evaluation/evidence/scenarios-8d-corrected03/docs-trace.md)
records additional discovery/processing with equal API counts against the concise
baseline; it provides no per-command token attribution or causal guarantee.
Earlier failed and inconclusive attempts remain separate, with their costs intact.
The first [size-series condition](../evaluation/evidence/scaling-8d-schema02/REPORT.md)
stopped after seven Base attempts exposed missing valid metadata reads and an
unstated optional-report convention. One recorded outcome is verified; all
1,258,267 provider tokens, including 1,112,364 from non-verified attempts, remain
counted and were independently recounted. The partial series is unbalanced and
supports no size curve or success-rate claim. Twenty unstarted Base slots and
27 untouched Sheets slots were canceled without model calls. A preflight audit
found analogous Sheets preview-read gaps before its first trial. These defects
require a separate corrected condition for new balanced comparisons.
The corrected size/business fixtures are committed at `c61ad62`, with
[independent Base/business review](reference/reviews/evaluation-fixture-v2-review.json)
and saved controlled results: [Base 15/15](../evaluation/evidence/scaling-base-v2-controlled.json),
[Sheets 19/19](../evaluation/evidence/scaling-sheets-v2-controlled.json), and
[business 14/14](../evaluation/evidence/business-v2-controlled.json). These include
actual clean-binary workflows; the reports retain their hash and synthetic-model
limitations. Sheets' final enum distinction has a separate focused pass. The
shared loader/diagnostic follow-up passes 10 tests with one explicit optional skip
on independent rerun; known failures cannot be promoted to inconclusive by a
separate unsupported read. Committed control copies and reviewed hashes were
checked. No Go runtime source or shipped guide changed. Earlier costs and outcomes
remain separate from fresh comparisons under this new evaluator condition.
The [interrupted archive probe](../evaluation/evidence/scaling-8d-fixture03/REPORT.md)
(`8a566f7`) preserves two later nonverified starts: one has unknown usage and the
other has 237,389 actual provider tokens. Independent recount confirms those
numbers plus 27 recorded API calls and 28 commands for the accounted attempt.
Missing archived trace-vocabulary data caused accounting loss after the first
model completed; the second's rejection cannot be attributed from retained data.
The stopped process is terminal; 52 remaining size slots and all 18 business
slots never started. The minimal `4b3abc6` preflight checks that dependency before
provider startup, with 16 independently rerun controlled tests passing. New
balanced trials use the complete pinned archive; these two interrupted
attempts remain outside its comparison and cannot establish a size curve.
The [completed fresh size series](../evaluation/evidence/scaling-8d-archive04/REPORT.md)
under `4b3abc6` contains 54 attempts: Base 25/27 and Sheets 26/27 verified.
Known provider cost is 9,780,669 tokens, including 499,819 from two nonverified
Base attempts. One concise-official Sheets 1,000-row attempt timed out under the
declared 180-second budget with unknown total usage, so the cost is a lower bound
and affected efficiency scores remain null. Independent recount verifies all
30 groups, 31 matched-success pairs, cached-input accounting, distributions and
API totals. The exported all-attempt plot passed visual inspection after correcting
overlapping labels. Next improves Base tokens at 10,000 rows but regresses at
smaller sizes; Sheets token savings coexist with more Next API requests at 10,000
rows under the historical synthetic condition. The native-fidelity amendment
withholds live same-fidelity conclusions; no uniform scaling claim follows.
The [representative business comparison](../evaluation/evidence/business-8d-archive03/REPORT.md)
completed 18/18 verified attempts with complete provider accounting: 2,741,785
tokens under the same immutable evaluator. Independent recount verifies all
12 groups and 12 matched-success pairs, distributions, cached-input accounting
and API totals. Next used fewer tokens and API requests than both baselines on
the approval and Mindnote tasks. These sampled synthetic workflows do not close
full generated-command or live acceptance. Runtime and guide remain frozen.
Held-out trials have not run. A coordinator search exposed fragments of the Base
and Sheets held-out prompts and one size field. The
[exposure record](../evaluation/evidence/held-out-exposure.json) quarantines those
examples for independent replacement after product freeze; full coordinator
blinding is not claimed.

[Live Contact evidence](sections/contact-live-evidence.json) preserves the initial
missing-`search:bot` failure and the authorized 2026-09-08 resumption (`e07178d`).
The narrow grant retained every previous scope and the exact account/tenant;
the selected current-account read suite then passed, including bot search.
Separate bot identity verification passed. Bot tenant verification remains
blocked by the additional, unrequested `tenant:tenant:readonly` app scope.

The [Drive privacy revalidation](sections/drive-privacy-prerequisites.json)
records readable official guidance on folder inheritance and configurable
new-document sharing defaults. It does not establish this account's effective
defaults or the exact API creation rule. A private parent alone therefore does
not close the precreation requirement; no authenticated calls or writes were made
for this documentation check.

The [2026-09-08 private fixture preflight](sections/private-live-preflight-20260908.json)
adds actual current-account evidence: the UI's new-document default was already
invited-only, a personal test folder was created with no invited collaborators,
and independent CLI checks matched its owner and sole collaborator to the verified
current account with link sharing closed. No sharing setting was changed.
The bounded [Drive lifecycle](sections/drive-private-live-evidence.json) passes
exact byte/hash preservation and independently confirmed deletion (`b5505a9`).
The [Docs live evidence](sections/docs-private-live-evidence.json) retains a failed
plain replacement that removed formatting and a passing explicit rich-XML
lifecycle (`f917e22`), with both documents independently confirmed deleted.
These are named private workflows, not full section live acceptance.

The [Markdown private lifecycle](sections/markdown-private-live-evidence.json)
(`45e4698`) passed exact source-file byte preservation and native Markdown
heading, bold/link, table, Unicode and code readback. Both run-owned resources
were independently confirmed deleted. The initial file-path validation attempt
created no resources and remains recorded separately.
The [batch closure](sections/private-live-summary-20260908.json) records 13 created
children, all independently confirmed deleted, followed by a fresh complete empty
parent listing and one parent delete. Its exact-folder listing returned native
`1061007 / file has been delete.`; successful post-delete permission metadata was
not accepted as deletion proof. No collaborators were invited, sharing defaults
were changed or unknown mutations replayed. The test browser tab was closed.

Native testing also exposed formula identity missing from raw Sheets exports and
Base empty text normalizing to null. The reviewed Base fix (`b746eff`) rejects
explicit empty text during complete typed-import preflight before destination
reads or writes; full Base offline suites pass (1,097 unit, 160 legacy and 178
process events, one existing broad-live skip). The separate
[supported Base live scenario](sections/base-private-supported-live-evidence.json)
verified native create/read, zero/null update/readback and unchanged no-op. Its
negative import assertion failed on a test file-path problem, so the selected
suite remains failed and the live negative check remains unverified. The harness
path and diagnostic capture are corrected; cleanup independently confirmed the
Base deleted. The offline regression proves empty-text rejection before any write.

The [Sheets fix and final private lifecycle](sections/sheets-private-live-evidence.json)
(`02b6c5f`, evidence `17b275e`) preserve raw scalar types and explicit formulas
using paired native renderings with matching revisions. Its complete affected
unit suite, 68 inherited checks, 300 process checks and 23 cross-section checks
pass; six broader live lifecycles remain unverified. A fresh live workbook passed
all six typed/formula assertions and exact-ID deletion. The coordinator rechecked
eight evidence hashes and the native artifact independently. Earlier raw-only
artifacts may lack formula identity and must be re-exported before reuse.
The [independent fix review](reference/reviews/p3-native-fidelity-review.json)
approves both changes at exact source hashes. Failed attempts remain retained;
no replacement model comparisons have run.

The [post-change checks](sections/private-live-final-gates-20260908.json) record
129 passing packages in the broad race run and four path-related failing packages.
All four pass their separate race rerun under a short isolated `/tmp` directory
(1,551 test events). The root package passes after updating its existing embedded
help expectation. Vet, formatting, unchanged module tidy, pinned Go lint, source
guards and the committed quality gate pass; original failed invocations and their
causes remain in the report.

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

The [actual Linux runtime checkpoint](sections/linux-runtime-evidence.json) at
`8ce0688` records 18,641 race-test passes and 14 skips, plus 3,544 offline
E2E/harness pass events and 57 skips across all 23 section suites and supporting
checks. The script gate also passes, including 166 Node tests. These runs used
the existing Linux aarch64 VM with no external network access or host mounts.
Independent recount verifies the suite totals and twelve evidence-report hashes.
Initial setup failures remain recorded. The original capability gate failed
because its Darwin reference differs on two platform-specific keychain risk
fields. An actual Linux reference capture confirms exactly those two differences.
The independently reviewed platform correction in `cb3a750` passes its actual
Linux CLI follow-up with all 5,742 approved differences and no new exceptions.
Four closure-artifact hashes and unchanged product source/binary were independently
verified. The original failed report and separate semantic-review requirement remain.

[Packaging evidence](sections/packaging-evidence.json) covers all 14 cross-built
platform/variant archives, local installation/update/removal, and actual Node 16
runtime tests. The final `8ce0688` refresh also verifies both darwin/amd64 variants
under existing Rosetta, including startup and project guide installation. That
is translated execution, not an additional physical-hardware requirement.
Both delivered Linux arm64 variants also pass startup and project guide
install/update/removal checks, including edited-file refusal and preservation of
unrelated files (`6dda7c0`). Their archive, raw-report and harness hashes were
independently verified. Linux runtime evidence is recorded separately above.
Both Linux amd64 archives pass the same lifecycle checks using the VM's existing
QEMU translation (`b10120e`), with independently verified archive/report hashes.
This does not imply a full amd64 runtime-suite run. Both riscv64 archives also
pass the eleven lifecycle checks under explicit QEMU invocation (`88ed1cc`),
using a test-only signed Debian dependency in an isolated container. Archive,
bootstrap, harness and raw-result hashes were independently verified. The earlier
`ENOEXEC` probe remains historical evidence; no host emulator or new VM was installed.
Broader riscv64 runtime suites remain unexecuted. Windows OS-specific behavior,
signed public-release checks and a trusted remote Next release channel remain
unverified or unavailable. Local evidence does not establish public release readiness.

## Outstanding release evidence

P2 domain acceptance, remaining private live scenarios,
remaining OS-specific and remote lifecycle checks, full release checks and held-out
evaluation remain open. The [source review aggregate](reference/p3-effective-review-current.json)
records exact-hash offline approvals for all runtime/build paths; it grants no
live or release acceptance. Its `cb3a750` checkpoint includes the platform guard
correction; the [earlier aggregate](reference/p3-effective-review-final.json)
retains the `8ce0688` boundary. The concrete cross-section workflow audit and 18-event
transfer/binding follow-up close the previously identified offline composition
gap; they do not establish live guarantees. Completed offline transfer and local
packaging checks remain milestones. Blocked or unverified checks remain
outstanding; offline success does not establish full live parity or satisfy the
complete goal.

| Remaining prerequisite | Evidence needed to close it |
| --- | --- |
| Private live workflows | The named private folder and current account were verified; all 13 run-created children and parent are deleted. Bounded Drive, explicit-XML Docs, Sheets and Markdown lifecycles pass. Sheets formula correction and its private lifecycle pass; Base supported steps pass but its selected suite retains a failed negative assertion. Remaining section live scenarios and broader type/concurrency guarantees stay open. |
| Contact bot tenant binding | Requested `search:bot` grant and selected read suite passed. Separate bot tenant verification still requires the unrequested `tenant:tenant:readonly` app scope. |
| OS-specific runtime coverage | Linux arm64 runtime suites and platform comparison pass; delivered archive lifecycles pass on arm64 and under amd64/riscv64 translation. Broader amd64/riscv64 runtime suites have not run. Windows behavior checks and native reference capture are deferred by the user's 2026-09-08 instruction; cross-built archives are not runtime verification. Darwin/amd64 smoke checks already ran under Rosetta. Additional gaps must name the unverified scenario, not require physical hardware absent from PLAN. |
| Remote release ownership | An owned Next repository/download destination and package identity, then the documented updater identity integration and remote install/update checks. Public signing/notarization and publication retain their separate prerequisites. |
| Comparative release evidence | Development comparisons retain all costs and tradeoffs. The native-fidelity audit withholds affected Docs/Sheets/Base-size same-fidelity claims; small Base update fidelity is unvalidated. New correctness changes have no model measurements. Retain older product conditions and the inconclusive Events baseline separately; perform independently replaced held-out tasks only at the final release gate. |

The read-only Windows-environment check found VirtualBox `7.1.6r167084`, but
`VBoxManage list vms` and `VBoxManage list runningvms` both returned empty lists
with exit 0. No Parallels, VMware Fusion or UTM installation was found in the two
standard Applications directories. A Windows runner or VM is still needed;
this evidence remains historical after the user's 2026-09-08 Windows deferral.
