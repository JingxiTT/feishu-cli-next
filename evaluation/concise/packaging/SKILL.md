---
name: lark-shared-concise
description: Use for lark-cli setup, authentication and identity, permissions, output interpretation, high-risk confirmation, and update notices.
---

# lark-cli shared guidance

Use the frozen `lark-shared` distribution and references for setup, auth, identity,
output contracts and high-risk operations. Read the relevant reference or command
help before invoking unfamiliar flags. Full original references remain available
in the supplied upstream `lark-shared/references/` directory.

User identity (`--as user`) acts as the person and accesses their resources;
bot identity (`--as bot`) acts as the application and accesses the bot's resources.
Bot reads of personal resources can succeed with empty results. Bot messages and
created documents belong to the bot. Explicit `--as` overrides automatic selection;
keep it explicit when an operation sequence must retain identity. `whoami` reports
the effective identity and selection logic. `auth status --json --verify` checks
login and token validity; use its identity, verified, user status/name/openId,
tokenStatus and scope fields rather than guessing the current account.

Bot permissions come from developer-console scopes: use the returned `console_url`
and do not run user login to fix bot authorization. User permissions require both
console enablement and user authorization. `missing_scopes` can describe alternate
acceptable scopes. `auth login` must specify `--scope`, `--domain` or `--recommend`;
repeated grants accumulate. Use `--domain all` for a requested full grant; domains
can repeat or be comma-separated. Prefer the precise required scope for recovery.
`auth logout --json` clears local login (`loggedOut:true`); it does not revoke
server-side authorization. The user revokes that in Feishu authorization management.
Individual granted-scope revocation is unsupported; choose minimum new grants or
use authorization management.

For agent-assisted authorization, use `auth login` with the selected scope/domain
and `--no-wait --json`. Show the opaque verification URL unchanged, followed by a QR
image generated with `auth qrcode <url> --output <relative-path>`; ASCII is only for
an explicit request. Tell the user to return after authorizing, then yield. After
the user confirms, run `auth login --device-code <code>` yourself; do not ask the
user to execute it or poll before they have seen the URL. Never reuse expired or
cross-flow URLs/codes. A restarted grant preserves scope/domain/recommend choices
and any exclusions while requesting a new URL/code. Configuration URLs, including
`console_url`, also require unchanged URL plus QR. First-time app setup uses
`config init --new`; start the blocking flow in the background and surface its URL.

Default JSON success is `ok:true` on stdout with exit 0; failures are `ok:false`
with typed error on stderr and nonzero exit. Numeric API `code` belongs to errors,
not the success envelope. Do not interpret absent `code` as failure or accidentally
repeat a successful write. Never print secrets or tokens. File flags and `@file`
accept paths relative to the current working directory, not absolute paths; stdin
is useful for large payloads and avoiding path/quoting issues.

Confirm user intent before writes/deletes. Where available, `--dry-run` previews
URL, parameters and body without mutation or high-risk confirmation. Exit 10 with
`confirmation_required` is a high-risk approval gate: show action, risk and key
parameters, await explicit agreement, then append the hinted confirmation flag to
the original argv array. Do not silently confirm, alter parameters, treat it as a
network/permission error, or compose a shell command string to retry. If declined,
stop. Shortcut help and generated-command schema expose `high-risk-write` risk.

Finish the current task before acting on `_notice.update`. Prefer `lark-cli update`
when an update is requested: it updates CLI and skills together. `_notice.skills`
means installed guidance is out of sync; `_notice.deprecated_command` identifies a
replacement. Do not make notices the main answer or interrupt work with repeated
help queries. Stable machine output can suppress update and skills notifiers using
`LARKSUITE_CLI_NO_UPDATE_NOTIFIER=1` and `LARKSUITE_CLI_NO_SKILLS_NOTIFIER=1`.

Before the next operation, read matching original references once, in this order
when several apply: identity/permissions for identity, auth and scope decisions;
output contract for JSON or wrapper interpretation; high-risk approval for risk
or confirmation; config-init for first setup; update-notice for notices/updates.
This condenses the frozen shared guidance; it adds no new installation command.
