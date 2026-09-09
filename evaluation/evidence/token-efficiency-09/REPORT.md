# Token efficiency 09: harder Docs workloads

All **18 fresh attempts** completed. The per-task strict >50% saving plus observed success nonregression target is **not met**: 0 met, 2 missed and 1 unavailable.

Retained outcomes: **17/18 verified**, 0 failed verifications, 1 inconclusive outcomes and 0 attempts with incomplete usage.

Three distinct harder Docs tasks compare original/full with the candidate, using three fresh repetitions per task and arm. Every scheduled attempt remains; there is no performance-based early stop between tasks. Native Codex CLI 0.153.4 uses gpt-5.6-luna at medium reasoning with shared filesystem-only guidance delivery. [Protocol](../../token-efficiency-09.md).

This is a new development cohort with table/list reading, focused retrieval from a larger document and multiple styled wording edits. It is not the fixed condition-08 cohort. No historical baseline or candidate attempts enter the scores, and differences between cohorts are not an improvement percentage.

## Per-task results

Tokens per verified success include all observed attempt, failure and recovery costs. Unknown usage, inconclusive outcomes or zero successes make the affected token score unavailable. Strict >50% uses integer cross-multiplication; exactly 50% fails. Observed success counts are not population success-rate parity.

| Task | Original verified/attempts | Next verified/attempts | Original tokens/success | Next tokens/success | Saving | Gate |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| docs-hard-rich-read | 3/3 | 3/3 | 38,588 | 23,865 | +38.2% | missed |
| docs-hard-focused-read | 3/3 | 3/3 | 36,630 | 25,224 | +31.1% | missed |
| docs-hard-multi-edit | 2/3 (1 inconclusive) | 3/3 | — | 54,036 | — | unavailable |

## Secondary diagnostics

Mean cost per attempt, matched-success pairs, API counts and command choices cannot satisfy a primary gate. Retained public commands and finite error labels do not allocate provider tokens to individual commands or recover omitted inputs.

| Task | Original mean tokens/attempt | Next mean tokens/attempt | Original median shell calls | Next median shell calls |
| --- | ---: | ---: | ---: | ---: |
| docs-hard-rich-read | 38,588 | 23,865 | 3 | 2 |
| docs-hard-focused-read | 36,630 | 25,224 | 3 | 2 |
| docs-hard-multi-edit | 122,218 | 54,036 | 7 | 4 |

Observed usage is **901,686 tokens**: 887,153 input plus 14,533 output. Input includes 550,144 cached tokens, counted once in full. Nonverified/inconclusive attempts account for 185,209 observed tokens. If any usage is incomplete, spend is a lower bound; no pooled percentage satisfies an individual task gate.

## Retained failure categories

| Task | Arm | Categories (overlapping counts) |
| --- | --- | --- |
| docs-hard-multi-edit | Original/full | fixture_unsupported: 1 |

## Conditions and limits

- Frozen candidate `cad221e` and evaluator `b28354f` use identical task state, 180/210-second agent/outer budgets and seeded balanced scheduling. Original source and full Docs guide bytes are unchanged; both arms read selected guidance inside the measured loop.
- Readable output, raw native blocks, full pagination and supported focused reads are legitimate alternatives. No command sequence, request count or exact Markdown whitespace is a success requirement.
- Native table/list shapes and selection behavior are bounded synthetic contracts. Source-backed list projections are not newly live-observed behavior. Readable rendering and product lifecycle controls are separate from task success counts.
- Multiple edits must preserve required native marks, link destinations and unrelated state. Native revision checks are not compare-and-swap; a successful synthetic trial does not establish live atomicity or universal rich-content editing.
- Complete guidance reading is a prompt requirement, not an added hidden success predicate. Corrected ambient-skill suppression applies to both arms. Skill copying is preparation, not a substitute for measured agent reads.
- Unknown or missing verification detail cannot identify a product cause. Unsupported valid strategies may be inconclusive; failed-attempt costs and all runtime classifications remain unchanged.
- Three repetitions and one larger document size do not establish stable savings, statistical success parity, scaling behavior, live-tenant performance or full original-functionality coverage. The provider model alias has no immutable snapshot.
- Original and candidate functionality and guidance differ together. No earlier conditions are pooled, rescored or used as fresh baselines. Reviewer, implementation and native-control work is outside measured provider spend.

## Evidence

[Frozen environment](environment.json), [readiness](readiness-review.json), [exact summary](summary.json), and [all18 retained attempts](docs/batch-01.json).
