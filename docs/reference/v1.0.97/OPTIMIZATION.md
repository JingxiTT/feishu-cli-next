# Upstream v1.0.97 comparison and optimization

Compared on 2026-10-11: original official `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`
→ official v1.0.97 `72579c80027c863ca51d5f9affda72a70ab0d8a6` (53 commits).
The fork comparison is pre-update `c0dccfd9` → synced `e79ea062` → this optimization.
The sync already preserves the fork's runtime, compact output, projections,
artifacts, receipts, discovery and request-only schema controls.

## What changed upstream

Official command nodes increased **938 → 941**, shortcuts **535 → 538**;
**251 generated methods and 25 event keys** remain. There are **3 added commands,
no removed commands and 92 existing command registration changes** in the frozen
Feishu snapshots. Registration counts do not include every implementation change.
[Complete registration delta](../../../evaluation/evidence/upstream-v1.0.97-optimization/command-delta.json)
records every changed command; the [sync baseline](README.md) describes both brands.

| Area | New or changed behavior | Next integration |
| --- | --- | --- |
| Apps | Source ZIP `+export`; release reason/approval handoff; DB subcode and file errors | Export already streams through `RuntimeContext`/`FileIO` as a read. Shared projection/artifact summaries now preserve the whole `current_node_info`, including approval URL, even with dotted selection. |
| Mail | `+thread-modify`, `+thread-trash`, sequential batches of 20 | Existing runtime, permissions, confirmation and compact output apply. Batch execution order stays unchanged. |
| IM | `--concise` chat/thread Markdown; file/stdin send/reply/edit inputs; calendar open IDs; bot-only chat creation | Shared preflight rejects incompatible result controls before API calls. JSON dry-run previews remain compatible. Valid JSON avoids a payload copy; concise output avoids a second full buffer. |
| Docs | Async create, draft-resource handling, explicit constraints | Receipt core tracks pending creation and reconciles only exact native completion observed in the same invocation, origin and identity. It reuses existing polling and adds no requests/retries. |
| Base | Full table/field listing; workspace limit 30; NPS/form ordering; unsupported export values | Existing typed field/export and projection paths retain values. No new parallel pagination or lossy field conversion. |
| Wiki / Drive | `node_by_token` lookups; better download/preview token recognition; folder move without task ID | Synced consumers reuse owning lookup/runtime code, including Docs, Base, Sheets and Slides. |
| Sheets / Slides | Accepted payload/flag spellings, chart sizing checks, XML schema/guidance | Existing read/write classification, request-only schema, output controls and literal XML support remain in force. |
| Calendar / VC / Minutes | Calendar response changes, bot-only meeting join/leave, deterministic search | Synced domain behavior and identity constraints retained. No invented result defaults or changed ordering. |
| Shared runtime | Lazy catalog and target-aware assembly; scope recommendations; credential/config/error handling; policy messages | Reuse upstream catalog loader. Discovery still requests the full tree; precise commands/schema load only relevant metadata. |

## Changes made here

- Keep release approval context in shared result projections and artifact stdout.
- Reject IM Markdown/result-control conflicts before command execution, while
  retaining the upstream concise and JSON dry-run contracts.
- Reconcile async Docs creation in the receipt core, using existing durable
  receipt update and bounded response replay. Processing, failure, missing or
  conflicting proof stays unknown. Response bytes and errors are preserved.
- Validate successful IM JSON without copying it into `json.RawMessage`.
  Invalid input retains its typed error, syntax offset and original cause.
- Write the existing concise message builder directly and forward the optional
  `io.StringWriter` capability through our agent output wrapper. This avoids a
  replacement byte copy while retaining safety checks and write-error handling.

No new dependencies, public SDK interfaces, configuration modes, commands or
extra API requests were introduced by this optimization.

## Local measurements

[Raw process samples](../../../evaluation/evidence/upstream-v1.0.97-optimization/probes.json)
use 31 fresh processes per variant, one warm-up, rotating arm order, isolated
empty configuration and blocked network access. Token counts use `o200k_base`
with tiktoken 0.12.0 and count stdout/stderr once. These are output-context tokens,
not provider usage or whole-task savings. Builds use the committed catalog and
Go trimpath/stripped binaries; binary hashes are recorded. These process probes
precede the final duplicate-task guard and string-writer forwarding, which do not
change the measured schema contracts. Host load is uncontrolled.

| Probe | Pre-update Next | Synced Next | Optimized Next |
| --- | ---: | ---: | ---: |
| Version | 28.00 ms | 12.08 ms | 11.91 ms |
| `schema calendar.events.get` | 28.49 ms | 14.10 ms | 13.97 ms |
| `schema drive.files.list` | 27.96 ms | 12.87 ms | 13.09 ms |
| `docs +create --help` | 28.15 ms | 12.15 ms | 12.09 ms |
| `im +chat-messages-list --help` | 28.13 ms | 14.54 ms | 14.30 ms |
| `discover '*' --limit 3` | 29.19 ms | 23.81 ms | 23.93 ms |

The startup improvement primarily comes from the upstream lazy loader already
present in the sync. The optimization is approximately equal to synced Next
for these local probes; it does not establish a speed lead over current official.
Pre-update schemas differ, so their timing is a version comparison, not an
isolated same-contract benchmark.

Across six schema probes, official v1.0.97 outputs total **15,167 tokens**,
Next full output **10,540 (30.5% fewer)** and Next `--input-only` **2,544
(83.2% fewer)**. The probe asserts identical request, permission and guidance
contracts across official, synced Next and optimized Next; only response schema
is omitted by `--input-only`. Synced and optimized full schemas are byte-identical.
Use full schemas when response fields are needed. New Apps/Mail command help
contains 51 more tokens than official because it advertises Next result controls;
there is no claim of smaller output for every command.

Five local microbenchmark samples per arm compare the synced implementation with
these optimizations. Concise benchmarks disable the safety provider for both arms
and use the same synthetic fixture with the actual `WithAgentOptions` stdout
wrapper; production safety checks remain enabled.

| Operation | Median before → after | Allocation before → after |
| --- | ---: | ---: |
| Validate 1 KiB valid IM JSON | 2,759 → 1,503 ns | 2,480 → 0 B; 5 → 0 allocations |
| Validate 64 KiB valid IM JSON | 174,437 → 92,275 ns | 147,632 → 0 B; 5 → 0 allocations |
| Render 50 concise messages | Not compared | 340,120 → 282,650 B; 1,035 → 1,033 allocations |
| Render 200 concise messages | Not compared | 1,424,760 → 1,203,327 B; 4,043 → 4,040 allocations |

Concurrent test load changed during the concise runs, so their timing samples
are not used for a speed claim. The 15.5–16.9% allocation reduction remains
measured with the production output wrapper. These microbenchmarks exclude Feishu network time.
Raw [JSON before](../../../evaluation/evidence/upstream-v1.0.97-optimization/content-before.txt),
[JSON after](../../../evaluation/evidence/upstream-v1.0.97-optimization/content-after.txt),
[concise before](../../../evaluation/evidence/upstream-v1.0.97-optimization/concise-synced-benchmark.txt)
and [concise after](../../../evaluation/evidence/upstream-v1.0.97-optimization/concise-candidate-benchmark.txt)
retain every sample. Reproduce with:

```sh
go test ./shortcuts/im -run '^$' \
  -bench 'Benchmark(ValidateMessageContentJSON|OutputMessagesConcise)' -benchmem -count=5
```

## Verification and limits

[Validation record](../../../evaluation/evidence/upstream-v1.0.97-optimization/validation.json)
records completed checks. Regression tests reproduced the original concise
preflight, release projection and premature Docs completion failures before fixes.
Transport fixtures cover exact identity/origin/task binding, synchronous/async
success, failures, redirects, response replay, persistence failure and bounded
tracking. The official capability gate checks the complete v1.0.97 surface.

Live tenant verification and provider-token task trials are not part of these
measurements. A live Docs fixture must create a private run-owned document through
an account where async creation occurs, verify it and delete it even after failure;
an approval handoff additionally needs a controlled app release workflow. No such
fixtures were selected for this change. Deterministic transport/process fixtures
provide substitute evidence, not live-service parity.

Mail batch endpoints still lack a verified per-item receipt completion classifier;
the shared receipt core conservatively reports unknown even when the shortcut's
API summary is successful. Reconcile before replaying a batch. Relaxing that
behavior requires authoritative endpoint completion evidence. Historical token
results remain unchanged and do not establish savings for these new workflows.

## Follow-up against merged Next `e6b64943`

The release comparison above and its 3 new / 92 changed registrations remain
the baseline. This follow-up builds on the already merged optimization; it does
not attribute that work or upstream's startup gains to these changes.

- `internal/meta` now validates numeric bounds once and keeps the exact JSON
  number, removing the redundant copy and decoder from both help and schema.
  Negative zero, large integers, exponents and JSON whitespace retain their
  original semantics; invalid and nonnumeric values remain rejected.
- `internal/output` skips object/array decoding for scalar failure scans. Mail
  `failed_thread_ids` and Apps `error_logs` with failure evidence preserve the
  entire result under dotted projection and artifact summaries, including
  embedded emitters without mutation receipts.
- The entry guide routes the three new shortcuts directly, recommends batching
  known Mail IDs, and covers async Docs recovery, IM concise/result controls and
  narrow Sheets input schemas. Command-specific export, approval and retry
  details are loaded through affordances and existing references. Reaction
  enrichment is skipped only when the task does not need it.

Apps ZIP streaming and Mail's sequential batches already use the shared runtime
and FileIO. Their requests, batch order, identities and receipt classification
are unchanged. No new dependency, command, flag or API call was added.

### Follow-up measurements

Five sequential samples per arm on Darwin/arm64, Apple M5. The numeric-bound
and complete-renderer benchmarks use `-benchtime=200ms -count=5`; the failure
scanner uses `-count=5`. Baseline overlays use `e6b64943` implementation files
with the same benchmark fixtures. These are local microbenchmarks, not network
or whole-command latency measurements; host load remains uncontrolled.

| Operation | Median before → after | Bytes/op before → after | Allocations/op |
| --- | ---: | ---: | ---: |
| Parse bound `100` | 393.7 → 28.82 ns | 2,656 → 16 | 12 → 1 |
| Parse bound `9223372036854775807` | 425.2 → 48.01 ns | 2,704 → 16 | 12 → 1 |
| Parse bound `1.25e+10` | 402.4 → 35.46 ns | 2,656 → 16 | 12 → 1 |
| Scan 20 messages for unsafe projection | 102.81 → 57.72 µs | 107,976 → 34,576 | 1,638 → 700 |
| Scan 200 messages for unsafe projection | 1,158.34 → 535.14 µs | 1,048,778 → 325,938 | 15,861 → 6,643 |
| Render 1,000 records with field selection | 49.84 → 33.58 ms | 18,713,036 → 16,311,802 | 119,178 → 89,160 |

Raw [bounds before](../../../evaluation/evidence/upstream-v1.0.97-followup/bounds-before-clean.txt)
/ [after](../../../evaluation/evidence/upstream-v1.0.97-followup/bounds-after-clean.txt),
[scan before](../../../evaluation/evidence/upstream-v1.0.97-followup/output-scan-before.txt)
/ [after](../../../evaluation/evidence/upstream-v1.0.97-followup/output-scan-after.txt),
and [renderer before](../../../evaluation/evidence/upstream-v1.0.97-followup/output-render-before.txt)
/ [after](../../../evaluation/evidence/upstream-v1.0.97-followup/output-render-after.txt)
retain all samples. Reproduce with `BenchmarkFieldBounds` in `./internal/meta`,
and `BenchmarkAgentProjectionFailureScan` / `BenchmarkAgentJSON/Fields` in
`./internal/output`, using `-run '^$' -benchmem` and the options above.

The [context probes](../../../evaluation/evidence/upstream-v1.0.97-followup/context.json)
verify that six full schemas remain byte-identical to merged Next and that
`--input-only` removes only `outputSchema`: 10,540 → 2,544 context tokens
(75.9% fewer). This saving already existed before this follow-up. The new entry
routes cost 235 extra tokens (884 → 1,119); new command help also grows to carry
decision and retry guidance. Avoided discovery/enrichment calls are guidance,
not measured whole-task token savings. Counts use `o200k_base`, tiktoken 0.12.0.

Regression tests fail against the previous implementation for lost failure
context and unnecessary allocations. The follow-up
[validation record](../../../evaluation/evidence/upstream-v1.0.97-followup/validation.json)
records final checks and remaining verification limits.
