# sheets

## +data-export

Use a native cell artifact for exact range transfer when DataFrame or CSV conversion would lose per-cell types, formulas, field presence, or number formats.

### Tips
- The explicit rectangle is read completely in bounded row pages. Missing rows, missing columns, and truncation fail the export; completeness applies to that rectangle, not the entire sheet.
- Native JSON keeps omitted values, null, empty text, leading zeros, decimal digits, formula fields, and date/time serials with their number formats distinct. Backend precision and formula metadata remain the backend's evidence; no timezone or snapshot consistency is inferred.
- `--save-result` stores the complete payload with account/environment binding. Use [[+data-import]] for Sheets or Base's native artifact importer with explicit column mapping for supported conversions.

### Examples

**Save an explicit range as a bound artifact**
```bash
feishu-cli-next sheets +data-export --spreadsheet-token sht_example --sheet-id sheet_example --range A1:D10000 --save-result ./cells.json
```

## +data-import

Use a bound native cell export to apply supported cell fields at an explicit destination. Unchanged cells produce no backend writes.

### Tips
- Imports require the same verified account, application, environment, and tenant binding. Choose the destination token, sheet ID, and equally sized rectangle explicitly.
- Missing fields preserve existing data. Changed null values need an explicit clear operation; unsupported conversions fail before any write. Static text beginning `=` is sent as an explicit text segment so it cannot become a formula.
- Formula source is written without its cached value. Date/time serials and formats are forwarded without inventing a timezone. CSV and DataFrame views have their own conversion semantics and are not native artifacts.
- Reads and sequential writes have no snapshot lock, compare-and-swap, or rollback. After interruption or an unknown outcome, inspect receipts and read the affected range before choosing a retry.

### Examples

**Apply a saved range at a new location**
```bash
feishu-cli-next sheets +data-import --spreadsheet-token sht_example --sheet-id sheet_example --range A1:D10000 --artifact @./cells.json
```
