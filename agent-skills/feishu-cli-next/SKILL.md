---
name: feishu-cli-next
description: Work with Feishu/Lark documents, messages, calendars, spreadsheets, Base, files, and other workspace resources through the feishu-cli-next CLI.
---

# Feishu CLI Next

Use `feishu-cli-next` for Feishu/Lark workspace operations. All inherited commands, generated OpenAPI methods, raw API access, events, and domain skills remain available. Start with a known command; discovery, help, and authentication checks are for missing information, not prerequisites to every operation. Combine independent reads or help lookups in one shell tool call when useful; keep dependent operations and mutations ordered.

## Common routes

The forms below follow the executable name; replace placeholders with the task's actual inputs. Use the identity authorized for the task. These are starting routes, not a restriction on other commands or compositions.

| Intent | Direct route and decision |
| --- | --- |
| Known user ID | `contact +get-user --user-id <id> --user-id-type open_id --as user` (also supports `--as bot`; match the ID type). For several known open IDs under user identity, `contact +search-user --user-ids <id,id> --as user`. |
| Find people or bots | `contact +search-user --query <keyword-or-email> --as user`; `contact +search-bot --query <keyword> --as user`. Check returned discriminators to resolve same-name matches. `has_more=true` requires refining the query/scope; these searches do not paginate. |
| Read a document | `docs +fetch --doc <URL-or-token> --detail with-ids` returns text with native block addresses. Use `simple` for text only, `full` for styles, references, and edits. XML is the native default; choose the needed detail in the first read. |
| Edit document text | Read `full`, then `docs +update --doc <doc> --command str_replace --pattern <exact-text> --content <replacement>`. Include existing bold/link markup in the replacement when it intersects styled text. For multiline or structural edits, inspect `docs +update --help` for `block_*` operations. Preserve the rest of the document and verify the affected content after writing. |
| Import supplied Markdown | `docs +create --doc-format markdown --content @./source.md --as user` imports the existing file directly; `--title` and a parent destination are optional. Preserve the source file and verify native content with `docs +fetch --doc <returned-id> --detail full`. Use the authoring guide for composing new content or unfamiliar rich structures. |
| Read/update known Base rows | `base +record-get --base-token <base> --table-id <table> --record-id <id> --format json` (repeat `--record-id`). If field types/writability are unknown, inspect `base +field-list` using the same base/table flags. `base +record-batch-update` uses those flags plus `--json '{"update_records":{"<id>":{"<field>":<value>}}}'`; submit only intended fields. `--skip-unchanged` reads and compares submitted cells before writing. Verify applied fields by reading affected IDs; success can include ignored fields. |
| Export exact sheet cells | `sheets +data-export --spreadsheet-token <token> --sheet-id <id> --range <A1-rectangle> --save-result ./cells.json` preserves native types, formulas, number formats and field presence. Completeness covers the explicit rectangle. Inspect the artifact locally; CSV is a separate conversion, not a lossless substitute. |
| Transfer Drive file bytes | `drive +upload --file ./source --folder-token <folder>` then `drive +download --file-token <returned-token> --output ./download`. Omit the folder for the caller's root. Compare local bytes/hashes for exact-transfer tasks without printing file bodies. Existing files require explicit overwrite intent; document content operations belong to Docs. |

## Look up only what is missing

For additional flags or command-specific guidance, read the selected command's `--help`. For an unknown command, `discover "contact open_id" --flags --limit 3` matches every keyword literally against paths, aliases, descriptions and public flags; use short keywords and widen the result window only if needed. Explore a known domain with `<domain> --help`. `schema` supplies API input shapes; `api` permits custom OpenAPI requests. Names beginning with `+` are workflow shortcuts.

Read a specific domain reference only when its concepts, syntax or limitations are needed: `skills read <name> [path]`; `skills list` locates other domains. Retained examples say `lark-cli`; invoke `feishu-cli-next`. Docs uses `lark-doc` (references `lark-doc-update.md`, `lark-doc-xml.md`, `lark-doc-create-workflow.md`). Base complex cell values use `skills read lark-base`; formula/lookup field definitions require the guide acknowledgement described by `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` and `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`. All other retained skills remain available on demand.

## Preserve results and intent

`--as user` and `--as bot` have different permissions. Follow concrete errors for missing authentication/scopes; `auth status` inspects login and `auth login` requires the user's browser participation. Native resource IDs differ by type; resolve ambiguity before sending, sharing, assigning, or changing a resource. Preserve the user's audience and destination.

Prefer offered file/stdin flags for substantial content. Keep document structure/reference maps, sheet formulas/types, and Base schema semantics. Omitted fields preserve data where supported; null, empty text, zero and false are distinct. Whole-document overwrite can lose unrelated blocks, IDs and comments; use it only for an intended full replacement.

JSON is compact without dropping fields. `--result-fields` selects result fields while retaining status/completeness metadata; select only when unselected data is unnecessary for the task and verification. `--result-detail` adds indentation. Where supported, `--save-result PATH` saves the full response without replacing an existing file; process large artifacts locally and return the needed evidence. Command-specific `--fields`/`--detail` retain their own meanings.

Check `ok`, exit status, pagination, warnings and partial-failure receipts; success need not contain OpenAPI `code`. A mutation timeout can leave an unknown effect: reconcile the affected resource or use documented backend idempotency before retrying a create, append, send or other non-idempotent write. Local receipts and readbacks do not provide a concurrency lock.
