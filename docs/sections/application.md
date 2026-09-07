# Application section

Status: offline implementation and full section suite pass; live backend checks
and comparative development evaluation remain open. This is not release acceptance.

## Complete reference mapping

Inspected frozen sibling `../feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, its entire
`shortcuts/application/` implementation and tests, `affordance/application.md`,
`events/application/`, and `docs/reference/{coverage.json,registrations.json.gz,sources.json}`.
The section has exactly five command inventory entries: the `application` domain
and four shortcuts. Both frozen Feishu/Lark registration variants reuse these
shortcuts. The raw v7 slash-command endpoints are deliberately absent from the
frozen generated catalog; the existing workflow implementations are retained.
`shortcuts/apps/` and its separate app/automation/database capabilities belong to
the Apps section. Application bot-menu event projection is owned by Events.

| Capability | Complete contract retained | Evidence owner |
| --- | --- | --- |
| `application` | Public domain discovery and profile selection | Shared discovery; process help assertions |
| `+slash-command-list` | GET current bound app; both user and bot; read scope; all returned item fields, nested description/i18n, top-level icon, IDs and timestamps; empty list/count; JSON and pretty output; dry-run | Existing list units/dry-run plus process lifecycle, identity, projection and artifact tests |
| `+slash-command-create` | Required command and description; trim command only; repeated `<language>=<text>` split once; Unicode/equals preservation; optional icon; optional boolean force; write scope and conditional read scope for collision recovery; exact known collision only; original failure cause/code/log ID | Existing create/common units, replacement units and process lifecycle, force and fault tests |
| `+slash-command-update` | Exactly one ID/name; description, repeated i18n, icon; i18n requires default description; whole description/i18n replacement with omitted top-level fields preserved; stable command name/ID; escaped path segment; direct ID needs only write scope, name resolution needs read | Existing update units/dry-run, replacement units and independent process state checks |
| `+slash-command-delete` | Exactly one ID/name; irreversible high-risk confirmation; both identities; write and conditional read scopes; escaped ID; current app targeting; new identity after recreation; cache/identity warnings on stderr | Existing delete units and all five inherited process tests; new process lifecycle/confirmation check |
| `application.bot.menu_v6` (Events dependency) | Event ID, header time/app/tenant, event key, menu time, operator open/union/user IDs/name; bot identity; malformed payload dropping | Inspected existing Events-owned implementation; seven existing event tests included by section runner |

Every shortcut retains shared `--as`, `--profile`, `--dry-run`, `--format`,
`--json`, and `--jq`; deletion retains `--yes`. No flags, identities, scopes,
content fields, or resource options were removed. Root output controls and receipt
mechanics remain owned by the accepted P1 foundation.

## Changes and intentional differences

- List responses retain upstream metadata and continuation fields in addition to
  the existing items/count output. Unknown item fields remain lossless. Typed
  projection validates the actual fields used for display and target selection;
  malformed list/item shapes return `api/invalid_response` instead of silently
  becoming an empty list or targeting a malformed item.
- Name resolution requires an exact unique match with a command ID and rejects
  a response marked `has_more:true`. No arbitrary first match is mutated. The
  endpoint is documented as unpaginated, with a maximum of 100 commands; if that
  backend contract changes, pagination support must be verified before by-name
  writes can use incomplete results. Reads retain the continuation for inspection.
- By-name updates compare the entire supplied replacement objects with the live
  state already fetched for resolution and return `action:unchanged` without
  PATCH when equal. The force collision path does the same after its existing
  POST/list steps. Description replacement includes the entire i18n map; an
  omitted existing language or an unknown existing replacement field prevents a
  false no-op. Omitted top-level fields never imply deletion.
- Direct-ID updates keep their original write-only permission contract and issue
  PATCH without an additional read. No local lock or comparison claims remote
  concurrency protection. The read/compare/write sequence has no backend ETag or
  conditional-write guarantee.
- Receipt classification is intentionally conservative: a code 40000000 collision
  remains unknown in the shared journal. A force collision followed by PATCH or
  no-op therefore exits 22 and preserves the full `updated`/`unchanged` result
  with `ok:false`; the later step never erases earlier evidence. Unknown responses
  are not retried. This limitation needs domain receipt interpretation before a
  collision rerun can report fully reconciled success; no parallel journal was added.
- Artifacts use the shared versioned result container and current identity/app
  binding. Full localized content and continuation survive projection. Existing
  destinations and known credential-bearing fields are rejected; mutation commands
  reject `--save-result` before dispatch. There is no Application artifact import
  command, and result containers are not accepted as replacement payloads.

## Section validation

Executed with synthetic transports or the real authsidecar CLI against local
`nextfixture` backends. All environments use `scripts/next_test.py` or its
`isolated_env` helper; no inherited personal credentials or live traffic.

- `python3 scripts/next_test.py --section application --suite all --report .tmp/application-all.json`
  passes: 57 unit/event test events, four inherited selected dry-run tests, and
  18 next-process test events. One separately named live scenario is explicitly
  skipped, not passed. The process suite covers lifecycle effects, whole-object
  replacement, no-op call counts, both identities, public help/flags, dry-run
  paths, validation before dispatch, duplicate/incomplete/missing targets, force
  recovery, missing scope, lost/malformed response, partial-item outcomes,
  no automatic replay, receipts, projection and credential-safe artifacts.
- Isolated `go test -json -count=1 ./tests/cli_e2e/application` runs all five
  inherited tests, including the confirmation test omitted by the standard
  `DryRun|Regression` selector. Evidence: `.tmp/application-reference-all.json`.
- Isolated `go test -json ./internal/affordance ./cmd/service ./internal/schema`
  passes 275 test events, zero skips. Evidence: `.tmp/application-guidance.json`.

- Isolated `go vet ./shortcuts/application ./tests/next_sections/application`
  passes. Evidence: `.tmp/application-vet.json`. Owned Go files are gofmt-clean,
  and `git diff --check` passes for the section.

Normal build, full repository unit/vet/lint/license gates and comparative
measurements are coordinated across sections; these scoped results do not claim
those release gates. No held-out tasks or live resources were used.

## Open acceptance work

`TestApplicationLivePrivateLifecycle` is a separately selectable blocked live
scenario. Slash commands affect the bound app and appear in clients of that app's
users. Current-account-only installation/visibility and safe private app creation
cannot be proven from the offline fixture. No organization/app write, installation,
permission change, or unrelated resource edit was attempted. Before executing a
live lifecycle, verify effective account and tenant, app visibility restricted to
that account (or unambiguously authorized Ying Wu), create a fresh localized
command, independently list/compare it, run unchanged and changed updates, delete
only that command, and verify server removal and client cache behavior. Privacy
must be verified before creating content. Cache timing and actual backend
replacement/permission guarantees remain unverified.

The coordinator schedules balanced comparative development tasks after the
section handoff. No efficiency claims are made. The section remains open pending
required live evidence, collision-receipt reconciliation, and comparative review.
