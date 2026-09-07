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

## Classification and storage boundary

GET, HEAD, OPTIONS, credential acquisition endpoints, and commands explicitly
marked read-only bypass the journal before credential resolution. This includes
known POST reads on generated and shortcut commands. Raw API POST/custom methods
are conservatively potential mutations; an unclassified raw read may receive a
receipt. Auth/config lifecycle effects are preserved from the reference and are
not content mutation receipts. This is a documented foundation ceiling, not
proof that all domain mutations or raw endpoint semantics have been verified.

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
