# Bounded scenario module interface

Own only your assigned `evaluation/scenarios/<section>.py`,
`tests/next_evaluation/test_scenario_<section>.py` and
`evaluation/concise/<section>/SKILL.md`. Do not edit shared harness/server/runner,
protocol files, runtime data of other sections, product code, or held-out tasks.
Do not stage/commit; evaluator integrates. No model calls. Controlled local CLI
checks are encouraged; binaries must be existing clean-archive fixtures, not
rebuilt dirty source. API server is always local, signed and non-forwarding.

Read your existing reviewed `evaluation/contracts/<section>.json`. Export:

- `TASK`: harness task dict `{id,section,prompt,fixture,expected}`. Public fixture
  contains only scenario name, TYPE-only `answer_schema`, and optionally
  `allow_synthetic_mutations:true` for the already-declared synthetic write task.
  Hidden backend state/expected values must never enter public fixture/schema.
- `class Scenario(ScenarioBase)` imported from `scenario_protocol`.
- `Scenario.dispatch(method, uri, body_bytes, headers)` returns `Reply` (same import)
  for supported real API routes, otherwise raises ValueError (server rejects).
  Common server has already verified HMAC/identity/target/body integrity.
  Validate resource IDs/query/body/identity, apply only intended synthetic state
  transitions. Support equivalent tested CLI/raw read/write strategies rather than
  prescribing a sequence. Reply accepts JSON OR bytes, HTTP status/content type,
  headers and synthetic proof metadata for independent verification. No forwarding.
- `Scenario.snapshot()` returns JSON-serializable initial/final synthetic state,
  or independent filesystem/listener observations. Captured only after model exits,
  under top-level `receipt.state`. Never trust a model-written file as server state.
- `verify(workspace, answer, receipt, expected)` returns bool using independent
  final state, actual local file bytes, signed request evidence and expected typed
  invariants. Receipt has scenario, requests[{route,status,returned_ids,proof?}],
  state. An answer/success claim without the required observation must fail.
  No live/API network calls in verifier; reject unintended effects, not legitimate
  read-only POSTs or local CLI journals. Expected constants are hidden from model.

Optional lifecycle methods inherited from ScenarioBase:
- setup(workspace): stage public inputs (e.g input.md, project/README.md) and
  start a local Unix event listener if needed. Keep expected/final backend state in
  memory, not readable workspace files. Set self.workspace when required.
- environment(): extra SYNTHETIC env (e.g TZ); never read/copy user credentials.
- unix_sockets(): allowed temporary local paths for the Codex sandbox if needed.
- close(): stop/join listeners before snapshot; make repeat calls harmless.

Offline Packaging may return no API requests, but verifier must inspect the
actual installed tree/content and preservation state independently. Extensions
may expose an additional `build_host(source_root, output_path, next_mode)` helper
for its small external-module wrapper from the reviewed contract; caller provides
clean archive source paths. No global builds/installation. The model still uses
one selected CLI with the same public host wrapper per arm; version difference
only selects the next marker. Events may use the approved bounded Unix listener
and must prove observed payloads + termination, not merely a prewritten artifact.

Tests: exercise state transitions, tampered/forged/wrong answers, rejected foreign
IDs/unintended changes and real CLI success where a supplied fixture binary is
available. Unit tests must be executable without optional binaries. Use
sys.executable for subprocesses, preserve JSON numeric/boolean distinction and
Unicode/empty values. Capture only sanitized checks in reports. Propose narrowly
needed protocol changes instead of silently weakening task invariants.

Concise Arm2: translate/condense the matching frozen official domain skill and
its relevant references into equivalent English. Keep all domain decisions,
identities/scopes, content/operation limits and safety semantics; no fixture IDs,
answers, task-specific coaching, new next-only flags or forced extra discovery.
Original reference files/sibling skill distribution remain available unchanged.
If no matching official skill exists, disclose the actual affordance fallback.

Transient trusted-observation hooks now supported by common runner:
- `selected_cli` property is assigned before setup (actual archived selected host
  binary); optional `selected_skill_source` is explicitly supplied for Packaging.
- `observe_execution(item)` may consume each real completed Codex command event
  including command, exit_code, aggregated_output. No raw item is published.
  Do not fabricate separate stderr, process-child exit codes, or filesystem access
  counters. Parse only invocations you can establish; record scope limitations.
- `audit_output(raw_jsonl)` optional bool result is written by runner as
  `receipt.output_audit={checked:true,passed:<bool>}` before raw data is discarded.
- Packaging receives trusted `isolation_proof` with
  `protected_home_writes_denied` and `network_denied`, based on an actual configured,
  controlled-tested OS permission profile. There is no syscall violation counter.

Local ignored `.tmp/evaluation-paths.json` contains clean build paths/revisions for
controlled binary checks; it contains no auth home or credential data. Never copy
its host paths into committed evidence. Published evidence uses placeholders.
