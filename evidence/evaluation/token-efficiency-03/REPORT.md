# Token efficiency 03: second-pass same-task comparison

The per-function 50% reduction and success nonregression gate was **not met**: **2/8 functions met**, **6 missed**, and **0 unavailable**.

Completed **72/72 attempts** with **62 verified successes**, **10 failed verifications**, **0 inconclusive outcome(s)**, and **1 attempt(s) with incomplete usage**. Model: `gpt-5.6-luna`, reasoning `medium`; native Codex CLI `0.153.4`.

The primary comparison is improved Next against unchanged Next, both rerun in this condition. Original/full is a separate compatibility and token comparator. These are fresh same-task trials under one frozen evaluator; no historical trial costs or outcomes are mixed into the results. [Declared protocol](../../token-efficiency-03.md).

This is the second optimization pass. [Condition 02](../token-efficiency-02/REPORT.md) remains unchanged: four of eight functions met its combined gate; Docs edit, Base, Sheets and Drive missed, with a Docs edit success regression. The new pass reruns all three arms; earlier results are context only and do not contribute to any score below.

## Per-function tokens and target

Tokens per verified success include all attempt costs, including failures. Savings are positive for lower usage. A dash means inconclusive outcomes, incomplete usage, or zero successes prevent that score or comparison. The gate also requires observed success nonregression against both baselines.

| Function | Original tokens/success | Before tokens/success | Improved tokens/success | Saving vs before | Saving vs original | Combined gate |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| discovery | 39,622 | 51,454 | 32,030 | +37.7% | +19.2% | missed |
| contact | 49,062 | 104,322 | 45,840 | +56.1% | +6.6% | met |
| docs-read | 37,523 | 65,643 | 43,806 | +33.3% | -16.7% | missed |
| docs-edit | — | 211,647 | 167,404 | +20.9% | — | missed |
| markdown | — | 115,011 | 83,786 | +27.2% | — | missed |
| base | 82,103 | 165,934 | 75,469 | +54.5% | +8.1% | met |
| sheets | — | 94,588 | 59,371 | +37.2% | — | missed |
| drive | 136,426 | 100,234 | 57,217 | +42.9% | +58.1% | missed |

## Verified outcomes

Entries are verified/attempts, followed by inconclusive counts when present. Inconclusive attempts remain unresolved; three repetitions cannot establish statistical success-rate parity.

| Function | Original/full | Next before | Next improved | Success nonregression vs both |
| --- | ---: | ---: | ---: | --- |
| discovery | 3/3 | 3/3 | 3/3 | matched/exceeded |
| contact | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-read | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-edit | 2/3 | 2/3 | 2/3 | matched/exceeded |
| markdown | 0/3 | 3/3 | 3/3 | matched/exceeded |
| base | 3/3 | 2/3 | 3/3 | matched/exceeded |
| sheets | 0/3 | 3/3 | 3/3 | matched/exceeded |
| drive | 3/3 | 3/3 | 3/3 | matched/exceeded |

## Accounting and workflow diagnostics

Observed provider spend was **7,249,494 tokens**: 7,162,695 input and 86,799 output. Input includes 5,324,032 cached tokens, counted once. Nonverified/inconclusive attempts consumed 2,469,237 observed tokens. Totals are lower bounds if any usage is incomplete; pooled spend is not the target score.

Median shell tool executions below are diagnostic, not a token attribution or correctness measure. Command choices and matched-success pairs are retained in [summary.json](summary.json); paired-success comparisons exclude failures and do not replace all-attempt tokens/success.

| Function | Original/full | Next before | Next improved |
| --- | ---: | ---: | ---: |
| discovery | 2 | 4 | 2 |
| contact | 4 | 6 | 3 |
| docs-read | 2 | 5 | 3 |
| docs-edit | 6 | 9 | 7 |
| markdown | 15 | 7 | 6 |
| base | 3 | 8 | 5 |
| sheets | 8 | 6 | 4 |
| drive | 5 | 5 | 4 |

## Failure evidence

| Function | Arm | Categories (overlapping counts) |
| --- | --- | --- |
| base | Next before | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Next improved | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Next before | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Original/full | outcome_failure: 1; runner_failure: 1; usage_unavailable: 1 |
| markdown | Original/full | misleading_success_claim: 3; outcome_failure: 3 |
| sheets | Original/full | misleading_success_claim: 3; outcome_failure: 3 |

Specific retained verification reasons are in summary.json. A fixture rejection alone does not establish a product defect; unsupported native behavior stays inconclusive. The report does not reinterpret runtime verdicts or infer causes from missing diagnostics.

## Conditions and limits

- Eight fixed development tasks, three repetitions and three arms; serial randomized trials, fresh contexts, equal tools, 180-second agent and 210-second outer budgets. The candidate was frozen before execution and was not tuned during the batch.
- Real provider model calls operate against signed synthetic Feishu fixtures. Native controls inform fixture semantics, but these results are not live-tenant performance or exhaustive proof of original-functionality compatibility. No APIs, formats or success requirements were removed to satisfy the target.
- The same original and previous-Next binaries are rerun alongside the improved candidate. Only the current frozen condition supports the comparisons above. Broader Markdown/Sheets fixture coverage does not retroactively rescore historical results.
- Three trials are a development check, not proof of a stable 50% reduction. Small fixed documents, ranges and a 4 KiB Drive file do not establish large-workload behavior. The requested model is a provider alias without an immutable snapshot.
- API counts are signed fixture requests, not live Feishu calls; Discovery verifier-only requests are excluded from model API counts. Success preserves task-specific data fidelity and absence of unintended changes through independent verification.
- Counts cover measured agent attempts only, excluding implementation, private native probes and reviews. Prompt-cache input remains fully counted. No raw credentials, private tenant content or model transcripts are included.
- Docs edit candidate trial 0 remains a failed verification with `request_outside_fixture`. The retained trace records an initial update exit 20 followed by read/recovery commands and a later update exit 0. The fixture maps ordinary validation exceptions, including stale revisions, to a rejected request; the verifier rejects any such historical request even after recovery. This is one possible explanation, not an identified cause. Neither an API request trace nor final native state was retained, so exact failure cause and final correctness cannot be recovered or rescored. Sources at the frozen evaluator revision: [revision check](https://github.com/Zhujingxi/feishu-cli-next/blob/9ac6c4b377bcf2eb563274e83ecf1ab0446e78fd/evaluation/scenarios/docs.py#L220), [rejection mapping](https://github.com/Zhujingxi/feishu-cli-next/blob/9ac6c4b377bcf2eb563274e83ecf1ab0446e78fd/evaluation/section_fixture.py#L146), [verifier](https://github.com/Zhujingxi/feishu-cli-next/blob/9ac6c4b377bcf2eb563274e83ecf1ab0446e78fd/evaluation/scenario_verifier.py#L35), and [retained trial](docs/batch-01.json).

## Evidence

[Frozen environment](environment.json) records source, binary, guidance, catalog, evaluator and Codex hashes. [Readiness review](readiness-review.json) records the prelaunch controls. [Machine-readable summary](summary.json) retains exact aggregates, gates, matched pairs, command diagnostics and source hashes. Per-section checkpoints retain every attempted result: [contact](contact/batch-01.json), [docs](docs/batch-01.json), [markdown](markdown/batch-01.json), [base](base/batch-01.json), [sheets](sheets/batch-01.json), [drive](drive/batch-01.json).
