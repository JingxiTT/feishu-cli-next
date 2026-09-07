# minutes
> skill: lark-meeting

## +search

### Skills
- lark-meeting/references/lark-minutes-search.md

## minutes get

## +detail

### Skills
- lark-meeting/references/lark-minutes-detail.md

## +download

### Skills
- lark-meeting/references/lark-minutes-download.md

## +upload

### Skills
- lark-meeting/references/lark-minutes-upload.md

## +update

### Tips
- For repeated replacements, use `--skip-unchanged` to compare the exact current title when read access is available. This is not concurrency protection.

### Skills
- lark-meeting/references/lark-minutes-update.md

## +speaker-replace

### Skills
- lark-meeting/references/lark-minutes-speaker-replace.md

## +word-replace
Batch-replace exact keywords in a minute transcript.

### Prerequisites
- `minute_token` from [[+search]], a minutes URL, or VC recording

### Tips
- Exact source words can be replaced directly. Preserve their spelling, case and spacing.
- Retry only unmatched words after confirming intent; never repeat successful replacements. Missing or unknown outcomes require a read-only transcript check before deciding whether to retry.

### Examples

**Replace several keywords in one call**
```bash
lark-cli minutes +word-replace --minute-token obcnxxxxxxxxxxxxxxxxxxxx --replace-words '[{"source_word":"旧词","target_word":"新词"},{"source_word":"Foo","target_word":"Bar"}]' --as user
```

## +summary
Replace the minute's AI summary text in full.

### Repeated replacements
Use `--skip-unchanged` when read access is available to avoid an identical replacement. The comparison preserves whitespace and does not protect against concurrent edits.

### Prerequisites
- Prefer reading the current summary via [[+detail]] with `--summary` before overwriting

### Examples

**Replace the AI summary**
```bash
lark-cli minutes +summary --minute-token obcnxxxxxxxxxxxxxxxxxxxx --summary "**会议结论**\n- 方案 A 通过\n- 下周跟进排期" --as user
```

### Skills
- lark-meeting/references/lark-minutes-summary.md

## +todo
Write AI todos that live inside a minute — not Lark Task list items.

### Repeated replacements
`--skip-unchanged` applies to updates of distinct todo IDs. Adds and deletes retain their explicit effects; a missing list never implies deletion.

### Avoid when
- Personal or shared Task lists → use the task domain (`lark-cli task`), not this shortcut

### Prerequisites
- `minute_token` from [[+search]], a minutes URL, or VC recording
- For update/delete: `todo_id` from [[+detail]] with `--todo`

### Examples

**Add one unfinished todo**
```bash
lark-cli minutes +todo --minute-token obcnxxxxxxxxxxxxxxxxxxxx --operation add --todo "跟进预算审批" --is-done=false --as user
```

### Skills
- lark-meeting/references/lark-minutes-todo.md

## +apply-permission

### Skills
- lark-meeting/references/lark-minutes-apply-permission.md
