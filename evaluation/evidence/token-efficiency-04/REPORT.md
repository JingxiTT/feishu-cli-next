# Token efficiency 04: exploratory candidate-only follow-up

The exploratory per-function 50% point-estimate and observed success nonregression gate was **not met**: **5/8 functions met**, **3 missed**, and **0 unavailable**.

Completed **24/24 new candidate attempts**, with **24 verified successes**, **0 failed verifications**, **0 inconclusive outcomes**, and **0 attempts with incomplete usage**. These are compared with **48 reused baseline attempts from condition 03**, not 72 new model calls. Model: `gpt-5.6-luna`, reasoning `medium`; native Codex CLI `0.153.4`.

The primary point estimate compares the new candidate with all unchanged-Next attempts from condition 03; original/full is a separate comparator. Baselines were run earlier, so timing, cache effects and the provider alias may differ. Trial indices are not contemporaneous pairs. [Declared protocol](../../token-efficiency-04.md).

Earlier evidence remains immutable: [condition 02](../token-efficiency-02/REPORT.md) met four of eight combined gates; [condition 03](../token-efficiency-03/REPORT.md) met two. This follow-up was selected after those results. All condition-03 original/full and Next-before attempts are reused, including failures and incomplete accounting; no condition-02 trials or condition-03 candidate trials enter the scores below.

## Per-function tokens and target

Tokens per verified success include all attempt costs, including failures. Savings are positive for lower usage. A dash means inconclusive outcomes, incomplete usage, or zero successes prevent that score or comparison. The gate also requires observed success nonregression against both baselines.

| Function | Original tokens/success | Before tokens/success | Improved tokens/success | Saving vs before | Saving vs original | Combined gate |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| discovery | 39,622 | 51,454 | 24,422 | +52.5% | +38.4% | met |
| contact | 49,062 | 104,322 | 32,485 | +68.9% | +33.8% | met |
| docs-read | 37,523 | 65,643 | 39,718 | +39.5% | -5.8% | missed |
| docs-edit | — | 211,647 | 77,543 | +63.4% | — | met |
| markdown | — | 115,011 | 65,399 | +43.1% | — | missed |
| base | 82,103 | 165,934 | 43,015 | +74.1% | +47.6% | met |
| sheets | — | 94,588 | 41,940 | +55.7% | — | met |
| drive | 136,426 | 100,234 | 60,146 | +40.0% | +55.9% | missed |

## Verified outcomes

Entries are verified/attempts, followed by inconclusive counts when present. Inconclusive attempts remain unresolved; three repetitions cannot establish statistical success-rate parity.

| Function | Original/full | Next before | Next improved | Success nonregression vs both |
| --- | ---: | ---: | ---: | --- |
| discovery | 3/3 | 3/3 | 3/3 | matched/exceeded |
| contact | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-read | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-edit | 2/3 | 2/3 | 3/3 | matched/exceeded |
| markdown | 0/3 | 3/3 | 3/3 | matched/exceeded |
| base | 3/3 | 2/3 | 3/3 | matched/exceeded |
| sheets | 0/3 | 3/3 | 3/3 | matched/exceeded |
| drive | 3/3 | 3/3 | 3/3 | matched/exceeded |

## Accounting and workflow diagnostics

The **24 new candidate attempts consumed 1,154,003 observed tokens**. The comparison dataset, including reused baseline usage, contains 6,876,132 observed tokens: 6,794,252 input and 81,880 output. Input includes 5,024,000 cached tokens, counted once. Nonverified/inconclusive attempts across the comparison dataset account for 2,334,582 observed tokens. Reused usage is not new spend. Incomplete original Docs-edit accounting makes its token comparison unavailable; these combined totals are lower bounds and pooled spend is not the target score.

Median shell tool executions below are diagnostic, not a token attribution or correctness measure. Command choices are retained in [summary.json](summary.json). No contemporaneous matched-pair result is claimed from reused trial indices.

| Function | Original/full | Next before | Next improved |
| --- | ---: | ---: | ---: |
| discovery | 2 | 4 | 2 |
| contact | 4 | 6 | 3 |
| docs-read | 2 | 5 | 3 |
| docs-edit | 6 | 9 | 4 |
| markdown | 15 | 7 | 4 |
| base | 3 | 8 | 2 |
| sheets | 8 | 6 | 3 |
| drive | 5 | 5 | 4 |

## Failure evidence

| Function | Arm | Categories (overlapping counts) |
| --- | --- | --- |
| base | Next before | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Next before | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Original/full | outcome_failure: 1; runner_failure: 1; usage_unavailable: 1 |
| markdown | Original/full | misleading_success_claim: 3; outcome_failure: 3 |
| sheets | Original/full | misleading_success_claim: 3; outcome_failure: 3 |

Specific retained verification reasons are in summary.json. A fixture rejection alone does not establish a product defect; unsupported native behavior stays inconclusive. The report does not reinterpret runtime verdicts or infer causes from missing diagnostics.

## Conditions and limits

- Eight fixed development tasks and three new candidate repetitions each. The launcher filters candidate entries after shuffling the existing balanced schedule. The 48 baseline attempts come from the earlier condition-03 run; this is not a new balanced experiment.
- Runner, verifier, fixtures, task manifest, tools, model alias, reasoning and time budgets match condition 03 by recorded hashes and metadata. Candidate guidance and evaluator orchestration are frozen before the new attempts. The reused baseline binary and guidance provenance remains condition 03.
- Real provider model calls operate against signed synthetic Feishu fixtures. Native controls inform fixture semantics, but these results are not live-tenant performance or exhaustive proof of original-functionality compatibility. No APIs, formats or success requirements were removed to satisfy the target.
- Reused baselines, selection after earlier outcomes, cache effects and the unpinned model alias limit causal attribution. Three repetitions cannot establish a stable 50% reduction or statistical success parity. Small fixed workloads do not establish large-workload behavior.
- The bounded Docs rejection/recovery limitation documented in [condition 03](../token-efficiency-03/REPORT.md) remains unchanged. An ordinary rejected request can fail verification despite later CLI recovery. Retained evidence cannot identify every exact cause or prove final native state; no historical verdict is rescored.
- API counts are signed fixture requests, not live Feishu calls; Discovery verifier-only requests are excluded. The report separates new candidate calls from all reused baseline costs and preserves unknown accounting, zero-success baselines and failures.
- Counts exclude implementation, private native probes and reviews. Cached input is fully counted once. No raw credentials, private tenant content or model transcripts are included.

## Evidence

[Frozen environment](environment.json) records new source hashes plus the reused condition-03 environment and exact checkpoint hashes. [Readiness review](readiness-review.json) records the prelaunch controls. [Machine-readable summary](summary.json) retains exact aggregates, gates, command diagnostics and separate new/reused source hashes. New candidate checkpoints retain every new attempt: [contact](contact/batch-01.json), [docs](docs/batch-01.json), [markdown](markdown/batch-01.json), [base](base/batch-01.json), [sheets](sheets/batch-01.json), [drive](drive/batch-01.json). Reused baselines are read directly from the immutable [condition-03 evidence](../token-efficiency-03/summary.json).
