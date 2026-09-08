# docs
> skill: lark-doc

## +create
Create a new Lark document from DocxXML or Markdown, optionally in a folder or Wiki node.

### Tips
- Match `--doc-format` to `--content`: XML is the default for rich DocxXML; use `--doc-format markdown` for Markdown input.
- XML preserves native rich structure; the XML guide describes supported blocks and reference maps when authoring unfamiliar content.
- For multiline `--content`, prefer `@file` or `-` (stdin) to avoid shell-escaping damage.

### Skills
- lark-doc/references/lark-doc-create-workflow.md
- lark-doc/references/lark-doc-create.md
- lark-doc/references/lark-doc-xml.md
- lark-doc/references/lark-doc-md.md

## +fetch
Read an entire Lark document, or limit the result to an outline, section, block range, or keyword match.

### Tips
- Use XML `full` detail when native IDs, styles and reference maps matter; Markdown and IM Markdown are explicit conversions.
- Bounds require a matching scope; `full` with range or keyword bounds fails instead of silently returning the entire document.
- `--save-result` saves the full response; the generic result artifact is not a document import payload. Preserve `document.reference_map` alongside `document.content`.

### Skills
- lark-doc/references/lark-doc-fetch.md

## +update
Apply targeted text or block edits, append content, or deliberately replace an entire Lark document.

### Tips
- Prefer `str_replace` or `block_*` commands for targeted edits. Use `overwrite` only when replacing the entire document is intended; it can discard unrelated rich content.
- For one occurrence, use the smallest `--pattern` established unique across the document; leave unchanged styled text outside the match. If uniqueness is uncertain, use a native block edit addressed by ID, preserving unrelated block IDs and resources.
- When the match includes styled text, copy its actual XML marks and link `href` into replacement content and change only the intended characters; plain replacement can remove existing formatting.
- After a targeted edit, fetch the affected native XML and verify text, styles and link destinations before reporting success.
- Native block IDs come from `docs +fetch --detail with-ids`; structural edits can invalidate earlier addressing.
- XML preserves native rich structure; the XML guide describes supported blocks and reference maps when authoring unfamiliar content.
- Match `--doc-format` to `--content`; for multiline content, prefer `@file` or `-` (stdin).

- XML overwrite first compares a full native snapshot and skips an exact unchanged body/reference map without a write. Markdown equality cannot prove rich-content equivalence.
- A revision is forwarded as the base revision; local receipts and readback do not prevent concurrent edits by other clients.
- After partial or unknown effects, inspect the returned receipts and fetch the affected content before deciding whether another write is needed.

### Skills
- lark-doc/references/lark-doc-update.md
- lark-doc/references/lark-doc-xml.md
- lark-doc/references/lark-doc-md.md

## +history-list

### Skills
- lark-doc/references/lark-doc-history.md

## +history-revert

### Tips
- A returned task ID can represent pending work. Poll [[+history-revert-status]]; do not replay the revert after an unknown response.

### Prerequisites
- `history_version_id` from [[+history-list]]

### Skills
- lark-doc/references/lark-doc-history.md

## +history-revert-status

### Prerequisites
- `task_id` from [[+history-revert]]

### Skills
- lark-doc/references/lark-doc-history.md
