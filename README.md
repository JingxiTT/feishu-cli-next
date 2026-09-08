# Feishu CLI Next

A Feishu/Lark CLI for AI agents that work through a shell and files. It builds on the official Lark CLI and adds command discovery, compact results, file artifacts, typed data transfer, and mutation recovery evidence.

The goal is to reduce the tokens needed to complete a task while preserving every original command and workflow, data fidelity, and successful outcomes. Results remain task-dependent; the evaluation tracks the **50% token-reduction target separately for each function**, with failed-attempt costs included.

[Quick start](#quick-start) · [Usage](#usage) · [Evaluation report](evaluation/evidence/token-efficiency-04/REPORT.md) · [Implementation status](docs/IMPLEMENTATION.md)

## What this project adds

| Capability | What it provides |
| --- | --- |
| Command discovery | Search registered commands with bounded results; optional flag descriptions expose accepted inputs without a separate help lookup. |
| Compact JSON | Full JSON data in compact form by default; optional indentation and field selection. |
| File artifacts | Save supported read results to versioned, identity-bound files for local processing and handoff. |
| Typed data workflows | Sheets export/import and Base record workflows with validation, type preservation, and operation-specific no-op detection. |
| Mutation evidence | Durable local receipts distinguish rejected, partial, and unknown outcomes without replaying writes. The exact raw Contact basic-profile POST read retains read-only behavior. |
| One entry skill | Install direct routes for common tasks, including native XML edits, single-row Base updates, exact Sheets exports and Drive transfers; load deeper guidance on demand. |
| Upstream compatibility | Retain domain shortcuts, generated service commands, raw API access, authentication, events, and the extension SDK. |

The inherited surface includes Docs, Sheets, Base, Drive, Wiki, Contacts, Messages, Calendar, Tasks, Mail, meetings, and other registered services. See the [capability inventory](docs/reference/README.md) and [section evidence](docs/sections/OWNERSHIP.md) for scope and verification status.

**Status:** development distribution. Full live parity and release acceptance remain open. Source builds and local installation are available; a public release/update channel is not configured. Windows execution is deferred. See [distribution details](docs/distribution.md).

## Quick start

Requires **Go 1.23+**, **Python 3**, and `make`. The first build may download Go dependencies. The frozen API catalog ships with the source and is verified/restored during the build.

```sh
git clone https://github.com/Zhujingxi/feishu-cli-next.git
cd feishu-cli-next
make build
./feishu-cli-next --version
```

Check an existing account configuration:

```sh
./feishu-cli-next auth status
```

For first-time setup, configure your Feishu/Lark app and authorize the user account:

```sh
./feishu-cli-next config init --new
./feishu-cli-next auth login --recommend
```

Authentication reuses upstream profiles, credential providers, brand selection, and `LARKSUITE_CLI_*` settings. User and bot identities have different permissions. See [authentication and setup](README.upstream.md#authentication) for scope and account options.

Install the entry skill into an existing agent project:

```sh
./feishu-cli-next guide install --project /path/to/your/project
```

This writes the CLI-owned skill under the project's `.agents/skills/` directory. `guide update` and `guide uninstall` manage that installation. To put the executable on your path, use `make install PREFIX="$HOME/.local"` and include `$HOME/.local/bin` in `PATH`; use the same prefix for `make update-install` or `make uninstall`.

## Usage

Use a known command directly. Help, discovery, and schema inspection are available when needed:

```sh
# Inspect one domain or search for a command.
./feishu-cli-next contact --help
./feishu-cli-next discover 'contact open_id' --flags --limit 3
./feishu-cli-next schema drive files list

# Search for a user or a visible bot using user identity.
./feishu-cli-next contact +search-user --query 'Alex' --as user
./feishu-cli-next contact +search-bot --query 'assistant' --as user

# Read a document.
./feishu-cli-next docs +fetch --doc DOCUMENT_TOKEN --as user

# Export an explicit Sheets range to a local artifact.
./feishu-cli-next sheets +data-export \
  --spreadsheet-token SPREADSHEET_TOKEN --sheet-id SHEET_ID \
  --range A1:E100 --save-result cells.json --as user
```

Replace resource placeholders with your own IDs. Discovery describes the local command surface; access still depends on the selected account and scopes. Shortcuts start with `+`; generated service commands expose catalog methods, and `api` provides raw OpenAPI access.

### Results and recovery

- `--result-detail` indents JSON. `--result-fields id,name` selects supported result fields while retaining outcome and completeness metadata. Domain flags such as `--fields` and `--detail` keep their original meanings.
- `--save-result PATH` saves the full result for supported reads without overwriting an existing file. These artifacts have domain-specific contracts; a saved envelope is not a raw API request body. See the [output contract](docs/contracts/output.md).
- Check both the JSON outcome and process exit status. A timeout after a write may leave its effect unknown. `receipts list` and `receipts inspect ID` expose local attempt evidence for reconciliation. See the [receipt contract](docs/contracts/receipts.md) before retrying an uncertain mutation.

### Why the old skills remain

[`agent-skills/feishu-cli-next/SKILL.md`](agent-skills/feishu-cli-next/SKILL.md) is the entry skill installed by `guide install`. The inherited [`skills/`](skills) directory supplies compatibility documentation and detailed references used by help and `skills read`.

Those guides are not all injected by installing the entry skill. They can still influence an agent and consume context when it reads them on demand. The evaluation includes that work; Next did call `skills read` in some trials. Upstream examples use `lark-cli`; use `feishu-cli-next` with this distribution.

## Latest evaluation

**[Read the latest report →](evaluation/evidence/token-efficiency-04/REPORT.md)**

Condition 04 is an **exploratory follow-up: 24 new candidate trials compared with 48 reused baseline trials from condition 03**. It is not a new balanced 72-trial experiment. The model is **GPT-5.6 Luna with medium reasoning**, run through native **Codex CLI 0.153.4**, on eight tasks with three fresh candidate contexts each. Feishu operations use signed synthetic fixtures informed by native behavior; this is not a live-tenant benchmark.

**24/24 new attempts verified · 1,154,003 new provider tokens · the 50% target is not fully met: 5/8 functions met it.** All eight functions used fewer tokens per verified success than the reused unchanged-Next baseline. Each candidate function verified 3/3, matching or exceeding both baselines' observed success counts.

| Function | Token saving vs unchanged Next | 50% target |
| --- | ---: | --- |
| Discovery | 52.5% | Met |
| Contact search | 68.9% | Met |
| Docs read | 39.5% | Missed |
| Docs edit | 63.4% | Met |
| Markdown creation | 43.1% | Missed |
| Base update | 74.1% | Met |
| Sheets export | 55.7% | Met |
| Drive roundtrip | 40.0% | Missed |

These percentages use the reused unchanged-Next baseline. Against original/full, Docs read used 5.8% more tokens; original Docs-edit accounting is incomplete and original Markdown/Sheets have zero verified successes, making those token comparisons unavailable. The full report retains these distinctions.

The primary target compares all-attempt tokens per verified success against unchanged Next; original/full is a separate compatibility and token comparator. Cached input is counted once, failed-attempt costs stay included, and unknown usage prevents an affected finite score. Baselines were reused to avoid another 48 model calls, so timing/cache differences and adaptive guidance tuning limit the comparison. Three repetitions do not establish statistical success-rate parity or stable 50% savings.

[Frozen environment](evaluation/evidence/token-efficiency-04/environment.json) · [Readiness review](evaluation/evidence/token-efficiency-04/readiness-review.json) · [Follow-up protocol](evaluation/token-efficiency-04.md) · [All evaluation conditions](evaluation/README.md)

Earlier reports remain unchanged: [original pilot 01](evaluation/evidence/paired-luna-medium-01/REPORT.md), [balanced optimization 02](evaluation/evidence/token-efficiency-02/REPORT.md), and [balanced optimization 03](evaluation/evidence/token-efficiency-03/REPORT.md). Their costs, failures, inconclusive outcomes and accounting gaps remain recorded separately.

## Development

```sh
make build
make evaluation-test   # Isolated harness tests; no model trials.
make next-script-test  # Isolated tooling tests.
make unit-test
make vet
make fmt-check
```

`make test` runs the broader suite. Section and integration runners record pass/fail/skip evidence while isolating developer configuration. Live tests require their documented fixture prerequisites. See [test isolation](tests/next_foundation/README.md), [evaluation tooling](evaluation/README.md), and [contributor instructions](AGENTS.md).

Project documentation:

- [Implementation ledger](docs/IMPLEMENTATION.md) — completed work, evidence, and remaining gates.
- [Design plan](docs/PLAN.md) — goals, scope, and acceptance criteria.
- [Distribution](docs/distribution.md) — local installation, packaging, and release status.
- [Contracts](docs/contracts) — output, artifacts, receipts, and skill lifecycle.

## Upstream and license

This is a separate development distribution derived from `larksuite/cli` at commit `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, not an official Lark release. The Go module remains `github.com/larksuite/cli` for source and extension compatibility. Original copyright notices, the [MIT license](LICENSE), and [upstream documentation](README.upstream.md) are retained.
