# Base process coverage

This suite executes the production CLI with the `authsidecar` build against an
isolated, local HTTP fixture. It is offline evidence, not a live-service
acceptance claim. The frozen service catalog has no generated `base` service;
Base capabilities are registered shortcuts plus the generic raw API surface.

`tests/next_sections/base/process_test.go` uses
`nextfixture.RunMain(m, "../../..")` and `nextfixture.NewFixture`. Each CLI
process has a temporary config and working directory, a synthetic current user
and tenant, and a local sidecar. The outer `scripts/next_test.py` isolation clears
personal credentials/configuration and sets a dead external proxy. The fixture
never forwards target headers to Feishu. Tests do not authenticate with or write
to a live tenant.

The source-backed scenarios independently specify ordered business requests,
complete JSON request bodies, HTTP methods, resource paths, and complete query
parameters. They inspect concrete response identities and content, process exit
codes, stdout/stderr placement, local files, and request counts. Expectations do
not invoke the command's dry-run implementation. A registration guard prevents a
new shortcut from silently lacking a named process scenario; this guard alone
is not functional evidence.

## Family evidence

| Family | Executing process coverage |
| --- | --- |
| Addressing | Local Base URL, Wiki-to-Base resolution, title search with highlight removal; raw API preserves unmodeled fields and large integers |
| Base and blocks | Get/create/copy; explicit first-table schema replacement; bot creation grants only the current synthetic user with notifications disabled; list with client-side type filtering; folder create, move to root, rename, delete |
| Tables | Offset/limit list; get plus field/view reads; create explicit schema and a view; update/delete; synchronous schema copy and asynchronous status recovery |
| Fields | List/get/create/update/delete/options; extension get/clear/row update; partial batch ledger identifies created, failed, and unattempted fields |
| Views | List/get/create/delete/rename; all six filter, visible-fields, group, sort, timebar, and card get/set pairs |
| Records | List/search/get native matrix JSON and default NDJSON artifacts; top-level field-map create/update; native create_records/update_records/batch-delete requests; share links; history |
| Artifacts | NDJSON exact large integer, Unicode/newline, null, empty-string/array preservation; 500-row first-page export with visible continuation; explicit next-offset export keeps each page’s schema and artifact separate; existing artifact remains unchanged on clobber rejection |
| No-op | Single and batch update compare typed matrix cells and avoid writes when unchanged; create with skip-unchanged fails before requests |
| Attachments | Real multipart upload, its explicit telemetry report, append canonical field identity, stream download with byte-for-byte verification and extra metadata, remove selected attachment |
| Forms | Create/get/update/delete; auto-paginated list; question create/update/list/delete preserving false/null and keep-field; share-token detail/submit; get/update sharing with explicit false |
| Permissions | Role create/list/get/update/delete; double-encoded role data; advanced permissions enable/disable; no live permission changes |
| Automation | Workflow list/get/create/update/enable/disable; automatic cursor pagination; button bind/get/unbind resolves the field name to its canonical ID; data query |
| Templates | Categories, category list, search, cursor and limit forwarding |
| Dashboards | List/get/create/update/delete/arrange; share settings; all block CRUD/list/computed-data operations, Markdown content and complete numeric position |
| BaseApp/workspaces | Workspace create/entity list/move; app create/get; page lifecycle and uniqueness reads; block list/get/create/update/computed-data, current-type read before data-config update |
| Import | `import_e2e_test.go` is owned by the Base implementation owner and verifies bound artifact import, including 10,000 rows. Its evidence is reported with that implementation. |

The original unit suite and deterministic dry-run regressions remain necessary:
these cover additional field/config types, semantic validation, aliases, layout
precision, multipart size thresholds, polling clocks, and compatibility input
shapes. This process suite samples executable family workflows and pins every
registered command; it does not imply every option/type combination or server
behavior has been live-tested.

## Failure and recovery contracts

Validation tests establish zero business requests for malformed JSON, duplicate
record IDs, conflicting projections, invalid limits, empty schema, partial
position, conflicting share updates, invalid workflow identity, and missing
confirmation. Read failures preserve typed authorization, not-found, and malformed
response errors. Mutation failures preserve cause metadata and a durable unknown
outcome without retry. Field batches retain the complete partial-success ledger.

Record batch create is confirmed only when the shared receipt classifier proves
that returned unique record IDs match the submitted count with no ignored fields.
Batch update and delete remain unknown at the generic receipt boundary, preserve
the backend result, and exit 22 rather than infer complete application. Unchanged
comparison reads have no write receipt. No-op comparison is not atomic with a
later changed write; concurrent edits can still be overwritten.

An existing NDJSON destination is rejected after the bounded read in the current
implementation; the test verifies the destination is not modified. It does not
claim the read is skipped.

## Live selection and blockers

`TestBaseLivePrivateLifecycle` is selectable and explicitly skipped as **BLOCKED**.
The section runner's `--suite live --allow-live` must likewise report blocked
until an authorized fixture can prove all of these prerequisites:

- Effective account, identity, and tenant are independently bound to the run.
- The destination and creation operation guarantee visibility only to the current
  account before creation, including inherited workspace, folder, group, link,
  and organization access. Restricting access after creation is insufficient.
- Any intentional other-user access resolves Ying Wu unambiguously by stable
  identity; no group, notification, workflow, public/link-wide, or unrelated-user
  delivery occurs.
- Every object is labeled and owned by the run, cleanup uses only recorded
  run-created IDs, and cleanup is available even after a failed step.

With those prerequisites verified, the pending live scenario is: create a
private Base, create a table with an explicit schema, create/read/update/export
synthetic records, verify supported native values and identity, remove only the
run-created records/table/Base, and verify cleanup. Compose forms, dashboards,
roles, BaseApp/workspace and attachment lifecycles only where their private
creation and cleanup can independently be established. Sharing/workflow cases
whose exposure cannot fit the authorization boundary stay blocked while offline
coverage continues. A skipped test is never passing live evidence. Legacy live
workflow tests are not safe substitutes merely because credentials exist.

Live effects, verified private-at-creation behavior, complete server type
semantics, comparative agent tasks/token measurements, and full P2 section
acceptance remain pending. Do not mark the Base section complete from this
process package alone.

## Recorded offline validation

On 2026-09-07 the assembled process package passed with **168 passing test
events, zero failures, and one explicitly blocked live skip**. The command was
`go test -json ./tests/next_sections/base -count=1 -timeout=10m`, launched through
`scripts/next_test.py`'s `isolated_env` and `run_command` helpers. Sanitized raw
command output is retained in `.tmp/base-process-final.json`. This run includes
the Base owner's import and native-field-schema process files alongside this
suite. The later `TestRecordUpsertIgnoredFieldsKeepsPartialEvidence` regression
was added after this run and is covered by the coordinator's follow-up gate.
`gofmt` and the focused whitespace check also passed. The complete
section unit/dry-run gates and repository-wide checks are reported separately by
the section coordinator; this result does not replace them.
