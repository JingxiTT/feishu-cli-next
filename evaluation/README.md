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

No real model runner, controlled Feishu backend adapter, or real comparative
results are bundled. This remains a release blocker. Use an external agent
runner and an independently implemented verifier with the protocol below; model
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
