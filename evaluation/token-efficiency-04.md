# Candidate-only guidance follow-up

This exploratory condition measures a small guidance correction after the
completed [second balanced run](evidence/token-efficiency-03/REPORT.md).
It contains **24 new candidate trials compared with 48 reused condition-03
baseline trials**, not a new balanced 72-trial experiment.

The candidate restores direct use of provided command guidance and makes small
stdout responses versus required/large saved artifacts explicit. It retains
all command routes, functionality, original skills, data fidelity checks and
applicable higher-priority instructions. Extra help and local reads observed in
condition 03 motivate this correction; the wording's causal effect is unproven.

Before launch, freeze the candidate, evaluator orchestration and all source,
binary, guidance and task hashes. The runner, verifier, fixtures, task manifest,
tools, timeouts, model alias GPT-5.6 Luna and medium reasoning remain identical
to condition 03. A narrow `--only-arm next` option filters the existing balanced
schedule after shuffling: eight tasks, three fresh candidate contexts per task.
Default balanced execution remains unchanged.

Reuse every original/full and Next-before attempt from condition 03, including
failures and incomplete accounting. Record their original condition and source
hashes separately from new evidence. Do not substitute condition-02 results or
count copied checkpoints as new model calls. No trial-index pairing is presented
as contemporaneous evidence.

The exploratory primary point-estimate target remains at least 50% fewer
all-attempt tokens per verified success than condition-03 Next-before for each
function, with no observed success regression against either reused baseline.
Report the original/full comparison separately. Retain every new failure,
recovery cost and missed target. Unknown usage, inconclusive outcomes and zero
successes make affected comparisons unavailable; do not replace them with zero.
In particular, the original Docs-edit comparison has incomplete usage, and the
original Markdown baseline has zero verified successes.

This follow-up was selected after earlier results. Baseline timing, cache effects
and an unpinned provider model alias may differ. Three repetitions and reused
baselines cannot establish stable 50% savings, statistical success parity or
comprehensive live functionality. Earlier reports remain immutable, and the
documented condition-03 Docs rejection/recovery limitation remains unchanged.
