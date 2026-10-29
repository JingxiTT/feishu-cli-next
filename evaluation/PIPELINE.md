# Evaluation pipeline

`pipeline.py` is the single entry point for new official-versus-Next agent
comparisons. **Every evaluated task uses real Feishu.** Both arms use production
binaries, the same model settings and budgets, fresh agent contexts, independent
live verification, and a balanced randomized schedule. Reports retain failed
attempts, recovery costs and incomplete accounting. They have no acceptance
threshold or savings gate.

Synthetic backends previously made repeatable comparisons possible without a
Feishu account or permission to create resources. They measured the simulator's
contracts, not production behavior. The fake endpoint, simulators, response tables,
synthetic task sets and their launchers have been removed. Ordinary offline tests use small process mocks to
check accounting and lifecycle behavior without implementing Feishu APIs.
Changing a synthetic fixture's backend label does not port its fake resource IDs,
responses, task goals or verification to Feishu.

## Evaluation and CI

CI runs software correctness tests, including mocked tests of this pipeline.
`make evaluation-tooling-test` and its `make evaluation-test` compatibility alias
never run agent evaluations, bill models or contact a tenant. Evaluation requires
an explicitly launched local `pipeline.py --execute` run; live execution refuses
CI. Test counters are not provider measurements. Local help/schema probes such
as `measure_surface.py` remain offline diagnostics and are not agent evaluations.

Historical evidence keeps its original labels. Old simulators and reproduction
launchers are available only at their recorded Git revisions. Historical pipeline
reports can still be rendered with `pipeline.py --output ... --report-only`.

## Configure and run

1. Obtain frozen **production** official and Next binaries and matching source
   distributions. Fixture-enabled builds are unnecessary. Keep source archives
   outside the output directory.
2. Authenticate locally to Feishu. Independently verify the effective account and
   tenant. Read [authorization boundaries](../docs/testing.md#authorization-boundaries-for-all-testing)
   before any tenant call, including fixture setup, verification and cleanup.
   Keep credentials in a protected CLI configuration directory, never in JSON
   configs, command arguments or version control.
3. Copy [pipeline.example.json](pipeline.example.json) to an ignored local path.
   Replace all absolute paths and the model ID. The bundled starter is one
   read-only current-account Contact task against real Feishu; it creates no
   remote objects and sends no content to other people. It requires an
   independently established owner-only binding file:

   ```json
   {"algorithm": "sha256", "digest": "<independently verified account binding digest>"}
   ```

   The digest is SHA-256 of UTF-8 `contact-current-account-v1`, a NUL byte, the
   verified account open_id, another NUL byte, and the verified tenant_key. This
   matches the existing private live-test binding format. Do not derive the
   expected identity from the same unverified run you are trying to check.
4. Pin one experiment per harness/model/reasoning combination. Native experiments
   use the Codex CLI 0.153.4 adapter and a protected `auth_home` for model-provider
   authentication. Feishu authentication separately uses `live.cli_config_dir` and
   `live.cli_data_dir` (the existing encrypted credential-store base on Linux).
   Both are explicit paths; the adapter never copies or reads their contents.

```sh
# Without config: the bundled live Contact task.
python3 evaluation/pipeline.py --list-sections

# With config: only the actual configured live task sets.
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json --list-sections

# Validate public inputs and print the selected schedule; no tenant/model calls.
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json --dry-run

# Prepare local guidance/adapters and reports; no tenant/model calls.
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json \
  --output /absolute/private-evaluation/contact-01

# Explicitly execute setup, agent attempts, live verification and cleanup.
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json \
  --output /absolute/private-evaluation/contact-01 --execute
```

Preparation validates paths, manifest/schema contracts and adapter settings. It
cannot establish live authentication, scopes, backend availability or resource
privacy; the fixture driver must establish these before the model starts.

`live_manifests` maps each selected suite to an absolute development manifest
with `backend: "live-feishu-v1"`. Use
[tasks.live-contact.json](tasks.live-contact.json) as the task-file example.
Every task has `id`, `section`, a user-goal `prompt`, `fixture.answer_schema`, and
`expected` verification guidance. Expected data stays out of the runner request.
Live resource IDs and source files belong in setup, rather than being replaced
with fake constants. A missing live manifest or unsupported task is **blocked**;
the pipeline never falls back to a simulator.

The nine suite names and workflow/diagnostic views remain available. Without a
`suites` selection, the pipeline schedules only configured live manifests in the
selected workloads. The current-account Contact task and the
[business workflows](LIVE_BUSINESS.md) have bundled live drivers;
other tasks require a real fixture driver and live manifests before execution.
There is no claim of live coverage for an unported task.

`--suite`, `--section`, `--workload` and `--experiment` replace the corresponding
config selections; each accepts multiple names or repeated flags. Selections
intersect. `--repeats` (at least two, default three), `--budget-seconds`, `seed`
and `guidance_delivery` (`file` or `inline`) apply symmetrically. Workflow and
diagnostic results are separate, never pooled. Experiments run serially.

## Private recipient configuration

Other-recipient delivery and access grants are disabled by default. No personal
recipient names or IDs belong in tracked code, docs, prompts or fixtures. When a
task needs an authorized other recipient, set `live.recipient_file` in the ignored
local pipeline config to an absolute private JSON file path, for example
`/absolute/private-evaluation/recipient.json`. Its contents are:

```json
{"open_id": "<authorized recipient open_id>", "tenant_key": "<verified tenant_key>"}
```

Use [recipient.example.json](recipient.example.json) as the tracked template;
replace its placeholders only in the private copy. The same selected identity
is the allowed destination for both test messages and document collaborator
access grants. Selecting a recipient does not itself send, share, seed a chat,
or authorize an action beyond the task.

The reusable checkout-local files are `.tmp/evaluation-private/recipient.json`
and `.tmp/evaluation-private/live.local.json`. The latter is a config overlay:

```json
{"live": {"recipient_file": "/absolute/worktree/.tmp/evaluation-private/recipient.json"}}
```

Merge that overlay's `live` object into a new experiment's `live` configuration
before freezing it. It is not a complete pipeline configuration or an implicit
default. Keep identity lookup results and verification proof in the same private
directory; never put the selected name, ID, or chat ID in the tracked template.

Restrict the file to its owner (`chmod 600`). Keep it outside version control;
`.tmp/` is ignored for local configuration. Omit this setting for the bundled
read-only Contact task. The file selects a destination but does not verify it or
authorize an action by itself. The trusted fixture driver must independently
verify that stable identity in the effective tenant and destination membership
before delivery, including notifications, invitations and shares. No display-name
matching or group with additional accounts is sufficient.

Delivery task manifests must set `fixture.requires_recipient: true`. The pipeline
blocks those tasks before setup when no recipient is configured, and before the
model starts when the driver does not return an exactly matching
`verified_recipient` object. Only a verified recipient reaches the model's
fixture as `authorized_recipient`; otherwise the adapter forbids other-recipient
delivery. The task must still explicitly authorize each action. Read-only tasks
can run without recipient verification. Custom model adapters must enforce the
same rule. The bundled evaluators have no IM, group-chat or delivery cases;
their implementations and manifests were removed under the user's permission
boundary. Generic recipient checks do not enable a delivery workflow.

Recipient file bytes are included in experiment/resume fingerprints, so changing
the destination requires a new output directory. Reports retain fingerprints,
not recipient fields; controller state and model workspaces remain private.
These are test-only restrictions and must never limit the shipped CLI.

## Live fixture driver protocol

`live.driver` is an absolute argv array for a trusted controller, separate from
the model runner. The bundled implementation is
[live_contact.py](live_contact.py). It uses a pinned production CLI for verified
current-account reads, checks the Feishu brand and independent account/tenant
binding, then re-reads Feishu to verify the agent's answer.

For other tasks, supply a driver implementing these phases. It reads one JSON
request on stdin and writes one JSON response on stdout. Every request contains:

```json
{
  "protocol": 1,
  "backend": "live-feishu-v1",
  "phase": "setup",
  "state_dir": "/private/controller-state",
  "workspace": "/private/controller-state/workspace",
  "cli_config_dir": "/protected/feishu-cli-config",
  "cli_data_dir": "/protected/feishu-cli-data",
  "recipient": null,
  "task": {"id": "task-id", "prompt": "User goal", "fixture": {}, "expected": {}},
  "trial": 0,
  "arm": "official"
}
```

`setup` and `cleanup` include `trial` and `arm`. `verify` receives the same task,
state directory and workspace, independently of the model's success claim.
`recipient` is the private configured `{open_id, tenant_key}` object, or `null`.

- **setup:** Bind the effective account/tenant and verify creation visibility
  before any write. Create an equivalent fresh, private fixture for each arm and
  repetition, persist run-owned IDs immediately, and write any task inputs in
  `workspace`. Return `{"backend":"live-feishu-v1", "ready":true,
  "account_verified":true, "privacy_verified":true, "prompt":"Goal with live source links"}`.
  The prompt supplies task data and authorization, without prescribing CLI
  commands or leaking the expected answer. Preserve the declared answer schema.
  For delivery tasks, independently verify `recipient` and return the matching
  `verified_recipient` object; do not merely echo configuration as verification.
- **verify:** Read the agent's `answer.json` and artifacts, independently query
  Feishu, and check the task outcome and preservation conditions. Return
  `{"backend":"live-feishu-v1", "success":true}` only on established success.
  An unsupported valid strategy returns `fixture_unsupported:true`; its attempt
  and cost remain inconclusive. Never substitute a success claim or simulator
  response for live evidence.
- **cleanup:** Reconcile remote outcomes, verify ownership, and clean up only
  resources created by this attempt. Return `{"cleaned":true}` only when cleanup
  is confirmed. A read-only fixture with no remote objects can return this
  directly. Unknown mutation outcomes must never be replayed; return
  `mutation_outcome_unknown:true` when unresolved.

Driver subprocesses receive a minimal environment with no inherited tenant
credentials, proxies or developer profiles. Use the explicit protected CLI
configuration and data paths for API authentication. The driver owns setup validation,
private recovery evidence and domain-specific verification; a declaration of
backend support alone is not an implementation. Drivers must not send content
or grant access outside the testing authorization boundaries. If privacy,
identity or necessary scopes cannot be verified, block before model billing.

Controller state and diagnostics remain under an owner-only `.private/`
directory. Keep all evaluation outputs local: they can contain private inputs,
model artifacts and resource IDs. Publish only a separately sanitized, add-only
evidence record, sealed with `python3 scripts/next_layout.py seal`.

A `RECONCILE` marker is written before setup. Failed/uncertain setup leaves it
in place without starting a model. Cleanup failure retains the attempted run's
known provider cost, stops execution and leaves the marker. A crash also leaves
the harness in-flight marker. Inspect the private state, reconcile actual remote
outcomes and accounting, and confirm cleanup before manually resolving markers;
resume never retries those mutations automatically.

## Other model harnesses

An experiment may use `runner_config` instead of native `codex`/`auth_home`:

```json
{
  "runner": ["/absolute/python3", "/absolute/model_adapter.py"],
  "mode": "real",
  "accounting_method": "Pinned harness provider per-turn usage; cached input included",
  "tokenizer": "provider-undisclosed",
  "tools": ["shell", "files"],
  "permissions": "workspace-write; real Feishu; private task resources only",
  "supported_backends": ["live-feishu-v1"]
}
```

The adapter implements `harness.py` protocol 1, including fresh contexts,
symmetric tool access, the supplied selected CLI/guidance, and real Feishu
configuration at `fixture.cli_config_dir` and `fixture.cli_data_dir`. It must protect controller state from modification and keep expected verification
data and credential contents out of model context. Native runs use a workspace
write sandbox with network enabled and the explicit CLI configuration/data directories
as additional writable roots for token refresh/locks. Custom adapters must
enforce their advertised permissions. No alternate-harness
implementation is bundled or claimed validated.

Return the exact requested `model` and `tokenizer`, `usage_source:"provider"`,
`accounting_complete`, `claimed_success`, and provider `turns` containing
`input_tokens`, `cached_input_tokens`, `output_tokens`. Report all billed turns,
including tool loops, failures and recovery. Input already includes cached input;
output includes billed reasoning where the provider counts it. Avoid counting
cumulative events twice. Interrupted streams retain completed known spans with
`accounting_complete:false`; never estimate missing totals. Optional nonnegative
`api_calls`, `retries` and `command_executions` remain diagnostics.

Controlled adapters and synthetic usage belong in tooling tests and are rejected
by the current evaluation pipeline. Offline tests exercise the real-backend
protocol with mock processes; their counters are never evaluation evidence.

## Reports and resume

Each experiment/suite/section retains `batch-01.json` with attempt outcomes,
costs, timing and provenance. Top-level `summary.json`, `summary.csv` and
`REPORT.md` show coverage and separate workflow/diagnostic comparisons. Progress
and checkpoint summaries go to stderr without raw prompts, API output or credential contents.

Token difference is `1 - next_tokens_per_success / official_tokens_per_success`;
each numerator includes every attempted run's cost. Failed attempts are retained.
Incomplete accounting stays a lower bound. Incomplete schedules, inconclusive
outcomes and zero successes have no finite comparison. Matched-success pairs are
supplementary, never a replacement for all-attempt costs. Do not pool models,
harnesses, workloads or task categories, infer prices, or treat a small sample
as population parity or exhaustive live coverage.

Repeat the identical `--execute` command to resume completed attempts, including
failures and unknown usage, without rebilling them. Changed configs, manifests,
driver bytes, binaries, public source/guidance, adapters, evaluator or Python
require a new output directory. Credential contents are never hashed. Create
`STOP` in the output root to stop between attempts; an active attempt completes
and checkpoints first. Remove it to resume the same schedule.

`--report-only` needs only `--output`, uses the retained plan and rejects
selection/budget overrides. It never contacts a tenant or model. Exit 0 means
prepared/reported, or fully completed execution with verified outcomes and
complete accounting; 1 retains task/accounting failures or an incomplete stopped
schedule; 2 means blocked/invalid/changed inputs or reconciliation required.
