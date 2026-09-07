---
name: lark-task
version: 1.0.0
description: "Manage Feishu tasks, tasklists, assignments, reminders, subtasks, attachments and task agents. Use for the Task center and project to-dos; use Meetings for to-dos inside meeting minutes."
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli task --help"
---

# Tasks

Read [shared authentication, identity, permission and output rules](../lark-shared/SKILL.md) before starting. Before invoking a command, read its reference or help. Use only shortcut tokens listed below or returned by `lark-cli task --help`; do not invent `+<verb>` commands. When no listed shortcut matches, discover the resource and method through help, then inspect `lark-cli schema task.<resource>.<method>` before constructing native `--params`/`--data`. On `unknown_subcommand`, return to help instead of guessing variants.

Task GUIDs are OpenAPI resource identities, not client display numbers such as `t104121` or `suite_entity_num`. Extract `guid` from task/tasklist applinks; `+update` and `+complete` also accept task applinks directly. If a list GUID is known, use it without searching. Search a list once only when resolving a name or keyword, then inspect `schema task.tasklists.tasks`, list that list's tasks, and use each returned task GUID.

## Routing and queries

To-dos inside a minute, recording, `/minutes/` URL or minute_token belong to Meetings' `minutes +todo`; do not create or search Task-center lists for those requests. This domain covers Task-center tasks, tasklists, assignments, project to-dos, due dates, reminders and subtasks.

For a real keyword or an explicit request to use the search skill, prefer `+search`; otherwise prefer list operations when scope filters alone suffice. “Search tasks I follow since this year” contains a relationship and time range, not necessarily a keyword: choose `+get-related-tasks`. Tasks assigned to self use `+get-my-tasks`. Its `--query` returns exact summary matches when available, otherwise partial matches. For tasklists with only creator/time conditions, inspect `task.tasklists.list`, list them with user identity, then filter creator/created_at locally. Do not turn a time phrase into a query keyword.

`+search` does not infer “related to me.” Resolve the current user's open_id and explicitly use `--assignee`, `--creator` or `--follower` for the intended relationship. In user-identity requests, “me” refers to the logged-in user. `+get-my-tasks`, `+get-related-tasks` and `+search` require user identity; app identity is not a substitute.

List/search shortcuts paginate up to 20 pages by default; `--page-limit` changes the bound, `--page-all` permits at most 40, and `--page-token` resumes. Retain `has_more` and continuation when results are incomplete. For `+get-related-tasks`, the token is an updated_at cursor in **microseconds**. For a time-bounded request, convert the start boundary to that cursor and page forward, stopping at 40 pages. Do not start with `--page-all` unless the user requests a full scan or no time boundary can be inferred. Its creator/follower filters are local; continuation still describes upstream pages and can lead to pages without matching tasks.

`+get-my-tasks` omits completion filtering by default: pass `--complete=false` for pending/daily-summary views and `--complete=true` for completed tasks. `+get-related-tasks --include-complete=false` keeps incomplete tasks. Search uses `--completed=false` or `true`. Search without a keyword requires another filter. Search/detail enrichment and tasklist search may make additional detail requests.

## Operations

| Shortcut and reference | Inputs and behavior |
| --- | --- |
| [`+create`](references/lark-task-create.md) | `--summary`, optional description, user/app assignee and follower, due, tasklist GUID/applink, idempotency key, and `--data` task object. |
| [`+update`](references/lark-task-update.md) | Task GUID/applink or comma-separated targets; summary, description, due or task-object `--data`. Report `updated_fields` and each task's server `confirmed` values. |
| [`+complete`](references/lark-task-complete.md) | Task GUID/applink. Read `status`, `completed_at`, `already_completed`; the latter means an already-completed task was observed and PATCH skipped. |
| [`+reopen`](references/lark-task-reopen.md) | Reopen the specified task GUID. |
| [`+assign`](references/lark-task-assign.md) | Task GUID and comma-separated `--add`/`--remove` assignees; people use open_id, apps use `cli_` IDs. Both actions can express a transfer. |
| [`+followers`](references/lark-task-followers.md) | Task GUID and comma-separated user/app follower IDs to add/remove. |
| [`+comment`](references/lark-task-comment.md) | Task GUID and literal `--content`; report the returned comment ID. |
| [`+set-ancestor`](references/lark-task-set-ancestor.md) | Child `--task-id` and optional `--ancestor-id`; omitting the ancestor deliberately clears it. |
| [`+reminder`](references/lark-task-reminder.md) | Preferred reminder interface: task GUID and `--set` minutes, `m`, `h` or `d`, or `--remove=true` for all reminders. Requires a due date. |
| [`+get-my-tasks`](references/lark-task-get-my-tasks.md) | Query, optional completion, `--created_at`, `--due-start`, `--due-end`, and paging. Creation/due filters accept date, relative or millisecond input. |
| [`+get-related-tasks`](references/lark-task-get-related-tasks.md) | Include-complete, created-by-me, followed-by-me and cursor/page controls. |
| [`+search`](references/lark-task-search.md) | Query, comma-separated creator/assignee/follower open_ids, completed, `--due start,end`, and paging. |
| [`+tasklist-create`](references/lark-task-tasklist-create.md) | Name, comma-separated `--member` editor open_ids and optional JSON array of initial task definitions in `--data`. Report batch creation results. |
| [`+tasklist-search`](references/lark-task-tasklist-search.md) | Query, comma-separated creator open_ids, `--create-time start,end`, and paging. |
| [`+tasklist-task-add`](references/lark-task-tasklist-task-add.md) | Tasklist GUID/applink, comma-separated task GUIDs and optional section GUID; omission selects the default section. Report successful and failed tasks. |
| [`+tasklist-members`](references/lark-task-tasklist-members.md) | Tasklist GUID/applink with add/remove or exact `--set` member open_ids. |
| [`+upload-attachment`](references/lark-task-upload-attachment.md) | One local file, at most 50 MB, with resource GUID/applink; resource type defaults to task, task agents use task_delivery, user ID type defaults to open_id. |

For creation, named flags override same-named `--data` fields. If a requested field has no dedicated flag, inspect `schema task.tasks.create` and use its exact type/nesting in `--data`; do not omit the field or duplicate fields already supplied by flags. Prefer the creation shortcut when `--data` expresses the request. Do not assume other shortcuts support that flag. A user saying “create a task for me” or “help me create a task” requires assigning it to the current user: obtain open_id from `auth status` (`identities.user.openId`) or `contact +get-user` (`data.user.open_id`) and pass `--assignee`.

Due inputs accept ISO 8601, dates, relative time and millisecond timestamps. Date/day-style inputs express all-day tasks. Repeat rules and reminders require due; when start and due are both supplied, start must not be after due. App/tenant-token member changes cannot cross tenants. Native APIs support richer fields and resources: tasks, tasklists, subtasks, members, sections, custom_fields, custom_field_options, agent registration/profile updates and agent_task_step_info records; use actual schema rather than inferred methods or JSON.

Scopes in the reference group by resource: task/subtask/member writes and task-agent operations use `task:task:write`; task reads use `task:task:read`. Tasklist reads/writes use `task:tasklist:read`/`write`; section reads/writes use `task:section:read`/`write`; custom-field reads use `task:custom_field:read`, and field/option writes use `task:custom_field:write`. Attachment upload requires `task:attachment:write`. Identity and exact alternative scope requirements come from the command's help/schema.

## Results and safety

Confirm intended task, fields, recipients and write effects under the user's authorization. Preview dangerous requests with `--dry-run`; on exit 10 follow shared explicit-approval handling rather than silently adding a confirmation flag. Never expose credentials. File inputs follow shared relative-path rules: attachments must be regular files within the working directory, one per call; move/split an oversized or out-of-scope file before uploading. Preserve returned attachment metadata and always surface its GUID and available URL.

JSON success uses `ok:true` or exit 0, not a nonexistent top-level `code:0`. Report actual server-confirmed state and useful task/tasklist URLs. If update `confirmed` fields or completion's `status:done` and nonzero `completed_at` already provide the needed confirmation, do not routinely append `tasks get`; query again only when required fields are absent or the user requests full verification. When presenting people, retain IDs and try Contact resolution for real names; render displayed timestamps in local timezone as `YYYY-MM-DD HH:mm:ss`.
