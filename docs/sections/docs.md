# Docs capability section

Status: offline section suite passed; required backend-dependent work remains open. Live backend guarantees
and comparative development evaluation remain open; neither is inferred from mocks.
Owner: Docs (`shortcuts/doc`, `affordance/docs.md`, `tests/cli_e2e/docs`,
`tests/next_sections/docs`). Process test author is the bounded Docs E2E agent;
section integration and all acceptance evidence belong to the Docs owner.

## Reference and complete mapping

Inspected the unchanged sibling at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`,
its registrations, tests and content implementation, and the frozen
`baseline/{coverage.json,registrations.json.gz,sources.json,meta_data.json.gz}`.
The frozen `docs` tree contains its domain node and **16 leaf shortcuts**. Fifteen
are returned by `shortcuts/doc/shortcuts.go`; `docs +whiteboard-update` is registered
by the Whiteboard section. There are no generated methods under the `docs` domain
in this frozen registration. Generated Docx/Drive APIs and raw API access remain
separate shared/resource surfaces, not substitutes for the workflows below.

All commands keep their `feishu-cli-next docs +…` identity, risk, scope, identity,
FileIO, stdin/file flags, compatibility flags, output and dry-run contracts unless
an intentional correction is listed below.

| Capability | Complete behavior retained | Test owners and acceptance |
| --- | --- | --- |
| `+create` | XML/Markdown, escaped explicit title, folder/wiki parent or parent position, native reference maps, user/bot, post-create current-user access attempt for bot, local image/file placeholders and binding | `docs_create*`, `local_doc_resources*`, `html5_block_resources*`; process XML/sidecar and errors; live private creation/permission semantics blocked |
| `+fetch` | XML/Markdown/IM Markdown; simple/with-ids/full; language, revision, full/outline/section/range/keyword scopes, sibling context, depth and share anchors; comments and HTML5 sidecars with local files; explicit downgrade warnings | `docs_fetch*`, `html5_block_resources*`; process every scope, full content artifact and continuation preservation; live native block/comment identity blocked |
| `+update` | All eight operations: `str_replace`, `block_delete`, `block_insert_after`, `block_copy_insert_after`, `block_replace`, `block_move_after`, `overwrite`, `append`; inclusive sibling ranges, sentinel IDs, base revision, empty string deletion for str_replace, XML/Markdown/reference maps and local resources | `docs_update*`, `local_doc_resources*`, `docs_result*`; process requests/effects/errors, unknown outcomes, unchanged content; live conditional writes and concurrent editors blocked |
| `+search` | User Search v2 for docs/wiki/sheets; query, doc/wiki-specific filters, date conversion, bounded pages and exact continuation, raw result identity and ISO time additions | `docs_search*`; process filter/pagination, ambiguity retained as multiple results rather than a guessed target |
| `+script` | Local init-draft and parse; unique reserved draft path, Presentation Decision JSON/file/stdin and shell quote recovery, tolerant XML, Unicode profile, required-block/word-count/resource diagnostics, remote parse | `docs_script*`, `internal/docxparse/*`; process local path/profile and dry-run; local assessment is distinct from command execution success |
| `+history-list` | Native historical version strings, page size 1–20 and continuation; docx/wiki addressing | `docs_history*`; process pagination and validation; live history availability blocked |
| `+history-revert` | Explicit positive history version, bounded wait, task identity | `docs_history*`; process async/unknown result preservation; live observed revert and revision fidelity blocked |
| `+history-revert-status` | Read-only task polling scoped to native doc/task IDs | `docs_history*`; process returned status; no automatic replay after uncertainty |
| `+media-upload` | Existing block-target upload, image/file types, multipart for large media, token binding | `doc_media*`, `local_doc_resources*`; process dry-run and validation; live upload/binding blocked |
| `+media-insert` | Append image/file, existing children address, upload and token binding, typed failure/recovery | `doc_media_insert*`; process full dry-run request plan; live independent downloaded bytes blocked |
| `+media-preview` | Preview image/file token content through source export with FileIO output | `doc_media*`; process dry-run and invalid input; stable live comment-image fixture blocked |
| `+media-download` | Export authorization, media download or whiteboard image, exact saved file and inferred extension, overwrite guard, permission/rate-limit recovery | `doc_media*`; process request plan/failure paths; live native export permissions blocked |
| `+resource-download` | Cover metadata/content, wiki resolution, FileIO/save/overwrite | `doc_resource_cover*`; process dry-run and metadata behavior; live cover bytes blocked |
| `+resource-update` | Cover from file/clipboard/HTTPS, dimensions, offsets, upload/patch | `doc_resource_cover*`, `clipboard*`; process dry-run/ambiguous sources; native clipboard and live cover mutation blocked |
| `+resource-delete` | Explicit cover clear; already-empty zero-write result | `doc_resource_cover*`; process independently counts zero PATCH on empty cover |
| `+whiteboard-update` | Existing whiteboard DSL and document integration | Owned implementation and unit suite: `shortcuts/whiteboard`, verified offline by Whiteboard commit `f355d99`; Docs process discovery/dry-run; live DSL fidelity and cross-domain gate depend on Whiteboard owner |

## Rich content and cross-section contracts

XML remains the native preservation representation; Markdown/IM Markdown are
explicit conversions. Do not infer block identity, styles, comments or editable
metadata from simplified Markdown. The retained parser profiles (without imposing
a new schema) title/headings 1–9, paragraphs, inline formatting/link/cite/date,
lists/checkboxes, table cells/rows/columns, code/LaTeX, grids/callouts, images/files,
HTML5, embedded Sheets/Base, diagrams/whiteboards, synced/readonly blocks, tasks,
OKR, polls/agendas, charts/bookmarks, wiki/folder catalogs and meeting cards.
Unknown XML containers survive parsing; profile counts are not proof of backend
support. XML preparation preserves supported attributes and reference-map groups,
resolves explicit local files through runtime FileIO, and validates resource
correlation before binding uploads. Numeric sidecar values retain decimal digits;
Unicode, leading-zero strings, null, empty values and omitted fields remain distinct.

Markdown owner coordinates conversion; Whiteboard owns the DSL operation; Drive
and Wiki own resolution and permission endpoints; Events owns event projections.
No Docs-specific event registration exists in the frozen event catalog inspected
here. Shared output artifacts are result containers, not Docs import payloads:
pass explicit `document.content` and its `document.reference_map` to authoring,
without treating a projected/partial response as a whole-document replacement.

## Intentional corrections

- Exact native XML overwrite compares a full read (IDs/styles/cite metadata and
  sidecar) in software before mutation. An identical observed body and sidecar
  returns `unchanged:true` and issues no PUT or mutation receipt. Explicit revision
  mismatches, incomplete/truncated results, warnings, or unknown metadata never
  establish a no-op. This comparison does not normalize XML or compare lossy
  Markdown, and does not suppress intentional append/copy/move/clear effects.
  Other block/text replacement forms and changed local-resource content retain
  their original semantics; complete no-op equivalence for those representations
  remains a backend-dependent open requirement. The read-before-write window has
  no additional protection against concurrent external editors.
- Extensible reference maps and search filters decode numbers losslessly rather
  than silently rounding through float64. Existing XML/media transformation remains.
- An explicitly supplied revision zero is forwarded verbatim; revisions below -1
  fail locally. No claim is made that the server enforces conditional writes.
- Full reads reject explicitly supplied bounds which the old request silently
  discarded; use an appropriate scope. Negative read limits fail before dispatch.
- Search validates its JSON filter before dry-run as well as execution. Numeric,
  null and empty time-range fields remain present; recognized nonempty date strings
  still convert to Unix seconds. Existing page-size normalization is retained.

## Evidence commands and outstanding verification

Implementation commit: `f72a408` (`fix(docs): preserve native content and exact input values`).
[Machine-readable evidence](../../evidence/sections/docs-evidence.json) preserves commands, selected/excluded
tests, source hashes, skips and limitations. The completed section boundary ran:

```sh
make build
python3 scripts/next_test.py --section docs --suite all --report .tmp/docs-section.json
python3 scripts/next_test.py --section docs --suite live --allow-live --report .tmp/docs-live.json
```

The offline suite passed **995 Go test events**: 889 domain/parser unit events,
42 inherited deterministic E2E events, and 64 new production-process events.
Two explicit skips are not passes: the Linux-only empty clipboard probe on macOS,
and `TestDocsLivePrivateLifecycle` without verified private creation fixtures.
The inherited selector includes 17 top-level tests and excludes 36; its historical
live notes are not counted as current evidence. All newly written offline tests
ran, including XML/Markdown/IM Markdown, identity/scope rejection, zero-write
no-ops, native failed/pending result classification and partial grant recovery.

An isolated `go test -json -count=1 ./internal/affordance ./cmd/service ./internal/schema`
passed 272 events without skips; isolated `go vet ./shortcuts/doc/...` passed.
`git diff --check` passed. The isolated runner removes developer credentials,
disables external proxies, and the process fixture uses a local backend without
forwarding any request. Exact full output remains in `.tmp/docs-section.json`.
Full repository unit/static/quality/release gates and comparative development
evaluation belong to the coordinator and have not been claimed here. Held-out
tasks remain untouched.

Live scenarios are separately selected and remain **blocked**, not passed. Before
creating any content, a live harness must independently bind the effective current
account and tenant and verify creation visibility is restricted to that account.
Bot creation's automatic current-user access attempt needs explicit verification
within this boundary. Only an unambiguously resolved Ying Wu may be an additional
recipient. Existing unrelated documents may not be edited; cleanup owns only the
run's fixtures. Legacy live tests do not supply these prerequisites and are not run.
Native backend representation, conditional revisions/concurrency, large-media
multipart recovery, clipboard behavior and live history/preview guarantees remain
unverified until their respective authorized fixtures run. Section remains open.

### Private live batch, 2026-09-08

The first bounded current-user lifecycle ran after the coordinator verified the
account's invited-only creation default and an isolated personal parent folder.
The test repeated account/tenant binding and the parent's closed sharing, sole
owner/container membership and metadata ownership before creating one document.
The returned document's ACL was checked independently. No other recipient,
existing document, new scope, login or parent mutation was involved.

Initial native bold/link/Unicode, table and unrelated paragraph checks passed.
The plain XML `str_replace` request `Old status` → `New status` returned success
without warnings, but native readback showed `New status` as plain text, losing
bold and hyperlink on `status`. This is a **failed preservation observation**, not
a passed rich-edit gate. The CLI forwards the supplied XML unchanged; frozen
reference guidance describes replacement within rich text but does not explicitly
promise style inheritance when replacement content omits markup. No runtime
semantics were changed to conceal this result. Synthetic inheritance assumptions
do not prove this backend guarantee.

One create, one edit and one delete occurred without retries. Cleanup completed
and a separate GET of the exact returned document ID reported native API code
1770003, `resource deleted`. The parent folder remains intact. The dedicated
`TestDocsNextPrivateLiveLifecycle` now persists a returned ID and registers guarded
cleanup before asserting create success, including partial/unknown CLI results;
it requires explicit deleted evidence rather than empty metadata URLs.

The executable test is in `tests/cli_e2e/docs/docs_next_private_live_test.go`. It
requires `NEXT_DOCS_LIVE=1`, `LARK_CLI_BIN`, and absolute owner-readable
`NEXT_DOCS_LIVE_FIXTURE`, `NEXT_DOCS_LIVE_BINDING`, and
`NEXT_DOCS_LIVE_EVIDENCE_DIR` paths supplied by the coordinator. It never logs
personal IDs or complete native output into committed evidence. Every invocation
uses `RunCmdWithRetry` with `Attempts:1`. The default explicit XML edit and optional
`NEXT_DOCS_LIVE_EDIT_MODE=plain` observation are reported separately; neither can
replace the other's result. `NEXT_DOCS_LIVE_CHECK_NOOP=1` is a separately selected
check because an unequal overwrite preflight can perform a real write.

See [sanitized live evidence](../../evidence/sections/docs-private-live-evidence.json). Script/no-op checks
were not reached in the first attempt. Explicit rich XML, native media/cover,
history/revert, concurrency, bot access and complete type parity remain open.
This narrow live batch does not close the section.

A separately authorized fresh lifecycle then used explicit inline XML replacement
`New <a href="...same URL..."><b>status</b></a>`. It **passed** native bold/link,
Unicode, table and exact unrelated-block preservation checks, script parsing,
and single-attempt deletion with independent exact-ID deleted evidence. This
control establishes the bounded explicit XML workflow; it does not waive the
plain replacement observation or prove automatic style inheritance. Two total
run-owned documents were created and deleted across the two attempts; the parent
was left intact. No-op was not selected.

One concise `+update` affordance tip now explains the observed replacement
semantics. Its source check and embedded help check pass. This help change follows
the frozen comparisons and is newly unmeasured; historical benchmark conditions
and any evaluation validity audit remain separate.
