# Private Sheets lifecycle and native formula fidelity

This is a narrow synthetic live lifecycle, not acceptance of the full Sheets capability inventory. The remaining six full capability lifecycles in `tests/next_sections/sheets/errors_test.go` remain explicitly unverified.

## Authorization and executable test

`TestSheets_PrivateUserLifecycle` in `tests/cli_e2e/sheets/sheets_private_live_test.go` replaces the unconditional placeholder for basic private workbook creation, use, and cleanup. Without explicit opt-in it skips before accessing credentials or making requests. Once opted in, failed identity, ownership, privacy, data, or cleanup checks fail the test.

Before enabling the test, the operator must independently verify the effective account/tenant, the account's new-document default, and an isolated private parent folder. The 2026-09-08 run used a personal folder created after the UI showed **Only invited members can access**, with no invitees. Independent Drive preflight established closed link access, exactly one current-account full-access container member, and matching metadata owner. Fresh CLI identity was compared in memory with the independent Contact account/tenant digest before each attempt; private IDs were not written into this document.

The test repeats current `user_info` and parent ACL/ownership reads before creating one workbook. It registers cleanup immediately from the returned spreadsheet token, checks the new workbook's own ACL and owner, and only then writes six synthetic cells. Every CLI process explicitly uses `--as user` and the existing E2E runner with `Attempts: 1`. No create, write, or delete is replayed. Cleanup acts only on that run's returned spreadsheet token; it does not delete the supplied parent folder or enumerate resources by name.

An operator who has completed that preflight supplies these variables through the environment, without printing their values:

- `FEISHU_NEXT_SHEETS_PRIVATE_LIVE=1`
- `FEISHU_NEXT_PRIVATE_LIVE_FOLDER_TOKEN`
- `FEISHU_NEXT_PRIVATE_LIVE_OPEN_ID`
- `FEISHU_NEXT_PRIVATE_LIVE_TENANT_KEY`
- `FEISHU_NEXT_SHEETS_LIVE_EVIDENCE_DIR` — a new private directory; the test creates it with mode 0700
- `LARK_CLI_BIN` — the absolute path of the canonical binary under test

Run only the selected opt-in test:

```sh
go test ./tests/cli_e2e/sheets -run '^TestSheets_PrivateUserLifecycle$' -count=1 -v
```

The test saves private per-command output with mode 0600 and a bound native cell artifact outside test stdout. The sample covers Unicode/newline text, empty text, leading-zero text with a text number format, numeric zero, boolean false, and an explicit `=1+1` formula with numeric cache. Raw number, boolean, string, formula, range, and verified artifact binding assertions are retained.

## Preserved initial evidence

The first three attempts used the isolated canonical binary built from `e07178d`. Every attempt created one workbook, wrote once, and deleted that exact workbook once. No other resource was edited or deleted. The original failed test logs are retained under ignored `.tmp/next-live-20260908/`; none is presented as a passing run.

| Attempt | Original test result | Diagnosis |
| --- | --- | --- |
| 1 | Failed | The test incorrectly expected raw JSON types from formula rendering. The native formula response rendered zero/false as strings and an empty cell as `{}`. Its cleanup check also did not recognize an explicit native deleted response. |
| 2 | Failed | The test passed an artifact path outside the CLI working directory. FileIO rejected it before a native export call. The test now uses a relative artifact path within its private working directory. |
| 3 | Failed | A test expectation typo shortened the Unicode payload. Independent inspection of the saved raw artifact additionally exposed a real formula fidelity defect: raw export kept the formula cache but omitted the formula field. |

The third run's native artifact SHA-256 is `c7a14a637eba98ecf61c1f9e6ffad0760b7351db30e37291daae17f3d8226b77`; the independent formula read evidence SHA-256 is `0a60e9161803eb97f36f9d01095041ba5d89b3a4899e3df0ab4d5c89f6a72276`. The artifact preserves raw zero, false, empty string, Unicode/newline text, and leading-zero text. Its sixth cell contains numeric `value: 2` and styles, but no formula. The formula-rendered read of that same range contains explicit `formula: "=1+1"` and `value: 2` at that coordinate. **Raw empty text is preserved**; the empty/omitted collapse observed in the first attempt was a rendering distinction, not proof of loss in the raw artifact.

Cleanup's exact-ID generated `sheets spreadsheets get` returned upstream code `1310249` and `Path param :spreadsheet_token is deleted`. That explicit native deletion proof is accepted alongside typed `not_found`; a successful read, cleared metadata URL alone, or an unrelated error leaves cleanup unverified. The initial test's narrower cleanup failure remains recorded.

## Fix contract and remaining limits

The existing source exposes two relevant native `get_cell_ranges` render modes: `raw_value` in `lark_sheet_table_io.go`, and `formula` in `lark_sheet_read_data.go`. There is no sourced single-mode `include_formulas` field. `lark_sheet_revision_get.go` documents a revision on every native read/write response; the retained formula read reports revision 1.

The native export/import reader now requests both renderings for each bounded page. It preserves the raw cell object and adds only an explicit formula field from formula rendering. Display strings never replace raw numbers, booleans, empty strings, nulls, omitted values, rich content, or styles. A literal value beginning `=` does not become a formula by inference.

Both responses must prove the complete requested rectangle and contain an explicit nonnegative integer revision. Every page and rendering must have the same revision. Malformed formula fields, conflicting formula/cache evidence, missing revisions, incomplete coordinates, and revision changes fail before an artifact or import write is produced. Revision equality is optimistic backend evidence; it is not a lock or a general snapshot-isolation guarantee. No automatic write retry is introduced.

Artifacts already saved by the earlier raw-only exporter may have lost formula identity while retaining the same `sheets-data/v1` schema. The revised importer cannot recover missing provenance from a cached number. Re-export potentially affected source ranges before reuse; do not infer formulas from cached values or retrofit old artifacts automatically.

Each page now makes two native reads. All previous performance measurements remain evidence for their historical binary and fixture conditions; they are not measurements of this corrected product condition. No model evaluation was rerun for this fix.

## Offline checkpoint before the final live confirmation

The complete Sheets unit package and scoped vet passed. The isolated E2E report `.tmp/sheets/paired-e2e-report.json` records 68 inherited and 300 process test passes, with no failures. That run compiled before the replaced basic-live placeholder was removed, so it retains seven historical skip events. A focused current-source run records the remaining six blocked capability lifecycles, one parent-test pass, and no failures in `.tmp/sheets/paired-focused-report.json`; its initial temporary-directory-prefix setup rejection is retained separately and ran no test/backend calls.

The revised cross-domain integration suite passed 23 test/subtest events with no failures or skips. Its 10,000-row supported primitive handoff uses 32 native reads (16 pages in each rendering), eight destination schema reads, and 50 Base batches. The [cross-domain evidence](../integration/sheets-base.md) separates this condition from earlier raw-only and empty-text assumptions. These offline results do not substitute for the final private live confirmation, which follows an immutable code review checkpoint.

## Final private confirmation

After independent review approved immutable product checkpoint `02b6c5f` (review commit `88b8c7e`), the single final opt-in lifecycle **passed** on 2026-09-08: one test, zero failures/skips, 15.25 seconds (package elapsed 24.626 seconds). Fresh account/tenant and parent/child privacy checks ran again. One workbook was created, one six-cell write was dispatched, and that exact returned workbook was deleted once. The supplied parent folder was not changed.

The corrected bound artifact contains exact Unicode/newline text, empty string, leading-zero text, numeric zero, boolean false, and explicit `formula: "=1+1"` alongside numeric cache `2`. It is 1,035 bytes with SHA-256 `d6d04518a0292a445b940ce788fe37ddf4afa34aa0287a531dff85dc550e9067`. Verified account/tenant binding and complete `A1:F1` extent passed. The independent exact-ID cleanup read confirmed upstream `1310249` (deleted); the CLI's `unknown` subtype was not mistaken for an uncertain resource state when this explicit native proof was present.

The [sanitized evidence manifest](../../evidence/sections/sheets-private-live-evidence.json) records binary/source/test/report hashes and the private artifact locations. Binary metadata is `02b6c5f-dirty`: the owned runtime and test files were committed and clean; the dirty suffix came from another owner's tracked Base test edit during the canonical build. No product code changed after the reviewed checkpoint. The original three failed attempts remain failed and are not replaced by this final success. Six broader capability lifecycles remain unverified, and no model evaluation was run.
