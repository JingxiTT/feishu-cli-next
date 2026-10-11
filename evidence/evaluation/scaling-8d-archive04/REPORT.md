# Scaling development comparison

> **Validity amendment (2026-09-08):** The [native-fidelity audit](../native-fidelity-validity-audit-20260908.md) qualifies synthetic fidelity assumptions for Base and Sheets size tasks. Original verdicts and provider costs below remain historical; they do not establish equal live fidelity or live-fidelity efficiency. Original failure attributions and unknown costs remain unchanged.

Completed **54 attempts**, with **51 verified successes** and 0 inconclusive outcomes. Known provider usage totals **9,780,669 tokens**; 1 attempt has unavailable usage. Non-success-attempt known costs total **499,819 tokens** and are retained.

The complete total is unknown: 9,780,669 is a lower bound because the concise-original 1,000-row Sheets trial reached the unchanged 180-second budget before a provider usage summary. Its missing total is not plotted as zero, and the affected group has no finite efficiency result. The two Base nonverified outcomes retain 499,819 known tokens; their generic rejected-request reason cannot identify a product cause.

Provider input includes cached input in full; output and failed/recovery turns are counted. Token distributions come from the provider, not byte estimates. A group with unknown usage, inconclusive outcomes, or no successful outcome has no finite tokens-per-success result.

| Task | Arm | Success | Tokens/success | Complete-usage range | API requests |
| --- | --- | --- | ---: | ---: | ---: |
| base-scale-dev-00100 | official | 3/3 | 127,404.33 | 96,308–147,036 | 16 |
| base-scale-dev-00100 | official_concise | 3/3 | 112,461.33 | 88,337–125,081 | 17 |
| base-scale-dev-00100 | next | 2/3 | 300,151.50 | 168,318–244,366 | 48 |
| base-scale-dev-01000 | official | 3/3 | 140,715.33 | 125,320–171,484 | 41 |
| base-scale-dev-01000 | official_concise | 3/3 | 126,638.00 | 110,553–148,161 | 43 |
| base-scale-dev-01000 | next | 2/3 | 271,262.00 | 94,802–255,453 | 72 |
| base-scale-dev-10000 | official | 3/3 | 221,430.33 | 144,992–372,564 | 347 |
| base-scale-dev-10000 | official_concise | 3/3 | 166,207.33 | 162,884–168,241 | 311 |
| base-scale-dev-10000 | next | 3/3 | 140,433.33 | 119,518–151,404 | 175 |
| sheets-size-dev-100 | official | 3/3 | 295,636.33 | 263,326–343,308 | 9 |
| sheets-size-dev-100 | official_concise | 3/3 | 222,044.33 | 202,589–238,844 | 9 |
| sheets-size-dev-100 | next | 3/3 | 100,604.67 | 89,709–106,211 | 6 |
| sheets-size-dev-1000 | official | 3/3 | 277,664.67 | 233,272–319,785 | 9 |
| sheets-size-dev-1000 | official_concise | 2/3 | unavailable | 178,052–238,517 | 7 |
| sheets-size-dev-1000 | next | 3/3 | 106,845.00 | 106,251–107,246 | 9 |
| sheets-size-dev-10000 | official | 3/3 | 328,489.33 | 304,000–346,756 | 9 |
| sheets-size-dev-10000 | official_concise | 3/3 | 256,691.00 | 216,763–278,297 | 13 |
| sheets-size-dev-10000 | next | 3/3 | 117,159.00 | 105,820–122,923 | 51 |

Next regressed on the smaller Base tasks but verified all three 10,000-row imports with fewer total tokens and API requests than either original arm at that size. For Sheets, Next used fewer tokens than the original at every size and the concise original at 100 and 10,000 rows; the concise 1,000-row total is unknown. At 10,000 rows, Next made 51 total API requests versus 9 original and 13 concise, despite its lower provider usage. There is no uniform API advantage.

Both original and Next file workflows show provider-token variation far smaller than the 100-fold row-count change in this sample. Three fixed sizes and three repetitions cannot establish an asymptotic bound or isolate how much of the difference comes from guidance, command discovery, formatting or model choices. Tokens/success divides all attempt costs, including known failures, by verified successes only when every attempt has complete usage.

The plot retains historical provider costs and synthetic verification markers. The [validity amendment](../native-fidelity-validity-audit-20260908.md) withholds live same-fidelity scaling conclusions for both tasks; the figure has not been regenerated.

![Actual provider tokens by size](usage-by-size.png)

The [standalone SVG](usage-by-size.svg) and [plot generator](../../plot_scaling.py) use every complete-usage attempt, including nonverified ones. The unknown-total attempt is disclosed rather than estimated. Independent recount and visual review were performed by the coordinator.

The product is `8d92e52`, reference `7fd6ef3`, and evaluator `4b3abc6`. Entry hashes, per-section seeded orders and paired observations are retained in the environment and batch manifests. All arms used Codex CLI 0.153.4, `gpt-6-astra`/high, equivalent shell/file tools, fresh workspaces and signed non-forwarding local fixtures. The new candidate guide includes `43bdff2`; earlier 74/312 runs used different guidance and remain separate.

These are three fresh repetitions per arm/task on fixed synthetic data. Small samples and variable model command choices limit generalization. Total API counts include identity and business requests; no backend request is silently subtracted. Per-turn token totals cannot establish the cost of a particular help response, output format, or individual command.

No held-out trials ran or held-out files changed. The coordinator’s earlier limited snippet exposure and affected-example quarantine remain documented in [the exposure record](../held-out-exposure.json). No live Feishu content, credentials, private paths or raw model traces are published.

The row-count family is 100/1,000/10,000, prefix-paired across sizes and fixed across arms/repetitions. Actual artifacts/backend typed state, complete coverage and protected source/sentinels were independently verified, then workspace artifacts were discarded. Controlled artifact-byte diagnostics remain separate in [scaling controls](../scaling-controlled.json); they are not per-model token measurements. Base starts with an identical typed exchange file/mapping for all arms, so comparisons include file interoperability and guidance rather than isolating serialization. No fixed percentage improvement or asymptotic complexity claim is justified by three sizes.

See the [execution-environment note](execution-environment.json) for concurrent correctness-workload limits on latency attribution.

The Base regression and its observed command choices are reviewed in [the trace assessment](base-trace.md); no automatic tuning or rerun was applied.

Sheets workflow and timeout observations are summarized in [the command-choice notes](trace-notes.md).
