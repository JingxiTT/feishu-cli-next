# Events and streaming

Status: offline verification passed; live verification and comparative
agent tasks remain open. No live subscription was started. The reference is the
unchanged sibling `feishu-cli-review` at `7fd6ef3`; before this section's edits,
all Go source in `events/`, `cmd/event/`, `shortcuts/event/`, and
`internal/event/` matched that sibling byte-for-byte.

## Complete mapping and ownership

The eight frozen command entries are retained: `event`, `event list`,
`event schema`, `event consume`, `event status`, `event stop`, hidden
`event _bus`, and hidden compatibility `event +subscribe`. The events owner
owns projections, the compatibility shortcut, command assembly for this domain,
its process tests, and this report. Shared transport, catalog, bus, sinks and
receipt mechanics retain their existing internal owners. Mail `+watch` is a
separate Mail workflow, not one of the 25 catalog keys.

| EventKeys | Output and lifecycle contract | Evidence owners |
| --- | --- | --- |
| `application.bot.menu_v6` | Processed operator/menu/timestamp metadata; no subscription API | `events/application`, frozen output and process replay |
| `approval.instance.status_changed_v4`, `approval.task.status_changed_v4` | Processed IDs/status/user metadata; user-only, one or both normalized INVOLVED/MANAGED relation subscriptions; durable relations are not unsubscribed on exit | `events/approval`, both process replays and recovery units |
| `im.message.receive_v1` | Processed message, rich content conversion, mentions, sender/thread/reply IDs and delivery timestamps; no subscription API | `events/im`, frozen output and process replay |
| `card.action.trigger` | Processed action/form/selection/context metadata and card content enrichment; callback preflight; single consumer | `events/im`, frozen output, process replay, preflight and single-consumer units |
| `im.message.message_read_v1`, `im.message.reaction.created_v1`, `im.message.reaction.deleted_v1` | Native V2 envelope and typed SDK schema; no subscription API | `events/im`, schema units and all-key process replay |
| `im.chat.member.bot.added_v1`, `im.chat.member.bot.deleted_v1`, `im.chat.member.user.added_v1`, `im.chat.member.user.deleted_v1`, `im.chat.member.user.withdrawn_v1`, `im.chat.updated_v1`, `im.chat.disbanded_v1` | Native V2 envelope; member/operator identities and change metadata; no subscription API | `events/im`, schema units and all-key process replay |
| `minutes.minute.generated_v1` | Processed minute token/title/source metadata; user-only, per-event subscription/unsubscription and detail enrichment | `events/minutes`, frozen output and process replay |
| `task.task.update_user_access_v2` | Native V2 envelope; user/bot, single consumer, durable subscription with `user_id_type=open_id`, no unsubscribe | `events/task`, units and process replay |
| `vc.meeting.participant_meeting_started_v1`, `vc.meeting.participant_meeting_joined_v1`, `vc.meeting.participant_meeting_ended_v1` | Processed meeting IDs/calendar reference/times; user-only, per-event subscription/unsubscription | `events/vc`, frozen output and process replay |
| `vc.note.generated_v1` | Processed note/source and both document artifact references; bounded detail enrichment retries and subscription/unsubscription | `events/vc`, note retry units, frozen output and process replay |
| `vc.recording.recording_started_v1`, `vc.recording.recording_ended_v1`, `vc.recording.recording_transcript_generated_v1` | Processed recording source/key and transcript speaker/text/times; unsupported sources dropped; subscription/unsubscription | `events/vc`, frozen output and process replay |
| `board.whiteboard.updated_v1` | Native V2 envelope; resource match and subscription identity include normalized whiteboard ID; matching resource subscribe/unsubscribe | `events/whiteboard`, scope units and process replay |

All 25 have compiled schemas and process discovery/dry-run coverage. All 13
processed outputs are compared with the unchanged reference output snapshot.
All 12 native outputs are replayed with Unicode, null, empty strings, leading-zero
strings, an integer beyond IEEE-754's exact range, and unknown future fields.
The process suite asserts the exact preserved integer token as well as JSON
content. Existing schema closure/instance tests cross-check projection outputs
against schemas; the process fixtures do not replace those tests.

## Behavior and deliberate changes

Command/flag, risk, identity, scope, output payload, subscription identity and
schema contracts are retained. `event consume` retains NDJSON, `--param`, `--jq`,
`--output-dir`, `--quiet`, `--max-events`, `--timeout`, `--dry-run`, and `--as`.
`event list --domain/--json`, `event schema --json`, status
`--current/--json/--fail-on-orphan`, and stop
`--app-id/--all/--force/--json` retain their reference contracts. The compatibility
shortcut keeps custom/catch-all event types, regex filtering/routing, file and
stdout destinations, compact/pretty modes, quiet mode and the single-instance
lock. It remains executable and hidden.

Product changes add mutation evidence, honest approval recovery, and the documented
JSON stop failure exit. `event stop --json` previously returned before its failure
check, exiting 0 even with refused/error results. It now retains the complete
JSON result on stdout and exits 2, matching text mode and command help.
`cmd/event/runtime.go` marks its lifecycle POST requests with the shared exact
method/path mutation marker. This covers subscribe and unsubscribe even though
the long-running command is read-risk. The task URL's query is retained on the
request but excluded from the marker's exact path match. Encoded board IDs
match the HTTP request's decoded path while retaining the encoded wire URL. Per-event enrichment
GETs create no receipts; there is no second journal or per-event persistence.
Compatibility mode retains the existing transport behavior. The receipt count
is bounded by actual lifecycle requests, not the number of delivered events.

Approval failures now say registration was not confirmed and may have applied;
they no longer infer non-application from a transport/API error. Previously
acknowledged relation types remain visible. Retry guidance requires verifying
server-side state, and only recommends the failed relation after confirmed
non-application. Typed category, subtype and cause remain unchanged.

Shared projection/artifact flags remain unavailable for `event consume` streams;
use the existing explicit jq or event-file sink. Stream payloads stay independent
NDJSON values even when later cleanup fails. The terminal receipt diagnostic on
stderr and exit 20/21/22 report lifecycle rejection/partial/unknown outcomes; an
already emitted event is not removed or rewritten. A success receipt acknowledges
an API request, not that the complete remote delivery workflow has been verified.

## Lifecycle and limits

The existing bus arbitrates the first/last consumer per subscription scope.
Preparation runs after the delivery handshake; cleanup has its own bounded
context after consumption cancellation. Multiple board IDs must never share
cleanup scope. A legacy bus cannot support that scope and is refused before
registration. Compatible legacy metadata still replays through the same current
projection/sink pipeline.

Bounded runs ignore stdin EOF; unbounded non-TTY runs stop on EOF. Signals and
output pipe closure enter graceful shutdown; timeout is a normal exit. Multiple
workers retain the documented possible `workers-1` overshoot of an emit bound.
Backpressure drops are diagnosed unless `--quiet` suppresses them. SDK source
reconnection is retained and exposed through status diagnostics. Neither
reconnection nor the legacy-frame compatibility tests promise durable event
replay or exactly-once delivery. A process crash can skip unsubscribe, and
receipt evidence never authorizes automatic mutation replay.

## Section scenarios and commands

The process suite uses the production authsidecar build, a private synthetic
HTTP backend and the production local IPC protocol/address. It never launches
an external WebSocket source or forwards requests outside the local fixture.
Fixture requests assert subscribe/unsubscribe method, path and body, not merely
successful stdout. Replay inputs are synthetic copies of the reference baseline
inputs; expected processed outputs remain the original baseline file.

`tests/next_sections/events/events_e2e_test.go` covers every key's process replay,
registration and enrichment, all schemas and dry-runs, invalid keys/parameters,
identity mismatch, incompatible output controls, jq, event files, timeout,
stdin EOF, interrupt cleanup, source reconnect status, sequence-gap diagnostics,
legacy native replay, legacy resource-scope refusal, local status and safe stop
refusal. It also injects rejected/unknown subscription and cleanup responses,
checks the original emitted data and terminal exit/receipts, and asserts that
failed lifecycle requests are not replayed.

The inherited complete units additionally cover canonical metadata conflict,
subscription concurrency, normalization/fingerprints, worker bounds, broken
pipes, malformed and oversize events, source SDK contracts, backpressure,
orphan discovery, daemon startup/stop and legacy shortcut processors/routers.
The three inherited CLI E2E tests cover shortcut dry-run and typed validation
errors. Hidden `_bus` is exercised through runtime/fork/source units; the offline
process fixture deliberately does not establish a live WebSocket connection.

Run the complete boundary in credential-isolated state:

```sh
python3 scripts/next_test.py --section events --suite all --report .tmp/p2-events-all.json
python3 scripts/next_test.py --section events --suite live --allow-live --report .tmp/p2-events-live.json
```

The live command is separately selectable and currently reports **blocked**,
exit 2. There is no verified fixture harness binding current account, tenant,
private resource visibility and permitted recipients before remote subscription
and event creation. Tenant-wide VC/Minutes registrations additionally need
verified scope ownership before changes to existing subscriptions. No live
backend guarantees, external reconnect behavior, or comparative token results
are claimed from offline evidence. Required live and comparative checks keep
this section open for final acceptance; held-out tasks remain untouched.

## Recorded verification

The completed offline boundary passed 740 unit/subtest events (248 domain and
shortcut; 492 command, bus, transport and stream), plus 70 process/subtest events
(67 next-interface and three inherited deterministic regressions), with zero
skips. The final targeted process run includes the escaped-ID receipt regression.
`go vet ./cmd/event/... ./events/... ./shortcuts/event/...
./tests/next_sections/events/...` passed, owned Go files are gofmt-clean, and
`git diff --check` passed. Machine-readable counts and commands are in
[events-evidence.json](../../evidence/sections/events-evidence.json); raw isolated logs are retained in
`.tmp/p2-events-*.json`.

The command/internal event packages were additionally run with
`scripts/next_test.py`'s `isolated_env` helper rooted in `/tmp`, since their
selection was not yet wired into the Events runner at the time of execution.
An earlier ad hoc `/var/folders` temporary root exceeded macOS's Unix socket
path limit; the corrected isolated-root run passed all 492 tests. An intermediate
process build also encountered a concurrently edited OKR compile gap, resolved
by its owner before the passing boundary. Neither is counted as passing evidence.
Post-commit `make quality-gate QUALITY_GATE_CHANGED_FROM=3a7a243^
QUALITY_GATE_DIR=.tmp/events-quality-gate` passed in isolated state. It retained
non-blocking legacy naming labels (`event _bus`, `drive +task_result`) and list
limit warnings (`event list`, `drive +member-list`); event list is bounded by the
25 registered definitions. Raw gate evidence is `.tmp/p2-events-quality.json`.
Global formatting/unit/vet and release checks remain coordinator-owned; this
section has not run global Go module/license gates independently.
