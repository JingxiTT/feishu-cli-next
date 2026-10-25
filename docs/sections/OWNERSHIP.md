# Capability section assignments

The testing authorization boundaries are in
[testing](../testing.md#authorization-boundaries-for-all-testing). This is
implementation coordination guidance; it is not part of the shipped agent skill.

P2 starts after the coordinator records the P1 shared-interface gate. Every
section inspects its frozen registrations in `baseline/coverage.json`,
catalog, source hashes, and the unchanged sibling reference. Reuse the existing
implementation. Do not replace complete domain commands with raw forwarding or
count registration alone as behavioral verification.

## Ownership

The coordinator owns root command assembly, shared contracts, inventory,
cross-section workflows, evaluation scheduling, entry-skill assembly, and release
acceptance. Shared contract changes require coordination before editing.

Each domain owner owns `shortcuts/<domain>/` (`docs` maps to `doc`), its existing
`tests/cli_e2e/<section>/` directory, a new `tests/next_sections/<section>/`
process suite, and, where one is kept, `docs/sections/<section>.md`. Domain affordance guidance is
owned by the matching section. The events owner owns all `events/` projections,
`shortcuts/event/`, and the event process suite; domain owners inspect their event
contracts and send required changes to that owner. The extensions owner owns
`extension/` and its process tests. Packaging owns distribution metadata,
install/release scripts, build variants and release workflows, coordinating any
shared Makefile or CI edits. Markdown owns its shortcut and conversion contract,
coordinating rich-content changes with Docs.

Agents share one checkout. Stage only owned paths or agreed hunks, preserve
others' changes, and make focused Conventional Commits. Split functions only
when independent ownership is useful. Do not create competing shared runtimes.

## Shared interfaces

- `common.RuntimeContext` remains the domain API, input and FileIO boundary.
  Preserve existing risk, identity, scope, dry-run, schema and flag metadata.
- Discovery reads actual registrations. Default JSON is compact and lossless;
  `--result-fields`, `--result-detail`, and `--save-result` are shared opt-in
  controls. Domain `--fields` and `--detail` retain their original meaning.
  See [output](../contracts/output.md). Generic result artifacts are not import
  payloads; typed consumers must verify schema, complete extent and binding.
- [Receipts](../contracts/receipts.md) journal before mutation and conservatively
  classify effects. Domain code supplies stronger interpretation or read-only
  reconciliation only from actual backend evidence. No automatic replay of an
  unknown mutation. Identity/tenant verification flags report evidence, not an
  inferred guarantee.
- Preserve full supported content and types. For replacement operations, verify
  appropriate unchanged-content no-op behavior in code. Append, send, assignment,
  notification and clear effects require their own explicit semantics. Do not
  infer deletion from omitted data or claim concurrency protection from a lock.
- Shared process fixtures use the real authsidecar build with isolated config
  and a synthetic backend. Domain suites live in separate packages so they can
  be selected independently. Normal build and live evidence remain distinct.

## Section completion

Before choosing changes, record the complete section mapping, test owner,
dependencies and acceptance scenarios. Preserve less common options and types.
Write meaningful units alongside changes and process tests during implementation.
At the complete section boundary, execute all section units and written E2E
tests, plus applicable reference regressions. Report omissions and skips explicitly.
Use `scripts/next_test.py` or its `isolated_env` helper; never run legacy live
helpers with personal credentials. Test fixtures must not forward live requests.

Each section report lists mapped capabilities and evidence, intentional interface
differences, exact commands and outcomes, changed files/commits, and unresolved
backend or authorization assumptions. Required unverified capabilities keep the
section open. Notify the coordinator when the whole section is ready for live
verification and comparative development tasks. Comparative evaluations run one
balanced workload at a time; keep held-out tasks untouched until P4.

Live tests obey the current-account private-fixture and Ying Wu-only other
recipient boundary. Verify identity, tenant and creation visibility before any
content write. No existing unrelated resource may be edited or deleted. When
privacy or recipient identity cannot be verified, write the selectable scenario,
record the live blocker, and continue independent offline work. Do not weaken
product capabilities or count blocked live cases as passes.
