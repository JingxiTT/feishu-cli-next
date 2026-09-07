---
name: official-cli-concise
description: Use lark-cli for Feishu setup, authentication, discovery, and domain operations.
---

# lark-cli

`lark-cli` supports Feishu documents, Sheets, Base, files, messaging, calendars,
tasks, mail, events, and other registered services. `--help` discovers commands;
`schema <service> <resource> <method>` describes generated API methods. Domain
shortcuts begin with `+`. Complete domain guides are available through
`lark-cli skills read <name>` when needed.

`config show` inspects configuration; `auth status --json` reports authentication.
`auth status --verify` makes a network request. `auth login` and `config init`
perform setup when requested. Pass app secrets through stdin. Existing agent
workspaces can use `config bind`. User identity (`--as user`) acts for the person;
bot identity (`--as bot`) acts for the application and has different access.
Missing scopes require the corresponding authorization, not blind retries.

JSON success uses `ok: true`. Failures have a typed `error` on stderr and nonzero
exit status; preserve partial or unknown mutation outcomes. Prefer relative
workspace paths for file flags and stdin for large input. Preserve empty values,
types, Unicode, object IDs, and pagination completeness.

Make writes within the user's authorized intent. Resolve ambiguous recipients
before delivery. Commands supporting `--dry-run` can preview a request.
High-risk exit code 10 requires the documented explicit confirmation before
retrying; do not invent a successful outcome. Never expose credentials. When
setup emits an authorization URL, forward it unchanged and use `auth qrcode`
to provide the matching QR image. `_notice` contains optional update guidance.
