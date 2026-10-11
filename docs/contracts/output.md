# Agent JSON and result artifacts

Owner: `internal/output` (`AgentOptions`, `WriteJSON`, `Emitter`). Root command
registration opts the next distribution into this contract. Existing SDK callers
retain legacy formatting unless they explicitly wrap their invocation's writer.

## JSON

JSON results are compact by default without dropping any fields. Success remains
on stdout; typed errors remain on stderr with the existing error category and
exit code. `--result-detail` indents the full JSON result. It cannot be combined
with `--result-fields`. Existing domain flags named `--fields` and `--detail`
keep their original semantics.

Embedded skill reads keep the file bytes intact on stdout. Main-skill reads add
a short version-matched reference-routing hint on stderr (or the separate
`guidance` field with `--json`); reference-file reads add no hint. The hint keeps
both the own-skill command and the cross-skill `../` rewrite without repeating a
paragraph on every read.

`--result-fields title,owner.name` selects business data fields. Selection applies
to `data`, or to each item in an array or an `items`, `records`, `rows`, `files`, `messages`, `tasks`, `created_tasks`, or `successful_tasks`
collection. Dotted paths preserve nested structure; arrays need no `[]` syntax.
Names may contain letters, digits, `_`, or `-`; wildcard, jq, and bracket syntax
are rejected before execution. Missing fields are omitted, not manufactured as
null. A missing field cannot be schema-validated generically before a response.

The complete envelope, resource IDs (`id`, `guid`, `token`, `*_id`, `*_ids`, `*_token`),
URLs, types, identity, status, `unchanged`/`no_op`/`noop` no-op markers, pagination, counts, errors, warnings, recovery,
notices and safety alerts remain visible. Task collections retain native GUIDs
and tasklist identity, `confirmed` fields, `updated_fields` and failed-task details.
Calendar's `updated_event` effect object retains the event identity and attendee
change counts. Apps release `current_node_info` stays complete, including its
approval URL and status, even with dotted selections or artifact summaries.
Other nested resource objects use explicit dotted paths. Results containing failed, partial,
unknown or unsupported outcomes retain their full data. This conservative shared
projection must not replace a domain-specific schema that has more outcome fields.
Projection never mutates caller data. JSON numbers use their serialized decimal
representation without a float64 conversion; null, empty string, omitted fields,
Unicode and leading-zero strings remain distinct. Numeric precision already lost
by an upstream decoder cannot be recovered here.

Projection and result artifacts require non-streaming JSON without `--raw`,
`--jq`, or IM's `--concise` Markdown mode. These conflicts are rejected before
command execution; explicitly disabling a boolean mode remains valid. IM dry-run
previews remain JSON and support result controls even with `--concise`.
Explicit CSV, NDJSON, table and pretty renderers retain their contracts.
Default agent formatting does not intercept arbitrary raw, jq or stream bytes.
Native structured XML/HTML renderers (`OutRaw`/`OutFormatRaw`) use an internal
`DisableHTMLEscape` encoding option, distinct from explicit raw mode. Their JSON
supports result controls, keeps inline XML/HTML unescaped, and preserves full
string values in artifacts. This does not permit user `--raw` combinations or
raw API artifact saves.
The command validates flag compatibility before execution. The emitter checks
again for embedded callers. `AgentOptions.Finish()` catches ignored legacy output
errors and commands that do not produce a structured result; arbitrary text
writes with projection/artifact controls are rejected rather than silently ignored.

For tracked mutations, the envelope includes `mutation.outcome`, receipt IDs and
`persistence_failed` when applicable. Unknown, partial, confirmed-unapplied and
receipt-persistence failures force `ok:false` even if a legacy command interpreted
the transport response as success. Raw and jq envelope rendering receives this
same status; jq remains an explicitly requested filter. Exit status is owned by
the receipt/command layer. Read-only output has no mutation field.

## Artifacts

`--save-result PATH` is available for declared business reads. It saves the full
JSON result regardless of `--result-fields`. Authentication/lifecycle results and
mutations cannot use it. This avoids confusing a remote mutation with failure to
save its local output. Known authentication credential fields in result data are
also rejected, including API keys, PATs, passwords, credential containers and
Git credentials. Field matching is case-insensitive. Resource and pagination
tokens remain supported. Generic raw API calls cannot save result artifacts
because their response sensitivity is unknown; their raw stdout stays available.

The invocation's `FileIO` provider must implement `fileio.ExclusiveFileIO`.
Preflight validates the path and refuses an existing destination. Commit is
atomic and exclusive, including a target created concurrently after validation.
The local provider creates parent directories with mode 0700 and files with mode
0600. The host provider's path policy remains authoritative; no direct filesystem
fallback bypasses it. Provider failures retain their cause and typed error.

The file is a versioned container, not a lossy projection:

```json
{"schema":"feishu-cli-next/result/v1","binding":{"environment":"feishu","app_id":"cli_example","identity":"bot","endpoint":"https://open.feishu.cn","tenant_key":"tenant_example","tenant_verified":true},"result":{"ok":true,"identity":"bot","data":{"items":[{"id":"1","title":"Example"}],"has_more":false}}}
```

Stdout keeps envelope fields, relevant resource/continuation facts, and an
`artifact` object with `schema`, canonical host `ref`, `bytes`, `binding`, and
`count`/`complete` only when known. Count comes from the returned collection or
explicit result metadata; it is not assumed to equal the entire backend resource.
Completeness comes from explicit pagination metadata, `has_more`, or `complete`;
serializing a response to EOF never implies completeness. Continuations stay
available in result metadata or retained data fields and in the full artifact.
Generic `--page-all` fails on any later-page API or transport error and on a
missing or repeated continuation token. Aggregated JSON and artifact output are
not published after these failures. Streaming formats may already have emitted
earlier successful pages; their nonzero exit and typed stderr error indicate that
the stream is incomplete. An explicit page limit retains `has_more:true` when
more pages exist; the aggregate is not a snapshot or a resumable cursor container.
HTTP 4xx/5xx remains a failure even if a JSON body omits a business error code or
contains `code:0`. A structured nonzero business error keeps its original typed
classification and diagnostic fields; otherwise the HTTP status supplies it.
Failed/partial/unknown results keep their full decision-critical data on stdout
as well, even when this makes the reference response longer. Bare object results
retain their native top-level metadata and never acquire a fabricated `ok:true`.
For a safe bare result only recognized collections are removed from stdout;
unknown fields remain visible. Mail `messages` collections contribute their
observed count, and mailbox identity, continuation and notices remain intact.

Binding records the brand environment, resolved API endpoint, app ID, effective
identity and optional profile. Requested artifact operations verify the selected
credential through a read-only identity endpoint, once per invocation, and reuse
current account enrichment when it already includes verified user and tenant
facts. Ordinary reads without artifact options do not add these lookups.

User operations call `GET /open-apis/authen/v1/user_info` with the selected UAT
when current evidence is unavailable. A persisted config user ID is checked
against the observed user; it is not treated as verification by itself. User
artifacts include `user_verified:true` and the observed user and tenant IDs.
Bot operations call `GET /open-apis/tenant/v2/tenant/query` with the selected TAT
and include the observed `tenant_key` and `tenant_verified:true`; they do not
resolve user credentials or infer a tenant from an app ID. The bot endpoint
requires `tenant:tenant:readonly`; its optional domain-field permission is not
needed. See the [official tenant query contract](https://open.feishu.cn/document/server-docs/tenant-v2/query).

Missing identity or tenant evidence and lookup permission failures stop the
artifact operation explicitly. The CLI never grants extra scopes or falls back
to unverified output. Binding contains no token or secret. Domain artifact
consumers must validate schema and current binding before reusing data; the
generic container is not an import command.

Domain code obtains the same binding through `RuntimeContext.ResolveArtifactBinding()`
and uses the `common.ArtifactBinding` alias. The command output and runtime method
share `Factory.ResolveArtifactBinding`; they do not independently resolve accounts.
Dry-run runtimes reject binding resolution so previews cannot fetch credentials.

Before reusing a file, domains call
`common.ValidateArtifactBinding(saved, current, "--input", requireVerifiedIdentity)`.
This requires matching nonempty brand, endpoint, app, effective identity and user
ID for user operations. Any known tenant must match; a known source tenant cannot
match an unknown current tenant. Profile names are local labels and are ignored.
The boolean is deliberately explicit: `true` requires verified tenant evidence
on both sides and verified user evidence on both sides for user identity. `false`
permits matching unverified evidence while still rejecting every identity mismatch;
it does not upgrade that evidence to verified. Legacy artifacts missing endpoint
or required identity facts need to be exported again. Consumers independently
validate their native schema, completeness, types and supported effects before a
write; this helper verifies only account binding.

Artifacts remain until the user removes their chosen files; there is no implicit
cleanup or overwrite. Large result payloads stay out of model context, but the
current shared JSON emitter still serializes an already materialized result in
process memory. This is not a bounded-memory backend streaming export, and the
artifact is not a proof of snapshot-consistent pagination.

## Introspection

Local API introspection has a separate schema envelope owned by `internal/schema`.
`schema <path>` retains its full input/output schema and `_meta` contract.
Explicit `--input-only` omits only `outputSchema`; input fields (including required
markers, types, carriers, bounds, defaults and enums), identities, scopes, risk,
danger and affordance guidance remain identical. It is a request-planning view,
not a complete MCP tool spec. Exact methods remain objects; broader paths remain
sorted collections, with the same strict-mode and distribution visibility filters.

## Verification

Focused tests: `go test ./internal/output`.
Introspection: `go test ./cmd/schema ./cmd/discover ./internal/schema`.
Process scenarios: `tests/next_foundation/output_e2e_test.go`, run by
`python3 scripts/next_test.py --section foundation --suite e2e` at the completed
foundation gate. The fixture uses the real authsidecar build against a local
synthetic backend. Offline tests do not verify live pagination or tenant binding.
