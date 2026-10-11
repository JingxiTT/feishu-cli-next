# Minutes section

Owner: `shortcuts/minutes/`, `affordance/minutes.md`,
`tests/cli_e2e/minutes/`, and `tests/next_sections/minutes/`. Shared transports,
receipts, discovery, output/artifacts, authentication and FileIO remain foundation
owned. Event registrations/projections remain events owned.

The frozen sibling at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3` was compared
file by file with this section before implementation. All ten shortcut implementations
are reused. The only pre-existing section difference was the deterministic media
DNS test correction from `111eefc`, which is preserved. The captured
`baseline/registrations.json.gz` and `coverage.json` also map the generic
`minutes minutes get` endpoint and two command groups. No endpoint was replaced
with raw API forwarding or dropped.

## Complete capability mapping

Every row retains its original command path, risk and identities. Shared
`--as`, `--profile`, `--format`, `--json`, `--jq`, and `--dry-run` behavior remains
available; next output controls come from the common owner. Original hidden flags
are retained. The table names all domain flags, including uncommon options.

| Command | Supported scope and types/options | Deterministic evidence |
| --- | --- | --- |
| `minutes +search` | User/bot; keyword, owner/participant open IDs including `me`, start/end date or ISO timestamps; `query`, `owner-ids`, `participant-ids`, `start`, `end`, `page-token`, `page-size` (1–30, default 15). One bounded page, continuation and notice retained; original search display/avatar policy retained. | Existing search units/dry runs; `TestSearchBoundedPageAndArtifact` checks user/bot wire filters, one request, continuation, null/Unicode/leading-zero strings and full exclusive artifact despite projected stdout. |
| `minutes +detail` | User/bot; `minute-tokens` (1–50), `summary`, `todo`, `chapter`, `transcript`, `keyword`, `output-dir`, `overwrite`; hidden `wait-ready`, `wait-timeout-seconds`, `wait-interval-seconds`. Title/note ID and selected native artifact shapes retained; transcripts saved to FileIO. | Existing detail units include conditional scopes, waiting/cancellation, errors, paths; `TestDetailArtifactsFilesAndFailures` checks complete transcript bytes, summary/todo/chapter/keywords, collision preservation, overwrite, bounded readiness, mixed ACL failure and file failure. |
| `minutes +download` | User/bot; `minute-tokens` (max 50), `output`, `output-dir`, `overwrite`, `url-only`. Signed URL discovery, streaming media, filename disposition/type fallbacks, collisions, duplicates, redirect/source validation and rate-limited batches retained. | Existing full media units (including synthetic DNS fix); `TestDownloadMediaAndBatch` uses a local non-forwarding HTTP proxy sink to verify actual media bytes, explicit destination, collision/overwrite, user/bot, URL-only and batch failure. |
| `minutes +upload` | User only; `file-token` of Drive media. Retains audio wav/mp3/m4a/aac/ogg/wma/amr and video avi/wmv/mov/mp4/m4v/mpeg/ogg/flv guidance, 6 GB/6 hour limits, generated URL/token extraction and quota errors. | Existing upload units/dry run; `TestTodoBatchSpeakerUploadAndPermission/upload` checks exact Drive-token handoff and returned identity. Backend generation readiness remains separate. |
| `minutes +update` | User only; `minute-token`, exact `topic`; optional new `skip-unchanged` requires basic read scope and compares the current title. | Existing title units/dry run; `TestMinutesSkipUnchanged`, `TestMutationBodiesAndNoop/update`, `TestReplacementReadback` verify changed content, independent readback and no second write. |
| `minutes +summary` | User only; `minute-token`, exact `summary` literal/file/stdin text. Markdown remains accepted verbatim; rendering subset guidance retained. Optional `skip-unchanged` requires artifact read scope. | Existing summary units; preservation/no-op units; `TestMutationBodiesAndNoop/summary` checks file input and exact whitespace; `TestSkipUnchangedFailsClosedOnUnreadableContent` verifies scope/malformed reads prevent mutation. |
| `minutes +todo` | User only; `minute-token`, `operation` add/update/delete, `todo`, explicit boolean `is-done`, `todo-id`; `todos` literal/file/stdin JSON array supports ordered mixed operations with string content/IDs and boolean completion. Optional `skip-unchanged` accepts only updates to distinct IDs and requires artifact read scope. | Existing single/batch validation and body units; preservation/no-op units; process tests cover all three ordered operations, false/true, leading-zero IDs, exact file content, no-op and invalid mixed-mode intent. |
| `minutes +speaker-replace` | User only; `minute-token`, opaque `from-speaker-id`, hidden legacy `from-user-id`, `to-user-id`; original explicit speaker precedence and `user_id_type=open_id` retained. | Existing identity/error/body units and both dry runs; both speaker process cases assert exact IDs, query and body. Ambiguous/display-name target validation is rejected. |
| `minutes +word-replace` | User only; `minute-token`, `replace-words` literal/file/stdin array. Exact source/target strings, explicit empty target deletion, unique exact source words, numeric/string counts and per-word recovery. | Existing error/quota/conflict units and dry run; new preservation/count-validation units and `TestWordReplaceOutcomes` cover applied/mixed/zero/missing/incomplete outcomes without mutation retries. |
| `minutes +apply-permission` | User/bot; `minute-token`, `perm` view/edit. This requests access; it does not promise an access grant. | Existing user/bot dry runs and validation units; view/edit process cases verify the notification-capable request body. Live requests are blocked without verified recipient boundaries. |
| `minutes minutes get` | Generic read endpoint, user/bot; `minute-token`, `user-id-type`, `params`, `output`; hidden shared pagination controls retained. Complete upstream metadata, owner/resource identity, duration/time strings and URLs remain accessible. | `TestGenericMetadata` verifies the real generated command's request/query and lossless string metadata, plus generic readback in the title lifecycle. |
| `minutes.minute.generated_v1` | User event; token, event ID, source timestamp/type/entity ID, title enrichment and subscription/unsubscription. | Seven existing `events/minutes` unit test events are included in the section suite. Events owner separately verifies `TestAllEventKeysReplayThroughCLI/minutes.minute.generated_v1` in `tests/next_sections/events/events_e2e_test.go` with its frozen replay fixture and exact subscription receipts. |

Dependencies: Drive media upload supplies `file-token`; VC recordings/search may
supply minute tokens; Docs owns optional `note_id` content. Those domains retain
their own writes and permissions. Generic result containers are not import
payloads. No event or shared runtime implementation is duplicated here.

## Intentional interface differences

- Summary text, todo content and transcript source words preserve accepted
  leading/trailing whitespace instead of silently trimming it. Explicit empty
  target strings still delete words; omitted/null targets now fail validation
  instead of being silently interpreted as deletion. Summary dry-run includes
  the actual replacement body.
- `+update`, `+summary` and `+todo` add opt-in `--skip-unchanged`, conditional
  read-scope metadata and read/write dry-run previews. Ordinary writes keep
  their original required scopes. An explicitly requested comparison that
  cannot read unambiguous content fails before writing. Equality is exact;
  no write/receipt is produced for unchanged content. This is an observation,
  not a conditional update or protection from other clients' concurrent edits.
  Adds, deletes, permission requests and speaker assignments retain their
  explicit effects; absent data never implies deletion.
- Mixed read/download failures now retain the complete stdout result with
  `ok:false` and exit 1. Transcript save errors are per-item failures rather
  than successful empty paths. Empty detail token lists are rejected. Existing
  transcripts remain untouched unless overwrite is requested, and long Unicode
  directory titles are truncated at a valid UTF-8 boundary.
- Word replacement emits structured counts and succeeded/failed or unknown
  words alongside its human-readable recovery message. Mixed results are
  non-successful; malformed, duplicated, negative or incomplete count evidence
  never invents all-success or authorizes replay. Shared receipt classification
  supplies conservative mutation outcomes; it does not imply backend rollback.

## Section boundary and remaining evidence

The process suite builds the real production `authsidecar` entry point through
`nextfixture.RunMain(m, "../../..")` and uses only `NewFixture` local synthetic
backends. The media proxy is a sink and cannot forward requests. Each test owns
its temporary state; no live credentials, user content or existing meetings are
read or mutated. Scope rejection and connection loss assert one dispatch and
20/22 mutation exit semantics. Local equality reads have no receipts.

Canonical commands:

```sh
make build
python3 scripts/next_test.py --section minutes --suite all --report .tmp/minutes-section.json
python3 scripts/next_test.py --section minutes --suite live --allow-live --report .tmp/minutes-live.json
```

The runner emits machine-readable command/test pass/fail/skip counts. Additional
checks use the same `scripts/next_test.py:isolated_env` helper: the entire inherited
`go test -json -count=1 ./tests/cli_e2e/minutes` package (including the invalid-perm
test excluded by the runner's legacy name selector),
`go test -json -count=1 ./internal/affordance ./cmd/service ./internal/schema`, and
`go vet ./shortcuts/minutes/... ./tests/next_sections/minutes`.

Recorded complete offline boundary on 2026-09-07 (shared word-outcome classifier
`cfb5afa` included): canonical `make build` passed; section units/events passed
195 test events; inherited deterministic selection passed 11; the new process
suite passed 30 with one explicit live blocker skip and no failures. The extra
full inherited package passed 12; affordance/service/schema checks passed 277;
scoped `go vet` and owned `git diff --check` passed. Raw machine reports are
`.tmp/minutes-section.json`, `.tmp/minutes-extra.json` and
`.tmp/minutes-live.json` (the live runner exits 2 with `status:blocked`).
Counts include named subtests. No comparative or held-out evaluation was run by
this owner. Full repository unit/static/license/quality gates are coordinator
release checks, not claimed as part of this section evidence.

Live status is **blocked**, not passing: there is no verified run-owned Drive
media fixture with current-account/tenant binding and private Minutes creation
visibility before upload. No unrelated pre-existing meeting may be used for
writes. `TestLiveMinutesPrivateLifecycle` is separately selectable and records
this blocker without dispatching. The authorized live lifecycle must first verify
a current-account-only Drive destination and Minutes creation visibility, then
upload synthetic media, await readiness, verify metadata/transcript/media and
perform/read back title/summary/todo/word updates on that created minute. Cleanup
must remove only run-owned resources through supported owners; the frozen
Minutes surface has no delete command. Permission requests and speaker rebinding
also require an unambiguous authorized recipient (only Ying Wu may be another
recipient); display names do not establish identity or delivery safety.

Comparative development tasks are coordinator scheduled; held-out tasks remain
untouched. Offline fidelity is not evidence of live ACLs, transcription accuracy,
quota availability, backend batch atomicity or notification behavior. Live and
comparative evidence keep final section acceptance open.
