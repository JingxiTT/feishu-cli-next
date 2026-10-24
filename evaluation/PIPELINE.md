# Evaluation pipeline

`pipeline.py` is the **single entry point** for a current comparison. It runs
the official CLI and CLI Next on the same user goals, preserves each attempt's
cost, and reports comparable observations for users and developers. It reuses
the existing signed synthetic fixtures, independent verifiers, fresh contexts,
balanced randomized schedules, and conservative provider accounting. No
third-party Python dependency is needed.

There is no acceptance threshold or savings gate. The report describes
verified completion and artifact effects, token use, time, API calls, retries and
failure/recovery costs. A positive difference is an observation, not a product
claim; a negative one is retained just as visibly. Historical reports and the
old `run_matrix.py` remain unchanged for provenance and are not a second
current entry point.

## Evaluation is separate from CI

| | CI correctness tests | Agent evaluation |
|---|---|---|
| Question | Does the software satisfy its contracts? | How well and at what token cost does an agent complete the work? |
| Execution | Automatically on pull requests and pushes | Explicitly launched with a pinned experiment configuration |
| Work | Unit, integration, CLI E2E and evaluation-tooling tests | Subsystem function and composed E2E task workloads |
| Model calls | None | Real harness/model calls for measured experiments |
| Result | Pass/fail/skip correctness reports | Verified outcomes, token distributions, latency and failure/recovery costs |
| Decision | Software correctness gates | Descriptive comparison evidence; no acceptance threshold or CI merge gate |

`make test` and `make evaluation-tooling-test` belong to CI. The older
`make evaluation-test` is a compatibility alias for the tooling tests. Those
tests may use synthetic token counters to test accounting; their results are
never model-efficiency measurements.

`python3 evaluation/pipeline.py ... --execute` belongs to evaluation. It is
independent of the CI job graph and is never invoked by the correctness
workflows or `make test`. A green CI run does not mean an agent workload was
evaluated. The pipeline's exit status reports process integrity and completeness;
task failures are retained observations and are not converted into a pass/fail
efficiency verdict.

The term E2E appears in both paths: deterministic CLI E2E tests exercise the
product directly, while E2E **evaluation** gives an agent a business goal and
measures its entire attempt. Neither replaces the other.

## Run one comparison

1. Build or obtain frozen official and Next binaries with their matching source
   distributions. Synthetic API tasks require fixture-enabled binaries; ordinary
   production binaries are insufficient for those tasks. Follow the existing
   [fixture setup](README.md), including its native sandbox prerequisites.
2. Copy [pipeline.example.json](pipeline.example.json) to an ignored local path,
   such as `.tmp/evaluation/config.json`, and replace its absolute paths and model
   ID. Keep credentials in the harness's protected auth store, never in config.
3. Add one experiment for each harness/model/reasoning combination. Each
   experiment has a unique `id`; its two arms share all settings and budgets.
   `harness` identifies the harness/version for reporting. Native experiments use
   the existing **Codex CLI 0.153.4** adapter; other harnesses use `runner_config`
   as described below. Native model availability depends on that harness/provider.
4. Prepare the entire matrix before billing, then execute:

```sh
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json \
  --output .tmp/evaluation/run-01
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json \
  --output .tmp/evaluation/run-01 --execute
```

Preparation validates every selected section, guidance distribution, runner
contract and extension build without model calls. It writes a plan and reports
unmeasured coverage. It does not prove provider authentication, model
availability, or a custom adapter's implementation. Execution is serial to
avoid contaminating comparisons with competing workloads. The default is
**45 development tasks, 270 attempts per experiment** with three repeats per
arm. More models multiply this cost; preparation prints the scheduled count
before execution.

The config selects two workload views:

- `workflow` is the user-facing comparison: goal-oriented work that may combine
  discovery, reads, edits, typed data, artifacts, decisions and recovery.
- `diagnostic` keeps narrow read/fidelity/discovery/scale probes that explain
  where behavior differs.

These are report sections, not pass/fail categories. They are never pooled.

Use `workloads` to select `workflow`, `diagnostic`, or both (the default). Use
`suites` to narrow the selected workloads to any of the nine suites and optional
`sections`, for example `"suites": ["business", "e2e"]`.
Omitting `suites` selects all suites in the selected workloads. An explicit
suite outside the selected workloads is an error. `repeats` must be at least two;
three is the default, not a statistical guarantee. `seed`, `budget_seconds` and
`guidance_delivery` (`file` or `inline`) apply symmetrically to every experiment.
The runner gets the model and reasoning strings unchanged.

## Workload coverage and real-use fit

Results have separate tables and workload labels in JSON/CSV; their
comparisons are never combined.

**Workflow evaluation** checks one user goal requiring several functions to
work together. It includes finding sources, interpreting their data, making a
decision and producing the final artifact or backend effect. Its full token
cost includes discovery, intermediate processing and recovery in one fresh
context. Agents may use shortcuts or any valid composition. Verification checks
outcomes and necessary data dependencies, not a prescribed command sequence.

**Diagnostic evaluation** checks a bounded capability: discovery, reading,
editing, preserving data types, transfers, installation and scale. It answers
where correctness or token use changes. A function may need several API calls;
call count alone does not make it a workflow.

For example, select only diagnostics with `"workloads": ["diagnostic"]`, or only
goal-oriented work with `"workloads": ["workflow"]` and no `suites` filter.

| Suite | Tasks | Work exercised by existing executable fixtures |
|---|---:|---|
| paired | 8 | Discovery, ambiguous Contact identities, Docs read/edit, Sheets, Base, Drive, Markdown |
| docs-hard | 3 | Tables/lists, focused large-document retrieval, styled edits |
| read | 12 | Application, Apps, Calendar, IM, Mail, Minutes, Note, OKR, Slides, VC, Whiteboard, Wiki |
| scenario | 9 | Docs, Base, Sheets, Drive, Markdown, Task, Events, Extensions, Packaging |
| scaling | 6 | Base import and Sheets export at 100, 1,000 and 10,000 rows |
| business | 2 | Approval and mindnote mutation/readback workflows |
| foundation | 2 | Unconfigured authentication and local command discovery |
| contact | 1 | Synthetic identity lookup |
| e2e | 2 | Docs + Sheets release readiness review; Docs + Task approved handoff completion |

The workflow tasks in [tasks.e2e.json](tasks.e2e.json) start from a local release brief
containing source links and business requirements, rather than endpoint/flag
instructions in the user prompt. Both use the full official shared skill and
its unchanged domain skill distribution versus Next's entry skill. Run these
through the pipeline. `run_read_batch.py` is an internal adapter and is not a
current operator entry point.

- **Release review:** read the approval document and spreadsheet, preserve a
  native typed cell/formula export, and combine their facts into a readiness
  decision and JSON review. Both source observations and the saved artifacts
  are independently verified.
- **Release handoff:** find the matching task by title, check the document's
  launch approval, then complete the task and save a cited handoff record.
  Verification rejects completion before approval evidence, changed assignees
  or unrelated tasks, and unsupported claims of success.

These are two initial composed scenarios, reusing small bounded native fixtures.
They cover the approved/ready path; no unapproved-release or blocked-release
dataset is scheduled yet. They do not establish representative real-workload
coverage. More business goals, varied datasets, negative decisions and
realistic failure conditions belong in the workflow section as independently
verified tasks.

The current workload is a useful starting point but is not yet a complete
picture of ordinary Feishu use. It is strong on shell/file execution, Docs,
Sheets/Base, Drive artifacts, contacts, task completion and independent
verification. It is weaker on message/mail/calendar triage, multi-party
coordination, negative or ambiguous decisions, pagination and permission
recovery. New workflow tasks should be written from user goals and leave the
agent free to choose commands, raw API, files or composition.

The fixtures check actual answers, signed requests, backend effects and artifacts
as applicable, rather than requiring one command sequence. Unicode, typed values,
artifact integrity, readback and preservation conditions come from the existing
contracts. Unsupported valid strategies remain inconclusive with their costs.
This is development sampling, **not exhaustive command or failure-mode
coverage**. In particular it does not prove live permission handling, arbitrary
pagination, all unknown-mutation recovery paths or production backend fidelity.
Unimplemented development placeholders and quarantined held-out tasks are never
scheduled. Extend the owning fixture, independent verifier and development
manifest together to add scenarios; the pipeline picks them up automatically.
Use `measure_surface.py` separately for exhaustive local help/schema coverage;
local output-token counts are not full-task provider usage.

## Use another harness

An experiment may replace `codex` and `auth_home` with an absolute `runner_config`
path. That JSON file declares exactly these settings:

```json
{
  "runner": ["/absolute/python3", "/absolute/my_harness_adapter.py"],
  "mode": "real",
  "accounting_method": "my-harness 1.2 provider per-turn usage; cached input included",
  "tokenizer": "provider-undisclosed",
  "tools": ["shell", "files"],
  "permissions": "workspace-write; external shell network denied; isolated synthetic sidecar",
  "supported_backends": ["offline-foundation-v1", "synthetic-contact-v1", "synthetic-read-v1", "synthetic-scenario-v1"]
}
```

This is a **full runner adapter**, not an arbitrary harness CLI command. It must
implement the existing `harness.py` protocol 1: read one JSON request from stdin,
write one JSON response to stdout, and use the supplied temporary workspace. Its
request contains `task_id`, `trial`, `arm`, `prompt`, `fixture`, `cli`, `skill`,
`settings` and `fresh_context`; expected outcomes are withheld. Write the answer
and fixture receipts expected by the existing verifier in that workspace. The
runner executable and absolute file arguments are hashed. The environment strips
inherited tenant credentials, proxies and user profiles. Use explicit protected
harness authentication paths; do not put tokens in argv.

A real response contains:

```json
{
  "claimed_success": true,
  "usage_source": "provider",
  "model": "the-exact-requested-model",
  "tokenizer": "provider-undisclosed",
  "accounting_complete": true,
  "turns": [
    {"input_tokens": 1200, "cached_input_tokens": 400, "output_tokens": 100}
  ],
  "api_calls": 2,
  "retries": 0,
  "command_executions": 3
}
```

Report **all** provider turns, including skill loading, tool results, failures
and recovery. Input includes cached input; never add cached input twice or
subtract it from total. Output includes provider-billed reasoning tokens where
included by that provider's output counter. Avoid double-counting cumulative
usage events. A timeout or incomplete stream returns completed known spans with
`accounting_complete: false`; never estimate a missing total. Do not fabricate
provider usage from a tokenizer or invent a price conversion.

For synthetic test adapters use `mode: "controlled"` and
`usage_source: "synthetic"`; these runs never receive a model-comparison score.
`tests/next_evaluation/test_pipeline.py` demonstrates the complete controlled
prepare/execute/resume/report lifecycle, including failed outcomes.

Adapters must enforce fresh contexts, symmetric tool access and the advertised
sandbox, deny real-tenant access, and own fixture setup/cleanup and independent
receipts. Declaring a backend is not implementing it. The native reference is
`codex_runner.run` plus `section_fixture.py` and `scenario_fixture.py`; scenario
observation hooks currently understand native Codex events, so another harness
must provide equivalent independently validated observations. Start with only
`offline-foundation-v1` and `"suites": ["foundation"]` until broader adapter tests
pass. No Claude/OpenCode-specific adapter is bundled or claimed validated here.

## Results, interruption and reproducibility

Each experiment/suite/section retains its `batch-01.json` with hashes, per-attempt
outcomes, input/cached/output totals, latency, request/retry diagnostics, failure
labels and matched-success pairs. Top-level `summary.json`, `summary.csv` and
`REPORT.md` provide task/section comparisons and planned/completed coverage,
separated into workflow and diagnostic workloads.

The descriptive token difference is `1 - next_tokens_per_success /
official_tokens_per_success`, where each numerator includes **every attempted
run's cost**, including failures. The report also shows verified outcome
changes. A positive token difference with worse outcomes is not a parity claim.
Unknown usage stays a lower bound; incomplete schedules, inconclusive outcomes,
zero successes and controlled runs have no finite comparison. Matched-success
pairs are supplementary and exclude failures, so they are never substituted
for the all-attempt observation. No totals are pooled across harnesses,
workloads or task categories. Token prices and dollar savings are not inferred.

Repeat the same `--execute` command to resume. Completed attempts are reused,
including failures and unknown usage; they are never retried to improve a score.
An unresolved in-flight marker stops execution for reconciliation. Changing
config, runner bytes, binaries, public source/guidance, evaluator or Python
requires a new output directory. The plan records hashes, not credential contents.
Freeze source archives outside the output directory; do not place run outputs
inside either measured source tree.

Create `STOP` in the top-level output directory to stop between attempts in the
active section and before later jobs. An active attempt finishes and checkpoints
its cost. Remove `STOP` to continue the identical schedule. Preparation and
report-only do not bill models. To render retained results without checking or
launching the current harness:

```sh
python3 evaluation/pipeline.py --config .tmp/evaluation/config.json \
  --output .tmp/evaluation/run-01 --report-only
```

Exit codes: 0 means prepared/report-rendered or all executed attempts completed
with complete accounting; 1 means executed task/accounting failures or an
incomplete stopped schedule; 2 means invalid/changed inputs or a job requiring
reconciliation. The numerical outcome remains descriptive and has no target.
Task failures continue the matrix; configuration/interruption errors stop it.
Raw model/tool output is not retained by the shared harness. Custom adapters
must meet the same sanitization contract. Keep exploratory runs in `.tmp/`;
publication is a separate add-only evidence record sealed with
`python3 scripts/next_layout.py seal`.

## CI tests for the evaluation tooling

```sh
make evaluation-tooling-test
```

These tests include a process-level custom-adapter pipeline run and safe resume.
They do not bill a model or contact a real tenant. Provider token comparisons
require an explicitly executed, configured matrix and are not results of these
software tests.
