# Contributing

How to make a change to `feishu-cli-next`. [AGENTS.md](../AGENTS.md) holds the
binding rules and check list; this guide explains where code goes and what a
complete change includes. The [docs index](README.md) lists every other guide.

## Set up

Requires Go (version in `go.mod`), `make`, and Python 3. `make test` also needs
Node and Ruby for the workflow and script checks.

```sh
make build          # writes ./feishu-cli-next with the committed API catalog
make unit-test vet fmt-check
```

No Feishu account is needed for any check that CI runs. See
[testing](testing.md) for the account-free and real-account parts.

## Know which files you own

This repository is a fork. Upstream paths never move, so the next upstream sync
stays a content merge.

- **Fork-owned:** `internal/nextattempt`, `internal/nextskill`, `cmd/discover`,
  `cmd/guide`, `cmd/receipts`, files named `*_next.go`, `agent-skills/`,
  `docs/`, `baseline/`, `evidence/`, `evaluation/`, `tests/next_*`,
  `scripts/next_*`.
- **Upstream-owned:** everything else. `baseline/v1.0.97/sources.json` pins the
  upstream bytes of most of it; [upstream-footprint.md](upstream-footprint.md)
  lists the pinned files the fork has already edited.

When fork behavior must hook into an upstream file, put the logic in a
fork-owned file (`<name>_next.go` beside `<name>.go`, or a `next*` package) and
keep the upstream edit to one call. Then run
`python3 scripts/next_layout.py footprint`.

## Choose the surface

| Need | Implement in | Rule |
|------|--------------|------|
| Workflow, composition, or smart defaults | `shortcuts/<domain>/` via `common.Shortcut` | Must add value beyond exposing one endpoint. |
| One-to-one OpenAPI method | `internal/registry/catalog/` + generic `cmd/service/` | Upstream-owned catalog; ask first. Never add a shortcut only to expose a method. |
| Arbitrary OpenAPI endpoint | Generic `cmd/api/` | Keep it endpoint-agnostic. |
| Auth, config, profile, update, lifecycle | `cmd/<area>/` + owning `internal/` package | Keep Cobra code as wiring. |
| Event key or payload | `events/<domain>/` | Shared mechanics stay in `internal/event/`. |
| Shared result shape (`--result-fields`, `--save-result`) | `internal/output` | Read the [output contract](contracts/output.md). |
| Mutation receipts and outcomes | `internal/nextattempt`, `cmd/receipts/` | Read the [receipt contract](contracts/receipts.md). |
| Command discovery | `cmd/discover/` | Read the [discovery contract](contracts/discovery.md). |
| Entry skill | `agent-skills/feishu-cli-next/`, `internal/nextskill`, `cmd/guide/` | Read the [skill contract](contracts/skill.md). Keep it short. |
| Cross-command mechanism | Owning `internal/<area>/` package | No generic utils package. |
| Plugin or host integration | `extension/` | Exported symbols are compatibility commitments. |
| When to use a command | `affordance/<domain>.md` | See the [affordance guide](../affordance/README.md). |
| Domain routing and how-to for agents | `skills/<name>/SKILL.md` and `references/` | `SKILL.md` routes; `references/` holds conditional detail. |

Go metadata owns *what* a command is; affordance owns *when* to use it;
`SKILL.md` owns domain routing; `references/` owns *how*. Do not restate one
layer in another.

## Add a shortcut

1. Write `shortcuts/<domain>/<name>.go` declaring a `common.Shortcut`: `Service`,
   `Command` (with a `+` prefix), `Description`, `Risk` (`read`, `write`, or
   `high-risk-write`), `Scopes` or per-identity scopes, `AuthTypes`, `Flags`,
   `DryRun`, and `Execute`. `--dry-run` is injected for you.
2. Call the API through `*common.RuntimeContext` and touch user files through
   `runtime.FileIO()`, `runtime.ValidatePath()`, and `runtime.ResolveSavePath()`.
   Return typed `errs.*` failures (see the [error contract](../errs/ERROR_CONTRACT.md)).
3. Append it to `Shortcuts()` in `shortcuts/<domain>/shortcuts.go`. That file is
   upstream-owned: add one line and run `python3 scripts/next_layout.py footprint`.
4. Add a unit test beside it. Tests that need a factory use
   `cmdutil.TestFactory(t, config)` and isolate config with
   `t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())`.
5. Add an isolated process test in `tests/next_sections/<section>/` using
   `tests/nextfixture`, which builds a fixture binary and serves a mock API.
   Assert the request and the compact result, including failure paths. These
   tests run only under `scripts/next_test.py`, which supplies the isolated HOME.
6. Add usage guidance under `## +<name>` in `affordance/<domain>.md`, and a
   source test in `internal/affordance/<domain>_next_source_test.go`.
7. If agents need routing or how-to beyond the affordance entry, update
   `skills/lark-<domain>/`. Every path a shipped doc references must be embedded
   (`content_embed.go`).
8. Run the capability gate. A new command is an interface addition: record it in
   `baseline/v1.0.97/next-interface-mapping.json.gz` only after asking first;
   the procedure is in [baseline/CI.md](../baseline/CI.md).
9. For a write, decide how receipts classify its response. Read the
   [receipt contract](contracts/receipts.md) before touching `internal/nextattempt`.

## Add a domain

Everything in "Add a shortcut", plus:

- Create `shortcuts/<domain>/shortcuts.go` with `Shortcuts()` and add one line to
  `init()` in `shortcuts/register.go`.
- The section runner discovers `shortcuts/<domain>` and `tests/next_sections/<domain>`
  automatically. If the section name differs from the package directory, add it
  to `DOMAIN_DIRS` in `scripts/next_test.py`.
- Create `affordance/<domain>.md` and, if needed, `skills/lark-<domain>/`.

## Test

Run the narrowest check while iterating, then the checks AGENTS.md lists for
your change. In short:

| Change | Checks |
|--------|--------|
| Any Go package | `go test ./path/...`, then `make unit-test vet fmt-check` |
| A domain | `python3 scripts/next_test.py --section <name> --suite all --binary ./feishu-cli-next --report .tmp/next-tests/<name>.json` |
| Commands, flags, schemas, risk, brand guards | The capability gate (command in AGENTS.md) |
| Help, schema, or `skills/` | Commit, then `make quality-gate` |
| `affordance/` | `go test ./internal/affordance ./cmd/service ./internal/schema` |
| `scripts/`, workflows, `docs/` layout | `make script-test` |
| `evaluation/` | `make evaluation-test` |
| Everything | `make test` |

Every behavior change needs a test that fails when the change is reverted.
Assert fields, requests, typed errors, or side effects, not message text alone.
Tests must not depend on developer profiles, keychains, home directories, or
real credentials. `next_test.py` exit 2 means blocked, not passed.

Real-account tests run only on your machine; read
[testing](testing.md) and its authorization boundaries before running one.

## Open a pull request

Use English Conventional Commits and PR titles, one goal per PR. Fill in the PR
template and add which relevant checks you did not run; the inherited template
has no field for it. Never commit
credentials, tokens, internal endpoints, or real tenant data.
