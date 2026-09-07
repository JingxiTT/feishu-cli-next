# Drive section

Status: implementation and completed offline section suite pass; live creation privacy and comparative development trials remain open. This is a section report, not shipped command guidance.

## Reference, ownership, dependencies

Inspected sibling `../feishu-cli-review` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, frozen `coverage.json`, `registrations.json.gz`, `meta_data.json.gz`, source registrations, all Drive shortcut implementations and existing units/process workflows. The frozen tree has 41 Drive shortcuts, 25 generated service methods and 11 service navigation nodes (77 descendant registrations total). All are reused; none is replaced by raw forwarding or removed. Generated methods retain the generic service/runtime implementation and frozen nested schemas. No Drive-specific affordance file or event projection existed in the reference. Common event handling and subscription delivery belong to the events owner.

Owned paths: `shortcuts/drive/`, `tests/cli_e2e/drive/`, `tests/next_sections/drive/`, this report. Shared dependencies are runtime/FileIO, generic service dispatch, receipt and artifact output (`cfb5afa` conservatively handles Drive import task tickets), metadata identity binding, Wiki node resolution, Docs Markdown fetch, multipart media storage and caller auth/scopes. Strict token/type metadata lookup was coordinated into the shared owner at `5179287`; Drive fixture metadata now supplies identity. The shared read-only metadata request marker prevents a rename guard from appearing as a content mutation.

## Complete command mapping

Every shortcut below retains its frozen flags, enum/type metadata, scope/identity declarations, validation, request orchestration and nearby `_test.go` regressions. `TestEveryDriveShortcutDryRun` additionally executes each through the production process entry point. Workflow/resource tests also execute every shortcut against controlled responses and assert requests, output, or independently observed file/state effects. The complete section unit suite verifies uncommon flags/types beyond the concise process examples; registration visibility alone is not counted as execution evidence.

| Shortcut | Preserved domain flags (shared output/auth flags omitted) |
| --- | --- |
| `drive +add-comment` | `--block-id`, `--content`, `--doc`, `--full-comment`, `--type` |
| `drive +add-reply` | `--comment-id`, `--content`, `--token`, `--type`, `--url` |
| `drive +apply-permission` | `--perm`, `--remark`, `--token`, `--type` |
| `drive +batch-query-comments` | `--comment-ids`, `--need-reaction`, `--need-relation`, `--token`, `--type`, `--url` |
| `drive +copy` | `--extra`, `--folder-token`, `--name`, `--token`, `--type`, `--url` |
| `drive +cover` | `--file-token`, `--if-exists`, `--list-only`, `--output`, `--spec`, `--version` |
| `drive +create-folder` | `--folder-token`, `--name` |
| `drive +create-shortcut` | `--file-token`, `--folder-token`, `--type` |
| `drive +delete` | `--file-token`, `--type`, `--yes` |
| `drive +delete-reply` | `--comment-id`, `--reply-id`, `--token`, `--type`, `--url`, `--yes` |
| `drive +download` | `--file-token`, `--output`, `--overwrite`, `--url`, `--wiki-token` |
| `drive +export` | `--doc-type`, `--file-extension`, `--file-name`, `--only-schema`, `--output-dir`, `--overwrite`, `--sub-id`, `--token`, `--url` |
| `drive +export-download` | `--file-name`, `--file-token`, `--output-dir`, `--overwrite` |
| `drive +import` | `--file`, `--folder-token`, `--name`, `--target-token`, `--type` |
| `drive +inspect` | `--type`, `--url` |
| `drive +list-comments` | `--comment-scope`, `--need-reaction`, `--need-relation`, `--page-size`, `--page-token`, `--solved-status`, `--token`, `--type`, `--url` |
| `drive +list-replies` | `--comment-id`, `--need-reaction`, `--page-size`, `--page-token`, `--token`, `--type`, `--url` |
| `drive +member-add` | `--member-id`, `--member-kind`, `--member-type`, `--need-notification`, `--perm`, `--perm-type`, `--token`, `--type`, `--yes` |
| `drive +member-list` | `--fields`, `--perm-type`, `--token`, `--type` |
| `drive +member-remove` | `--member-id`, `--member-kind`, `--member-type`, `--perm-type`, `--token`, `--type`, `--yes` |
| `drive +move` | `--file-token`, `--folder-token`, `--type` |
| `drive +permission-get-setting` | `--token`, `--type` |
| `drive +preview` | `--file-token`, `--if-exists`, `--list-only`, `--output`, `--type`, `--url`, `--version`, `--wiki-token` |
| `drive +pull` | `--delete-local`, `--folder-token`, `--if-exists`, `--local-dir`, `--on-duplicate-remote`, `--yes` |
| `drive +push` | `--delete-remote`, `--folder-token`, `--if-exists`, `--local-dir`, `--on-duplicate-remote`, `--yes` |
| `drive +react-reply` | `--action`, `--emoji`, `--reply-id`, `--token`, `--type`, `--url` |
| `drive +resolve-comment` | `--comment-id`, `--token`, `--type`, `--url` |
| `drive +restore-comment` | `--comment-id`, `--token`, `--type`, `--url` |
| `drive +search` | `--chat-ids`, `--commented-since`, `--commented-until`, `--created-by-me`, `--created-since`, `--created-until`, `--creator-ids`, `--doc-types`, `--edited-since`, `--edited-until`, `--folder-tokens`, `--mine`, `--only-comment`, `--only-title`, `--opened-since`, `--opened-until`, `--original-creator-ids`, `--page-size`, `--page-token`, `--query`, `--sharer-ids`, `--sort`, `--space-ids` |
| `drive +secure-label-list` | `--lang`, `--page-size`, `--page-token` |
| `drive +secure-label-update` | `--label-id`, `--token`, `--type` |
| `drive +status` | `--folder-token`, `--local-dir`, `--quick` |
| `drive +sync` | `--folder-token`, `--local-dir`, `--on-conflict`, `--on-duplicate-remote`, `--quick` |
| `drive +task_result` | `--file-token`, `--scenario`, `--task-id`, `--ticket` |
| `drive +update-reply` | `--comment-id`, `--content`, `--reply-id`, `--token`, `--type`, `--url` |
| `drive +update-title` | `--on-extension-mismatch`, `--title`, `--token`, `--type`, `--url` |
| `drive +upload` | `--file`, `--file-token`, `--folder-token`, `--name`, `--wiki-token` |
| `drive +version-delete` | `--file-token`, `--version`, `--yes` |
| `drive +version-get` | `--file-token`, `--output`, `--overwrite`, `--version` |
| `drive +version-history` | `--cursor`, `--file-token`, `--limit` |
| `drive +version-revert` | `--file-token`, `--version` |

Generated service methods are directly exercised by `TestDriveCatalogMethods` using 25 explicit synthetic cases in `catalog_cases.json`: exact method, resolved path, all supplied query/body fields (including nested values), one dispatch, Unicode, empty/null values, leading-zero strings and continuation in the returned envelope. The data fixture was projected once from the frozen catalog; tests do not derive expected requests from the running implementation.

| Generated method | Preserved wire path |
| --- | --- |
| `drive quota_details get` | `GET /open-apis/drive/v2/quota_details/fixture` (synthetic IDs) |
| `drive files copy` | `POST /open-apis/drive/v1/files/fixture/copy` (synthetic IDs) |
| `drive files create_folder` | `POST /open-apis/drive/v1/files/create_folder` (synthetic IDs) |
| `drive files list` | `GET /open-apis/drive/v1/files` (synthetic IDs) |
| `drive files patch` | `PATCH /open-apis/drive/v1/files/fixture` (synthetic IDs) |
| `drive file.comment.replys create` | `POST /open-apis/drive/v1/files/fixture/comments/fixture/replies` (synthetic IDs) |
| `drive file.comment.replys delete` | `DELETE /open-apis/drive/v1/files/fixture/comments/fixture/replies/fixture` (synthetic IDs) |
| `drive file.comment.replys list` | `GET /open-apis/drive/v1/files/fixture/comments/fixture/replies` (synthetic IDs) |
| `drive file.comment.replys update` | `PUT /open-apis/drive/v1/files/fixture/comments/fixture/replies/fixture` (synthetic IDs) |
| `drive metas batch_query` | `POST /open-apis/drive/v1/metas/batch_query` (synthetic IDs) |
| `drive file.statistics get` | `GET /open-apis/drive/v1/files/fixture/statistics` (synthetic IDs) |
| `drive file.view_records list` | `GET /open-apis/drive/v1/files/fixture/view_records` (synthetic IDs) |
| `drive file.comments patch` | `PATCH /open-apis/drive/v1/files/fixture/comments/fixture` (synthetic IDs) |
| `drive file.comments batch_query` | `POST /open-apis/drive/v1/files/fixture/comments/batch_query` (synthetic IDs) |
| `drive file.comments create_v2` | `POST /open-apis/drive/v1/files/fixture/new_comments` (synthetic IDs) |
| `drive file.comments list` | `GET /open-apis/drive/v1/files/fixture/comments` (synthetic IDs) |
| `drive permission.members auth` | `GET /open-apis/drive/v1/permissions/fixture/members/auth` (synthetic IDs) |
| `drive permission.members create` | `POST /open-apis/drive/v1/permissions/fixture/members` (synthetic IDs) |
| `drive permission.members transfer_owner` | `POST /open-apis/drive/v1/permissions/fixture/members/transfer_owner` (synthetic IDs) |
| `drive permission.public get` | `GET /open-apis/drive/v1/permissions/fixture/public` (synthetic IDs) |
| `drive permission.public patch` | `PATCH /open-apis/drive/v1/permissions/fixture/public` (synthetic IDs) |
| `drive user remove_subscription` | `DELETE /open-apis/drive/v1/user/remove_subscription` (synthetic IDs) |
| `drive user subscription` | `POST /open-apis/drive/v1/user/subscription` (synthetic IDs) |
| `drive user subscription_status` | `GET /open-apis/drive/v1/user/subscription_status` (synthetic IDs) |
| `drive file.comment.reply.reactions update_reaction` | `POST /open-apis/drive/v2/files/fixture/comments/reaction` (synthetic IDs) |

## Behavioral acceptance scenarios

| Capability family | Retained semantics and evidence |
| --- | --- |
| Navigation, links, copies and moves | URL/bare-token ambiguity, aliases, wiki unwrapping vs wiki-node rename identity, My Space root resolution, all source resource kinds, copy custom extras, reference shortcut creation, synchronous file moves and async folder task polling. `drive_copy*`, `drive_move*`, `drive_inspect*`, `drive_create_*` units; process `TestWikiCopyMoveAndGuardedRename`, `TestResourceAndVersionOperations`, catalog copy/list/metas cases. |
| Upload/download | Small and multipart (>20 MB) upload, folder/wiki targets, in-place overwrite/file-token stability, version aliases, remote metadata naming, export permission preflight, FileIO portability/path constraints, overwrite conflicts. `drive_io_test.go` and source/file/preview units; `TestUploadDownloadAndVersionBytes` independently compares uploaded and downloaded binary/Unicode bytes, historical bytes and export artifact bytes. Multipart unit fixtures exercise chunks and finish; live transfer remains unverified. |
| Import/export | Markdown/docx/pdf/xlsx/csv/base and supported type/format constraints, schema-only Base exports, sub-ID, imported filename/mount/target semantics, Wiki addressing, async polling, resume command, safe filename and overwrite, complete Markdown bytes. `drive_import*`, `drive_export*`, `drive_task_result*` units and process `TestImportExportPreservesSourceAndAsyncRecovery`. Offline fake PDF verifies byte transport only, not document conversion quality. |
| Preview/cover/history | All preview type/status mappings, source-file path, seven cover presets, output conflict policies; full version flags, cursors, get/revert/delete. Preview/version units and resource process cases plus version byte fidelity. |
| Mirrors | Exact hash vs explicitly quick timestamp comparison, duplicate fail/newest/oldest/rename policies, local/remote conflict strategies, remote non-file preservation, file-only deletion, local FileIO constraints, partial outcomes. Full status/push/pull/sync and duplicate units; process exact-hash status/sync no-op, push/pull byte preservation, repeated/missing continuation, folder cycle and duplicate fail-before-write. |
| Comments | Full/local anchors for doc/docx/file/sheet/slides/Base, applications read/reply operations, rich text/link/mention elements and escaping/limits, comment/reply pagination, relation and reaction data, batch identity matching, resolve/restore, reply update/delete and reactions. All comment units and lifecycle process tests preserve actual content and observe each mutation. |
| Permissions/sharing | Resource-kind distinctions incl folder/wiki/apps/minutes; member type/kind/permission scope, notification flags, single/batch grants, batch missing/mismatched IDs and partial result, removal, permission application, full public settings, ownership transfer and generic auth. Permission units, catalog cases and `TestPermissionsReadGrantRemoveAndPartial`; synthetic recipients only. No test-specific recipient restrictions added to product behavior. |
| Search/labels | All owner/original-creator, timestamp-window, resource/location, chat/sharer, text/comment and sort filters; Unicode code-point limits, bounded page size and cursor; tenant secure-label types/IDs/errors. Search/label units; bound search artifact continuation and account/tenant binding; resource process cases. |
| Failure/recovery | No hidden replay after a lost create response; durable unknown receipt and offline inspection; scope rejection retains code; incomplete mirror does not modify local/remote files; conservative partial grant output keeps missing IDs. All tested via real CLI processes against non-forwarding local backends. |

## Intentional changes and limits

- Guarded file rename (`--on-extension-mismatch=keep`) compares the extension-resolved title with the exact bound current name and returns `updated:false, unchanged:true` without PATCH when equal. `allow` retains its explicit write-only contract, and other resource types retain their no-pre-read rename behavior. This observation is not a compare-and-swap guarantee; another client can edit after the read. Reply edits, label changes and version reverts preserve their reference mutation semantics; no unsupported global no-op promise is made.
- Recursive mirror listings fail with typed `internal/invalid_response` for folder cycles, missing/repeated continuation tokens, malformed collections/pagination/entries or missing names/tokens/types, before accepting a partial tree as complete. Valid pagination/large trees remain supported; this is not a fixed resource-count cap or snapshot-consistency guarantee.
- Deleting mirrors (`+pull --delete-local` and `+push --delete-remote`) additionally require explicit `files` arrays and boolean `has_more` on every page and child listing. Missing/null collection or omitted pagination evidence stops before transfers/deletion with `validation/failed_precondition`; malformed present pagination retains `internal/invalid_response`. Frozen upstream list fields are optional, so ordinary listing and non-deleting mirrors preserve omitted-field compatibility. Explicit `files:[], has_more:false` remains a valid empty mirror. Process tests independently verify local byte preservation, zero transfer/mutation calls, child/page completeness and real empty deletion.
- JSON uses shared compact formatting and opt-in result/artifact controls. Legacy dry-run assertions were changed to parse fields, preserving the request contract independently of indentation. Generic artifacts are not accepted as import source payloads.
- Shared metadata validates the returned resource identity; it cannot prove backend isolation or concurrency. Process fixture binding is synthetic evidence, separately labeled from live account/tenant verification.

## Live fixture investigation and blockers

No live write was made. The existing legacy live workflows are retained but must not run with developer credentials under this plan. They include default bot root creation/auto-grants and do not establish creation visibility beforehand. The offline runner deliberately excludes these tests and reports them as unverified.

Local source evidence: `drive_create_folder.go` posts only `name` and `folder_token`; omitted parent selects caller root, and bot creation attempts to grant current CLI user full access after creation. `drive_copy.go`/`drive_move.go` resolve My Space through `GET /open-apis/drive/explorer/v2/root_folder/meta`. `+permission-get-setting` reads the target's own settings, without validating inherited child visibility; `+member-list` complements but does not prove creation defaults/inheritance. Closed link sharing alone cannot establish that no group/member/organization access exists. Frozen create schemas contain no atomic private-visibility field.

Official pages checked: [create folder](https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/create_folder) and [permission settings](https://open.feishu.cn/document/server-docs/docs/permission/permission-public/get) loaded as empty client-rendered pages through web tools, so they supplied no verifiable private-default guarantee. No unsupported claim about private creation was inferred.

Required live setup before any write: effective current user and tenant evidence, a destination whose current and inherited membership/public settings are verified account-only, and a verified creation-visibility rule for that destination. Then create a clearly named run-owned folder, run transfer/copy/move/title/mirror/comment/version/permission scenarios only on its own objects, independently read effects, and clean up only run-owned IDs. All other-recipient effects remain blocked unless Ying Wu is resolved unambiguously; public/org/group access and unrelated existing edits/deletes remain forbidden for testing. Secure-label flows also require tenant label configuration and may require approval; notification/subscription side effects require recipient verification. These are test boundaries, not product allowlists.

Live creation visibility, backend conversion/preview fidelity, propagation, actual scope combinations, async completion, current account/tenant binding and authorized collaborator delivery remain unverified. These blockers keep P2 Drive open. Comparative development trials are scheduled by the coordinator; held-out tasks were not accessed.

## Validation

Canonical command: `python3 scripts/next_test.py --section drive --suite all --report .tmp/drive-all.json`. It restores the frozen catalog, builds the normal CLI for legacy dry-runs, runs all Drive units, the reference `DryRun|Regression` process selector, and the production authsidecar process suite with isolated local backends. The process fixture uses `nextfixture.RunMain(m, "../../..")`/`NewFixture`; no requests are forwarded to live services. Parent config/auth environment is removed by `isolated_env`.

Completed boundary (2026-09-07): `make build` and `python3 scripts/next_test.py --section drive --suite all --report .tmp/drive-p3-all.json` exited 0; 1,353 Drive unit test events, 232 inherited deterministic process events, and 116 new process events passed with zero failures or skips. Version dry-runs now use the shared `LARK_CLI_BIN` resolver and synthetic user credentials, so every version case executes against the supplied canonical next binary. Earlier inherited version checks used a hardcoded legacy executable and must not be treated as next-binary evidence. Legacy live/workflow cases excluded by the deterministic selector remain unverified. The new deletion process cases cover absent/null fields, subsequent pages and child folders with no changes, optional-field read compatibility, and successful explicit empty deletion. Raw report: `.tmp/drive-p3-all.json`. Current isolated scoped vet passed (`.tmp/drive-p3-vet.json`); owned changed Go paths are gofmt-clean.

Implementation commits: `eb060fe` (incomplete/cyclic mirror guard), `4e6cf20` (unchanged guarded file rename), `d84442d` (malformed listing shape guard), `dcd58f2` (inherited assertions under shared contracts). The process-suite/report commit contains `catalog_cases.json` and seven Go process test files; each fixture uses its own temporary config/workspace and bounded process invocation. Shared fixes consumed: `5179287` metadata identity/read classification, `da1337e` relative telemetry command name, `cfb5afa` async import ticket classification.

Repository-wide unit/vet/tidy/lint/license/quality gates, authorized live runs and comparative agent evaluations are coordinator release checks and were not claimed here. A passing offline suite does not close the live and comparative blockers above.
