# Note capability section

Status: implementation and complete written offline suite pass; live and
comparative acceptance remain open. Owner: Note section. Foundation: `ce68188`; reference:
unchanged sibling `../feishu-cli-review` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`.

## Complete mapping and dependencies

The frozen `coverage.json` maps exactly `note`, `note +detail`, and
`note +transcript` to this section. Frozen Feishu/Lark registrations, source
hashes, and sibling source agree. Note is a shortcut-only namespace, with no
additional generated methods, create/update/delete operations, or independent
event projections. Source implementation is reused, not raw API forwarding.

| Capability | Preserved contract | Test owner/evidence |
| --- | --- | --- |
| `note` | Registered domain, command help, actual-tree discovery | `tests/next_sections/note`: discovery and frozen flags |
| `note +detail` | `--note-id`; GET detail; `vc:note:read`; user and bot; read risk | Legacy units/dry-run plus process identities and detail values |
| Detail types | Normal/unified/unknown display strings; numeric enum handling; fallback `display_type`; creator/time; main/verbatim/shared document tokens; empty token strings; omitted empty shared list | `shortcuts/note/note_test.go`, process detail types and artifact test |
| `note +transcript` | User only; `vc:note:read`; read risk; detail prerequisite; normal-note recovery with actual verbatim token | Legacy identity/recovery units plus process validation and normal-note recovery |
| Transcript format/input | `--transcript-format` markdown/plain_text separate from `--format`; `--locale` explicit/profile/brand; `--note-id` resource validation | Legacy unit/dry-run plus process both formats, Unicode and exact output bytes |
| Full pagination | Page size 200; successive cursor IDs including exact large decimal strings; empty chunks; cycle/stall detection; 500-page guard; cancellation; no partial save after a failed page | Legacy cursor/cancellation units, new malformed-page units and multi-page process tests |
| File output | `--output`; default `notes/{id}/unified_transcript.{md,txt}`; existing destination requires `--overwrite`; exact bytes and size; invocation FileIO | Legacy file tests, process default paths/overwrite and failed-overwrite preservation |
| Shared flags/output | `--as`, `--profile`, `--dry-run`, `--format`, `--json`, `--jq`; additive result controls | Frozen registration inspection/help assertions, legacy dry-run plus saved complete detail artifact process test |
| Typed recovery | API 121005 maps to authorization/permission_denied retaining cause/scope/log metadata; other typed failures propagate; invalid/empty detail and transcript fail | Legacy error units and process second-page permission failure |

`vc +notes` imports Note's shared `FetchDetail`/`Detail.ToMap`; its meeting lookup
and Note-generated events belong to VC/events owners. Normal note transcript
content uses Docs' `docs +fetch --doc <verbatim_doc_token>` handoff. API transport,
identity, FileIO, compact JSON/projection/artifacts and errors use foundation
owners. No permission scope or additional remote prerequisite was introduced.
`schema` is the reference generated-service catalog and does not expose these
shortcut-only Note commands; their supported guidance is `note <command> --help`
and next's actual-tree `discover note`.

## Intentional behavior difference

Malformed transcript page content (missing object, selected format absent/null/
non-string) or a present non-boolean `has_more` now produces typed
`internal/invalid_response`. Previously a malformed intermediate page could be
silently skipped, or a malformed pagination boolean treated as terminal, while
saving an incomplete transcript as success. Empty string chunks are still valid;
the aggregate empty transcript still fails. The legacy omitted `has_more`
terminal behavior remains supported. Backend pagination does not guarantee a
snapshot: content can change between pages, and the CLI cannot prove otherwise.

No remote mutations exist in this section, so receipts, mutation no-op,
notification, clear and unknown-write reconciliation do not apply. Downloads
remain read operations with local file effects; explicit overwrite replaces the
file with fetched bytes. The inherited Stat-then-Save overwrite check is not a
cross-process exclusive-create guarantee. Buffering remains bounded by page
count, not by total bytes. No new global mechanism or permission was added.

## Reproducible verification

Use the canonical binary and isolated runner (synthetic sidecar backend never
forwards live traffic):

```sh
make build
python3 scripts/next_test.py --section note --suite all --report .tmp/next-tests/note-all.json
python3 scripts/next_test.py --section note --suite live --allow-live --report .tmp/next-tests/note-live.json
```

The completed section command runs every Note unit, all inherited dry-run E2Es,
and the full new real-process suite using `nextfixture.RunMain(m,"../../..")`.
Machine-readable reports distinguish offline results, exclusions and live
blockers. No legacy credential-dependent live helper is invoked.

## Recorded offline outcomes (2026-09-07)

- `make build`: pass, frozen catalog restored; normal production build.
- Complete `--section note --suite all`: pass, 44 unit test events, four inherited
  dry-run test events, 17 real-process test events; zero failures or skips, no
  inherited Note E2E exclusions. Report: `.tmp/next-tests/note-all.json`.
- Isolated `go test -json -count=1 ./shortcuts/vc -run Note`: pass, 44 test events,
  zero failures/skips. Related report: `.tmp/next-tests/note-related.json`.
- Isolated `go vet ./shortcuts/note/... ./tests/next_sections/note`: pass.
- `gofmt -l` on all four changed Go files: empty output (pass).
- Frozen Note source SHA-256 hashes: all match unchanged reference sibling.
- Live command above: blocked, exit 2; report `.tmp/next-tests/note-live.json`.

The process fixture intentionally uses the real authsidecar build, which hides
profile selection; normal-build profile registration remains frozen and unchanged.
Initial process-test failures were assertion issues (hidden sidecar profile flag,
macOS canonical `/private/tmp` output paths, pretty stderr JSON); these were fixed
and the whole section rerun successfully. No product gates were weakened.

Changed files: `shortcuts/note/note_transcript.go`, its nearby
`note_transcript_test.go`, `tests/cli_e2e/note/note_dryrun_test.go`,
`tests/next_sections/note/note_e2e_test.go`, and this section report. Existing
Note affordance and command registrations require no changes. Root-wide unit,
vet/fmt and committed quality gates remain coordinator-owned.

## P3 discovery regression follow-up

Public-flag search increased `discover note` beyond its default 20-result page.
The section test now follows `meta.next_offset` until `meta.complete`, checks
forward progress, unique entries and the complete reported total, and verifies
exact membership of both Note commands. It no longer assumes either command's
position in the result order. Product search and ranking are unchanged.

The full boundary rerun passed:
`python3 scripts/next_test.py --section note --suite all --report .tmp/next-tests/note-discovery-boundary.json`.
All 44 unit, four inherited dry-run and 17 process test events passed with zero
failures/skips. This repairs the test failure observed in
`.tmp/next-evidence/p3-e2e-next-fixed-next-sections.json`; it does not establish
any additional live or comparative evidence. Evaluation adapter files were not
changed by this follow-up.

## Open acceptance evidence

Live verification requires a run-owned private normal Note and unified Note,
verified effective current account and tenant, supported user/bot read grants,
and known expected full transcript bytes for both formats/locales. Creation must
be private from the outset; neither Note command creates such fixtures. The
available runner has no verified private Note provisioning/visibility harness,
so `--suite live --allow-live` is deliberately blocked, not passing evidence.
When a harness is available, select the Note live suite to compare detail tokens
and complete file bytes, verify unauthorized identity/scope errors on those
fixtures, and clean up only objects created by that run. No other recipient is
needed; any collaboration setup remains limited to unambiguously resolved Ying
Wu. Do not use unrelated existing meeting artifacts as substitute fixtures.

Coordinator schedules comparative development tasks across the three pinned
arms. Held-out tasks remain untouched. Required live/comparative results and the
cross-section VC/Docs handoff keep final section acceptance open.
