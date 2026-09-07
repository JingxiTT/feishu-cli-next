# Sheets section reconciliation

Status: **open**. Offline process evidence is distinct from live capability acceptance. No comparative evaluation or held-out task has run for this section. Live mutation verification remains blocked until effective account, tenant, and account-only creation visibility can be proved before a write.

## Scope and ownership

The frozen reference is sibling `feishu-cli-review` revision `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, captured clean. The complete source/capture identity is in [the reference manifest](../reference/README.md). The catalog and registration contracts are `docs/reference/meta_data.json.gz` and `docs/reference/registrations.json.gz`; neither is hand-edited. The Sheets surface has 103 nodes: the service root, 91 shortcuts, three generated resource groups, and eight generated methods. Every frozen entry maps to the identical `feishu-cli-next sheets …` path; no frozen operation is replaced by raw API forwarding.

The Sheets owner maintains `shortcuts/sheets/`, reference regression tests under `tests/cli_e2e/sheets/`, and the section process tests under `tests/next_sections/sheets/`. The coordinator owns `tests/nextfixture`, generic schemas, identity/bindings, result artifacts, mutation receipts, raw API access, and registration reconciliation. Drive owns generic upload/export/import mechanics; Wiki owns resource resolution. Source reuse preserves the reference flags, generated types, risk, scopes, and identity choices. The authsidecar process intentionally hides the unavailable local `--profile` selector; the normal CLI retains it. Generated `--yes` follows effective command risk, not the catalog's separate `danger` boolean.

## Capability mapping

Every command in the tables below is prefixed with `feishu-cli-next sheets`. Common output and identity controls remain shared: `--as`, `--format`, `--json`, `--jq`, local `--profile`, `--dry-run`, applicable `--yes`, plus opt-in `--result-fields`, `--result-detail`, and `--save-result`. `--spreadsheet-token` retains the `--token` alias. Structured inputs retain `@file` and the supported stdin convention. A Sheets domain `--range`, `--include`, or `--output-path` keeps its domain meaning.

The following table is generated from the complete frozen shortcut registration inventory. An asterisk identifies a flag marked required by that registration; cross-field validation can impose additional selector or payload requirements. Flag defaults, types, aliases, enum values, required fields and nested JSON schemas remain in the frozen artifact and executable `--help` / `--print-schema --flag-name NAME` surface rather than being reinterpreted here.

Evidence labels: **P** means the process suite runs both dry-run and real CLI execution against a synthetic backend; **D** means this process suite asserts dry-run behavior and the specialized execution path is covered by nearby unit/reference tests, with live execution unverified. These labels describe test depth, not full functional acceptance. Each path is explicitly represented in `cases.json`; `TestShortcutRequests` fails if a frozen shortcut lacks a case.

| Shortcut | Registered domain flags | Tool/request family | Process |
| --- | --- | --- | --- |
| `+batch-chart-create` | `--url`, `--spreadsheet-token`, `--operations`*, `--continue-on-error` | `batch_update` | P |
| `+batch-chart-update` | `--url`, `--spreadsheet-token`, `--operations`*, `--continue-on-error` | `batch_update` | P |
| `+batch-update` | `--url`, `--spreadsheet-token`, `--operations`*, `--continue-on-error` | `batch_update` | P |
| `+cells-batch-clear` | `--url`, `--spreadsheet-token`, `--ranges`*, `--scope` | `batch_update` | P |
| `+cells-batch-set-style` | `--url`, `--spreadsheet-token`, `--ranges`*, `--background-color`, `--font-color`, `--font-family`, `--font-size`, `--font-style`, `--font-weight`, `--font-line`, `--horizontal-alignment`, `--vertical-alignment`, `--word-wrap`, `--number-format`, `--border-styles` | `batch_update` | P |
| `+cells-clear` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--scope` | `clear_cell_range` | P |
| `+cells-get` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--include`, `--max-chars`, `--output-path`, `--skip-hidden` | `get_cell_ranges` | P |
| `+cells-merge` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--merge-type` | `merge_cells` | P |
| `+cells-replace` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--find`*, `--replacement`*, `--range`, `--match-case`, `--match-entire-cell`, `--regex`, `--include-formulas` | `replace_data` | P |
| `+cells-search` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--find`*, `--range`, `--match-case`, `--match-entire-cell`, `--regex`, `--include-formulas`, `--max-matches`, `--offset` | `search_data` | P |
| `+cells-set` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`, `--cells`, `--writes`, `--allow-overwrite`, `--max-cells`, `--copy-to-range` | `set_cell_range` | P |
| `+cells-set-image` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--image`*, `--name` | `Drive media upload and cell write` | D |
| `+cells-set-style` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--background-color`, `--font-color`, `--font-family`, `--font-size`, `--font-style`, `--font-weight`, `--font-line`, `--horizontal-alignment`, `--vertical-alignment`, `--word-wrap`, `--number-format`, `--border-styles` | `set_cell_range` | P |
| `+cells-unmerge` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`* | `merge_cells` | P |
| `+changeset-get` | `--url`, `--spreadsheet-token`, `--start-revision`*, `--end-revision` | `get_changeset` | P |
| `+chart-config-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-id`*, `--title`, `--subtitle`, `--legend-position`, `--x-axis-title`, `--y-axis-title`, `--secondary-y-axis-title`, `--x-axis-label-angle`, `--y-axis-label-angle`, `--x-axis-min`, `--x-axis-max`, `--y-axis-min`, `--y-axis-max`, `--data-labels`, `--data-label-position`, `--last-point-label`, `--stack`, `--stacked`, `--smooth`, `--color-palette`, `--colors` | `manage_chart_object` | P |
| `+chart-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--properties`*, `--print-example` | `manage_chart_object` | P |
| `+chart-create-basic` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-type`*, `--data-range`*, `--header-range`, `--data-direction`, `--x-axis-numbers-as`, `--x-axis-min`, `--x-axis-max`, `--y-axis-min`, `--y-axis-max`, `--dim1-index`, `--dim2-indexes`, `--series-types`, `--series-y-axes`, `--key-index`, `--x-index`, `--y-index`, `--group-index`, `--size-index`, `--title`, `--subtitle`, `--legend-position`, `--x-axis-title`, `--y-axis-title`, `--secondary-y-axis-title`, `--x-axis-label-angle`, `--y-axis-label-angle`, `--data-labels`, `--data-label-position`, `--stack`, `--stacked`, `--smooth`, `--color-palette`, `--colors`, `--anchor-cell`, `--width`, `--height` | `manage_chart_object` | P |
| `+chart-data-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-id`*, `--data-range`*, `--header-range`, `--data-direction`, `--dim1-index`, `--dim2-indexes`, `--key-index`, `--x-index`, `--y-index`, `--group-index`, `--size-index` | `manage_chart_object` | P |
| `+chart-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-id`* | `manage_chart_object` | P |
| `+chart-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-id` | `get_chart_objects` | P |
| `+chart-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--chart-id`*, `--properties`* | `manage_chart_object` | P |
| `+cols-resize` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--width`, `--widths`, `--type`, `--range` | `resize_range` | P |
| `+cond-format-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--properties`*, `--rule-type`*, `--ranges`* | `manage_conditional_format_object` | P |
| `+cond-format-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--rule-id`* | `manage_conditional_format_object` | P |
| `+cond-format-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--rule-id` | `get_conditional_format_objects` | P |
| `+cond-format-result-get` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--max-chars` | `get_cell_ranges` | P |
| `+cond-format-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--rule-id`*, `--properties`*, `--rule-type`*, `--ranges`* | `manage_conditional_format_object` | P |
| `+csv-get` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`, `--max-chars`, `--output-path`, `--include-row-prefix`, `--skip-hidden` | `get_range_as_csv` | P |
| `+csv-put` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--start-cell`*, `--csv`*, `--allow-overwrite`, `--range` | `set_range_from_csv` | P |
| `+dim-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`, `--ranges` | `batch_update` | P |
| `+dim-freeze` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--dimension`, `--count`, `--rows`, `--cols` | `modify_sheet_structure` | P |
| `+dim-group` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--depth`, `--group-state`, `--range`* | `modify_sheet_structure` | P |
| `+dim-hide` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`* | `modify_sheet_structure` | P |
| `+dim-insert` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--inherit-style`, `--position`*, `--count`* | `modify_sheet_structure` | P |
| `+dim-move` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--source-range`*, `--target`* | `Sheets v3 move_dimension` | D |
| `+dim-ungroup` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--depth`, `--range`* | `modify_sheet_structure` | P |
| `+dim-unhide` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`* | `modify_sheet_structure` | P |
| `+dropdown-delete` | `--url`, `--spreadsheet-token`, `--ranges`* | `batch_update` | P |
| `+dropdown-get` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`* | `get_cell_ranges` | P |
| `+dropdown-set` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--options`, `--colors`, `--multiple`, `--highlight`, `--source-range` | `set_cell_range` | P |
| `+dropdown-update` | `--url`, `--spreadsheet-token`, `--ranges`*, `--options`, `--colors`, `--multiple`, `--highlight`, `--source-range` | `batch_update` | P |
| `+filter-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--properties` | `manage_filter_object` | P |
| `+filter-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `manage_filter_object` | P |
| `+filter-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `get_filter_objects` | P |
| `+filter-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--properties`*, `--range`* | `manage_filter_object` | P |
| `+filter-view-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--properties`*, `--range`*, `--view-name` | `manage_filter_view_object` | P |
| `+filter-view-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--view-id`* | `manage_filter_view_object` | P |
| `+filter-view-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--view-id` | `get_filter_view_objects` | P |
| `+filter-view-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--view-id`*, `--properties`*, `--range`, `--view-name` | `manage_filter_view_object` | P |
| `+float-image-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--image-name`*, `--image-token`, `--image-uri`, `--position-row`*, `--position-col`*, `--size-width`*, `--size-height`*, `--offset-row`, `--offset-col`, `--z-index`, `--image` | `manage_float_image_object` | P |
| `+float-image-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--float-image-id`* | `manage_float_image_object` | P |
| `+float-image-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--float-image-id` | `get_float_image_objects` | P |
| `+float-image-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--float-image-id`*, `--image-name`*, `--image-token`, `--image-uri`, `--position-row`*, `--position-col`*, `--size-width`*, `--size-height`*, `--offset-row`, `--offset-col`, `--z-index` | `manage_float_image_object` | P |
| `+formula-verify` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`, `--max-locations`, `--exit-on-error`, `--ai-only` | `verify_formula` | P |
| `+history-list` | `--url`, `--spreadsheet-token`, `--end-version` | `history_list` | P |
| `+history-revert` | `--url`, `--spreadsheet-token`, `--history-version-id`* | `history_revert` | P |
| `+history-revert-status` | `--url`, `--spreadsheet-token`, `--transaction-id`* | `history_revert_status` | P |
| `+pivot-create` | `--url`, `--spreadsheet-token`, `--properties`*, `--target-position`, `--target-sheet-id`, `--target-sheet-name`, `--source`*, `--range` | `manage_pivot_table_object` | P |
| `+pivot-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--pivot-table-id`* | `manage_pivot_table_object` | P |
| `+pivot-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--pivot-table-id` | `get_pivot_table_objects` | P |
| `+pivot-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--pivot-table-id`*, `--properties`* | `manage_pivot_table_object` | P |
| `+range-copy` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--source-range`*, `--target-sheet-id`, `--target-range`*, `--paste-type` | `transform_range` | P |
| `+range-fill` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--source-range`*, `--target-range`*, `--series-type` | `transform_range` | P |
| `+range-move` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--source-range`*, `--target-sheet-id`, `--target-range`* | `transform_range` | P |
| `+range-sort` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`*, `--sort-keys`*, `--has-header` | `transform_range` | P |
| `+revision-get` | `--url`, `--spreadsheet-token` | `get_workbook_structure` | P |
| `+rows-resize` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--height`, `--heights`, `--type`, `--range` | `resize_range` | P |
| `+sheet-copy` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--title`, `--index` | `modify_workbook_structure` | P |
| `+sheet-create` | `--url`, `--spreadsheet-token`, `--title`*, `--index`, `--row-count`, `--col-count`, `--type` | `modify_workbook_structure` | P |
| `+sheet-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `modify_workbook_structure` | P |
| `+sheet-hide` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `modify_workbook_structure` | P |
| `+sheet-hide-gridline` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `modify_workbook_structure` | P |
| `+sheet-info` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--include`, `--range` | `get_sheet_structure` | P |
| `+sheet-list` | `--url`, `--spreadsheet-token` | `get_workbook_structure` | P |
| `+sheet-move` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--index`*, `--source-index` | `modify_workbook_structure` | P |
| `+sheet-rename` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--title`* | `modify_workbook_structure` | P |
| `+sheet-set-tab-color` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--color`* | `modify_workbook_structure` | P |
| `+sheet-show-gridline` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `modify_workbook_structure` | P |
| `+sheet-unhide` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name` | `modify_workbook_structure` | P |
| `+sparkline-create` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--properties`* | `manage_sparkline_object` | P |
| `+sparkline-delete` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--group-id`* | `manage_sparkline_object` | P |
| `+sparkline-list` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--group-id` | `get_sparkline_objects` | P |
| `+sparkline-update` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--group-id`*, `--properties`* | `manage_sparkline_object` | P |
| `+styles-put` | `--url`, `--spreadsheet-token`, `--styles`* | `batch_update` | P |
| `+table-get` | `--url`, `--spreadsheet-token`, `--sheet-id`, `--sheet-name`, `--range`, `--max-chars`, `--output-path`, `--no-header` | `DataFrame typed projection` | D |
| `+table-put` | `--url`, `--spreadsheet-token`, `--sheets`*, `--styles` | `DataFrame to typed write plan` | D |
| `+workbook-create` | `--title`*, `--folder-token`, `--values`, `--sheets`, `--styles` | `Sheets create and initialization` | D |
| `+workbook-export` | `--url`, `--spreadsheet-token`, `--file-extension`, `--sheet-id`, `--output-path` | `Drive export task and download` | D |
| `+workbook-import` | `--file`*, `--folder-token`, `--name` | `Drive upload and import task` | D |
| `+workbook-info` | `--url`, `--spreadsheet-token` | `get_workbook_structure` | P |

Generated methods use `schema sheets RESOURCE METHOD` plus `--params` and `--data` (including file inputs) through the generic service runtime. `TestGeneratedSchemasAndRequests` reads the frozen metadata, recursively checks all request/response fields and types including array item fields, then checks each method's dry-run and actual synthetic HTTP method, resource identity, request body, query parameters and lossless result forwarding.

| Generated resource/method | Endpoint | Data/schema responsibility |
| --- | --- | --- |
| `spreadsheets create` | `POST /sheets/v3/spreadsheets` | title, folder token, without_mount; created workbook identity |
| `spreadsheets get` | `GET /sheets/v3/spreadsheets/{spreadsheet_token}` | owner identity selected by user_id_type; workbook metadata |
| `spreadsheets patch` | `PATCH /sheets/v3/spreadsheets/{spreadsheet_token}` | title update |
| `spreadsheet.sheet.filters create` | `POST /sheets/v3/spreadsheets/{spreadsheet_token}/sheets/{sheet_id}/filter` | range, column, condition and expected values |
| `spreadsheet.sheet.filters get` | `GET /sheets/v3/spreadsheets/{spreadsheet_token}/sheets/{sheet_id}/filter` | filter range, filtered rows, nested filter conditions |
| `spreadsheet.sheet.filters update` | `PUT /sheets/v3/spreadsheets/{spreadsheet_token}/sheets/{sheet_id}/filter` | column and replacement condition |
| `spreadsheet.sheet.filters delete` | `DELETE /sheets/v3/spreadsheets/{spreadsheet_token}/sheets/{sheet_id}/filter` | explicit workbook/sheet identity |
| `spreadsheet.sheets find` | `POST /sheets/v3/spreadsheets/{spreadsheet_token}/sheets/{sheet_id}/find` | find text, range, case/whole-cell/regex/formula conditions and result positions |

The root and three generated resource groups remain navigation nodes. The command-independent `api` endpoint remains available through the shared foundation and is not counted as a Sheets shortcut implementation.

## Source and test families

- Workbook, sheet identity, tab color/visibility/gridlines and creation: `lark_sheet_workbook.go`, `lark_sheet_sheet_list.go`; matching workbook/sheet unit tests and `sheets_sheet_*`, create and CRUD reference workflows.
- Cell reads, styles, dropdowns, CSV and table conversion: `lark_sheet_read_data.go`, `lark_sheet_write_cells.go`, `lark_sheet_table_io.go`, `lark_sheet_styles_put.go`; cell shapes, typed tables, read projections, CSV overwrite guards and style vocabulary regressions.
- Dimensions and range operations: `lark_sheet_sheet_structure.go`, `lark_sheet_range_operations.go`; qualified range, sparse deletion ordering, simultaneous row/column freeze, merge/copy/fill/sort and dimension workflow regressions.
- Object lifecycles: `lark_sheet_object_crud.go`, `lark_sheet_object_list.go`, `lark_sheet_chart.go`; chart snapshots and semantic chart edits, pivot placement, conditional format rule-specific attributes, filter scope, filter-view replacement rules, sparkline IDs and floating image metadata.
- Batch translation and partial failures: `lark_sheet_batch_update.go`, `batch_op_dispatch.go`, `sheet_ai_api.go`; per-operation flag/schema validation, fail-fast and continue-on-error semantics and readable partial failures.
- History and formula status: `lark_sheet_history_list.go`, `lark_sheet_history_revert.go`, `lark_sheet_revision_get.go`, `lark_sheet_changeset.go`, `lark_sheet_formula_verify.go`; predicate result placement, polling identity and missing/invalid revision/result errors.
- Native transfer: `data_transfer.go`; raw native cell JSON, complete extent validation, binding, conversion validation and unchanged-cell suppression.

`TestFrozenDiscovery` exercises all frozen help nodes and compares every structured flag schema exactly to the frozen schema; it does not equate help presence with behavior. The independent repository registration comparison also owns flags' types/defaults/aliases, hidden flags, risk/identity/scope metadata and both brand variants. The process suite exercises 91 explicit shortcut scenarios, all eight generated methods, and independent error/native-transfer assertions. Reference tests use the isolated deterministic `DryRun|Regression` selector; excluded live workflows are not counted as passes.

## Intentional new interface and data integrity

`+data-export --spreadsheet-token TOKEN --sheet-id ID --range A1:D100 --save-result PATH` adds a native, versioned, fully bounded cell artifact. `+data-import` consumes that artifact with explicit target identity and extent. The artifact contains `feishu-cli-next/sheets-data/v1` inside the shared bound result envelope. Import requires verified current tenant/identity binding. It rejects incomplete or incompatible artifacts, validates every conversion before any write, and emits `noop`, changed-cell and acknowledged-page evidence. Read pages contain at most 5,000 cells; the supported explicit extent is at most 1,000,000 cells and 5,000 columns. This is not a snapshot or compare-and-swap guarantee.

Native JSON preserves decimal tokens, large integer values, leading zeros in strings, booleans, formula fields, styles and field presence. Omitted cell fields are patches, not inferred deletion. Unsupported cell write fields and null writes without a verified conversion fail before mutation. Literal text starting `=` uses a native text conversion rather than silently evaluating a formula. Equal cached formula results do not justify replacing formula identity. Existing `+table-get` / `+table-put` stay DataFrame conversion tools; CSV/DataFrame output is not a native round-trip artifact. Native cross-section transfer to Base remains P3 work, including the required 10,000-row combined workflow. Base currently accepts primitive cell values only and rejects formulas/styles rather than silently converting them; formula/style transfer is not claimed supported by that consumer.

Process evidence explicitly uses 5,001 rows to force two native read pages, preserves `9007199254740993` and `0.1234567890123456789` in the saved artifact, reads the same two pages for import, and asserts zero write calls and no mutation receipt files for unchanged data. A changed native import also asserts the exact large-integer wire token, one changed cell and one acknowledged write page. Truncated export must fail without creating an artifact. Formula errors preserve their predicate result on stdout while returning a typed failure on stderr. Malformed tool output is a typed invalid response. Partial batch failure reports an unknown mutation outcome with the original backend cause and log ID; succeeded work is not claimed rolled back and the test asserts no automatic replay.

## Verification and limits

The executable isolated process command is:

```sh
make build BINARY=.tmp/sheets/feishu-cli-next
python3 scripts/next_test.py --section sheets --suite all --binary .tmp/sheets/feishu-cli-next --report .tmp/sheets/section-report.json
```

The process package also supports direct `go test ./tests/next_sections/sheets` only inside `scripts/next_test.py`'s `isolated_env`; `TestMain` rejects an unisolated developer environment. It builds the real authsidecar CLI against the restored frozen catalog. Synthetic fixtures never forward requests to real Feishu endpoints. Normal production build evidence remains separate from this alternate transport build.

The process suite includes `TestBatchMemberFailureInsideSuccessfulEnvelope`: even when the outer API code is zero, a nested member failure returns exit 22 with an unknown outcome and exactly one backend invocation. Shared fix `1bcdb68` interprets the known Sheets tool envelope. `batch_update` and asynchronous `history_revert` acknowledgements remain unknown until domain-specific reconciliation establishes completion. Malformed nested write output also remains unknown. Every batch/async case in the process matrix asserts this conservative outcome while preserving the actual request assertions.

The final combined isolated suite passed on 2026-09-07 with the canonical normal build at `1bcdb68` plus the accompanying test assertions. Counts include Go parent and subtest events:

| Suite | Pass | Fail | Explicit skips |
| --- | ---: | ---: | ---: |
| `go test -json -count=1 -timeout=10m ./shortcuts/sheets/...` | 1,545 | 0 | 0 |
| `go test -json -count=1 -timeout=10m ./tests/cli_e2e/sheets -run 'DryRun|Regression'` | 68 | 0 | 0 |
| `go test -json -count=1 -timeout=10m ./tests/next_sections/sheets` | 277 | 0 | 7 |

The reference selector leaves 14 named live workflows unselected; they are unverified, not passing tests. Full sanitized local logs and selection evidence are in `.tmp/sheets/section-report.json`. Scoped `go vet ./shortcuts/sheets/...`, canonical `make build`, and `go test ./internal/affordance ./cmd/service ./internal/schema` also passed. The coordinator owns whole-repository lint, inventory reconciliation, comparative evaluation and release gates.

Artifact process assertions verify that saved cell matrices are absent from stdout while the reference retains exact row count and completeness. The native unchanged import performs zero writes and creates zero mutation receipts. Earlier fixture failures and the original false batch-success regression are resolved by the final run; no failed attempt is counted as passing evidence.

P3 review identified a further replacement comparison error: plain `{"value":"hello"}` was considered unchanged against equal displayed text backed by a link or multiple-value content. The importer now compares content identity before claiming a plain-value no-op. The inverse regression fails on the preceding implementation for both native fields. `TestNativePlainImportReplacesStructuredIdentityThenNoops` exercises a real saved export, then checks the exact plain native write, independently stored replacement, one applied receipt, and a second import with zero further writes or receipts. The complete affected isolated gate passed on 2026-09-07: 1,548 unit events, 68 reference regression events, and 280 process events, with zero failures and the same seven explicit live blockers. The canonical build and scoped vet passed. Evidence is `.tmp/sheets/rich-replacement-report.json`; no live behavior is inferred from these fixtures.

The selectable `TestLivePrivateSheetsAcceptance` scenarios cover private workbook create/use/cleanup; typed cells/formulas/native transfer; dimensions/ranges; object lifecycles; file import/export; history/changesets; and generated filter/find methods. They are deliberately skipped with an explicit blocker. The harness has not proved private creation visibility for the effective live account and tenant. No live content, permissions, notifications or cleanup runs occur. Ying Wu is the sole authorized other recipient, and no scenario here requires delivery to that person. Existing unrelated resources must remain untouched. Skips do not satisfy live acceptance.

Backend limitations remain material: facade history revert/status availability is not proven by a successful synthetic response; sparse/large reads can be incomplete and native export rejects them; reference APIs do not establish remote concurrency control; rich-field conversion and server rendering need live verification. All supported flag combinations and media transfer paths require their existing unit/reference regressions in addition to this process suite. No efficiency claim is made without the scheduled comparative development evaluation. Live and comparative requirements keep the section open.

## Implementation commits

- `9777e05`: native cell export/import, exact raw-number boundary, canonical flag metadata, read-only tool requests, focused units and affordance.
- `e7cfec6`: reported range/row/column validation and static-value no-op correctness with empty formula metadata.
- `838d8df`, `cfd8f6c`: preserve rich-content identity and reject ambiguous empty structured-content conversions before writes.
- `afca95d`: complete frozen mapping and executable process coverage. The final accompanying assertions pin conservative batch/async outcomes, malformed-write uncertainty and compact artifact stdout.

No live mutation, recipient delivery, benchmark or agent evaluation was performed by the section owner. The implementation is ready for authorized private-fixture verification and coordinator-scheduled comparative development tasks; those outstanding gates keep the section open.
