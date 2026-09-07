---
name: lark-note-concise
description: Read Feishu intelligent notes, their document links, and normal or unified transcripts from a known note ID or linked document. Meeting discovery and Minutes use their own commands.
---

# Intelligent notes

Note is an AI meeting-summary collection identified by `note_id`. Its summary
and shared documents use Doc tokens. A normal transcript is a separate Doc;
a unified transcript is fetched through Note. Minutes are recording artifacts
identified by `minute_token`; Calendar `meeting_note` is a manually linked Doc.
These identifiers are strings and cannot substitute for one another.

Keep the identity that produced an ID across subsequent Note, Docs and Drive
reads. If none is established, use the user's requested identity, otherwise
explicit `--as user` unless the task specifically requires an application bot.
`note +detail` supports user and bot; `note +transcript` supports only user.
If the next command does not support the established identity, explain the
limitation; switch only with the user's agreement. Bot access also depends on
permission to view the note's main document. Both Note reads require
`vc:note:read`. Consult `lark-shared` for actual authentication, token, identity
or scope errors; no authentication preflight is needed for ordinary reads.

Use a supplied `note_id`, the one returned by `vc +detail` or `minutes +detail`,
or an explicit `vc-node-id` in a Doc's `<vc-transcribe-tab>`. Do not infer it
from a title, Doc token, backlink or another kind of ID. With only a document
URL/token, read it as a document; a Minutes URL can lead through
`minutes +detail` to its returned `note_id`. If no explicit Note ID is found,
continue treating the source as a Doc.

```bash
lark-cli note +detail --note-id <note_id> --as <source_identity>
```

The result contains `note_display_type`, `note_doc_token`,
`verbatim_doc_token`, creator/time metadata and any `shared_doc_tokens`.
If only links or IDs were requested, return those without fetching content.
For names and URLs, `drive metas batch_query` accepts up to ten document tokens
per request:

```bash
lark-cli drive metas batch_query --data '{"request_docs":[{"doc_type":"docx","doc_token":"<doc_token>"}],"with_url":true}' --as <source_identity>
```

Read a summary or other note-body content from `note_doc_token`:

```bash
lark-cli docs +fetch --doc <note_doc_token> --doc-format markdown --as <source_identity>
```

When reading the body, its first `<whiteboard token="...">` is the note cover.
If present, download it with `docs +media-download --type whiteboard --token
<token> --output ./notes/<note_id>/cover --as <source_identity>` and present it
with the body. Missing cover is fine; other whiteboards are not automatically
covers. Shared documents use Docs for content or Drive for metadata. Let the
user choose among multiple shared documents unless they requested them all;
preserve individual tokens and report each unavailable document separately.

For a normal or unknown display type with a `verbatim_doc_token`, read that
Doc using `docs +fetch --doc <verbatim_doc_token> --doc-format markdown` and
the established identity. Unknown type without a transcript Doc token has no
confirmed transcript route; explain that instead of guessing unified.
For a confirmed unified note with user identity:

```bash
lark-cli note +transcript --note-id <note_id> --as user
```

The command checks display type, fetches all transcript pages and saves the
combined content. A page failure aborts without saving partial content.
Default output is `./notes/<note_id>/unified_transcript.md`.
`--transcript-format plain_text` uses `.txt`; `--output` selects a path.
`--locale` accepts a requested locale such as `zh_cn`, `en_us` or `ja_jp`;
otherwise it follows profile language or brand. An existing output fails;
add `--overwrite` only when the user requested replacement. Read the saved file
when the task needs its content. Transcript content format is separate from
CLI `--format` output formatting. `--dry-run` previews the request without
fetching content. Help is available when exact options are needed.

Use the note body for existing AI summaries, and actual transcript content for
independent analysis or attribution. Do not assume a fixed speaker/timestamp
line format across normal Docs, unified Note transcripts and Minutes.

This is concise English coverage of the official `lark-meeting` Note scene
and its `lark-note-detail` / `lark-note-transcript` references. The official
`lark-note` skill is a compatibility entry that delegates to `lark-meeting`.
