---
name: feishu-cli-next
description: Work with Feishu/Lark documents, messages, calendars, spreadsheets, Base, files, and other workspace resources through the feishu-cli-next CLI.
---

# Feishu CLI Next

Use `feishu-cli-next` for Feishu/Lark workspace operations. It includes workflow shortcuts, generated OpenAPI commands, raw API access, and event subscriptions. Choose direct commands or compose them with shell tools and files as the task requires.

`feishu-cli-next discover "search documents" --limit 8` finds commands from the executable's current command definitions. Command `--help` describes flags and operation-specific guidance; `schema` exposes API input shapes. Names beginning with `+` are workflow shortcuts. `api` provides raw OpenAPI access when custom requests are useful.

For deeper domain concepts or format limitations, `skills list` and `skills read <name> [path]` expose the inherited upstream documentation on demand. Examples in those documents use `lark-cli`; invoke this binary as `feishu-cli-next`. Base formula and lookup field definitions have a guide acknowledgement flag; their syntax and flag contract are available through `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` and `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`.

Authentication and profiles select the account and Feishu/Lark environment. `auth status` reports the current login; `auth login` handles user authorization. `--as user` and `--as bot` select distinct identities with distinct permissions. A browser authorization step needs the user's participation; follow the concrete recovery hint when authentication or scopes are missing.

Use native resource IDs or supported URLs. Display names can be ambiguous, and user IDs, chat IDs, document tokens, and Base record IDs are different identifiers. Resolve ambiguity before sending, sharing, assigning, or changing a resource. Message sends and invitation/notification options affect other people; preserve the user's intended audience.

Prefer file/stdin flags offered by the chosen command for substantial content. Document structure, spreadsheet formulas and types, and Base field schemas have domain-specific representations; select a format that preserves the required data. An omitted field is not necessarily an explicit clear or delete.

JSON is compact by default without dropping fields. `--result-detail` restores indentation; `--result-fields` selects result fields while retaining status and decision-critical metadata. Where supported, `--save-result PATH` saves the full result without overwriting an existing file. Command-specific `--fields` or `--detail` flags keep their own meanings.

JSON results use `ok` and the process exit status; a successful response need not contain an OpenAPI `code`. Pagination and partial-failure metadata describe completeness. A timeout after a mutation can mean the effect is unknown: reconcile the affected resource or use documented backend idempotency before retrying a create, append, send, or other non-idempotent write.
