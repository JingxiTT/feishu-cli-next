# Complete local CLI-surface comparison — 2026-10-07

**A lead across every original area and function is not established.** This run replaces the six-method sample with every original registered command, API method and event key, plus every original embedded guidance file. It exposes larger outputs and unmeasured functions individually.

Measured candidate product code: `ccdb0e6` (the prior schema/discovery work plus the shorter shared skill-read hint). Next-before: `d383eb1`. The coverage-audit implementation is `c442dd5`. Frozen official reference: [`7fd6ef3`](https://github.com/larksuite/cli/tree/7fd6ef3c07182257ce776cdc5a614e122d5bd4b3). Current official source: [`7beffb0`](https://github.com/larksuite/cli/tree/7beffb086d7fa3c5b843d8affa7c089f49cfc65e), package 1.0.97. These are canonical source builds, not npm-wrapper timings.

## Exhaustive local results

| Surface | Cases | Smaller / equal / larger | Original context tokens | Next context tokens | Sum reduction |
| --- | ---: | --- | ---: | ---: | ---: |
| API request schemas (`--input-only`) | 251 | 251 / 0 / 0 | 341,149 | 130,121 | 61.86% |
| Event schemas (`--json`) | 25 | 25 / 0 / 0 | 18,930 | 11,726 | 38.06% |
| Command help (root and hidden nodes included) | 938 | 2 / 0 / 936 | 263,570 | 313,876 | -19.09% |
| Inherited `slide` alias help | 30 | 0 / 0 / 30 | 8,104 | 9,712 | -19.84% |
| Embedded skill/reference files | 493 | 25 / 464 / 4 | 1,197,820 | 1,197,292 | 0.04% |

All **1,738** scheduled local surfaces were compared. The run retains **21,606 timed fresh subprocesses**, three per arm/case after one warm-up. There are **zero required probe failures and zero contract failures**. Counts include all **938 original command nodes / 535 shortcuts**, all **251 methods**, all **25 event keys**, **30 inherited alias spellings**, **493 guidance files** and version. The original catalog and current official catalog have the same 251 method names. This schedule does not enumerate hypothetical raw-API endpoints or every flag combination.

All 251 request schemas are smaller: reductions range from **16.3% to 95.2%**, with **61.9%** less context in the sum. The earlier six-method **81.4%** figure is a selected sample and must not be used as the whole-catalog result. Default full schemas save **30.1%** across the entire catalog. Their bytes remain identical to Next-before; request-only rendering removes only `outputSchema`, preserving all request requirements, permissions, risk and guidance. It is an explicit planning view; keep the full schema when response definitions are needed.

All 25 event-schema JSON definitions retain semantic equality and use **38.1%** fewer context tokens in total. These results do not measure event consumption, daemon reliability or recovery.

Help remains a broad overhead: **936/938** canonical help outputs are larger, **19.1% more context** in total; all 30 alias help outputs are larger as well. The two smaller outputs are root and `update` help, which have deliberate distribution differences. Existing agent output controls and the longer executable name contribute repeated help content; this run does not isolate their individual costs. Preserve metadata and safety when addressing it.

The shorter hint saves **30 context tokens on each of the 28 main-skill reads** versus Next-before, without changing the skill file bytes in this patch. The prior shared-guide change adds 41 tokens to its body. Across all 493 original guidance files, **25 are smaller, 464 equal and four larger** versus the frozen official CLI, saving only **528 tokens / 0.044%** in the sum. The four larger files are `lark-contact/SKILL.md`, `lark-contact/references/lark-contact-get-user.md`, `lark-doc/SKILL.md` and `lark-shared/SKILL.md`; their maintained routing/workflow requirements remain. This hypothetical one-read-per-file sum is not an agent task score or a recommendation to load every file.

## Local process speed

Medians below are the median of case medians within each surface, not task latency or a same-output comparison. They describe this machine only.

| Surface | Frozen original (ms) | Next full (ms) | Current official (ms) |
| --- | ---: | ---: | ---: |
| schema | 30.15 | 30.72 | 14.33 |
| event | 32.74 | 33.40 | 13.57 |
| help | 29.24 | 30.07 | 13.89 |
| guidance | 29.78 | 30.79 | 12.58 |

Next input-only schema case medians have a median of **30.29 ms**. **Current official remains faster** in the broad local comparison. Only **13/251** current official method envelopes have the exact same request/permission/guidance contract as the frozen reference; the other 238 differences are retained explicitly, so their sizes cannot establish an equivalent-contract advantage. One current-official guidance file is unavailable (`lark-sheets/references/lark-sheets-legacy-command-migration.md`); its failed process has no speed or token score.

CLI/framework caches are warmed, but every timing starts a fresh process. Variant order rotates; configuration/home are isolated, notices and remote metadata disabled, and proxies point to an unreachable loopback port. The brand is Feishu with a synthetic app account. No workspace API call or provider/model trial is performed. No broad Go checks ran concurrently with these measurements. Uncontrolled host activity and three samples prevent statistical or population speed claims.

## Function and workflow coverage

The [function ledger](function-coverage.json.gz) names every original command node, all event keys, the **301-name exported SDK symbol index**, and **61 platform build conditions**. The symbol index is an evidence index, not an exhaustive SDK signature or behavior test. It binds local probes to the frozen inventory and reconstructs historical gates from hash-bound attempted work, including failures and incomplete accounting.

The retained 43-task matrix shows non-help calls to **72/938 original nodes**; **866 have no observed non-help call in those retained traces**. There are 78 observed routes in total, including candidate-only routes. Missing or sanitized traces cannot prove that a command was never used. Observed use is not verification of every flag/mode, and verified task success does not attribute the whole task saving to each command it called.

Historical task gates remain **13 met, 22 missed, 8 unavailable**. Those measured binaries differ from this candidate. **No current per-function provider-token savings are established** by the local probes or by reclassifying historical trials. The held-out suite was not used.

Every area stays in scope: Apps, Base, Calendar, Contact, Docs, Drive, IM, Mail, Markdown, Minutes, Note, OKR, Sheets, Slides, Task, VC, Whiteboard, Wiki, applications, foundation/lifecycle, events, extensions and packaging. Mutations, permission conditions, pagination, recovery, streams, binary transfers, SDK behavior and platform operations still need their own correctness and comparative workflow evidence. Extensions and packaging are explicitly unmeasured here; Windows execution and live tenant parity have no fresh evidence.

## Reproduction and validation

[Probe evidence](probes.json.gz) stores every arm, arguments, output hashes, context sizes, sample duration and contract verdict. [Function ledger](function-coverage.json.gz) records capability-level observations and historical binary hashes. Both files are deterministic gzip JSON containers. Follow [`measure_surface.py` and `function_coverage.py`](../../README.md#offline-cli-probes) to reproduce them with new output paths. Exact binary, catalog, inventory, tokenizer (`o200k_base`, tiktoken 0.12.0) and runner hashes are recorded; no private output contents or host paths are published.

Validation results are recorded in [validation.json](validation.json).
