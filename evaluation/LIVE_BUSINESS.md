# Live business workflows

[tasks.live-business.json](tasks.live-business.json) contains four development
workflows. [tasks.live-business-extended.json](tasks.live-business-extended.json)
adds six further workflows. [tasks.live-business-complex.json](tasks.live-business-complex.json)
adds three harder reconciliation workflows, for thirteen reusable business cases. They seed fictitious
business data into **real Feishu resources**;
there is no API simulator, mock endpoint or canned task result. Each attempt
gets fresh resources and independently reads the resulting native state. Setup,
verification and cleanup are controller work, not agent success measurements.

| Workflow | Work and independently checked outcome |
|---|---|
| `business-cross-source-report` | Read typed spreadsheet cells and all 205 Base records, aggregate filtered totals, write a JSON report and append one native document paragraph; preserve every source cell/record and all previous native blocks/styles/links. |
| `business-document-edit` | Replace a release status while preserving its bold link, native block IDs, Unicode and all unrelated content. |
| `business-sheet-reconcile` | Set one numeric value to zero and one text value to empty, preserving leading-zero codes, booleans, formulas, styles and guard cells. Formula results may recalculate. |
| `business-base-reconcile` | Append two intake records containing zero, false, null and Unicode. Read every page to establish that all 205 original rows remain unchanged. |

The extended suite adds:

| Workflow | Additional outcome coverage |
|---|---|
| `business-base-audit-large` | Audit all 2,505 invoices, crossing the CLI's 2,000-record page boundary; distinguish checked positive, zero, negative and unchecked rows, and preserve the full table. |
| `business-base-conditional-close` | Update every qualifying record's checkbox and note while preserving amounts, titles, unselected rows and record IDs. |
| `business-sheet-intake` | Apply a quoted multiline Unicode CSV by exact leading-zero code and existing boolean approval; skip unapproved rows and preserve styles/formulas. |
| `business-sheet-export` | Export every raw JSON cell type and formula, including computed values, zero, false, empty strings and Unicode; preserve the live workbook. |
| `business-document-audit-large` | Audit more than 500 native blocks, including the last continuation page; filter releases/statuses and exclude zero/negative amounts without changing native formatting or IDs. |
| `business-budget-close` | Save a selection from the original Base, close its records, update the Sheet's numeric total and note, and append one native document settlement paragraph while preserving all unrelated state. |

Use a separate experiment/output directory for the extended suite. Point
`live_manifests.paired` to its manifest, retain the same frozen binaries and
model settings, and use at least three repeats per arm. The new workload mix
must not be pooled with the earlier four-case sample as an efficiency claim.
Large fixture setup is not agent execution or model usage.

The complex suite adds a shuffled, versioned multi-currency journal with exact
per-event decimal rounding, duplicate/void handling and existing approval gates;
a 205-row Base reconciliation with request revisions, original-value conflicts,
conditional writes and null/zero/negative values; and a closeout that updates
every eligible invoice across more than 500 native document blocks, preserves
their IDs and rich formatting, and appends a computed audit summary. Every case
verifies a local report as well as all relevant remote data and preserved state.
Use the same driver with the complex manifest in a new experiment directory.
In edited closeout paragraphs, adjacent text runs may be split differently by
Feishu only when all metadata except text content is identical. The verifier
coalesces those runs while preserving exact text, style/link metadata and native
block IDs. Unrelated blocks and root order remain exact comparisons. A report
that includes zero-amount invoices fails even when its total happens to match.

The workflows require the existing granular Docs, Sheets, Base, My Space
metadata, folder-create and document-delete permissions.
They do not create or update Tasks, request calendar/mail scopes, enumerate
unrelated Drive folders, deliver messages or add collaborators. Missing scopes
block a workflow rather than triggering login or being counted as a failure of
the model. Never expand authorization automatically to make a test run.

All top-level resource names start with `[TEST]`.
The Base table name starts with `TEST`, because Feishu forbids square brackets
in table names; its containing Base and folder retain `[TEST]`. Content is
fictitious and includes deliberate leading-zero, empty-string, zero, false,
null, Unicode, rich-text and pagination cases.

## Private setup

Read the [testing authorization boundaries](../docs/testing.md#authorization-boundaries-for-all-testing).
The current user must authorize creating these test resources. Provision a fresh
`[TEST]` parent with that identity, record its token immediately, and verify its
metadata owner, closed link sharing and sole current-account membership before
seeding content. The driver repeats those checks before creating children and
checks each child's ACL before writing task data. Do not use an unrelated or
shared production folder, or broaden permissions to make a fixture work.

Keep an owner-only resources file outside version control:

```json
{
  "private_parent": {
    "folder_token": "<run-owned TEST parent>",
    "creation_authorized_by_user": true,
    "label": "[TEST]",
    "account_binding": "<digest from the independent account binding file>"
  }
}
```

The authorization flag records the user's actual authorization; it is not proof
of identity or ACL. The driver independently verifies both. The account binding
uses the Contact driver's format described in [PIPELINE.md](PIPELINE.md).
Credentials stay in the existing protected CLI configuration and data directories.

Configure the existing pipeline with `suites: ["paired"]`,
`workloads: ["workflow"]`, and `live_manifests.paired` pointing to the absolute
business manifest path. Its `live.driver` is:

```json
[
  "/absolute/python3", "/absolute/source/evaluation/live_workspace.py",
  "--cli", "/absolute/frozen-official-cli",
  "--binding", "/absolute/private/account-binding.json",
  "--resources", "/absolute/private/resources.json"
]
```

The same frozen controller is used for both arms. Keep the model, reasoning,
tool access, budget and guidance delivery identical. Use at least two repeats.
Public schemas and user goals are shipped; account IDs, remote tokens, binding
proofs, controller state and model artifacts stay in protected local outputs.

## Verification and recovery

The controller reads native blocks, raw/formula cells and complete Base pages.
It compares task outcomes, exact JSON types and preservation conditions, rather
than trusting the model's answer. Unknown or nonadvancing pagination fails
verification. Source changes cannot be hidden by an otherwise correct summary.

Every controller mutation is journaled **before** dispatch, with returned IDs
persisted immediately. Uncertain outcomes leave `pending` set and block cleanup
and replay. Reconcile those exact operations first; never clear a marker merely
to rerun a failed setup. Validation or explicit authorization rejections can be
recorded as confirmed unapplied after inspecting their protected diagnostics.

Cleanup uses the exact run-owned IDs journaled by creation or validated output,
verifies owner/ACL, and requires the delete command's confirmed server outcome.
It does not require broad Drive-list permissions. The fresh run-owned folder is
deleted after its known children; the controller never traverses unrelated
folders. Unknown model-created files outside that folder cannot be claimed
cleaned. Keep any unresolved state and report the limitation.

Remove the run's parent only after every attempt and calibration fixture is
confirmed cleaned. Do not publish raw outputs or real tenant identifiers.

Offline tests in `tests/next_evaluation/test_live_business.py` check preservation,
type distinctions, pagination, private binding, file paths and mutation recovery.
Run `make evaluation-test` after changing the driver. These mocks are tooling
tests, not live task or token-efficiency evidence.

A previously attempted file-handoff case is excluded: its byte/filename/owner
checks could not independently establish destination-folder placement with the
granted metadata scope. Its recorded measurements must remain labeled partial;
they do not establish full-task success or comparative tokens per completion.
