# Function-specific token reduction versus the original Feishu implementation

The current target is **strictly more than 50% lower token usage for Docs read
and Docs edit**, and **at least 30% lower for every other evaluated function**,
versus the original implementation. This condition replaces the previous
Next-before comparison for the new decision; it does not rewrite historical
results.

**Evaluation is on hold.** The local Markdown workflow is pending product work
and validation. The requirements below are acceptance criteria, not a claim that
the workflow is implemented or ready. Do not launch model trials until the
candidate, native behavior controls and evaluator are ready and frozen.

## Planned comparison

- Run 48 fresh attempts: the same eight development tasks, three repetitions,
  two arms. `official` is the original CLI at `7fd6ef3` with its full domain
  guidance; `next` is the improved candidate and its shipped entry skill.
- Use GPT-5.6 Luna, medium reasoning, native Codex CLI 0.153.4, fresh contexts,
  the existing randomized section schedules, 180-second agent and 210-second
  outer timeouts. Record exact source, binary, guidance and evaluator hashes.
- Both arms use the same signed synthetic fixtures and independent verifier.
  Preserve every original command, format and required result. Do not modify
  original guidance to help the baseline or weaken checks to obtain a score.
- Freeze and validate before model execution. Keep the candidate and evaluator
  unchanged during the batch. Retain all attempts, errors and recovery costs.

The original Markdown-import task now selects the original `lark-drive` entry
skill. The previously selected `lark-markdown` skill explicitly excludes this
operation and redirects it to Drive import. The original skill contents and
references remain unchanged; this corrects selection, not the baseline CLI.
Other original domain selections are unchanged. Record this correction in the
environment and report instead of carrying forward the earlier routing error.

Record the final candidate's shipped guidance after product work is complete.
It should distinguish factual reads from editing metadata and make additional
reference reads conditional on missing information. All original commands,
formats, skill paths and references must remain available. The original
baseline's skill tree remains byte-identical to its frozen source.

## Planned local Markdown workflow acceptance

- A local Markdown working copy must preserve access to the native information
  needed for a correct update, including styles, link destinations, reference
  maps, resource identities and the observed revision. Markdown alone does not
  establish rich-document equivalence.
- Apply supported local changes as targeted native edits. Preserve unchanged
  content, formatting, Unicode, unrelated block IDs and resources. An unchanged
  working copy must not cause a remote content write. No lossy whole-document
  overwrite may be introduced as a shortcut to lower token usage.
- Unsupported or ambiguous edits must stop before writing and leave the local
  work available for correction. Reconcile partial or unknown remote outcomes
  before another write; do not interpret a failed command as permission to replay.
  Detect a stale baseline where the native API provides the needed evidence;
  do not claim concurrency protection from local state or readback alone.
- Validate local edit handling and native effects before evaluation, including
  preserved bold/link combinations, Unicode, untouched resources, repeated
  text, no-op edits and unsupported changes. Verify resulting native text,
  styles and link destinations when a write is required.
- Existing direct commands and advanced workflows remain supported. This
  local workflow must improve efficiency without removing original functions,
  weakening verification or requiring agents to use one fixed command sequence.

Both arms must solve the same user task with the same required fidelity. The
baseline may use any original supported workflow; it must not be required to
invoke candidate-only commands or produce a candidate-specific intermediate
format. If the local workflow needs fixture coverage, validate native semantics
and equivalent original routes before freezing that coverage. Record any task
or fixture changes explicitly; do not change them during model trials.

## Primary acceptance

For **each function**, compute total tokens across all attempts divided by
verified successes. With `C` and `O` denoting candidate and original/full tokens
per verified success, use reduction `1 - C/O` and the following gates:

| Function | Required reduction | Equivalent cost gate |
| --- | --- | --- |
| Docs read | Strictly greater than 50% | `C < 0.50 × O` |
| Docs edit | Strictly greater than 50% | `C < 0.50 × O` |
| Discovery, Contact, Base, Drive, Sheets, Markdown create | At least 30% each | `C <= 0.70 × O` for each function |

Compare unrounded values; exactly 50% does not pass either Docs gate. Each gate
also requires no observed success regression or loss of required data fidelity.
Cached input is included once. A pooled percentage cannot satisfy a per-function
gate, and savings on one task cannot compensate for another task's failure.

Incomplete usage, inconclusive coverage or zero verified successes in either
arm makes the affected successful-task comparison unavailable. Do not replace
missing values with zero or treat failed baseline workflows as completed tasks. Report mean
cost per attempt separately when accounting is complete, together with success
counts; that secondary measure cannot satisfy the primary acceptance gate.

The previous [condition-04 result](evidence/token-efficiency-04/REPORT.md) gives
exploratory original/full savings of 38.4% for discovery, 33.8% for Contact,
47.6% for Base and 55.9% for Drive. Docs read used 5.8% more. Original Docs-edit
usage was incomplete, and original Markdown/Sheets had no verified successes.
Those historical limitations remain; this fresh condition is not a rescore.

Three repetitions on development tasks are point estimates, not statistical
proof of stable savings, success-rate parity or full live functionality. Report
missed and unavailable targets explicitly, including remaining fixture limits.
