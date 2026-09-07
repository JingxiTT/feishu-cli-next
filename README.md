# feishu-cli-next

A Feishu/Lark CLI for capable shell and file agents. This development build reuses the pinned upstream CLI's domain commands, generated service catalog, raw API, authentication, events and extension SDK. It adds bounded command discovery, compact JSON, selected result fields, file artifacts, durable mutation evidence, and one project-installable English skill.

Implementation and verification are in progress. [The implementation ledger](docs/IMPLEMENTATION.md) distinguishes passing offline checks and scoped foundation measurements from outstanding live, capability and comparative evaluation requirements. Full-product efficiency and release acceptance remain unverified.

Measured results are mixed: [foundation trials](evaluation/evidence/foundation/REPORT.md) show lower average usage for Next on the tested tasks, while [Contact trials](evaluation/evidence/contact/REPORT.md) show higher usage than both comparison arms despite all nine runs succeeding. The [nine-section development batch](evaluation/evidence/read-312af93/REPORT.md) also records savings against many original guides but regressions against concise guidance, with Wiki attribution limits. These scoped results do not establish an overall performance improvement.

## Build and setup

Requires Go 1.23 or newer and Python 3. The first Go build may download dependencies. The audited API catalog is bundled with the source; canonical builds verify and restore its checksum without fetching a changing catalog.

```sh
make build
./feishu-cli-next --version
./feishu-cli-next auth status
./feishu-cli-next guide install --project /path/to/your/project
```

Authentication retains the upstream configuration, profiles, credential providers, Feishu/Lark brands and `LARKSUITE_CLI_*` environment variables. See [upstream setup and authentication](README.upstream.md) for account configuration. Existing domain command names and flags are preserved. The binary is called `feishu-cli-next`; the Go module remains `github.com/larksuite/cli` for source and extension compatibility.

`make install PREFIX=/your/prefix` installs an owned binary; `make update-install` updates it and `make uninstall` removes it using the same prefix. These commands refuse unrelated or edited files. `guide update` and `guide uninstall` manage only the intact CLI-owned project skill. [Distribution instructions](docs/distribution.md) cover checksummed archives and local npm installation. The development build has no published next update channel; it cannot replace itself from the official upstream npm package. Release packaging remains open.

## Use

```sh
feishu-cli-next discover docs --limit 8
feishu-cli-next discover 'docs +fetch' --flags
feishu-cli-next docs +fetch --help
feishu-cli-next schema drive files list
feishu-cli-next guide
```

Discovery uses this executable's registered commands and respects hidden subtrees. Results include query, total and continuation offset. It describes the local surface, not live permissions. Domain help and schemas provide additional detail on demand; the inherited skill content remains available through `skills read` as compatibility documentation.

JSON results retain all fields by default with compact formatting. `--result-detail` indents them. `--result-fields id,name` projects supported JSON data while retaining outcome and completeness information. These names deliberately avoid existing domain input flags named `--fields` and `--detail`. Raw, jq, human and streaming formats retain their own contracts.

For supported read operations, `--save-result PATH` writes a versioned, identity-bound JSON artifact through the host's atomic file interface and returns a reference. It refuses existing files and unsupported combinations before the operation. It is an envelope artifact, not a raw API request body or a universal cross-domain data format. Inspect [the output contract](docs/contracts/output.md) for projection and handoff semantics.

Mutations persist attempt evidence before dispatch. A timeout is not proof of failure; an accepted asynchronous request is not proof of completed effects. `receipts list`, `receipts inspect ID`, and explicit `receipts cleanup ID` expose local evidence. The CLI does not replay unknown writes. Read [mutation evidence and its limitations](docs/contracts/receipts.md), including incomplete tenant evidence and domain-specific reconciliation still required.

Success data is on stdout and diagnostics on stderr. Existing validation/auth/network exit codes remain. Attempt evidence uses exit 20 for confirmed unapplied requests, 21 for a known mixture of applied and unapplied requests, and 22 for unknown outcomes or failed post-dispatch persistence. Inspect both outcome data and the process exit status.

## Development checks

```sh
python3 -m unittest discover -s scripts -p 'next_*test.py'
python3 -m unittest discover -s tests/next_evaluation
python3 scripts/next_test.py --section foundation --suite all \
  --binary ./feishu-cli-next --report .tmp/next-tests/foundation.json
python3 scripts/next_inventory.py check --target . \
  --reference-registrations docs/reference/configured-brand-registrations.json.gz
```

[The offline runner](tests/next_foundation/README.md) isolates local configuration and credentials. Its inherited domain selector reports excluded tests; that subset is not full domain acceptance. Live tests require separately verified authorized fixtures. Ordinary CI must not use personal credentials. [Evaluation tooling](evaluation/README.md) supports three balanced arms with actual model usage and independent effect verification; controlled tests are not product measurements.

Reference source: `larksuite/cli` at `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, MIT licensed. Original copyright notices, [LICENSE](LICENSE), [reference inventory](docs/reference/README.md), and [upstream documentation](README.upstream.md) are retained. This is a separate development distribution, not an official Lark release.
