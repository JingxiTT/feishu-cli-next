# Token reduction with original functionality preserved

This condition evaluates the token-efficiency changes requested after the
[first Luna pilot](evidence/paired-luna-medium-01/REPORT.md). Historical results
are immutable. No functions, formats, required evidence, or success checks may
be removed to reach the target.

## Comparison declared before model execution

- The same eight development tasks in `tasks.paired.json`, three repetitions,
  three arms, 72 attempts. All arms use the same frozen evaluator and fixtures.
- `official`: the original CLI and its complete domain skill from `7fd6ef3`.
- `next_before`: the first pilot's frozen Next CLI and entry skill from `7d28655`.
- `next`: the improved CLI and its shipped entry skill, frozen before launch.
- Model `gpt-5.6-luna`, reasoning `medium`, native Codex CLI `0.153.4`.
  Fresh contexts, serial randomized trials, 180-second agent and 210-second
  outer timeout; existing isolation, tools and provider accounting unchanged.
- The launcher retains its original three-arm mode by default. Supplying both
  `--previous-next-source` and `--previous-next-cli` selects the before/after
  condition for paired tasks. The concise baseline is not renamed or reused as
  evidence for an unchanged Next build.

The primary improvement target is **at least 50% fewer tokens per verified
success than `next_before`, for each function**. Also report the percentage
against `official` separately. Success must match or exceed the original and
previous Next on each task; data fidelity and absence of unintended effects
remain independently verified. No pooled percentage can satisfy this gate.

All attempt costs count, including failures and recovery. Input includes cached
input once; cached usage is separately reported. Incomplete usage, inconclusive
fixture outcomes, or zero baseline successes prevent a comparative target
claim. Do not replace missing scores with zero, omit difficult tasks, or treat
an unsupported fixture as a successful product workflow.

## Readiness and interpretation

Before billed trials: commit the complete changes atomically, run isolated
product and evaluator regressions, exercise both baseline binaries and the
candidate through native fixture controls, and record source/binary/guidance,
catalog, evaluator and native Codex hashes. Newly modeled backend behavior
requires independent native observations, not request-schema guesses. Validate
private synthetic probes and clean up only their recorded created resources.

Do not tune the candidate during a batch. Retain every attempted trial and any
missed target. Three repetitions are a development check, not statistical proof
of success-rate parity or a stable 50% improvement. A later balanced repeat or
fresh task variant is a new recorded condition, never a replacement for an
unfavorable result. This condition does not establish live tenant performance or
full original-functionality coverage beyond the compatibility tests and tasks.
