---
name: lark-apps
description: Develop and operate Spark/Miaoda apps, including local or cloud development, creative HTML, deployment, databases, app storage, integrations, observability, environment variables, collaborators, roles and automation. Use for Miaoda/Spark apps and app runtime domains; not ordinary Drive uploads, document editing or native slides.
---

# Apps

Use `lark-cli apps` with `--as user`: Miaoda apps are user assets. Reuse working authentication. Only after an explicit authentication or missing Apps scope error, run `lark-cli auth login --domain apps`, then resume the original operation. Reads generally require `spark:app:read`, writes `spark:app:write`; some previews require write scope. Respect command-specific scopes and confirmation gates. Explain `error.hint`; it is recovery advice, not authorization for another operation.

## Resolve the app

Use a supplied `app_...` ID or extract it from its app URL. Otherwise read `.spark/meta.json` in an initialized project, or use `+list --keyword "<name>"`; resolve multiple candidates with the user. Never create another app merely because an existing target is unclear. A `cli_...` authentication application ID is not a Miaoda app ID.

A tenant `/page/<meta_token>` link identifies a creative HTML app, not an ordinary document. Resolve it using `+get --app-id <meta_token>`; the same command accepts a bare meta token. App details expose `app_id` and `meta_token`. Use `+get` for details and `+update` for name/description. Omitted update fields are not a request to clear them.

## Choose development and publication

For a new app, decide app type and who writes the code separately. Server persistence/database needs imply `full_stack`; static presentations, decks, mockups or visual material imply `html`; JavaScript interaction without a database implies `frontend`. Prefer frontend when persistence is not requested; clarify an ambiguous persistence requirement. Ask whether the user wants local code development or cloud AI generation if they have not expressed a preference. Do not start initialization, cloning or a cloud session before that choice. An existing `.spark/meta.json` project continues locally; an existing app otherwise follows explicit cloud preference, defaulting to local when clear.

Local flow: `+init` or `+git-credential-init` plus native Git, then implement and validate in the repository. Read matching project `.agents/skills/` guides, including `plugin-guide` after initialization when present. Wire pages/controllers/modules into actual routing/bootstrap and run existing typecheck/build. Commit and push the intended code before `+release-create`; deployment does not include unpushed edits. For any clone/pull/push failure, refresh `+git-credential-init` and retry the original operation. If refresh fails, report both errors and environment limitations; do not switch to HTML upload as a workaround.

Cloud flow: `+session-create` → `+chat` → `+session-get`. Use `+session-list` to locate a session rather than guessing. Respect `next_poll_after_ms`; otherwise incremental work polls about every 5–10 seconds and first initialization every 60–120 seconds. First generation may take 20–50 minutes. Streaming/running/building is unfinished. Use `+session-stop` when stopping is requested. For turn output, read `latest_turn.turn_id` from `+session-get`, then `+session-messages-list --turn-id <id>` with cursors as needed.

New HTML/creative work follows the local Git/release flow. `+html-publish` is only the legacy path for existing HTML apps without Git, never a fallback for new creative work or failed Git. Local preview alone does not authorize app creation or deployment. For legacy publication, inspect sizes before proceeding: each HTML file ≤10 MB, compressed archive ≤20 MB, uncompressed candidates ≤200 MB; stop and report any excess. Require an `index.html` entry and a clean output directory. Credential files are blocked; use `--allow-sensitive` only for explicitly requested credential examples/tutorial content after naming the included candidates.

A request for online/shareable access requires publication. Poll the current `release_id` with `+release-get`: about every 20 seconds, up to approximately five minutes. Report an unfinished status after timeout. Only `finished` establishes deployment; return its actual `online_url`. On `failed`, use returned `error_logs`, never invent a cause or URL. A completed cloud session or historical `is_published=true` does not establish that the latest changes are online. Online access is initially creator-only; explain that and adjust visibility only within authorization. Creative HTML has one development/publication URL; other app types have separate editing and online links. Frontend-to-full-stack upgrades happen through the cloud app editor, not a CLI type-upgrade command.

## Operational routing

- Database structure/data: `+db-table-list/get`, `+db-execute`, `+db-data-export/import`; environments `+db-env-create/diff/migrate`; recovery `+db-recovery-diff/apply`; history `+db-changelog-list`, `+db-audit-status/enable/disable/list`; usage `+db-quota-get`.
- Base synchronization: `+db-sync-create/get/list/update/enable/disable/delete`. Each task maps one Base table to one target table; explain and plan separate tasks for multiple tables.
- App storage: `+file-upload/download/list/get/sign/delete/quota-get`. Ordinary cloud-drive files belong to Drive, table data to database commands.
- Environment: `+env-list/set/delete`; `+env-pull` repairs local startup variables. Use complex values through supported file/stdin input. Summaries identify app/environment/key without repeating secrets.
- Observability: `+log-list/get`, `+trace-list/get`, `+metric-list`, `+analytics-list`. Resolve the app first; do not search workspace files for platform monitoring. Requests metrics without a series selector include total/error; latency includes p50/p99. Add `--api` for an actual named endpoint, never invent grouping options. PV/UV/activity use analytics.
- Cache: `+cache-get/delete/clear`. A single delete with `deleted_key_count=0` means absent/expired, not a deletion. To prove an effect, observe presence before, a count of one, then absence.
- Integrations: `+plugin-install/list/uninstall`; OpenAPI credentials: `+openapi-key-list/get/create/update/enable/disable/delete/reset`. Create/reset secrets are returned once; do not persist or echo them in summaries.
- ID conversion: `+user-id-convert` with explicit direction and 1–100 IDs. Preserve order and duplicates; inspect missed positions rather than guessing conversions. Directions are `miaoda-to-open-id`, `miaoda-to-union-id`, `open-id-to-miaoda`, `union-id-to-miaoda`, `miaoda-to-feishu-user-id`. Scope is `spark:directory.user.id_convert:read`; conversion does not grant resource access.

## Data and source boundaries

Local file/path/output parameters use paths within the working directory; use relative paths or change directory. Remote storage paths remain platform identifiers. Images/fonts/audio/video are app resources: upload through `+file-upload`, use returned remote URLs, and upload separately for each app. Do not commit these assets, embed base64, or reuse another app's storage URL. Uploads are limited to 100 MB. Signed links expire: default one day, maximum 30 days. File deletion reports individual outcomes; inspect each result.

Use `--environment dev|online`, never the removed `--env`. Most database/cache commands auto-select dev for multi-environment apps and online otherwise; do not invent a dev branch for a single-environment app. Sync defaults to online: explicitly select dev when creating tables in a multi-environment app. Online DDL restrictions are intentional, not something to bypass.

SQL runs through `+db-execute`, not a connection string extracted from environment variables. `--sql` and `--file` are alternatives; `--sql -` accepts stdin. Execution requires `--yes`, including SELECT; authorized read-only/reversible operations may use it. Results have a 1,000-row ceiling; use bounds, aggregation or pagination. No automatic transaction wraps multiple statements: earlier statements may commit before a later failure. Do not replay an entire partial batch. Use explicit transaction boundaries when atomicity is needed and evaluate the actual failure evidence.

Follow platform PostgreSQL conventions: business tables include four audit columns, RLS and four policies in the same creation operation; personnel fields use `user_profile`; JSONB fields need their documented type comment. Do not alter platform auth/users resources or create databases/schemas/users/roles to bypass restrictions. UPDATE requires a precise WHERE and must not become DELETE+INSERT. Count unclear affected rows first. Destructive changes need explicit informed authorization. For existing rows, add nullable fields, backfill, then apply NOT NULL rather than deploying an unsafe required field.

Native export format follows `.csv/.json/.sql` output extension, not global rendering format; it does not export to stdout. Respect the 1 MB and 5,000-row transfer limits; do not assume a bounded export covers the whole source. Import uses CSV/JSON and a deliberate target table. Recovery overwrites current data: preview with `+db-recovery-diff` before applying. Environment splitting/migration is consequential and requires preview/confirmation.

Sync preview is recommended before creation, but do not force it when direct creation is explicitly authorized and server automatic mapping is intended. Use plural `field_maps`/`option_mappings`; explicit mappings need an enabled field. Get current configuration before updating. Batch/import tasks are one-shot and cannot be re-enabled; inspect status/result and create a streaming task for ongoing synchronization.

Application runtime code uses the project's documented SDKs, never CLI subprocesses. Read actual exported types and documented input/output paths; do not fabricate ambient types or assertions. Check each action's complete prerequisites and SDK contract even if compilation succeeds. Current platform members/IDs are facts, not implicit authorization policy to hardcode.

## Permissions and automation

Keep three concepts separate: `+access-scope-get/set` controls runtime visibility; `+member-*` and `+member-settings-*` control development collaboration; `+role-list/get/create/update/delete`, `+role-member-list/add/remove`, `+role-match-list` manage app business roles. Source-level authorization follows the project's authz guide. Creative HTML comments use Drive comments.

Collaborator reads require app-read scope. Writes require app-write scope, precise target preview and authorized `--yes`. Write IDs are explicit `openid`/`openchat`/`opendepartmentid`, never inferred numeric IDs. List filters instead use `user`/`department`/`chat` and `view`/`edit`/`full_access`; direct collaborators are not paginated. Fetch app details separately. Do not preclude collaboration by app type. If `feature_not_available` reports unsupported collaboration, stop and direct the user to backend permissions; do not substitute roles/access scope. `external_invite` follows `external_access`; `copy_download_by` is read-only. Neither is an independent writable setting.

Automation uses `+automation-list/get/create/update/enable/disable` for cron, record-change, webhook and Feishu approval triggers. Show the proposed trigger configuration before requesting missing details. Resolve its unique name; never try several candidate names with writes. Create defaults to disabled. Runtime handlers bind the exact trigger name and execute published app code; configuration or release history alone does not prove handler execution.

Separate configuring, publishing, enabling and probing. An enable-only request does not authorize code changes or fabricated events. Check trigger state and a finished release, enable as requested, then read back. Runtime tests require authorization for the concrete event, payload, effects, observations and cleanup, plus any temporary enable. Preserve an initially enabled state; restore an initially disabled trigger even after failure or interruption. Publishing a handler publishes the whole app. Do not publish or start it merely because configuration was requested; uncertain publication must not be treated as success.

Webhook URL/token actions are irreversible and one-time outputs, not disk artifacts. Before authorized update/rotation, identify the exact trigger and affected environment; URL reset requires an explicit preview/runtime choice. Explain old credentials becoming invalid, one-time output and irreversibility before confirmation. Keep token validation or an IP allowlist. If both are requested off, explain unauthenticated public triggering, offer retaining either protection, and require explicit informed confirmation for disabling both.

## Authorization boundaries

Honor existing concrete authorization or an explicit end-to-end instruction; do not repeatedly request permission for already authorized workflow steps. Generic delegation does not authorize irreversible data loss, role deletion, removing all role members, bulk collaborator removal, or clearing an entire cache environment. Preview the exact impact and obtain informed confirmation for those actions; an exit-10 confirmation gate is not permission to append `--yes`. Environment deletion also requires confirmed app/environment/key and consequences. Never infer an environment for a cache clear. Authentication recovery does not expand authorization. Stop at HTML size limits rather than silently changing the publication strategy.
