# base

## +record-upsert

Use for an explicitly selected record or a new row. With an existing record, `--skip-unchanged` can avoid unnecessary writes when the extra read scope is available. The comparison does not prevent concurrent edits. Omitted fields remain untouched; a new invocation without a record ID creates another row.

## +record-batch-update

Use when known record IDs need different field changes. `--skip-unchanged` compares all selected records before any write and leaves equal rows unchanged. A successful batch envelope alone does not prove that every record exists or every field applied; retain ignored fields and inspect returned receipts before recovery.

## +record-import

Use to append supported primitive values from a complete typed Sheets result artifact without moving the row payload through model context. Map every source column to an existing field explicitly and use `--output` to retain the created IDs in a file. This creates new rows every invocation.

### Prerequisites

The artifact and current operation must have matching verified account and tenant evidence. Resolve destination fields through [[+field-get]] or [[+field-list]] when their identities or types are unknown.

### Avoid when

Source cells contain formulas, date styles, display formatting, rich content or identity values: the conversion rejects these rather than discarding them. Numeric decimals that cannot round-trip through Base double storage are also rejected. Keep the original artifact or choose an explicit supported conversion outside this command.

### Recovery

An unknown batch stops the import. The report distinguishes confirmed created IDs from the first unconfirmed source row; that position does not authorize replay. If report publication fails, full observed effects remain on stdout. Read existing records and inspect receipts before deciding how to continue.

## +record-list

For analysis, NDJSON keeps full typed rows in a file. Check continuation and query extent before treating an export as complete; multiple pages are not a snapshot. Duplicate field names or a field named `record_id` cannot be represented by the row-object format and are rejected. JSON preserves the native matrix for those schemas.

## +field-create

Use native field JSON for full schema configuration. Formula and lookup definitions retain their reference acknowledgement requirement; read the specific protocol on demand with `feishu-cli-next skills read lark-base references/lark-base-field-formula.md` or `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`, then acknowledge it for that command.

## +field-update

This replaces the complete field definition and can change an entire column. Retain the writable configuration that must survive. For formula or lookup protocol details, use the same executable `skills read` references as field creation; the source checkout and an installed legacy skill are not prerequisites.
