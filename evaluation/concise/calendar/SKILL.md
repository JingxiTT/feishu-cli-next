---
name: lark-calendar
version: 1.0.0
description: "Manage Feishu calendar events, attendees, availability and meeting rooms. Use for calendar reads/search, scheduling, updates, recurrence and organizer transfer. Use Meetings for historical video meetings and Tasks for to-dos."
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli calendar --help"
---

# Calendar

Use [shared authentication, identity, permissions and output guidance](../lark-shared/SKILL.md). Read the corresponding command reference or help before invoking a command; `lark-cli schema calendar.<resource>.<method>` supplies native schemas. Calendar resources are `calendars`, `events`, `event.attendees` and `freebusys`. Generic methods remain available for fields the shortcuts do not expose.

Choose identity by event ownership: the logged-in user's events use `--as user`; the app's events use `--as bot`. “I” means the logged-in user and “you” means the app when identifying the actor; a requested attendee or meeting owner does not change that actor. `primary` is the selected identity's primary calendar. Resolve people with Contact and groups with IM using user identity; clarify ambiguous names or entity types rather than guessing. Bots have `ou_` attendee IDs too; an app ID is not an attendee ID.

## Read and route

- `+agenda [--calendar-id <ID>] --start <START> --end <END>` reads events, defaults to today, filters cancelled events and splits ranges exceeding 40 days. Present an ascending timeline grouped by date; distinguish an empty result from a cleared or deleted calendar.
- `+search-event` searches by keyword, time range and attendee IDs; it returns basic fields. Follow `has_more` / `page_token` when completeness matters. `+get --event-id <ID>` supplies full single-event detail and video meeting links, but not attendee/room lists.
- [`+list-attendees`](references/lark-calendar-list-attendees.md) lists `user`, `resource`, `chat` and `third_party` attendees; repeat or comma-separate `--type`, and preserve page continuations. Chat attendees have no group-level RSVP; use native APIs for members' RSVP.
- [`+meeting`](references/lark-calendar-meeting.md) accepts up to 50 comma-separated event IDs and user identity. It returns associated historical `meeting_id` and the manually attached `meeting_note`, which is distinct from AI notes. For AI notes or recordings, continue to `vc +detail`, then `note +detail` or `minutes +detail`, then Docs when needed. Missing artifacts can be legitimate.
- Historical video meetings belong to Meetings, including unscheduled meetings absent from Calendar. For today's meetings, combine ended-meeting history with ongoing/future calendar events. To-dos belong to Tasks.
- Share an event using the returned `calendar events share_info` link, then authorized IM or document actions. Do not invent an applink or substitute a video meeting URL. [`+join-event`](references/lark-calendar-join-event.md) accepts only the share token (`--token`, alias `--share-token`) from a link, decoded QR code or card. A series token joins the series; an exception token joins that exception. [`+rsvp`](references/lark-calendar-rsvp.md) changes accept/decline/tentative only for an invited event; joining and replying are different actions.

## Time and scheduling

Convert timestamps with a script/system tool and an explicit target timezone; never rely on the process's default timezone. Use offset-bearing ISO 8601 for shortcut timed inputs and Unix-second strings for native timestamp fields. Preserve all-day date semantics: shortcut display uses an inclusive last date, while native fields must follow the selected API schema. Relative days cover the entire local day; weeks start Monday. Do not schedule a wholly past interval. Clarify ambiguous AM/PM or personal expressions such as “after work.”

Before scheduling, rescheduling or finding rooms, read [the scheduling workflow](references/lark-calendar-schedule-meeting.md) and its [explicit-time](references/lark-calendar-schedule-clear-time.md) or [fuzzy-time](references/lark-calendar-schedule-fuzzy-time.md) branch. First distinguish creation from editing. An existing-event anchor plus a change request means edit: find a unique event/instance ID and retain it throughout; show ambiguous matches for selection. When only a new start is given, read the original duration and preserve it in the new end.

For creation, infer title and duration from context; otherwise use “Meeting,” 30 minutes and self as attendee. With no time supplied, propose a reasonable search interval rather than asking for every field. Vague time or a time range requires [`+suggestion`](references/lark-calendar-suggestion.md), which considers working hours and rest, not just empty intervals. Pass non-bot attendees, duration, timezone, recurrence and excluded earlier suggestions as appropriate. Explain conflicts honestly: a recommendation is not necessarily conflict-free. Follow `ai_action_guidance` or suggest changing the range, duration or attendees when all options conflict.

For a definite slot, find rooms if needed, then check busy time unless the user already selected a suggestion. `+freebusy --type busy` merges per-user intervals; `raw_busy` preserves separate events and RSVP; `free` returns per-user gaps; `common_free` computes their intersection. `--min-duration` applies only to free views. Bot identity must name target users. Skip bot busy checks; do not expand groups; for more than five attendees check self and core participants.

[`+room-find`](references/lark-calendar-room-find.md) requires definite `--slot START~END` blocks. For fuzzy requests, pass suggested blocks together, then show times and their numbered room lists on separate lines. Preserve returned room names verbatim. Extract city only when explicit, remove a repeated city prefix from building, normalize floor to forms such as `F2`, and split compound floor/room labels; comma-separate multiple room names. Nearby-location recommendations can legitimately differ from filters. Check `reserve_until_time` for recurrence, explain a shorter horizon, and shorten the rule only after the user chooses that room.

Show time/room alternatives and wait for the user's selection before creating or updating. Explain conflicts and obtain the user's choice to continue or change time. Adding a room is incremental; remove an existing room only when replacing/removing it was requested. Rooms are resource attendees, not standalone bookings.

## Changes and recurrence

Read [`+create`](references/lark-calendar-create.md) or [`+update`](references/lark-calendar-update.md) before writing. Use recurring rules for recurring intent, not many unrelated events. The scheduling/update references require converting a limited repetition count to `UNTIL` rather than passing `COUNT` to planning.

`+create` accepts title, offset-bearing start/end, Markdown description, attendees and recurrence. Defaults include editable attendee access, busy status, a five-minute reminder and Feishu video meeting. `--meeting-owner-id` applies only to bot creation on the app calendar and requires an in-tenant user. For advanced fields, follow the native event/attendee schema: location, visibility, custom reminders, access, busy status, color, attachments, video settings, optional attendees and all-day dates. The reference prefers shortcut creation followed by a native patch where suitable; do not broaden the user's requested visibility or effects.

Descriptions use Markdown, `@file` or stdin. Relative local images inside the working directory upload to Drive; remote URLs pass through and Feishu document links become inline mentions. Preserve rich content. Use properly nested bold/italic delimiters, not triple-asterisk styling. Explicit empty title/description clears it; omitted flags leave it unchanged.

`+update` independently changes fields, removes attendees and adds attendees, in that order. Add/remove accept `ou_` users/bots, `oc_` chats and `omm_` rooms; addition is not replacement. Time changes require both start and end. `--notify` defaults true; `--notify=false` requests silence. Multi-step updates do not roll back earlier success. Creation compensates for failed attendee addition by deleting the newly created event. Room approval uses user identity and native `event.attendees create` with `approval_reason` (at most 200 characters). Never automatically add `--skip-room-check` after a room failure: explain the room/reason and obtain agreement. A failed availability service can instead warn and allow the update.

Before editing/deleting recurrence, read [the recurrence reference](references/lark-calendar-recurring.md); clarify scope when the user has not specified it. `{uid}_0` is normal or master; positive suffixes are occurrences. `is_exception` distinguishes an independently edited exception from an unmaterialized instance. Normal events use single; masters require all; instances allow single/all/this-and-following; exceptions allow single/all only.

`--apply-to single` affects one occurrence. `all` explicitly handles master and exceptions, propagating only requested fields; moving series time deletes exceptions first. `this-and-following` requires an ordinary instance pivot: truncate the master, delete future exceptions, and for updates create a following series inheriting defaults with requested overrides. Do not infer scope from a title or assume master mutation cascades automatically.

[`+transfer`](references/lark-calendar-transfer.md) acts as the current organizer, targets a same-tenant `ou_` user or bot, is irreversible and requires `--yes`. Any recurring occurrence transfers the entire series, including exceptions; obtain whole-series agreement before `--transfer-series`. Shared calendars force removal of the old organizer. If `original_organizer_removed` is absent, do not claim either retention or removal.

Confirm intended effects before writes; preview dangerous requests with `--dry-run`. Exit 10 requires explicit approval before adding the returned confirmation flag to the original command. Report writes from the command's actual result; reread only when requested or needed to answer missing information. Do not infer atomicity or replay safety from an error. JSON success is `ok:true`/exit 0; failures and warnings belong on stderr. Never expose credentials.
