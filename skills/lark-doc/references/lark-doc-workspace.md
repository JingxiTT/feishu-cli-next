# Local document wording edits

Use `docs +pull` to read native paragraphs and headings as Markdown with block
anchors. With no output file, read the returned view directly. Unsupported rich
regions are identified explicitly; use `docs +fetch --detail full` for their
native representation, comments and reference maps.

For a local editing session:

```sh
feishu-cli-next docs +pull --doc <document> --output ./note.md --as user
# Read note.md, then edit its wording with local tools.
feishu-cli-next docs +push --file ./note.md --as user
```

Keep the sidecar alongside the Markdown file. It contains the original native
blocks and document/account binding. Do not edit the sidecar, block anchors,
Markdown formatting or unsupported regions. Local files contain document data;
the CLI does not upload the sidecar or replace the entire document.

This workflow supports unambiguous text changes within existing text runs. It
preserves native block IDs, existing styles, link destinations and unchanged
elements. Changes to structure or formatting, and text changes spanning
ambiguous style boundaries, require the existing `docs +update` workflow.
An unsupported local change is rejected before any remote write.

Push plans all changes, checks the saved native state, applies changed blocks,
and reads back the result. A detected remote change stops the push. The native
revision parameter is not compare-and-swap: the server accepts stale revisions.
Another client can still edit between the check and the write, so this workflow
does not provide an atomic conflict check or safe automatic merge.
Use the returned verified paragraphs when they
satisfy the task; another manual readback is unnecessary. An unchanged workspace
does not produce mutations. Multiple writes are not a transaction: if one fails,
inspect the reported completed and unresolved blocks before deciding recovery.
Do not blindly retry a mutation after a timeout.

The saved workspace remains the original base. Pull to a new path before a later
editing session. Existing output files are protected from replacement. A local
snapshot and a pre-read are not themselves a concurrency lock.
