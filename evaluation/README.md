# Agent comparison evaluation

Use [pipeline.py](pipeline.py) for new official-versus-Next comparisons against
real Feishu. [PIPELINE.md](PIPELINE.md) documents configuration, the live fixture
lifecycle, provider accounting and safe resume. Start from
[pipeline.example.json](pipeline.example.json).

The [full evaluation report](reports/evaluation-overview-20261029.md) consolidates
all current live Luna runs, per-workflow results, coverage gaps, failures and
accounting qualifications, with a separate index of historical evaluations and
offline CLI probes.

The fake Feishu endpoint, API simulators, canned response tables, synthetic task
sets and legacy launchers have been removed. There is no synthetic evaluation
backend or historical simulator launcher in this checkout. Ordinary offline
unit tests mock model processes and lifecycle messages; they do not implement
Feishu APIs or produce model-efficiency evidence.

The bundled [live Contact task](tasks.live-contact.json) reads the current
account and tenant. The [business workflows](LIVE_BUSINESS.md) add native rich
document editing, typed spreadsheet reconciliation, paginated Base intake
and a report spanning Docs, Sheets and Base. The extended suite adds six cases,
including 2,505-record and 500-block audits and a settlement updating all three
domains. Three complex cases add versioned FX journals, conflict-aware Base
adjustments, and bulk document closeout with native ID preservation. They use fresh
`[TEST]` resources in the authenticated account, with independent native
readback and confirmed cleanup. The [collaboration workflows](LIVE_COLLABORATION.md)
add document creation, image/file attachments, Mermaid and SVG drawings.
IM/group-chat evaluation code and cases were removed under the user's permission
boundary. Other suites need independently verified live fixtures.
The [complex Luna run](reports/live-complex-luna-20261029.md) records the new
business sample, original verifier results, native formatting reassessment,
blocked IM setup, cleanup and all provider costs separately.
`pipeline.py --list-sections` lists the bundled live task without a config, or
actual configured live tasks with `--config`. Missing live fixtures are blocked,
never replaced with fake responses.

| Role | Modules |
|---|---|
| Entry point | `pipeline.py`; internal batch adapter `live_batch.py` |
| Model adapter and accounting | `codex_runner.py`, `harness.py` |
| Live setup, verification and cleanup | `live_fixture.py`, `live_contact.py` |
| Business and collaboration lifecycle | `live_workspace.py`, `live_collaboration.py`, `live_domains/` |
| Local probes and historical analysis | `measure_cli.py`, `measure_surface.py`, `function_coverage.py`, `plot_scaling.py` |

Do not read or use `tasks.held-out.json` during development.

CI checks software correctness using `make evaluation-tooling-test`
(`make evaluation-test` is its compatibility alias). These tests neither bill a
model nor contact a tenant. Actual agent evaluation is an explicitly launched
local experiment and refuses CI. See [CI and evaluation](PIPELINE.md#evaluation-and-ci).

## Offline CLI probes

The [complete October 7 surface comparison](../evidence/evaluation/cli-surface-20261007/REPORT.md)
records 1,738 local surfaces: all original commands, methods, event keys and
embedded guidance files. Every request schema is smaller (61.9% in the sum), but
936/938 help outputs are larger and current upstream is faster. The function
ledger exposes 866 original nodes without observed non-help calls in retained
workflow traces. A lead across every area/function remains unestablished.

[`measure_surface.py`](measure_surface.py) derives its complete schedule from
the frozen catalog and configured-brand inventory: every command node (including
hidden commands and inherited command aliases), all 251 methods, every event
key, and every file discoverable in the original binary's embedded skills.
It compares each surface separately rather than using a domain average. Failed,
unstable or unavailable probes have no efficiency score; a brand guard that
bypasses help is recorded without dispatching the command. Extensions and
packaging are explicitly unmeasured by local introspection.

```sh
python3 evaluation/measure_surface.py \
  --reference-cli /absolute/frozen-official-cli \
  --before-cli /absolute/next-before \
  --next-cli /absolute/next-after \
  --current-cli /absolute/current-official-cli \
  --brand feishu --repetitions 3 --encoding o200k_base \
  --output /absolute/new-surface-probes.json.gz
```

The optional tokenizer and isolated process protocol match `measure_cli.py`
below. The runner refuses to overwrite retained evidence. It checks unchanged
default schema bytes, request/permission/guidance parity for every method, and
semantic parity of event definitions. Current upstream differences are retained
separately and never treated as equivalent work. Three timings describe this
machine's observations; they cannot establish a universal speed advantage.
Help/schema/file coverage is **not** execution coverage: mutations, pagination,
recovery, streams, binary transfers, permission conditions, host SDKs and platform
builds still require their own correctness and task-efficiency evidence. No
whole-CLI savings claim follows from a pooled local-output reduction.

[`function_coverage.py`](function_coverage.py) joins this exhaustive surface
report to the original command inventory and the retained workflow traces. It
verifies checkpoint hashes, reconstructs task gates using all retained attempts,
excludes help calls from execution observations, and keeps historical binary
identities. Each original node is present even when no task called it. The SDK
symbol index and platform build conditions remain explicit unverified surfaces.
A verified workflow that calls a command does not establish every mode of that
function or attribute the entire workflow saving to it.

```sh
python3 evaluation/function_coverage.py \
  --probes /absolute/new-surface-probes.json.gz \
  --matrix evidence/evaluation/full-matrix-20260911 \
  --output /absolute/function-coverage.json.gz
```

The [October 7 comparison](../evidence/evaluation/cli-efficiency-20261007/REPORT.md) records
the request-only schema and discovery changes, both local measurement conditions,
and the remaining startup gap against current upstream.

[`measure_cli.py`](measure_cli.py) compares process latency and output context
size for six generated methods, shared guidance and bounded discovery. Supply
canonical builds of the [frozen official reference](../baseline/README.md),
Next before/after the change, and optionally a pinned current upstream build:

```sh
python3 evaluation/measure_cli.py \
  --reference-cli /absolute/frozen-official-cli \
  --before-cli /absolute/next-before \
  --next-cli /absolute/next-after \
  --current-cli /absolute/current-official-cli \
  --repetitions 31 --output /absolute/local-probes.json
```

Optional `--encoding o200k_base` requires `tiktoken` in the selected Python
environment; without it token counts remain unavailable. Counts cover stdout
and stderr once, not provider usage or tokens per completed agent task.
Binary/filesystem caches are warmed; each timed invocation is a fresh process
in rotating order with isolated configuration, remote metadata and notices
disabled. Any failed or unstable probe stops reporting.
The script requires exact request/permission/guidance parity against the frozen
reference, unchanged default schema bytes and unchanged discovery bytes. The
current upstream arm records parity separately because its catalog may differ.
Binary hashes, per-process durations and output hashes are retained.

These probes complement live agent evaluation; they cannot establish
live API speed, successful task parity or the full-task token-efficiency gates.

## Historical records

Retained [evaluation evidence](../evidence/evaluation/), the
[assessment](reports/ASSESSMENT.md), and the
[full-matrix write-up](reports/full-matrix.md) describe their recorded revisions.
Synthetic results remain labeled synthetic and do not establish live parity.
They have not been deleted or rescored. Commands and source paths in old reports
belong to those revisions; use their recorded Git revisions when inspecting old
implementations. The retired simulators cannot run from this checkout.

`pipeline.py --output /absolute/existing-run --report-only` can still render
retained pipeline results without a model or tenant call. `function_coverage.py`
can inspect sealed historical traces without the old simulation runtime.
