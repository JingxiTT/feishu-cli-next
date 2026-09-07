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
The following [81-trial scenario series](../evaluation/evidence/scenarios-8d-schema02/REPORT.md)
at `8d92e52` records 63 verified outcomes and 18 inconclusive outcomes after
explicit interpretation reviews; frozen raw verdicts remain intact. Known usage
is a lower bound of 12,201,326 tokens, including 2,282,405 from non-verified
attempts; one attempt has incomplete accounting. Independent recount confirms
three trials per arm/task, all 27 group token/API totals and full cached-input
accounting. Of the five fully verified tasks, Next improves over both baselines
on Drive, Sheets and Tasks, regresses against both on Base, and regresses against
concise guidance on Markdown. Fixture corrections for Docs, Extensions and
Packaging will receive separate balanced comparisons. Events retains an explicit
missing-exit-proof limitation. Real data-size and generated-business comparisons
are still running; no full-product efficiency claim follows from these results.
Held-out trials have not run. A coordinator search exposed fragments of the Base
and Sheets held-out prompts and one size field. The
[exposure record](../evaluation/evidence/held-out-exposure.json) quarantines those
examples for independent replacement after product freeze; full coordinator
blinding is not claimed.

[Live Contact evidence](sections/contact-live-evidence.json) verifies the current
account and the sequential user reads. The full test failed on bot search; a
single bounded diagnosis identified missing `search:bot` user scope. No permission
was granted and no content was written. This remains an unmet live prerequisite.

The [Drive privacy revalidation](sections/drive-privacy-prerequisites.json)
records readable official guidance on folder inheritance and configurable
new-document sharing defaults. It does not establish this account's effective
defaults or the exact API creation rule. A private parent alone therefore does
not close the precreation requirement; no authenticated calls or writes were made
for this documentation check.

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

P2 domain acceptance, verified private live fixtures,
remaining development comparisons and provider-token scaling measurements,
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
| Private live fixtures | Verify current-account and tenant binding plus account-only creation visibility before writing; run the owning live suites and clean up only their resources. |
| Contact bot search | Current-user authorization for `search:bot`, followed by the failed live scenario and bot binding checks. |
| OS-specific runtime coverage | Linux arm64 runtime suites and platform comparison pass; delivered archive lifecycles pass on arm64 and under amd64/riscv64 translation. Broader amd64/riscv64 runtime suites have not run. Execute the retained Windows behavior checks and native reference capture on an appropriate OS runner or VM; cross-built archives do not substitute. Darwin/amd64 smoke checks already ran under Rosetta. Additional gaps must name the unverified scenario, not require physical hardware absent from PLAN. |
| Remote release ownership | An owned Next repository/download destination and package identity, then the documented updater identity integration and remote install/update checks. Public signing/notarization and publication retain their separate prerequisites. |
| Comparative release evidence | Finish the serialized development/scaling comparisons, review their tradeoffs, and perform independently replaced held-out tasks only at the final release gate. |

The read-only Windows-environment check found VirtualBox `7.1.6r167084`, but
`VBoxManage list vms` and `VBoxManage list runningvms` both returned empty lists
with exit 0. No Parallels, VMware Fusion or UTM installation was found in the two
standard Applications directories. A Windows runner or VM is still needed;
the user has been asked for an available target while independent comparisons run.
