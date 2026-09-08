# Feishu CLI Next

A Feishu/Lark CLI for AI agents that work through a shell and files. It builds on the official Lark CLI and adds command discovery, compact results, file artifacts, typed data transfer, and mutation recovery evidence.

The goal is to reduce the tokens needed to complete a task while preserving every original command and workflow, data fidelity, and successful outcomes. Results remain task-dependent. The current evaluation targets **more than 50% fewer tokens for Docs read/edit and at least 30% fewer for each other evaluated function versus the original CLI**, with failed-attempt costs included and no observed success regression.

[Quick start](#quick-start) · [Usage](#usage) · [Evaluation report](evaluation/evidence/token-efficiency-05/REPORT.md) · [Implementation status](docs/IMPLEMENTATION.md)

## What this project adds

| Capability | What it provides |
| --- | --- |
| Command discovery | Search registered commands with bounded results; optional flag descriptions expose accepted inputs without a separate help lookup. |
| Compact JSON | Full JSON data in compact form by default; optional indentation and field selection. |
| Local Docs editing | Pull readable Markdown with native block IDs, edit supported text locally, then push targeted native changes with verified text results. |
| File artifacts | Save supported read results to versioned, identity-bound files for local processing and handoff. |
| Typed data workflows | Sheets export/import and Base record workflows with validation, type preservation, and operation-specific no-op detection. |
| Mutation evidence | Durable local receipts distinguish rejected, partial, and unknown outcomes without replaying writes. The exact raw Contact basic-profile POST read retains read-only behavior. |
| One entry skill | Install direct routes for common tasks, including local Docs text edits, native XML operations, single-row Base updates, exact Sheets exports and Drive transfers; load deeper guidance on demand. |
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

# Read document facts as Markdown with native block IDs.
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --as user

# Export an explicit Sheets range to a local artifact.
./feishu-cli-next sheets +data-export \
  --spreadsheet-token SPREADSHEET_TOKEN --sheet-id SHEET_ID \
  --range A1:E100 --save-result cells.json --as user
```

Replace resource placeholders with your own IDs. Discovery describes the local command surface; access still depends on the selected account and scopes. Shortcuts start with `+`; generated service commands expose catalog methods, and `api` provides raw OpenAPI access.

### Edit Docs through a local working copy

```sh
./feishu-cli-next docs +pull --doc DOCUMENT_TOKEN --output note.md --as user
# Edit supported text in note.md; keep its block markers and formatting intact.
./feishu-cli-next docs +push --file note.md --dry-run --as user
./feishu-cli-next docs +push --file note.md --as user
```

Pull creates `note.md` and its identity-bound `note.md.feishu.json` native sidecar without overwriting either file. Keep them together. Push validates the entire local edit plan, checks current native state, and changes supported paragraph and heading text while preserving its existing styles and link destinations. Its compact result includes `verified` and complete `changed_blocks` texts with native IDs. An unchanged working copy produces no content write. Pull to a fresh path before making further edits after a push.

This is a text-editing workflow, not a universal Markdown round trip. Unsupported structural or formatting changes stop before writing. Existing `docs +fetch`, `docs +update`, native block operations, media commands and Drive import remain available for rich documents and other original workflows. See the [local Docs guide](skills/lark-doc/references/lark-doc-workspace.md).

Snapshot checks detect observed conflicts but cannot close the race with a concurrent writer; the native revision parameter is not compare-and-swap. Unknown or partial outcomes require reconciliation, not a blind push retry. A [private native lifecycle](evaluation/evidence/docs-workspace-native-live-20260909.json) verified two text edits, bold/link/Unicode preservation and an untouched table through independent original-CLI readback. That bounded test does not establish every rich-document or concurrency case.

### Results and recovery

- `--result-detail` indents JSON. `--result-fields id,name` selects supported result fields while retaining outcome and completeness metadata. Domain flags such as `--fields` and `--detail` keep their original meanings.
- `--save-result PATH` saves the full result for supported reads without overwriting an existing file. These artifacts have domain-specific contracts; a saved envelope is not a raw API request body. See the [output contract](docs/contracts/output.md).
- Check both the JSON outcome and process exit status. A timeout after a write may leave its effect unknown. `receipts list` and `receipts inspect ID` expose local attempt evidence for reconciliation. See the [receipt contract](docs/contracts/receipts.md) before retrying an uncertain mutation.

### Why the old skills remain

[`agent-skills/feishu-cli-next/SKILL.md`](agent-skills/feishu-cli-next/SKILL.md) is the entry skill installed by `guide install`. The inherited [`skills/`](skills) directory supplies compatibility documentation and detailed references used by help and `skills read`.

Those guides are not all injected by installing the entry skill. They can still influence an agent and consume context when it reads them on demand. The evaluation includes that work; Next did call `skills read` in some trials. Upstream examples use `lark-cli`; use `feishu-cli-next` with this distribution.

## Latest evaluation

**[Read condition 05 →](evaluation/evidence/token-efficiency-05/REPORT.md)**

The target was **not met: 3/8 functions met their gate, 3 missed and 2 were unavailable**. Condition 05 completed 48 fresh attempts against the original CLI: eight tasks, three repetitions, two arms, using **GPT-5.6 Luna with medium reasoning through native Codex CLI 0.153.4**. The benchmark uses signed synthetic Feishu fixtures; private native QA is separate evidence.

**40/48 attempts verified: Next 22/24, original 18/24.** Docs edit regressed from original 3/3 to Next 2/3. Observed provider usage was **3,602,432 tokens, a lower bound** because four Drive attempts had incomplete usage.

| Function | Tokens per verified success vs original | Gate |
| --- | ---: | --- |
| Discovery | 14.8% more | Missed |
| Contact search | 58.2% less | Met |
| Docs read | 6.4% less; both 3/3 verified | Missed |
| Docs edit | 172.0% more; Next 2/3 vs original 3/3 | Missed |
| Markdown creation | 60.7% less | Met |
| Base update | 36.6% less | Met |
| Sheets export | Unavailable: original 0/3 verified | Unavailable |
| Drive roundtrip | Unavailable: incomplete usage and original 0/3 | Unavailable |

The primary gate requires strictly more than 50% savings for Docs read/edit and at least 30% for each other function, with no observed success regression. All attempt costs, including failures and recovery, count toward tokens per verified success. Cached input is counted once; missing usage is never zero. These small development samples do not establish stable savings, statistical success-rate parity or full live functionality.

Docs-edit traces show rejected `+push` calls followed by `+update` fallback. The retained traces omit local file contents and validation-error text, so they do not establish the rejection's exact cause. The failed result and fallback costs remain included.

[Frozen environment](evaluation/evidence/token-efficiency-05/environment.json) · [Readiness review](evaluation/evidence/token-efficiency-05/readiness-review.json) · [Protocol](evaluation/token-efficiency-05.md) · [All evaluation conditions](evaluation/README.md)

Historical [condition 04](evaluation/evidence/token-efficiency-04/REPORT.md) remains a separate exploratory comparison: 24 new candidate attempts, 48 reused baselines and 5/8 gates met against unchanged Next. Its 50% target and reused-baseline results do not satisfy condition 05's original-CLI comparison. Earlier [pilot 01](evaluation/evidence/paired-luna-medium-01/REPORT.md), [condition 02](evaluation/evidence/token-efficiency-02/REPORT.md) and [condition 03](evaluation/evidence/token-efficiency-03/REPORT.md) retain all their costs, failures and accounting limits.

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
