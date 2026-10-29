# Live collaboration evaluation

[tasks.live-collaboration.json](tasks.live-collaboration.json) adds five
permission-bounded workflows to the [real Feishu pipeline](PIPELINE.md).
The controller is [live_collaboration.py](live_collaboration.py); it shares
account binding, private ACL checks and mutation recovery with the business
fixtures. No synthetic API backend or held-out task is used.

| Case | Independent outcome verification |
|---|---|
| `collaboration-document-create` | Create a new private TEST document from supplied complete XML. Verify title, native heading, bold hyperlink, Unicode, leading-zero codes and 3-by-2 table. |
| `collaboration-image-upload` | Append two different PNGs with identical basenames to a private document. Verify native image order, display dimensions, exact downloaded bytes and preservation of every prior block. |
| `collaboration-file-upload` | Append multiline Unicode CSV and binary bytes containing NULs as native document attachments. Verify filenames, view/file parent-child links, order, downloaded bytes and prior blocks. |
| `collaboration-mermaid-create` | Append a native whiteboard from supplied Mermaid. Verify exact stored source, three participants, four message connectors and a conditional fragment. |
| `collaboration-drawing-create` | Append a native whiteboard from supplied SVG. Verify editable circle/rounded rectangle and text, geometry, colors, labels and connector endpoints; inspect the diamond's vector fallback. |

Document attachments satisfy the upload cases. These cases do not claim Drive
folder-placement verification, IM resource upload, or delivery to anyone.
The drawing checks native nodes and their geometry; it is not a subjective
visual-design score. Feishu converts SVG into nodes instead of retaining an
original SVG syntax block. Mermaid input has no terminal newline because the
server removes it during import. File attachments have native view containers;
those containers and their child relationships are checked rather than ignored.

## Private fixtures and scope

Read [all testing authorization boundaries](../docs/testing.md#authorization-boundaries-for-all-testing).
Existing user scopes only: no login/consent flow, group creation, invitations
or permission changes. IM/group-chat evaluation, message seeding and delivery
are excluded; their cases and implementation have been removed. All new
top-level resources are named `[TEST]` and use current-account private ACLs.

The five document cases use the same owner-only `private_parent` configuration
as [the business driver](LIVE_BUSINESS.md). A creation result is eligible for
cleanup only after independent account/owner/ACL verification, exact unique
run title and native creation time. `creation.json` is the required immediate
creation journal; a valid returned answer is a recovery fallback. Unknown
creation outcomes remain blocked for reconciliation, not automatically retried.
The document itself is checked; destination-folder placement is not scored.

Docs and attachments/boards are cleaned by their registered document IDs,
then the test folders. Unknown outcomes retain all recovery state. Never replay
an uncertain mutation. Sealed historical reports and controller snapshots retain
the earlier blocked IM attempts for audit; they are not active evaluation cases.

## Running

Use three repeats per arm with the same frozen production binaries, guidance,
model, reasoning and budget. Point `live_manifests.paired` at this manifest and
`live.driver` at `live_collaboration.py` with the same `--cli`, `--binding` and
`--resources` arguments as the business driver. Use a fresh experiment/output
directory and freeze the evaluator before billing models.

`--section collaboration` selects the five document cases (30 planned attempts
at three repeats). No IM or group-chat section is registered. Keep this workflow
mix separate from prior comparison aggregates.

Run `make evaluation-test` for offline verifier tests. Calibration against real
Feishu is controller validation, not task success or token-efficiency evidence.
Record all calibration failures and model attempts, including their full costs.
