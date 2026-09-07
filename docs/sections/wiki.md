# Wiki section

Status: implementation and complete offline section suite pass; live, cross-section content verification and comparative acceptance remain open.

Reference: sibling `../feishu-cli-review` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; all Wiki shortcut source initially byte-identical. Frozen `coverage.json` and `registrations.json.gz` contain the following complete command mapping. Every command retains its original name, flags, identity, risk and registered schema; shared opt-in result controls are additive.

| Reference and next command | Retained behavior | Evidence owner |
| --- | --- | --- |
| `wiki` | Navigation / discovery group | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +delete-space` | Delete whole space; sync/async task status, confirmation, resume hint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +member-add` | Grant all seven member ID kinds and both roles; notification omitted/false/true; personal-library resolution | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +member-list` | Members, role/type identity, explicit cursor and bounded auto-pagination | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +member-remove` | Revoke exact ID/type/role tuple; retain input identity if backend omits echo | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +move` | Move node with source/target resolution, or all seven Drive object kinds into Wiki; apply-for-approval and async polling | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +move-to-drive` | Move to folder or personal root; preserve backing object token/type; async status and read-only continuation | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +node-copy` | Copy one node, preserving original subtree; mutually exclusive space/parent destination, optional title, lock-only retries | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +node-create` | Origin and shortcut nodes; six object kinds including file shortcuts; explicit space/parent/my_library; bot current-user auto-grant | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +node-delete` | Resolve Wiki/document token or URL; explicit space; cascade or lift children; confirmation and sync/async delete | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +node-get` | Wiki or backing-object token/URL; all seven kinds; deprecated alias; asserted space; native identifiers and creator/timestamp metadata | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +node-list` | Root or parent navigation; numeric/my_library space; URL parent normalization; bounded cursor pagination | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +space-create` | Named space and description, user identity only; normalized string flags; flattened space metadata | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki +space-list` | Accessible spaces with visibility/sharing metadata and bounded cursor pagination (my_library resolved separately) | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki members` | Navigation / discovery group | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki members create` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki members delete` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki members list` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki nodes` | Navigation / discovery group | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki nodes copy` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki nodes create` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki nodes list` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki spaces` | Navigation / discovery group | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki spaces create` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki spaces get` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki spaces get_node` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |
| `wiki spaces list` | Generated typed service endpoint | `shortcuts/wiki`, `tests/cli_e2e/wiki`, `tests/next_sections/wiki` |

Dependencies: shared runtime/auth/output/receipts; Drive backing document permission auto-grants and Drive/Wiki moves; Docs/Sheets/Base own the backing resource content operations. Wiki has no event projection in the frozen tree. No Wiki content replacement/update endpoint is registered; unchanged-content replacement no-op is not applicable here. Create, copy, move, permission grants, notification and delete are explicit effects, never deduplicated on matching titles.

Acceptance scenarios: every registered leaf executes against a synthetic backend; navigation and personal-library resolution; every create/move/get/delete object kind; origin versus shortcut identity; member kinds and roles, explicit notification false/true/omission; bounded and cyclic pagination with continuation; ambiguous input, identity and authorization failures; copy/move/delete cleanup, task polling and safe read-only continuation; unknown/partial receipts without automatic mutation replay; full result artifact and native identifiers. Reference units retain lock-contention-only retries, context cancellation, polling timeouts and subtree semantics.

Object kinds remain `doc`, `docx`, `sheet`, `bitable`, `mindnote`, `slides`, and `file` where supported. Origin creation excludes legacy `doc` and `file`; `file` creation requires shortcut mode plus an origin token. Member ID kinds remain `openid`, `userid`, `email`, `unionid`, `openchat`, `opendepartmentid`, and `appid`; roles remain `admin` and `member`. Bot department grants and bot `my_library` remain invalid. These are reference API contracts, not test-recipient restrictions.

The generic methods preserve their original JSON body/query schemas, including `lang`, explicit `need_notification:false`, original empty/Unicode/whitespace content, source/destination IDs, and nested native response values. Generated `nodes copy` accepts both target fields and does not require `--yes`; the ergonomic shortcut intentionally requires one destination and confirmation, unchanged from the reference. Generic member writes and space creation retain their high-risk confirmation gate.

## Intentional adaptation

`d7c1624` prevents space/node/member `--page-all --page-limit 0` from looping on a repeated or cycling backend cursor. The already received records and final cursor remain available; `has_more:true` and a stderr diagnostic explicitly retain the incomplete extent. Default one-page, explicit-cursor precedence and bounded page-limit behavior are preserved. This is not snapshot consistency or deduplication across concurrent remote edits.

Full output and artifacts use the shared runtime with native resource identities retained. Wiki read artifacts are evidence containers; this section introduces no import consumer. Backing document content and formatting remain the responsibility of the corresponding Docs/Sheets/Base/Drive operations rather than being reserialized by Wiki navigation.

## Live fixture and remaining acceptance

`python3 scripts/next_test.py --section wiki --suite live --allow-live --report .tmp/wiki-live.json` returned **blocked**, exit 2. The selectable `TestWikiAuthorizedLiveFixture` likewise skips with the required fixture conditions. No live write was attempted. Reference tests such as `TestWiki_NodeWorkflow` choose any available host space and may inherit other-user visibility; they are retained as source evidence and excluded from the offline runner, never counted as passes.

A live run must first prove effective account and tenant plus private creation visibility for a run-owned Wiki space and both sides of Drive transfers. It must verify Ying Wu's identity before any optional collaborator grant, own cleanup of only newly created objects, and independently inspect navigation, backing-object identity and content, permissions and async results. Until that fixture exists, those backend-dependent guarantees remain unverified. Comparative development workloads and cross-section content transfer are coordinated separately; no comparative result or efficiency claim is made here, and held-out tasks are untouched.

No generic async receipt reconciliation is claimed. Async task acceptance remains conservatively unknown even when a separate Wiki poll reports `ready:true`; read-only task lookup preserves the task identifier and never repeats the mutation. Space/node deletion, transfers, failed grant after node creation, transport loss and scope rejection must retain the shared outcome/exit contract.

## Offline suite and checks

Run `python3 scripts/next_test.py --section wiki --suite all --report .tmp/wiki-all.json` after a canonical `make build`. The runner strips personal configuration and credentials, blocks outbound proxy traffic, and builds the real authsidecar process used by `tests/nextfixture`. Each process fixture owns temporary config/work directories and an HTTP server which never forwards requests. The JSON report records individual test pass/fail/skip events plus excluded tests.

The suite includes all `shortcuts/wiki` units, all inherited `DryRun|Regression` cases, and all `tests/next_sections/wiki` tests. Four pure helper tests were renamed with the `Regression` suffix to keep deletion-target identity and transient-result checks in the existing selector. The only excluded legacy workflows are Drive download/preview of Wiki sources, Wiki-to-Drive move, file shortcut creation, shortcut lifecycle and node lifecycle; their live fixtures do not prove the PLAN visibility boundary.

`go vet ./shortcuts/wiki/... ./tests/next_sections/wiki` ran through `scripts/next_test.py`'s `isolated_env` and passed. `gofmt -l shortcuts/wiki tests/next_sections/wiki tests/cli_e2e/wiki/helpers_test.go` returned no files. Repository-wide static and release gates are coordinated by the root owner; these focused results do not stand in for them.

Final section boundary: `.tmp/wiki-all.json` records **264 unit pass events, 36 inherited deterministic pass events, 96 process-suite pass events, zero failures and one explicit blocked-live skip**. Five inherited live workflows are excluded by name and remain unverified. The process suite exposed and now verifies the shared Cobra stderr correction (`8c2b5a4`) and concrete Wiki async-receipt classification. These shared changes are owned by the foundation coordinator; the Wiki suite retains the regression assertions. The lifecycle fixture also synchronizes server state before independent effect assertions; its focused final rerun is recorded in `.tmp/wiki-lifecycle-final.json`.

Owned changed files: the three Wiki list implementations and `wiki_pagination_test.go` (`d7c1624`); `tests/cli_e2e/wiki/helpers_test.go`; `tests/next_sections/wiki/{wiki_e2e,catalog_e2e,async_e2e}_test.go`; this section report. Full reference workflows were reused. No new affordance, public plugin API, event, root or generic command implementation was introduced by this section.
