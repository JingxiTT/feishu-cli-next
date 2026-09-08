---
name: feishu-cli-next
description: Use the Feishu/Lark CLI for documents, contacts, messages, calendars, Base, sheets, files and other workspace resources.
---

This entry guide is complete in context. After any required skill-file read, run `feishu-cli-next` with these routes directly; consult help/references for missing requirements. Select authorized identity (`--as user` below; bot permissions differ).

- **Profile:** `contact +get-user --user-id <id> --user-id-type open_id --as user` (also bot). Richer/batched user profiles: `contact +search-user --user-ids <id,id> --as user`.
- **Search:** `contact +search-user --query <name-or-email> --as user` / `contact +search-bot --query <keyword> --as user`. Match discriminators; refine `has_more` results (no pagination).
- **Docs read:** `docs +pull --doc <URL-or-token> --as user`; answer from Markdown/anchors. Unsupported regions/comments/maps: `docs +fetch --doc <doc> --detail full`; scope flags bound reads.
- **Docs edit:** `docs +pull --doc <doc> --output ./note.md --as user && cat ./note.md`; edit locally; `docs +push --file ./note.md --as user`. Preserve sidecar/anchors/marks/links; change minimal text within existing runs. Only when `data.verified=true`, answer from `data.document_id` and `data.changed_blocks[].text` (complete plain paragraphs); no extra fetch. Ambiguous cross-run/structural edits: `docs +update`, `skills read lark-doc`. Pull afresh later.
- **Markdown import:** `docs +create --doc-format markdown --content @./source.md --as user`; preserve source; verify with `docs +fetch --doc <id> --detail full`.
- **Base:** `base +record-get --base-token <base> --table-id <table> --record-id <id> --format json`. Update: `base +record-upsert`, same identifiers plus `--json '{"<field>":<value>}'`; omitted record ID creates. Unknown types/writability: `+field-list`. Verify affected IDs; multirow: `+record-batch-update`.
- **Sheets:** `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <rectangle> --save-result ./cells.json`. Check success/artifact; inspect cells when needed. CSV may lose types/formulas.
- **Drive:** `drive +upload --file ./source --folder-token <folder> --as user` (omit folder for root); then `drive +download --file-token <data.file_token> --output ./download --as user`. Compare bytes/hashes locally.

All original commands/formats, generated APIs, raw `api`, events and guides remain: `discover <keywords> --flags --limit 3`, `schema`, `skills list`, `skills read <name> [path]`. Base acknowledgements: `feishu-cli-next skills read lark-base references/lark-base-field-formula.md`; `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`.

Read small stdout; use files/stdin for substantial content. Preserve formatting/IDs and omitted/null/empty/zero/false. Gate dependent steps on exit, JSON `ok`, IDs/completeness/partial receipts. Reconcile unknown mutations before retry; snapshots aren't locks. Resolve ambiguity; preserve audience/destination; overwrite only intentionally. Follow auth/scope errors; browser authorization requires user participation.
