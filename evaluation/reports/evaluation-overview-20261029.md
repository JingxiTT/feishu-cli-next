# Full evaluation report — 2026-10-29

> Current scope update: IM/group-chat evaluation cases, fixture code and dispatcher
> routes were removed at the user’s request after these measurements. IM status
> below describes the recorded run; sealed evidence and results are unchanged.

This report consolidates the current real-Feishu GPT-6 Luna evaluations and indexes the retained historical model evaluations and offline CLI measurements. It introduces no model trials, tenant operations, permission changes or rescoring of original records.

## Findings

The current series measures **19 live workflow types**. There are **106 model attempts on workflows whose complete outcomes can be independently verified**, plus **four partially verified file-handoff attempts**, for **110 provider-accounted model trials**. Two additional IM workflow types are implemented but blocked. Four earlier adapter failures occurred before provider launch and remain separately recorded with their original null accounting.

Next used fewer tokens per successful task in each of the five current experiment groups, with observed reductions from **11.1% to 38.6%**. Success was equal in four groups and higher for Next in the creation/media group. This does not establish a universal advantage: Next used more tokens on four individual workflows, and its median latency was higher in the complex group. Groups have different tasks, repeats and budgets; no pooled efficiency score or population success-rate claim is made.

## Current live results by experiment

Success below means the verified native task outcome. Tokens per success include every attempted model cost in the corresponding task or experiment, including failures.

| Experiment | Workflow types | Repeats / arm / type | Official success | Next success | Official tokens/success | Next tokens/success | Next change |
|---|---:|---:|---:|---:|---:|---:|---|
| Current-account Contact diagnostic | 1 | 3 | 3/3 (100.0%) | 3/3 (100.0%) | 186,075 | 165,061 | 11.3% fewer |
| Initial business workflows | 4 | 2 | 8/8 (100.0%) | 8/8 (100.0%) | 308,628 | 189,483 | 38.6% fewer |
| Extended business workflows | 6 | 3 | 17/18 (94.4%) | 17/18 (94.4%) | 454,324 | 317,858 | 30.0% fewer |
| Creation, media and diagrams | 5 | 3 | 14/15 (93.3%) | 15/15 (100.0%) | 321,347 | 212,719 | 33.8% fewer |
| Complex business workflows | 3 | 3 | 8/9 (88.9%) | 8/9 (88.9%) | 553,854 | 492,435 | 11.1% fewer |

The Contact row is a read-only diagnostic; its task asks for the current account and tenant, rather than a general contact search. The initial business row excludes the four partial file-handoff trials described below. Extended native outcomes separate one successfully completed Next task from a controller cleanup timeout. Complex native outcomes include one documented formatting-only verifier correction per CLI. Original raw flags are unchanged.

## Complete workflow results and coverage

Each row is one distinct live task type, with two or three attempts per CLI as shown. A negative reduction is written as “more”; it remains part of the report.

| Workflow | Official success | Next success | Official tokens/success | Next tokens/success | Next change |
|---|---:|---:|---:|---:|---|
| Current account and tenant lookup | 3/3 | 3/3 | 186,075 | 165,061 | 11.3% fewer |
| Cross-source financial report | 2/2 | 2/2 | 507,784 | 374,524 | 26.2% fewer |
| Rich-text document edit | 2/2 | 2/2 | 249,616 | 80,768 | 67.6% fewer |
| Typed Sheet reconciliation | 2/2 | 2/2 | 309,637 | 125,378 | 59.5% fewer |
| Base record intake | 2/2 | 2/2 | 167,473 | 177,264 | 5.8% more |
| 2,505-record Base audit | 3/3 | 3/3 | 280,881 | 222,930 | 20.6% fewer |
| Conditional Base closing | 3/3 | 3/3 | 265,340 | 286,180 | 7.9% more |
| Cross-domain budget settlement | 3/3 | 3/3 | 907,600 | 809,070 | 10.9% fewer |
| 520-invoice document audit | 3/3 | 2/3 | 249,829 | 308,911 | 23.6% more |
| Typed Sheet export | 2/3 | 3/3 | 725,202 | 87,781 | 87.9% fewer |
| Approval-gated CSV intake | 3/3 | 3/3 | 387,384 | 189,292 | 51.1% fewer |
| Create a rich native document | 3/3 | 3/3 | 173,396 | 144,336 | 16.8% fewer |
| Create an SVG drawing | 2/3 | 3/3 | 749,250 | 285,949 | 61.8% fewer |
| Upload CSV and binary attachments | 3/3 | 3/3 | 245,728 | 221,292 | 9.9% fewer |
| Upload two same-named images | 3/3 | 3/3 | 226,555 | 186,033 | 17.9% fewer |
| Create a Mermaid whiteboard | 3/3 | 3/3 | 354,443 | 225,985 | 36.2% fewer |
| Versioned Base conflict reconciliation | 3/3 | 3/3 | 382,805 | 409,080 | 6.9% more |
| Long-document invoice closeout | 2/3 | 2/3 | 1,008,528 | 971,838 | 3.6% fewer |
| Versioned multi-currency Sheet journal | 3/3 | 3/3 | 421,786 | 256,189 | 39.3% fewer |

The strongest observed reductions were typed Sheet export (87.9%), rich document editing (67.6%), SVG drawing creation (61.8%), and typed Sheet reconciliation (59.5%). Those are small task-specific samples. The regressions were Base intake (5.8% more), conditional Base closing (7.9% more), the large-document audit (23.6% more), and Base conflict reconciliation (6.9% more).

| Workflow | Independently checked work |
|---|---|
| Current account and tenant lookup | Verified current-account read only; does not establish arbitrary-person lookup performance. |
| Cross-source financial report | Read Sheet + complete 205-row Base, compute filtered totals, append a document paragraph; preserve all sources. |
| Rich-text document edit | Replace a status while retaining bold/link formatting, native IDs, Unicode and unrelated blocks. |
| Typed Sheet reconciliation | Write numeric zero and empty text; preserve leading-zero codes, booleans, styles, guard cells and formulas. |
| Base record intake | Append two typed records with null/false/zero/Unicode while preserving 205 existing records. |
| 2,505-record Base audit | Read beyond the CLI 2,000-record boundary; positive/zero/negative predicates and complete native pagination. |
| Conditional Base closing | Select all qualifying records; change only checkbox/note and preserve every other row and field. |
| Cross-domain budget settlement | Save original Base selection; update selected records, Sheet total/note and one document summary. |
| 520-invoice document audit | Filter release/status/strictly positive amounts across more than 500 blocks; exact ID/code/count/typed total report. |
| Typed Sheet export | Export raw JSON values and explicit formulas using native render modes; preserve the workbook. |
| Approval-gated CSV intake | Join quoted multiline Unicode CSV by exact leading-zero code; update only already-approved rows. |
| Create a rich native document | Create a heading, bold link, Unicode/leading-zero content and a native table; verify resulting document structure. |
| Create an SVG drawing | Verify eight native drawing nodes, shapes/colors, text, connectors, geometry and vector diamond fallback. |
| Upload CSV and binary attachments | Verify downloaded bytes, Unicode filenames, native view/file parentage and all original document content. |
| Upload two same-named images | Different PNG bytes/dimensions with identical basenames; verify downloaded bytes and display sizes. |
| Create a Mermaid whiteboard | Verify stored Mermaid source, participants/connectors/branches and preservation of the original document. |
| Versioned Base conflict reconciliation | Resolve 79 adjustment rows against all 205 records: latest request revisions, original-value conflicts, approval gates and null/zero/negative writes. |
| Long-document invoice closeout | Update 107 strictly positive approved invoices across 520 paragraphs in place; append one summary and preserve IDs, formatting and unrelated blocks. |
| Versioned multi-currency Sheet journal | Resolve 34 shuffled inputs / 24 event IDs: revisions, duplicates, voids, exact codes, approvals and per-event decimal FX rounding. |

Sheet fixtures use a six-by-six range with four data rows and a guard row; the journal has 34 local input rows. They test types, formulas, joins and business rules, **not large multi-sheet workbook scale**. The large remote datasets are Base with 2,505 records and Docs with 520 invoice paragraphs plus guards. Image/file uploads are document attachments; they do not establish generic Drive folder placement or IM resource delivery. Mermaid/SVG checks verify native structure, stored source/geometry and preservation, rather than subjective visual quality. Fixture provisioning and cleanup exercise controller capabilities, not agent creation performance unless the creation itself is the task.

## Failures, corrections and incomplete verification

Five genuine native task failures remain across the fully verifiable current workflows:

| Experiment / arm | Failure and implication |
|---|---|
| Extended / official | Sheet export emitted ten numeric/boolean literals as display strings, including `0` as `"0"` and `false` as `"FALSE"`; the workbook itself was unchanged. |
| Extended / Next | Large-document audit found the right IDs/codes and total magnitude but wrote `total_amount` as string `"1006"` instead of numeric `1006`; the document was unchanged. |
| Collaboration / official | SVG drawing contained 16 nodes instead of the required eight: two copies of the drawing. |
| Complex / official | Selected six zero-amount invoices despite the strictly-positive predicate: 113 records instead of 107. The total still matched, so a sum-only check would miss this failure. |
| Complex / Next | The same zero-amount selection error occurred in one trial. |

In the complex trials with wrong reports, post-edit content preservation was not assessed because the frozen verifier stopped at the report mismatch. Cleanup still independently checked ownership/private ACL before exact-ID deletion.

The extended experiment retains raw run success of **17/18 official and 16/18 Next**, while native task success is **17/18 for each**. One Next settlement passed all task checks and then its controller cleanup timed out. Later exact-ID reads established the remaining folder and cleanup completed; no model trial was replayed. Raw run tokens/success show 25.7% fewer Next tokens; the task-only interpretation shows 30.0% fewer. Both are retained.

The complex experiment retains raw acceptance of **7/9 for each CLI**. One result per CLI was falsely rejected because Feishu split text into adjacent runs with identical metadata and formatting. Complete independently recorded native readback supports **8/9 native task successes for each** under a narrowly corrected check: only identical adjacent run metadata may coalesce in selected edited paragraphs. Exact text, style/link data, block IDs, root order and unrelated blocks still have to match. No model was rerun; all original flags and costs remain available.

The initial file-handoff diagnostic has two attempts per CLI. Official passed checked requirements once and failed once; Next passed checked requirements twice. Byte content, filename, owner, ACL and guard checks were available, but the granted metadata could not independently establish the destination folder. Therefore **three apparent positive results remain inconclusive for full-task success**, and one result remains a known failure. No finite full-task efficiency score is supported. Costs remain **432,611 official tokens and 246,874 Next tokens**, including 215,075 tokens on the failed official trial.

## Blocked and unmeasured live coverage

| Area | Current status |
|---|---|
| Direct-message reads | Case and recipient checks implemented. Native setup send was rejected with Feishu `230027` (`user_unauthorized`); zero messages were created. No live read/model success is claimed. |
| Group message history | Bounded-history/pagination case implemented. The candidate group contained an unconfigured bot, so it was neither read nor messaged. No eligible private `[TEST]` group was available. |
| Task/tasklist writes | Excluded under the user-granted permission boundary; no new consent flow or out-of-scope Feishu task was created. |
| Document sharing / collaborator grants | Private recipient config exists, but no comparative live sharing task was executed. Config selection alone does not authorize a send or share. |
| Generic Drive destination placement | File handoff remains only partially verified; broad Drive listing was not added. |
| Large/multi-sheet models, pivots, charts and conditional formats | No current paired live model evaluation. |
| Calendar, Mail, Wiki, Slides, Minutes, Meetings, OKR, Apps and Approval | Historical or other correctness evidence may exist, but no fresh paired live Luna workflow in this series. |
| Event consumption, extensions, packaging, SDK behavior and cross-platform execution | Not established by the current live business evaluations. |

All recipient identities, tenant/chat IDs, configuration and raw native artifacts remain outside Git in protected local files. Existing permissions were retained; no new scopes, group creation, public links or broad sharing were introduced.

## Accounting, latency and reproducibility

Provider accounting is complete for **110/110 current model trials**. The administrative token ledger, including the four partial file diagnostics, is:

| Arm | Known provider tokens |
|---|---:|
| Official | 20,113,056 |
| Next | 14,791,768 |
| Total | 34,904,824 |

These totals are accounting inventory, **not a pooled efficiency score**. Cached input is included once inside input tokens; total tokens equal input plus output. Costs on failed attempts and recovery work stay included. Controller setup/calibration/verification/cleanup uses no model tokens and is excluded. No provider pricing or monetary savings is inferred. API-call and retry totals are unavailable from the production adapter.

| Experiment | Agent budget | Official median agent seconds | Next median agent seconds |
|---|---:|---:|---:|
| Current-account Contact diagnostic | 180s | 58.0 | 45.7 |
| Initial business workflows | 360s | 62.4 | 44.2 |
| Extended business workflows | 480s | 81.4 | 61.0 |
| Creation, media and diagrams | 480s | 62.6 | 57.5 |
| Complex business workflows | 480s | 86.3 | 104.4 |

All current runs use **`codex/gpt-6-luna` through Magpie**, medium reasoning, native Codex **0.165.0**, shell/files tools, file-delivered guidance, fresh model contexts and balanced randomized schedules. Official is **v1.0.98**, commit `ac81f5056ab88f37d581d25916f3dab85abfca76`; Next is commit `478827611a3ad6735477ecc2da3dbfb31812c260`. The exact binary hashes are in the consolidated assessment and original provenance. Official was built with Go 1.25.5 and Next with Go 1.26.10, limiting attribution of latency differences. The model ID is a provider alias, not an immutable model snapshot identifier.

Initial Contact budgets were 180 seconds; initial business budgets 360 seconds; subsequent suites 480 seconds. The complex sample has a higher Next median latency even though its token total is lower. Different workload mixes, budgets and repetition counts remain separate; two or three repeats do not establish statistical parity, broad release readiness or a universal CLI advantage.

Four initial complex adapter failures omitted the frozen command registry and occurred before the provider subprocess could launch. Original records retain null accounting; all fixtures were cleaned. An independently validated corrected snapshot produced the 18 measured model trials. One extended fixture setup stopped before model launch on a native Base revision change; read-only reconciliation and cleanup preceded a fresh fixture for the same unbilled schedule slot. Neither was silently counted as a model success or a free billed attempt.

Calibration failures are retained separately: early file/board representation assumptions were corrected before media trials; CSV clipping changed a native diagnostic until a fresh fixture widened the column; and a complex Base calibration used the wrong HTTP method, received a definitive validation rejection, and was confirmed unchanged before the corrected POST. No unknown mutation was automatically replayed.

## Cleanup and current software checks

The four business/collaboration experiments confirmed deletion of **326 registered resources** in total, including calibration/interruption/attempt resources and dedicated parents: initial business 76 children plus one parent, extended 121 including parent, collaboration 77 including parent, complex 51 including parent. Contact created no remote resources. Attachment/board cleanup is via their owning documents. No unresolved run/mutation markers remain in the finalized records, and temporary model authentication was removed. These counts cover registered resources rather than an unrelated-tenant inventory.

Latest evaluator software checks: **149 passed, zero failed, one platform-specific native Codex isolation check skipped**. `make script-test` passed. These offline tests check the harness and verifiers; their pass counts are not live model task-success measurements. Layout/seal, staged diff and new-file private-value scans passed. Earlier reports retain their validation state at the time; for example the first business report records a then-missing Ruby prerequisite, whereas subsequent full script checks passed with local tooling.

## Historical evaluations and local probes

Historical trials use older product revisions, models and harness/fixture conditions. Most model comparisons used signed local simulated Feishu backends rather than a real tenant; the foundation tasks were offline CLI tasks. They remain useful for diagnostics and historical regressions, but are not added to the current live success/token scores. Some conditions reuse earlier trials, so summing historical rows would double-count models and spend. All original outcomes and unknown usage remain intact.

The historical full matrix records **43 task types and 258 attempts**: 240 fresh attempts plus 18 reused condition-09 attempts. It reports **13 task gates met, 22 missed, eight unavailable**, with **six attempts of incomplete usage**. Known tokens are **31,650,572 fresh plus 901,686 reused**. The raw matrix records 102/129 accepted official outcomes with two inconclusive outcomes, and 107/129 accepted Next outcomes; these are fixture verdicts, not live success rates. No finite complete-accounting pooled efficiency score follows.

Historical qualifications materially affect interpretation: Docs plain-replacement style inheritance, Sheets raw/formula rendering, and Base create-time empty text assumptions were contradicted by native controls. Conditions 05/06 also had incomplete suppression of ambient skill catalogs. Some domain-read failures were correct typed answers rejected for an unsupported fixture route: the matrix qualification identifies one such official attempt and nine Next attempts. Those original flags remain unchanged, and the evidence does not justify turning every historical fixture rejection into a product regression. See the [native fidelity audit](../../evidence/evaluation/native-fidelity-validity-audit-20260908.md), [skill isolation qualification](../../evidence/evaluation/token-efficiency-05-06-skill-isolation-qualification.md), and [matrix interpretation limits](../../evidence/evaluation/full-matrix-20260911/QUALIFICATIONS.md).

The October 7 offline surface comparison covers **1,738 local surfaces** and **21,606 timed subprocesses** with zero required probe/contract failures. These are local output/context and process measurements, with no provider/model or Feishu calls:

| Surface | Cases | Observed Next output-context change |
|---|---:|---|
| Request-only API schemas | 251 | 61.86% fewer tokens in the sum; every case smaller |
| Event schemas | 25 | 38.06% fewer; semantic definitions retained |
| Canonical help outputs | 938 | 19.09% more; 936 of 938 larger |
| Inherited alias help | 30 | 19.84% more; all larger |
| Embedded guidance files | 493 | 0.04% fewer in the sum; 25 smaller, 464 equal, four larger |

The six-method probe’s 81.4% reduction is a selected schema-output sample, not the whole-catalog result or provider tokens per completed task. Current upstream was faster in the broad local startup comparison (roughly 12.6–14.5 ms surface medians versus 30.1–33.4 ms for Next on that machine). Historical traces observed non-help calls to 72/938 original nodes; 866 lacked observed non-help calls in that retained trace set. That ledger is not updated execution coverage for the current 19-workflow sample and does not prove the other commands are unavailable.

### Retained historical condition index

Counts below are the original condition’s reported metrics. “—” means not extracted/reported in that summary, not zero. Reused baselines and separate diagnostics remain subject to their linked reports; do not sum these rows.

| Condition | Evidence type | Recorded attempts | Original accepted outcomes | Known tokens | Unknown accounting |
|---|---|---:|---:|---:|---:|
| [business-8d-archive03](../../evidence/evaluation/business-8d-archive03/REPORT.md) | historical model trials on simulated Feishu | 18 | 18 | 2,741,785 | 0 |
| [cli-efficiency-20261007](../../evidence/evaluation/cli-efficiency-20261007/REPORT.md) | offline CLI probes | — | — | — | — |
| [cli-surface-20261007](../../evidence/evaluation/cli-surface-20261007/REPORT.md) | offline CLI probes | — | — | — | — |
| [contact](../../evidence/evaluation/contact/REPORT.md) | historical model trials on simulated Feishu | 9 | 9 | 502,586 | 0 |
| [foundation](../../evidence/evaluation/foundation/REPORT.md) | offline foundation tasks | 18 | 18 | 887,335 | 0 |
| [full-matrix-20260911](../../evidence/evaluation/full-matrix-20260911/REPORT.md) | historical model trials on simulated Feishu | 258 | 209 | 32,552,258 | 6 |
| [paired-luna-medium-01](../../evidence/evaluation/paired-luna-medium-01/REPORT.md) | historical model trials on simulated Feishu | 72 | 54 | 8,442,585 | 0 |
| [read-312af93](../../evidence/evaluation/read-312af93/REPORT.md) | historical model trials on simulated Feishu | 81 | 75 | 8,419,135 | 0 |
| [read-8d92e5](../../evidence/evaluation/read-8d92e5/REPORT.md) | historical model trials on simulated Feishu | 27 | 27 | 3,647,948 | 0 |
| [scaling-8d-archive04](../../evidence/evaluation/scaling-8d-archive04/REPORT.md) | historical model trials on simulated Feishu | 54 | 51 | 9,780,669 | 1 |
| [scaling-8d-fixture03](../../evidence/evaluation/scaling-8d-fixture03/REPORT.md) | historical model trials on simulated Feishu | 2 | 0 | 237,389 | 1 |
| [scaling-8d-schema02](../../evidence/evaluation/scaling-8d-schema02/REPORT.md) | historical model trials on simulated Feishu | 7 | 1 | 1,258,267 | 0 |
| [scenarios-8d-corrected03](../../evidence/evaluation/scenarios-8d-corrected03/REPORT.md) | historical model trials on simulated Feishu | 27 | 27 | 3,702,291 | 0 |
| [scenarios-8d-schema02](../../evidence/evaluation/scenarios-8d-schema02/REPORT.md) | historical model trials on simulated Feishu | 81 | 63 | 12,201,326 | 1 |
| [scenarios-8d92e5](../../evidence/evaluation/scenarios-8d92e5/REPORT.md) | historical model trials on simulated Feishu | 10 | 1 | 75,555 | 9 |
| [token-efficiency-02](../../evidence/evaluation/token-efficiency-02/REPORT.md) | historical model trials on simulated Feishu | 72 | 64 | 7,218,218 | 0 |
| [token-efficiency-03](../../evidence/evaluation/token-efficiency-03/REPORT.md) | historical model trials on simulated Feishu | 72 | 62 | 7,249,494 | 1 |
| [token-efficiency-04](../../evidence/evaluation/token-efficiency-04/REPORT.md) | historical model trials on simulated Feishu | 24 | 24 | 1,154,003 | 0 |
| [token-efficiency-05](../../evidence/evaluation/token-efficiency-05/REPORT.md) | historical model trials on simulated Feishu | 48 | 40 | 3,602,432 | 4 |
| [token-efficiency-06](../../evidence/evaluation/token-efficiency-06/REPORT.md) | historical model trials on simulated Feishu | 12 | 12 | 761,257 | 0 |
| [token-efficiency-07](../../evidence/evaluation/token-efficiency-07/REPORT.md) | historical model trials on simulated Feishu | 12 | 11 | 720,698 | 0 |
| [token-efficiency-08](../../evidence/evaluation/token-efficiency-08/REPORT.md) | historical model trials on simulated Feishu | 12 | 12 | 677,392 | 0 |
| [token-efficiency-09](../../evidence/evaluation/token-efficiency-09/REPORT.md) | historical model trials on simulated Feishu | 18 | 17 | 901,686 | 0 |
| [wiki-diagnostic-74](../../evidence/evaluation/wiki-diagnostic-74/REPORT.md) | historical model trials on simulated Feishu | 6 | 2 | 1,091,185 | 0 |
| [wiki-native-74](../../evidence/evaluation/wiki-native-74/REPORT.md) | historical model trials on simulated Feishu | 9 | 9 | 1,595,048 | 0 |

The foundation row is its compatible 18-trial batch; its report additionally retains 20 earlier attempts with unknown usage. The stopped `scenarios-8d92e5` row has nine Base startup failures plus one completed Docs attempt and a separate diagnostic with unknown usage. Condition 04 has 24 new candidate attempts and 48 reused baselines; matrix condition-09 rows are also reused. Conditions 06–08 stopped after 12 of 48 planned attempts. Wiki’s six-attempt mechanism probe is unbalanced; its later nine-trial “native” label describes source-backed fixture routes, not a live Feishu tenant.

### Original full-matrix task ledger

These are the unchanged historical simulated/fixture results and historical thresholds. The native-fidelity, route-coverage, ambient-skill and incomplete-accounting qualifications above remain applicable. They are not current release gates or present live outcomes.

| Suite / task | Original official accepted | Original Next accepted | Historical token saving/success | Historical gate | Source |
|---|---:|---:|---:|---|---|
| paired / paired-discovery | 3/3 | 3/3 | -11.9% | missed | fresh in that matrix |
| paired / paired-contact | 3/3 | 3/3 | 8.8% | missed | fresh in that matrix |
| paired / paired-docs-read | 3/3 | 3/3 | 39.2% | missed | fresh in that matrix |
| paired / paired-docs-edit | 3/3 | 3/3 | 52.8% | met | fresh in that matrix |
| paired / paired-markdown | 3/3 | 3/3 | 60.4% | met | fresh in that matrix |
| paired / paired-sheets | 1/3 | 3/3 | unavailable | unavailable | fresh in that matrix |
| paired / paired-base | 1/3 | 3/3 | 81.9% | met | fresh in that matrix |
| paired / paired-drive | 3/3 | 3/3 | 65.3% | met | fresh in that matrix |
| docs-hard / docs-hard-rich-read | 3/3 | 3/3 | 38.2% | missed | reused |
| docs-hard / docs-hard-focused-read | 3/3 | 3/3 | 31.1% | missed | reused |
| docs-hard / docs-hard-multi-edit | 2/3 | 3/3 | unavailable | unavailable | reused |
| read / application-dev-01 | 1/3 | 2/3 | -71.3% | missed | fresh in that matrix |
| read / apps-dev-01 | 3/3 | 3/3 | -92.0% | missed | fresh in that matrix |
| read / im-dev-01 | 2/3 | 1/3 | -87.0% | missed | fresh in that matrix |
| read / minutes-dev-01 | 3/3 | 3/3 | 12.3% | missed | fresh in that matrix |
| read / note-dev-01 | 3/3 | 2/3 | -89.5% | missed | fresh in that matrix |
| read / slides-dev-01 | 3/3 | 3/3 | -18.9% | missed | fresh in that matrix |
| read / vc-dev-01 | 2/3 | 2/3 | 26.5% | missed | fresh in that matrix |
| read / whiteboard-dev-01 | 3/3 | 3/3 | 30.8% | met | fresh in that matrix |
| read / wiki-dev-01 | 1/3 | 0/3 | unavailable | missed | fresh in that matrix |
| read / calendar-dev-01 | 3/3 | 1/3 | -443.9% | missed | fresh in that matrix |
| read / mail-dev-01 | 3/3 | 3/3 | 26.7% | missed | fresh in that matrix |
| read / okr-dev-01 | 3/3 | 3/3 | -28.4% | missed | fresh in that matrix |
| scenario / base-dev-01 | 3/3 | 3/3 | 60.2% | met | fresh in that matrix |
| scenario / docs-dev-01 | 3/3 | 3/3 | 55.7% | met | fresh in that matrix |
| scenario / drive-dev-01 | 3/3 | 3/3 | 62.7% | met | fresh in that matrix |
| scenario / events-dev-01 | 1/3 | 3/3 | unavailable | unavailable | fresh in that matrix |
| scenario / extensions-dev-01 | 1/3 | 1/3 | unavailable | unavailable | fresh in that matrix |
| scenario / markdown-dev-01 | 3/3 | 3/3 | 50.8% | met | fresh in that matrix |
| scenario / packaging-dev-01 | 3/3 | 3/3 | 36.5% | met | fresh in that matrix |
| scenario / sheets-dev-01 | 3/3 | 1/3 | -142.4% | missed | fresh in that matrix |
| scenario / task-dev-01 | 3/3 | 3/3 | -51.3% | missed | fresh in that matrix |
| scaling / base-scale-dev-00100 | 0/3 | 3/3 | unavailable | unavailable | fresh in that matrix |
| scaling / base-scale-dev-01000 | 0/3 | 2/3 | unavailable | unavailable | fresh in that matrix |
| scaling / base-scale-dev-10000 | 0/3 | 3/3 | unavailable | unavailable | fresh in that matrix |
| scaling / sheets-size-dev-100 | 3/3 | 3/3 | 40.6% | met | fresh in that matrix |
| scaling / sheets-size-dev-1000 | 3/3 | 1/3 | unavailable | missed | fresh in that matrix |
| scaling / sheets-size-dev-10000 | 3/3 | 2/3 | -8.8% | missed | fresh in that matrix |
| business / business-dev-approval-flow | 3/3 | 3/3 | 30.2% | met | fresh in that matrix |
| business / business-dev-mindnote-edit | 3/3 | 2/3 | -146.5% | missed | fresh in that matrix |
| foundation / foundation-unconfigured-status | 3/3 | 3/3 | 5.6% | missed | fresh in that matrix |
| foundation / foundation-document-discovery | 0/3 | 0/3 | unavailable | unavailable | fresh in that matrix |
| contact / contact-dev-01 | 3/3 | 3/3 | 54.3% | met | fresh in that matrix |

## Source reports and evidence

Current reports: [initial business](live-business-luna-20261029.md), [extended business](live-business-extended-luna-20261029.md), [creation/media](live-collaboration-luna-20261029.md), and [complex workflows](live-complex-luna-20261029.md). The Contact diagnostic’s six original sanitized outcome/usage records are retained with this consolidation.

Consolidated [assessment](../../evidence/evaluation/evaluation-overview-20261029/assessment.json), [Contact records](../../evidence/evaluation/evaluation-overview-20261029/contact-attempts.json), and [source fingerprints](../../evidence/evaluation/evaluation-overview-20261029/README.md). Historical interpretation is also summarized by [the prior assessment](ASSESSMENT.md) and [matrix write-up](full-matrix.md). No retained evidence was overwritten and no held-out task file was read or used.
