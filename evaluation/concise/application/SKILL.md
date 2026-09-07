---
name: lark-application
description: Manage the currently bound Open Platform app's slash commands with lark-cli. Use for listing, registering, editing or deleting commands; Feishu Apps development belongs to the separate apps domain.
---

Use `lark-cli application` for the bound app's slash commands. Both `--as user`
and `--as bot` are supported; retain the intended identity across operations.

- `+slash-command-list` returns all commands and their stable `command_id` values.
  There is no pagination; an app supports at most 100 commands.
- `+slash-command-create --command greet --description "Say hello"` registers a
  new name without its leading slash. Add repeatable
  `--description-i18n zh_cn=问候` and optional `--icon-key`. Each localization splits
  on the first `=`. Use `--force` only when the user intends a same-name create
  rerun to update the existing command; ordinary edits use update.
- `+slash-command-update` accepts exactly one of `--command-id` or `--command`
  and changes `--description`, `--description-i18n`, or `--icon-key`. Name
  targeting resolves the ID through a live list. Omitted top-level fields remain
  unchanged, but a supplied description replaces its whole object. Localizations
  replace the whole map, dropping omitted languages, and require `--description`.
- `+slash-command-delete` accepts exactly one ID or name. Deletion is irreversible;
  recreating or renaming a command produces a new ID. Pass `--yes` only after
  explicit user confirmation of the target and deletion. Never self-approve an
  exit-10 confirmation gate.

Server changes appear in list results immediately; clients can cache old commands
for about five minutes. Updates retain the command name and ID; renaming requires
intentional deletion and recreation.

Listing requires `application:app_slash_command:read`; writes require
`application:app_slash_command:write`. Name targeting and force collision recovery
also require read scope. Bot scope is configured in the developer console; do not
run user login to fix bot permissions. User access also needs user authorization.
Follow the actual error's missing scope and recovery hint. For user authorization,
use `auth login --scope "<scope>" --no-wait --json`, present the unchanged returned
URL with an `auth qrcode` image, and return control to the user. Complete with
`auth login --device-code <device_code>` after they confirm authorization.

Confirm user intent before writes and use supported `--dry-run` for dangerous
requests. Never expose credentials. Default JSON success uses `ok:true` or exit
zero, not a top-level OpenAPI `code`; errors go to stderr. CLI file paths are
relative to the working directory.
