---
name: lark-minutes
description: "Search, read, download, create and edit Feishu Minutes and manage their permissions. Use minute tokens, Minutes URLs or recording links; route meeting scheduling to Calendar and live meeting interaction to VC."
metadata:
  requires:
    bins: ["lark-cli"]
---

# Minutes

Minutes are recording or uploaded-media artifacts, identified by `minute_token`. They contain summary, todos, chapters, keywords, transcript and original media. They can exist without a meeting. Meeting IDs, nine-digit meeting numbers, Calendar event IDs, Note IDs and Doc tokens identify different objects; never substitute one for another. Preserve IDs as strings, including leading zeros. Calendar `meeting_note` is a user-linked Doc, separate from AI Notes and Minutes.

Use known tokens directly; extract a token from a Minutes URL's `/minutes/<token>` path without its query. Existing meeting context can supply it through `vc +detail` or `vc +recording`; avoid repeating a search. Route scheduling, availability and rooms to Calendar; meeting discovery/recordings to VC; linked intelligent notes to Note and their document content to Docs. A missing Note does not imply missing Minutes or vice versa.

## Identity and permissions

Carry the source identity explicitly as `--as user` or `--as bot` through searches, reads, downloads, permission requests and linked Note/Doc operations. If none is established, use the user's chosen identity; otherwise default to user unless the task explicitly needs the application. Do not switch identities to bypass an ACL. If a command does not support the source identity, explain and stop until the user agrees to switching.

Search, detail, download, generic metadata and permission applications support both identities. Other Minutes writes require user identity. `me` resolves the current user only; bot queries need explicit `ou_` open IDs. Use shared authentication guidance when an actual authentication, identity or scope problem occurs; no preliminary authentication probe is needed.

Distinguish missing OAuth/application scopes from resource ACL denial. User scope recovery uses the indicated `auth login --scope`; bot scope recovery uses application configuration, never user login. Applying for access cannot fix missing scopes. Preserve the identity that encountered the resource denial.

## Search and read

- `minutes +search`: at least one of `--query`, `--owner-ids`, `--participant-ids`, `--start`, `--end`. IDs are comma-separated; query length is at most 50 characters. Page size defaults to 15, maximum 30. Follow `has_more` with the returned `--page-token`. For “my participated Minutes,” union separate owner and participant searches and deduplicate by token unless the user explicitly wants only participant matches. Do not combine the two filters and assume union semantics.
- Dates and ISO timestamps are accepted; timestamps without zones use local time. A date-only `--end` includes that entire day. For a single day, use that same date as start and end. Retrieve all pages for explicit all/count/sort requests; otherwise, after more than 50 results with further pages, confirm whether to continue. Present multiple candidates with title, time, owner and URL for selection.
- `minutes minutes get --minute-token <token>` reads full basic metadata such as title, duration, owner, cover and URL. `--params '{"minute_token":"<token>"}'` is also supported. Stop when metadata satisfies the request.
- `minutes +detail --minute-tokens <tokens>` accepts up to 50 comma-separated tokens. Request only needed `--summary`, `--todo`, `--chapter`, `--keyword`, `--transcript`; without those flags it returns basic information and any linked `note_id`. Artifacts contain summary text and native todo/chapter/keyword arrays. Existing AI summaries answer requests for the saved summary; fresh analysis or independent summarization should use original transcript speech.
- `--transcript` saves text and returns `artifacts.transcript_file`; read that file. Default path is `./minutes/<token>/transcript.txt`; `--output-dir <dir>` uses `<dir>/artifact-<title>-<token>/transcript.txt`. Existing files are preserved unless `--overwrite` is requested. `--wait-ready` waits for processing, with optional timeout/interval seconds. A processing response is not a ready transcript.
- To follow a linked Note, use the returned `note_id` directly with `note +detail --note-id <id>` under the same identity. Follow returned Doc tokens as appropriate; do not pass a minute token to Note or loop back through VC. If no Note is linked, stop that branch.

## Media and creation

`minutes +download --minute-tokens <tokens>` downloads original audio/video, not transcript text. It accepts up to 50 tokens, `--output <path>` or `--output-dir <dir>` (mutually exclusive), `--overwrite`, and `--url-only`. An existing output directory has directory semantics; batch output must not target an existing file. Default files share `./minutes/<token>/` with transcripts. Use returned `saved_path` and `size_bytes`; inspect each batch error. Download URLs last one day; renew expired links. Media requires completed transcription; processing code 2091003 means wait. The API limit is five requests per second; large downloads can be cancelled.

For user-requested media conversion, upload through `drive +upload`, then `minutes +upload --file-token <file_token> --as user`. This workflow uses Minutes rather than substituting local ASR. Accepted audio: wav, mp3, m4a, aac, ogg, wma, amr; video: avi, wmv, mov, mp4, m4v, mpeg, ogg, flv. Source media must be at most 6 GB and six hours. The Minutes command only accepts a Drive token.

An upload response submits asynchronous creation; retain its URL/token. If the user only wanted submission or a link, stop. Otherwise read requested artifacts with `+detail --wait-ready`; do not claim they exist before readiness. Preserve a successful Drive file token after creation failure, and preserve a minute token while processing: resume the failed stage instead of repeating successful uploads or creations.

## Edits

A token alone does not authorize modification. Establish the unique target and requested change. `--dry-run` previews requests. Structured content flags support literal values, `@file` and `@-` where listed below. Use JSON output for processing; command help/schema is available when parameters or behavior are unclear, without a mandatory discovery sequence.

| Need | Command and inputs |
| --- | --- |
| Rename | `minutes +update --minute-token <token> --topic <title>` |
| Replace complete summary | `minutes +summary --minute-token <token> --summary <text-or-@file-or-@->` |
| Single todo | `minutes +todo --minute-token <token> --operation add\|update\|delete`; add needs `--todo` and explicit `--is-done` or `--is-done=false`; update also needs `--todo-id`; delete accepts only the ID. |
| Todo batch | `minutes +todo --minute-token <token> --todos <json-or-@file-or-@->`; ordered objects contain `operation`, add/update `content` and boolean `is_done`, update/delete `todo_id`. Mixed operations are supported; do not combine batch and single-item flags. |
| Word replacement | `minutes +word-replace --minute-token <token> --replace-words '[{"source_word":"old","target_word":"new"}]'`; combine replacements in one array; file/stdin input is supported. |
| Speaker replacement | `minutes +speaker-replace --minute-token <token> --from-speaker-id <speaker_id> --to-user-id <open_id>` |

Read the current summary before replacing it when needed to confirm intent. Any summary text is accepted, but plain text, line breaks, headings 1–3, bold and lists render best; other Markdown may appear literally. Todo content is plain text. Read current todos with `+detail --todo` before updates/deletions and identify exact `todo_id`, not list position. Minutes todos stay in Minutes, even when an owner is requested; do not reroute them to Task.

Todo ownership is represented by literal `@Name` in content. Use an explicitly supplied name directly. For “me,” get the current name with `contact +get-user --as user`; if unavailable, create the requested todo without an owner suffix rather than writing literal `@me` or blocking creation. Do not confuse execution identity with this owner text or describe it as an actual clickable mention.

For word replacement, use supplied exact words directly; no transcript read is required merely to guess spelling or recheck a completed per-word response. Report both Succeeded and Failed sets: the command may succeed when some words miss. Retry only failed words after confirming exact spelling, case and spacing; never repeat successful replacements. All-zero matches are `not_found`, not a permission failure. If outcomes are missing or uncertain, follow the returned recovery guidance before deciding on a retry.

For speakers, first GET `/open-apis/minutes/v1/minutes/<token>/transcript/speakerlist` with `lark-cli api`. Match the intended source; use transcript context and user selection for same-name candidates. Pass opaque `speaker_id`, never its display name. Resolve a target name through Contact to a unique `ou_` open ID. Hidden legacy `--from-user-id` remains compatible, but new workflows use the speaker ID.

Read back modified fields unless the command already reports complete per-item outcomes. Report partial/multistep results without assuming rollback or repeating successful effects. ASR/AI quota code 2091008 cannot be fixed by scope changes or repeating the request: direct the user to Minutes quota details. Upload quota failure means no minute was created; summary/todo/word quota failures concern incomplete generated artifacts.

## Access management

`minutes +apply-permission --minute-token <token> --perm view|edit --as <source_identity>` notifies the owner with an application for the current identity. Run only when the user explicitly requests that application. Submission is not a grant; user and application access are separate.

To inspect collaborators, use `drive +member-list --token <minute_url>`. To grant another member access, use `drive +member-add --token <minute_url> --member-id <open_id> --member-type openid --perm view|edit --yes` only after target, member, permission and authorization are clear. Bare tokens need `--type minutes`; `full_access` is unsupported. Resolve ambiguous recipients instead of guessing. Verify the requested permission through `member-list` by exact member ID; a successful add receipt does not prove an existing grant was overwritten or downgraded.

Scopes: search `minutes:minutes.search:read`; basic detail `minutes:minutes.basic:read`, selected artifacts additionally `minutes:minutes.artifacts:read`; media `minutes:minutes.media:export`; upload `minutes:minutes.upload:write`; edits `minutes:minutes:update`, with speaker replacement also `minutes:minutes:readonly`; permission applications `minutes:permission:apply`.
