---
name: lark-wiki
version: 1.0.3
description: "Manage Wiki spaces, members, nodes and hierarchy; resolve Wiki links, create pages or shortcuts, and move or copy nodes. Route /wiki/ links by path and token, including doubao.com. Use Drive for uploads, file search and permissions; Docs, Sheets or Base for underlying content."
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli wiki --help"
---

# Wiki

Use the [shared authentication, permission and safety guidance](../lark-shared/SKILL.md). Prefer explicit `--as user` for personal Wiki resources; `auto` can resolve to bot and expose a different resource set. Use bot when the user requests the app's view. Keep the same identity throughout resolution, mutation and task continuation; do not silently switch identities to work around an error.

`my_library` means the user's Wiki personal document library, not the Drive root. The space-list API excludes it; resolve it with `lark-cli wiki spaces get --params '{"space_id":"my_library"}' --as user`. Supported shortcuts also resolve this alias. Bot has no personal-library fallback and cannot use `my_library`.

## Resolve and route

`lark-cli wiki +node-get --node-token <TOKEN_OR_URL> --as user` accepts a Wiki token or typed resource URL. A raw backing-object token needs `--obj-type doc|docx|sheet|bitable|mindnote|slides|file`; a raw token without that flag means a Wiki node token. URL path determines token type, independent of hostname. `--space-id` asserts the resolved space. `--token` is the deprecated alias; conflicting aliases are invalid.

Use returned `space_id`, `node_token`, `obj_token` and `obj_type` for their distinct purposes. Wiki hierarchy operations use the node token; content operations use the backing object and its type. The read does not synthesize a canonical URL. Prefer this shortcut for resolution; the generated `wiki spaces get_node` remains available for raw fields the shortcut omits or older CLI versions without the shortcut. Help and `lark-cli schema wiki.<resource>.<method>` expose command and API details when needed.

- Underlying document content belongs to [Docs](../lark-doc/SKILL.md), [Sheets](../lark-sheets/SKILL.md) or [Base](../lark-base/SKILL.md).
- In-place rename: `lark-cli drive +update-title --url <WIKI_URL> --title <TITLE>` preserves the node token. Do not create or copy a replacement node to rename it.
- Upload below a Wiki node: [Drive](../lark-drive/SKILL.md), `drive +upload --wiki-token <NODE_TOKEN>`.
- Download a backing `file`: `drive +download --wiki-token <NODE_TOKEN>` or `--url <WIKI_URL>`. Export online documents (`docx`, `sheet`, `bitable`, `slides`) with `drive +export`.
- Name/content search, document comments and document permissions belong to Drive. Topic-based collection uses the [Drive workflow registry](../lark-drive/references/lark-drive-workflow.md) and [topic_move_collector](../lark-drive/references/lark-drive-workflow-topic-move-collector.md), including broad Drive search rather than only a local Wiki traversal.
- Inventory, categorization, directory restructuring or a move plan uses that registry's [knowledge_organize](../lark-drive/references/lark-drive-workflow-knowledge-organize.md), which coordinates Drive/Wiki/personal-library inventory, planning, confirmation and verification.

## Spaces and navigation

- [`+space-list`](references/lark-wiki-space-list.md): accessible spaces, including visibility/sharing metadata. [`+space-create`](references/lark-wiki-space-create.md): `--name <NAME> [--description <TEXT>]`, user only; blank names are invalid.
- [`+node-list`](references/lark-wiki-node-list.md): `--space-id <NUMERIC_SPACE_ID|my_library> [--parent-node-token <NODE_TOKEN_OR_WIKI_URL>]`. Omit parent for roots; descend through each `has_child:true` node. Resolve non-Wiki URLs to their Wiki node token first.
- [`+member-list`](references/lark-wiki-member-list.md): `--space-id <SPACE_ID|my_library>`, returning the exact member ID/type/role tuple.

All three list shortcuts fetch one page by default: `--page-size` is 1–50 (default 50); `--page-all` uses `--page-limit` (default 10, 0 unlimited). Explicit `--page-token` overrides auto-pagination and reads one page. Preserve `has_more` and the continuation token; a capped result is not exhaustive. JSON is default; supported list/read presentation includes pretty, table, CSV and NDJSON. Never pass a title, document token or URL as a numeric space ID.

## Create, copy and move

Confirm the intended source, destination and effect before writes. `--dry-run` previews the orchestration without mutation.

- [`+node-create`](references/lark-wiki-node-create.md): `[--space-id <SPACE>] [--parent-node-token <PARENT>] [--title <TITLE>]`. Explicit space wins, otherwise infer from parent; user with neither falls back to `my_library`. Space and parent must agree. Default `--node-type origin --obj-type docx`. Origin supports `sheet`, `mindnote`, `bitable`, `docx`, `slides`; shortcut supports those plus `file` and requires `--origin-node-token`. Origin cannot take an origin token. Correct invalid combinations instead of bypassing validation through raw APIs. Bot creation may also grant the current CLI user access; inspect `permission_grant`.
- [`+node-copy`](references/lark-wiki-node-copy.md): `--space-id <SOURCE_SPACE> --node-token <NODE> (--target-space-id <SPACE> | --target-parent-node-token <PARENT>) [--title <TITLE>] --yes`. Choose exactly one target. Copies this node and its content, not descendants; omitted title preserves the title. The CLI retries `131009` lock contention twice with bounded backoff.
- [`+move`](references/lark-wiki-move.md), Wiki to Wiki: `--node-token <NODE> [--source-space-id <SPACE>]` and a target space and/or `--target-parent-token <PARENT>`. Source is resolved when omitted; target parent and explicit space must agree. A target space alone means its root.
- `+move`, Drive to Wiki: `--obj-token <OBJECT> --obj-type doc|docx|sheet|bitable|mindnote|file|slides --target-space-id <SPACE> [--target-parent-token <PARENT>] [--apply]`. Do not mix these options with node mode. `--apply` can submit a permission-dependent move request; `applied:true, ready:false` means approval is pending, not a completed move.
- [`+move-to-drive`](references/lark-wiki-move-to-drive.md): `--node-token <WIKI_NODE> [--folder-token <DRIVE_FOLDER>]`. Omitted folder means the current identity's Drive root. This removes the node from the Wiki tree and replaces Wiki inherited permissions with destination permissions. Use `drive +move` only when both source and destination are already in Drive.

## Members

Resolve person/group/department/app identity before a grant. Prefer `contact +search-user --query <NAME_EMAIL_OR_PHONE>` for `openid`; `im +chat-search --query <NAME>` for `openchat`; use a known app ID for `appid`. `userid` and `unionid` require the corresponding resolved user identifiers, not an assumed conversion. Department search is `api POST /open-apis/contact/v3/departments/search --as user --params '{"department_id_type":"open_department_id"}' --data '{"query":"<NAME>"}'`.

[`+member-add`](references/lark-wiki-member-add.md) and [`+member-remove`](references/lark-wiki-member-remove.md) require `--space-id`, `--member-id`, `--member-type` and `--member-role`. Types: `openid`, `userid`, `email`, `unionid`, `openchat`, `opendepartmentid`, `appid`. Roles: `admin` or `member`. Department grants with bot identity are unsupported: explain that restriction without probing or silently switching to user.

Adding supports `--need-notification`; omission leaves the query unset, `=false` explicitly opts out and `=true` requests notification. Removal needs the original ID/type/role tuple; inspect membership when unknown. Change roles by removing the old grant then adding the new one. Missing-grant removal is an API no-op error. A Wiki URL used for membership must first be resolved to `space_id` with the same downstream identity.

## Delete and recover

[`+node-delete`](references/lark-wiki-node-delete.md) accepts `--node-token <TOKEN_OR_URL> [--space-id <SPACE>] --yes`. A raw token requires `--obj-type wiki|doc|docx|sheet|bitable|mindnote|slides|file`; typed URLs infer it. Default `--include-children=true` deletes the subtree; `=false` lifts direct children to the parent. `131011` requires Wiki UI delete approval; `131003` can mean the subtree is too large.

[`+delete-space`](references/lark-wiki-delete-space.md) requires a real `--space-id` and `--yes`, deleting the space and all nodes. Resolve URLs with `+node-get`. For a name, page through `wiki spaces list`, stopping when exact matches are found or pages end; use trim/case-insensitive/substring matching only after exhausting all pages without an exact match. Early stopping can miss later duplicates. Present candidate name, ID, description and space type, and obtain the user's explicit ID selection even for one candidate. With no candidates, clarify the name or access; do not guess another name.

Async moves/deletes poll up to 30 times at two-second intervals. `ready:false` or timeout does not prove failure: preserve the complete opaque `task_id` and use the returned `next_command` with the same profile and identity instead of resubmitting. [Drive task-result guidance](../lark-drive/references/lark-drive-task-result.md) covers `wiki_move`, `wiki_move_to_drive`, `wiki_delete_node` and `wiki_delete_space` scenarios.

Treat invalid parameters, not-found and permission-denied responses as terminal for the current inputs. `131006` is resource access: request access from the owner/admin, not repeated authorization or identity switching. `131012` means deleted, `131013` invalid token, `131014` not mounted in Wiki (route to the backing domain). Creation `131003` means a structural limit; change the parent/layout/resource rather than retrying unchanged input. Rate limits need the supplied delay or exponential backoff with jitter; node-get stops after three total attempts.

Generated methods remain available: `wiki spaces create|get|get_node|list`, `wiki nodes create|copy|list`, and `wiki members create|delete|list`. Their schemas and risk gates may differ from shortcut flags; use the matching schema rather than guessing request fields.
