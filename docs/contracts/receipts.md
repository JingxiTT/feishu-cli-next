# Mutation evidence

Agent mode opts the existing direct HTTP and SDK transports into attempt
journaling. Compatibility mode leaves the reference transport behavior unchanged.
Each potentially mutating request receives a random receipt ID. The CLI persists
`unknown` / `dispatch_started` before calling the transport. A failed initial
save prevents that request. Journaling adds no retries.

Receipts contain version 1, timestamps, method, effective environment, profile
when available, app ID, resolved identity, verified user ID when applicable,
and numeric response evidence. They contain no request URLs, query strings,
headers, bodies, response content, or credentials. Existing user-info enrichment supplies a verified `tenant_key` when the backend
returns one. `user_verified:true` identifies a current user-info response; a
persisted/provider user ID remains resolved account evidence. Missing user IDs
are resolved with the selected credential source and a cached user-info read.
Missing tenant information alone adds no network call; `tenant_verified:false`
explicitly records the limitation for bot identities and incomplete responses.
Full tenant binding is **unfinished** for those identity modes. Receipts are inspection evidence, never replay
inputs; no operation reuses them to issue a write under another binding.

`receipts list` and `receipts inspect <id>` read only local evidence, including
its original binding. `receipts cleanup <id>` explicitly removes that receipt,
even when its outcome is unknown; it does not undo any remote effect. Retention
is indefinite until explicit cleanup. Empty lock files remain to avoid inode
reuse races. Corrupt/incompatible records require manual repair and are never
silently reinterpreted or deleted. Receipt commands do not resolve credentials
or contact a backend; inspection remains available after switching accounts.

## Outcomes and exits

The command boundary emits a separate JSON object on stderr:
`{"mutation": {"receipts": ["<receipt-id>"], "outcome": "...", "applied": 1,
"unapplied": 0, "unknown": 0}}`. This supplements original result and typed
error contracts. The normal diagnostic includes receipt IDs and counts;
`receipts inspect` returns full binding and numeric evidence. JSON stdout marks
`ok:false` and includes the mutation outcome for unknown, partial, unapplied,
or failed final persistence, preserving the original data and typed errors.

| Evidence | Recorded outcome | Agent exit override |
| --- | --- | --- |
| Successful synchronous API envelope (`code:0`, 2xx other than 202), without detected item failures | `confirmed_applied` | none; original exit remains |
| HTTP 401/403/429, known gateway token/scope rejection, or cancellation observed before dispatch | `confirmed_unapplied` | 20 |
| Both applied and unapplied requests in one invocation | `partial` summary; individual receipts retain their outcomes | 21 |
| Timeout/cancellation after dispatch, transport failure, lost/malformed/oversized response, arbitrary domain error, batch/async response needing domain interpretation | `unknown` | 22 |
| Backend response observed, final local save fails | in-memory remote evidence plus `persistence_failed:true`; durable record may remain unknown | 22 |

A successful API response acknowledges that request; it does not verify an
entire multi-step workflow or external downstream effects. Multi-request
invocations expose every observed request outcome. Batch envelope success
cannot prove all individual items applied, and a domain failure can follow
partial effects. Those cases stay unknown until a domain owner implements
endpoint-specific interpretation and read-only reconciliation. Generic 204 or
non-JSON responses also stay unknown; HTTP success alone is insufficient proof.

No generic reconcile/retry command is advertised: without saved target IDs and
backend-specific semantics it could not establish non-application or safe retry.
A persisted `dispatch_started` receipt after a crash means unknown, including
when the process may have stopped before dispatch. Never infer failure from it.

Embedded hosts receive a typed `api/mutation_outcome` error with the same
20/21/22 exit mapping when evidence is non-successful, even if the command
returned an API success envelope. The original typed cause and its scope/code
diagnostics remain available; its retry guidance does not authorize replay.

## Known endpoint evidence

The shared classifier recognizes the following established response shapes in
addition to generic code/status evidence. Its exact method/path matching does
not generalize field names to unrelated endpoints.

- Docs native create/update `result:failed` and all history/revert acknowledgements
  stay unknown; there is no verified completed status for the history operation.
- IM urgency, membership, settings, and join-request handling retain unknown for
  rejected, nonexistent, pending-approval, or failed recipients. Empty failure
  arrays preserve the normal acknowledgement semantics. Missing/null outcome
  arrays do not prove success; explicit join-request result lists remain unknown
  until request/result coverage is reconciled.
- Minutes word replacement requires positive numeric counts covering exactly the
  requested source words. Missing, zero, duplicate, or unmatched counts stay unknown.
- Base v3 create/update acknowledgements with ignored fields remain unknown.
  Batch-create requires one unique nonempty returned record ID per input
  row and no ignored fields or unrecognized response extensions. Missing proof
  stays unknown. The SDK request body's bounded `GetBody` copy supplies counts;
  unclonable/oversized bodies stay unknown and are never consumed or persisted.
- Apps SQL string results must decode to nonempty statement results without an
  ERROR entry. Mixed or malformed results stay unknown; no SQL text is parsed.
- VC SELECTED invitations require exact requested-recipient identity/count
  coverage with successful statuses and no failed/truncated results. Missing or
  mismatched targets stay unknown. ALL_SUGGESTED has no explicit target set for
  this comparison and remains unknown until full selection coverage is verified.
- Wiki move/delete task acknowledgements stay unknown even if a separate status
  read later succeeds; generic receipt reconciliation is not implemented.
- Slides native replacement's failed-part index/reason stays unknown; advisory
  issues alone do not turn a successful response into a failure. Slides history
  reverts and Drive import task acknowledgements also remain unknown.
- Sheets invoke-write decodes its nested output only at the known RPC endpoint.
  Malformed/empty output or nested failures remain unknown. Batch-update tool
  responses and history-revert acknowledgements remain unknown until complete
  per-operation reconciliation
  exists; other nonempty decoded responses retain acknowledgement semantics.

These are request evidence rules backed by existing domain contracts and
synthetic regression tests, not live backend or conditional-write verification.

Mindnote node creation requires one unique nonempty created-node ID per submitted
new node. Missing, short or malformed ID evidence remains unknown. The same
endpoint supports updates, but its frozen response schema only defines created
IDs, so requests containing existing `node_id` values remain unknown even when
response IDs match. No update coverage or backend atomicity is inferred.

Native Base batch creation also checks IDs already confirmed in the same
invocation, scoped to the effective account/identity and exact host/table path.
An overlapping later response becomes durably unknown; earlier confirmed
batches stay applied. This synchronized memory holds at most 10,000 IDs and
4 MiB of ID/scope strings, matching the import's row ceiling. Exhausted evidence
capacity remains unknown instead of evicting earlier IDs. These IDs and paths
are never persisted in receipts; this does not detect reuse across invocations.

## Classification and storage boundary

GET, HEAD, OPTIONS, credential acquisition endpoints, and commands explicitly
marked read-only bypass the journal before credential resolution. This includes
the unified POST `/oauth/v3/token` endpoint on the two trusted Accounts HTTPS
origins; the same path on other origins remains subject to mutation evidence.
Token minting does not create a user-content receipt before a subsequent write.
This also includes
known POST reads on generated and shortcut commands. A write workflow can
mark an individual domain-known read with
`common.WithReadOnlyRequest(ctx, method, apiPath)` or
`runtime.CallReadOnlyAPITyped(method, apiPath, params, data)`. The marker matches
only the exact method and URL path; other requests retain normal journaling.
Generated Attendance `user_tasks query` and Approval `approvals search` declare
their exact POST reads this way: their scopes and endpoint schemas establish
query-only semantics despite the frozen catalog's write risk. In agent mode,
their command risk annotations are read, enabling the verified read-artifact
pipeline. The catalog and legacy command annotations stay unchanged; partial
Attendance employee coverage fields remain native read data.
It preserves cancellation and command risk, and must never be derived from
arbitrary raw API input or request content. Conversely,
`common.WithMutationRequest(ctx, method, apiPath)` identifies an exact known
lifecycle mutation inside a read workflow, such as subscription registration.
Credential acquisition stays excluded; this never changes the command risk. Raw API POST/custom methods
are conservatively potential mutations; an unclassified raw read may receive a
receipt. Auth/config lifecycle effects are preserved from the reference and are
not content mutation receipts. This is a documented foundation ceiling, not
proof that all domain mutations or raw endpoint semantics have been verified.

Known presigned object uploads can use
`common.WithStoragePutRequest(ctx, apiPath)` with the invocation's
`Factory.ExternalHTTPClient`. This declaration matches only PUT and the exact
decoded URL path, including inside read workflows. The storage protocol's final
HTTP 200, 201 or 204 acknowledgement confirms the object write without requiring
an OpenAPI JSON envelope. HTTP 202, server failures and lost responses remain
unknown; the declaration never adds retries. Upload bytes, signed URL queries,
headers and paths are not persisted. Later callback or publication requests keep
their own receipts, so a stored object does not establish successful publication.

SDK and streaming API calls bind receipts to the identity selected for that
request's access token. An explicit bot call inside a user workflow records bot
identity without changing the invocation default or other concurrent requests.
External requests without API identity metadata retain the invocation binding.

CLI-owned state lives at `<effective-config-dir>/next/receipts`. Directories use
0700 and files 0600; Unix refuses permissive state and symlinks at the receipt
storage boundary. Writes sync a temporary file, atomically commit it, and sync
the directory. Per-receipt process locks cover dispatch and final save; cleanup
cannot remove active evidence. Windows retains atomic replacement and file
sync, but directory fsync and Unix permission enforcement are unavailable.
Local locks do not prevent other clients from editing remote resources. Backend
conditional writes, no-op detection, recipient validation, batch interpretation,
and read-only reconciliation remain domain responsibilities and require their
own contract tests; this journal does not solve those guarantees.

Validation: unit fault injection covers pre-dispatch save failure, durable
pre-write evidence, exact body/error preservation, lost responses, cancellation,
post-success save failure, active cleanup contention, corruption, retention,
partial summaries, and conservative batch classification. Process tests live in
`tests/next_foundation/receipts_e2e_test.go` and run with the completed foundation
suite through `python3 scripts/next_test.py --section foundation --suite all`. Synthetic offline tests
are not evidence of live backend guarantees.
