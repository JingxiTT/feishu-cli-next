# Calendar section evidence

Status: offline section boundary passed; live backend verification and coordinator comparative development tasks remain outstanding. This is not P2 release acceptance.

## Ownership and frozen mapping

Reference: sibling `../feishu-cli-review` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; catalog and registrations are the captured `docs/reference/meta_data.json.gz` and `registrations.json.gz`. All 14 shortcut implementations and all 18 generic v4 methods are retained. Existing implementations supply workflow behavior; no replacement forwarding layer was added. Parent command groups remain reachable. No frozen Calendar event projection exists; event mechanics stay with the events owner.

Owner: `shortcuts/calendar/`, Calendar affordance when present, `tests/cli_e2e/calendar/`, `tests/next_sections/calendar/`, and this report. Dependencies: shared runtime, identity resolution, file input, generic catalog/schema, result output/artifacts/receipts; rich-description images use Drive upload. The coordinator owns these shared surfaces.

The target for every row is the same command under `feishu-cli-next`. Shortcut metadata, identities, scopes, risk, aliases, formats and flags retain the frozen contract; shared next output controls are additive. The 32 leaf registrations contain 438 flag entries including shared flags. The generic method schemas retain their full optional and nested request/response shapes.

| Frozen leaf command | Mapping and process evidence | Domain flags / schema boundary |
| --- | --- | --- |
| `calendar +agenda` | EveryShortcutDryRun + ReadsAndPlanning | `--calendar-id`, `--end`, `--start` |
| `calendar +create` | EveryShortcutDryRun + EventLifecycleAndNoOp | `--attendee-ids`, `--calendar-id`, `--description`, `--end`, `--meeting-owner-id`, `--rrule`, `--start`, `--summary` |
| `calendar +delete` | EveryShortcutDryRun + EventLifecycleAndNoOp, RecurringScopes | `--apply-to`, `--calendar-id`, `--event-id`, `--notify` |
| `calendar +freebusy` | EveryShortcutDryRun + ReadsAndPlanning (four views) | `--end`, `--min-duration`, `--start`, `--type`, `--user-id` |
| `calendar +get` | EveryShortcutDryRun + ReadsAndPlanning, ArtifactPreservesFullEvent | `--calendar-id`, `--event-id` |
| `calendar +join-event` | EveryShortcutDryRun + ActionsAndAttendees | `--token` |
| `calendar +list-attendees` | EveryShortcutDryRun + ReadsAndPlanning (types/continuation) | `--calendar-id`, `--event-id`, `--page-size`, `--page-token`, `--type` |
| `calendar +meeting` | EveryShortcutDryRun + ReadsAndPlanning | `--calendar-id`, `--event-ids` |
| `calendar +room-find` | EveryShortcutDryRun + ReadsAndPlanning (constraints) | `--attendee-ids`, `--building`, `--city`, `--event-rrule`, `--floor`, `--max-capacity`, `--min-capacity`, `--room-name`, `--slot`, `--timezone` |
| `calendar +rsvp` | EveryShortcutDryRun + ActionsAndAttendees | `--calendar-id`, `--event-id`, `--rsvp-status` |
| `calendar +search-event` | EveryShortcutDryRun + ReadsAndPlanning (continuation) | `--attendee-ids`, `--calendar-id`, `--end`, `--page-size`, `--page-token`, `--query`, `--start` |
| `calendar +suggestion` | EveryShortcutDryRun + ReadsAndPlanning (timezones/attendees) | `--attendee-ids`, `--duration-minutes`, `--end`, `--event-rrule`, `--exclude`, `--start`, `--timezone` |
| `calendar +transfer` | EveryShortcutDryRun + ActionsAndAttendees, RecurringAmbiguityDoesNotWrite | `--calendar-id`, `--event-id`, `--remove-original-organizer`, `--to-user-id`, `--transfer-series` |
| `calendar +update` | EveryShortcutDryRun + EventLifecycleAndNoOp, ActionsAndAttendees, RecurringScopes, PartialAndUnknownMutation | `--add-attendee-ids`, `--apply-to`, `--calendar-id`, `--description`, `--end`, `--event-id`, `--notify`, `--remove-attendee-ids`, `--rrule`, `--skip-room-check`, `--start`, `--summary` |
| `calendar calendars create` | GeneratedMethods / `calendars create` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars delete` | GeneratedMethods / `calendars delete` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars get` | GeneratedMethods / `calendars get` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars list` | GeneratedMethods / `calendars list` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars patch` | GeneratedMethods / `calendars patch` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars primary` | GeneratedMethods / `calendars primary` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar calendars search` | GeneratedMethods / `calendars search` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar event.attendees batch_delete` | GeneratedMethods / `event.attendees batch_delete` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar event.attendees create` | GeneratedMethods / `event.attendees create` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar event.attendees list` | GeneratedMethods / `event.attendees list` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events create` | GeneratedMethods / `events create` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events delete` | GeneratedMethods / `events delete` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events get` | GeneratedMethods / `events get` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events instance_view` | GeneratedMethods / `events instance_view` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events patch` | GeneratedMethods / `events patch` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events search_event` | GeneratedMethods / `events search_event` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar events share_info` | GeneratedMethods / `events share_info` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |
| `calendar freebusys list` | GeneratedMethods / `freebusys list` | Full frozen `--params` / `--data` schema; method, path, optional query and body type preservation |

## Behavior and limitations

Existing nearby units cover Markdown/rich images and workspace boundaries, automatic self attendee inclusion and rollback, attendee user/chat/resource/third-party projections, RSVP, ownership transfer, room availability/approval handling, search and attendee pagination, all-day dates, explicit offsets/timezone warnings, free/busy merging and suggestions, recurring master/instance/exception classification, exception pagination/deduplication/caps, UNTIL/COUNT handling and exception ordering/cancellation. No personal-state legacy live tests were executed.

Target and recurring-scope validation now precede local description image resolution, preventing image uploads for an ambiguous update. A nearby regression checks that the scope error wins before an absent local image could be read.

The new change suppresses a single-event exact replacement when the already fetched event has the same nonempty summary/rich description/recurrence and requested timed timestamps. The response adds `no_op:true`; no event PATCH or attendee notification is sent. Explicit `--notify=true`, explicit clears and attendee add/remove requests retain their effects. A synthetic instance reconstructed from its master is marked internally and cannot establish no-op evidence. This does not claim atomic compare-and-swap or protection from concurrent external edits.

Whole-series updates retain their existing semantics: each exception may have an independent override, and an equal master value is not evidence that every exception matches. This-and-following is an explicit split with deletion/truncation/new-series effects. Whole-series per-exception exact no-op suppression is not implemented; the current exception projection has only IDs and start times. No blanket no-op claim is made for these workflows. Explicit clear still sends the clear even when a fetched value is empty.

The full process suite uses the real authsidecar production CLI, private local config/work state, and an HTTP fixture that never forwards to Feishu. It covers every shortcut, every frozen generic method, successful event lifecycle/readback/cleanup, rich Unicode and precision-preserving attachments, full result artifact identity/tenant binding, all four free/busy views, constrained room/suggestion reads, pagination continuations, three recurring update/delete strategies, inherited following-series attachment/check-in/third-party attendee preservation, ambiguity before writes, scope denial, partial multi-step effects and malformed-response unknown outcomes without replay. Generic method fixtures assert requests and full response preservation; they are not claims about actual backend side effects.

`attendees/batch_delete` remains conservatively unknown (exit 22) even for a successful envelope; the fixture explicitly asserts that result. Without endpoint-specific item evidence the CLI cannot prove all removals. Other synchronous success receipts acknowledge requests, not downstream notification delivery or complete workflow guarantees. Calendar creation retains the reference compensating deletion on attendee failure; it is not an atomic transaction. Crash/cancellation mechanics retain the shared foundation contract. `TestCalendarRecurringProcessInterruptionKeepsUnknownReceipt` now kills the real CLI during an exception PATCH in `--apply-to=all`: the backend inspects the durable `unknown` / `dispatch_started` receipt before applying the synthetic effect, never acknowledges the request, and the test cancels the process. The durable receipt remains identical after termination. Independent readback proves the exception changed while the master remains unchanged; exactly one mutation was attempted, with no replay or attendee request. Forced process termination has a nonzero process exit rather than a fabricated final API envelope.

## Commands and results

Run the complete isolated boundary:

```bash
python3 scripts/next_test.py --section calendar --suite all --report .tmp/calendar-section-interruption.json
```

The runner first executes all Calendar units, then the existing deterministic `DryRun|Regression` selector, then the complete new section process package. The new package uses `nextfixture.RunMain(m, "../../..")` and builds the real authsidecar binary. Prerequisites: canonical built CLI for reference regressions, restored frozen catalog, Python 3, cached Go modules/toolchain. The inherited environment is isolated by `scripts/next_test.py`; no real credentials or personal state enter these commands.

Latest completed boundary:

| Command | Result | Go test events |
| --- | --- | --- |
| `go test -json -count=1 -timeout=10m ./shortcuts/calendar/...` | PASS | 390 pass, 0 fail, 0 skip |
| `go test -json -count=1 -timeout=10m ./tests/cli_e2e/calendar -run "DryRun|Regression"` | PASS | 13 pass, 0 fail, 0 skip |
| `go test -json -count=1 -timeout=10m ./tests/next_sections/calendar` | PASS offline | 77 pass, 0 fail, 1 blocked live skip |

Counts include parent tests and subtests. Sanitized machine-readable command/count evidence is in `calendar-evidence.json`; detailed offline logs remain under `.tmp/calendar-section-interruption.json`. An early package build was temporarily blocked by a concurrent Base implementation edit; the final boundary rebuilt and passed. Earlier failing fixture assumptions (pretty error JSON, command-specific `--yes`, conservative batch outcomes) were corrected without weakening product contracts.

## Outstanding live and acceptance evidence

`TestCalendarLivePrivateLifecycle` is separately selectable and explicitly skips with a blocker. The shared runner's `--suite live --allow-live` also reports blocked until a harness verifies effective current account, tenant, private Calendar creation visibility, run-owned cleanup and authorized destinations before any content write. A display name does not establish Ying Wu's identity. Calendar invitations, notifications, room/resource subscriptions, transfers, joined events, chats and inherited attendees can expose content; none were tested live. These tests remain outstanding, not passed.

The live selectable gate currently records prerequisites and prevents unauthorized execution; the authenticated live lifecycle implementation still needs a verified fixture harness from the coordinator. Live read-after-write consistency, recurrence expansion/timezone behavior, room approvals, rich-image rendering, notification delivery, recipient authorization, sharing and tenant scope remain backend assumptions. No efficiency or comparative claims are made; development comparison is coordinator-owned and held-out tasks were untouched.

Additional focused check: `go vet ./shortcuts/calendar/... ./tests/next_sections/calendar` passed in `isolated_env`; owned files pass `gofmt` and `git diff --check`. Root-wide Make/quality/license checks are coordinator-owned and were not run independently by this section.

Implementation commits: `56dc52d` (exact observed replacements) and `44ad3a9` (scope validation before uploads).

Changed implementation: `calendar_update.go`, `calendar_get.go`, `calendar_delete.go`; new nearby regression: `calendar_noop_test.go`; process tests: `calendar_e2e_test.go`, `dryrun_e2e_test.go`, `generated_e2e_test.go`, `recurring_e2e_test.go`, `interruption_e2e_test.go`. No shared contracts, generated metadata, personal resources or held-out workloads were edited.
