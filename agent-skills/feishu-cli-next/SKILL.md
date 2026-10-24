---
name: feishu-cli-next
description: Use the Feishu/Lark CLI for documents, contacts, messages, calendars, Base, sheets, files and other workspace resources.
---

Run these routes directly. Use `--as user` for personal data; bot permissions differ.

## Routes

- **Profile:** `contact +get-user --user-id <id> --user-id-type open_id --as user`. Several users: `contact +search-user --user-ids <id,id> --as user`.
- **Search people:** `contact +search-user --query <name-or-email> --as user`; bots: `contact +search-bot --query <keyword> --as user`. Refine if `has_more` is true.
- **Docs read:** `docs +pull --doc <URL-or-token> --as user`; for one fact add `--scope keyword --keyword <literal>`. `complete=false` means an excerpt. Unsupported regions or comments: `docs +fetch --doc <doc> --detail full`.
- **Docs edit:** `docs +pull --doc <doc> --output ./note.md --as user`; change text in `note.md`; then `docs +push --file ./note.md --as user`. Keep `note.feishu.json` and `<!-- block:<id> -->` lines unedited. Structure or formatting changes: `docs +update`. If `data.verified` is true, do not fetch again.
- **Markdown import:** `docs +create --doc-format markdown --content @./source.md --as user`; verify with `docs +fetch --doc <id> --detail full`. Creation waits for async completion; after a timeout or unknown outcome, inspect the receipt and destination before retrying.
- **Base read:** `base +record-get --base-token <base> --table-id <table> --record-id <id> --format json`.
- **Base write:** `base +record-upsert --base-token <base> --table-id <table> --record-id <id> --json '{"<field>":<value>}'`; omit `--record-id` to create. Unknown field types: `base +field-list`.
- **Base many rows:** `base +record-batch-create --base-token <base> --table-id <table> --json '{"create_records":[{"<field>":<value>}]}'`; at most 200 rows per call.
- **Base formula or lookup fields** (create or update only): read `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` or `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`, then pass `--i-have-read-guide` to `base +field-create` or `base +field-update`.
- **Sheets read:** `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <A1:B2> --save-result ./cells.json`. If it fails, use `sheets +cells-get` with the same flags. CSV can lose types and formulas.
- **Sheets write:** `sheets +cells-set --spreadsheet-token <token> --sheet-id <id> --range <A1> --cells '[[{"value":"x"}]]' --as user`. Unfamiliar flag: `--print-schema --flag-name cells`.
- **Drive:** `drive +upload --file ./source --as user` (add `--folder-token <folder>` for a folder); then `drive +download --file-token <data.file_token> --output ./download --as user`.
- **App source:** `apps +export --app-id <app> --output ./source.zip --as user`; a shared `/page/<token>` link uses `--meta-token <token>` instead. Uncommitted work is excluded.
- **App list:** `apps +list --ownership mine --as user`.
- **Mail threads:** `mail +thread-modify --thread-ids <id,id> --add-folder <folder> --as user`. To delete: `mail +thread-trash --thread-ids <id,id> --as user --dry-run`, then add `--yes` only for authorized whole-thread deletion, after user confirmation. Check `success_thread_ids` and `failed_thread_ids`.
- **IM send:** `im +messages-send --chat-id <oc_id> --text "<text>" --as user`; for a person, use `--user-id <ou_id>`. Add `--idempotency-key <key>` to prevent duplicates. Confirm the recipient first.
- **IM read:** `im +chat-messages-list --chat-id <oc_id> --as user`; add `--start`, `--end`, `--page-all` for history, and `--no-reactions` to skip reactions. `--concise` (Markdown, on request only) excludes `--result-fields`, `--save-result`, `--format`, `--json`, `--jq`.
- **Calendar agenda:** `calendar +agenda --start <ISO-8601> --end <ISO-8601> --as user`; defaults to today.
- **Calendar create:** `calendar +create --summary <title> --start <ISO-8601> --end <ISO-8601> --attendee-ids <ou_id,...> --as user`.
- **Tasks list:** `task +get-my-tasks --as user`; add `--complete false` for open tasks.
- **Task create:** `task +create --summary <title> --due date:<YYYY-MM-DD> --as user`. **Task complete:** `task +complete --task-id <guid> --as user`.
- **OKR read:** `okr +cycle-list --user-id <ou_id> --as user`, then `okr +cycle-detail --cycle-id <id> --as user`.
- **Wiki node:** `wiki +node-get --node-token <token-or-url> --as user`.

## Unlisted tasks

Run `feishu-cli-next discover <keywords> --limit 5`, or `feishu-cli-next skills read lark-<domain>` (that domain's SKILL.md; its references only if it points to one). Request shape: `schema <service.resource.method> --input-only`; bare `schema` dumps the whole catalog.

Do not pre-read `lark-shared` or run auth checks. Read `feishu-cli-next skills read lark-shared` only after an auth, token, identity, or scope error, or before a high-risk write. Browser authorization needs the user.

## Rules

- Keep large output in files (`--save-result`, `--output`); read small stdout only. Reuse guidance already read for this version.
- Gate each next step on exit code, JSON `ok`, IDs, and completeness; stop on partial receipts.
- Preserve IDs, formatting, and omitted/null/empty/zero/false values exactly.
- Never retry a mutation whose outcome is unknown; inspect the target first.
