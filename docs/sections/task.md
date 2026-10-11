# Tasks capability section

Status: offline implementation and complete section suite verified; P2 acceptance
remains open for authorized live verification and comparative agent evaluation.

Reference: unchanged sibling `feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; the task shortcut tree and frozen
`baseline/registrations.json.gz` / `meta_data.json.gz` were inspected before
changes. There are 17 shortcuts and 34 generated v2 methods. All retain their
original command paths, flags, identities, scopes, and rich payload escape hatches.
No endpoint is replaced with an unreviewed raw forwarding command.

Owner: Tasks owns `shortcuts/task/`, `tests/cli_e2e/task/`,
`tests/next_sections/task/`, task affordance and this report. Shared runtime,
output, receipts, registration and catalog owners remain Foundation. Task event
projection belongs to Events; its existing regressions run in the task suite. The sole frozen event is
`task.task.update_user_access_v2`: native payload, user/bot identities,
`task:task:read`, pre-consume subscription initialization and single-consumer
behavior remain unchanged; live streaming acceptance belongs to Events.

## Complete mapping

Every row maps `lark-cli task ...` to the same `feishu-cli-next task ...` suffix.
The shortcut unit suite is retained in full. Process evidence runs a production
authsidecar binary against a synthetic backend; nothing is forwarded to Feishu.

| Shortcut | Preserved capability/options and evidence |
| --- | --- |
| `+create` | Summary, description, generic nested data, user/app assignee and follower, due/date/relative/millisecond time, tasklist/applink, client token. Body/output/time units; resource/recipient process test. |
| `+update` | All task object fields, explicit flag overrides, multi-GUID/applink update, server-confirmed fields. Update/body/no-op units; full replacement process lifecycle and dry-run regression. |
| `+complete` | Read status, mark completion, already-complete no-op, normalized GUID/applink. Complete units and repeated complete process flow. |
| `+reopen` | Reopen completion and unchanged timestamp no-op. Output/no-op units and repeated reopen process flow. |
| `+assign` | Add/remove assignees, user/app inference, add-only client token. Assignment units and recipients plus partial/unknown process tests. |
| `+followers` | Add/remove followers, user/app inference and client token. Follower units and recipients process test. |
| `+comment` | Literal comment content and task resource identity. Comment live scenario retained; literal content process request verified. |
| `+set-ancestor` | Assign parent and explicit detach when ancestor omitted. Ancestor units and both process variants. |
| `+reminder` | Minute/hour/day replacement, removal, no-existing removal, identical setting no-op. Reminder units and set/no-op/remove process cases. |
| `+get-my-tasks` | Optional completed true/false vs omission, query exact-first filtering, creation/due range filtering, page limit/all/token. Existing filter/pagination units, legacy dry-run and real two-page process. |
| `+get-related-tasks` | Include complete, creator/follower filters, current user identity and cursor/page bounds. Related-task units and real two-page process. |
| `+search` | Query, creator, assignee, follower, completed, due range, pagination and detail enrichment fallback. Search/filter/pagination units, dry-run regression and two-page process. |
| `+upload-attachment` | Single <=50MB file, real filename/bytes, resource/applink, task/task_delivery and user ID type. Full FileIO/size/error units, dry-run and multipart process. |
| `+tasklist-create` | Name, editor members, initial task array, assignee shorthand, concurrent child creation, partial child errors. Create/output/error units and list-plus-child process. |
| `+tasklist-search` | Query, creator, creation range, page controls and detail enrichment fallback. Search/pagination units and two-page process. |
| `+tasklist-task-add` | Multiple tasks, tasklist and optional section, per-task partial errors. Membership/batch units and section-placement process. |
| `+tasklist-members` | Read, add/remove editors, exact ID-set difference and unchanged set. Members units and read/add/remove/no-op process variants. |

Each generated method below is executed twice by `TestFrozenCatalogRequests`:
first dry-run with zero business dispatch, then a real CLI request to the fixture.
The fixture verifies its exact method set against the frozen registration archive.
The checked-in `catalog_cases.json` specifies its complete top-level body field
set, nested object/array examples, all path/query flags, HTTP method and route.
The fixture checks request bodies, query values, backend result preservation and
exactly one dispatch. This is transport/type preservation evidence; it cannot
establish live backend validity for every possible option combination.

| Generated command | Request body fields |
| --- | --- |
| `agent register_agent` | Read/path/query operation; no body |
| `agent update_agent_profile` | `profile_content` |
| `agent_task_step_info append_task_steps` | `idempotent_key`, `task_guid`, `task_steps` |
| `custom_field_options create` | `color_index`, `insert_after`, `insert_before`, `is_hidden`, `name` |
| `custom_field_options patch` | `option`, `update_fields` |
| `custom_fields add` | `resource_id`, `resource_type` |
| `custom_fields create` | `datetime_setting`, `member_setting`, `multi_select_setting`, `name`, `number_setting`, `resource_id`, `resource_type`, `single_select_setting`, `text_setting`, `type` |
| `custom_fields get` | Read/path/query operation; no body |
| `custom_fields list` | Read/path/query operation; no body |
| `custom_fields patch` | `custom_field`, `update_fields` |
| `custom_fields remove` | `resource_id`, `resource_type` |
| `members add` | `client_token`, `members` |
| `members remove` | `members` |
| `sections create` | `insert_after`, `insert_before`, `name`, `resource_id`, `resource_type` |
| `sections delete` | Read/path/query operation; no body |
| `sections get` | Read/path/query operation; no body |
| `sections list` | Read/path/query operation; no body |
| `sections patch` | `section`, `update_fields` |
| `sections tasks` | Read/path/query operation; no body |
| `subtasks create` | `agent_task_progress`, `agent_task_status`, `client_token`, `completed_at`, `custom_complete`, `custom_fields`, `description`, `docx_source`, `due`, `extra`, `is_milestone`, `members`, `mode`, `origin`, `reminders`, `repeat_rule`, `start`, `summary`, `tasklists`, `text_deliveries` |
| `subtasks list` | Read/path/query operation; no body |
| `tasklists add_members` | `members` |
| `tasklists create` | `archive_tasklist`, `members`, `name` |
| `tasklists delete` | Read/path/query operation; no body |
| `tasklists get` | Read/path/query operation; no body |
| `tasklists list` | Read/path/query operation; no body |
| `tasklists patch` | `origin_owner_to_role`, `tasklist`, `update_fields` |
| `tasklists remove_members` | `members` |
| `tasklists tasks` | Read/path/query operation; no body |
| `tasks create` | `agent_task_progress`, `agent_task_status`, `client_token`, `completed_at`, `custom_complete`, `custom_fields`, `description`, `docx_source`, `due`, `extra`, `is_milestone`, `members`, `mode`, `origin`, `positive_reminders`, `reminders`, `repeat_rule`, `start`, `summary`, `tasklists`, `text_deliveries` |
| `tasks delete` | Read/path/query operation; no body |
| `tasks get` | Read/path/query operation; no body |
| `tasks list` | Read/path/query operation; no body |
| `tasks patch` | `task`, `update_fields` |

Supported nested data includes task start/due all-day timestamps, origins and
platform hrefs, custom completion links, user/app members and roles, tasklists
and sections, reminders and positive reminders, recurrence, completion and
countersign mode, milestones, custom field values, document sources, task agent
status/progress, text deliveries and idempotency keys. Custom fields retain
member, datetime, number, single-select, multi-select and text settings/value
representations; options retain position, hidden state and color settings.
Numbers encoded as strings stay strings; create/update/list-child generic JSON
numbers now remain decimal JSON numbers without float64 rounding. Backend field
validation and type-specific workflows remain live-dependent.

## Intentional changes and effects

- Create/update reject null, non-object and trailing JSON before writes. List
  creation validates every task object and summary before creating the list;
  `[null]` previously created a list and silently recovered a child panic.
  App assignee shorthand now retains app member type in list-created tasks.
- Explicit `--description=` clears description; omission leaves it unchanged.
  Empty summaries are rejected. Raw JSON null/empty values remain distinct and
  are forwarded for backend interpretation. Update field ordering is stable.
- Update compares supplied replacement fields against a current read and skips
  exact unchanged fields only when every supplied field is supported/present.
  Supported comparisons are summary, description, due, start, extra, repeat_rule,
  completed_at, mode, is_milestone and custom_complete. Missing data never proves
  equality. Members, custom fields and arbitrary other fields retain explicit
  mutation behavior. Changed replacements retain one complete PATCH per task.
- Reopen skips when current completed_at is exactly `"0"`; complete already
  skipped completed tasks. Reminder replacement skips one identical reminder,
  validates nonnegative duration/overflow locally, and otherwise removes then adds.
- Update/reopen add GET previews and reads. The existing shortcut scope is
  `task:task:write`; the frozen GET method accepts that scope as well as read.
  Generated `tasks patch` preserves direct `task:task:writeonly` capability.
  Reads are comparisons, not conditional writes or protection against races.
- Assign/follow, comments, uploads, list placement and ancestor changes remain
  explicit effects. Adding or removing members can notify or change access;
  ID-based membership set retains existing roles. Empty/omitted tasklist set is
  a read; explicit removals are the supported clear route. Omitted ancestor is
  deliberately a detach, matching the reference.
- Shared receipts preserve successful first requests when later requests fail.
  Process tests require partial exit 21 and unknown exit 22, no replay, and
  distinct receipt diagnostics. List child and multi-task failure details retain
  their reference typed output tests. No rollback or complete workflow success
  is inferred from an individual request acknowledgement.

## Verification and open acceptance

Canonical build: `make build BINARY=.tmp/task-cli`.
Whole section: `python3 scripts/next_test.py --section task --suite all
--binary .tmp/task-cli --report .tmp/task-summary-provenance.json`.
The full offline run passes 228 unit/event test events, 14 deterministic legacy
E2E events and 77 process test events with zero failures/skips. These counts
include subtests, not distinct capabilities. `internal/affordance`, `cmd/service`
and `internal/schema` tests pass in the same isolated environment. Reports are
local `.tmp` artifacts and contain synthetic state only. Focused `go vet` of
`shortcuts/task` and `tests/next_sections/task` passes. The task read-artifact
process test retains full custom-field decimal strings, explicit null and
verified synthetic account binding outside the projected stdout result.

Eight legacy live workflows are deliberately excluded by the isolated runner:
comment, task ID handling, reminders, completion/reopen, update, tasklist placement,
and tasklist lifecycle under bot/user. They remain written in `tests/cli_e2e/task/`
but are not authorized live fixtures; they were not run with personal credentials.

Required live batch scenario, pending verified fixture setup: bind current user
and tenant; establish account-private task/list creation visibility before the
first write; create run-owned task/list; exercise replacement/no-op/clear/status,
subtask and section placement, custom fields/options, comment, reminder,
attachment, pagination, then clean up only run-owned objects. Assignments,
followers, tasklist editors and any indirect notification require unambiguously
resolved Ying Wu as the only possible other recipient. Agent registration and
agent-step/profile capabilities need separately verified bot/tenant visibility.
The explicit `python3 scripts/next_test.py --section task --suite live
--allow-live --report .tmp/task-live.json` attempt returned BLOCKED (exit 2).
No private creation policy or tenant-bound Task fixture was verified in this
section, so these live cases are BLOCKED, not passed. A later runner must not
substitute an existing shared list or create broadly then restrict afterward.

The coordinator owns balanced development comparisons and held-out evaluation.
No comparison or token saving percentage is claimed here. Fewer repeated writes
are verified mechanically; discovery/token/capability tradeoffs remain awaiting
agent evaluation. P2 stays open until live and comparison evidence is accepted.

Implementation commit: `552c531`. Shared task collection projection was fixed by
Foundation in `44362bc`; the task process regression requires GUID, confirmed
fields and unchanged status to survive `--result-fields summary`.

P3 summary-input review: create/update now attribute an invalid summary supplied
inside `--data` to `error.param:"--data"`; an explicit `--summary` override owns
its own validation error. The full section rerun above includes null, empty and
numeric data summaries, invalid empty overrides, and valid flag overrides of
invalid data summaries through both dry-run and execution. Invalid inputs never
dispatch a business request; valid overrides preserve the requested summary.
