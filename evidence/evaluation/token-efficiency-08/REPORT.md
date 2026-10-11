# Token efficiency 08: filesystem skill handoff

The declared Docs-first stop rule stopped evaluation after **12 fresh attempts**; **36 remaining attempts across six functions were not run**.

Per-function gates: **1 met**, **1 missed**, **0 unavailable**, **6 not measured**. The target is strictly more than 50% lower all-attempt tokens per verified success for each Docs function, and at least 30% for each other function, with observed success nonregression versus original/full.

Retained outcomes: **12 verified**, **0 failed**, **0 inconclusive**, with **0 attempts having incomplete usage** out of 12 attempts. Native Codex CLI `0.153.4`, `gpt-5.6-luna`, reasoning `medium`.

This is a conditional development comparison, not an independent confirmation: Docs runs first and determines whether other functions are measured. Every measured arm has three fresh contexts on the same task. No older baselines, replacement attempts, pooled conditions or retrospective rescoring enter the scores. [Protocol](../../token-efficiency-08.md); [recorded stage decision](stage-decision.json).

Candidate `6366274` has a shorter entry guide; its Go runtime and native fidelity checks remain unchanged. Evaluator `6d478f1` delivers the selected skill by filesystem path only for both arms. Each arm is instructed to read the complete file in its measured tool loop; no inline copy or uncounted preloading substitutes for that read. Original skill bytes remain unchanged, and all trials/baselines here are fresh. This condition changes both entry length and delivery, so comparison with prior conditions is not a pure estimate of either factor.

Corrected ambient-skill suppression is retained for both arms. The [native no-model control](../codex-skill-isolation-native-20260909.json) and [05/06 qualification](../token-efficiency-05-06-skill-isolation-qualification.md) establish the bounded catalog finding, without assigning historical file reads or billed-token overhead to it. Reused validation of unchanged runtime code is separate from the new candidate build and embedded-guide checks; no candidate-binary identity reuse is claimed.

## Per-function results

Tokens per verified success include every attempt cost, including failed attempts and recovery. Unknown usage, any inconclusive outcome or zero verified successes makes the affected score unavailable. Exactly 50% does not pass a Docs gate; integer cross-multiplication decides the boundary. Observed success counts do not establish population success-rate parity.

| Function | Required saving | Original verified/attempts | Next verified/attempts | Original tokens/success | Next tokens/success | Saving | Combined gate |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| discovery | >=30% | not run | not run | — | — | — | not measured |
| contact | >=30% | not run | not run | — | — | — | not measured |
| docs-read | >50% | 3/3 | 3/3 | 40,293 | 22,435 | +44.3% | missed |
| docs-edit | >50% | 3/3 | 3/3 | 117,367 | 45,703 | +61.1% | met |
| markdown | >=30% | not run | not run | — | — | — | not measured |
| base | >=30% | not run | not run | — | — | — | not measured |
| sheets | >=30% | not run | not run | — | — | — | not measured |
| drive | >=30% | not run | not run | — | — | — | not measured |

**Secondary combined Docs workload:** original/full used 472,979 tokens with 6/6 verified; Next used 204,413 with 6/6 verified, a +56.8% saving. This sums three read and three edit attempts per arm, so the more expensive function contributes more weight. It describes this small fixed workload only and **cannot satisfy either per-function gate** or establish stable savings on other task mixes.


## Secondary cost and diagnostics

Mean tokens per attempt requires complete accounting but can be reported for zero-success arms. It cannot satisfy the primary successful-task gate. Shell counts and reported CLI errors are diagnostics, not per-command token attribution or backend correctness evidence.

| Measured function | Original mean tokens/attempt | Next mean tokens/attempt | Original median shell calls | Next median shell calls |
| --- | ---: | ---: | ---: | ---: |
| docs-read | 40,293 | 22,435 | 3 | 2 |
| docs-edit | 117,367 | 45,703 | 8 | 4 |

Observed usage for these **12 new attempts** totals **677,392 tokens**: 667,345 input plus 10,047 output. Input includes 490,752 cached tokens, counted once. Nonverified/inconclusive attempts consumed 0 observed tokens. Any incomplete usage makes the total a lower bound. Unrun trials have no measured cost, and pooled spend cannot satisfy per-function gates.

## Scope and limitations

- The candidate has shorter entry guidance; Contact identity routing is corrected; physical CRLF, BOM and blank block separators are tolerated, with a clearer missing-file recovery hint. Existing command paths and native write/fidelity algorithms remain available. A guidance size reduction is not itself a measured token reduction.
- Docs pull exposes a Markdown view and optional saved native sidecar. Push maps supported local text-only changes to native blocks and verifies readback, preserving IDs, marks, links and opaque regions. This is not universal Markdown roundtrip: leading blank lines before the first anchor and unsupported structural/rich changes remain bounded. Original document commands handle other workflows.
- Native snapshot checks are not atomic compare-and-swap: a stale-revision PATCH was accepted in separate native evidence. Reconciliation uses at most three read-only snapshots and never repeats PATCH. Unknown effects remain unknown. Private lifecycle checks, health probes, implementation and reviews are separate evidence, outside task success counts and measured provider spend.
- Original/full source and skills are unchanged. Fresh Markdown imports use the unchanged lark-drive skill; old lark-markdown excludes that workflow. Original historical native Docs GET/PATCH requests are validated before unsupported strategies become inconclusive; no historical outcomes are rescored.
- Signed synthetic fixtures independently verify required fidelity and unintended effects. Agent/outer budgets are 180/210 seconds. Three repetitions on fixed development tasks and a provider model alias do not prove stable savings, success parity, full original-functionality coverage or live-tenant performance.
- Traces retain command/flag names and finite error labels, without input values or private content. They cannot establish unrecorded local edits, full error text or final native state beyond retained verification evidence. Matched-success diagnostics exclude failures and cannot replace all-attempt costs.

## Evidence

[Frozen environment](environment.json), [readiness](readiness-review.json), [stage decision](stage-decision.json), and [exact summary](summary.json). New checkpoints: [docs](docs/batch-01.json). Prior conditions remain unchanged.
