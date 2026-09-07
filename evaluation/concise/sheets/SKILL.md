---
name: lark-sheets
version: 3.1.8
description: Create, read and edit Lark/Feishu spreadsheets, cells, formulas, formatting, sheet structure, charts, pivots, filters and other native objects. Use for spreadsheet analysis and financial models, including /sheets/ links on doubao.com. Use lark-drive to find a cloud file by name or keyword.
metadata:
  requires:
    bins: ["lark-cli"]
    siblings: ["lark-shared"]
  cliHelp: "lark-cli sheets --help"
---

# Sheets

Read `../lark-shared/SKILL.md` before starting; its identity, authentication, permission, output and high-risk rules apply. Select the identity deliberately: user acts as the person, bot as the application and can return empty results for inaccessible user resources. Read the matching shared reference when selecting identity, handling scopes or authorization, interpreting output, scripting, encountering exit 10, configuring the CLI or handling `_notice`. Never print credentials. Preserve authorization URLs exactly and accompany them with the shared QR-code workflow.

A spreadsheet/workbook is the top-level container; a sheet/tab is its child. A `sheet_id` is stable. An object's `reference_id` is its stable object identifier, not an uploaded image URI. Object selectors are `--sheet-id`, `--chart-id`, `--pivot-table-id`, `--float-image-id`, `--rule-id`, `--view-id` and `--group-id`; do not invent a uniform flag.

## Preserve and verify edits

Change only the requested cells or columns. Preserve unrelated values, formulas, sheet names/order/visibility, merges, structures and formatting. Put intermediate results to the right of the source or on a new empty sheet. Do not delete, rename, hide or move existing sheets unless explicitly requested and their impact is confirmed. Fill-in tasks write only empty cells; leave existing suspicious values intact and report them. Preserve existing number display formats. If existing display has lost decimals or become unreadable scientific notation, fix the display without changing the underlying value. Give new calculated columns and summary rows explicit semantic number formats consistent with their neighbors: appropriate decimals, percentages or thousands separators.

For editing tasks, deliver actual online changes and read them back with `+csv-get`, `+cells-get` or object get/list commands. An `ok` response acknowledges a request; it does not prove the intended result. Read back formula fields with `+cells-get --include formula`, not only calculated displays. Check sorting/filtering at the beginning and clearing/deletion at its actual target. Remove display annotations such as `49.6(V-Align: bottom)` and residual quote wrappers before reusing displayed text as a pure value.

Before batch completion, correction or filling, find the real final data row. Do not stop at the initial sample. Derived totals, ratios, growth, extraction and lookups should use formulas so they recalculate. Extracted text must be exact, contiguous source text from the same row and column, including all matching fragments. Semantic judgments that cannot be expressed reliably as formulas use per-row static values; do not force fixed offsets or generic regular expressions. Guard empty formula inputs explicitly because empty cells can calculate as zero without an error. Aggregation bounds must exclude headers and include the last record; verify both ends. Read `references/lark-sheets-formula-translation.md` before generating/translating formulas, and use `+formula-verify` after writing. Wrap date, lookup and numeric conversions in appropriate `IFERROR`; check the first and last five results for error codes and avoid circular references. After three failed attempts at the same formula approach, a static fallback must disclose its failure reason and lack of recalculation.

Appending, copying or extending must inherit fonts, sizes, colors, borders on all four sides, alignment, alternating fills, row heights, column widths and merges. Prefer `+dim-insert --inherit-style before|after` for new rows/columns; this chooses the neighboring style source, not insertion direction. Row height is not inherited: read the neighboring height and set `+rows-resize`, particularly for long text. For existing blank or nonadjacent destinations, read source styles with `+cells-get --include style` and write them with the values. Extend merged cross-column titles and reproduce row merge segments only after verifying group boundaries.

Use `+styles-put` for a finishing combination of styles, merges, sizes and freezing. Use a command's plural range/map form for repeated operations of one kind. Reserve `+batch-update` for ordered chains of different operation types, such as insert → header → data; it requires high-risk approval. Prefer native pivots for grouping, native charts for visualization, native sorting/filtering and conditional formatting over static local approximations. Sorting by rewriting cells can retain old values at empty destinations and lose styles.

Turn requested outcomes into independently verifiable checks, including boundaries and styling. Assert expected record counts before full processing, and sample front/middle/end. Search → replace → search each old value because one replacement can leave tail matches. If completion is impossible, deliver an openable main artifact and explicitly identify the unfinished items. Every claimed report, chart, comparison or format must exist at a concrete sheet/object/location; text claims cannot substitute for content. Report known errors individually.

Give added columns distinct headers. Preserve specified sheet names, titles, notes and legend text verbatim, including modifiers and bilingual labels. Scan whole date columns to determine month/day positions before converting. Charts need a title, axis labels and legend; long text needs wrapping and adequate width. Put units, source and measurement basis in separate subtitle space or field names, never over existing header/data cells. Disclose the basis, units and uncertainty of external parameters; label unverified estimates and never invent sources. Missing source facts remain blank with the established missing-value notation or an explicit unknown/pending note.

## Choose the native operation

Read the corresponding reference before executing; help/schema owns exact current flags. If a skill read was truncated, read the missing relevant portion rather than assuming a capability does not exist.

| Need | Command / decision | Reference under `references/` |
| --- | --- | --- |
| Displayed values or CSV | `+csv-get`; omit range for the whole sheet, or use bounded windows | `lark-sheets-read-data.md` |
| Typed column data / DataFrame round trip | `+table-get`; returns columns, data, dtypes, formats and ranges | `lark-sheets-read-data.md` |
| Exact cells, formulas, styles, comments or validation | `+cells-get --include value,formula,style,comment,data_validation`; `truncation` and `conditional_format` are optional categories | `lark-sheets-read-data.md` |
| Plain text CSV without type-sensitive labels | `+csv-put --start-cell <anchor>`; range alias uses its top-left cell | `lark-sheets-write-cells.md` |
| Typed quantities, dates or preserved text labels in an existing workbook | `+table-put --sheets` with columns/data/dtypes/formats; optional `--styles`; absent sheet names create sheets | `lark-sheets-write-cells.md` |
| Create a workbook with typed data | `+workbook-create --sheets`, optionally `--styles`; same typed protocol | `lark-sheets-workbook.md` |
| Precise rectangular values, formulas, rich text, comments or cell images | `+cells-set --range ... --cells ...`; `--cells` is always a 2-D array; use `--writes` for multiple regions/sheets | `lark-sheets-write-cells.md` |
| Change styles without values/formulas | Small single region: `+cells-set-style`; combined/multiple regions: `+styles-put` | `lark-sheets-write-cells.md`, `lark-sheets-styles-put.md` |
| Workbook/sheet metadata | `+workbook-info`, `+sheet-info` | `lark-sheets-workbook.md`, `lark-sheets-sheet-structure.md` |
| Insert/delete/hide/freeze/group dimensions | Native dimension commands; grouping/outline only for explicit row/column outline requests | `lark-sheets-sheet-structure.md` |
| Resize | `+cols-resize` / `+rows-resize`, or sizes within `+styles-put` | `lark-sheets-range-operations.md` |
| Native charts | `+chart-create-basic`, `+batch-chart-create`, `+chart-data-update`, `+chart-config-update`; advanced fields use `+chart-create` / `+chart-update` with minimal properties | `lark-sheets-chart.md` |
| Grouped aggregation / cross-tabulation | `+pivot-create`; no target defaults to a new sheet without overwriting | `lark-sheets-pivot-table.md` |
| Sort complete rows | `+range-sort` keeps native values/styles/empties together | `lark-sheets-range-operations.md` |
| Filter visible rows | `+filter-create`; materialized selected columns belong on a separate result sheet, preserving the source | `lark-sheets-filter.md` |
| Multiple saved filter states | `+filter-view-create`; view-local hiding does not change collaborators' views | `lark-sheets-filter-view.md` |
| Find or replace text | `+cells-search --find`, `+cells-replace`; search is not structural inspection or analysis | `lark-sheets-search-replace.md` |
| Conditional highlighting, duplicates, data bars or scales | `+cond-format-create` | `lark-sheets-conditional-format.md` |
| Trends inside cells | `+sparkline-create` and related operations; not the unsupported SPARKLINE formula | `lark-sheets-sparkline.md` |
| Image tied to a record and moving with its row | `+cells-set-image` | `lark-sheets-write-cells.md` |
| Freely positioned decorative image | `+float-image-create` | `lark-sheets-float-image.md` |
| Clear one/many ranges | `+cells-clear` / `+cells-batch-clear --scope content|formats|all`; high-risk approval | `lark-sheets-range-operations.md`, `lark-sheets-batch-update.md` |
| Ordered heterogeneous writes | `+batch-update` | `lark-sheets-batch-update.md` |
| Local xlsx/xls/csv → new online spreadsheet | `+workbook-import --file`; use Drive import only when the desired type is bitable | `lark-sheets-workbook.md` |
| Append sheets to an existing template workbook | Inspect, then `+sheet-copy` template sheets and edit data; without a template use `+sheet-create` then typed data/styles | `lark-sheets-workbook.md` |
| History, asynchronous restoration | History list/revert/status; retain task ID and poll status | `lark-sheets-history.md` |
| Verify changes between revisions | Changeset read; revision gap at most 20 | `lark-sheets-changeset.md` |
| Diagnose formula errors | `+formula-verify`; record `errors_found`/`partial` and decide repairs by task risk | `lark-sheets-formula-verify.md` |
| Old command migration | Use the actual replacement and changed payload/output contract | `lark-sheets-legacy-command-migration.md` |

CSV writing can numericize `12.10` and `001`. Preserve these labels with typed `object` columns. Genuine quantities must remain numeric rather than formatted strings such as `$1,234` or `30.5%`. Creating a typed workbook should not use `--values`, which writes text. `+cells-set --copy-to-range` copies values too and is not a style-only brush. Native chart migration must also delete and verify removal of obsolete floating-image charts. Freezing replaces the whole state, so supply both desired axes together. Numeric dimension positions denote rows, letters denote columns; `--inherit-style` does not choose direction. Read general visual standards before editing and use the formula translation reference for formulas.

## Read completeness, coordinates and local processing

Small, explicitly scoped tasks may use the CLI directly. Optional repository scripts `lark_inspect_workbook.py`, `lark_detect_subtables.py` and `lark_profile_table.py` help inspect workbook structure, region boundaries and fields. They are read-only helpers, not prerequisites or write tools; embedded distributions omit them. Equivalent CLI inspection uses `+workbook-info`, `+sheet-info` and small `+csv-get` windows. Large/ambiguous tables benefit from structure inspection and a profile of a confirmed untruncated range before deciding formulas, pivots, sorting or batch edits.

Before calculations, sorting, filtering or extraction, sample at least 50 rows (or every row of a smaller table) to discover type/date/number/empty variants and handle them. Long numeric identifiers over 15 digits must be compared through exact `+cells-get` values or text, never formatted CSV scientific notation. Typed table reads support DataFrame reconstruction, but must still be checked for completeness and actual extent.

Read `warning_message` before consuming returned data. Check `has_more`, range/sheet/top-level truncation, and saved-output `complete`/`unread_sheets`. `--output-path` saves the data payload and prints a small receipt; it is bounded, not proof of completeness. CSV/cell reads default to 500,000 characters; saving raises the default to 20 million for memory protection. An explicit positive `--max-chars` wins; zero means the normal default, not unlimited. Use smaller ranges or a sufficient explicit cap after truncation. Large data should go to files and local processing, not an unbounded conversation dump.

Native cell reads report `actual_range`, `row_indices` and `col_indices`: use these coordinates, never array offsets. CSV's `annotated_csv` is the data source; `[row=N]` denotes the real row and `col_indices` the real columns. Parse logical RFC 4180 records, including quoted embedded newlines. Never count commas or infer row numbers from ID/sequence values. Hidden rows/columns are included by default; `--skip-hidden` can introduce gaps. Use sheet metadata to identify hidden intervals and preserve the returned row/column segments when writing.

Physical `row_count` is an upper bound, often larger than the data. `current_region` is a contiguous nonempty block that can stop at an internal blank row or include titles, totals and notes. Neither proves the data endpoint. Identify the header/start; scan beyond internal blanks up to a justified endpoint; inspect the tail and exclude summaries, signatures and notes. Inspect all column headers, and full row-label columns for cross-tab layouts. Verify boundaries before writes, filters, sorts and aggregations.

Profiles are heuristic. Check `summary`, field map, warnings, visibility, data row/column segments, special rows and safe append hints. Mixed types, long numeric IDs and formula errors require native/typed reads. Ambiguous/missing headers, many empty cells and special rows need targeted reads; multi-row headers may need merge metadata. Hidden intervals and row/column gaps require explicit segments. An append hint is a candidate: inspect that destination for values, formulas, styles, hidden columns and object dependencies. Helper truncation errors require smaller windows and cross-window boundary verification. Consult the read-data reference for helper flags, limits and warning-specific actions.

Separate stdout JSON from stderr diagnostics; never merge them when parsing. Read an error before changing a command, and obtain an actual attempted-call error before declaring a supported scenario unavailable. For NLP translation/classification/rewrite, use semantic processing in batches (usually 30 rows, at most roughly 300 generated rows per batch), preserving row mappings and writing each batch promptly. Scripts may coordinate batches and writes but cannot replace semantic judgment.

## Resources, inputs and risk

Choose exactly one `--url` or `--spreadsheet-token`. URL routing uses `/sheets/`, `/spreadsheets/` or `/wiki/` paths, including non-Feishu domains; wiki URLs resolve their underlying spreadsheet. Commands acting on a sheet also require exactly one `--sheet-id` or `--sheet-name`; a sheet prefix in a range does not replace this selector. Use a known sheet name, or discover it with workbook metadata; never guess `Sheet1`. Quote A1 references containing `!` appropriately for the current shell.

Workbook create/import have no existing-resource selector. Workbook-level info/export, batch/style operations and sheet creation do not take sheet selectors; pivot creation has its own optional mutually exclusive target sheet selectors. Read the command's reference for exceptions such as dropdown update/delete and batch clearing.

Use `--print-schema --flag-name <name>` for unfamiliar composite JSON, optionally a dotted subtree; it is local and requires no other mandatory inputs. Chart `--print-example <type>` supplies a minimal supported template. Simple scalar arrays and plain CSV are not composite schemas. Paths and `@file` inputs must be relative to the working directory. Prefer stdin (`-`) for large/quoted/multiline payloads where supported, one stdin flag per invocation. Use UTF-8 without BOM. Follow the host's temporary-file placement policy; otherwise keep temporary payloads outside the user's project. On PowerShell/cmd use supported file/stdin forms instead of copying Bash heredocs or guessing escapes. Do not use `set +H` as a portable quoting fix.

High-risk commands include batch update, cell clearing/batch clearing, sheet/dimension/dropdown deletion, and chart/pivot/conditional-format/filter/filter-view/sparkline/floating-image deletion. Preview with `--dry-run`, show the action and affected scope, and obtain explicit user agreement before appending `--yes` to the original command. Exit 10 is the approval gate; never silently add confirmation and retry. Writes do not automatically read back; perform the appropriate verification yourself. Preserve failed/partial evidence and inspect current state before retrying.
