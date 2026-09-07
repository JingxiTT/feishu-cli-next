# OKR section

Status: offline implementation and section verification ready; P2 remains open for
live backend verification and coordinator-run comparative development tasks.
Reference: `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; shared foundation accepted
at `ce68188`. Source reconciliation verified all 43 frozen OKR implementation and
legacy test files against `docs/reference/sources.json` and the unchanged sibling.

## Scope, ownership and dependencies

The section owns `shortcuts/okr/`, `tests/cli_e2e/okr/`,
`tests/next_sections/okr/`, and this report. It reuses the complete reference
implementation, typed ContentBlock converters, runtime/FileIO, generated service
machinery, and existing unit and legacy dry-run tests. No new command forwarding,
permissions, or endpoint inventory was invented. There is no matching existing
OKR affordance file or OKR event projection to migrate.

The frozen registration set has 57 paths: the service root, 22 shortcuts,
11 resource groups, and 23 generated methods. The process discovery test checks
every path and visible option. `--profile` is deliberately hidden by the shared
authsidecar build, whose variant contract differs from normal builds; normal
legacy dry-run tests retain the profile-aware executable. Root assembly, generic
metadata, output/receipt contracts, events and distribution remain coordinator
or corresponding section responsibilities.

## Complete shortcut mapping

All rows retain their reference command, risk, identities, scope and flags.
Shared execution/output options are covered by foundation; domain options below
come directly from the frozen registrations. Tests are owned by this section:
`shortcuts/okr/*_test.go`, the legacy suite, and process scenarios described below.

| Shortcut | Domain options |
| --- | --- |
| `okr +batch-create` | `--category-id`, `--cycle-id`, `--input`, `--json`, `--user-id-type` |
| `okr +comment-create` | `--content`, `--json`, `--ref-comment-id`, `--select-all`, `--selected-text`, `--style`, `--target-id`, `--target-type`, `--user-id-type` |
| `okr +comment-delete` | `--comment-id`, `--json` |
| `okr +comment-detail` | `--cycle-id`, `--json`, `--style` |
| `okr +comment-get` | `--comment-id`, `--json`, `--style`, `--user-id-type` |
| `okr +comment-list` | `--json`, `--page-size`, `--page-token`, `--style`, `--target-id`, `--target-type`, `--user-id-type` |
| `okr +comment-patch` | `--comment-id`, `--content`, `--json`, `--style`, `--user-id-type` |
| `okr +comment-reopen` | `--comment-id`, `--json`, `--style`, `--user-id-type` |
| `okr +comment-solve` | `--comment-id`, `--json`, `--style`, `--user-id-type` |
| `okr +create` | `--category-id`, `--content`, `--cycle-id`, `--json`, `--level`, `--notes`, `--objective-id`, `--style`, `--user-id-type` |
| `okr +cycle-detail` | `--cycle-id`, `--json`, `--style` |
| `okr +cycle-list` | `--json`, `--page-size`, `--page-token`, `--time-range`, `--user-id`, `--user-id-type` |
| `okr +indicator-update` | `--id`, `--json`, `--level`, `--value` |
| `okr +patch` | `--content`, `--deadline`, `--json`, `--level`, `--notes`, `--score`, `--style`, `--target-id`, `--user-id-type` |
| `okr +progress-create` | `--content`, `--json`, `--progress-percent`, `--progress-status`, `--source-title`, `--source-url`, `--style`, `--target-id`, `--target-type`, `--user-id-type` |
| `okr +progress-delete` | `--json`, `--progress-id` |
| `okr +progress-get` | `--json`, `--progress-id`, `--style`, `--user-id-type` |
| `okr +progress-list` | `--department-id-type`, `--json`, `--page-size`, `--page-token`, `--target-id`, `--target-type`, `--user-id-type` |
| `okr +progress-update` | `--content`, `--json`, `--progress-id`, `--progress-percent`, `--progress-status`, `--style`, `--user-id-type` |
| `okr +reorder` | `--cycle-id`, `--json`, `--level`, `--objective-id`, `--ops` |
| `okr +upload-image` | `--file`, `--json`, `--target-id`, `--target-type` |
| `okr +weight` | `--cycle-id`, `--json`, `--level`, `--objective-id`, `--weights` |

## Generated methods

`TestOKRGenericOperations` executes all 23 frozen methods through a real CLI
process, verifies method/path/body at the synthetic backend, preserves large ID
strings and response types, and verifies each dry run makes no business request.
Generated schema, identity/scope, pagination flags and less-common query options
are reused from the frozen catalog, not replaced by shortcuts.

- `okr alignments delete`
- `okr alignments get`
- `okr categories list`
- `okr cycle.objectives create`
- `okr cycle.objectives list`
- `okr cycles list`
- `okr cycles objectives_position`
- `okr cycles objectives_weight`
- `okr indicators patch`
- `okr key_result.indicators list`
- `okr key_results delete`
- `okr key_results get`
- `okr key_results patch`
- `okr objective.alignments create`
- `okr objective.alignments list`
- `okr objective.indicators list`
- `okr objective.key_results create`
- `okr objective.key_results list`
- `okr objectives delete`
- `okr objectives get`
- `okr objectives key_results_position`
- `okr objectives key_results_weight`
- `okr objectives patch`

## Content and workflow preservation

The inherited 300+ top-level unit tests cover ContentBlock and V1/V2 conversions,
list numbering/indentation, paragraphs, galleries, document links, mentions,
text links, bold/strike-through, RGBA colors, fractional image dimensions,
Unicode, timestamps, categories, user/department owners, score and weight values,
progress statuses and source metadata. JSON/file/stdin-capable inputs retain the
runtime resolver. `TestOKRRichContentFileAndArtifact` independently checks a rich
file request and a saved cycle artifact with every supported content element,
false/zero values, Unicode, links, mention IDs and image tokens intact.
Rich cycle/progress output retains its documented serialized ContentBlock string;
simple output remains an explicit domain conversion, not a full-fidelity export.

`TestOKRShortcutDispatch` covers standalone read/write dispatch, objective and KR
levels, content/notes/category/score/deadline, progress percentages/status/source,
comment selection/reply and user_key identity options, and distinct comment
solve/reopen/delete actions. `TestOKRCompositionAndNoop` covers both levels of
reorder, weight normalization and indicator lookup/update. Cycle/comment detail
exercise hierarchy traversal, progress target discovery and thread grouping.
Image upload verifies multipart bytes and progress on stderr. Batch creation
checks file input, created IDs, successful rollback after rejection, and a lost
KR response without automatic create replay. Receipt evidence remains conservative
about unknown effects even when the command attempts its documented rollback.

## Intentional behavior changes

- Complete-collection readers reject `has_more:true` with a missing or repeated
  continuation token. This prevents silent truncation and infinite reads before
  reorder/weight replacement and cycle/comment-detail output. Tokens remain
  opaque. Complete two-page traversal is independently checked.
- Objective, KR, cycle and progress collection decoding now fails on malformed
  items instead of silently skipping them. Errors preserve typed causes.
- Reorder avoids a write only when all known positions and requested IDs exactly
  match; absent position evidence still uses the original write. Weight replacement
  avoids a write only when every current weight is known and exactly matches the
  complete normalized result. Both add `noop` to output and retain resource IDs.
  No-op does not claim snapshot consistency or protection from another client.
- Two inherited dry-run tests now parse `user_id_type` instead of depending on
  spaces in JSON; compact foundation output preserves the same values.

Direct objective/KR patches, progress updates and comment patches keep their
write-only permission contracts. They do not fetch current state merely to detect
a no-op, so identical replacement content can still dispatch. Adding mandatory
read scopes would regress supported write-only use; backend equality/no-notification
semantics remain unverified. Create, append, solve/reopen, delete and upload effects
retain their explicit reference behavior and are not deduplicated by content.

## Evidence and open acceptance work

Run the complete section with:

```sh
python3 scripts/next_test.py --section okr --suite all --report .tmp/okr-section.json
```

Final complete boundary run: 349 passing test events in unit tests, 42 passing test events in legacy dry-run/regression tests, 129 passing test events in new process tests. The process suite has one explicit blocked-live skip and no failures.

This uses isolated config/home, placeholder credentials, blocked external proxies
and a non-forwarding local backend. The process package builds the production
`authsidecar` entry point through `nextfixture.RunMain(m, "../../..")`. Unit,
legacy `DryRun|Regression`, and all new process tests run separately; the report
records counts, exits and every excluded legacy live test. Successful deterministic
runs are offline evidence only. The report is local sanitized build evidence;
source tests and this report are version controlled.

Live verification remains blocked: no current-account/tenant-bound, demonstrably
private OKR cycle and run-owned objective/progress/comment cleanup target has been
established; notification visibility for OKR writes also needs backend evidence.
`TestOKRLivePrivateLifecycle` is explicitly selectable and skipped with these
prerequisites; inherited live lifecycle implementations remain excluded by the
runner. No live read/write, mention, notification or cleanup was attempted.
Existing unrelated OKRs cannot be used as write fixtures, and a display-name match
for Ying Wu is not recipient verification. Live-dependent content, notification,
permission, visibility, pagination and cleanup guarantees remain unverified.

Comparative development tasks, token accounting and cross-section identity/artifact
handoffs are coordinator-owned and not run here. Held-out tasks remain untouched.
Scoped `go vet ./shortcuts/okr/... ./tests/next_sections/okr/...` passed under
`isolated_env`; scoped `git diff --check` passed. Broad shared-tree
unit/vet/lint/quality gates are coordinator-owned; this section
reports only its scoped checks, not full release acceptance.

## P3 malformed native-ID review

Independent review found that duplicate native IDs in a fetched collection could
collapse the reordered ID list and panic in its no-op comparison. The isolated
`TestReorderRejectsDuplicateNativeIDs` regression reproduced an index-out-of-range
panic before the fix (`.tmp/okr-duplicate-repro.json`).

Reorder now validates native IDs before constructing the replacement order;
weight replacement validates before matching assignments or normalizing weights.
The complete fetched snapshot must contain distinct, positive int64 IDs. Missing,
nonnumeric, nonpositive, overflowing and repeated IDs fail with typed
`internal/invalid_response`, retaining a parsing cause where available and a
read-again recovery hint. Valid ID strings remain verbatim. A corrupt collection
is never downgraded to an ordinary write or a claimed no-op. Validation is scoped
to these whole-collection mutations; read-only domain behavior is unchanged.

Process regressions exercise both operations at objective and KR levels, with
same-page and cross-page duplicates and each malformed-ID case. They independently
assert zero writes, the exact error category/subtype and exit, empty stdout, and
bounded reads. Existing successful KR test fixtures now use numeric native IDs
instead of alphanumeric placeholders while preserving all tested relationships.
Evaluation contracts, runtime fixtures and concise guidance were not changed.

Validation after the fix: `python3 scripts/next_test.py --section okr --suite all
--report .tmp/okr-duplicate-section.json` passed with 357 unit, 42 legacy and 154
process test pass events; the existing blocked-live skip remains explicit.
Scoped isolated `go vet ./shortcuts/okr/... ./tests/next_sections/okr/...` and
`git diff --check` passed. No live actions or model evaluations were run.
