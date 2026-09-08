# Paired function token comparison

Status: completed. The [report](evidence/paired-luna-medium-01/REPORT.md) records
all 72 trials: 54 verified successes, seven inconclusive outcomes, 11 failed
verifications and 8,442,585 provider tokens with complete usage. No trial was
removed or overwritten. Markdown coverage and some Sheets strategies prevent
complete comparisons; those limitations and their costs remain in the report.

The frozen evaluator is `596373b`, selected with `--task-kind paired`; product,
guidance, native Codex and catalog hashes are recorded in the evidence directory.

## Fixed settings

- Model: `gpt-5.6-luna`, explicitly selected by the user.
- Reasoning: `medium` for every arm and task.
- Agent harness: native Codex CLI `0.153.4`, invoked through `codex_runner.py`.
- Run mode: fresh `codex exec --json --ephemeral` sessions, with user config and
  rules ignored and the existing isolated shell/files environment retained.
- Arms: official CLI + official skill; the same official CLI + concise equivalent
  guidance; Next CLI + its single entry skill. No additional agent skills are
  injected. Selected reference files remain available on demand.
- Repetitions: three per task/arm, randomized order, serial provider execution.
- Initial scope: eight tasks, three arms, three repetitions = 72 attempts.
- Limits: equal 180-second agent budget and 210-second outer timeout initially;
  retain timeouts and their known/unknown usage. Predeclare any revised condition.

New launcher invocations must include
`--model gpt-5.6-luna --reasoning medium`. The adapter forwards the model through
`--model` and reasoning through `--config 'model_reasoning_effort="medium"'`.
It already collects provider usage from JSONL `turn.completed` events, as
documented by [Codex non-interactive mode](https://learn.chatgpt.com/docs/non-interactive-mode)
and the [configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference).
No model substitution is permitted if the selected model is unavailable. Local
configuration/process tests do not establish provider availability. Freeze binary,
complete guidance, evaluator, fixture and Codex hashes before billed trials; retain
the limitation that the provider model name may be an alias.

## Same-task comparison matrix

Every arm receives identical task wording, equivalent initial state and the same
required final result. Permit alternate command/API/file strategies.

| Function | Task and independent check | Pilot condition |
| --- | --- | --- |
| Discovery | Find a correct command to retrieve a user by ID | Evaluator validates returned read command after model exit |
| Docs read | Find a specified fact with its document/block reference | Independent fact and reference proof; no synthetic block-ID side channel |
| Docs edit | Replace a sentence while retaining bold/link and unrelated blocks | Plain replacement loses omitted styles; explicit rich content preserves them |
| Markdown create | Create headings, links, table and code; inspect native rendering | Bounded native rich subset; actual parsed content is checked |
| Sheets export | Export exact range to artifact, retaining raw types and explicit formulas | Separate explicit renderings, matching revisions; unverified defaults unsupported |
| Base update | Update two fields; preserve other fields and rows | Numeric zero and explicit JSON null; all unrelated fields preserved |
| Drive transfer | Upload/download identical file bytes | Signed 4 KiB upload/download, exact bytes, private current-account access |
| Contact search | Find exact user and bot identities | User and bot search with exact identity corroboration |

Historical manifests are separate from this eight-task pilot. Controlled
checks exercise both binaries and independent verifiers before model trials. The
[native-fidelity amendment](evidence/native-fidelity-validity-audit-20260908.md)
explains why old Docs/Sheets/Base assumptions cannot be reused unchanged. Keep
historical costs/verdicts intact. Do not access held-out tasks for development.

## Accounting and report

Count actual provider input plus output across the complete attempt, including
skill loading, help, tool responses and recovery. Input includes cached tokens;
report caching separately without subtracting or double-counting it. Retain failed
attempt costs. Unknown usage remains unknown, never zero. Tool output byte sizes
and local tokenizer estimates are diagnostics, not measured agent token usage.

Report each task separately: successes/attempts, total tokens, tokens per verified
success, matched-success token differences, distributions, API calls and elapsed
time. Tokens per success is all-attempt tokens divided by verified successes;
unknown usage or zero successes prevents a finite score. Do not treat reduced
fidelity or early failure as efficiency, pool unlike functions into one headline
percentage, or claim statistical sufficiency from three repeats.

After the pilot, add balanced repeats where variability prevents a conclusion,
then separately measure 100/1,000/10,000-row scaling, no-op, validation and uncertain
mutation recovery tasks. These are later conditions, not extra uncounted pilot work.
