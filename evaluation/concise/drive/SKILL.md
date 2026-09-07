---
name: lark-drive
version: 1.0.0
description: Manage Lark/Feishu Drive files and folders: upload/download, copy/move/delete, metadata, permissions, comments, subscriptions, titles, versions, secure labels and local-file import. Use for cloud-storage organization, resource URLs/tokens and Word/Markdown/Excel/CSV/PPTX/.base imports; doubao.com resources follow the same path/token routing. Use lark-doc for document content, lark-sheets/lark-base for internal data, lark-wiki for knowledge-space nodes/members and lark-markdown for native Markdown read/write/patch/diff.
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli drive --help"
---

# Drive

Before starting, read `../lark-shared/SKILL.md`. Its identity, authentication, permissions, output and high-risk rules apply. Read the relevant command reference or `--help` before invoking it; prefer shortcuts where available. “Cloud disk,” “cloud storage” and “My Space” refer to Drive.

Choose identity deliberately: `--as user` acts as the person and accesses their personal resources; `--as bot` acts as the application and sees its own accessible resources. Bot access to user resources can return empty success. Resource permissions and OAuth scopes are separate. Read the matching shared reference when diagnosing identity/auth/scopes, interpreting stdout/stderr or scripting, handling high-risk approval, setting up config or handling `_notice`. Never print credentials. Forward authorization/configuration URLs unchanged, followed by a PNG from `lark-cli auth qrcode`; use ASCII only when requested. JSON success is `ok:true` or exit 0, not top-level `code:0`. File paths must be relative to cwd; prefer stdin for large JSON or file inputs.

For writes, establish user intent. Before deletion, public-permission changes, ownership transfer, version delete/revert or bulk move/overwrite/sync, resolve exact executable targets and all details (type, sharing scope, new owner, version, destination and conflict policy), and obtain explicit confirmation of those targets and details. A vague desired outcome requires read-only discovery and a concrete plan before confirmation. Preview dangerous requests with `--dry-run` when supported. Exit 10 is the confirmation gate: show action, risk and key parameters, obtain explicit approval, then append the indicated confirmation flag to the original argv. Do not silently bypass it.

## Route the request

- Copy an online resource into Drive with `drive +copy`; do not export/import or reconstruct its content to make a copy. With a usable URL/token, use `references/lark-drive-copy.md`; with only a title, first resolve it uniquely using `references/lark-drive-search.md`. Copy into Wiki with `wiki +node-copy` and its reference.
- Move an existing Wiki node out to Drive/My Space with `wiki +move-to-drive` through `lark-wiki`. Confirm the source and destination because ownership location and inherited permissions change. Do not send a Wiki node token to `drive +move`. Atomic movement of a known Drive object remains `drive +move`.
- Clear URL paths can directly identify type/token: `/docx/`, `/doc/`, `/sheets/`, `/bitable/`, `/slides/`, `/file/` and `/drive/folder/`. `/page/` is `apps`; folder paths supply `folder_token`. A `/wiki/` token is a node, not an underlying file token. For ambiguous URLs/tokens, titles/canonical URLs or unknown underlying Wiki types, use `drive +inspect` and `references/lark-drive-inspect.md`; retain returned type, token, title and URL. Wiki may wrap docx, sheet, bitable, slides or file. See `../lark-shared/references/lark-wiki-token-routing.md`. Do not guess the underlying type or try another write interface after inspect fails. Feishu/Lark and doubao.com resource URLs use this routing; do not fall back to WebFetch because of the hostname.
- Document-body reading, summaries, creation/editing and embedded images/attachments go to `lark-doc` (`docs +fetch` can accept URLs). Spreadsheet cells/formulas/styles go to `lark-sheets`; Base tables/fields/records/views/dashboards go to `lark-base`; Wiki hierarchy/spaces/members go to `lark-wiki`. Local file upload under a Wiki node still uses `drive +upload --wiki-token`.
- Native `.md` file upload/create/read/patch/overwrite and Markdown historical/local-draft comparison go to `lark-markdown`, including `markdown +diff`; get versions with `drive +version-history` when needed. Importing Markdown into online docx remains Drive.

## Workflows and permissions

Read `references/lark-drive-workflow.md` first and select its registry workflow for:

- `permission_governance`: permission/risk audit or governance, public/link/external access, copy/download rights, secure labels, permission applications, ownership transfer or bulk ownership changes. Follow `references/lark-drive-workflow-permission-governance.md`.
- `topic_move_collector`: find material by topic/keyword/content across containers and collect it into Drive/Wiki. Follow `references/lark-drive-workflow-topic-move-collector.md` for recall, content verification, relevance classification, move planning, approval and verification; do not start directly with search/move.
- `knowledge_organize`: inventory or restructure known folders/libraries/Wiki, including unfiled, temporary, duplicate or empty-folder discovery. Follow `references/lark-drive-workflow-knowledge-organize.md`. Default to a plan; folder creation, moves and permission applications each require separate confirmation.

Standalone search uses `drive +search` and `references/lark-drive-search.md`. Resolve duplicate titles before acting. Map natural-language criteria to flat flags: original creator is `--created-by-me`, current owner is `--mine`; use edit/open time and document type filters as appropriate instead of guessing nested JSON.

For collaborators/public permissions, granting the application access or public permission errors 91009/91010/91011/91012, read `references/lark-drive-permission-guide.md`. `+permission-get-setting` reads the target's own public, sharing, collaborator-management, security and comment settings; it does not recursively audit children. Bare tokens require `--type`; read its matching reference. `+apply-permission` requests access from the owner as user. Tenant security, external sharing and secure-label policy errors are not automatically missing scopes.

`+member-add` grants one or at most ten members and requires `--yes`; `+member-list` reads collaborators; `+member-remove` removes one exact resource/member ID/type and Wiki permission scope with confirmation and `--yes`. Read the matching references. For available or changed secure labels, read `references/lark-drive-secure-label.md` and use `+secure-label-list`/`+secure-label-update`.

## Transfers and versions

- Use `+import --type bitable` first for local Excel/CSV/`.base` becoming Base; switch to `lark-base` only after import. Use `--type docx` for `.md/.docx/.doc/.txt/.html`, `--type sheet` for `.xlsx/.xls/.csv`, and `--type slides` for `.pptx` (500 MB limit). Read `references/lark-drive-import.md`. Serialize imports into the same folder, default root or target token. On 232140101/232140100/233523001 concurrency conflicts, wait several seconds before each retry, at most three retries per failed item, then stop and report.
- Use `+upload` for local files into a folder or Wiki node; updating an existing file should prefer overwrite upload instead of creating another file. Use `+create-folder` for folders (including its bot creation/automatic grant behavior), and `+create-shortcut` for a reference in another folder. Read the matching references.
- `+download` and `+preview` handle uploaded Drive **files**. Select exactly one of `--file-token`, `--url`, `--wiki-token`; Wiki resolves to the underlying object, which must be `file`. Online docx/sheet/bitable/slides require `+export` to a supported local format, preferably `+export --url '<url>' --file-extension <format>`. Read `references/lark-drive-export.md` for formats, Wiki addressing and errors; `+export-download` downloads an export artifact.
- Download to a relative path with `lark-cli drive +download --file-token <token> --output ./report.pdf`. Omitting `--output` saves into the current directory. An existing destination fails unless `--overwrite` was requested. Direct file/Wiki URLs work with `--url`; use `--wiki-token` for a bare Wiki node token. The result includes Wiki node and underlying token/type details when resolved. If uncertain, `+inspect --url <token> --type wiki` checks the node. On `permission_denied` or final HTTP 403, follow the error hint to `drive +preview --file-token <token> --type source_file --output <path>`. On rate limiting, stop immediate retries and use delayed exponential backoff. See `references/lark-drive-download.md`.
- `+preview` lists available preview formats or retrieves source/PDF/HTML/text/image variants; read `references/lark-drive-preview.md`. For covers, use `+cover --list-only` first, then select `--spec`; read its reference.
- `+version-history`, `+version-get`, `+version-revert`, `+version-delete` support user and bot; prefer bot for automation. Read each matching `references/lark-drive-version-*.md`. Use the returned version identifier, not a tag. Delete/revert need the exact version and high-risk approval.
- `+task_result` queries import/export/move/delete async outcomes; read `references/lark-drive-task-result.md`. `+delete` polls asynchronous folder deletion; `+update-title` renames files/folders/online documents/Wiki. Read their matching references.

For inspect/upload, stop retrying on not found, permission denied or missing scope; only rate limits or temporary network errors justify bounded retries. For wrong-token/not-exist errors, recheck type and unwrap Wiki; for invalid file type, use its actual object type. Handle resource permission errors for the selected identity.

## Directory mirrors

Read `references/lark-drive-status.md`, `lark-drive-pull.md` or `lark-drive-push.md` before that operation. `+status` compares SHA-256 exactly; `--quick` is an approximate modification-time comparison. `+pull` supports incremental download and duplicate-path handling. `+push` supports skip/smart/overwrite and confirmed remote deletion. Resolve destination and conflict/deletion policy before bulk changes.

`+sync` pulls `new_remote`, pushes `new_local` and resolves `modified` with `--on-conflict=remote-wins|local-wins|keep-both|ask`; `--quick` is approximate. Duplicate remote policy supports fail/newest/oldest. Sync transfers only `type=file`, skips online documents/shortcuts, and does not delete extra files on either side.

## Comments and native API

Before any comment operation, read the command's corresponding reference under `references/lark-drive-<verb>.md`. Prefer `+add-comment`, `+list-comments`, `+batch-query-comments`, `+resolve-comment`, `+restore-comment`, `+add-reply`, `+list-replies`, `+update-reply`, `+delete-reply`, `+react-reply`. Add whole/local comments to doc/docx/file/sheet/slides/base(bitable), not apps. Replies update by whole replacement; reply deletion is high risk and needs `--yes`. Resolve sets solved true; restore sets it false. Use `references/lark-drive-comment-location.md` to locate body anchors.

Before native API use, run `lark-cli schema drive.<resource>.<method>`; never guess `--data`/`--params`. Then call `lark-cli drive <resource> <method> [flags]`.

| Resource | Native operations and choice |
| --- | --- |
| `files` | copy/create_folder/list/patch; prefer copy/title shortcuts. Before folder listings read `references/lark-drive-files-list.md`, pass `--params` per its template and manually page. Do not parse `--page-all` output as one JSON document. |
| `permission.members` | auth/create/transfer_owner; follow permission and approval guidance. |
| `metas` | batch_query for bound document metadata. |
| `user` | subscription/remove_subscription/subscription_status for user/application events, including comment-added events. |
| `file.statistics` | get; prefer `--file-token <token> --file-type <type> --format json`; JSON params remain valid. |
| `file.view_records` | list; prefer typed file-token/file-type/page-size/viewer-id-type flags, with `--viewer-id-type open_id` when requesting open IDs; JSON params support paging/raw compositions. |
| `file.comment.reply.reactions` | update_reaction; prefer `+react-reply`. |
| `quota_details` | get as **user only**, passing the current user's `user_id` as `quota_detail_id`. |
