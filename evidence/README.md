# Retained evidence

Records of what was measured, reviewed, or reconciled at a specific revision.
Every file here except this index and `MANIFEST.sha256` is sealed: it is never
edited, renamed, or deleted. A correction is a new file that names the record
it supersedes.

| Directory | Holds |
|-----------|-------|
| `evaluation/` | Agent-comparison runs, controlled fixtures, and their reports |
| `sections/` | Per-section offline and live evidence, cited from [docs/sections/](../docs/sections/OWNERSHIP.md) |
| `reference/` | Phase ledgers, reconciliation reports, capability checkpoints, and `reviews/` |
| `integration/` | Cross-domain binding evidence |

The baselines the capability gate compares against are not evidence; they live
in [baseline/](../baseline/README.md).

## Adding a record

Write the new files, then seal them:

```sh
python3 scripts/next_layout.py seal
```

`seal` appends each new file's SHA-256 to `MANIFEST.sha256` and never rewrites
an existing line. `python3 scripts/next_layout.py check` (part of
`make script-test`) fails when a sealed file changes or disappears, or when an
unsealed file is present.

## Paths inside records

A record cites repository paths as they were at the revision it measured, so
check those paths out at that revision rather than at `HEAD`. Records sealed
before this directory existed use the earlier locations:

| Path in a record | Current location |
|------------------|------------------|
| `evaluation/evidence/…` | `evidence/evaluation/…` |
| `docs/sections/<name>.json` | `evidence/sections/<name>.json` |
| `docs/integration/sheets-base-binding-evidence.json` | `evidence/integration/sheets-base-binding-evidence.json` |
| `docs/reference/reviews/…`, `docs/reference/p0-*` … `p3-*`, `ci-capability-*`, `P1.md`, `P3.md` | `evidence/reference/…` |
| `docs/reference/…` (everything else) | `baseline/…` |
| `evaluation/token-efficiency-NN.md`, `ASSESSMENT.md`, `full-matrix.md`, `paired-functions-luna-medium.md` | `evaluation/reports/…` |

Twelve sealed reports contain relative links written for those earlier
locations; resolve them through this table.
