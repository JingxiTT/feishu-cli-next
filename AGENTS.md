# AGENTS.md

`feishu-cli-next` is a fork of the official Lark CLI (`larksuite/cli`, synced to
v1.0.97) for AI agents that work through a shell and files. It keeps every
inherited command and reduces the tokens an agent spends on the whole task. A
shorter response that forces an extra call, or hides a fact the agent needs, is
a regression.

The module path `github.com/larksuite/cli`, the `LARKSUITE_CLI_*` variables, and
`lark-cli` in the inherited `skills/` docs are deliberate; do not rename them.

[docs/contributing.md](docs/contributing.md) explains how to add a shortcut or
domain and how each layer is tested; [docs/README.md](docs/README.md) indexes
every guide and contract.

## Commands

```bash
make build                      # writes ./feishu-cli-next
go test ./path/to/package/...   # while iterating
make unit-test vet fmt-check    # before finishing any Go change
make test                       # broad change; also needs Python 3, Node, Ruby
```

After touching any command, flag, shortcut, schema, risk level, or brand guard,
run the capability gate. It needs no credentials:

```bash
python3 scripts/next_inventory.py gate --target . \
  --reference-dir baseline/v1.0.97 \
  --mapping baseline/v1.0.97/next-interface-mapping.json.gz \
  --out .tmp/capability-gate.json
```

| Change | Also run |
|--------|----------|
| One domain, or shared output/receipts/discovery (`foundation`) | `python3 scripts/next_test.py --section <name> --suite all --binary ./feishu-cli-next --report .tmp/next-tests/<name>.json` |
| Command, help, or schema surface; `skills/` | `make quality-gate` (compares the base with `HEAD`, so commit first) |
| `affordance/` | `go test ./internal/affordance ./cmd/service ./internal/schema` |
| `scripts/`, workflows, an upstream file not yet in `docs/upstream-footprint.md`, or a file added, moved, or removed under `docs/`, `evidence/`, or the root | `make script-test` |
| `evaluation/` | `make evaluation-test` |
| `extension/` | `make examples-build`, `go test ./tests/plugin_e2e/...` |

`next_test.py` exit 2 (`blocked`) is not a pass. CI runs on Linux, offline and
dry-run only. In the PR, state which relevant checks you did not run.

## Boundaries

**Always**

- Keep every inherited command, shortcut, flag, event key, and brand guard
  working with its original contract. Add an opt-in control instead of changing
  an inherited default.
- Keep compact output lossless: errors, missing scopes, truncation, pagination,
  partial results, and unknown mutation outcomes stay visible.
- Add a test that fails if the change is reverted.
- Label evidence by what it measured. Offline is not live, and a skipped or
  blocked test is not a pass.

**Ask first**

- Breaking or renaming an inherited contract.
- Running anything against a real tenant.
- Updating `next-interface-mapping.json.gz`; see
  [baseline/CI.md](baseline/CI.md).
- Editing upstream-owned files outside a sync: `internal/registry/catalog/`,
  `CHANGELOG.md`, `README.upstream.md`.
- Adding a dependency.

**Never**

- Remove or narrow a capability to save tokens or simplify testing.
- Weaken a test, lint rule, CI check, allowlist, or the capability mapping to
  get a pass.
- Replay a mutation whose outcome is unknown, or report a partial read as
  complete.
- Edit, rename, or delete retained evidence (`evidence/`, frozen `baseline/`
  snapshots) or use `evaluation/tasks.held-out.json`. Add a record and run
  `python3 scripts/next_layout.py seal`; see
  [evidence/README.md](evidence/README.md).
- Claim an efficiency or parity result that no recorded run supports.
- Commit credentials, tokens, internal endpoints, or real tenant data.

## Live testing

Offline is the default; `scripts/next_test.py` runs with an isolated HOME and no
developer credentials. Before any command that can reach a real tenant, read
[Authorization boundaries for all testing](docs/PLAN.md#authorization-boundaries-for-all-testing).
Use only private resources the run created, and clean them up. These limits
apply to testing; never encode them in the product or a shipped skill.

## Layout

Paths that upstream ships never move, so a sync stays a content merge. The fork
adds a bounded layer, and each of its directories has one job:

| Path | Holds |
|------|-------|
| `docs/` | Markdown to read: contracts, section guides, plan, status |
| `baseline/` | What the capability gate compares against; replaced only by a sync |
| `evidence/` | Sealed records of past runs and reviews; add-only |
| `evaluation/` | The comparison harness, task sets, fixtures, and `reports/` |
| `agent-skills/` | The entry skill |
| `tests/next_*`, `scripts/next_*` | Fork suites and tooling beside upstream's |

Fork code inside upstream's tree goes, in order of preference, in:

1. A package the fork owns: `internal/next<area>/`, `cmd/<name>/`, or a
   sub-package under the domain.
2. A new file in the upstream package, with `_next` in its name. Declarations
   that support an upstream file `<name>.go` live in `<name>_next.go` (tests in
   `<name>_next_test.go`); `scripts/next_layout.py check` rejects an unmarked one.
3. An edit to the upstream file, kept to a call into 1 or 2. Never re-indent,
   rename, or relocate upstream lines to wrap them; for the config lock use
   `core.LockConfig()` with `defer release(&err)`.

[docs/upstream-footprint.md](docs/upstream-footprint.md) lists every pinned
upstream file the fork edits, then the upstream paths with no pinned hash
(`skills/`, `affordance/`, `lint/`, ...), whose edits it cannot see.
`scripts/next_layout.py check` enforces that list, the
root allowlist, markdown-only `docs/`, and the evidence seal; run
`python3 scripts/next_layout.py footprint` after editing or restoring an
upstream file.

## Where changes go

| Change | Goes in |
|--------|---------|
| Workflow, composition, or smart defaults | `shortcuts/<domain>/` via `common.Shortcut`; must add value beyond exposing one endpoint |
| One-to-one OpenAPI method | Upstream catalog + `cmd/service/`; do not add a shortcut just to expose it |
| Shared result shape (`--result-fields`, `--result-detail`, `--save-result`) | `internal/output` |
| Mutation receipts and outcomes | `internal/nextattempt`, `cmd/receipts/` |
| Command discovery | `cmd/discover/` |
| Entry skill and its install lifecycle | `agent-skills/feishu-cli-next/`, `internal/nextskill`, `cmd/guide/` |
| Event keys and payloads | `events/<domain>/` |
| Cross-command mechanism | The owning `internal/<area>/` package, not a utils package |
| Plugin or host integration | `extension/`; exported symbols are compatibility commitments |
| Per-command guidance | `affordance/<domain>.md` |
| Domain guidance for agents | `skills/<name>/SKILL.md` and `references/` |
| Result of a run, review, or reconciliation | A new sealed file under `evidence/`; its write-up under `evaluation/reports/` or `docs/sections/` |

## Contracts

Read the owning contract before changing behavior it covers:

- Output and artifacts: [docs/contracts/output.md](docs/contracts/output.md)
- Mutation receipts: [docs/contracts/receipts.md](docs/contracts/receipts.md)
- Errors: [errs/ERROR_CONTRACT.md](errs/ERROR_CONTRACT.md)
- Discovery, entry skill, recovery text: [docs/contracts/](docs/contracts/)
- Command guidance: [affordance/README.md](affordance/README.md)
- Source guards and waivers: [lint/README.md](lint/README.md)

Rules the linters do not catch:

- Success data goes to stdout; typed errors, progress, warnings, and hints go to
  stderr.
- Shortcuts call the API through `*common.RuntimeContext` and touch user files
  through `runtime.FileIO()`, `runtime.ValidatePath()`, and
  `runtime.ResolveSavePath()`.
- Command failures return typed `errs.*`, never a final `fmt.Errorf` or
  `errors.New`.
- Keep the entry skill short: no routine checklists, exhaustive command lists,
  or development process.
- Every path a shipped doc references must be embedded; see `content_embed.go`.

## Commits and PRs

Use English Conventional Commits and PR titles. Keep each PR to one goal.
