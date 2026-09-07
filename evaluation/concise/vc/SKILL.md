---
name: lark-meeting
description: "Read historical or live Feishu meetings, meeting notes and recordings; manage Minutes; operate an authorized meeting bot and in-meeting interactions. Use Calendar for scheduling, availability and rooms."
metadata:
  requires:
    bins: ["lark-cli"]
---

# Meetings, notes and recordings

Use the available command directly when its inputs are known. Consult relevant command help or schema when a parameter, response field or constraint is missing; do not require a discovery sequence. Auth checks are not a prerequisite for ordinary work. Use shared authentication guidance for authentication requests or actual token, identity or scope errors.

## Identity and identifiers

Carry the identity that produced each resource ID through the workflow, explicitly passing the same `--as user` or `--as bot` to subsequent VC, Minutes, Note, Doc and Drive commands. With no existing identity, follow the user's choice; otherwise use user identity except for operations explicitly involving the app bot. If a command does not support the source identity, explain and stop that branch. Change identity only with explicit user agreement; never change it to bypass an ACL or make an error disappear. Describe identities to users as “user identity” and “app identity.”

Keep IDs as strings. A nine-digit `meeting_no` belongs in `--meeting-number`; it is not the long numeric `meeting_id`. A lowercase alphanumeric `minute_token` comes from a Minutes URL's last path segment, excluding query parameters. Calendar `event_id`, Note `note_id`, Minutes token and Doc tokens identify different objects and cannot substitute for one another.

Calendar events can schedule a meeting, but are not the meeting record. Their manually attached `meeting_note` Doc is independent of AI Note. AI Note and recording-derived Minutes are separate, optional branches: either, both or neither may exist. Uploaded media can produce Minutes without any meeting or event. Never infer one ID from another token or a document title.

## Find and read meeting records

| Need | Command and decision |
| --- | --- |
| Historical meetings | `vc +search` with a real keyword/meeting number, time, organizer, participant or room filter; at least one filter is required. |
| Known long ID | Use it directly; do not repeat search. |
| Historical nine-digit number | Resolve each number separately with `vc +search --query <number> --format json`; retain the result's `id` and source identity. Normalize all input identifiers before batching. |
| Calendar event | `calendar +meeting` supplies meeting links/IDs and the distinct manual `meeting_note`. |
| Meeting metadata or participant history | `vc meeting get --params '{"meeting_id":"<id>","with_participants":true}'`; omit participant expansion when unnecessary. This is a server snapshot and does not require bot attendance. |
| Associated Note/Minutes IDs | `vc +detail --meeting-ids <ids>`; comma-separated batches of at most 50. |
| Recording link or Minutes bridge | `vc +recording --meeting-ids <ids>` or `--calendar-event-ids <ids>`, exclusively; at most 50. Its recording `duration` is milliseconds. |

Search is for ended meetings. For today's complete schedule, combine ended records with Calendar's current/upcoming events. A natural-language note title without meeting clues belongs in Drive/Doc search, not meeting keyword search. Do not turn action words such as “summarize” into search keywords or infer decisions from meeting titles.

VC and Minutes search use JSON, page size 15 by default, maximum 30, and returned `has_more`/`page_token`. A larger page size does not establish completeness. Follow pages automatically; when the user has not requested all results, ask before continuing beyond 50 accumulated results while more remain. Explicit all/statistics/sorting intent authorizes complete pagination and deduplication. Split searches longer than one month into monthly windows. Convert relative dates to explicit dates; a date-only `--end` includes that day's 23:59:59, so a single-day query uses the same start/end date. ISO times without an offset use the local timezone.

When several meetings match, show topic, time, organizer and IDs for selection; do not guess the latest. Stop once the requested IDs or links are supplied. Recording generation must finish first; recordings shorter than five seconds may yield no file. Report missing, generating or inaccessible artifacts while retaining known IDs.

If both Note and Minutes exist, respect the user's requested branch; otherwise prefer Note. Use the available branch when only one exists. Do not automatically merge overlapping AI summaries. Return existing AI summaries/todos/chapters when requested; for independent analysis, retrospective or “who said what,” read original transcripts and analyze them rather than rewriting an AI summary as new evidence. If transcripts are unavailable, state the evidence limit.

## Note and Doc routing

`note +detail --note-id <id>` supports user and app identities and supplies `note_display_type`, `note_doc_token`, `verbatim_doc_token` and `shared_doc_tokens`. A valid Note ID must be explicit or returned by VC/Minutes detail or a Doc's `<vc-transcribe-tab vc-node-id="...">`. A Doc token, backlink or title is not a Note ID.

For a Note Docx URL, `docs +fetch --doc <url> --doc-format markdown` can reveal the explicit vc-node-id, “transcript” Docx link, Minutes URL or cited shared Docs. If only a Minutes URL appears, query `minutes +detail` for its top-level `note_id`; absent that ID, continue as a Doc. Preserve the user's original document URL when returning it.

Read Note body with `docs +fetch --doc <note_doc_token> --doc-format markdown`. Its first `<whiteboard token="...">`, if present, is the cover: download via `docs +media-download --type whiteboard --token <token> --output ./notes/<note_id>/cover` and show it with the body. Absence is not failure; do not download every whiteboard.

Transcript routing follows display type:

- `normal`, or `unknown` with `verbatim_doc_token`: read that Doc.
- `unified`: `note +transcript --note-id <id> --as user`. App-origin workflows need explicit agreement before switching. The command paginates completely and does not save partial transcripts after a page failure.
- `unknown` without a transcript Doc: explain that the route is unresolved; do not guess unified.

Unified transcripts default to `./notes/<note_id>/unified_transcript.md`; `--transcript-format plain_text` selects `.txt`. Use `--overwrite` only when replacement is authorized. Note and Minutes transcript formats are distinct; do not assume a fixed speaker/timestamp serialization.

For names/URLs only, use `drive metas batch_query` with `request_docs`, `doc_type:docx` and `with_url:true`; shared Doc metadata batches contain at most 10 tokens. Fetch bodies only when requested. For multiple shared Docs, show choices unless the user requests all; preserve individual inaccessible tokens and failures. Calendar's manual note is not a Note shared Doc.

## Minutes reading and editing

Reuse a known token or meeting-derived token. Otherwise `minutes +search` supports keyword, `--owner-ids`, `--participant-ids` and time filters, with the same search pagination/date rules above. `me` is user-only. “My Minutes” means owned; “Minutes I participated in” normally means separate owner and participant searches, unioned by token. A strictly participant-only request uses that filter alone. App identity needs explicit open IDs.

- Metadata such as title, duration, owner, cover or URL: `minutes minutes get --params '{"minute_token":"<token>"}'`.
- Artifacts: `minutes +detail --minute-tokens <tokens>` with only the desired `--summary`, `--todo`, `--chapter`, `--keyword`, `--transcript`; up to 50 tokens. Without these flags it returns basic information and a possible top-level `note_id`, not all artifacts. Follow that Note ID directly rather than searching VC again.
- Media: `minutes +download --minute-tokens <tokens>`; up to 50. `--url-only` gives a one-day link; transcription must be ready. The export endpoint allows five requests/second. `--output` names a single file (an existing directory behaves as a directory); `--output-dir` handles batches, and the two are exclusive. Default media and transcript files share `./minutes/<token>/`; use returned paths and authorized overwrite only. Detail's `--output-dir` uses its documented artifact-title/token layout.

Creation and edits require explicit intent, a unique target and the intended content; providing a token is not authorization to change it. Minutes creation/content edits require user identity; permission requests support either identity. Preserve a completed step when a later step fails; never recreate successful uploads or automatically undo successful batch items.

For local audio/video, use `drive +upload` to obtain a `file_token`, then `minutes +upload --file-token <token> --as user`; do not substitute local ASR. Source limits: six hours, 6 GB; audio wav/mp3/m4a/aac/ogg/wma/amr, video avi/wmv/mov/mp4/m4v/mpeg/ogg/flv. Creation returns a URL/token before processing finishes. Report submission, not ready artifacts. If immediate artifacts are requested, use `minutes +detail --wait-ready` with requested flags; poll processing status without recreating. If only submission or the link was requested, stop there.

| Edit | Guidance |
| --- | --- |
| Title | `minutes +update --minute-token <token> --topic <title>` |
| AI summary | `minutes +summary --minute-token <token> --summary <text-or-@file-or-@->` replaces the whole summary. Prefer plain text, headings 1–3, bold and lists; unsupported Markdown remains literal rather than causing rejection. |
| AI todos | `minutes +todo` is not a Task operation. Read current `--todo` artifacts to resolve exact IDs before update/delete. Single `--operation add|update|delete` is exclusive with batch `--todos` JSON (`@file`/`@-` supported). Add/update require content and explicit done state; update/delete need ID; delete omits content/done. Batch order is preserved in requests, though the UI may group by done state. |
| Words | `minutes +word-replace --minute-token <token> --replace-words '[{"source_word":"<old>","target_word":"<new>"}]'`. With explicit source/target words, perform replacement without transcript pre/post reads. Report per-word success/failure; retry only failed words. All-unmatched is not_found, not an ACL failure; ask for exact spelling rather than guessing. |
| Speaker | First GET `/open-apis/minutes/v1/minutes/<token>/transcript/speakerlist` using `lark-cli api`. Match the original display name to its opaque speaker_id; disambiguate duplicate names with transcript evidence and the user. Resolve the target through Contact to an open ID, then `minutes +speaker-replace --minute-token <token> --from-speaker-id <id> --to-user-id <open_id>`. Never pass the display name as speaker_id. |

A todo's named owner is plain inline `@Name`, not a separate Task assignee or clickable mention. Use a supplied name directly. For “me,” get the current user's real name via `contact +get-user --as user`; if unavailable, create the requested todo without an owner suffix, never literal `@me`, and do not let name resolution block creation. Do not reroute to Tasks or explain API field limitations in the result.

Read only relevant current fields before editing, and verify writes with the corresponding read unless the command already returns per-item results, as word replacement does. Preserve partial successes and report the failed stage. `quota_exceeded` means insufficient ASR/AI quota: upload may not create Minutes, or existing AI artifacts may never have completed. Explain and stop; login, permission applications and blind retries cannot supply quota.

## Live meetings and visible actions

With a known source-bound meeting ID, continue directly. Otherwise use `vc +meeting-list-active --as user`, or `--as bot --user-id <open_id>`. App results include only meetings shared by the target user and the bot; an empty list does not mean the user is not meeting. Select among multiple results; match a supplied nine-digit number against `meeting_no`. A number or content question never authorizes joining. User-identity empty results may lead to today's recently ended meetings; app-empty results do not authorize historical search or joining.

Read current content with `vc +meeting-events --meeting-id <id> --as <same_identity> --page-all --format pretty`. Query fresh events before answering current/latest questions or current-meeting summaries, unless explicitly analyzing an old snapshot. Preserve the last page token for incremental reads. Single-page mode clamps size to 20–100; page-all uses 100 with an internal safety cap. Respect remaining continuation and state incomplete coverage. Use JSON for exact fields/document context or IM forwarding, NDJSON for streamed rows. Pretty text escapes newlines.

The bot must be currently present during an ongoing meeting, or have attended within the five-minute post-end grace period. Event coverage depends on attendance and backend visibility; it is not a historical attendee snapshot. Beyond that window, use meeting detail, participant snapshots and post-meeting artifacts.

Visible actions require the user's corresponding intent and confirmed target/content:

- Join/attend: `vc +meeting-join --as bot --meeting-number <number>`; only an explicit Calendar-start request adds `--action start`. Pass a password when required, and an invitation event's original `--call-id` only when supplied. Save returned `meeting.id`; waiting-room submission is not confirmed attendance. Multiple simultaneous bot meetings are supported; no preliminary leave is needed.
- Invite: `vc +meeting-invite --as bot --meeting-id <id> --type SELECTED --open-ids <ids>` (up to 200), or `ALL_SUGGESTED` without IDs. Calendar VC and enabled Agent capability are required. Single-target and bulk permissions differ; bulk/suggested generally require host/co-host. Inspect returned statuses; invited does not mean joined. Suggested invitations are server-selected, capped at 200; their truncation notice is not pagination.
- Send: `vc +meeting-message-send --meeting-id <id> --text <text>` or `--msg-type reaction --emoji-type <key>`; optional `--uuid` is an idempotency key. Use the source identity. Never silently resend after failure or switch identity; distinguish in-meeting text from IM/group messages.
- Countdown: `vc +meeting-countdown --meeting-id <id> --action set|prolong --duration <minutes>`. Only set allows end-audio and one positive reminder offset strictly smaller than duration. `end_in_advance`/`close_window` omit duration, reminder and audio. Do not autojoin or retry visible failures.
- End everyone’s meeting: `vc +meeting-end --as bot --meeting-id <id> --yes`; only the current host bot can do this.
- Leave only the bot: `vc +meeting-leave --as bot --meeting-id <id>`. Confirm it is present. Leave only when requested, never automatically when another task finishes.

Use `--dry-run` when a request preview is useful; it does not perform the action. For join failures, inspect meeting number, password, started state, Agent enablement, waiting room/approval and identity restrictions before assuming a missing scope.

Screenshot only when the question requires current visual information unavailable from events, transcript/chat or a readable shared Doc. `vc +meeting-screenshot --meeting-id <id> --as <same_identity>` saves JPEG; optional `--output` is a safe relative path, and authorized `--overwrite` replaces an existing file. Default files are under `meeting-screenshots/`. The server chooses the composed image; arbitrary regions cannot be selected. Do not use screenshots as a generic meeting-summary fallback.

## Shared content, comments and forwarding

Read relevant shared Docs before summarizing their content; a title/share event is insufficient. If reading fails, label the summary as based only on events. For document context, use JSON and associate each item's exact `share_id` with its `magic_share_started_items[].share_doc`. Both started and detected shares establish mappings; conflicting mappings stop resolution. End events close only the matching share. An unresolved incremental share requires history containing its start; never use the most recent or current document as a substitute. Context-item `share_doc` is retained raw, not a source of title/URL.

For `comment_focus.focused=true`, use `drive +batch-query-comments --url <share_url> --comment-ids <exact_id>` with the source identity. Require exactly one matching returned comment; do not scan all comments as fallback. `quote` identifies location; the first embedded reply is the root. If the comment's own `has_more` is false, use embedded replies. If true, reconstruct from the first `drive +list-replies` page without a token, then follow returned tokens in order. Stop partial on missing/repeated continuation, changed ID or API failure; only the first page has the root comment. Clearing focus requires no read.

Section-location events need no API read: preserve parent-title order followed by title, removing empty trimmed segments. Explicit preview intent permits only `open + image + token` → `docs +media-preview --token <token> --output <path>`, or `open + whiteboard + token` → `docs +media-download --type whiteboard --token <token> --output <path>`. Close, missing tokens or unknown types/actions cause no download. Preserve raw payload, identifiers and usable sibling items when parsing or linking fails; do not guess file types or overwrite as recovery.

Forward meeting chat only when requested, using structured JSON events to construct Feishu post content. For chat message_type 3, preserve the exact reaction key: supported IM keys become `emotion` nodes; VC-only/unknown keys become text `[key]`. Other chat becomes text. Send with `im +messages-send --msg-type post --content <json>`, not Markdown. If one reaction is rejected, downgrade only that key. Explicit “send to me” already authorizes the current-user destination and send action; do not ask again. Use app identity for IM by default unless the user requests sending as themselves. Resolve ambiguous other recipients before sending.

## Permissions and recovery

Separate OAuth scopes from per-resource ACLs. For user missing-scopes errors, follow the returned authorization hint. App scope failures require app permissions, publication/tenant installation and accessible-data-range configuration, not `auth login`. For meeting app access, the configured owner condition must include the app's availability range. Preserve error codes, `log_id` and supplied `console_url` if configuration is correct but access still fails.

Key scopes: meeting search `vc:meeting.search:read`; user active/events `vc:meeting.meetingevent:read`; bot join/active/events `vc:meeting.bot.join:write`; messages `vc:meeting.message:write`; countdown `vc:meeting.interaction:write`; host end `vc:meeting.bot.manage:write`; recording `vc:record:readonly`; Minutes search `minutes:minutes.search:read`, media export `minutes:minutes.media:export`, edits `minutes:minutes:update`, permission requests `minutes:permission:apply`. Use actual error metadata for additional requirements.

An ACL denial does not authorize asking the owner automatically. `minutes +apply-permission --minute-token <token> --perm view|edit --as <source_identity>` notifies the owner and requests access for that identity only; run it only when the user explicitly requests an application, and never report access granted merely because submission succeeded.

Granting somebody else access is different: `drive +member-list` inspects members; `drive +member-add --token <minutes_url> --member-id <id> --member-type openid --perm view|edit --yes` grants access only when target, member, level and authorization are clear. Bare tokens need `--type minutes`; Minutes does not support full_access. Resolve ambiguous names through Contact. Read back the exact member's permission before claiming success or a downgrade.

Use the CLI's `{ok,identity,data}` result and per-item errors, not a presumed raw `{code,msg}` envelope. Keep already obtained IDs/content when another branch fails, distinguish submission from completed work, and do not claim missing or inaccessible evidence exists.

## Reaction keys

Keys are case-sensitive; never uppercase mixed-case values or invent a key. Choose a known semantic match, or clarify ambiguity. Standard reaction keys below may be forwarded as IM emotion nodes; the four `VC_` keys use text fallback in IM.

```text
OK THUMBSUP THANKS MUSCLE FINGERHEART APPLAUSE FISTBUMP JIAYI
DONE SMILE BLUSH LAUGH SMIRK LOL FACEPALM LOVE
WINK PROUD WITTY SMART SCOWL THINKING SOB CRY
ERROR NOSEPICK HAUGHTY SLAP SPITBLOOD TOASTED GLANCE DULL
INNOCENTSMILE JOYFUL WOW TRICK YEAH ENOUGH TEARS EMBARRASSED
KISS SMOOCH DROOL OBSESSED MONEY TEASE SHOWOFF COMFORT
CLAP PRAISE STRIVE XBLUSH SILENT WAVE WHAT FROWN
SHY DIZZY LOOKDOWN CHUCKLE WAIL CRAZY WHIMPER HUG
BLUBBER WRONGED HUSKY SHHH SMUG ANGRY HAMMER SHOCKED
TERROR PETRIFIED SKULL SWEAT SPEECHLESS SLEEP DROWSY YAWN
SICK PUKE BETRAYED HEADSET EatingFood MeMeMe Sigh Typing
Lemon Get LGTM OnIt OneSecond VRHeadset YouAreTheBest SALUTE
SHAKE HIGHFIVE UPPERLEFT ThumbsDown SLIGHT TONGUE EYESCLOSED RoarForYou
CALF BEAR BULL RAINBOWPUKE ROSE HEART PARTY LIPS
BEER CAKE GIFT CUCUMBER Drumstick Pepper CANDIEDHAWS BubbleTea
Coffee Yes No OKR CheckMark CrossMark MinusOne Hundred
AWESOMEN Pin Alarm Loudspeaker Trophy Fire BOMB Music
XmasTree Snowman XmasHat FIREWORKS 2022 REDPACKET FORTUNE LUCK
FIRECRACKER StickyRiceBalls HEARTBROKEN POOP StatusFlashOfInspiration 18X CLEAVER Soccer
Basketball GeneralDoNotDisturb Status_PrivateMessage GeneralInMeetingBusy StatusReading StatusInFlight GeneralBusinessTrip GeneralWorkFromHome
StatusEnjoyLife GeneralTravellingCar StatusBus GeneralSun GeneralMoonRest MoonRabbit Mooncake JubilantRabbit
TV Movie Pumpkin BeamingFace Delighted ColdSweat FullMoonFace Partying
GoGoGo ThanksFace SaluteFace Shrug ClownFace HappyDragon
VC_CanNotSee VC_NoSound VC_LooksGood VC_SoundsClear
```
