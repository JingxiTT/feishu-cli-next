---
name: feishu-cli-next
description: Use the Feishu/Lark CLI for documents, contacts, messages, calendars, Base, sheets, files and other workspace resources.
---

Use `--as user` for personal data.

## Routes

- Profile: `contact +get-user --as user` (current); another user: `--user-id <id> --user-id-type open_id`. Batch: `contact +search-user --user-ids <id,id> --as user`.
- People: `contact +search-user --query <name-or-email> --as user`; bots: `contact +search-bot --query <keyword> --as user`. `has_more=true`: refine.
- Docs read: `docs +pull --doc <URL-or-token> --as user`; for one fact add `--scope keyword --keyword <literal>`. `complete=false` means an excerpt. Unsupported/comments: `docs +fetch --doc <doc> --detail full`.
- Docs text edit (including rich text): `docs +pull --doc <doc> --output ./note.md --as user`; edit `note.md`; `docs +push --file ./note.md --as user`. Preserve `note.feishu.json` and `<!-- block:<id> -->`. Structure/styles: `docs +update`. `data.verified=true`: done.
- Import: `docs +create --doc-format markdown --content @./source.md --as user`; verify with `docs +fetch --doc <id> --detail full`. Creation waits for async completion; after a timeout or unknown outcome, inspect the receipt and destination before retrying.
- Base scan: `base +record-list --base-token <base> --table-id <table> --output ./rows.ndjson`; Parse locally. Follow `next_offset` until `has_more=false`; keep page files distinct.
- Base row: `base +record-get --record-id <id> --format json`; `base +record-upsert --json '{"<field>":<value>}'` (`--record-id <id>` updates; omit to create). Add `--base-token <base> --table-id <table>`. Types: `base +field-list`.
- Base batch: `base +record-batch-create --base-token <base> --table-id <table> --json '{"create_records":[{"<field>":<value>}]}'`; max 200/call. First: `base +field-list`. Confirmed returned IDs need no rescan.
- Formula/lookup: read `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` or `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`, then pass `--i-have-read-guide` to `base +field-create` or `base +field-update`.
- Sheet: `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <A1:B2> --save-result ./cells.json`. Fallback: `sheets +cells-get`, same flags. Preserve types/formulas.
- Cells: `sheets +cells-set --spreadsheet-token <token> --sheet-id <id> --range <A1> --cells '[[{"value":"x"}]]' --as user`. Schema: `--print-schema --flag-name cells`. Range/cells changes values while preserving styles; group disjoint range calls in one shell.
- Drive: `drive +upload --file ./source --as user` (folder: `--folder-token <folder>`); `drive +download --file-token <data.file_token> --output ./download --as user`.
- Source: `apps +export --app-id <app> --output ./source.zip --as user`; `/page/<token>` uses `--meta-token <token>`. Excludes uncommitted work.
- Apps: `apps +list --ownership mine --as user`.
- Mail: `mail +thread-modify --thread-ids <id,id> --add-folder <folder> --as user`. Delete: `mail +thread-trash --thread-ids <id,id> --as user --dry-run`, add `--yes` after user confirmation for authorized whole-thread deletion. Check `success_thread_ids` and `failed_thread_ids`.
- IM send: `im +messages-send --chat-id <oc_id> --text "<text>" --as user`; person: `--user-id <ou_id>`. Add `--idempotency-key <key>`. Confirm recipient first.
- IM read: `im +chat-messages-list --chat-id <oc_id> --as user`; add `--start`, `--end`, `--page-all` for history; `--no-reactions` skips reactions. `--concise` (Markdown, on request) excludes `--result-fields`, `--save-result`, `--format`, `--json`, `--jq`.
- Agenda: `calendar +agenda --start <ISO-8601> --end <ISO-8601> --as user`; default today.
- Meeting: `calendar +create --summary <title> --start <ISO-8601> --end <ISO-8601> --attendee-ids <ou_id,...> --as user`.
- Tasks: `task +get-my-tasks --as user`; add `--complete false` for open tasks.
- Task: `task +create --summary <title> --due date:<YYYY-MM-DD> --as user`. Task complete: `task +complete --task-id <guid> --as user`.
- OKR: `okr +cycle-list --user-id <ou_id> --as user`, then `okr +cycle-detail --cycle-id <id> --as user`.
- Wiki: `wiki +node-get --node-token <token-or-url> --as user`.

## Other

Run `discover <domain> <keywords> --result-fields command,plan --limit 3`: inputs/guidance; literal terms; shorten empty queries. Flags: `discover <command> --flags`. `skills read lark-<domain>` with `--outline`, then `--section <slug>`; follow references. `schema <service.resource.method> --input-only` API input.

Do not pre-read `lark-shared` or run auth checks. Use `feishu-cli-next skills read lark-shared` after auth/scope errors or before high-risk writes. Browser auth needs the user.

## Rules

- Large results: files (`--output`, `--save-result`, stdout redirect). Parse/filter/join with a local script after successful export; print needed facts/completeness. Reuse guidance already read for this version.
- Run known workflows in one local script: read inputs, call CLI, validate, compute/write. Group independent reads. Batch writes when supported; check every item.
- Check exit code, JSON `ok`, IDs and completeness; stop on partial receipts.
- Preserve IDs, formatting, and omitted/null/empty/zero/false values exactly.
- Never retry a mutation whose outcome is unknown; inspect the target first.
