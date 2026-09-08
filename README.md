# Feishu CLI Next

A Feishu/Lark CLI for AI agents that work through a shell and files. It builds on the official Lark CLI and adds command discovery, compact results, file artifacts, typed data transfer, and mutation recovery evidence.

The goal is to reduce the tokens needed to complete a task while preserving its data and outcome. **Results are task-dependent:** the latest evaluation found substantial savings for Drive, higher usage for several lookup/read tasks, and gaps that prevent conclusions for Markdown.

[Quick start](#quick-start) · [Usage](#usage) · [Evaluation report](evaluation/evidence/paired-luna-medium-01/REPORT.md) · [Implementation status](docs/IMPLEMENTATION.md)

## What this project adds

| Capability | What it provides |
| --- | --- |
| Command discovery | Search the executable's registered commands with bounded results and optional flag detail. |
| Compact JSON | Full JSON data in compact form by default; optional indentation and field selection. |
| File artifacts | Save supported read results to versioned, identity-bound files for local processing and handoff. |
| Typed data workflows | Sheets export/import and Base record workflows with validation, type preservation, and operation-specific no-op detection. |
| Mutation evidence | Durable local receipts distinguish rejected, partial, and unknown outcomes without automatically replaying writes. |
| One entry skill | Install a concise English skill into a project; load detailed domain guidance on demand. |
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
./feishu-cli-next discover 'contact open_id' --limit 8
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

**[Read the complete report →](evaluation/evidence/paired-luna-medium-01/REPORT.md)**

The pilot ran **72 real Codex CLI trials** with **GPT-5.6 Luna, medium reasoning**: eight tasks × three arms × three repetitions. The arms were official CLI/full guidance, the same CLI/concise guidance, and Next/its entry skill. Feishu operations used isolated synthetic fixtures informed by native behavior; this was not a live-tenant benchmark.

**54 verified successes · 11 failed verifications · 7 inconclusive outcomes · 8,442,585 tokens.** Every attempt has complete provider accounting. Cached input is included once, and failed-attempt costs are retained.

| Function | Observed result |
| --- | --- |
| Drive roundtrip | Next used 51.1% fewer tokens than concise guidance, with both at 3/3 successes; 58.7% fewer than official/full on two matched successful trials. |
| Discovery | Next used 33.0% more tokens than official/full; both passed 3/3. |
| Contact search | Next used 47.9% more tokens than official/full; both passed 3/3. |
| Docs read | Next used 91.5% more tokens than official/full; both passed 3/3. |
| Docs edit | Next used 2.9% fewer tokens than official/full; both passed 3/3. |
| Base update | Next used 2.9% more tokens than official/full; both passed 3/3. |
| Sheets export | Next verified 3/3; neither baseline produced a verified export, with one inconclusive baseline attempt. No comparative saving percentage. |
| Markdown creation | No verified outcomes; six inconclusive attempts. Broader fixture coverage is needed before comparing efficiency. |

Three repetitions do not establish statistical significance or overall product superiority. The full report includes distributions, matched comparisons, fixture limitations, and observed discovery overhead.

[Raw trials and summary](evaluation/evidence/paired-luna-medium-01) · [Frozen environment](evaluation/evidence/paired-luna-medium-01/environment.json) · [Independent audit](evaluation/evidence/paired-luna-medium-01/result-review.json) · [Evaluation protocol](evaluation/paired-functions-luna-medium.md) · [Historical evaluations](evaluation/README.md)

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
