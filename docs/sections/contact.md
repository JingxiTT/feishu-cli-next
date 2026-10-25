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
is a cross-section action, requires independently verified configured recipient identity or
current-account-only private visibility, and is outside these Contact reads.
An initial missing-fixture blocker was superseded by the coordinator-authorized
current-account live attempt recorded below. That attempt remained open after its final bot-search read failed; the
authorized resumption below subsequently passed the selected workflow. Comparative development evidence is
owned by the coordinator under `evidence/evaluation/contact/`; this section owner
did not run model evaluations or held-out tasks.

## Recorded offline results

Final complete-section command:
`python3 scripts/next_test.py --section contact --suite all --report .tmp/contact-section.json`
passed with 228 Contact unit test events, 21 process test events and one explicit
live-blocker skip. The legacy `DryRun|Regression` selection compiled successfully
but ran zero tests. Raw synthetic machine-readable results are in the report
path (ignored local evidence). The live command above returned `blocked`, exit
2, recorded in `.tmp/contact-live.json`; that initial offline-boundary invocation
made no live requests. The later authorized live attempt is separate below.

`go vet ./shortcuts/contact/... ./tests/next_sections/contact ./tests/cli_e2e/contact`
was run with `scripts/next_test.py`'s `isolated_env` helper. `gofmt` and
`git diff --check` cover the owned changes. Full repository unit/vet/format,
quality-gate, licensing/module checks, normal-build variants, comparative tasks
and held-out evaluation are coordinator/release checks and were not run by this
section owner.


## Authorized current-account live attempt

The coordinator subsequently authorized only
`TestContactNextLiveCurrentAccountReads`. A dedicated normal CLI was built from
committed candidate `312af93804a44fe38cbaf1fb7ffea36a24b60357` in a temporary
source snapshot, using the checksum-verified frozen catalog. The working tree and
root binary were not used for the build. Current account configuration remained
in place; legacy `TEST_*` credential fallbacks were removed from child processes.
The inspected harness performs no login or content mutations; its existing
bounded retry policy applies only when a read returns a typed retryable error.

Preflight `auth status --json --verify` confirmed user identity. A separate own
`GET /open-apis/authen/v1/user_info --as user` returned the same open ID and a
nonempty tenant key. Those values were compared and supplied to the exact test
only in process memory/environment, never committed or printed. The exact
selection was `go test -json -count=1 -timeout=4m ./tests/cli_e2e/contact -run
^TestContactNextLiveCurrentAccountReads$`, with the dedicated binary selected.

The test failed at `contact_next_live_test.go:69`, the final rare-keyword
`+search-bot` read. Earlier sequential assertions completed: self lookup,
exact-ID get-user round-trip, ID-restricted user-search shape and returned
identities, and self personal-profile array/returned-identity checks. Those
assertions do not establish full Contact live coverage or completeness of every
profile field. The generic failure deliberately omits upstream response content;
the initial failure did not classify its cause. A subsequently authorized single
read diagnostic is recorded below; no login or scope change was performed.

The selected test recorded zero passes, one failure and zero skips. Full Contact
live acceptance, bot-search success and bot-identity tenant binding remain open.
[Sanitized live evidence](../../evidence/sections/contact-live-evidence.json) retains only build/test
hashes, revision, status, binding booleans and the code failure location. Raw
auth/profile data, identifiers, tokens and test output were kept in memory only
and discarded; the temporary build directory was removed. No content was
created, changed, sent or shared.


The coordinator then authorized one bounded diagnostic of the same final
`+search-bot` command on the same committed candidate and current account. It
returned exit 3 with typed `authorization/missing_scope` and
`missing_scopes:["search:bot"]`; no numeric error code was present. This
establishes a missing-scope blocker for the selected user identity, not a
demonstrated bot-search response/projection defect. The original complete live
test still counts as failed. The separately recorded diagnostic also failed,
and neither is converted into a pass. It issued no login, scope request or
content mutation; its raw output was discarded after extracting only the typed
error fields. At that checkpoint bot-search success was unverified because the selected
credentials lacked the scope. The authorized resumption below closes that
particular blocker.


## Authorized search:bot resumption

On 2026-09-08 the user authorized the missing `search:bot` permission and resumed
the selected private live reads. Canonical `make build` ran in a temporary source
snapshot of `d6ec668174030586e5b249fef867a7c6d90f8111`, using the verified frozen
catalog. A fresh verified auth status and independent own-user API read bound
the account and tenant before any selected workflow ran. The initial bot-search
preflight still returned `authorization/missing_scope`; that failed read remains
recorded separately.

The existing device flow requested only `search:bot` plus its automatic
`offline_access` refresh scope. Normal CLI login completion can replace the
profile's user and stored tokens, so a temporary helper reused the existing
device-flow and token-storage functions with checks before persistence. It
verified the returned open ID and tenant against the original account, required
every prior scope to remain granted, checked for concurrent config/token changes,
and stored only the same account's token. Browser consent was coordinated by the
root agent. Authorization succeeded, previous grants were retained, the profile
hash stayed unchanged, and no tokens were deleted. No consent URL, device code,
identity value or credential is retained in committed evidence.

The complete `TestContactNextLiveCurrentAccountReads` selection then passed:
one pass, zero failures and zero skips. This includes current-account lookup,
exact-ID round-trip, ID-restricted search (which returned a match), personal
profile response checks, and the bounded rare-keyword bot search. The two earlier
failed attempts retain their original status in
[the live evidence](../../evidence/sections/contact-live-evidence.json). No other live test was selected.

A separate bot-info read matched the independently verified bot identity. Bot
**tenant** binding remains blocked: its read returned exit 3,
`authorization/app_scope_not_applied`, code `99991672`, requiring
`tenant:tenant:readonly`. That additional scope was not requested. The selected
workflow pass does not establish that separate binding or completeness of every
profile field. No content was created, changed, delivered or shared. Raw response
and test output stayed in memory; the temporary source, binary and authorization
helper were removed. Private-resource verification and its ledger remain owned
by the coordinator.
