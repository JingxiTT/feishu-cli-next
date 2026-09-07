# Agent comparison evaluation

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
Other sections still need controlled Feishu backend adapters, and real comparative
results remain pending. Use the runner and independently implemented verifier
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
environment labels, and file hashes are persisted. A release reviewer must
separately inspect deliberately sanitized traces for repeated discovery, missing
detail, and autonomous choices; this minimal harness does not retain raw traces.

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

The installed native `codex-cli 0.146.0` provides `exec --json`, `--ephemeral`,
`--ignore-user-config`, and `--ignore-rules`. The adapter checks its version before
execution. Supply the native executable, not the npm Node shim, because child
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
  `codex-cli 0.146.0 turn.completed provider usage, aggregate per user turn`.
- Set `runner` to `["/absolute/python3", "/absolute/evaluation/codex_runner.py",
  "--codex", "/absolute/native/codex", "--auth-home", "/absolute/codex/auth-home"]`.
  Codex consumes its existing login internally. The adapter does not read,
  copy, or publish credentials. Do not put secret values in any argument.
- Set `verifier` to `["/absolute/python3", "/absolute/evaluation/foundation_verifier.py"]`.
- Use the captured reference binary and `skills/lark-shared/SKILL.md` for
  `official`; the same binary and `evaluation/official-concise/SKILL.md` for
  `official_concise`; and the next binary and
  `agent-skills/feishu-cli-next/SKILL.md` for `next`.

The current local model/effort selected by the coordinator is `gpt-6-astra` /
`high`; neither value is an adapter default. Pin both explicitly for all arms.
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
