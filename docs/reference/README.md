# Frozen reference inventory

For configured Feishu/Lark guard coverage, use the [separate brand supplement](BRANDS.md) and pass `--reference-registrations docs/reference/configured-brand-registrations.json.gz` to the checker. The original frozen registration file is an unrestricted bootstrap inventory; its two brand labels do not establish configured runtime conditions.

The source reference is `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3` of
`github.com/larksuite/cli`, with no tracked or untracked local changes at capture.
The checkout is never modified by the inventory scripts. Copyright notices and
the MIT license must accompany redistribution. The target reuses the upstream Go
module to retain public import compatibility; the product binary is
`feishu-cli-next`. Packaging owns the resulting distribution identity.

The captured catalog has 938 command nodes including the root, 535 shortcuts,
251 generated API methods across 15 services, and 25 compiled event keys. The
source inventory records 2,577 contract files, 1,252 test files, and 61 files with
explicit Go build constraints. Counts describe discovered capabilities; none is
marked functionally verified merely because it is registered.

## Reproduce the inventory

Prerequisites: Python 3, Git, and the Go toolchain declared in `go.mod`; initial Go
dependency downloads may require network access. Exporting registrations never
executes a CLI operation, invokes a live backend, or reads the user's CLI profile.
Both brand variants run with isolated configuration and remote metadata disabled.
The exporter uses a temporary copy of tracked working files, retaining tracked
local modifications if present. Untracked source additions are recorded in Git
status but are not incorporated; reconcile those before freezing a new reference.

```sh
python3 -m unittest discover -s scripts -p next_inventory_test.py -v
python3 scripts/next_inventory.py freeze \
  --reference ../feishu-cli-review \
  --metadata internal/registry/meta_data.json
python3 scripts/next_inventory.py check --surface legacy --target ../feishu-cli-review
python3 scripts/next_inventory.py check --surface next --target .
```

The canonical build fetched metadata from the public Feishu API because the
clean reference contains only an empty fallback. `meta_data.json.gz` preserves
the exact downloaded bytes; its SHA-256 is in `coverage.json`. Rebuilding the
frozen catalog does not require another metadata fetch:

```sh
python3 -c 'import gzip,pathlib; pathlib.Path("internal/registry/meta_data.json").write_bytes(gzip.decompress(pathlib.Path("docs/reference/meta_data.json.gz").read_bytes()))'
make build
```

The Feishu and Lark brand exports deliberately share this exact build catalog.
Runtime remote-overlay loading, separately selected brand catalogs, and platform
behavior remain contracts covered by their source/tests, not by this snapshot's
registration comparison. Later upstream catalog changes require a separate
reconciliation; they must not silently redefine the frozen reference.

## Evidence and reconciliation

- `coverage.json` maps every registered command to its target entry point, section,
  owner, contract pointer, and verification status. Non-command events,
  extensions, and packaging retain separate sections.
- `registrations.json.gz` comes from real Cobra trees, registered shortcuts and
  typed schemas, supported flag schemas, and the compiled event catalog. It
  includes inherited flags, types/defaults/aliases/constraints, conditional
  scopes, identities, event parameters, and resolved event output schemas.
- `meta_data.json.gz` retains complete generated request/response fields, enum
  options, nested resource/method identities, and type definitions.
- `sources.json` hashes all source contracts, tests, scripts and build files. The
  public symbol index is a navigation aid, not a Go API compatibility checker;
  complete file hashes also detect grouped declarations and signature changes.
- `p0-e2e.json` is the machine-readable result of the complete process-based
  self-reconciliation against the reference. Five synthetic tests additionally
  exercise nested metadata, missing flags/events/schemas/types, build conditions,
  source changes, deterministic compression, and malformed-input exit behavior.

`check --surface next --target .` is the candidate/release inventory command.
`next` is the default; it uses the same `WithAgentMode()` Build option as the
product's `main.go`. Use `--surface legacy` explicitly for upstream reference
self-reconciliation. The script inserts that option only into its temporary
candidate exporter, so the shipped exporter remains compilable against the
untouched upstream source, which has no `WithAgentMode` symbol. The report names
the selected surface and candidate Git revision, reports command counts and new
commands/root flags, and lists additions separately from compatibility failures.

`check` exits 0 when frozen contracts and source files agree, 1 when review is
required, and 2 for execution/input errors. Its JSON report distinguishes missing
contracts, changed contracts, missing source, and changed source needing review.
It allows new dictionary entries but reports changed existing contracts. Arrays
are compared exactly, including enum options and aliases. The report also records tracked-tree SHA-256 fingerprints, revisions and dirty
scope before and after each brand export. A concurrent tracked-file change is
reported as `tracked_source_changed_during_check`; it must not be presented as
one stable snapshot. Untracked paths are visible in dirty scope but are not
compiled: commit new implementation files before final reconciliation. Explicit
presence fields distinguish removed commands, shortcuts, event keys and flags
from additions, and include exact changed existing flag contracts.

It does not whitelist
an implementation change automatically. Section owners must explain intentional
changes and attach relevant tests; source equality is only a conservative
starting point for review. The candidate registration comparison always compiles
against the pinned catalog, so inspect the actual release build's embedded
catalog hash separately. It does not prove that an independently built binary
used the pinned metadata.

The global source-test index is discovery evidence. Operation-specific unit,
process E2E, live, and agent evaluation results must still be mapped by each
section owner. Live behavior, mutation safety, external races, platform execution,
and token savings are **unverified** by P0. Held-out tasks were not used.

## Build and host surfaces to retain

The reference release configuration declares Darwin, Linux and Windows plus
amd64, arm64 and riscv64. These are declared dimensions, not a promise that all
nine combinations are valid Go targets: Go supports riscv64 here on Linux. The
npm installer separately maps Node `darwin/linux/win32` and `x64/arm64/riscv64`,
requires Node >=16, verifies downloaded checksums, extracts Windows ZIP or Unix
tar archives, and runs an installation wizard. Native Make install/uninstall,
source build, npm installation, self-update, skills synchronization, shell
completion, optional macOS signing/notarization, and release checksums are
preserved source surfaces.

`main_authsidecar.go` and extension credential/transport providers use the
`authsidecar` tag. The demo server uses `authsidecar_demo`. All explicit tag
expressions, including Linux/macOS/Windows keychains, process control, locks,
audit, file I/O, and updater implementations, are listed in `sources.json`.
Implicit filename platform selection is also retained by the complete source
list. Reference commands for broader platform/host validation include
`make sidecar-test`, `make examples-build`, `make script-test`, and
`go test ./tests/plugin_e2e/...`; P0 does not substitute for running them.

The public extension packages include command definitions, credential providers,
transport interceptors, platform command policies/observers and distribution
configuration, event hosting, file I/O, and bundled examples. Their exact files,
exported-symbol navigation index and tests are retained for the foundation and
extension section owners. Apps have a Feishu-only brand restriction; other
conditional behavior must be evaluated in its owning package, including failure
contracts rather than help visibility alone.
