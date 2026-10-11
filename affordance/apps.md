# apps

## +export
Use a source snapshot for inspection or archiving, including a shared app you may download.

### Avoid when
- Continuing local development → use [[+init]] for a Git workspace.

### Tips
- Exports the last commit, excluding uncommitted sandbox work; possession of a share token does not grant download permission.
- The native ZIP is saved at `output`. Generic saved results describe that file; they do not contain its source archive.

### Skills
- lark-apps/references/lark-apps-export.md

## +release-create

### Tips
- Use the known app type: `html` omits the reason; `frontend` and `full_stack` use the confirmed reason. Follow the release reference when preparing it.
- Keep the returned ID and query the same release_id with [[+release-get]]; creating another release is not a status check.

### Skills
- lark-apps/references/lark-apps-release-create.md

## +release-get

### Tips
- Terminal status wins over stale node data: `finished` establishes deployment and its returned `online_url`; `failed` uses the returned error evidence. No follow-up app listing is needed.
- Before completion, `current_node_info.current_status=PENDING` means stop polling and hand off to the configured approval owner. Read the release reference for approval-link handling; do not approve or guess the approver.
- Poll the same release only while publishing without a pending approval. Stop on top-level pending or unknown status; report the observed state without creating another release.

### Skills
- lark-apps/references/lark-apps-release-get.md

## +update
Use a metadata edit when the target app already exists.

### Tips
- Only provided nonempty fields are patched; omitted fields remain unchanged. The default path needs only write scope. Opt into --skip-unchanged to read the app and avoid an identical replacement; this additionally needs read scope. The read and patch are separate requests, so this does not prevent concurrent edits.

### Examples
**Change an app's name**
```bash
feishu-cli-next apps +update --app-id app_fixture --name "Journal"
```

## +db-data-export
Use native CSV or JSON output when rows will be handed to [[+db-data-import]].

### Tips
- Inspect complete, rows_counted, source_total and has_more before treating the file as the whole table. The source total is observed separately from the export; it is not an atomic snapshot. SQL dump row counts are estimates.
- A generic saved result describes the export; the native file at output contains the importable rows.

### Examples
**Export a bounded native JSON file**
```bash
feishu-cli-next apps +db-data-export --app-id app_fixture --table orders --output orders.json --limit 100
```

## +db-data-import
Use a native CSV or JSON data file after checking its intended table and environment.

### Tips
- Generic result artifacts cannot be imported. Use the native output from [[+db-data-export]] and inspect its completeness before proceeding.
- If the import outcome is unknown, inspect the target rows before repeating the write; automatic replay could duplicate rows.

### Examples
**Preview a native import request**
```bash
feishu-cli-next apps +db-data-import --app-id app_fixture --file orders.json --table orders --dry-run
```

## +env-list
Use the default redacted view for environment inventories and saved artifacts.

### Tips
- Explicit value output can contain credentials and cannot be combined with a generic saved result. Request values only when needed for the current operation.

### Examples
**Inspect environment metadata**
```bash
feishu-cli-next apps +env-list --app-id app_fixture
```

## +file-upload
Use this workflow to upload bytes and register their file metadata.

### Tips
- Storage upload and metadata registration are separate effects. If the response is lost or registration fails, inspect the app's files before repeating the upload; bytes may already exist.

### Examples
**Preview an upload**
```bash
feishu-cli-next apps +file-upload --app-id app_fixture --file logo.png --dry-run
```

## +html-publish
Use this workflow for an HTML app after inspecting the local publication payload.

### Tips
- A stored archive does not prove deployment succeeded. Inspect [[+release-get]] after release creation; if an upload response is lost, inspect existing state before repeating publication.

### Examples
**Inspect the publication payload**
```bash
feishu-cli-next apps +html-publish --app-id app_fixture --path site --dry-run
```
