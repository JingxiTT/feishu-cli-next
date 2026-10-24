# Feishu CLI Next

A Feishu/Lark CLI for AI agents that work through a shell and files. It builds on the official Lark CLI and adds command discovery, compact results, file artifacts, typed data transfer, and mutation recovery evidence.

The goal is to reduce the tokens needed to complete a task while preserving every original command and workflow, data fidelity, and successful outcomes. Results remain task-dependent. Evaluations measure this against per-task gates of **more than 50% fewer tokens for Docs read/edit and at least 30% fewer for each other function versus the original CLI**, with failed-attempt costs included; most gates are not yet met. See [Evaluation](#evaluation).

Synced with [official Lark CLI v1.0.97](https://github.com/larksuite/cli/releases/tag/v1.0.97). See the [upgrade comparison and optimization measurements](baseline/v1.0.97/OPTIMIZATION.md). The original frozen reference remains available for historical comparisons.

[中文说明](README.zh.md) · [Quick start](#quick-start) · [Usage](#usage) · [Evaluation](#evaluation) · [Contributing](docs/contributing.md) · [All docs](docs/README.md)

## What this project adds

| Capability | What it provides |
| --- | --- |
| Command discovery | Search registered commands with bounded results; optional flag descriptions expose accepted inputs without a separate help lookup. |
| Compact JSON | Full JSON data in compact form by default; optional indentation and field selection. |
| Docs reads and local edits | Read native text and table relationships, select focused excerpts, or save a compatible working copy for verified text edits across unambiguous formatting boundaries. |
| File artifacts | Save supported read results to versioned, identity-bound files for local processing and handoff. |
| Typed data workflows | Sheets export/import and Base record workflows with validation, type preservation, and operation-specific no-op detection. |
| Mutation evidence | Durable local receipts distinguish rejected, partial, and unknown outcomes without replaying writes. The exact raw Contact basic-profile POST read retains read-only behavior. |
| One entry skill | Install direct routes for common tasks, including local Docs text edits, native XML operations, single-row Base updates, exact Sheets exports and Drive transfers; load deeper guidance on demand. |
| Upstream compatibility | Retain domain shortcuts, generated service commands, raw API access, authentication, events, and the extension SDK. |

The inherited surface includes Docs, Sheets, Base, Drive, Wiki, Contacts, Messages, Calendar, Tasks, Mail, meetings, and other registered services. See the [capability inventory](baseline/README.md) and [section evidence](docs/sections/OWNERSHIP.md) for scope and verification status.

**Status:** development distribution. Full live parity and release acceptance remain open. Source builds and local installation are available; a public release/update channel is not configured. Windows execution is deferred. See [distribution details](docs/distribution.md).

## Quick start

Requires **Go 1.26.10+** and `make`. The first build may download Go dependencies. The official v1.0.97 API catalog ships with the source and is embedded directly. Python 3 is needed for inventory, packaging, evaluation, and prefix installation scripts.

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

# Read document facts as Markdown with native block IDs.
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --as user

# Export an explicit Sheets range to a local artifact.
./feishu-cli-next sheets +data-export \
  --spreadsheet-token SPREADSHEET_TOKEN --sheet-id SHEET_ID \
  --range A1:E100 --save-result cells.json --as user
```

For a generated method's request requirements, use `schema <service.resource.method> --input-only`.
It retains the full input schema, permissions, risk and command guidance while
omitting the response schema. Omit the flag when you need response field definitions.

Replace resource placeholders with your own IDs. Discovery describes the local command surface; access still depends on the selected account and scopes. Shortcuts start with `+`; generated service commands expose catalog methods, and `api` provides raw OpenAPI access.

### Read focused Docs content

Without `--output`, `docs +pull` returns a read-only Markdown view with native anchors, readable lists, tasks, code, quotes, deeper headings and table/cell relationships. Unsupported content stays visibly labeled. Tables retain native cell order and content links; the CLI does not invent coordinates or merged-cell geometry. Contradictory table/list parent-child relationships are rejected before output.

```sh
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --scope keyword --keyword 'budget' --as user
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --scope outline --as user
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --scope section --start-block-id HEADING_ID --as user
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --scope range --start-block-id FIRST_ID --end-block-id LAST_ID --as user
```

Keyword search is literal, case-insensitive and confined to individual native text blocks. Sections end before the next heading of equal or higher level; range bounds are inclusive. Selection runs locally after reading the complete native snapshot. Check `returned_blocks`, `total_blocks` and `complete`; a complete block selection does not make opaque content readable. A missing keyword match does not prove absence from opaque content. Scoped results cannot be saved as editable working copies. Existing server-scoped `docs +fetch` remains available.

### Edit Docs through a local working copy

```sh
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --output note.md --as user
# Edit supported text in note.md; keep its block markers and formatting intact.
./feishu-cli-next docs +push --file note.md --dry-run --as user
./feishu-cli-next docs +push --file note.md --as user
```

Pull with `--output` requires full scope and preserves the original v1 editing projection, including opaque unsupported regions. It creates `note.md` and its identity-bound `note.md.feishu.json` native sidecar without overwriting either file. Existing saved workspaces remain compatible; keep both files together. Push validates the entire local edit plan, checks current native state, and changes supported paragraph and heading text while preserving its existing styles and link destinations. Multiple native text runs can change when unchanged formatting uniquely separates them; adjacent unmarked runs or ambiguous boundaries are rejected rather than assigned guessed styles. Its compact result includes `verified` and complete `changed_blocks` texts with native IDs. An unchanged working copy produces no content write. Pull to a fresh path before making further edits after a push.

This is a text-editing workflow, not a universal Markdown round trip. Unsupported structural or formatting changes stop before writing. Existing `docs +fetch`, `docs +update`, native block operations, media commands and Drive import remain available for rich documents and other original workflows. See the [local Docs guide](skills/lark-doc/references/lark-doc-workspace.md).

Snapshot checks detect observed conflicts but cannot close the race with a concurrent writer; the native revision parameter is not compare-and-swap. Unknown or partial outcomes require reconciliation, not a blind push retry. A [private native lifecycle check](evidence/evaluation/docs-improvements-native-live-final-20260909.json) records which reads and edits were verified against a real document; it does not cover every rich-document or concurrency case.

### Results and recovery

- `--result-detail` indents JSON. `--result-fields id,name` selects supported result fields while retaining outcome and completeness metadata. Domain flags such as `--fields` and `--detail` keep their original meanings.
- `--save-result PATH` saves the full result for supported reads without overwriting an existing file. These artifacts have domain-specific contracts; a saved envelope is not a raw API request body. See the [output contract](docs/contracts/output.md).
- Check both the JSON outcome and process exit status. A timeout after a write may leave its effect unknown. `receipts list` and `receipts inspect ID` expose local attempt evidence for reconciliation. See the [receipt contract](docs/contracts/receipts.md) before retrying an uncertain mutation.

### Why the old skills remain

[`agent-skills/feishu-cli-next/SKILL.md`](agent-skills/feishu-cli-next/SKILL.md) is the entry skill installed by `guide install`. The inherited [`skills/`](skills) directory supplies compatibility documentation and detailed references used by help and `skills read`.

Those guides are not all injected by installing the entry skill. They can still influence an agent and consume context when it reads them on demand. The evaluation includes that work; Next did call `skills read` in some trials. Upstream examples use `lark-cli`; use `feishu-cli-next` with this distribution.

## Evaluation

Each evaluation runs the same agent tasks against Next and the original CLI and counts every attempt's provider tokens, including failures and recovery. The latest [full matrix](evidence/evaluation/full-matrix-20260911/REPORT.md) covered **43 development tasks / 258 attempts**: Next verified **107/129** attempts and the original CLI **102/129**. Per-task gates: **13 met, 22 missed, 8 unavailable**. Six attempts lack complete usage, so whole-matrix savings are unavailable.

These are small development samples on synthetic fixtures; they do not establish statistical success parity, stable savings, or live functionality. [Evaluation](evaluation/README.md) has every condition's results, protocol, and how to run a comparison.

## Development

```sh
make build
make unit-test vet fmt-check   # Go checks while iterating
make offline-test              # everything CI runs; also needs Node and Ruby
```

No check that CI runs needs a Feishu account. Real-account tests run only on your machine with `make live-test SECTION=<section>`; see [testing](docs/testing.md). Start with [contributing](docs/contributing.md) for where code goes and how to add a shortcut or domain, and [AGENTS.md](AGENTS.md) for the binding rules. The [docs index](docs/README.md) lists every guide and contract.

## Upstream and license

This is a separate development distribution derived from `larksuite/cli` and synced to v1.0.97 (`72579c80027c863ca51d5f9affda72a70ab0d8a6`), not an official Lark release. The Go module remains `github.com/larksuite/cli` for source and extension compatibility. Original copyright notices, the [MIT license](LICENSE), and [upstream documentation](README.upstream.md) are retained.
