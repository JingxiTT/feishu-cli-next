# Base section

Status: deterministic section gate passed; live, cross-section and comparative acceptance remain open.

Reference: unchanged sibling `../feishu-cli-review` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`. The imported implementations are retained: 117 native shortcuts across Base, tables, fields, views, records, templates, roles/permissions, forms, workflows/buttons, dashboards, workspaces and BaseApps. Frozen default Feishu registrations contain all 117 runnable Base shortcuts plus the domain, including upload/download/remove attachment commands. The new typed import adds one command. Generated service machinery remains owned by the foundation; this section adds no raw forwarding substitute.

## Complete command mapping

All reference commands retain names, schemas, flags/defaults/aliases, auth modes, risk and dry-run hooks unless a difference below is explicit. The full flag/type/default metadata remains authoritative in `docs/reference/registrations.json.gz` and runtime schema. This table identifies the implementation owner; `tests/next_sections/base` owns process assertions for every registration, including attachment commands. Existing `tests/cli_e2e/base` owns detailed option dry-runs; nearby `shortcuts/base/*_test.go` cover domain validation and wire projection.

| Command | Implementation | Reference-specific flags (excluding shared output/auth controls) |
| --- | --- | --- |
| `+advperm-disable` | `shortcuts/base/base_advperm_disable.go` | `--base-token` |
| `+advperm-enable` | `shortcuts/base/base_advperm_enable.go` | `--base-token` |
| `+app-block-create` | `shortcuts/base/app_block_create.go` | `--app-token`, `--data-config`, `--name`, `--no-validate`, `--page-id`, `--sub-type`, `--type` |
| `+app-block-get` | `shortcuts/base/app_block_get.go` | `--app-token`, `--block-id`, `--page-id` |
| `+app-block-get-data` | `shortcuts/base/app_block_get_data.go` | `--app-token`, `--base-token`, `--block-id` |
| `+app-block-list` | `shortcuts/base/app_block_list.go` | `--app-token`, `--page-id`, `--page-size`, `--page-token` |
| `+app-block-update` | `shortcuts/base/app_block_update.go` | `--app-token`, `--block-id`, `--data-config`, `--name`, `--no-validate`, `--page-id` |
| `+app-create` | `shortcuts/base/app_create.go` | `--name`, `--theme-style`, `--workspace-token` |
| `+app-get` | `shortcuts/base/app_get.go` | `--app-token` |
| `+app-page-create` | `shortcuts/base/app_page_create.go` | `--app-token`, `--name` |
| `+app-page-delete` | `shortcuts/base/app_page_delete.go` | `--app-token`, `--page-id` |
| `+app-page-get` | `shortcuts/base/app_page_get.go` | `--app-token`, `--page-id` |
| `+app-page-list` | `shortcuts/base/app_page_list.go` | `--app-token`, `--page-size`, `--page-token` |
| `+app-page-update` | `shortcuts/base/app_page_rename.go` | `--app-token`, `--name`, `--page-id` |
| `+base-block-create` | `shortcuts/base/base_block_create.go` | `--base-token`, `--name`, `--parent-id`, `--type` |
| `+base-block-delete` | `shortcuts/base/base_block_delete.go` | `--base-token`, `--block-id` |
| `+base-block-list` | `shortcuts/base/base_block_list.go` | `--base-token`, `--parent-id`, `--type` |
| `+base-block-move` | `shortcuts/base/base_block_move.go` | `--after-id`, `--base-token`, `--before-id`, `--block-id`, `--parent-id` |
| `+base-block-rename` | `shortcuts/base/base_block_rename.go` | `--base-token`, `--block-id`, `--name` |
| `+base-copy` | `shortcuts/base/base_copy.go` | `--base-token`, `--folder-token`, `--name`, `--time-zone`, `--without-content` |
| `+base-create` | `shortcuts/base/base_create.go` | `--fields`, `--folder-token`, `--name`, `--table-name`, `--time-zone` |
| `+base-get` | `shortcuts/base/base_get.go` | `--base-token` |
| `+button-rule-bind` | `shortcuts/base/button_rule.go` | `--base-token`, `--field-id`, `--table-id`, `--workflow-id` |
| `+button-rule-get` | `shortcuts/base/button_rule.go` | `--base-token`, `--field-id`, `--table-id` |
| `+button-rule-unbind` | `shortcuts/base/button_rule.go` | `--base-token`, `--field-id`, `--table-id` |
| `+dashboard-arrange` | `shortcuts/base/dashboard_arrange.go` | `--base-token`, `--dashboard-id`, `--user-id-type` |
| `+dashboard-block-create` | `shortcuts/base/dashboard_block_create.go` | `--base-token`, `--dashboard-id`, `--data-config`, `--name`, `--no-validate`, `--position`, `--type`, `--user-id-type` |
| `+dashboard-block-delete` | `shortcuts/base/dashboard_block_delete.go` | `--base-token`, `--block-id`, `--dashboard-id` |
| `+dashboard-block-get` | `shortcuts/base/dashboard_block_get.go` | `--base-token`, `--block-id`, `--dashboard-id`, `--user-id-type` |
| `+dashboard-block-get-data` | `shortcuts/base/dashboard_block_get_data.go` | `--base-token`, `--block-id`, `--dashboard-id` |
| `+dashboard-block-list` | `shortcuts/base/dashboard_block_list.go` | `--base-token`, `--dashboard-id`, `--page-size`, `--page-token` |
| `+dashboard-block-update` | `shortcuts/base/dashboard_block_update.go` | `--base-token`, `--block-id`, `--dashboard-id`, `--data-config`, `--name`, `--no-validate`, `--position`, `--user-id-type` |
| `+dashboard-create` | `shortcuts/base/dashboard_create.go` | `--base-token`, `--name`, `--theme-style` |
| `+dashboard-delete` | `shortcuts/base/dashboard_delete.go` | `--base-token`, `--dashboard-id` |
| `+dashboard-get` | `shortcuts/base/dashboard_get.go` | `--base-token`, `--dashboard-id` |
| `+dashboard-list` | `shortcuts/base/dashboard_list.go` | `--base-token`, `--page-size`, `--page-token` |
| `+dashboard-share-get` | `shortcuts/base/dashboard_share.go` | `--base-token`, `--dashboard-id` |
| `+dashboard-share-update` | `shortcuts/base/dashboard_share.go` | `--access-scope`, `--base-token`, `--dashboard-id`, `--enabled`, `--show-source` |
| `+dashboard-update` | `shortcuts/base/dashboard_update.go` | `--base-token`, `--dashboard-id`, `--name`, `--theme-style` |
| `+data-query` | `shortcuts/base/base_data_query.go` | `--base-token`, `--dsl` |
| `+field-create` | `shortcuts/base/field_create.go` | `--base-token`, `--i-have-read-guide`, `--json`, `--table-id` |
| `+field-delete` | `shortcuts/base/field_delete.go` | `--base-token`, `--field-id`, `--table-id` |
| `+field-extension-get` | `shortcuts/base/field_extension.go` | `--base-token`, `--field-id`, `--table-id` |
| `+field-extension-update` | `shortcuts/base/field_extension.go` | `--base-token`, `--field-id`, `--json`, `--table-id` |
| `+field-extension-update-cells` | `shortcuts/base/field_extension.go` | `--base-token`, `--field-id`, `--record-id`, `--table-id`, `--type`, `--view-id` |
| `+field-get` | `shortcuts/base/field_get.go` | `--base-token`, `--field-id`, `--table-id` |
| `+field-list` | `shortcuts/base/field_list.go` | `--base-token`, `--help`, `--limit`, `--offset`, `--table-id` |
| `+field-search-options` | `shortcuts/base/field_search_options.go` | `--base-token`, `--field-id`, `--help`, `--keyword`, `--limit`, `--offset`, `--table-id` |
| `+field-update` | `shortcuts/base/field_update.go` | `--base-token`, `--field-id`, `--i-have-read-guide`, `--json`, `--table-id` |
| `+form-create` | `shortcuts/base/base_form_create.go` | `--base-token`, `--description`, `--name`, `--table-id` |
| `+form-delete` | `shortcuts/base/base_form_delete.go` | `--base-token`, `--form-id`, `--table-id` |
| `+form-detail` | `shortcuts/base/base_form_detail.go` | `--share-token` |
| `+form-get` | `shortcuts/base/base_form_get.go` | `--base-token`, `--form-id`, `--table-id` |
| `+form-list` | `shortcuts/base/base_form_list.go` | `--base-token`, `--page-size`, `--table-id` |
| `+form-questions-create` | `shortcuts/base/base_form_questions_create.go` | `--base-token`, `--form-id`, `--questions`, `--table-id` |
| `+form-questions-delete` | `shortcuts/base/base_form_questions_delete.go` | `--base-token`, `--form-id`, `--keep-field`, `--question-ids`, `--table-id` |
| `+form-questions-list` | `shortcuts/base/base_form_questions_list.go` | `--base-token`, `--form-id`, `--table-id` |
| `+form-questions-update` | `shortcuts/base/base_form_questions_update.go` | `--base-token`, `--form-id`, `--questions`, `--table-id` |
| `+form-share-get` | `shortcuts/base/form_share.go` | `--base-token`, `--form-id`, `--table-id` |
| `+form-share-update` | `shortcuts/base/form_share.go` | `--access-scope`, `--allow-anonymous`, `--base-token`, `--enabled`, `--form-id`, `--require-login`, `--table-id` |
| `+form-submit` | `shortcuts/base/base_form_submit.go` | `--base-token`, `--json`, `--share-token` |
| `+form-update` | `shortcuts/base/base_form_update.go` | `--base-token`, `--description`, `--form-id`, `--name`, `--table-id` |
| `+record-batch-create` | `shortcuts/base/record_batch_create.go` | `--base-token`, `--json`, `--table-id` |
| `+record-batch-update` | `shortcuts/base/record_batch_update.go` | `--base-token`, `--json`, `--table-id` |
| `+record-delete` | `shortcuts/base/record_delete.go` | `--base-token`, `--json`, `--record-id`, `--table-id` |
| `+record-download-attachment` | `shortcuts/base/record_upload_attachment.go` | `--base-token`, `--file-token`, `--output`, `--overwrite`, `--record-id`, `--table-id` |
| `+record-get` | `shortcuts/base/record_get.go` | `--base-token`, `--field-id`, `--field-names`, `--fields`, `--help`, `--jq-records`, `--json`, `--minimal-stdout`, `--output`, `--overwrite`, `--record-id`, `--table-id` |
| `+record-history-list` | `shortcuts/base/record_history_list.go` | `--base-token`, `--max-version`, `--page-size`, `--record-id`, `--table-id` |
| `+record-import` | `shortcuts/base/record_import.go` | `--artifact`, `--base-token`, `--mapping`, `--output`, `--table-id` |
| `+record-list` | `shortcuts/base/record_list.go` | `--base-token`, `--field-id`, `--field-names`, `--fields`, `--filter-json`, `--help`, `--jq-records`, `--limit`, `--minimal-stdout`, `--offset`, `--output`, `--overwrite`, `--sort-json`, `--table-id`, `--view-id` |
| `+record-remove-attachment` | `shortcuts/base/record_upload_attachment.go` | `--base-token`, `--field-id`, `--file-token`, `--record-id`, `--table-id` |
| `+record-search` | `shortcuts/base/record_search.go` | `--base-token`, `--field-id`, `--field-names`, `--fields`, `--filter-json`, `--help`, `--jq-records`, `--json`, `--keyword`, `--limit`, `--minimal-stdout`, `--offset`, `--output`, `--overwrite`, `--search-field`, `--sort-json`, `--table-id`, `--view-id` |
| `+record-share-link-create` | `shortcuts/base/record_share_link_create.go` | `--base-token`, `--help`, `--record-id`, `--table-id` |
| `+record-upload-attachment` | `shortcuts/base/record_upload_attachment.go` | `--base-token`, `--field-id`, `--file`, `--name`, `--record-id`, `--table-id` |
| `+record-upsert` | `shortcuts/base/record_upsert.go` | `--base-token`, `--json`, `--record-id`, `--table-id` |
| `+role-create` | `shortcuts/base/base_role_create.go` | `--base-token`, `--json` |
| `+role-delete` | `shortcuts/base/base_role_delete.go` | `--base-token`, `--role-id` |
| `+role-get` | `shortcuts/base/base_role_get.go` | `--base-token`, `--role-id` |
| `+role-list` | `shortcuts/base/base_role_list.go` | `--base-token` |
| `+role-update` | `shortcuts/base/base_role_update.go` | `--base-token`, `--json`, `--role-id` |
| `+table-copy` | `shortcuts/base/table_copy.go` | `--base-token`, `--name`, `--range`, `--table-id`, `--timeout`, `--wait` |
| `+table-copy-status` | `shortcuts/base/table_copy.go` | `--base-token`, `--task-id` |
| `+table-create` | `shortcuts/base/table_create.go` | `--base-token`, `--fields`, `--name`, `--view` |
| `+table-delete` | `shortcuts/base/table_delete.go` | `--base-token`, `--table-id` |
| `+table-get` | `shortcuts/base/table_get.go` | `--base-token`, `--table-id` |
| `+table-list` | `shortcuts/base/table_list.go` | `--base-token`, `--help`, `--limit`, `--offset` |
| `+table-update` | `shortcuts/base/table_update.go` | `--base-token`, `--name`, `--table-id` |
| `+template-categories` | `shortcuts/base/template_categories.go` | No domain flags; see shared output/auth controls |
| `+template-list` | `shortcuts/base/template_list.go` | `--category-key`, `--help`, `--limit`, `--offset` |
| `+template-search` | `shortcuts/base/template_search.go` | `--help`, `--keyword`, `--limit`, `--offset` |
| `+title-resolve` | `shortcuts/base/base_resolve.go` | `--help`, `--title` |
| `+url-resolve` | `shortcuts/base/base_resolve.go` | `--help`, `--url` |
| `+view-create` | `shortcuts/base/view_create.go` | `--base-token`, `--json`, `--table-id` |
| `+view-delete` | `shortcuts/base/view_delete.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get` | `shortcuts/base/view_get.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-card` | `shortcuts/base/view_get_card.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-filter` | `shortcuts/base/view_get_filter.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-group` | `shortcuts/base/view_get_group.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-sort` | `shortcuts/base/view_get_sort.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-timebar` | `shortcuts/base/view_get_timebar.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-get-visible-fields` | `shortcuts/base/view_get_visible_fields.go` | `--base-token`, `--table-id`, `--view-id` |
| `+view-list` | `shortcuts/base/view_list.go` | `--base-token`, `--help`, `--limit`, `--offset`, `--table-id` |
| `+view-rename` | `shortcuts/base/view_rename.go` | `--base-token`, `--name`, `--table-id`, `--view-id` |
| `+view-set-card` | `shortcuts/base/view_set_card.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+view-set-filter` | `shortcuts/base/view_set_filter.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+view-set-group` | `shortcuts/base/view_set_group.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+view-set-sort` | `shortcuts/base/view_set_sort.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+view-set-timebar` | `shortcuts/base/view_set_timebar.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+view-set-visible-fields` | `shortcuts/base/view_set_visible_fields.go` | `--base-token`, `--json`, `--table-id`, `--view-id` |
| `+workflow-create` | `shortcuts/base/workflow_create.go` | `--base-token`, `--json` |
| `+workflow-disable` | `shortcuts/base/workflow_disable.go` | `--base-token`, `--workflow-id` |
| `+workflow-enable` | `shortcuts/base/workflow_enable.go` | `--base-token`, `--workflow-id` |
| `+workflow-get` | `shortcuts/base/workflow_get.go` | `--base-token`, `--user-id-type`, `--workflow-id` |
| `+workflow-list` | `shortcuts/base/workflow_list.go` | `--base-token`, `--page-size`, `--status` |
| `+workflow-update` | `shortcuts/base/workflow_update.go` | `--base-token`, `--json`, `--workflow-id` |
| `+workspace-create` | `shortcuts/base/workspace_create.go` | `--name` |
| `+workspace-entity-list` | `shortcuts/base/workspace_entity_list.go` | `--page-size`, `--page-token`, `--type`, `--workspace-token` |
| `+workspace-move-in` | `shortcuts/base/workspace_move_in.go` | `--entity-token`, `--workspace-token` |


## Field and content coverage

Field definitions are passed intact through `+field-create` / `+field-update` (`PUT` replacement); table field arrays and all JSON inputs retain decimal tokens. Supported native definitions include text with phone/URL/email/barcode styles; number with plain/currency/progress/rating style configuration; static or dynamic select (options, colors, multiplicity, defaults); datetime; user/group_chat; created_at/updated_at/created_by/updated_by; links and bidirectional field identity; formula expression; lookup from/select/where/aggregate; auto_number rules; attachment; location; checkbox; button configuration. Optional description and defaults remain in native JSON. Older convenience field-spec aliases remain available but are not substitutes for the full native JSON schema.

Cell reads/writes preserve backend types, field/record IDs, Unicode, literal date text, null, empty arrays/strings and omitted keys. Formula/lookup/system values remain read-only values rather than being rewritten as user fields. Attachment upload/download/remove keeps the dedicated file protocol. Number payloads preserve decimal tokens to the backend; live numeric storage semantics are not inferred from this local precision guarantee. NDJSON validates every matrix shape/type, schema consistency across pages, record identity, pagination evidence and field-name uniqueness.

Views retain filter/group/sort/visibility/timebar/card native structures. Dashboard and App blocks retain nested configs, layout precision, ranking, data-query and all data source options. Workflow JSON retains trigger/actions, AI classification/analysis settings and button bindings. Forms retain full question definitions, ordering, updates, submissions and attachments. Shares, roles and advanced permissions retain all explicit boolean and member/role configuration fields; no testing restriction is a product allowlist.

## Intentional interface differences and safety evidence

- `--skip-unchanged` on record updates adds an explicit optional `base:record:read` prerequisite; legacy write scopes are unchanged. It compares exact submitted cells in a validated backend matrix and sends zero mutations when unchanged. Missing/ambiguous/incomplete reads stop the whole update before writing; omitted fields never delete data. Numbers are compared conservatively by token, so `1` and `1.0` may still write. Read and write are not atomic; other clients may race.
- All Base JSON object/array inputs now use number-preserving decoding. Syntax errors retain their original typed cause. Role response inner/double-encoded JSON likewise retains numeric precision.
- NDJSON no longer changes null selections/users to `[]`, null checkboxes to false, or local datetime strings to RFC3339. Manifest physical types describe nullable arrays/booleans; null and empty counts are separate. Numeric bounds preserve exact source values, approximate averages are marked; unparsed datetime statistics are explicit. Name collisions (including a user field named `record_id`) and duplicate field IDs fail instead of dropping values. Raw JSON reads still preserve those native matrices.
- List/search NDJSON requires explicit `has_more`; missing continuation evidence fails instead of treating the result as complete. Batch-get remains explicitly selected-record scoped. The 2,000 requested-row NDJSON ceiling remains; continuation metadata is retained and larger workflows page explicitly.
- `+record-import` consumes the existing shared result artifact containing `sheets-data/v1`, verifies current observed identity/tenant binding, checks exact complete source extent and destination field identity/type, and appends in 200-row batches. Every column is explicitly mapped. Unformatted primitive text/number/checkbox cells are supported; typed import rejects decimals that cannot round-trip through the documented Base double storage (including 9007199254740993); formulas, formatting (including date styles), rich/identity/object cells or unsupported conversion types are rejected before any write. Null, empty and omitted values remain distinct; a destination default that would fill an omitted source cell is rejected. It creates records every invocation, with no implicit business-key matching or deletion.
- Optional import `--output` saves all observed record IDs via existing invocation-scoped exclusive FileIO, keeping the 10,000-row payload outside stdout. Destination collision is checked before writes and races cannot overwrite files. If final report save fails, full observed effects remain on stdout with a typed failure. Unknown batches stop; IDs must be unique across the entire import, and an ID repeated from an earlier batch leaves the later batch unresolved without dispatching the next batch. Receipts are evidence, never automatic replay inputs.

## Dependencies and acceptance

Shared `common.RuntimeContext`, FileIO, precision-safe output, exact-request read markers, mutation journals and strict artifact binding remain the single foundation interfaces. Sheets owns `sheets-data/v1` producer; Base owns the explicit conversion and record mutation. The coordinator owns actual Sheets-export→Base-import cross-section runs, the frozen catalog reconciliation, section comparative trials and held-out evaluation. Base event envelopes are owned by events; this section does not implement another stream runtime.

Offline section command: `python3 scripts/next_test.py --section base --suite all --report .tmp/base-section.json`. Process suites use the real authsidecar binary via `nextfixture.RunMain` and synthetic non-forwarding HTTP handlers, isolated config and owned files. Written scenarios cover every command family, attachments, schemas, pagination, errors, unchanged zero-write updates, batch unknown/partial effects, large typed import, source/binding failures, no-clobber and report-save races. Exact executed commands and counts are recorded below; this offline evidence does not verify live backend behavior.

Live command is separately selectable: `python3 scripts/next_test.py --section base --suite live --allow-live`. It remains blocked until the harness proves the effective current account, tenant and private-at-creation visibility before content writes. No live write has been performed. Role/share/assignment/notification scenarios also require an unambiguous authorized other identity when applicable; a display-name match is insufficient. Existing reference live workflows are excluded from deterministic runs and are not counted as passes.

Comparative development trials are coordinator-scheduled and not yet run. No held-out task was used. Live-dependent capability acceptance remains open.

## Completed offline gate

Canonical build: `make build BINARY=.tmp/base-cli`, executed through `scripts/next_test.py`'s isolated environment. This uses the frozen metadata restore and standard build flags without replacing other sections' binaries.

`python3 scripts/next_test.py --section base --suite all --binary .tmp/base-cli --report .tmp/base-duplicate-section-gate.json` passed:

| Suite | Passed test events | Failed | Skipped |
| --- | ---: | ---: | ---: |
| `./shortcuts/base/...` | 1096 | 0 | 0 |
| `./tests/cli_e2e/base -run DryRun\|Regression` | 160 | 0 | 0 |
| `./tests/next_sections/base` | 177 | 0 | 1 |

The skip is the explicit live private-fixture blocker. Counts include Go subtests. The process suite verifies all 117 reference commands plus typed import; full source details are in [base-process.md](base-process.md). It also exercises 27 native field definitions/style variants and a 10,000-row import in 50 batches, with record IDs written to the domain report instead of stdout. Destination defaults cannot manufacture values for omitted source cells. Malformed, ignored-field or incomplete batch results stop the transfer, and simultaneous API/report-save failures preserve both error causes and full observed effects.

Focused shared affordance/schema regression (`go test -json ./shortcuts/base/... ./internal/affordance ./cmd/service ./internal/schema -count=1`) passed 1364 test events before the final small import/atomic-publication additions. The complete Base gate above includes those additions. `go vet ./shortcuts/base/... ./tests/next_sections/base` passed; owned Go files are formatted. Repository-wide lint, full unit/vet, quality-gate, coverage inventory reconciliation, live and comparative checks remain coordinator gates.

The first canonical dry-run run exposed inherited assertions that depended on pretty JSON spacing; they now inspect actual decoded request fields and respect the selected next binary. The recorded final gate is fully against the next implementation. Default NDJSON publication now preflights both destinations and uses existing `ExclusiveFileIO`; unsupported providers fail safely. Explicit `--overwrite` keeps replacement semantics. The two files are not one transaction: a manifest failure after record publication reports the retained record path and cause.

Commits owned by this section: `7c9c12b`, `72b932e`, `543fa68`, `8c63250`, `b16453c`, `a8e9983`, `e0b042f`, `b9529e9`, `1226fe4`; shared receipt and binding dependencies are tracked by the foundation owner. [Machine-readable gate summary](base-evidence.json) retains commands, counts and explicit limitations. Raw synthetic logs remain in `.tmp`.

Formula/lookup guide recovery is integrated: field validation retains the acknowledgement flag and names executable `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` or `lark-base-field-lookup.md` recovery commands in the structured error hint. Base affordances and the concise project skill route to the same existing embedded references. The foundation owner verified installation and byte-identical full reference reads in `b7833ae`; Base recovery commit `79fa235` passed 54 focused test events, including four real-process create/update error cases. This follow-up does not change the open live or release gates.

Cross-section testing found that IDs unique inside one response could repeat a previously confirmed batch. `f080e43` validates IDs across the whole import before accepting a batch; shared receipt fix `d91e695` checks invocation-scoped evidence for the same identity and table before finalization. The mixed-overlap process regression confirms only the first 200 of 401 requested rows, retains the ambiguous second response, dispatches exactly two batches, exits 22, and records one applied plus one unknown receipt. Stdout and stderr also report unknown. The full Base gate above was rerun after both fixes.
