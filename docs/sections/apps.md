# Apps (aPaaS) section

Status: implementation reused and offline section boundary validated. Live and comparative acceptance remain outstanding.

Frozen reference: `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`. The unchanged sibling source and frozen registration inventory contain 96 shortcuts plus the hidden Git credential helper. This section is separate from `application`. [Complete mapping](../../evidence/sections/apps-mapping.json) records every command, a pointer to full frozen flag definitions/defaults/aliases/requiredness, identity/risk, source SHA-256, and nearby unit owners.

The existing implementation remains the execution surface: 1,542 unit/subtest events pass the section gate. No commands were replaced with raw API forwarding. The request fixture was captured from a separately built frozen sibling CLI in the isolated environment, with synthetic credentials, fixed time ranges, synthetic files, and no live API calls. The 92 process dry-run cases plus five local/helper cases map every frozen leaf; these request cases do not alone establish workflow or live parity.

## Scope and preserved contracts

- App create/get/list/metadata patch; HTML, frontend and full_stack policies; meta-token addressing, ownership filtering and manual cursors. Access ranges retain user/department/chat targets, approval configuration and login semantics. Roles, role members, matching, collaborators and settings retain typed external identifiers, additive/removal/clear distinctions and notification flags.
- HTML packaging retains hidden-file handling, sensitive-file gates, manifest/index checks and pre-release/upload/release sequence. Initialization retains git/scaffold/environment/skill phases and app-type policies. Release create/list/get, sessions/create/list/get/stop/messages and chat keep asynchronous effects and polling guidance.
- SQL retains SELECT, DML, DDL, multi-statement and legacy result forms, explicit transactional=false, dev/online semantics, SQL file/stdin, row import multipart CSV/JSON (1 MB ceiling), export CSV/JSON and limit, full table columns/indexes/constraints/defaults. Sync preserves batch/streaming, Base source URL/table addressing, target create/use_existing, enabled field mappings, schema_only, preview/exported config and status operations. Environment split, diff/migrate, recovery diff/apply, audit/changelog/quota remain available.
- Files retain upload, metadata/list/filter, signed URLs, download, delete and quotas. Cache read/delete/clear, environment metadata/value opt-in, protected variable handling, environment pull merge and credential-local permissions remain available.
- Observability retains log/trace list/get filters, metrics requests/latency/CPU/memory and series, analytics users/page-view, nanosecond/second distinctions, ranges and pagination. Plugins retain registry/local archive installs, version and already-installed paths, batch installation, dependency guards and package.json preservation.
- OpenAPI keys retain create/get/list/update/enable/disable/delete/reset, scope combinations and one-time secret output/redaction. Git retains scoped repository helper, keychain metadata and local cleanup. Automation retains cron, record-change (INSERT/UPDATE/UPSERT/DELETE and field selectors), webhook/IP lists, Feishu approval instance/task filters, create/update/status, pagination and four URL/token lifecycle actions. User ID conversion retains all five directions, input order/duplicates and missed positions.

## Intentional corrections

- Agent JSON app lists preserve full app metadata, including icon_url and created_at; human pretty tables remain curated.
- Metadata replacement defaults to the original write-only PATCH. The added --skip-unchanged flag reads the current app and skips unchanged requested fields; omitted fields never imply clearing. Missing/malformed or mismatched targets or failed reads prevent the patch. Only the opt-in path needs the conditional spark:app:read scope and exposes GET plus conditional PATCH in dry-run; it is an observed comparison, with no backend concurrency guarantee. Callers that omit the new flag retain their original patch flow, including users without app-read scope.
- Automation aggregation refuses has_more=true without a usable continuation instead of claiming complete results. Repeated cursors and the existing page cap remain guarded.
- SQL SELECT decoding preserves numeric decimal text, large integers, leading-zero strings, null, booleans and nested values. Malformed row arrays/statement data fail instead of becoming a successful empty result. Legacy empty DDL results remain supported.
- Environment values requested with --include-values cannot be saved as generic artifacts: arbitrary key/value records can contain credentials that a field-name guard cannot detect. Default redacted environment reads retain artifact support.
- File deletion keeps missing per-file outcomes unknown, with explicit inspection guidance; it no longer synthesizes successful deletion.
- Database exports use the existing streaming transport to preserve native CSV/JSON/SQL bytes, count serialized CSV/JSON records (including quoted multiline CSV), and expose limit, source_total when observed, rows_counted, has_more when known and complete. SQL dumps remain estimated and never claim completeness from a guessed count. Native import refuses generic result-artifact envelopes; byte-preserving native export/import remains supported.
- Both file storage and HTML archive PUTs use the invocation external transport with precise storage-effect markers. Only final HTTP 200/201/204 storage acknowledgements permit metadata registration or release creation. Upload clients do not follow redirects, preventing PUT-to-GET changes or replay; downloads retain redirects. HTTP 202, unresolved 3xx and lost responses stop the workflow with durable unknown outcomes without replay. Gateway authorization and signed URLs are absent from persistent receipts.
- Domain-known POST previews and plugin metadata/download reads are explicitly excluded from mutation journaling. Actual schema/sync/content writes retain conservative receipts. Partial/malformed SQL outcomes use the shared SQL receipt classifier and pass the process regression.

## Executable evidence

`python3 scripts/next_test.py --section apps --suite all --report .tmp/apps-boundary-final.json` runs every Apps unit, inherited deterministic DryRun/Regression E2E, and the new process package using the production authsidecar build. The runner isolates config/home/credentials and blocks external proxy traffic. Process fixtures never forward requests.

All 91 API-backed shortcut cases also execute their initial dispatch against a controlled denial, asserting the real method/path and no automatic retry. The remaining local/helper leaves are covered by process-local scenarios; initialization is mapped by dry-run and its existing unit workflows.

New process scenarios assert actual requests, independently observed state, no-op write counts, malformed/ambiguous/identity failures, lost-response non-replay, SQL precision, automation four-family lifecycle and rotations, missing/repeated/failed pagination, artifact completeness/redaction, credential artifact refusal before issuance, plugin archive install/list/idempotent version/uninstall preservation, local Git absence/cleanup/helper behavior and preview config handoff. Nearby units retain the extensive uncommon-option/type and upload/scaffold fault paths. File upload and HTML publication each cover storage success, lost PUT, and downstream callback/release failure with distinct observed effect counts. [Machine-readable evidence](../../evidence/sections/apps-evidence.json) records commands, counts, skips and limits.

The complete inherited Apps E2E package is also executed without the default name selector: 144 passing events and four explicitly skipped live workflows. Affordance, service and schema tests and scoped go vet pass. Narrow command guidance is in [apps affordance](../../affordance/apps.md).

All 97 command mappings:

| Command | Frozen implementation | Process request case |
| --- | --- | --- |
| `apps +access-scope-get` | `shortcuts/apps/apps_access_scope_get.go` | `+access-scope-get` |
| `apps +access-scope-set` | `shortcuts/apps/apps_access_scope_set.go` | `+access-scope-set` |
| `apps +analytics-list` | `shortcuts/apps/apps_analytics.go` | `+analytics-list` |
| `apps +automation-create` | `shortcuts/apps/apps_automation_create.go` | `+automation-create` |
| `apps +automation-disable` | `shortcuts/apps/apps_automation_disable.go` | `+automation-disable` |
| `apps +automation-enable` | `shortcuts/apps/apps_automation_enable.go` | `+automation-enable` |
| `apps +automation-get` | `shortcuts/apps/apps_automation_get.go` | `+automation-get` |
| `apps +automation-list` | `shortcuts/apps/apps_automation_list.go` | `+automation-list` |
| `apps +automation-update` | `shortcuts/apps/apps_automation_update.go` | `+automation-update` |
| `apps +cache-clear` | `shortcuts/apps/apps_cache_clear.go` | `+cache-clear` |
| `apps +cache-delete` | `shortcuts/apps/apps_cache_delete.go` | `+cache-delete` |
| `apps +cache-get` | `shortcuts/apps/apps_cache_get.go` | `+cache-get` |
| `apps +chat` | `shortcuts/apps/apps_chat.go` | `+chat` |
| `apps +create` | `shortcuts/apps/apps_create.go` | `+create` |
| `apps +db-audit-disable` | `shortcuts/apps/apps_db_audit_set.go` | `+db-audit-disable` |
| `apps +db-audit-enable` | `shortcuts/apps/apps_db_audit_set.go` | `+db-audit-enable` |
| `apps +db-audit-list` | `shortcuts/apps/apps_db_audit_list.go` | `+db-audit-list` |
| `apps +db-audit-status` | `shortcuts/apps/apps_db_audit_status.go` | `+db-audit-status` |
| `apps +db-changelog-list` | `shortcuts/apps/apps_db_changelog_list.go` | `+db-changelog-list` |
| `apps +db-data-export` | `shortcuts/apps/apps_db_data_export.go` | `+db-data-export` |
| `apps +db-data-import` | `shortcuts/apps/apps_db_data_import.go` | `+db-data-import` |
| `apps +db-env-create` | `shortcuts/apps/apps_db_env_create.go` | `+db-env-create` |
| `apps +db-env-diff` | `shortcuts/apps/apps_db_env_migrate.go` | `+db-env-diff` |
| `apps +db-env-migrate` | `shortcuts/apps/apps_db_env_migrate.go` | `+db-env-migrate` |
| `apps +db-execute` | `shortcuts/apps/apps_db_execute.go` | `+db-execute` |
| `apps +db-quota-get` | `shortcuts/apps/apps_db_quota_get.go` | `+db-quota-get` |
| `apps +db-recovery-apply` | `shortcuts/apps/apps_db_recovery.go` | `+db-recovery-apply` |
| `apps +db-recovery-diff` | `shortcuts/apps/apps_db_recovery.go` | `+db-recovery-diff` |
| `apps +db-sync-create` | `shortcuts/apps/apps_db_sync_create.go` | `+db-sync-create` |
| `apps +db-sync-delete` | `shortcuts/apps/apps_db_sync_operate.go` | `+db-sync-delete` |
| `apps +db-sync-disable` | `shortcuts/apps/apps_db_sync_operate.go` | `+db-sync-disable` |
| `apps +db-sync-enable` | `shortcuts/apps/apps_db_sync_operate.go` | `+db-sync-enable` |
| `apps +db-sync-get` | `shortcuts/apps/apps_db_sync_get.go` | `+db-sync-get` |
| `apps +db-sync-list` | `shortcuts/apps/apps_db_sync_list.go` | `+db-sync-list` |
| `apps +db-sync-update` | `shortcuts/apps/apps_db_sync_update.go` | `+db-sync-update` |
| `apps +db-table-get` | `shortcuts/apps/apps_db_table_get.go` | `+db-table-get` |
| `apps +db-table-list` | `shortcuts/apps/apps_db_table_list.go` | `+db-table-list` |
| `apps +env-delete` | `shortcuts/apps/apps_env.go` | `+env-delete` |
| `apps +env-list` | `shortcuts/apps/apps_env.go` | `+env-list` |
| `apps +env-pull` | `shortcuts/apps/apps_env_pull.go` | `+env-pull` |
| `apps +env-set` | `shortcuts/apps/apps_env.go` | `+env-set` |
| `apps +file-delete` | `shortcuts/apps/apps_file_delete.go` | `+file-delete` |
| `apps +file-download` | `shortcuts/apps/apps_file_download.go` | `+file-download` |
| `apps +file-get` | `shortcuts/apps/apps_file_get.go` | `+file-get` |
| `apps +file-list` | `shortcuts/apps/apps_file_list.go` | `+file-list` |
| `apps +file-quota-get` | `shortcuts/apps/apps_file_quota_get.go` | `+file-quota-get` |
| `apps +file-sign` | `shortcuts/apps/apps_file_sign.go` | `+file-sign` |
| `apps +file-upload` | `shortcuts/apps/apps_file_upload.go` | `+file-upload` |
| `apps +get` | `shortcuts/apps/apps_get.go` | `+get` |
| `apps +git-credential-init` | `shortcuts/apps/git_credential.go` | `+git-credential-init` |
| `apps +git-credential-list` | `shortcuts/apps/git_credential.go` | `TestAppsLocalPluginLifecycleAndCredentialIsolation` |
| `apps +git-credential-remove` | `shortcuts/apps/git_credential.go` | `TestAppsLocalPluginLifecycleAndCredentialIsolation` |
| `apps +html-publish` | `shortcuts/apps/apps_html_publish.go` | `+html-publish` |
| `apps +init` | `shortcuts/apps/apps_init.go` | `+init` |
| `apps +list` | `shortcuts/apps/apps_list.go` | `+list` |
| `apps +log-get` | `shortcuts/apps/apps_logs.go` | `+log-get` |
| `apps +log-list` | `shortcuts/apps/apps_logs.go` | `+log-list` |
| `apps +member-add` | `shortcuts/apps/apps_member.go` | `+member-add` |
| `apps +member-list` | `shortcuts/apps/apps_member.go` | `+member-list` |
| `apps +member-remove` | `shortcuts/apps/apps_member.go` | `+member-remove` |
| `apps +member-settings-get` | `shortcuts/apps/apps_member.go` | `+member-settings-get` |
| `apps +member-settings-set` | `shortcuts/apps/apps_member.go` | `+member-settings-set` |
| `apps +member-update` | `shortcuts/apps/apps_member.go` | `+member-update` |
| `apps +metric-list` | `shortcuts/apps/apps_metrics.go` | `+metric-list` |
| `apps +openapi-key-create` | `shortcuts/apps/apps_openapi_key_create.go` | `+openapi-key-create` |
| `apps +openapi-key-delete` | `shortcuts/apps/apps_openapi_key_delete.go` | `+openapi-key-delete` |
| `apps +openapi-key-disable` | `shortcuts/apps/apps_openapi_key_disable.go` | `+openapi-key-disable` |
| `apps +openapi-key-enable` | `shortcuts/apps/apps_openapi_key_enable.go` | `+openapi-key-enable` |
| `apps +openapi-key-get` | `shortcuts/apps/apps_openapi_key_get.go` | `+openapi-key-get` |
| `apps +openapi-key-list` | `shortcuts/apps/apps_openapi_key_list.go` | `+openapi-key-list` |
| `apps +openapi-key-reset` | `shortcuts/apps/apps_openapi_key_reset.go` | `+openapi-key-reset` |
| `apps +openapi-key-update` | `shortcuts/apps/apps_openapi_key_update.go` | `+openapi-key-update` |
| `apps +plugin-install` | `shortcuts/apps/apps_plugin_install.go` | `+plugin-install` |
| `apps +plugin-list` | `shortcuts/apps/apps_plugin_list.go` | `TestAppsLocalPluginLifecycleAndCredentialIsolation` |
| `apps +plugin-uninstall` | `shortcuts/apps/apps_plugin_uninstall.go` | `TestAppsLocalPluginLifecycleAndCredentialIsolation` |
| `apps +release-create` | `shortcuts/apps/apps_release_create.go` | `+release-create` |
| `apps +release-get` | `shortcuts/apps/apps_release_get.go` | `+release-get` |
| `apps +release-list` | `shortcuts/apps/apps_release_list.go` | `+release-list` |
| `apps +role-create` | `shortcuts/apps/apps_role.go` | `+role-create` |
| `apps +role-delete` | `shortcuts/apps/apps_role.go` | `+role-delete` |
| `apps +role-get` | `shortcuts/apps/apps_role.go` | `+role-get` |
| `apps +role-list` | `shortcuts/apps/apps_role.go` | `+role-list` |
| `apps +role-match-list` | `shortcuts/apps/apps_role_member.go` | `+role-match-list` |
| `apps +role-member-add` | `shortcuts/apps/apps_role_member.go` | `+role-member-add` |
| `apps +role-member-list` | `shortcuts/apps/apps_role_member.go` | `+role-member-list` |
| `apps +role-member-remove` | `shortcuts/apps/apps_role_member.go` | `+role-member-remove` |
| `apps +role-update` | `shortcuts/apps/apps_role.go` | `+role-update` |
| `apps +session-create` | `shortcuts/apps/apps_session_create.go` | `+session-create` |
| `apps +session-get` | `shortcuts/apps/apps_session_get.go` | `+session-get` |
| `apps +session-list` | `shortcuts/apps/apps_session_list.go` | `+session-list` |
| `apps +session-messages-list` | `shortcuts/apps/apps_session_messages_list.go` | `+session-messages-list` |
| `apps +session-stop` | `shortcuts/apps/apps_session_stop.go` | `+session-stop` |
| `apps +trace-get` | `shortcuts/apps/apps_traces.go` | `+trace-get` |
| `apps +trace-list` | `shortcuts/apps/apps_traces.go` | `+trace-list` |
| `apps +update` | `shortcuts/apps/apps_update.go` | `+update` |
| `apps +user-id-convert` | `shortcuts/apps/apps_user_id_convert.go` | `+user-id-convert` |
| `apps git-credential-helper` | `shortcuts/apps/git_credential.go` | `TestAppsLocalPluginLifecycleAndCredentialIsolation` |

## Outstanding evidence

Live is blocked, not passed. `TestAppsLivePrivateLifecycle` is separately selectable and explicitly skipped: no verified fixture currently proves the effective account and tenant, app creation visibility limited to that account, a run-owned app cleanup API, or downstream automation/notification recipients limited to the account and unambiguously resolved Ying Wu. Existing legacy live role/file fixtures are not accepted proof under this run authorization. No live business content, automation or recipient effects were performed.

The inherited default E2E selector excludes LocalE2E, Validation and live-named cases; an additional full-package run covers its deterministic local/validation tests and reports all four live skips. Full process initialization/git push, live presigned storage, backend async completion, exact backend transaction/rollback semantics, authorization/privacy and external app effects remain live-dependent and unverified. The existing synchronous units are substitute evidence only.

Comparative development evaluations are scheduled by the coordinator after the section suite; no held-out tasks were used. Cross-section dependencies are Base source data for sync, shared file/artifact binding, shared SQL mutation evidence and root discovery/guidance assembly.
