# Whiteboard capability section

Status: offline implementation and section suite pass. Live rendering, resource
identity/permission preservation, and comparative evaluation remain open. A skipped
live fixture is not passing evidence.

## Reference and ownership

The unchanged sibling is pinned at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`.
Before editing, `diff -qr ../feishu-cli-review/shortcuts/whiteboard shortcuts/whiteboard`
reported no differences. Inspection included the frozen registrations, coverage
manifest and source inventory, all whiteboard shortcut registrations and their
request/response types, raw/code/image/SVG exports, path/error helpers, unit and
legacy E2E contracts, and whiteboard event registration/matching/subscription.
There are four leaf shortcuts including the Docs compatibility command. The
frozen tree has no generated `board` methods; raw API is not a substitute for
these workflows.

Whiteboard owns `shortcuts/whiteboard`, `tests/cli_e2e/whiteboard`, this report and
`tests/next_sections/whiteboard`. `docs +whiteboard-update` is implemented and
process-tested here, although the frozen inventory assigns its discovery entry
to Docs. Docs owns creation/embedding and must coordinate the private live fixture.
Events owns `board.whiteboard.updated_v1` projections and process lifecycle tests;
the section runner also executes its inherited unit regressions. No shared runtime,
root, event or skill changes are needed.

## Full capability mapping

| Command/capability | Preserved contract and acceptance evidence |
| --- | --- |
| `whiteboard +update` | Raw OpenAPI nodes and wrapped whiteboard-cli `data.to=openapi` / `data.result.nodes`; Mermaid, PlantUML and SVG source; explicit token, source inline/file/stdin, default raw format, client idempotency token, append and overwrite. Units cover all selectors/envelopes, validation and malformed/API responses. Process tests cover all four formats under user and bot, exact request source/query/body, full numeric/rich node fidelity, default raw, file handoff and observed append/overwrite/clear state. Stdin handling remains the shared runtime input contract. |
| `docs +whiteboard-update` | Same shared implementation, flags, risk and scopes. Process tests exercise all four formats under both identities through this command name, including dry-run and receipt evidence. |
| `whiteboard +export` raw | Full opaque node objects, nested future fields, native IDs, large numbers, exact decimals, Unicode, leading-zero strings, null, empty strings and arrays. Direct JSON, explicit raw JSON file, extension inference, overwrite protection and shared result artifact/projection retained. Process tests export/import files and reject the generic result container as authoring input. |
| `whiteboard +export` source | Recognized Mermaid/PlantUML syntax and `.mmd`/`.puml` files; missing nodes, absent recognized code and multiple code blocks retain distinct messages. The command does not guess one source from an ambiguous board. Units/process cover both source languages and empty/unsupported/multiple-source cases. |
| `whiteboard +export` SVG | POST export with `export_type=svg`, base64 decoding, exact SVG text or `.svg` file. Read risk bypasses mutation journaling despite POST. Units cover HTTP/API failures, malformed base64/JSON, pretty and file output; process tests check bytes, legacy alias and no applied receipt. |
| `whiteboard +export` preview | GET image download through runtime/FileIO; response PNG/JPEG content type owns extension, explicit mismatch fails, content-disposition cannot redirect output. Units cover PNG/JPEG, directory/explicit paths, unsupported MIME, collisions and typed errors. Process tests independently read JPEG bytes, existing-file refusal, overwrite and mismatch safety. |
| Hidden `whiteboard +query` | Legacy `output_as=image\|svg\|code\|raw` selectors map to the same export behavior, remain executable and hidden. Process tests use every legacy selector; inherited units pin registration and selector validation. |
| Domain discovery/metadata | All existing user/bot identities, `board:whiteboard:node:create` versus `board:whiteboard:node:read`, write/read risk, required flags, dry-run, JSON/pretty/jq compatibility remain. No command/flag removal. Shared result controls are additive. |
| Events dependency | Per-whiteboard subscription identity and exact event filter; native whiteboard ID and operator open/union/user IDs; paired subscribe/unsubscribe cleanup and required console event. Inherited event tests run here; process/stream acceptance belongs to Events. |

## Corrections and effect boundaries

Raw authoring now decodes numbers with `UseNumber`, fixing float64 corruption of
values such as `9007199254740993` and `1.234567890123456789`. Both supported raw
input envelopes retain all nested values. The decoder still rejects trailing JSON
or garbage with a typed `--source` validation failure. Update dry-run now includes
its supplied `client_token`, matching execution for every format. The previously
unselected required-selector regression is named for the deterministic section
runner so it executes at the boundary.

These are fidelity and preview corrections, not interface replacements. Existing
native node/image/style/source payloads remain intact; no simplified node schema
or conversion is introduced. A generic `--save-result` container is not an import
payload and is rejected without dispatch. Exported raw `--output` JSON is the
existing explicit import representation; backend-generated identities after
creation remain subject to backend semantics.

An unchanged-content no-op is deliberately **not claimed**. The reference update
API creates nodes; overwrite explicitly replaces content. Source equality alone
does not prove the complete current board is identical, and the write-only scope
does not prove read access or current edit permission. A local equality shortcut
would bypass backend permission checks and could omit unrelated nodes. Append
always dispatches; explicit empty overwrite dispatches even when already empty.
Process tests observe 1→2→1→0→0 nodes through independent reads and five writes.
Expand no-op behavior only with an exact complete-state and permission-aware
backend contract; do not infer it from image/source equality. There is no
conditional revision flag or concurrency guarantee from receipt locks.

Receipts are supplied by the shared runtime before dispatch. Synthetic process
faults prove a permission rejection is confirmed-unapplied, malformed/lost
responses remain unknown without retry, and item failure evidence is not promoted
to complete success. An acknowledged API envelope is request evidence, not proof
of live rendering fidelity or preserved backend identity. Unknown overwrite is
never automatically replayed.

## Evidence and remaining work

Run the complete section with:

```sh
python3 scripts/next_test.py --section whiteboard --suite all --report .tmp/whiteboard-section.json
```

The runner isolates HOME/config/credentials and uses a dead external proxy; the
new process package builds the production authsidecar CLI via `nextfixture` and
serves only a local synthetic backend. Tests exercise real process commands and
inspect exits, stdout/stderr, requests, saved bytes, fixture state and receipts.
The full result and every Go test event are retained in the report above.
Final boundary results: **138 unit/event tests passed**, **9 inherited dry-run/
regression test events passed**, and **31 process test events passed** with **one
explicitly blocked live scenario skipped**; no failures. Counts include subtests.
The implementation commit is recorded with the section handoff. The legacy JPEG live workflow is excluded because an external
whiteboard token does not prove run ownership or private creation visibility.

`TestWhiteboardLivePrivateFixture` is the separately selectable blocked scenario.
The live gate command is:

```sh
python3 scripts/next_test.py --section whiteboard --suite live --allow-live --report .tmp/whiteboard-live.json
```

It remains blocked until a harness verifies current account and tenant, private
creation visibility before writing, and run-owned cleanup. Then create a private
Docs/whiteboard fixture, import all formats, independently export nodes/source/
SVG/preview, check identity and edit permissions after replacement and clear, and
clean up only the run's resources. No other recipient is required. If collaboration
is tested, only unambiguously identified Ying Wu is authorized. Existing unrelated
boards are never edited. Native rendering, identity/permission preservation,
idempotency guarantees and interruption reconciliation remain live-unverified.

Comparative development tasks are scheduled by the coordinator; none have been
run for this section and held-out tasks remain untouched. Shared repository unit,
vet, formatting, license and committed quality gates are coordinator-level release
checks; this section does not substitute its focused suite for those gates.
