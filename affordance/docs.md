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

## +pull
Use a readable local view for document facts or wording changes while keeping native block references.

### Tips
- Read stdout directly for small fact questions. Save a workspace when local editing or a file is needed; keep the Markdown file and its native sidecar together.
- Unsupported rich blocks remain visible as immutable regions. Use [[+fetch]] for their full native content, comments and reference maps.
- Existing Markdown marks and block anchors identify the original native content; changing them is not a supported text edit.

### Skills
- lark-doc/references/lark-doc-workspace.md

## +push
Apply wording edits from a saved local view while preserving the existing native block identities and formatting.

### Tips
- Edit text inside existing paragraphs or headings; leave anchors, structure and Markdown formatting intact. Ambiguous edits across text runs are rejected before writing; use [[+update]] for richer edits.
- The command checks the saved native state and verifies applied changes internally. Use a successful verified result without repeating the same readback.
- Known remote changes stop the push. Feishu accepts stale revision parameters, so another client can still edit between the check and write; this is not an atomic conflict check.
- A workspace is a base snapshot, not a live mirror. Pull a new file for further work; after partial or unknown effects, reconcile the returned block outcomes before another mutation.

### Skills
- lark-doc/references/lark-doc-workspace.md

## +fetch
Read an entire Lark document, or limit the result to an outline, section, block range, or keyword match.

### Tips
- For facts and native block addresses, choose XML `with-ids`; use `full` when styles and edit/reference metadata are needed. Markdown and IM Markdown are explicit conversions.
- Bounds require a matching scope; `full` with range or keyword bounds fails instead of silently returning the entire document.
- Read small fact-lookup results directly from JSON stdout. Use `--save-result` for requested exports or large responses, preserving `document.reference_map` alongside `document.content`; the artifact is not a document import payload.

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
