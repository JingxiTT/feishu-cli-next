---
name: lark-base-concise
description: Operate Feishu Base tables, fields, records, views, forms, dashboards, workflows, roles, template center, BaseApp pages/components and Workspace resources. BaseApp belongs to Base, not lark-apps.
---

# Base

Use `lark-cli base`. Prefer `--as user`; use `--as bot` when the user explicitly requests application identity. Repair scope or resource ACL failures through `lark-shared` using the original identity. Change the actor only with explicit user agreement.

First establish `base_token` and entity type. Use supplied resource IDs or `<bitable>` / `<base_refer>` context directly. For URLs, `+url-resolve --url '<url>'` returns authoritative `resource_type` / `block_type` and coordinates such as table, view, record, dashboard, workflow, Docx or share tokens. BaseApp `/app/` links return `app_token` and any Workspace/page coordinates. For an ordinary Base title or keyword, use `+title-resolve --title '<keyword>'`; disambiguate multiple results using owner and update time, asking the user if still ambiguous. To list existing candidate Bases by owner, creator, time or recency, use Drive `+search --doc-types bitable`; `--sort open_time --opened-since 3m --page-size 20`, `--mine`, and `--created-by-me` distinguish those intentions. Enter Base commands using the returned URL/token. For BaseApp use a real `/app/` link or known Workspace with `+workspace-entity-list --type baseapp`; do not globally guess an app token from a name.

`+base-get` reads Base metadata. Create a new Base, its first table and fields together with `+base-create --name <name> --table-name <name> --fields '<array>'`. `+base-copy` copies a whole Base. The public template center is distinct from the user's Drive: when the user wants a ready-made template and supplies no existing-resource anchor, read `references/lark-base-template-center.md` and use `+template-categories`, `+template-list` or `+template-search`.

## Resource model

A Base owns a tree of Blocks: `table`, `dashboard`, `workflow`, `docx`, `folder`. Every Block has an ID, type, mutable name, parent folder and sibling order. Advanced permissions and Roles are Base-level configuration, outside this tree. Workspace organizes Bases and BaseApps. BaseApp organizes pages and components referencing Base data; it is not another name for Base.

Use `+base-block-list` to discover the tree; `+base-block-create`, `+base-block-rename`, `+base-block-move` and `+base-block-delete` manage it. Move uses `--parent-id` for hierarchy and `--before-id` / `--after-id` for sibling order. When initial type-specific content is already known, use `+table-create --fields`, `+dashboard-create` with theme, or `+workflow-create --json` with its complete definition. Folder and Docx use Block creation. Table/dashboard/workflow Block IDs are their module IDs; Docx carries a separate `docx_token`, whose body belongs to `lark-doc`. Folder IDs organize child Blocks; list direct children with `--parent-id`.

## Tables, schema and cells

Tables contain Fields, Records, Views and Forms; these are not Base Blocks. Locate tables with `+table-list`; `+table-get` reads detail, `+table-copy` copies a table and `+table-copy-status` observes its asynchronous status. Ordinary reads with known target and field names can go straight to Record commands. Obtain field schema when writing, filtering or linking requires field types. Independent tables' field lists may run concurrently.

Many table updates propagate asynchronously. Prefer the write response as confirmation. If the task needs final-state verification or the response is insufficient, finish related changes then read back the target together; immediate per-item readbacks may lag.

Fields use stable `field_id`; names are mutable display labels. Read with `+field-list`, `+field-get`, `+field-search-options`. Prefer one `+field-create --json '<field-object-array>'` for multiple new fields. Update/delete with `+field-update` / `+field-delete`. Read `references/lark-base-field-create.md` or `lark-base-field-update.md` before constructing schema; these route to full Field JSON, Formula and Lookup protocols. Field plugins generate values from other cells in the same row into an existing text, select or number field; read `lark-base-field-extension.md` before configuring or triggering them.

A Record is one row. Its nonempty unique `record_id` is the stable key; the table's primary display field is not that key. A write is `{field_id_or_name: CellValue}`. Preserve native value types:

- Text is a string. URL text accepts a bare URL or Markdown link; phone/email text retains its valid string representation.
- Select is an array of existing option strings; a single-select has at most one element.
- Number is an unformatted double, not a formatted numeric string.
- Datetime accepts a timezone-bearing string, a timezone-free string interpreted in the Base timezone, or Unix milliseconds. Do not substitute seconds.
- Checkbox is a boolean. Users and group chats use arrays of `{id}` with the field's single/multiple constraint. Linked records use arrays of `{id}` from the destination table. Location is `{lng,lat}`.
- `null` explicitly clears a cell; `[]` clears array-valued cells and differs structurally from null. Omitted keys are not submitted changes.

Use the dedicated attachment upload/download/remove shortcuts. `created_at`, `updated_at`, `created_by`, `updated_by`, `auto_number`, `formula`, and `lookup` cells are read-only. Writing them can return `ignored_fields` while other submitted fields are applied; inspect that evidence.

## Reading records

Known IDs: `+record-get --record-id <id>` repeated as needed. Keyword search: `+record-search --keyword <text> --search-field <field>` with at least one search field. Other reads: `+record-list`; use `--filter-json` and `--sort-json` for structured conditions and sorting.

Project necessary fields with repeated `--field-id`. Write record reads to NDJSON artifacts with `--format ndjson --output <path>.ndjson`; each line is one row, and stdout includes `records_count` and `has_more`. Inspect bounded head/tail samples and use local jq or Python for analysis. The default and maximum record limit is 2000; `--offset` paginates. Treat a result as complete only when `has_more=false` and the query covers the question. For fewer than roughly 500 rows, prefer local filtering; for large inputs push supported predicates to the server and handle regex/derived predicates locally.

Tuple filters use `{"logic":"and","conditions":[...]}`; `or` requires any condition. Text containment and select membership use `intersects`; links use `intersects` with an array of `{id}`. `empty` / `non_empty` have no value. Numbers support `==`, `!=`, `>`, `>=`, `<`, `<=`. Date `== "ExactDate(YYYY-MM-DD)"` means that day in the Base timezone. Dates do not support `>=`; an inclusive lower day boundary can use `>` the previous day's last millisecond, with `<` the next boundary for a range. Read `references/lark-base-filter-condition.md` for full operators and typed values. For complete large-table or View reads, joins, collections, time series, semantic or specialized analysis, read `lark-base-record-query-and-analysis-sop.md`.

## Writing records

`+record-batch-create --json '{"create_records":[{...},{...}]}'` creates rows and returns record IDs. `+record-batch-update --json '{"update_records":{"<record_id>":{...}}}'` updates exact rows; submit only changed fields. `+record-upsert --record-id <id> --json '{...}'` is the single-record update path. Use `--json @file.json` for larger payloads. A batch holds at most 200 rows. Write sequentially to the same table; concurrent writes can produce `1254291` conflicts.

`+record-delete` and `+record-share-link-create` take repeated record IDs. Read `references/lark-base-record-history-list.md` for `+record-history-list`. Attachments use `+record-upload-attachment`, `+record-download-attachment`, `+record-remove-attachment`; do not replace their dedicated file protocol with a guessed cell representation.

## Views and forms

Views are persistent filters, sorting, grouping and display configuration over the same records, not data copies. Use direct Record reads for one-off queries; use a View when the UI arrangement must persist or be shared. `+view-list` / `+view-get` discover Views; `+view-get-filter/sort/group/visible-fields/timebar/card` read their configuration. Manage lifecycle with `+view-create/rename/delete`, and corresponding `+view-set-*` commands for settings. Read `references/lark-base-view-set-filter.md` and its condition reference for filters.

Forms belong to a table and use Fields as questions; each valid submission creates a Record. Read configuration using `+form-list`, `+form-get`, `+form-questions-list` with the owning Base/table. Manage forms with `+form-create/update/delete`, and questions with `+form-questions-create/update/delete`; read the corresponding question create/update references. A new-field question requires `title` and `type`; an existing-field question uses `use_existing_field:true` plus `field_id` and must not contain type/style/options. Adding an existing field does not create a field or change data. List questions first; if a title already exists, prefer updating unless a separate same-name question was explicitly requested.

Question deletion normally deletes the underlying Field and all its record values. To remove only the question, use `--keep-field`; the retained Field can later be added as an existing-field question. Read current sharing with `+form-share-get`, then change only one share setting at a time with `+form-share-update`, explicitly retaining true/false values. For filling a shared Form, resolve its URL to a share token, read actual questions/requirements/conditions via `+form-detail` and `references/lark-base-form-detail.md`, then use `lark-base-form-submit.md` to construct a valid submission and attachments.

## Dashboards, BaseApp and Workspace

A Dashboard Block is a themed layout container. `+dashboard-list/get/update/arrange` discover/read/style/arrange it. Read sharing before changing it with `+dashboard-share-get/update`; change one setting at a time and preserve explicit false. Dashboard-internal chart, metric and text components are also called Blocks but are not part of the Base resource tree. `+dashboard-block-list/get` read layout/configuration including text; `+dashboard-block-create/update/delete` manage configuration; `+dashboard-block-get-data` reads calculated chart/metric results. Read `references/lark-base-dashboard.md` before operating components.

Workspace owns Base and BaseApp resources. BaseApp must belong to a Workspace and owns Pages containing chart/list/rich-text components. Their `data_config` references Base data without changing data ownership. Workspace uses `+workspace-create`, `+workspace-entity-list`, `+workspace-move-in`; apps use `+app-create/get`, pages use `+app-page-list/get/create/rename/delete`, and components use `+app-block-list/get/create/update/get-data`.

Read `references/lark-base-app.md` completely before Workspace/BaseApp/Page/component work; read `lark-base-app-block-data-config.md` when constructing component configuration. BaseApp is not `lark-apps`. BaseApp copy, complete Page copy, page icons and moving resources out of Workspace are unsupported; do not substitute empty new objects or Drive moves. Keep `app_token`, `page_id` and `block_id` distinct from the source data's `base_token`. A copied data_config is only a structural template: align every user requirement before its first create/update, and explicitly set requested sort direction in `group_by[].sort.order` or top-level `sort.order`. Current output order or an omitted old direction is not evidence of the requested sort. App and dashboard component IDs are not interchangeable even though their underlying model is related.

## Workflows and permissions

Workflow is a Base Block containing a `steps` execution graph connected by `next` / `children`. Triggers, actions, branches and loops support schedules, record changes, notifications, data access and cross-system calls. `+workflow-list/get` reads title, status and complete steps. Read `references/lark-base-workflow.md` before constructing or changing full definitions with `+workflow-create/update`. `+workflow-enable/disable` changes running state without changing steps.

Advanced permissions enable granular Base access. Roles control Base, table, view, field, record, dashboard and Docx visibility/editing/copy/download/data rules. Read `is_advanced` with `+base-get` and roles with `+role-list/get`. Use `+advperm-enable/disable` and `+role-create/update/delete` only after reading `references/lark-base-advanced-permission-and-role.md` and its permission JSON protocol.

For every update, distinguish full replacement from delta: use a trustworthy current configuration for read-modify-write replacement, and only the intended changes for delta. Prefer write-response confirmation; read back when insufficient or requested. When a command has a confirmation gate, establish target and effects before using `--yes`.

Authentication, setup, scopes and identity recovery belong to `lark-shared`. Import/export between local Excel/CSV/`.base` files and Base belongs to `lark-drive`; online Base copying uses `+base-copy`. Embedded Docx bodies belong to `lark-doc`, and spreadsheet contents to `lark-sheets`. Full original references remain available for protocol details.
