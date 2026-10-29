# Real Feishu business evaluation with GPT-6 Luna — 2026-10-29

Across four fully verified business workflows, official and Next both succeeded **8/8 (100%)**. Next used **38.6% fewer provider tokens per successful task**: 189,483 versus 308,628. The balanced sample contains two fresh-context trials per CLI per workflow. This is descriptive evidence for these workflows, not whole-CLI parity or population success rates.

| Fully verified workflow | Official success | Next success | Official tokens/success | Next tokens/success | Next token difference |
|---|---:|---:|---:|---:|---:|
| Cross-source financial report | 2/2 | 2/2 | 507,784 | 374,524 | 26.2% fewer |
| Native rich-text edit | 2/2 | 2/2 | 249,616 | 80,768 | 67.6% fewer |
| Typed Sheet reconciliation | 2/2 | 2/2 | 309,637 | 125,378 | 59.5% fewer |
| Base intake (205 existing records) | 2/2 | 2/2 | 167,473 | 177,264 | 5.8% more |

The cross-source workflow reads a live Sheet and all 205 Base rows, computes filtered totals, writes a local JSON report and appends one native document paragraph. Independent API readback checks every original block/style/link and every source cell/record. The standalone cases test rich edits with native IDs, typed zero/empty-string updates with formula recalculation, and a null/false/zero/Unicode intake while preserving the existing dataset. Agent execution coverage is task-scoped; fixture creation and cleanup are controller work, not comparisons of either agent's creation commands.

The primary comparison retains every attempt on those four workflows. Official used 2,469,020 tokens; Next used 1,515,866. Cached input is a subset of input and is counted once. Uncached input was 355,175 for official and 166,343 for Next. No provider price or monetary saving is inferred. Base intake used **5.8% more** tokens on Next in this two-trial sample; Next did not lead on every workflow.

## File diagnostic and original verifier limitation

All **20/20** planned model attempts completed with full provider accounting. Four extra binary handoff attempts are retained separately: Next passed the byte/name/owner/ACL/guard checks **2/2**, official **1/2**. The failed official trial attempted an unnecessary folder-list call, received missing_scope and returned empty file/path fields. Its 215,075 tokens remain in the evidence.

Those checks could not independently prove destination-folder placement: the available individual-file metadata contains no parent field, while broad Drive listing needs a permission the user declined. Therefore the three positive file results remain **inconclusive for full-task success**. The known failed check remains a failure. The original raw flags are preserved, and [assessment.json](../../evidence/evaluation/live-business-luna-20261029/assessment.json) supplies the qualified interpretation. No finite full-suite success/token comparison is claimed. The reusable manifest now rejects that case before model billing. This exclusion follows a verifier audit, not its model result or cost.

All-attempt costs, including these diagnostics and the failure, were **2,901,631** official tokens and **1,762,740** Next tokens. They must not be substituted for the four-workflow comparison or pooled with the earlier Contact diagnostic.

## Conditions, authorization and cleanup

Model: `codex/gpt-6-luna` through Magpie, using Codex CLI **0.165.0**, medium reasoning, file-delivered product guidance, shell/files tools, a balanced randomized schedule (seed **2026102902**) and equal **360-second** budgets. Official is **v1.0.98** (`ac81f5056ab88f37d581d25916f3dab85abfca76`); Next is **47882761**. Both binaries and their guidance were frozen from the previous comparison. Official uses Go 1.25.5 and Next Go 1.26.10, a limitation on timing comparisons. Binary and evaluator hashes are retained in provenance.json.

Every scenario used production CLI calls against **real Feishu**, with fictitious data in fresh `[TEST]` resources. The account/tenant were bound to the independent proof; parent and child ownership and private ACLs were verified. The Base's nested table uses `TEST` because Feishu rejects square brackets in table names. The user restricted the run to existing scopes. No additional authorization flow was finalized, no Task/calendar/mail case was scheduled, no unrelated production resources were edited and no messages were delivered.

Server-confirmed deletion was established for **76 registered child resources**, including calibration fixtures, and the dedicated test parent. There are no unresolved run markers. Cleanup used exact journaled IDs and existing permissions. Temporary model-provider credential copies were removed. Private IDs, configuration, model workspaces and API diagnostics remain outside version control.

`make evaluation-test` completed **116** offline tests: **115 passed, 1 optional macOS native-skill test skipped**. These are tooling checks, not live success or usage evidence. A synthetic simulator and held-out tasks were not used. Two repeated samples per case do not establish statistical parity, broad business coverage or release readiness.

[Sealed measurements](../../evidence/evaluation/live-business-luna-20261029/assessment.json) · [All original attempts](../../evidence/evaluation/live-business-luna-20261029/attempts.raw.json) · [Measured evaluator snapshot](../../evidence/evaluation/live-business-luna-20261029/controller.tar.gz) · [Reusable live workflows](../LIVE_BUSINESS.md)

Validation limits: `next_layout.py check` and the staged diff check passed. `make script-test` passed its Python script harness and placeholder checks, then stopped in the inherited CI workflow test because Ruby is unavailable. The remaining script-test targets were not run; this is not a full script-test pass. No Go, CLI command/flag/schema, skill or capability-mapping changes were made.
