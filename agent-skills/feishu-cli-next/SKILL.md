---
name: feishu-cli-next
description: Use the Feishu/Lark CLI for documents, contacts, messages, calendars, Base, sheets, files and other workspace resources.
---

# Feishu CLI Next

Use `feishu-cli-next` followed by the routes below. Supplied syntax is sufficient for these workflows; consult help only for missing flags or requirements. Use the authorized identity: examples show `--as user`; bot permissions differ. Preserve native IDs and resolve ambiguous targets before sending, sharing or changing resources.

## Direct routes

- **Known user ID:** `contact +get-user --user-id <id> --user-id-type open_id --as user` for basic details; also supports bot identity. For richer user-identity fields or several open IDs, `contact +search-user --user-ids <id,id> --as user`.
- **Find people/bots:** `contact +search-user --query <name-or-email> --as user` or `contact +search-bot --query <keyword> --as user`. Match the requested discriminators. For `has_more`, refine the query/scope; these searches do not paginate.
- **Read Docs:** `docs +pull --doc <URL-or-token> --as user`. Answer directly from returned Markdown/native block anchors. Unsupported regions, comments and reference maps use `docs +fetch --doc <doc> --detail full`; bounded reads use its scope flags.
- **Edit Docs wording:** `docs +pull --doc <doc> --output ./note.md --as user && cat ./note.md`; edit `note.md` locally; then `docs +push --file ./note.md --as user`. Keep the sidecar, anchors, Markdown marks and links unchanged. Edit the smallest text span within existing runs. When `data.verified` is true, use `data.document_id` and `data.changed_blocks[].text` (complete plain paragraphs) for confirmation/answers; another fetch is unnecessary. Ambiguous cross-run or structural changes use `docs +update` and its reference. Pull afresh for later sessions.
- **Import Markdown:** `docs +create --doc-format markdown --content @./source.md --as user`; optionally specify title/destination. Preserve the source; verify the returned document with `docs +fetch --doc <id> --detail full`.
- **Base rows:** `base +record-get --base-token <base> --table-id <table> --record-id <id> --format json`. For one existing row, `base +record-upsert` takes the same base/table/record flags plus `--json '{"<field>":<value>}'`; omitting the record ID creates. Check `+field-list` if writability/types are unknown; verify affected IDs after writes. Several rows use `+record-batch-update`.
- **Exact sheet export:** `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <rectangle> --save-result ./cells.json`. It validates native types, formulas, completeness and revisions. Check success/artifact reference; inspect cells only when needed. CSV is potentially lossy.
- **Drive bytes:** `drive +upload --file ./source --folder-token <folder> --as user`; omit folder for root. After success, `drive +download --file-token <data.file_token> --output ./download --as user`. Compare local bytes/hashes without printing contents.

## More functionality and safeguards

All original commands, formats, skills, generated APIs, raw `api` and events remain available. Unknown command: `discover <keywords> --flags --limit 3`; exact API shape: `schema`; domain guides: `skills list` and `skills read <name> [path]` (Docs: `lark-doc`). Use references for missing details, not routine preflight. Base field acknowledgements: `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` and `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`.

Read small stdout directly; use files/stdin for substantial content and saved artifacts when required. Preserve formatting and distinctions between omitted, null, empty, zero and false. Guard dependent steps with exit status, JSON `ok`, IDs and completeness/partial receipts. Unknown mutations require reconciliation before retrying; snapshot checks are not locks. Follow actual auth/scope errors; browser authorization needs the user. Preserve audience/destination; overwrite only when intended.
