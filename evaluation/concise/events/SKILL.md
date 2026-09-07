---
name: lark-event-concise
version: 1.0.0
description: Listen to Feishu/Lark real-time events as NDJSON, including messages, reactions, chat changes, approvals, tasks, meetings, recordings, minutes, bot menus, and whiteboards. Use for bounded samples or long-running event consumers, not historical API queries.
---

# Event listening

Use the shared Lark guidance for configuration, authentication, identity selection,
permissions and operation safety. Each consumer accepts one exact EventKey;
multiple keys require separate processes sharing the local bus. Comma-separated
keys and wildcards are unsupported. Avoid narrowing the stream unless requested.

`lark-cli event list [--domain DOMAIN] [--json]` lists the current catalog. Domains
are application, approval, board, card, im, minutes, task and vc.
`event schema KEY --json` supplies parameters, allowed identities/scopes,
`resolved_output_schema` and `jq_root_path`. Use these when the key or field
semantics are uncertain; retain field descriptions when inspecting schemas.

`event consume KEY` streams one JSON value per line. `--param key=value` is
repeatable; multi-values can be comma-separated. Unknown parameters fail with
the legal names. `--jq EXPR` filters/transforms each event; empty results skip it.
`--max-events N` counts successful emits; `--timeout D` limits elapsed time, and
the first bound reached wins. Zero means unlimited. `--output-dir DIR` saves
individual JSON event files subject to the CLI's path policy. `--as user|bot|auto`
selects an allowed identity. Keep stderr separate from stdout. Avoid `--quiet`
when integrity/readiness matters: it suppresses ready/exit/drop diagnostics.

For subprocesses, wait for `[event] ready event_key=KEY` on stderr rather than
sleeping. Positive event/time bounds ignore stdin EOF. An unbounded non-TTY
consumer exits on stdin EOF, so keep stdin open if it must continue. Normal
shutdown uses a bound, SIGTERM/Ctrl+C, or EOF. Do not use SIGKILL: VC, Minutes
and Board keys can leak their server subscription when cleanup is skipped.

Exit 0 with `reason: limit`, `timeout` or `signal` is normal; check whether it
actually fulfilled the requested event count. Startup/runtime failures use a
JSON error envelope on stderr. Branch on `error.type`/`subtype`, use `param`,
`missing_scopes` and `hint`, and preserve identity diagnostics. Reference exits
are API business error 1, validation 2, auth/scope 3, network 4, internal 5.
`event status [--current] [--json] [--fail-on-orphan]` inspects buses.
`event stop [--app-id ID|--all] [--force] [--json]` stops the bus; force permits
active-consumer shutdown and a later forced kill, so assess affected consumers.

A schema root of `.` means flattened fields; `.event` means a native V2 envelope.
Do not blindly apply `fromjson`: processed text is often already decoded. A jq
runtime error skips an event with a warning rather than terminating the stream.
Semantic formats distinguish IDs, email and timestamps even when all are strings.

## Domain semantics

- IM message receive is flattened: sender ID is an open_id, and normal text/post/
  image/file/audio content is rendered text; interactive cards retain a JSON
  string. Inspect message type before parsing content. Message IDs identify the
  message; delivery IDs identify the event. Read/reaction/chat/member events use
  native envelopes. Card action callbacks expose structured action/form/context
  data; use the matching card-action reply guidance for responses.
- Application `application.bot.menu_v6` uses bot identity, flattened menu/operator
  fields, no params, and optional jq filtering by menu key/operator.
- Approval instance/task status events require user identity and their respective
  `approval:instance:read`/`approval:task:read` scopes. Output is flat. The optional
  `subscription_type` registers INVOLVED_APPROVAL, MANAGED_APPROVAL, or both when
  omitted; a single value, comma-list or JSON string array is accepted. These are
  durable relations with no exit unsubscribe. It is setup selection, not an event
  field or local filter; one event can match both, so deduplicate event IDs when
  needed. Two approval keys still require two consumers. Partial setup identifies
  acknowledged and failed relation types; preserve that evidence when recovering.
  Use instance for forms/orders, task for approval to-dos/approver operations, and
  both when the user asks broadly for approval events. Involved means requested/
  approved/participated; managed means definitions the user manages. Broad or
  unspecified relation coverage can register both; ask only if that is materially
  harmful. Do not split the same key into two consumers merely for relations, and
  do not treat approval_code as a subscription dimension. Canceling future
  durable delivery requires a separate unsubscribe operation outside the consumer.
- Task `task.task.update_user_access_v2` supports user/bot with `task:task:read`.
  User sees accessible tasks; bot sees app-responsible tasks. It is single-consumer
  per local subscription and creates a durable relation without unsubscribe.
  Native `.event.task_guid` and `.event.event_types` report create/delete, summary,
  description, completion, assignee, follower, reminder, or start/due changes.
- All seven VC keys use user identity and flat output. Meeting started/joined/ended
  require `vc:meeting.meetingevent:read`; note generated requires `vc:note:read`;
  recording started/transcript/ended require `vc:recording:read`. They subscribe
  on first consumer and unsubscribe on last. Meeting times are rendered local
  RFC3339, not raw Unix seconds. Notes may come from meetings, recordings or uploads
  and can enrich document artifacts. Note/verbatim tokens may be empty when detail
  is unavailable, and note_source appears only for meeting sources. Recording
  events retain only recording_bean
  sources; transcript items preserve speaker, text, sentence IDs and times.
- Minutes generated requires user identity and `minutes:minutes.basic:read`, with
  subscription cleanup. Title enrichment may degrade to empty on detail failure;
  token/delivery fields remain. Source metadata is present only for meeting-sourced
  minutes and does not depend on enrichment success.
- Whiteboard updated supports user/bot with `board:whiteboard:node:read` and manage
  access to the board. It requires `-p whiteboard_id=TOKEN`; subscriptions are per
  board, so preserve that resource scope. Native events carry board and operator
  triple-ID fields; user_id availability depends on granted contact scope. Operator
  IDs form an array: collaborative edits can combine several editors in one event.

The frozen official `lark-event/references/` documents retain full field tables,
source-type details and optional jq recipes for each domain; use the relevant
reference when those details matter. Shared identity/permission guidance and the
IM card-action reply reference remain applicable. This is an English condensation
of that distribution, with no task-specific examples or next-only flags.
