# Local CLI efficiency comparison — 2026-10-07

Implemented code: `2a7e48e` (initial change `456a666`, then two discovery allocation/path refinements). Next-before is `d383eb1`. The frozen official reference is [`7fd6ef3`](https://github.com/larksuite/cli/tree/7fd6ef3c07182257ce776cdc5a614e122d5bd4b3); the current upstream source was pinned to [`7beffb0`](https://github.com/larksuite/cli/tree/7beffb086d7fa3c5b843d8affa7c089f49cfc65e), package 1.0.97. These are source builds, not npm-wrapper timings.

The request-planning probes use **81.4% fewer output context tokens in total** than the frozen official CLI, with exact request/permission/guidance parity. This is the sum of six fixed schema outputs, **not agent tokens per task**. Broad discovery is **21.0% faster** than Next-before; the two keyword probes are effectively unchanged in these observations. **There is no whole-product speed/token lead claim:** current upstream startup remains faster, its schemas differ, and the full provider-token matrix was not rerun.

## Output context size

`o200k_base`, tiktoken 0.12.0; stdout and stderr each counted once. Next-input uses `schema <method> --input-only`. All figures are observed local context sizes, with no model calls or provider accounting.

| Method | Frozen official | Next full | Next input-only | Reduction vs frozen |
| --- | ---: | ---: | ---: | ---: |
| `calendar.events.get` | 5,291 | 3,849 | 632 | 88.1% |
| `im.messages.patch` | 818 | 683 | 671 | 18.0% |
| `drive.files.list` | 1,525 | 1,058 | 604 | 60.4% |
| `sheets.spreadsheets.get` | 783 | 574 | 394 | 49.7% |
| `task.tasks.get` | 4,670 | 3,246 | 226 | 95.2% |
| `mail.user_mailbox.messages.get` | 2,830 | 1,876 | 433 | 84.7% |
| **Sum of six outputs** | **15,917** | **11,286** | **2,960** | **81.4%** |

Response-heavy reads benefit most. Message patch has almost no response schema to remove, so it gains much less. Request-only output is an explicit planning view; use the default full schema when response definitions or a complete MCP tool spec are needed.

The shared skill grows from 1,244 to 1,285 context tokens to explain when to reuse already-read guidance. Its intended reduction in repeated help/reference calls has no fresh agent measurement. The entry guide now routes request planning to a specific method and discourages unbounded catalog dumps when a method is known.

## Fresh process latency

Each cell is a median of 31 subprocesses. Binary/filesystem caches are warmed once per variant; every timed invocation starts a fresh process. Arm order rotates. Configuration is empty and isolated; remote metadata and update/skill notices are disabled. No Feishu API request is made. Background host activity is uncontrolled; these are descriptive measurements, not a statistical performance guarantee.

| Probe | Frozen official (ms) | Next before (ms) | Next full (ms) | Next input-only (ms) | Current upstream (ms) |
| --- | ---: | ---: | ---: | ---: | ---: |
| `version` | 27.90 | 28.68 | 28.78 | — | 12.09 |
| `calendar.events.get` | 28.61 | 29.32 | 29.00 | 28.76 | 14.49 |
| `im.messages.patch` | 34.65 | 34.69 | 36.95 | 35.30 | 17.03 |
| `drive.files.list` | 30.28 | 30.40 | 30.58 | 30.64 | 14.01 |
| `sheets.spreadsheets.get` | 32.94 | 33.32 | 33.66 | 32.77 | 14.05 |
| `task.tasks.get` | 33.37 | 33.34 | 33.30 | 32.42 | 17.30 |
| `mail.user_mailbox.messages.get` | 35.85 | 33.79 | 33.17 | 33.10 | 16.54 |
| `shared-skill` | 33.24 | 33.71 | 32.71 | — | 13.53 |
| `discover-all` | — | 47.07 | 37.18 | — | — |
| `discover-docs` | — | 41.45 | 41.52 | — | — |
| `discover-contact-flags` | — | 41.70 | 41.75 | — | — |

Discovery rows compare identical commands and byte-identical outputs within Next. The upstream CLI has no `discover` command, so its cells are unavailable. The frozen reference and Next use the same catalog; default Next schema bytes remain unchanged for all 251 methods, and request-only output equals full output minus `outputSchema` across that entire catalog.

All six current-upstream schema probes have different request/permission/guidance contracts from the frozen reference. Their latency and context sizes are retained in raw evidence, but do not establish a same-contract advantage. Its [lazy catalog snapshot](https://github.com/larksuite/cli/blob/7beffb086d7fa3c5b843d8affa7c089f49cfc65e/internal/registry/snapshot.go) is relevant to further startup work; this probe does not isolate how much latency it explains. Next still eagerly parses/registers its catalog. Preserving remote overlays, plugin-visible command trees and exported host contracts must be part of any future loader change.

## Evidence and checks

[Final probes](probes.json) retain all 44 rows, binary/output hashes and all 1,364 timed subprocess samples. [Initial condition](probes-condition01.json) retains another 44 rows/1,364 samples at `456a666`; it showed 22.5% faster broad discovery and a 5.3% slower flag-keyword case. Subsequent code removes unmatched-term allocations and reuses traversal paths. The final run is a separate condition; cross-run differences do not isolate a causal effect. No provider tokens were spent by these probes.

[Validation provenance](validation.json) records builds, code/catalog/runner identities and checks. Focused/race tests, offline integration suites, examples, skills format, script/evaluator tests, vet, formatting, module tidy, lint, licenses and the post-code-commit quality gate passed. The full unit run failed only in the existing Drive listing fixture because this Mac uses `/var` temporary paths outside the FileIO allowlist; that package passed with `TMPDIR=/tmp`, without changing the allowlist or product code. Other unit packages passed. Final discovery refinements were checked again with focused race tests.

Fresh provider-token trials, live Feishu API trials, live skills-sync and sidecar-specific checks were not run. The changes affect local introspection/guidance, not sync, sidecar or workspace API behavior. The existing full-matrix results remain unchanged: 13 task gates met, 22 missed, 8 unavailable. These local probes do not replace those acceptance gates.
