# Extensions and public hosts

Owner: `extension/`, `tests/plugin_e2e/`, and
`tests/next_sections/extensions/`. Shared command assembly, runtime/output and
receipt implementations remain coordinator-owned. Packaging owns released
platform artifacts and non-native runtime checks.

The implementation reuses the complete frozen SDK. All 92 files under
`extension/` in `docs/reference/sources.json` match their recorded SHA-256 and
the clean sibling reference `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`.
The ledger indexes 301 declarations in 54 files, with 31 reference test files.
These counts describe the ledger, not a promise that its simple symbol extractor
enumerates every constant or method signature. Byte comparison covers the full
files, including less common methods, keyed/unkeyed structs and build tags.
No reference extension signature, platform or supported type is removed.

## Complete compatibility mapping

| Reference surface | Next mapping and contracts | Test owner / evidence |
| --- | --- | --- |
| `extension/command` declarations, domains, typed args/results, shapes, authorization, aliases, value sources, relations, normalization/validation, dry-run and renderers | Same `Define`, `WithCommandSets`, context callbacks and erased host declarations; schema/error validation remains mandatory. Request construction, escaped path segments and numeric decoding retain their existing semantics. | All package tests, `commandtest` suites, external wrapper command process test; new external-module host suite. |
| Command requests and pagination | Same GET/POST/PUT/PATCH/DELETE builders, query/body views, conditional scope checks, `Page`, `CollectPages`/`CollectAllPages`, bounded/interrupted pagination and validated result protocol. Input-stage and dry-run contexts cannot execute network callbacks. | `extension/command`, `extension/command/commandtest`; shared `internal/commandhost` P1 evidence. |
| Command files and download streams | Same `FileTarget`, overwrite policy, file intent, artifacts, URL/OpenAPI download paths, immutable/mutable representation, range validation, exact lengths, idle timeouts, cancellation and close behavior. | `extension/download`, command file tests, wrapper backup dry-run and `commandtest` file scenarios. |
| Credential provider chain | Same registration priority, handle/skip/block behavior, Feishu/Lark accounts, bot/user/auto identities, supported-identity masks, token kinds/scopes/source, environment provider and optional tenant-token lookup. | `extension/credential`, env and sidecar units; public synthetic host account/token process integration. |
| Transport providers | Same last registration selection, pre/post hooks, request cloning/context, optional `ScopedProvider` and `AbortableInterceptor`, preserved typed abort causes. Legacy interceptors still apply to both request classes. | `extension/transport`, P1 runtime transport regressions; public host redirect process and production sidecar signature tests. |
| Auth sidecar variant | Same `authsidecar` build tag, sentinel credential source, account enrichment, HMAC protocol, proxy validation, identity/auth-header commitment and stripping. Normal builds retain the normal credential path. | P1 sidecar/package/demo/round-trip matrix; new production CLI artifact suite tests both brands and both identities. |
| Sidecar deployment examples | Same `authsidecar_demo` and `authsidecar_multi_tenant_demo` builds, target/identity/auth-header validation, redirected credential stripping, audit sanitization, per-client keys and user-map persistence. | P1 single-tenant demo tests; P2 multi-tenant demo tagged tests and executable build. Real deployment remains separate from these synthetic tests. |
| Host FileIO | Same four-method `FileIO` and provider registration; `ExclusiveFileIO` and `WorkspaceFileIO` remain optional. Runtime providers stay invocation-scoped. Host path policies and canonical references remain authoritative. | External-module process tests use a memory backend, ordinary legacy-provider read, denied path, unsupported exclusive save, full artifact contents and concurrent destination collision; P1 local FileIO regressions cover OS operations. |
| Content safety | Same provider registry, concurrent scan contract, normalized command/data input, provider stderr writer and alerts. | `extension/contentsafety` and P1 command/output content-safety integration tests. |
| Plugin builders and registrar | Same registration/snapshots, required CLI versions, fail-open/fail-closed policy, observer, onion wrappers, aborts, lifecycle events, selectors, risk/identity parsing and command restrictions. | Complete `extension/platform` suite plus inherited plugin process matrix, including startup failures/panics, denied-command observation, wrapper aborts and failure isolation. |
| Distribution skills/guidance | Same host-wired embedded base, optional `EmbeddedSkillsRegistrar`, allow/remove/overlay/base semantics, dependency validation, reference remaps and typed startup guards. Concealment remains a host opt-in; restriction remains enforcement. | Existing plugin process tests cover base wiring, domain help, missing dependencies, replacements/remaps, removed/concealed pointers and filesystem panic handling. |
| Public example distributions | Same audit-observer, readonly-policy and typed chat-brief examples. Their public package imports remain valid. | `make examples-build`, chat-brief package tests and existing external customer-module builds. |

The source ledger is the exhaustive file/symbol mapping; the table groups those
entries by behavior instead of maintaining a second handwritten symbol list.
Generated commands, platform keychains, socket/event adapters and skill
installation lifecycle are mapped by the foundation, events and packaging owners.

## Agent host interface

Existing wrappers calling `cmd.Execute()` or `cmd.ExecuteWithOptions(...)` retain
legacy formatting and flags. A wrapper opts into the next interface with
`cmd.WithAgentMode()`. `cmd.SetAgentSkillContent(fs.FS)` supplies its project
entry skill; `cmd.SetEmbeddedSkillContent` and
`cmd.SetEmbeddedAffordanceContent` retain the compatibility content contract.
Wrappers must wire their own content; Go embedding cannot reach into a replaced
dependency module. See the public [platform guide](../../extension/platform/README.md).

In agent mode, public command contributions participate in compact full JSON,
projection, read artifacts and durable mutation outcomes without new business
command APIs. The [output contract](../contracts/output.md) requires an optional
exclusive FileIO capability only for result artifacts. A legacy provider remains
usable for ordinary reads. Unsupported saves and denied paths fail before the
business request. A concurrent destination is rejected at commit and preserved.
Artifacts retain full Unicode/content, leading-zero IDs and the host's canonical
reference; a projection does not become the stored data.

The shared verified-binding follow-up resolves bot tenant evidence with the
selected TAT at `GET /open-apis/tenant/v2/tenant/query`. The external host fixture
supplies that explicit evidence and counts verification separately from business
requests. It does not manufacture a tenant from an app ID. User artifacts retain
their observed user/tenant evidence. Ordinary reads and artifact preflight
failures do not issue this additional verification request.

The [receipt contract](../contracts/receipts.md) applies to host commands too.
The process suite independently counts requests and verifies acknowledged,
rejected and asynchronous/unknown results, typed failure envelopes and exit
codes. It does not infer completion from `command.Success` when the transport
only acknowledged asynchronous acceptance. No request is automatically replayed.

## Section acceptance evidence

All checks run with `scripts/next_test.py` or its `isolated_env` helper. The helper
replaces HOME/config/temp, strips inherited credentials, disables metadata and
update traffic, and sends external proxy traffic to a closed local port.
Synthetic backends never forward the target headers they inspect. The external
host fixture is compiled as a separate Go module, so imports of CLI internals
would fail. It uses synthetic bot credentials and memory FileIO; its private
local report file is test evidence, separate from product stdout/stderr.

The complete selectable section command is:

```sh
python3 scripts/next_test.py --section extensions --suite all --report .tmp/next-tests/extensions.json
```

This includes all extension packages, the inherited plugin fork matrix and all
new section process tests. For the P2 boundary, the unchanged P1 full plugin and
sidecar matrix is reused explicitly rather than rebuilding every unchanged fork.
[P1 evidence](foundation.md) records 456 passing plugin/sidecar test events with
no skips, customer forks built from `d4c705c`, both public example builds and the
authsidecar build. The P2 checks and counts below are recorded separately and
never added to that historical count as if they were one run.

P2 checks began from shared HEAD `4ce36da`, with the new owned files and parallel
working-tree changes present. Raw offline results are in
`.tmp/extensions-final.json` and `.tmp/extensions-e2e-final.json`:

| Executed command inside `isolated_env` | Outcome |
| --- | --- |
| `go test -json -count=1 -timeout=10m ./extension/... ./tests/next_sections/extensions` | 335 passing test events: 318 extension events and 17 new process events; no failures or skips. |
| `go test -json -count=1 -timeout=10m ./tests/next_sections/extensions` after removing unnecessary test-build caching | All 17 process events pass again, no failures or skips. |
| `go test -json -count=1 -tags authsidecar ./extension/credential/sidecar ./extension/transport/sidecar` | All 21 events pass, no failures or skips. |
| `go test -json -count=1 ./tests/plugin_e2e -run 'TestSmokeForkBoots\|TestExistingRestrictPluginKeepsLegacyEnvelopeWithoutHostOptIn\|TestForkSkillsAllowRemoveComposeAgainstEmbeddedBase'` | All three selected complete customer-fork scenarios pass, no failures or skips. The unselected unchanged matrix retains its explicitly historical P1 evidence. |
| `make examples-build` | Both public platform examples compile successfully. |
| `go test -json -count=1 -tags authsidecar_multi_tenant_demo ./sidecar/server-multi-tenant-demo` | All 53 events pass, no failures or skips; `.tmp/extensions-multi-tenant.json` records this additional build variant. |
| `go build -tags authsidecar_multi_tenant_demo -o <isolated-temp>/multi-tenant-demo ./sidecar/server-multi-tenant-demo` | The multi-tenant demo executable compiles successfully. |

After the shared verified-binding implementation and central fixture update
`08ca8e8`, the complete 17-event section process package passed again with zero
failures or skips. `.tmp/extensions-verified-binding.json` records the isolated
run. The external provider now supplies genuine synthetic tenant endpoint
evidence; bot artifacts assert that evidence rather than the old unverified
binding. Ordinary reads and rejected artifact preflights still perform no tenant
verification, and separate request counts preserve the collision/no-retry checks.

`git diff --check` passes for the owned files. No product behavior or exported
API changed in this section; the new tests prove that the shared next interface
works through public external-module integrations. Full repository lint, vet and
release gates remain coordinator-owned and are not claimed by these scoped runs.

## Remaining verification boundaries

Offline host/provider integration needs no Feishu content write. Live tenant
authorization, real sidecar deployment, host-specific remote FileIO atomicity,
and backend effect semantics are not established by synthetic fixtures. Hosts
must implement their declared exclusive/atomic capability; the SDK cannot
prove an arbitrary third-party provider honors its contract.

Native evidence is Darwin/arm64. Windows permission/directory-fsync differences
and platform keychains remain as documented in the shared contracts; Linux and
Windows runtime/distribution checks belong to the packaging release matrix and
are not claimed as passed here. Source/build-tag preservation is not runtime
verification. P2 comparative development tasks are coordinator-scheduled; the
held-out suite remains untouched. These limits keep release acceptance open
without narrowing the shipped SDK.
