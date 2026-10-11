# Agent comparison evaluation

## Offline CLI probes

The [complete October 7 surface comparison](../evidence/evaluation/cli-surface-20261007/REPORT.md)
records 1,738 local surfaces: all original commands, methods, event keys and
embedded guidance files. Every request schema is smaller (61.9% in the sum), but
936/938 help outputs are larger and current upstream is faster. The function
ledger exposes 866 original nodes without observed non-help calls in retained
workflow traces. A lead across every area/function remains unestablished.

[`measure_surface.py`](measure_surface.py) derives its complete schedule from
the frozen catalog and configured-brand inventory: every command node (including
hidden commands and inherited command aliases), all 251 methods, every event
key, and every file discoverable in the original binary's embedded skills.
It compares each surface separately rather than using a domain average. Failed,
unstable or unavailable probes have no efficiency score; a brand guard that
bypasses help is recorded without dispatching the command. Extensions and
packaging are explicitly unmeasured by local introspection.

```sh
python3 evaluation/measure_surface.py \
  --reference-cli /absolute/frozen-official-cli \
  --before-cli /absolute/next-before \
  --next-cli /absolute/next-after \
  --current-cli /absolute/current-official-cli \
  --brand feishu --repetitions 3 --encoding o200k_base \
  --output /absolute/new-surface-probes.json.gz
```

The optional tokenizer and isolated process protocol match `measure_cli.py`
below. The runner refuses to overwrite retained evidence. It checks unchanged
default schema bytes, request/permission/guidance parity for every method, and
semantic parity of event definitions. Current upstream differences are retained
separately and never treated as equivalent work. Three timings describe this
machine's observations; they cannot establish a universal speed advantage.
Help/schema/file coverage is **not** execution coverage: mutations, pagination,
recovery, streams, binary transfers, permission conditions, host SDKs and platform
builds still require their own correctness and task-efficiency evidence. No
whole-CLI savings claim follows from a pooled local-output reduction.

[`function_coverage.py`](function_coverage.py) joins this exhaustive surface
report to the original command inventory and the retained workflow traces. It
verifies checkpoint hashes, reconstructs task gates using all retained attempts,
excludes help calls from execution observations, and keeps historical binary
identities. Each original node is present even when no task called it. The SDK
symbol index and platform build conditions remain explicit unverified surfaces.
A verified workflow that calls a command does not establish every mode of that
function or attribute the entire workflow saving to it.

```sh
python3 evaluation/function_coverage.py \
  --probes /absolute/new-surface-probes.json.gz \
  --matrix evidence/evaluation/full-matrix-20260911 \
  --output /absolute/function-coverage.json.gz
```

The [October 7 comparison](../evidence/evaluation/cli-efficiency-20261007/REPORT.md) records
the request-only schema and discovery changes, both local measurement conditions,
and the remaining startup gap against current upstream.

[`measure_cli.py`](measure_cli.py) compares process latency and output context
size for six generated methods, shared guidance and bounded discovery. Supply
canonical builds of the [frozen official reference](../baseline/README.md),
Next before/after the change, and optionally a pinned current upstream build:

```sh
python3 evaluation/measure_cli.py \
  --reference-cli /absolute/frozen-official-cli \
  --before-cli /absolute/next-before \
  --next-cli /absolute/next-after \
  --current-cli /absolute/current-official-cli \
  --repetitions 31 --output /absolute/local-probes.json
```

Optional `--encoding o200k_base` requires `tiktoken` in the selected Python
environment; without it token counts remain unavailable. Counts cover stdout
and stderr once, not provider usage or tokens per completed agent task.
Binary/filesystem caches are warmed; each timed invocation is a fresh process
in rotating order with isolated configuration, remote metadata and notices
disabled. Any failed or unstable probe stops reporting.
The script requires exact request/permission/guidance parity against the frozen
reference, unchanged default schema bytes and unchanged discovery bytes. The
current upstream arm records parity separately because its catalog may differ.
Binary hashes, per-process durations and output hashes are retained.

These probes complement the full agent matrix below; they cannot establish
live API speed, successful task parity or the full-task token-efficiency gates.

## Automatic full matrix

[`run_matrix.py`](run_matrix.py) completed all **43 implemented development tasks / 258 attempts** across eight suites: paired functions, harder Docs, domain reads, workflow scenarios, scaling, business, foundation and Contact. Each task compares original/full with Next for three repetitions per arm. The [completed report](../evidence/evaluation/full-matrix-20260911/REPORT.md) contains **240 new attempts and 18 reviewed condition 09 attempts reused unchanged**; it is not a fresh simultaneous 258-attempt comparison. See the [execution and measurement protocol](reports/full-matrix.md).

The per-task target remains **unmet: 13 gates met, 22 missed and 8 unavailable**. Next verified **107/129** attempts; original verified **102/129**, with **two inconclusive**. Known usage is a **32,552,258-token lower bound**, comprising **31,650,572 new** and **901,686 historical tokens**. Six attempts lack complete usage, so both whole-matrix tokens-per-verified-success savings and raw token-spend reduction are **unavailable**. An [independent final audit](../evidence/evaluation/full-matrix-20260911/result-review.json) accepted the retained accounting and report claims.

Read the [fixture qualifications](../evidence/evaluation/full-matrix-20260911/QUALIFICATIONS.md) before interpreting the counts. The completed domain-read subset has 26/36 Next and 30/36 original verifications; nine Next and one original failure had correct typed answers and required successful reads but extra requests outside the bounded fixture. Recorded outcomes and all recovery costs remain unchanged.

Create an ignored local JSON configuration containing only absolute paths:

```json
{
  "official_source": "/absolute/frozen-original-source",
  "official_cli": "/absolute/frozen-original-cli",
  "next_source": "/absolute/frozen-next-source",
  "next_cli": "/absolute/frozen-next-cli",
  "codex": "/absolute/native-codex-0.153.4",
  "auth_home": "/absolute/existing-codex-home"
}
```

Use matching frozen source archives and fixture-enabled binaries; keep credentials in the existing protected Codex home, never in this JSON. The matrix pins Luna/medium, native Codex CLI 0.153.4, symmetric file guidance and 180/210-second budgets.

```sh
# Validate inputs, prepare extension hosts and render retained results; no model calls.
python3 evaluation/run_matrix.py --config /absolute/local-config.json \
  --output evidence/evaluation/full-matrix-20260911

# Run missing attempts or safely resume the same fixed matrix.
python3 evaluation/run_matrix.py --config /absolute/local-config.json \
  --output evidence/evaluation/full-matrix-20260911 --execute
```

Unchanged completed jobs are skipped even when their results failed, were inconclusive or lacked complete usage. Matching sealed, complete jobs can also be reused from sibling output directories under the same parent, including their original failure and accounting records. This is not a global cache; imported usage is counted as historical reuse, never new spend. Partial/in-flight sibling matches, busy locks, changed seals or conflicting evidence stop execution for reconciliation. Checkpoint validation preserves those attempts and costs; an unresolved in-flight marker requires reconciliation and is never automatically replayed. Changed measured inputs require a new output directory. Report-only changes do not trigger model reruns. Three section workers run isolated fixtures, retaining each section's serial balanced schedule; task failures and missed savings targets do not stop full coverage. Create `STOP` in the output directory to stop between completed attempts.

Python is part of the measured input identity. Use the same executable and version for reuse; the retained September 11 output pins `/opt/homebrew/opt/python@3.14/bin/python3.14`, with its exact version and hash in `environment.json`. The generic `python3` examples are suitable for a new setup, but a different interpreter cannot reuse this output.

Historical import requires the explicit condition, frozen evaluator and migration-review arguments described in [the protocol](reports/full-matrix.md#run-and-reuse). It verifies the complete prior schedule and measured inputs rather than selecting favorable rows. Reports separate historical usage from new spend and retain unavailable comparisons. Held-out release acceptance and exhaustive live parity remain separate.

### Reading the matrix results

The generated report contains each suite's task rows and evidence links. Keep these measures separate:

- **Coverage:** completed scheduled attempts, with fresh and historically reused work identified separately. A completed attempt is not necessarily verified.
- **Outcomes:** original and Next verified counts, failures and inconclusive results under each bounded fixture contract. Read the linked qualifications before attributing differences to live product behavior.
- **Gates and accounting:** each task's all-attempt tokens per verified success, with failed/recovery costs included. Docs require >50% savings and other tasks ≥30%, with observed success nonregression. Missing complete usage or an unavailable outcome comparison prevents a finite primary score; known spend remains a lower bound.

The report retains all six incomplete-usage records and their available costs; it does not estimate missing usage or treat it as zero. No failed or unknown-cost trial is automatically retried to improve the result. The retained 18 condition 09 attempts and their costs remain identified separately from new work.

## Completed condition 09

The completed [condition-09 report](../evidence/evaluation/token-efficiency-09/REPORT.md) completed **18 fresh attempts: Next 9/9 verified, original/full 8/9 verified and one inconclusive**. The **strict >50% savings target per task remains unmet: zero gates met, two missed and one unavailable**. Table/list reading saved **38.2%** and focused retrieval from a larger document saved **31.1%**, with both arms **3/3 verified for each task**. Multiple styled-edit comparison is **unavailable** because the original arm had one strategy outside fixture coverage; its other two attempts and all three Next attempts verified.

All usage was complete: **901,686 tokens**, including **185,209** for the inconclusive attempt. Inconclusive coverage is not scored as product failure, unknown usage is never zero, and every attempt and recovery cost remains. Three harder Docs tasks ran three repetitions per arm using candidate `cad221e`, evaluator `b28354f`, `gpt-5.6-luna` / `medium`, native Codex CLI `0.153.4` and symmetric filesystem-only full guidance. The [final private native lifecycle](../evidence/evaluation/docs-improvements-native-live-final-20260909.json) separately passed **9/9 correctness checks** on the same candidate; it does not measure model token savings. [Protocol](reports/token-efficiency-09.md) · [Frozen environment](../evidence/evaluation/token-efficiency-09/environment.json) · [Exact summary](../evidence/evaluation/token-efficiency-09/summary.json).

This new development cohort uses different documents and goals from condition 08. Condition 08's historical **56.8% combined Docs saving** is not a direct before/after improvement for condition 09, and no historical attempts are reused or pooled. No other functions were measured in condition 09. Small samples, bounded fixtures and one larger document size do not establish statistical success parity, stable savings, scaling behavior or complete live functionality.

The historical [condition-08 report](../evidence/evaluation/token-efficiency-08/REPORT.md) records **12/12 verified attempts**, original/full **6/6** and Next **6/6**, on the declared Docs-first stage. **The per-function target remains unmet:** Docs read saved **44.3%** and missed >50%; Docs edit saved **61.1%** and passed. Both arms verified **3/3 for each function**. The stop rule left **36 attempts across six other functions not run**, not failed or measured. All usage was complete: **677,392 tokens** across both arms. [Protocol](reports/token-efficiency-08.md) · [Stage decision](../evidence/evaluation/token-efficiency-08/stage-decision.json) · [Frozen environment](../evidence/evaluation/token-efficiency-08/environment.json).

The **secondary combined Docs workload saved 56.8%**: original **472,979 tokens, 6/6 verified**, versus Next **204,413 tokens, 6/6 verified**. This sums three read and three edit attempts per arm; expensive edits have greater token weight. The combined percentage cannot satisfy the separate read target or predict other workload mixes. All attempt and recovery costs remain included.

In condition 08, both arms received only the selected skill's filesystem path and had to read its contents within the measured tool loop. Next also used the shorter entry guide; its Go runtime, native fidelity checks and full original command surface stayed unchanged for that comparison. File delivery and guide length changed together, so that comparison does not isolate either change. Corrected ambient-skill suppression, `gpt-5.6-luna` / `medium`, native Codex CLI `0.153.4`, signed fixtures, independent verification and fresh contexts remain. Three repetitions and conditional continuation do not establish population success parity, stable savings or complete functionality coverage. No earlier baselines enter the scores.

Historical [condition 07](../evidence/evaluation/token-efficiency-07/REPORT.md) verified **11/12** (Next 6/6, original 5/6), with **36.1% read savings** (both 3/3) and **67.0% edit savings** (Next 3/3, original 2/3), using **720,698 tokens**. [Condition 06](../evidence/evaluation/token-efficiency-06/REPORT.md) verified **12/12**, with **1.0% more read tokens**, **42.1% fewer edit tokens**, and **761,257 total tokens**. Both stopped after Docs; all other functions were unmeasured. Their results remain separate and unchanged.

**Conditions 05/06 have an additional isolation qualification:** a later no-model native control confirmed that disabled-skill directory paths left eight public `CODEX_HOME` catalog entries advertised, while exact `SKILL.md` entry paths suppressed them. Both arms used the affected configuration. Selected guidance was explicitly supplied once; the probe found no real-home `.agents` paths. No billed-token impact or cause of file-read loops is established. Read the [qualification addendum](../evidence/evaluation/token-efficiency-05-06-skill-isolation-qualification.md) and its native evidence before attributing these results solely to the selected CLI guidance. Historical outcomes and costs remain unchanged.

The previous full [condition-05 report](../evidence/evaluation/token-efficiency-05/REPORT.md) records 48 fresh attempts: eight tasks, three repetitions, original/full versus Next. The target is **strictly more than 50% lower all-attempt tokens per verified success for Docs read/edit and at least 30% lower for every other evaluated function**, with no observed success regression. It replaces the unchanged-Next comparator for this new decision. **The target was not met: 3/8 gates met, 3 missed and 2 were unavailable.** [Protocol](reports/token-efficiency-05.md) · [Frozen environment](../evidence/evaluation/token-efficiency-05/environment.json) · [Readiness review](../evidence/evaluation/token-efficiency-05/readiness-review.json).

The batch verified **40/48 attempts**: Next 22/24 and original 18/24. Contact, Markdown creation and Base met their gates. Docs read saved only **6.4%**, with both arms 3/3 verified; Docs edit used **172.0% more tokens per verified success**, with Next **2/3 versus original 3/3**. Discovery used 14.8% more. Sheets' original arm had no verified successes; Drive had four incomplete usage records and no original successes, so both primary comparisons are unavailable. Observed provider spend is a **3,602,432-token lower bound**. The report retains all failed-attempt costs and missing accounting.

Docs-edit traces show `+push` exit 2 and `+update` fallback, but omit edited local files and validation-error text. Their exact rejection cause is unknown. Neither fallback costs nor the failed verification are removed from the results.

The candidate adds a local Docs `+pull` / `+push` text workflow while retaining all original commands and rich-content routes. A [private native lifecycle](../evidence/evaluation/docs-workspace-native-live-20260909.json) passed independent original-CLI checks of text, styles, links, IDs and an untouched table. That product check is separate from the synthetic model benchmark and does not establish universal Markdown conversion or atomic concurrency protection. The [revision probe](../evidence/evaluation/docs-workspace-native-cas-20260909.json) showed stale native PATCH revisions are accepted; current-state fixtures must not fabricate a compare-and-swap guarantee.

Condition 05 retains `gpt-5.6-luna` / `medium`, native Codex CLI `0.153.4`, signed fixtures and independent verification. Original Markdown import now starts from its unchanged `lark-drive` skill, correcting the prior entry selection. Both original guidance bytes and historical results remain intact. Incomplete accounting, inconclusive fixture coverage or zero verified successes make the affected primary comparison unavailable; failed and inconclusive attempts retain their costs. Three repetitions provide point estimates, not population success-rate parity.

The historical [candidate-only guidance follow-up, condition 04](../evidence/evaluation/token-efficiency-04/REPORT.md), contains **24 new candidate trials compared with 48 reused condition-03 baseline trials**. This exploratory comparison is not a new balanced 72-trial run. It keeps the eight tasks, three candidate repetitions, `gpt-5.6-luna` / `medium`, native Codex CLI `0.153.4`, and native-informed synthetic fixtures. [Protocol and reuse limitations](reports/token-efficiency-04.md).

The new candidate verified **24/24 attempts**, with complete accounting for **1,154,003 new provider tokens**. All eight functions improved tokens per verified success against reused unchanged Next, and all observed success counts matched or exceeded both reused baselines. The combined 50% gate was **not fully met: 5/8 functions met it**. Docs read (39.5%), Markdown (43.1%) and Drive (40.0%) remained below the target. Reused baseline usage is not new spend; the report preserves original Docs-edit incomplete accounting and original Markdown/Sheets zero-success comparisons.

Condition 04's 50% point-estimate target applies to each function's all-attempt tokens per verified success against unchanged Next, with observed success nonregression against both unchanged Next and original/full. Reused baselines, adaptive selection and three repetitions do not establish statistical parity, stable savings or live-product equivalence. Failed-attempt costs remain included; incomplete usage, inconclusive outcomes and zero successes make affected scores unavailable.

Historical conditions remain immutable:

- [Optimization 03](../evidence/evaluation/token-efficiency-03/REPORT.md): 72 fresh trials, 62 verified, 10 failed, one incomplete usage record; 2/8 combined function gates met. [Protocol](reports/token-efficiency-03.md).
- [Optimization 02](../evidence/evaluation/token-efficiency-02/REPORT.md): 72 fresh trials, 64 verified, seven failed, one inconclusive; 4/8 gates met, with a Docs-edit success regression. [Protocol](reports/token-efficiency-02.md).
- [Original paired pilot 01](../evidence/evaluation/paired-luna-medium-01/REPORT.md): 72 fresh trials, 54 verified, 11 failed, seven inconclusive and complete accounting for 8,442,585 tokens. It compared original/full, original/concise and Next; later optimization conditions use unchanged Next instead of original/concise. [Protocol](reports/paired-functions-luna-medium.md).

These are per-task comparisons, not pooled efficiency rankings. Broader fixture coverage does not retrospectively rescore earlier outcomes. Other historical scenario batches below remain separate conditions.

The [provisional assessment](reports/ASSESSMENT.md) compares completed conditions, retains
regressions and links the remaining release evidence.

Run the isolated harness acceptance suite:

```sh
python3 -m unittest discover -s tests/next_evaluation -v
```

These executable tests launch the actual harness and separate runner/verifier
processes. They check fixture isolation, task loading, balanced randomized trials,
independent scoring, output artifacts, failed attempts, cache accounting, and
held-out protection. Counts in these tests are **synthetic**, not model usage or
CLI efficiency measurements. They make no Feishu calls. No credentials are needed.

`tasks.development.json` and `tasks.held-out.json` define initial scenarios and
success criteria before tuning. Section owners must implement fixture adapters
and validate criteria against their capability inventory. They are not passing
capability evidence. Held-out scenarios have not been executed; do not use them
for interface tuning. The release flag is intentionally required to execute them.

## Real comparison prerequisite

`codex_runner.py` and `foundation_verifier.py` implement real Codex execution
and independent checks for two offline foundation tasks described below.
The [foundation report](../evidence/evaluation/foundation/REPORT.md) records the completed
18-trial compatible comparison and all earlier failed attempts. All predeclared sections now have controlled adapters; broader comparative results
remain incomplete. The corrected [Wiki comparison](../evidence/evaluation/wiki-native-74/REPORT.md)
and [nine-section batch](../evidence/evaluation/read-312af93/REPORT.md) retain mixed results and failures.
Use the runner and independently implemented verifier
with the protocol below; model
usage must come from the provider across every turn. A CLI output byte count,
locally estimated tokenizer count, or synthetic runner is not a substitute.

The runner must provision a fresh agent context and equivalent fresh backend
fixture for each request, collect provider usage even on errors, and clean up its
own backend fixtures on failure. It must use the pinned settings and task budget.
The verifier must inspect actual backend state/artifacts, verify user intent and
absence of unintended effects, and never trust the final agent claim. Run only
offline controlled fixtures until live account/tenant/visibility restrictions in
`docs/PLAN.md` have been verified. No test delivery beyond the authorized recipient
is permitted. The harness does not authorize live calls or enforce backend ACLs.

Create a local JSON config with these fields (all paths must be absolute):

```json
{
  "mode": "real",
  "backend": "controlled-backend@fixture-revision",
  "model": "exact-provider-model-version",
  "tokenizer": "exact-tokenizer-version",
  "reasoning": "pinned-reasoning-setting",
  "tools": ["shell", "files"],
  "permissions": "offline-fixture-only",
  "budget": {"max_turns": 30},
  "accounting_method": "provider per-turn usage; input includes cached input",
  "timeout_seconds": 300,
  "runner": ["/absolute/agent-runner"],
  "verifier": ["/absolute/independent-outcome-verifier"],
  "arms": {
    "official": {"cli": "/absolute/reference-cli", "skill": "/absolute/official-skill.md"},
    "official_concise": {"cli": "/absolute/reference-cli", "skill": "/absolute/concise-equivalent-official-skill.md"},
    "next": {"cli": "/absolute/feishu-cli-next", "skill": "/absolute/next-skill.md"}
  }
}
```

All environment labels must be public, sanitized identifiers. Do not place
credentials, private endpoints, or user content in config or task manifests.
Official and official-concise arms must use the same binary. Match guidance
coverage before comparison and document differences in the release report. Skills
with references must be pinned with their complete distribution outside this
harness; the entry file hash alone does not capture linked content. Pin runner,
verifier, tools, backend fixture, and model versions in the release manifest.

```sh
python3 evaluation/harness.py --tasks evaluation/tasks.development.json \
  --config /absolute/local-config.json --output /absolute/results.json \
  --repeats 3 --seed 20260907
```

Each task/arm receives the same number of independent trials (minimum two).
The global order is deterministically shuffled. Three repeats are an initial
sample, not a statistical sufficiency claim; increase balanced repeats when
variability prevents a conclusion. Warm-session experiments must be separately
identified and are not implemented by this fresh-context harness.

## Process protocol, version 1

Each process receives one JSON request on stdin and returns exactly one JSON
object on stdout. Diagnostic stderr is discarded. The runner receives `task_id`,
`trial`, `arm`, `prompt`, `fixture`, `workspace`, `fresh_context: true`, `cli`,
`skill`, and common `settings`. It does **not** receive `expected`. The fixture
also exists as `workspace/fixture.json`. The runner may create local artifacts
within this unique temporary workspace. The harness deletes it after verification.
Backend adapters should keep a run resource ledger in it for independent checking.

Runner response:

```json
{
  "claimed_success": true,
  "usage_source": "provider",
  "model": "exact-provider-model-version",
  "tokenizer": "exact-tokenizer-version",
  "accounting_complete": true,
  "turns": [
    {"input_tokens": 120, "cached_input_tokens": 40, "output_tokens": 25}
  ],
  "api_calls": 2,
  "retries": 0
}
```

`input_tokens` includes cached input. Count every model turn, including skill
injection, help, tool responses, failures, and recovery; `accounting_complete`
attests that no turn is omitted. Normalize provider APIs that report uncached
input separately. Never subtract cached tokens. A failed process can still emit
complete usage; its cost is retained. Missing/malformed usage or timeout is
unknown, never zero, and prevents a finite tokens-per-success score for that
group. A provider-backed run with no usage must remain inconclusive.

The verifier gets `task_id`, `workspace`, `fixture`, `expected`, `backend`, and
`claimed_success`. It returns `{"success": true}` only after checking all expected
effects and preserved content. Quality, recovery, identity, type preservation,
pagination completeness, and unintended writes belong in the predefined expected
criteria. Alternate strategies are valid. Wrong recipients, hidden truncation,
corruption, or a misleading claim fail verification. Its executable must differ
from the runner. This process separation is not an adversarial sandbox: adapters
are trusted evaluation code and must be reviewed for independence.

Child processes receive only a standard PATH, isolated HOME/TMPDIR, and locale.
No inherited Feishu credentials, profiles, proxy configuration, or private content
are forwarded. Real adapters must manage their explicitly authorized access.
Keep credentials out of argv. Raw stdout, stderr, final claims, and traces are
not published. Only fixed categories, token counts, numeric telemetry, sanitized
environment labels, file hashes, and the restricted command-choice records below
are persisted. A release reviewer can inspect those records for repeated
discovery and autonomous choices; this minimal harness does not retain raw traces.

Results contain task/domain/arm success, all-attempt token distributions, cached
input, failed costs, latency, API calls, retries, and failure categories.
`tokens_per_success` is all attempt tokens divided by verified successes, or
`null` when there are no successes or any usage is unknown. Matched-success pairs
are reported separately for both official baselines against next. Real mode and
complete usage permit measurement reporting, but do not by themselves establish
a speedup, representative sampling, or release acceptance.

Exit status is 0 when all outcomes and usage checks pass, 1 for failed trials,
and 2 for invalid configuration/manifests. Stdout gives machine-readable status;
configuration failures go to stderr without echoing potentially private values.

## Codex adapter: offline foundation development

Native Codex provides `exec --json`, `--ephemeral`,
`--ignore-user-config`, and `--ignore-rules`. The adapter checks its version before
execution. `gpt-6-astra` rejected version `0.146.0`; the completed comparison uses
`0.153.4`, supplied explicitly through `--codex-version`. Supply the native
executable, not the npm Node shim, because child
PATH intentionally excludes user-installed executables. Keep its binary hash
with release evidence. The [official non-interactive documentation](https://learn.chatgpt.com/docs/non-interactive-mode)
documents the JSONL `turn.completed.usage` fields. Each such usage object covers
an entire Codex **user turn**, including its internal model/tool loop; the adapter
labels `usage_granularity: codex_user_turn`. It does not claim unavailable
individual model-turn counts. Incomplete runs retain known usage spans while
their complete total and efficiency score remain unknown.

Use `tasks.foundation.json` only at the completed foundation section boundary.
It asks the model to inspect unconfigured auth state and discover document-read
flags without any backend operation. `foundation_verifier.py` independently runs
the selected CLI, checks the factual JSON answer against predefined criteria,
and ensures the isolated Feishu configuration stayed empty. No live Feishu
credential is provided. This narrow suite is not evidence for authenticated
setup, login, permission recovery, or backend workflows.

Start from the config above with these changes:

- Set `backend` to `offline-foundation-v1`, `model` to the explicitly selected
  model, `reasoning` to its pinned effort, and `budget` to
  `{"timeout_seconds": 180}`. Set outer `timeout_seconds` above that budget, for
  example 210. The adapter enforces wall time; it does not claim to enforce a
  turn-count budget.
- Set `tokenizer` to `provider-undisclosed` if Codex does not disclose its version;
  never invent a tokenizer version. Set `accounting_method` to
  `codex-cli 0.153.4 turn.completed provider usage, aggregate per user turn`.
- Set `runner` to `["/absolute/python3", "/absolute/evaluation/codex_runner.py",
  "--codex", "/absolute/native/codex", "--auth-home", "/absolute/codex/auth-home",
  "--codex-version", "codex-cli 0.153.4"]`.
  Codex consumes its existing login internally. The adapter does not read,
  copy, or publish credentials. Do not put secret values in any argument.
- Set `verifier` to `["/absolute/python3", "/absolute/evaluation/foundation_verifier.py"]`.
- Use the captured reference binary and `skills/lark-shared/SKILL.md` for
  `official`; the same binary and `evaluation/official-concise/SKILL.md` for
  `official_concise`; and the next binary and
  `agent-skills/feishu-cli-next/SKILL.md` for `next`.

The user selected `gpt-5.6-luna` / `medium` for the new comparison. Pass
`--model gpt-5.6-luna --reasoning medium` to `run_read_batch.py`; both values are
shared by all arms and forwarded explicitly to Codex CLI. The launcher's old
`gpt-6-astra` / `high` defaults remain for compatibility with historical commands;
they are not the selected settings for the new comparison. Neither is an adapter
default. Pin both explicitly for all arms.
Provider alias changes cannot be excluded without an immutable provider model
identifier. Report that limit alongside measured results.

Each run uses fresh context and a fresh temporary workspace, HOME, TMPDIR and
Feishu configuration. It disables metadata fetching, update/skill notices,
agent shell network access, plugins, apps, browser/computer tools, memories,
subagents, and skill search. `--ignore-user-config` suppresses installed user
configuration; `project_doc_max_bytes=0` suppresses unrelated AGENTS guidance.
Skill path names under the existing auth home's skills directory are enumerated
and disabled without reading their contents. Only the selected skill directory
is copied, and its entry is injected equally for each arm. Relative references
remain available under `guidance/`. Selected binaries are exposed under both
conventional CLI names so reference examples remain runnable.

The isolated settings were accepted by a local `codex debug prompt-input` check
with an empty temporary Codex home; no model was called and the full prompt was
not published. Managed/system-level host policy may still affect tools. Preserve
the pinned host/CLI environment for all arms and audit unexpected tool events;
the adapter rejects observed MCP, web-search, or collaboration tool use. This is
controlled fresh-context evaluation, not an adversarial filesystem sandbox.

The concise official guidance covers the shared setup/identity/output/safety
semantics with shorter English prose. The official shared skill is Chinese and
mandates more reference reading. Arm 1 versus 3 is therefore a product comparison,
not a language-specific result. Arm 2 versus 3 narrows the guidance difference,
but differing domain detail still limits interface-only attribution. No forced
help lookup, planning sequence, or model-token ceiling is part of the task.

### Sanitized command-choice records

Future runs retain `command_executions` and `control_trace`. The count is completed
Codex command events, not API requests or a count of interrupted commands. Each
trace record contains the submitted shell command's overall exit code and a list
of conservative choices: a recognized CLI command path and public flag names,
or a fixed file-read, file-search, script, other-shell, or unparsed-shell category.
Command/flag names are restricted to the captured registration vocabulary plus
the new `discover` root. The harness revalidates the vocabulary before writing
results. Unknown flags, flag values, environment values, file paths, input/output
content, and arbitrary tool output are omitted. Nested common shell wrappers are
recognized; complex scripts and invocations may remain unparsed. Shell conditions
can prevent submitted commands from running, and a compound command has one
overall exit code, so these records show choices rather than independently
verified per-command effects. They can identify repeated help/schema/skill
discovery without retaining document text or user identifiers.

The foundation batches predate this instrumentation. Their raw output was
discarded, and no command choices can be recovered from their retained evidence.
They were not rerun merely to obtain traces.

### Synthetic Contact development task

`tasks.contact.json` instantiates the predeclared `contact-dev-01` task. Its
abstract person IDs map to API-valid `ou_fixture_1` and `ou_fixture_2`; both are
named Alex and have distinct synthetic `.invalid` email addresses. This avoids
penalizing exact-ID search through its required `ou_` client validation. The task
asks for the second person's ID/name/email without prescribing discovery or a
command sequence. The expected answer is withheld from the runner.

`section_fixture.py` is a loopback HTTP sidecar with no outbound forwarding code.
It validates the CLI's signed sidecar requests and implements user batch lookup,
bot lookup, user search, and caller identity reads against the same two records.
All other methods/routes and invalid signatures fail. Authsidecar-tagged binaries
are built from clean Git archives with the same pinned metadata, not working-tree
snapshots. They use sentinel credentials and isolated empty Feishu configuration.
The official Contact skill and its references are supplied intact; its shared
skill dependency is copied alongside. Arm 2 uses concise English Contact guidance;
Arm 3 uses the new product entry skill and discovers further guidance itself.

Codex0.153.4 uses a named workspace permission profile, managed network proxy,
allow rule for127.0.0.1, and `allow_local_binding=true`. The latter is necessary
because Go's HTTP transport connects directly to loopback without its HTTP proxy.
The controlled native-sandbox test checks actual signed CLI reads succeed and
an external direct socket is denied by the OS. This permits local communication,
not arbitrary external networking; do not enable it for live credentials.

The fixture keeps its request ledger outside the agent workspace during the run
and writes its receipt only after the model process ends. `contact_verifier.py`
requires the exact profile answer, a successful signed read exposing that ID,
no rejected requests, and no created Feishu configuration files. It does not
trust an answer or a success claim without backend observation. Published evidence
retains request counts and sanitized command choices, never raw model/tool text.
Each completed harness attempt is checkpointed before the next billed attempt.

Run all unit/process tests with `python3 -m unittest discover -s tests/next_evaluation -v`.
To include the optional real CLI and OS-sandbox controlled checks (no model calls),
set `EVAL_CONTACT_BINARIES` to a JSON array of the two archived binary paths and
`EVAL_NATIVE_CODEX` to the pinned native Codex executable. The Contact evidence
configuration template specifies a three-repeat balanced real batch; resolve
its local path placeholders in a temporary file, including the existing Codex
auth home without reading or copying its credentials. Held-out tasks stay closed.

### Other bounded read-only sections

`runtime/*.json` normalizes nine section-owner contracts into reviewed signed
read response tables; `tasks.read-sections.json` contains only public prompts/type
schemas plus hidden verifier expectations. The server data stays outside agent
workspaces. GET and explicitly reviewed POST reads are allowed; other routes are
rejected without forwarding. Selectors distinguish IDs, formats, revisions and
identities. This is a bounded fixture, not a general mock of the whole service.
Unknown valid strategies may expose a fixture limitation and require a separately
versioned balanced rerun; do not silently call that a product defect.

`read_verifier.py` compares typed values (false differs from zero), optional
unordered answer arrays, required independent signed-read groups and rejected
request absence. Static route state has no mutation implementation. It permits
local `next/receipts` journals created for conservative raw POST accounting while
rejecting auth/profile/config changes. The post-run server receipt remains outside
the model's control during execution. Expected output constants are never placed
in the model-facing answer schema.

`run_read_batch.py` accepts explicit archived source/binary paths and the pinned
native Codex/auth-home paths, stages original references, and runs each section's
three-arm/three-repeat batch serially. It refuses existing output to preserve
failed and partial attempts. `--prepare-only` validates/stages without model calls.
Use a frozen evaluator code snapshot as well as frozen product binaries when
preparing later adapters concurrently. The evidence must identify both revisions.
All official skill siblings remain available for relative dependencies, including
Note/Minutes/VC compatibility routing through `lark-meeting`; only the selected
entry is injected. Concise English entries replace that main entry while retaining
the same original reference files. Application has no official domain skill: its
unchanged upstream affordance is the documented Arm1 fallback. Differences in
language, reference-loading and domain coverage limit interface-only attribution.

Set `EVAL_READ_BINARIES` to the JSON array of archived authsidecar binaries to run
all reviewed CLI examples and independent proofs as controlled tests. No provider
calls occur during this test. Native loopback/external-denial checks use the
same sandbox profile as Contact. A completed small read task does not establish
section-wide safety, mutation fidelity, live privacy or held-out generalization.

## Executable development fixtures

`tasks.read-sections.json` now instantiates twelve read-only sections;
`tasks.scenarios.json` instantiates the nine state, artifact, event and installation
sections. Together with Foundation and Contact they retain all 23 predeclared
sections. They instantiate development tasks only; held-out tasks are unchanged.
The reviewed contracts and concise guidance remain section-specific.

`run_read_batch.py --task-kind read|scenario` stages the same frozen official
reference distribution for both official arms, substitutes only the concise
entry for Arm 2, and runs sections serially with three trials per arm. Use its
`--prepare-only` flag to validate all supplied clean sources and build the small
Extensions host before billing a model. Extensions has no official skill; its
Arm 1 source is the frozen extension and platform READMEs plus the public
chat-brief example. Application uses its frozen affordance fallback. Packaging
installs the unchanged selected distribution as data; its two official arms
receive identical original `lark-shared` installation inputs.

`scenario_protocol.py` is a small interface, not a general backend emulator.
Scenarios validate signed requests, record independent native state/effects and
check actual files after execution. Binary responses preserve bytes. Transport
shutdown drains bounded handlers before final evidence capture. Events bind
kernel peer observations to the selected executable and require defensible
provider-reported command exit evidence. Extensions likewise accepts only command
forms whose execution and artifact capture it can establish. The Drive output
audit recognizes common large binary dumps, not every possible encoding.
Packaging probes the actual native sandbox before model execution: project
writes succeed while protected home/config writes and external sockets fail.

Trusted observation hooks consume raw provider events transiently; raw commands,
payloads, private paths and content are not published. Hook or cleanup errors
retain already reported model usage. An explicitly recognized valid strategy
outside a fixture's implementation is inconclusive, retains all billed cost and
suppresses success-rate/efficiency claims for its affected group. Unsupported
strategies require adapter expansion before a product conclusion. These fixtures
do not establish live backend behavior, complete workflow fidelity or arbitrary
filesystem-read auditing.

The first controlled integration against candidate `312af93` found a Mail
`user_mailbox.messages batch_get` metadata read returning exit 22 as an unknown
mutation; the original reference returned 0. This occurred before any Mail model
trial. The corrected accepted candidate `8d92e5` passed controlled acceptance and
the subsequent [Calendar/Mail/OKR comparison](../evidence/evaluation/read-8d92e5/REPORT.md), with
all 27 independently verified trials. The earlier 81-trial nine-section read batch
keeps its original candidate and evaluator snapshots.

A coordinator search accidentally exposed limited held-out snippets during later
scaling-coverage review. Files remain unchanged and no held-out trial ran, but
complete blinding can no longer be claimed for the affected examples. See
[evidence/held-out-exposure.json](../evidence/evaluation/held-out-exposure.json). They require
independent blind replacement after product freeze before release evaluation.

Future read-only runs retain only allowlisted route IDs, HTTP methods/statuses and
fixed verifier failure codes. These distinguish answer mismatch, missing read
proof, a request outside the fixture, and forbidden configuration changes without
retaining paths, queries, payloads or content. The earlier frozen `cc745f5` runs
lack these diagnostics; failed raw-API attempts there cannot be attributed to a
specific route or product defect retrospectively.

Docs fixture v2 (`57a2a9b`) keeps the task prompt and answer unchanged while
checking native character-level bold, hyperlink URL and Unicode semantics instead
of invented hidden inline GUIDs or a fixed edited-paragraph ID. Untouched blocks,
resources and metadata remain exact. Bounded XML replacement and one-paragraph
replacement have controlled coverage; general reconstruction remains unsupported.
Historical v1 results are not rescored as successes.

The Extensions observer accepts the native `--json`/`--json=true` shorthand after
`121f405`; its earlier rejection was an evaluator defect. Composed-script execution
still requires independent proof rather than trusting a model-written artifact.
Events likewise retains its predeclared zero-exit requirement: kernel delivery and
process disappearance alone cannot establish a child's zero exit status. Missing
script-child exit evidence is a comparative limitation, not a demonstrated CLI bug.
No process-supervision infrastructure was added to disguise this limit.

Packaging fixture v2 (`4548e65`) accepts either the exact installed skill directory
or its `SKILL.md`, because the unchanged prompt permits both interpretations of
`installed_path`. Normalization permits only one leading `./` and one trailing
slash on the exact directory. Artifact/reference, unrelated-file and protected-tree
checks remain required. Earlier discarded answers cannot be recovered or declared
successful from this correction.

Future accounting diagnostics retain only four fixed reason codes for malformed
provider events, failed or mismatched turns, and unexpected tools. They do not
change accounting rules or recover reasons discarded by older evaluator versions;
any completed spans remain known partial usage, never an invented complete total.

For an operator stop request, create `STOP` under the batch output root. The
launcher/harness checks it between complete attempts, checkpoints all active
attempt usage first, and cancels unstarted slots without calling them product
failures. A partial series is not a balanced performance comparison. Direct
harness users may supply an explicit `--stop-file` path.

## Development transfer-size experiment

`tasks.scaling.json` declares separate Base import and Sheets export tasks at
100, 1,000 and 10,000 rows. `run_read_batch.py --task-kind scaling` uses the same
required source/binary/auth/Codex/output arguments as other batch kinds; add
`--prepare-only` to validate setup without starting any model. Each domain runs
27 balanced, seeded, serial fresh-context attempts (three sizes, three arms,
three repetitions). Only the evaluator starts these workloads after the active
batch finishes and the candidate and harness are frozen. The `STOP` file in the
output root stops before the next model, retaining any active attempt's cost.

The fixed synthetic source family is shared across arms and repetitions; smaller
sources are prefixes of the 10,000-row source. This limits generalization across
data distributions. The Base source is a typed exchange file with an explicit
mapping, identical for every arm: the reference can parse it and submit bounded
native batches; the candidate may use native import. That measures file
interoperability and guidance as well as interface defaults, not an isolated
formatting change. Sheets permits legacy file output and native bound export.
No command sequence is prescribed to the model.

The two adapters independently check actual complete artifacts/backend state,
all supported scalar types and omissions, signed read/write coverage, immutable
schema/sentinels and input bytes. Their controlled tests cover both clean
binaries at all sizes. Artifact sizes and request counts are diagnostics only;
provider token measurements are **not yet run** for this experiment. Future
reports must retain failures, cached input, input/output totals, paired outcomes
and size-dependent variation without a fixed improvement target. These fresh
fixtures derive from the public plan's scaling requirement; they do not reuse
quarantined held-out examples.

## Supplemental generated business sampling

`tasks.business.json` adds two representative development tasks: an approval
creation/readback workflow and a native mindnote edit/readback workflow. Twelve
controlled cases cover all 17 generated methods; the remaining methods have no
model comparison claim. `run_read_batch.py --task-kind business` runs only the
two declared tasks, for 18 balanced fresh trials after the existing queue.

These domains have no separate frozen domain skill. Both reference arms receive
the same complete `lark-shared` distribution; the concise arm replaces its entry
with the reviewed shared English guidance also used by the packaging comparison.
The candidate receives its pinned entry skill. This is source-matched general
CLI guidance, not an invented official business skill. The adapter uses the same
signed loopback transport, provider accounting and independent state verifier.
No supplemental model trials have run yet.

## Completed size-series plot

The [native-fidelity audit](../evidence/evaluation/native-fidelity-validity-audit-20260908.md) qualifies the Base and Sheets fixtures underlying the retained size plot. Its provider costs remain historical, and its outcome markers denote synthetic checks; it does not establish live same-fidelity efficiency. The original figure remains unchanged.

After both Base and Sheets batches contain all 27 planned attempts, export the
provider-token plot with Matplotlib (3.10.8 used for this work):

```sh
python3 evaluation/plot_scaling.py evidence/evaluation/scaling-8d-archive04
```

The script produces PNG/SVG files beside the batch reports and embeds their
source hashes. It rejects incomplete batches. All complete-usage attempts remain
visible, including nonverified outcomes; unknown totals are disclosed and are
never plotted at zero. Inspect the actual exported images before including them
in the final report. The script alone is not completed scaling evidence.
