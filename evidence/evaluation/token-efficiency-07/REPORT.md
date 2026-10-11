# Token efficiency 07: corrected skill isolation

The declared Docs-first stop rule stopped evaluation after **12 fresh attempts**; **36 remaining attempts across six functions were not run**.

Per-function gates: **1 met**, **1 missed**, **0 unavailable**, **6 not measured**. The target is strictly more than 50% lower all-attempt tokens per verified success for each Docs function, and at least 30% for each other function, with observed success nonregression versus original/full.

Retained outcomes: **11 verified**, **1 failed**, **0 inconclusive**, with **0 attempts having incomplete usage** out of 12 attempts. Native Codex CLI `0.153.4`, `gpt-5.6-luna`, reasoning `medium`.

This is a conditional development comparison, not an independent confirmation: Docs runs first and determines whether other functions are measured. Every measured arm has three fresh contexts on the same task. No older baselines, replacement attempts, pooled conditions or retrospective rescoring enter the scores. [Protocol](../../token-efficiency-07.md); [recorded stage decision](stage-decision.json).

Candidate `dd0fc7a` uses the exact source, binary and selected guidance from condition06. Evaluator `5c86d01` corrects disabled-skill paths to `SKILL.md` entry files for both arms. Only product validation is reused; every trial and baseline in this condition is fresh. [Native no-model control](../codex-skill-isolation-native-20260909.json) confirms the corrected catalog behavior; the [05/06 qualification](../token-efficiency-05-06-skill-isolation-qualification.md) preserves historical limitations. Differences from earlier conditions cannot be attributed solely to product changes or used to quantify the prior catalog overhead.

## Per-function results

Tokens per verified success include every attempt cost, including failed attempts and recovery. Unknown usage, any inconclusive outcome or zero verified successes makes the affected score unavailable. Exactly 50% does not pass a Docs gate; integer cross-multiplication decides the boundary. Observed success counts do not establish population success-rate parity.

| Function | Required saving | Original verified/attempts | Next verified/attempts | Original tokens/success | Next tokens/success | Saving | Combined gate |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| discovery | >=30% | not run | not run | — | — | — | not measured |
| contact | >=30% | not run | not run | — | — | — | not measured |
| docs-read | >50% | 3/3 | 3/3 | 34,792 | 22,215 | +36.1% | missed |
| docs-edit | >50% | 2/3 | 3/3 | 183,832 | 60,670 | +67.0% | met |
| markdown | >=30% | not run | not run | — | — | — | not measured |
| base | >=30% | not run | not run | — | — | — | not measured |
| sheets | >=30% | not run | not run | — | — | — | not measured |
| drive | >=30% | not run | not run | — | — | — | not measured |

## Secondary cost and diagnostics

Mean tokens per attempt requires complete accounting but can be reported for zero-success arms. It cannot satisfy the primary successful-task gate. Shell counts and reported CLI errors are diagnostics, not per-command token attribution or backend correctness evidence.

| Measured function | Original mean tokens/attempt | Next mean tokens/attempt | Original median shell calls | Next median shell calls |
| --- | ---: | ---: | ---: | ---: |
| docs-read | 34,792 | 22,215 | 2 | 2 |
| docs-edit | 122,555 | 60,670 | 7 | 5 |

Observed usage for these **12 new attempts** totals **720,698 tokens**: 709,779 input plus 10,919 output. Input includes 503,552 cached tokens, counted once. Nonverified/inconclusive attempts consumed 88,429 observed tokens. Any incomplete usage makes the total a lower bound. Unrun trials have no measured cost, and pooled spend cannot satisfy per-function gates.

## Retained failures

| Function | Arm | Categories (counts can overlap) |
| --- | --- | --- |
| paired-docs-edit | Original/full | misleading_success_claim: 1; outcome_failure: 1 |

## Scope and limitations

- The unchanged candidate has compressed entry guidance; Contact identity routing is corrected; physical CRLF, BOM and blank block separators are tolerated, with a clearer missing-file recovery hint. Existing command paths and native write/fidelity algorithms remain available. A guidance size reduction is not itself a measured token reduction.
- Docs pull exposes a Markdown view and optional saved native sidecar. Push maps supported local text-only changes to native blocks and verifies readback, preserving IDs, marks, links and opaque regions. This is not universal Markdown roundtrip: leading blank lines before the first anchor and unsupported structural/rich changes remain bounded. Original document commands handle other workflows.
- Native snapshot checks are not atomic compare-and-swap: a stale-revision PATCH was accepted in separate native evidence. Reconciliation uses at most three read-only snapshots and never repeats PATCH. Unknown effects remain unknown. Private lifecycle checks, health probes, implementation and reviews are separate evidence, outside task success counts and measured provider spend.
- Original/full source and skills are unchanged. Fresh Markdown imports use the unchanged lark-drive skill; old lark-markdown excludes that workflow. Original historical native Docs GET/PATCH requests are validated before unsupported strategies become inconclusive; no historical outcomes are rescored.
- Signed synthetic fixtures independently verify required fidelity and unintended effects. Agent/outer budgets are 180/210 seconds. Three repetitions on fixed development tasks and a provider model alias do not prove stable savings, success parity, full original-functionality coverage or live-tenant performance.
- Traces retain command/flag names and finite error labels, without input values or private content. They cannot establish unrecorded local edits, full error text or final native state beyond retained verification evidence. Matched-success diagnostics exclude failures and cannot replace all-attempt costs.

## Evidence

[Frozen environment](environment.json), [readiness](readiness-review.json), [stage decision](stage-decision.json), and [exact summary](summary.json). New checkpoints: [docs](docs/batch-01.json). Prior conditions remain unchanged.
