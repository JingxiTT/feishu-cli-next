# Token efficiency 02: same-task comparison

The per-function 50% reduction and success nonregression gate was **not met**: **4/8 functions met**, **4 missed**, and **0 unavailable**.

Completed **72/72 attempts** with **64 verified successes**, **7 failed verifications**, **1 inconclusive outcome(s)**, and **0 attempts with incomplete usage**. Model: `gpt-5.6-luna`, reasoning `medium`; native Codex CLI `0.153.4`.

The primary comparison is improved Next against unchanged Next, both rerun in this condition. Original/full is a separate compatibility and token comparator. These are fresh same-task trials under one frozen evaluator; no historical trial costs or outcomes are mixed into the results. [Declared protocol](../../token-efficiency-02.md).

## Per-function tokens and target

Tokens per verified success include all attempt costs, including failures. Savings are positive for lower usage. A dash means inconclusive outcomes, incomplete usage, or zero successes prevent that score or comparison. The gate also requires observed success nonregression against both baselines.

| Function | Original tokens/success | Before tokens/success | Improved tokens/success | Saving vs before | Saving vs original | Combined gate |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| discovery | 47,265 | 51,704 | 20,510 | +60.3% | +56.6% | met |
| contact | 48,524 | 70,669 | 22,244 | +68.5% | +54.2% | met |
| docs-read | 44,714 | 57,415 | 18,452 | +67.9% | +58.7% | met |
| docs-edit | — | 139,382 | 77,564 | +44.4% | — | missed |
| markdown | — | 364,891 | 70,672 | +80.6% | — | met |
| base | 128,349 | 102,907 | 92,604 | +10.0% | +27.8% | missed |
| sheets | 810,413 | 75,340 | 41,105 | +45.4% | +94.9% | missed |
| drive | 153,082 | 82,355 | 58,745 | +28.7% | +61.6% | missed |

## Verified outcomes

Entries are verified/attempts, followed by inconclusive counts when present. Inconclusive attempts remain unresolved; three repetitions cannot establish statistical success-rate parity.

| Function | Original/full | Next before | Next improved | Success nonregression vs both |
| --- | ---: | ---: | ---: | --- |
| discovery | 3/3 | 3/3 | 3/3 | matched/exceeded |
| contact | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-read | 3/3 | 3/3 | 3/3 | matched/exceeded |
| docs-edit | 2/3; 1 inconclusive | 3/3 | 2/3 | regressed |
| markdown | 0/3 | 2/3 | 3/3 | matched/exceeded |
| base | 3/3 | 3/3 | 3/3 | matched/exceeded |
| sheets | 1/3 | 3/3 | 3/3 | matched/exceeded |
| drive | 3/3 | 3/3 | 3/3 | matched/exceeded |

## Accounting and workflow diagnostics

Observed provider spend was **7,218,218 tokens**: 7,131,783 input and 86,435 output. Input includes 5,254,144 cached tokens, counted once. Nonverified/inconclusive attempts consumed 2,336,230 observed tokens. Totals are lower bounds if any usage is incomplete; pooled spend is not the target score.

Median shell tool executions below are diagnostic, not a token attribution or correctness measure. Command choices and matched-success pairs are retained in [summary.json](summary.json); paired-success comparisons exclude failures and do not replace all-attempt tokens/success.

| Function | Original/full | Next before | Next improved |
| --- | ---: | ---: | ---: |
| discovery | 2 | 4 | 0 |
| contact | 4 | 6 | 1 |
| docs-read | 2 | 5 | 1 |
| docs-edit | 5 | 8 | 3 |
| markdown | 15 | 7 | 5 |
| base | 6 | 7 | 7 |
| sheets | 7 | 5 | 2 |
| drive | 6 | 6 | 5 |

## Failure evidence

| Function | Arm | Categories (overlapping counts) |
| --- | --- | --- |
| docs-edit | Next improved | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Original/full | fixture_unsupported: 1 |
| markdown | Next before | misleading_success_claim: 1; outcome_failure: 1 |
| markdown | Original/full | misleading_success_claim: 3; outcome_failure: 3 |
| sheets | Original/full | misleading_success_claim: 2; outcome_failure: 2 |

Specific retained verification reasons are in summary.json. A fixture rejection alone does not establish a product defect; unsupported native behavior stays inconclusive. The report does not reinterpret runtime verdicts or infer causes from missing diagnostics.

## Conditions and limits

- Eight fixed development tasks, three repetitions and three arms; serial randomized trials, fresh contexts, equal tools, 180-second agent and 210-second outer budgets. The candidate was frozen before execution and was not tuned during the batch.
- Real provider model calls operate against signed synthetic Feishu fixtures. Native controls inform fixture semantics, but these results are not live-tenant performance or exhaustive proof of original-functionality compatibility. No APIs, formats or success requirements were removed to satisfy the target.
- The same original and previous-Next binaries are rerun alongside the improved candidate. Only the current frozen condition supports the comparisons above. Broader Markdown/Sheets fixture coverage does not retroactively rescore historical results.
- Three trials are a development check, not proof of a stable 50% reduction. Small fixed documents, ranges and a 4 KiB Drive file do not establish large-workload behavior. The requested model is a provider alias without an immutable snapshot.
- API counts are signed fixture requests, not live Feishu calls; Discovery verifier-only requests are excluded from model API counts. Success preserves task-specific data fidelity and absence of unintended changes through independent verification.
- Counts cover measured agent attempts only, excluding implementation, private native probes and reviews. Prompt-cache input remains fully counted. No raw credentials, private tenant content or model transcripts are included.

## Evidence

[Frozen environment](environment.json) records source, binary, guidance, catalog, evaluator and Codex hashes. [Readiness review](readiness-review.json) records the prelaunch controls. [Machine-readable summary](summary.json) retains exact aggregates, gates, matched pairs, command diagnostics and source hashes. Per-section checkpoints retain every attempted result: [contact](contact/batch-01.json), [docs](docs/batch-01.json), [markdown](markdown/batch-01.json), [base](base/batch-01.json), [sheets](sheets/batch-01.json), [drive](drive/batch-01.json).
