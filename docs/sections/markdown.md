# Markdown source files — P2 section evidence

Owner: Markdown. Frozen reference: `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`
in the unchanged sibling `feishu-cli-review`. The five shortcut implementations
were identical to that snapshot before this section's changes. Registration
contracts are `baseline/registrations.json.gz#/feishu/commands/markdown*`
(and the corresponding Lark entries); coverage is the six `markdown` entries
including the parent group in `baseline/coverage.json`. No Markdown-only
generated OpenAPI methods or events exist in that group.

## Complete mapping and owners

Every mapping retains the command name and supports user and bot identities,
JSON/pretty formatting, jq/json compatibility flags, profiles, and dry-run.
Shared next discovery and result controls remain owned by the foundation.

| Reference capability → same next command | Inputs and preserved behavior | Outputs and behavioral evidence |
| --- | --- | --- |
| `markdown +create` | Exactly one of inline/`@file`/stdin content and local `.md` file; name requirements; Drive root/folder or Wiki parent; URL/token normalization and ambiguous-target rejection; nonempty content; upload_all through 20 MiB and prepare/parts/finish above it; bot current-user permission grant | file token, name, byte size, real metadata URL and permission-grant outcome; inherited upload/validation/grant units, dry-run target/body checks, process root/folder/Wiki and multipart byte checks |
| `markdown +fetch` | Latest source preview (`preview_type=16`); content or local file/directory; remote-name path sanitization; existing output refusal and explicit overwrite; invocation FileIO | Full text/byte size or saved path/name/size; JSON escaping and pretty text preserved; inherited FileIO/error units and process Unicode/CRLF/source/save checks |
| `markdown +diff` | Numeric from/to versions preserved as strings, historical-to-latest, historical-to-historical, remote-to-local; configurable nonnegative context; JSON/pretty only; 10 MiB input limit and line-based diff | Changed predicate, labels, mode, versions, added/deleted counts, hunks, unified diff; no-final-newline marker; inherited multi-hunk/limit/error tests and process versions/leading zero/local equality checks |
| `markdown +patch` | Literal all-match replacement or RE2 regex/capture expansion; file/stdin inputs; empty replacement allowed unless it empties the whole file; fetch/replace/overwrite; preserves remote name | Updated predicate, match count, before/after sizes and returned version; inherited literal/regex/deletion tests; new exact-byte no-op units and process read/no-upload and changed content checks |
| `markdown +overwrite` | Explicit version upload; inline/file inputs, explicit/local/remote/fallback names; nonempty content; both upload paths; upload and metadata scopes unchanged | File token/name, string version (including data_version fallback), byte size; inherited name/version/multipart/error units and process local/inline/identical-content version checks |

Native document conversion is a separate Docs-owned dependency, not an implicit
operation of these file commands. The reference `shortcuts/doc/docs_fetch_im_markdown.go`
and its tests cover `docs +fetch --output-format im-markdown`: title, paragraph,
heading, lists, table, grid/column, callout, blockquote, image/resource links,
bookmark, sheet/whiteboard, user/document/citation references, tag handling,
recursive nesting, escaping, Unicode and malformed fragments. The Docs owner
retains that implementation and its conversion regressions; this section does
not edit or claim independent acceptance of those shared/native representations.
Markdown source commands store rich syntax verbatim rather than converting it.

## Deliberate changes

- A patch whose final bytes equal the downloaded bytes returns `updated:false`
  while retaining the actual match count. It performs no metadata lookup or
  upload. Literal equality, capture identity and zero-width empty replacements
  have revert-failing unit coverage.
- Upload-all, prepare, part and finish no longer retry network failures, 5xx or
  arbitrary retryable backend errors: their effects may already have happened.
  Only explicit gateway rate-limit rejection remains retryable. The typed cause
  is preserved. Process coverage exercises disconnect-after-dispatch, 502 and
  503, asserts exactly one upload, exit 22 and unknown receipt evidence.
- Metadata-name lookup reuses the shared typed, identity-matched, read-only
  `common.FetchDriveMetaTitle` helper (shared commit `5179287`). Read retries and
  missing-metadata filename fallback remain. A metadata POST no longer creates
  a false mutation receipt; the process lifecycle asserts one receipt for the
  actual create upload.
- Existing dry-run assertions now inspect JSON values, independent of pretty
  whitespace, without weakening request assertions. New command affordance
  explains source/native distinctions, capture replacement and write semantics.

`+overwrite` explicitly uploads a version even for equal content. Adding a
mandatory download comparison would change its existing scope requirements;
the coordinator chose to preserve that explicit operation. `+patch` already
reads the source and suppresses unchanged writes. Neither operation has a
backend conditional-version guard: another client's changes between read and
write may be overwritten. Receipts do not provide concurrency protection.

`schema` remains the reference generated-API catalog interface and excludes
shortcuts; `discover markdown` and each shortcut's `--help` expose all five
operations. Discovery coverage follows `meta.next_offset` until `meta.complete`
and asserts exact command membership; public flag matches can place Markdown
commands beyond the first page. Tests do not invent an unsupported
shortcut-schema contract.

## Verification and remaining gates

The section suite uses real CLI subprocesses through
`nextfixture.RunMain(m, "../../..")` and `NewFixture`, with isolated config and
synthetic local responses only. Source contents include Markdown styles, table,
checkbox, blockquote, image/link, fenced code, escaped punctuation, literal dollar
signs, leading-zero strings, Chinese, emoji, CRLF and a missing final newline.
The multipart fixture compares every byte of a payload over 20 MiB.

- Full offline boundary: `python3 scripts/next_test.py --section markdown --suite all --report .tmp/markdown-section.json` **passed: 124 unit test events, 11 inherited dry-run tests and 15 process test events**. The sole unit skip is the oversized-block-size guard that requires an `int` narrower than `int64`; this host is 64-bit.
- Guidance checks: isolated `go test -json -count=1 ./internal/affordance ./cmd/service ./internal/schema` **passed: 275 test events, no failures or skips**.
- Scoped `go vet ./shortcuts/markdown/... ./tests/next_sections/markdown`: **passed**.
- Live gate: `python3 scripts/next_test.py --section markdown --suite live --allow-live --report .tmp/markdown-live.json` returns **blocked**, exit 2. No verified fixture harness establishes effective account/tenant and creation visibility before content writes. No live objects were created or modified.

The inherited selectable `TestMarkdownLifecycleWorkflow` and
`TestMarkdownCreateWorkflow_WikiParent` describe the live lifecycle and Wiki
creation but are excluded from ordinary isolated CI; their legacy opt-in flag
alone is not authorization. Before running either, a live fixture must verify
current account and tenant, private creation visibility (including inherited
Wiki access), run-owned cleanup, and any recipient binding. A name match cannot
identify the allowed other collaborator from private local configuration. Never create first and
restrict visibility afterward. Live multipart/version/concurrency behavior and
permission grants remain unverified until those conditions can be met.

Comparative development evaluation is coordinated by the root owner one balanced
workload at a time. It remains pending here; held-out tasks are untouched. These
live and evaluation gaps keep P2 acceptance open despite offline parity evidence.

Whole-repository pre-PR checks (`make unit-test`, vet, formatting, module tidy,
license policy and committed quality gate) are coordinated by the root owner;
these section results do not stand in for those release checks.

P3 discovery follow-up: the full isolated Markdown boundary passed again with
124 unit passes (one architecture-only skip), 11 inherited dry-run passes and
15 process passes after discovery began matching public flags. Evidence:
`.tmp/markdown-discovery-boundary.json`. The process test follows continuation
metadata and checks exact command membership without assuming first-page
completeness or a particular ranking. No product behavior changed.

## Bounded private live continuation — 2026-09-08

The earlier no-fixture live blocker was resolved for one current-user source file
and one native rendered document under the coordinator's private personal parent.
`TestMarkdownNextPrivateLiveLifecycle` repeats independent account/tenant binding,
parent closed sharing, sole owner/container membership and metadata ownership
before writing, rechecks parent access before each phase, and verifies each
returned child's ACL. It uses explicit user identity and single-attempt commands;
returned IDs are persisted and guarded cleanup is registered before success
assertions, including partial/unknown CLI results. The parent is never deleted.

Both phases **passed**. `markdown +create` uploaded one `.md` file to the explicit
folder; `markdown +fetch` read the preview-type-16 source and saved it with exact
byte equality, including Unicode, Markdown markup and leading-zero text. Native
rendering is the separate Docs dependency: `docs +create --doc-format markdown`
created one document in the same parent, and XML/native block reads verified the
heading, bold, link URL, table, Unicode, leading-zero content and code block.
These file/source and native rendering results are not conflated.

Both run-owned children were deleted once. Independent exact-ID read evidence
confirmed deletion: file permission auth returned API 1063005, `Resource is
deleted`; native Docx GET returned API 1770003, `resource deleted`. Empty metadata
URLs were not accepted as proof. All command stdout/stderr, including failures,
was retained in owner-readable ignored evidence before assertions.

The initial invocation's two create commands failed local FileIO validation
because the source was outside the Go test package's working-directory allowlist;
no business create was dispatched. That failure remains recorded. Setting the
child request's working directory to the private evidence directory corrected
only the test harness; product file guards and runtime code were unchanged.

Select the dedicated test with `NEXT_MARKDOWN_LIVE=1`, an explicit `LARK_CLI_BIN`,
and absolute `NEXT_MARKDOWN_LIVE_FIXTURE`, `NEXT_MARKDOWN_LIVE_BINDING`, and
`NEXT_MARKDOWN_LIVE_EVIDENCE_DIR` paths supplied by the coordinator. It creates at
most the two described resources. No new scope, login, Wiki/shared space, bot,
recipient, version/patch operation or model task is involved. See
[the sanitized evidence](../../evidence/sections/markdown-private-live-evidence.json) for byte hashes,
command counts and retained failure hashes; private IDs are omitted.

This narrows the live gap but does not close full P2 acceptance. Multipart,
version/overwrite/patch, concurrency, bot grants, Wiki inheritance and complete
native conversion/type parity remain separate unverified gates. No runtime or
evaluation behavior changed.
