# Token efficiency 05: Docs priority versus the original

The per-function tokens-per-success and observed success nonregression gate (>50% savings for Docs read/edit; >=30% elsewhere) was **not met**: **3/8 functions met**, **3 missed**, and **2 unavailable**.

Completed **48/48 fresh attempts**, with **40 verified successes**, **8 failed verifications**, **0 inconclusive outcomes**, and **4 attempts with incomplete usage**. Model: `gpt-5.6-luna`, reasoning `medium`; native Codex CLI `0.153.4`.

This condition compares only the original CLI with full domain guidance and the improved candidate: eight tasks, three repetitions, two arms. The user prioritizes strictly more than 50% lower tokens per verified success for both Docs read and Docs edit, and at least 30% for every other function, all versus original/full with no observed success regression. Exactly 50% does not pass either Docs gate. No previous-Next, concise-guidance or reused baseline trials enter the scores. [Declared protocol](../../token-efficiency-05.md).

Original Markdown guidance routing is corrected for this fresh condition: the task receives the unchanged full `lark-drive` skill because the original `lark-markdown` skill excludes Markdown-to-online-document import. Original skill bytes and the guidance tree are unchanged. Earlier reports and their outcomes remain immutable; this is not a retrospective rescore.

The candidate adds `docs +pull` for a Markdown reading view or a saved working copy with its native sidecar, and `docs +push` for supported local text-only edits mapped to the original native blocks. Existing IDs, styles, links and opaque rich blocks are retained through native block operations and verified readback. Original document commands remain available for other workflows; this is not a universal Markdown roundtrip or a new rich-format editor.

## Primary per-function result

Tokens per verified success include all attempt costs, including failures and recovery. Savings are positive for lower usage. Incomplete usage, inconclusive outcomes or zero verified successes make the affected score unavailable; a pooled percentage cannot satisfy the gate.

| Function | Required saving | Original tokens/success | Improved tokens/success | Saving vs original | Primary combined gate |
| --- | --- | ---: | ---: | ---: | --- |
| discovery | >=30% | 34,905 | 40,078 | -14.8% | missed |
| contact | >=30% | 56,745 | 23,740 | +58.2% | met |
| docs-read | >50% | 32,476 | 30,409 | +6.4% | missed |
| docs-edit | >50% | 109,365 | 297,492 | -172.0% | missed |
| markdown | >=30% | 144,689 | 56,906 | +60.7% | met |
| base | >=30% | 68,601 | 43,512 | +36.6% | met |
| sheets | >=30% | — | 65,434 | — | unavailable |
| drive | >=30% | — | — | — | unavailable |

## All outcomes

Entries are verified/attempts, with inconclusive counts where present. Failures stay in the denominator and cost accounting. These observed rates do not establish statistical success-rate parity.

| Function | Original/full | Next improved | Observed success nonregression |
| --- | ---: | ---: | --- |
| discovery | 3/3 | 3/3 | matched/exceeded |
| contact | 3/3 | 3/3 | matched/exceeded |
| docs-read | 3/3 | 3/3 | matched/exceeded |
| docs-edit | 3/3 | 2/3 | regressed |
| markdown | 3/3 | 3/3 | matched/exceeded |
| base | 3/3 | 3/3 | matched/exceeded |
| sheets | 0/3 | 3/3 | matched/exceeded |
| drive | 0/3 | 2/3 | matched/exceeded |

## Secondary cost per attempt

Mean cost per attempt is reported only with complete accounting, even if no attempt succeeded. Read it alongside the outcome table: cheap failed attempts do not establish successful-task efficiency. This secondary measure cannot satisfy the primary gate.

| Function | Original mean tokens/attempt | Improved mean tokens/attempt | Secondary saving |
| --- | ---: | ---: | ---: |
| discovery | 34,905 | 40,078 | -14.8% |
| contact | 56,745 | 23,740 | +58.2% |
| docs-read | 32,476 | 30,409 | +6.4% |
| docs-edit | 109,365 | 198,328 | -81.3% |
| markdown | 144,689 | 56,906 | +60.7% |
| base | 68,601 | 43,512 | +36.6% |
| sheets | 260,205 | 65,434 | +74.9% |
| drive | — | — | — |

## Accounting and workflow diagnostics

Observed provider spend across all 48 fresh attempts was **3,602,432 tokens**: 3,555,543 input and 46,889 output. Input includes 2,621,696 cached tokens, counted once. Nonverified/inconclusive attempts consumed 948,275 observed tokens. Totals are lower bounds if any usage is incomplete; pooled spend is not the target score.

Median shell tool executions below are diagnostic, not a token attribution or correctness measure. Command choices and matched-success pairs are retained in [summary.json](summary.json); matched-success subsets exclude failures and do not replace all-attempt tokens/success.

| Function | Original/full | Next improved |
| --- | ---: | ---: |
| discovery | 2 | 3 |
| contact | 4 | 1 |
| docs-read | 2 | 2 |
| docs-edit | 6 | 10 |
| markdown | 5 | 4 |
| base | 4 | 3 |
| sheets | 7 | 4 |
| drive | 1 | 3 |

## Failure evidence

| Function | Arm | Categories (overlapping counts) |
| --- | --- | --- |
| docs-edit | Next improved | misleading_success_claim: 1; outcome_failure: 1 |
| drive | Next improved | outcome_failure: 1; runner_failure: 1; usage_unavailable: 1 |
| drive | Original/full | outcome_failure: 3; runner_failure: 3; usage_unavailable: 3 |
| sheets | Original/full | misleading_success_claim: 3; outcome_failure: 3 |

Specific retained verification reasons are in summary.json. A fixture rejection alone does not identify a product defect; unsupported behavior remains inconclusive. No runtime verdict is reinterpreted or rescored.

Candidate Docs-edit traces include `docs +push` exit 2 followed by `docs +update` fallback. The retained traces omit local file contents and validation-error text, so the exact cause of those push rejections cannot be established. Final verifier outcomes, all fallback costs and failures remain unchanged. [Docs checkpoints](docs/batch-01.json).

## Conditions and limits

- Two fresh arms use the same frozen tasks, fixtures, tools and time budgets: 180 seconds for the agent and 210 seconds outside. Section schedules are randomized with three fresh contexts per task and arm. Candidate and evaluator remain frozen during execution.
- Real provider calls operate against signed synthetic Feishu fixtures. Required data fidelity and absence of unintended effects are independently checked. No original commands, formats or success requirements are removed to satisfy the target.
- The Markdown original-guidance correction changes this fresh baseline condition, while retaining exact original source bytes. Results are not pooled with earlier experiments. Native Docs historical block-list GET and block PATCH requests are validated for malformed inputs before being classified as an unmodeled historical-revision strategy; those valid but unsupported paths are inconclusive. DocsAI handling is unchanged, and no historical verdict is rescored.
- Native evidence showed a stale-revision PATCH can be accepted: snapshot checks detect observed conflicts but are not atomic compare-and-swap or a lock. Pull/push snapshot reconciliation allows at most three read-only attempts after revision drift; it never repeats a PATCH. Unknown write effects or unstable reads remain explicit outcomes, not automatic retries or success claims.
- Sanitized traces include the new public pull/push command names without input values. They describe agent choices, not per-command token attribution. Private native-probe outcomes and cleanup are separate readiness evidence, outside model success counts and provider spend; earlier unsuccessful probes are not counted as passed.
- Three repetitions on fixed development tasks are point estimates, not proof of stable savings at the requested thresholds, statistical success parity, full original-functionality coverage or live-tenant performance. Small documents/ranges and a 4 KiB Drive file do not establish large-workload behavior. The model alias has no immutable provider snapshot.
- API counts are signed fixture requests, not live Feishu calls; Discovery verifier-only requests are excluded. Cached input is counted once in full. Implementation, native probes and reviews are outside the reported spend.
- No raw credentials, private tenant content or model transcripts are included. All prior evidence remains unchanged.

## Evidence

[Frozen environment](environment.json) records source, binary, guidance, catalog, evaluator and Codex hashes. [Readiness review](readiness-review.json) records prelaunch controls. [Machine-readable summary](summary.json) retains exact aggregates, gates, diagnostics and source hashes. Every attempt remains in its checkpoint: [contact](contact/batch-01.json), [docs](docs/batch-01.json), [markdown](markdown/batch-01.json), [base](base/batch-01.json), [sheets](sheets/batch-01.json), [drive](drive/batch-01.json).
