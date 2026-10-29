---
name: feishu-cli-next
description: Use the Feishu/Lark CLI for documents, contacts, messages, calendars, Base, sheets, files and other workspace resources.
---

Use `--as user` for personal data.

## Routes

- **Profile:** `contact +get-user --as user` (current); add `--user-id <id> --user-id-type open_id` otherwise.
- **Search people:** `contact +search-user --query <name-or-email> --as user`. Refine if `has_more` is true.
- **Docs read:** `docs +pull --doc <URL-or-token> --as user`; for one fact add `--scope keyword --keyword <literal>`. `complete=false` means an excerpt. Unsupported content/comments: `docs +fetch --doc <doc> --detail full`.
- **Docs text edit (including rich text):** `docs +pull --doc <doc> --output ./note.md --as user`; change text in `note.md`; then `docs +push --file ./note.md --as user`. Preserve `note.feishu.json` and `<!-- block:<id> -->`. Change structure/styles: `docs +update`. `data.verified=true` needs no re-fetch.
- **Markdown import:** `docs +create --doc-format markdown --content @./source.md --as user`; verify with `docs +fetch --doc <id> --detail full`. Creation waits for async completion; after a timeout or unknown outcome, inspect the receipt and destination before retrying.
- **Base scan:** `base +record-list --base-token <base> --table-id <table> --output ./rows.ndjson`; Parse locally. Follow `next_offset` until `has_more=false`; keep page files distinct.
- **Base many rows:** `base +record-batch-create --base-token <base> --table-id <table> --json '{"create_records":[{"<field>":<value>}]}'`; max 200 rows/call. Read `base +field-list` first. Confirmed returned IDs need no rescan.
- **Base formula/lookup**: read `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` or `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`, then pass `--i-have-read-guide` to `base +field-create` or `base +field-update`.
- **Sheets read:** `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <A1:B2> --save-result ./cells.json`. Fallback: `sheets +cells-get`; same flags; preserve types/formulas.
- **Sheets write:** `sheets +cells-set --spreadsheet-token <token> --sheet-id <id> --range <A1> --cells '[[{"value":"x"}]]' --as user`. Input schema: `--print-schema --flag-name cells`. Range/cells changes values while preserving styles; for disjoint ranges group separate calls in one shell.
- **Drive:** `drive +upload --file ./source --as user` (`--folder-token <folder>` targets a folder); `drive +download --file-token <data.file_token> --output ./download --as user`.
- **App source:** `apps +export --app-id <app> --output ./source.zip --as user`; `/page/<token>` uses `--meta-token <token>`. Excludes uncommitted work.
- **App list:** `apps +list --ownership mine --as user`.
- **Mail threads:** `mail +thread-modify --thread-ids <id,id> --add-folder <folder> --as user`. Delete: `mail +thread-trash --thread-ids <id,id> --as user --dry-run`, then add `--yes` after user confirmation of authorized whole-thread deletion. Check `success_thread_ids` and `failed_thread_ids`.
- **IM send:** `im +messages-send --chat-id <oc_id> --text "<text>" --as user`; person: `--user-id <ou_id>`. Add `--idempotency-key <key>`. Confirm recipient first.
- **IM read:** `im +chat-messages-list --chat-id <oc_id> --as user`; add `--start`, `--end`, `--page-all` for history; `--no-reactions` skips reactions. `--concise` (Markdown, on request) excludes `--result-fields`, `--save-result`, `--format`, `--json`, `--jq`.
- **Calendar agenda:** `calendar +agenda --start <ISO-8601> --end <ISO-8601> --as user`; default today.
- **Calendar create:** `calendar +create --summary <title> --start <ISO-8601> --end <ISO-8601> --attendee-ids <ou_id,...> --as user`.
- **Tasks list:** `task +get-my-tasks --as user`; add `--complete false` for open tasks.
- **Task create:** `task +create --summary <title> --due date:<YYYY-MM-DD> --as user`. **Task complete:** `task +complete --task-id <guid> --as user`.
- **OKR read:** `okr +cycle-list --user-id <ou_id> --as user`, then `okr +cycle-detail --cycle-id <id> --as user`.
- **Wiki node:** `wiki +node-get --node-token <token-or-url> --as user`.

## Other

Run `discover <domain> <keywords> --result-fields command,plan --limit 3`: local flags, guidance and API inputs. Literal terms; shorten empty queries. Globals: `discover <command> --flags`. Detail: `skills read lark-<domain>` with `--outline`, then `--section <slug>`; follow references as needed. `schema <service.resource.method> --input-only` inspects a known API.

Do not pre-read `lark-shared` or run auth checks. Read `feishu-cli-next skills read lark-shared` after auth/scope errors or before high-risk writes. Browser authorization needs the user.

## Rules

- Keep large results in files (`--output`, `--save-result`, or redirect stdout). Parse/filter/join with a local script in the same call after a successful export; print needed facts/completeness. Reuse guidance already read for this version.
- Group independent reads in one shell call; check each exit/`ok`. Gate writes on verified inputs. Batch writes when supported; check every item.
- Check exit code, JSON `ok`, IDs and completeness; stop on partial receipts.
- Preserve IDs, formatting, and omitted/null/empty/zero/false values exactly.
- Never retry a mutation whose outcome is unknown; inspect the target first.
