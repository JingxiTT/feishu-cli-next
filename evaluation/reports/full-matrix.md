# Historical full matrix

This report reproduces the frozen matrix through `run_matrix.py`. It is
historical evidence, not the current evaluation entry point; use
`evaluation/pipeline.py` for new comparisons.

Run all **43 implemented development tasks**, **258 logical attempts**: original/full versus Next, three repetitions per task and arm. Suites are paired functions (8), harder Docs (3), domain reads (12), workflow scenarios (9), scaling (6), business (2), foundation (2), and Contact (1). Together they cover the 23 inventory sections. The development/held-out inventory files describe coverage; they are not extra executable trials. Held-out release acceptance and live tenant parity remain separate.

## Run and reuse

Create an ignored local JSON configuration with absolute paths for `official_source`, `official_cli`, `next_source`, `next_cli`, `codex`, and `auth_home`. Use frozen source archives and their matching binaries. The candidate binary must include the synthetic auth-sidecar build used by the existing harness. Keep actual authentication in the existing protected Codex home; never copy credentials into the JSON or commit the local configuration.

```sh
# Plan, validate pinned inputs, build isolated extension hosts, and render results.
python3 evaluation/run_matrix.py --config /absolute/local-config.json \
  --output evidence/evaluation/full-matrix-20260911

# Execute missing slots or resume a safely checkpointed interruption.
python3 evaluation/run_matrix.py --config /absolute/local-config.json \
  --output evidence/evaluation/full-matrix-20260911 --execute
```

The same command on unchanged completed inputs makes **zero model calls**, including when previous results failed, were inconclusive, or had incomplete accounting. A new output directory also discovers matching sealed, complete jobs in sibling directories under the same parent. For example, dated outputs under `evidence/evaluation/` can reuse each other's exact matching jobs. This is **same-parent reuse**, not a machine-wide cache. Environment and job fingerprints must match; reuse copies the complete job without selecting favorable rows. Every imported token is counted as reused rather than fresh spend. Jobs already complete in the same output retain their existing fresh/reused accounting.

Sibling reuse preserves exact measured extension-host binaries without rebuilding, historical provenance, qualifications and diagnostic attachments. Locks coordinate the parent cache and individual outputs. A partial/in-flight match, a busy source/cache lock, changed seal, conflicting completed bytes or conflicting qualification requires reconciliation before dispatch; these states do not authorize a replacement trial. Lock files and STOP markers are not imported.

The command regenerates the report from retained evidence. It never repeats a trial to seek a passing outcome. Inputs changing in an existing output directory fail closed; select a new directory deliberately after reviewing the code change. A report-only edit is not a reason to rebuild the candidate binary or rerun the evaluation. Existing frozen candidate `cad221e` remains the product under evaluation for this matrix.

Historical condition 09 can be imported once with `--reuse-condition evidence/evaluation/token-efficiency-09 --reuse-evaluator /absolute/condition09-frozen-evaluator --reuse-review /absolute/migration-review.json`. Import requires its accepted independent review, unchanged evidence hashes, matching binaries/full guidance/Codex/settings, identical measured trial/fixture/scoring code, and the exact complete schedule. It retains all 18 attempts, including the inconclusive editing result. The imported 901,686 tokens are historical usage, not new spend. The other **240 attempts are new**. This is explicitly a matrix with historical reuse, not a fresh 258-trial simultaneous experiment.

Python is part of the cache identity: both the resolved executable hash and version must match. To reuse the retained September 11 output, invoke `/opt/homebrew/opt/python@3.14/bin/python3.14` in place of `python3`; `environment.json` records the exact interpreter version/hash. Generic `python3` is suitable for a new setup, but a different interpreter requires a distinct measured environment.

## Execution contract

- Native Codex CLI **0.153.4**, **GPT-5.6 Luna / medium**, symmetric filesystem-only full selected guidance, fresh agent context and fixture per attempt, 180-second agent and 210-second outer budgets.
- Three independent section workers; each section retains the existing serial, seeded balanced arm/trial schedule. Ports, workspaces and fixtures are isolated. Account limits and host resources are shared, so latency is not compared with historical serial runs.
- Every section continues after task failures and missed efficiency targets. No Docs-first performance stop gate prevents full coverage.
- Content fingerprints bind CLI and extension-host binaries, complete frozen source and guidance, fixture/runtime resources, selected task manifest, Python/Codex, launcher semantics, seeds and budgets. Temporary paths and report prose are not input identities. The provider alias does not identify an immutable model snapshot; historical reuse is noncontemporaneous.
- Output locks prevent concurrent execution against the same evidence directory. Checkpoints are atomic and durable. Before a model call, a durable in-flight marker is written. A started call without a recorded result **requires reconciliation and is never automatically replayed**. Completed failed/inconclusive/unknown-usage rows are retained on resume; a changed resume contract is refused.
- Create `STOP` in the matrix output directory to stop between complete attempts. Remove it only when intentionally continuing the same fixed run. Execution-integrity errors also stop new starts; task failures do not.

## Measurement and report

`environment.json` records content identities; `plan.json` enumerates every task/section and expected arm binary/guide hashes. Each job retains its checkpoint, execution-code hashes, and immutable completed-batch checksum. Historical jobs additionally retain source/review provenance. `summary.json` and `REPORT.md` are regenerated automatically from the checkpoints, including partial progress.

For each task, tokens per verified success include every observed attempt, failure, recovery, and output token. Cached input is counted once in full. Unknown accounting, inconclusive outcomes, or zero verified successes make the affected primary score unavailable. Known partial cost remains a lower bound, never zero. Docs require **strictly more than 50%** savings; other tasks require **at least 30%**, with observed success nonregression. Three repetitions do not establish statistical success parity or population savings. A pooled value, when available, describes only this fixed task mixture and cannot replace individual task gates.

Publish only public JSON evidence, reports and review artifacts. Keep local configuration, host binaries, lock files and stop symlinks out of Git. Retain private preparation logs locally.
