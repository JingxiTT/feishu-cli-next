# Contact section

Status: offline implementation and process verification passed; live backend
verification and comparative development tasks remain open. Reference:
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3` in the unchanged sibling
`feishu-cli-review`; P1 interfaces accepted by the coordinator at `ce68188`.

## Ownership and complete mapping

The frozen `coverage.json` and both Feishu/Lark trees in
`registrations.json.gz` contain six Contact nodes: the `contact` and
`contact user_profiles` discovery groups and the four executable leaves below.
The frozen `meta_data.json.gz` Contact service contains one generated endpoint,
`POST /open-apis/profile/v2/user_profiles/batch_query`. Source inspection included
all files under the sibling `shortcuts/contact/`, current inherited Contact E2E,
and the frozen catalog's complete request/response schema. Before the change,
`git diff --no-index ../feishu-cli-review/shortcuts/contact shortcuts/contact`
was empty. There is no registered department CRUD or Contact event projection;
department IDs/profile attributes and projected search department names remain
supported. Arbitrary department endpoints remain the existing foundation-owned
`api` surface, not an additional Contact registration.

| Reference → next mapping (same suffix) | Complete capability and options | Evidence owner |
| --- | --- | --- |
| `contact +get-user` | Self user-info GET; user basic-batch POST for one ID; bot full-profile GET. `--user-id`, `--user-id-type` (`open_id`, `union_id`, `user_id`), user/bot identity, JSON/table output, dry-run. Full returned profile fields remain unfiltered by the shortcut. | Existing `contact_get_user_test.go`, `helpers_legacy_test.go`; process identity-route and full-profile test, self lookup and dry runs. |
| `contact +search-user` | Keyword/email; open-ID CSV and `me`; query+IDs restriction; filter-only; four boolean filters; locale override; page-size 1–30; up to 100 IDs and 50-rune keywords. `--queries` deduplicates up to 20 terms, runs at most 5 workers, retains ordered matches and per-query errors/extent/notices. | Existing `contact_search_user_test.go`; process all-filter request, self-ID resolution, same-name candidates, partial/all-failed, scope failures, invalid input and dry runs. |
| `contact +search-bot` | Query or fanout; up to 100 chat IDs; chatted filter; page-size 1–30; 50-rune keywords and 20-term/5-worker fanout. Terminal authorization/context/internal failures retain error exits; recoverable partial results retain summaries. | Existing `contact_search_bot_test.go`, `contact_search_bot_fanout_test.go`; process scoped request, bot metadata, partial/all-failed and dry runs. |
| `contact user_profiles batch_query` | Generated user-only read with ID type, `user_ids` array, both `query_option` booleans. Personal-status ID/title/i18n/icon/interval/DND boolean and description default locale/value/i18n all pass through, including null/empty/string distinctions. `--data`, `--params`, output/format/jq and shared result controls remain generated machinery. Hidden generic page flags remain registration-compatible; the endpoint has no declared pagination contract. | Frozen catalog; process request/body routing, typed Unicode/nested/empty result, exclusive artifact preservation and dry-run. |

All leaves retain their registered read risk, identities, scopes and original
flags; no raw-forwarding replacement or new convenience wrapper is introduced.
Shared profile selection, JSON/NDJSON/CSV/pretty formatting, jq, output files,
artifact binding, and auth live with their foundation owners. Contact units
cover format-specific projections and stderr hints. Process fixtures prove that
these read-only POST endpoints do not create mutation reports.

## Preserved semantics and deliberate correction

Search is bounded to one backend page per query. `has_more` is always retained;
fanout reports it separately for each query. Neither search accepts a page token
or auto-pagination, so backend tokens remain intentionally hidden as in the
reference. Empty arrays are arrays, not null. Name duplicates remain separate
candidates with stable IDs; neither an apparent name match, result order,
department string nor `is_cross_tenant:false` establishes recipient identity.
Search never sends content or grants access. A caller composing delivery must
independently resolve the intended person and destination.

Search user fields preserve open ID, localized name (explicit locale, brand,
fixed fallbacks, deterministic locale order, then ID), personal and enterprise
email, activated/cross-tenant booleans, P2P ID/chatted fact, department, signature,
recency hint and matched segments. Bot fields preserve ID, name, description,
P2P chat ID even when empty, join-group and agent booleans, tenant ID and matched
segments. Unknown backend search fields are not newly advertised: these remain
the reference's intentionally projected workflows; full-profile and generated
profile reads retain their supported complete shapes.

One defect is corrected: when every user-search fanout query fails due to
permissions or authentication, the aggregate now retains the original typed
category/subtype, API code, log ID, missing scopes, recovery hint and cause.
Previously these failures became `internal/unknown` and exit 5. Permission
failures now retain authorization exit 3. The representative error is copied,
not mutated. Successful, empty, recoverable-partial, and bot-terminal behavior
is unchanged. Nearby regression tests fail when this fix is reverted.

There are no replacement/mutation/no-op/unknown-write workflows in these four
read-only leaves. Cancellation and the five-worker bound are inherited unit
contracts; partial results and no automatic retries are additionally checked at
the CLI process boundary. Read-side server faults are not mutation outcomes.

## Executable acceptance scenarios

`tests/next_sections/contact/contact_e2e_test.go` uses
`nextfixture.RunMain(m, "../../..")` and `NewFixture`: a real authsidecar build,
private config/work directories, a local synthetic backend, no live forwarding,
per-command timeout, exact request assertions and independently observed request
counts. The suite covers all four leaves, both supported get-user identities,
all three get-user ID types, self lookup, Unicode/empty/nested fields, explicit
same-name ambiguity, search extent, deduplicated fanout partial/all failures,
missing scopes, request validation, dry-run isolation, result artifact account
and tenant binding and collision protection. Existing unit tests add the full
validation limits, locale fallbacks, error causes, API malformed/transport
responses, formatting, worker limits and cancellation.

Run at the complete section boundary:

```sh
python3 scripts/next_test.py --section contact --suite all --report .tmp/contact-section.json
python3 scripts/next_test.py --section contact --suite live --allow-live --report .tmp/contact-live.json
```

The runner restores no personal credentials, uses synthetic fixture data and
writes machine-readable pass/fail/skip outcomes. The inherited legacy selector
`DryRun|Regression` currently selects zero Contact tests; that is explicitly no
legacy live evidence. The new process suite supplies deterministic coverage.

## Live and comparison limitations

`TestContactLiveAccountBoundReads` records the offline suite’s live blocker.
`tests/cli_e2e/contact/contact_next_live_test.go` supplies the executable opt-in
`TestContactNextLiveCurrentAccountReads` workflow. It requires
`NEXT_CONTACT_LIVE=1`, independently established `NEXT_CONTACT_LIVE_ACCOUNT_ID`
and `NEXT_CONTACT_LIVE_TENANT_KEY`, and the legacy harness’s explicit live user
credential configuration. It verifies self ID/tenant before follow-up user
reads, round-trips the account, checks ID-restricted search/profile responses
and runs a bounded bot search. It neither writes nor logs personal profiles.
Select it only in the coordinator’s verified live environment with
`go test ./tests/cli_e2e/contact -run ^TestContactNextLiveCurrentAccountReads$ -count=1`.
The normal isolated runner intentionally excludes it. Empty self searches are
reported as unverified completeness, not proof of successful identity resolution.
Bot-identity tenant verification remains a separate blocker. The legacy
`TestContact_LookupWorkflowAsUser`, `TestContact_LookupWorkflowAsBot` and bot-search
workflows remain executable in their original package but must not run with
personal credentials through the offline runner. Legacy bot discovery picks a
user from a department listing and does not establish the current account/tenant
binding required by this plan. Existing Contact coverage notes also record that
self-derived user searches were unreliable in UAT; empty search success cannot
prove identity resolution.

The required authorized live batch should bind the effective current account
and tenant with read-only self info, round-trip its open ID through user and bot
get-user where scopes permit, read that account's personal status/signature, and
exercise user/bot search without minimum-match assumptions. Capture only
sanitized shape/identity-equality evidence, never personal profiles in committed
logs. Contact needs no content creation or notification. Any subsequent delivery
is a cross-section action, requires independently verified Ying Wu identity or
current-account-only private visibility, and is outside these Contact reads.
No tenant-bound live harness or verified current-account fixture was supplied;
these checks are blocked and are not passing evidence. No comparative or held-out
tasks were run by this section; the coordinator schedules them.

## Recorded offline results

Final complete-section command:
`python3 scripts/next_test.py --section contact --suite all --report .tmp/contact-section.json`
passed with 228 Contact unit test events, 21 process test events and one explicit
live-blocker skip. The legacy `DryRun|Regression` selection compiled successfully
but ran zero tests. Raw synthetic machine-readable results are in the report
path (ignored local evidence). The live command above returned `blocked`, exit
2, recorded in `.tmp/contact-live.json`; no live requests were made.

`go vet ./shortcuts/contact/... ./tests/next_sections/contact ./tests/cli_e2e/contact`
was run with `scripts/next_test.py`'s `isolated_env` helper. `gofmt` and
`git diff --check` cover the owned changes. Full repository unit/vet/format,
quality-gate, licensing/module checks, normal-build variants, comparative tasks
and held-out evaluation are coordinator/release checks and were not run by this
section owner.
