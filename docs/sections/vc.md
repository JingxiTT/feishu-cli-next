# VC section

Status: complete frozen implementation reused; the complete offline section
suite passes. Live behavior and comparative development evaluation remain outstanding.
The section is not accepted for release until those required checks pass.

## Complete mapping and ownership

The reference is sibling `feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`. Inspection of
`docs/reference/registrations.json.gz`, `meta_data.json.gz`, `coverage.json` and
`sources.json` found **16 VC registrations: 13 shortcuts, one generated API
method, and two grouping commands**. The frozen catalog exposes only
`vc meeting get`; it has no reservation or room/resource CRUD registrations.
Those unregistered upstream endpoints are not silently substituted for the
reference surface. All 59 frozen VC shortcut, event and reference E2E source
files matched the recorded SHA-256 hashes before section work.

| Registration | Reused owner and behavior | Process evidence |
| --- | --- | --- |
| `vc`, `vc meeting` | Existing command assembly and catalog grouping | `TestFrozenVCDiscovery` |
| `vc meeting get` | Generic catalog method; meeting identity, participant/device structures, ability statistics, meeting artifacts, query modes 0/1, open/user/union ID types, params/output and formats | `TestVCSearchAndGenericMeetingArtifacts`; existing generic schema/request units |
| `vc +search` | `vc_search.go`; keyword/time/organizer/participant/room filters, Unicode bound, ordered participant deduplication, one bounded page with explicit continuation | `TestVCSearchAndGenericMeetingArtifacts`, `TestVCValidationPreventsBusinessRequests` |
| `vc +detail` | `vc_detail.go`; batches to 50, notes/recording links, ongoing meeting hints, per-item errors | `TestVCDetailRecordingAndNotesResolution`, `TestVCReadPartialAndInProgress` |
| `vc +recording` | `vc_recording.go`; meeting or calendar instance batches to 50, recording URL/minute token/duration, fallbacks | `TestVCDetailRecordingAndNotesResolution`, `TestVCReadPartialAndInProgress` |
| `vc +notes` (hidden, callable) | `vc_notes.go`; meeting/minute/calendar input, note details, AI summary/todos/chapters/keywords, transcript FileIO, overwrite and path validation | `TestVCDetailRecordingAndNotesResolution`, `TestVCValidationPreventsBusinessRequests` |
| `vc +meeting-join` | Bot join endpoint; 9-digit number, password/call ID, join/start and bot-only start | `TestVCMutatingWorkflows`, `TestVCValidationPreventsBusinessRequests` |
| `vc +meeting-invite` | Bot-only selected/all-suggested modes, open_id recipients, ordered deduplication and 200 limit; per-item outcomes and candidate limit notice | `TestVCMutatingWorkflows`, `TestVCInvitePartialOutcome`, `TestVCValidationPreventsBusinessRequests` |
| `vc +meeting-end` | Host app bot endpoint, high-risk confirmation, meeting ID | `TestVCMutatingWorkflows`, `TestVCValidationPreventsBusinessRequests`; existing host/end regressions |
| `vc +meeting-leave` | Existing identity and meeting ID request | `TestVCMutatingWorkflows` |
| `vc +meeting-list-active` | Current user versus explicit target in bot mode, identity-specific scope diagnostics | `TestVCActiveAndEventsPagination`, `TestVCValidationPreventsBusinessRequests` |
| `vc +meeting-events` | Typed meeting/current-identity/event/actor projections; participant join/leave, transcript, chat/reaction, magic share/document/comment context, countdown and unknown event payloads; time range, JSON/NDJSON/table/pretty; single page or at most 200 pages | `TestVCActiveAndEventsPagination`, `TestVCMeetingEventsBoundedPagination`; complete existing event/speaker units |
| `vc +meeting-message-send` | Text/reaction selection, payload conflicts, 48 KiB text and 128-byte UUID bounds; optional caller UUID | `TestVCMutatingWorkflows`, `TestVCMutationUnknownAndRejectedNeverReplay`, `TestVCValidationPreventsBusinessRequests` |
| `vc +meeting-screenshot` | JPEG POST read, 8 MiB bound, JPEG decoding, invocation FileIO, explicit overwrite, digest | `TestVCScreenshotBytesAndExistingOutput`; complete invalid image/content-type/path unit regressions |
| `vc +meeting-countdown` | Set/prolong/end-in-advance/close-window, positive minutes, reminder before duration, explicit false audio preservation | `TestVCMutatingWorkflows`, `TestVCValidationPreventsBusinessRequests` |

The section owns `shortcuts/vc`, `tests/cli_e2e/vc`,
`tests/next_sections/vc`, this report and matching affordance. Shared receipts,
output, runtime, catalog, root assembly and manifest are coordinator-owned.
Events under `events/vc` are inspected and exercised as reference regressions;
the events owner retains implementation ownership. Calendar relation resolution,
Note detail parsing and Minutes artifacts are reused cross-section dependencies.

## Preserved contracts and limits

There is no product interface narrowing or domain implementation replacement.
Existing hidden `+notes` stays callable. Standard build exposes profile
selection; authsidecar deliberately conceals `--profile`, so process discovery
records that conditional build exception. The shared frozen normal-build
inventory remains the profile metadata owner.

Default agent JSON is compact; opt-in projection/artifact controls come from the
shared output owner. Domain-specific historical projections remain: search
omits avatar metadata, meeting detail returns its consolidated field set, and
meeting events project/compact meeting state while retaining event payloads.
The generic meeting get surface returns all backend fields. These existing
projections are not new claims that every shortcut returns raw response bytes.

Reference normalization is retained: text messages trim surrounding whitespace
while preserving internal Unicode and newlines; meeting numbers, IDs, passwords,
UUIDs and related flags retain their established normalization. `+meeting-events`
clamps ordinary page sizes to 20–100 and uses 100 in page-all mode; reaching the
200-page cap retains `has_more` and its token. It does not claim snapshot
consistency or deduplicate a repeating cursor. Search fetches one page and
preserves continuation. All of those are explicit bounds, not proof of live
exhaustiveness.

No supported VC shortcut is a remote content replacement. Sending, inviting,
joining, countdown actions, leaving and ending retain action semantics; repeating
them is not an inferred no-op. Screenshot collisions avoid dispatch unless
`--overwrite` is explicit. Existing transcripts are reused unless overwrite is
explicit. Local locks cannot prove remote concurrency protection.

Mutation receipts establish request acknowledgement or conservative uncertainty,
not participant delivery, invitation acceptance or complete workflow success.
Lost/malformed/202 responses are never automatically replayed; UUID remains the
caller's optional server idempotency key. VC invitation per-item failure evidence
requires the shared receipt classifier and remains unknown rather than a claim
that every recipient was invited. No automatic reconcile or replay was added.
Read batches keep the reference partial-result contract: some per-item errors may
coexist with `ok:true`; all recording failures return `ok:false` with full results.
Consumers inspect per-item errors and hints, not only process success.

## Validation and outstanding work

Run the complete section boundary with:

```sh
python3 scripts/next_test.py --section vc --suite all --report .tmp/vc-section.json
```

This executes all VC shortcut and event units, inherited deterministic
`DryRun|Regression` process tests and the complete new real-entry-point suite.
The new suite builds the actual authsidecar binary using
`nextfixture.RunMain(m, "../../..")`; every business request terminates at its
owned loopback synthetic backend. Config, credentials, home and artifact paths
are isolated with `scripts/next_test.py`. No fixture forwards to a real tenant.
The separately selectable `TestVCLivePrivateLifecycle` records a blocked live
case; a skip never counts as live success.

The completed 2026-09-07 boundary run passed (exit 0), recorded in
`.tmp/vc-section-final.json`:

| Exact selected command | Passed test events | Failed | Skipped |
| --- | ---: | ---: | ---: |
| `go test -json -count=1 -timeout=10m ./shortcuts/vc/... ./events/vc/...` | 482 | 0 | 0 |
| `go test -json -count=1 -timeout=10m ./tests/cli_e2e/vc -run 'DryRun|Regression'` | 22 | 0 | 0 |
| `go test -json -count=1 -timeout=10m ./tests/next_sections/vc` | 68 | 0 | 1 live blocker |

Additionally, `go test -json -count=1 -timeout=10m ./tests/cli_e2e/vc`
ran through `scripts/next_test.py`'s `isolated_env`/`run_command` helpers and
passed all 40 test events without skips, including the normal-build embedded
meeting skill and routing/link contracts. Its raw evidence is
`.tmp/vc-reference-all.json`. These counts include parent/subtest events and
are not unique workflow counts; the separate legacy run overlaps the selector.
`gofmt` and the owned-path `git diff --check` pass. Shared pre-PR/release-wide
checks remain the coordinator's gate.

The new per-item invitation regression initially failed with a false
`confirmed_applied` outcome. The shared receipt owner's narrow typed
`/vc/v1/bots/invite` classifier makes that regression pass; this section's
process evidence depends on that shared change. Source reuse leaves domain
implementation and its existing unit tests unchanged. Owned changes are this
report, `tests/next_sections/vc/vc_e2e_test.go` and the next-distribution addendum
to `tests/cli_e2e/vc/coverage.md`.

Live prerequisites: verify effective current account and tenant, private
Agent-enabled Calendar fixture visibility before creation, host bot ownership,
and every actual/eligible meeting participant. Any other recipient must be
unambiguously Ying Wu, including indirect invite/notification recipients.
Never use unrelated existing meetings or create broad visibility then restrict
it. The run must create, read, join/start, exercise actions and notes/recording
where available, leave/end, delete its own event, and independently verify
cleanup. The legacy manual ALL_SUGGESTED transcript in reference coverage is
historical context, not authorized evidence for this run. These guarantees are
not currently verifiable through an available live fixture, so live meeting
exposure, notifications, host authority, recording generation and cleanup remain
unverified. The product still supports the full authorized-user capability set.

Comparative development tasks are scheduled by the coordinator with balanced
arms and token accounting. No held-out task was inspected or run, and no token
or efficiency improvement claim is made by this section.
