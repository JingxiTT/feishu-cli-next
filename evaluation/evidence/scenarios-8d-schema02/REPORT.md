# Scenario development comparison

Completed **81 attempts**, with **63 verified successes** and 18 inconclusive outcomes. Known provider usage totals **12,201,326 tokens**; one attempt has unavailable complete usage. Known costs of non-verified/inconclusive attempts total **2,282,405 tokens** and are retained.

The known total is a lower bound because one Extensions attempt has only a known partial 360,167-token span; its complete total remains unavailable. The range column uses only attempts with complete provider usage.

Provider input includes cached input in full; output and failed/recovery turns are counted. Token distributions come from the provider, not byte estimates. A group with unknown usage, inconclusive outcomes, or no successful outcome has no finite tokens-per-success result.

| Task | Arm | Success | Tokens/success | Complete-usage range | API requests |
| --- | --- | --- | ---: | ---: | ---: |
| base-dev-01 | official | 3/3 | 89,044.67 | 55,692–117,829 | 25 |
| base-dev-01 | official_concise | 3/3 | 93,971.67 | 81,778–100,091 | 33 |
| base-dev-01 | next | 3/3 | 126,085.67 | 110,814–134,317 | 25 |
| docs-dev-01 | official | 3/3 | 212,633.33 | 185,753–248,475 | 42 |
| docs-dev-01 | official_concise | 3/3 | 75,534.67 | 75,521–75,556 | 24 |
| docs-dev-01 | next | 2/3; 1 inconclusive | unavailable | 228,173–279,520 | 31 |
| drive-dev-01 | official | 3/3 | 160,570.33 | 148,384–179,756 | 10 |
| drive-dev-01 | official_concise | 3/3 | 141,622.00 | 120,968–152,429 | 12 |
| drive-dev-01 | next | 3/3 | 66,597.67 | 66,538–66,643 | 9 |
| events-dev-01 | official | 0/3; 3 inconclusive | unavailable | 140,229–173,614 | 12 |
| events-dev-01 | official_concise | 3/3 | 99,176.67 | 97,803–101,423 | 12 |
| events-dev-01 | next | 3/3 | 92,663.00 | 78,919–99,552 | 6 |
| extensions-dev-01 | official | 2/3; 1 inconclusive | unavailable | 148,659–195,139 | 0 |
| extensions-dev-01 | official_concise | 1/3; 2 inconclusive | unavailable | 178,671–403,398 | 0 |
| extensions-dev-01 | next | 1/3; 2 inconclusive | unavailable | 91,343–147,103 | 0 |
| markdown-dev-01 | official | 3/3 | 252,573.67 | 202,385–296,129 | 23 |
| markdown-dev-01 | official_concise | 3/3 | 161,417.33 | 155,812–166,411 | 18 |
| markdown-dev-01 | next | 3/3 | 182,177.00 | 167,860–210,766 | 12 |
| packaging-dev-01 | official | 0/3; 3 inconclusive | unavailable | 57,630–74,719 | 0 |
| packaging-dev-01 | official_concise | 0/3; 3 inconclusive | unavailable | 56,977–70,325 | 0 |
| packaging-dev-01 | next | 0/3; 3 inconclusive | unavailable | 54,260–71,764 | 0 |
| sheets-dev-01 | official | 3/3 | 238,373.00 | 237,729–238,754 | 9 |
| sheets-dev-01 | official_concise | 3/3 | 196,719.33 | 185,300–204,126 | 10 |
| sheets-dev-01 | next | 3/3 | 78,226.00 | 72,421–88,948 | 6 |
| task-dev-01 | official | 3/3 | 272,985.33 | 224,538–312,899 | 36 |
| task-dev-01 | official_concise | 3/3 | 185,645.33 | 142,544–213,277 | 34 |
| task-dev-01 | next | 3/3 | 123,521.00 | 113,156–128,847 | 24 |

Five tasks have nine verified outcomes each: Base, Drive, Markdown, Sheets and Task. Next used fewer tokens than both baselines on Drive, Sheets and Task; it regressed against both on Base and against concise guidance on Markdown. These mixed results do not support a uniform improvement claim.

Docs v1 has one recorded unsupported-strategy outcome costing 279,520 tokens. Its raw request values were discarded, so the exact XML/reconstruction branch cannot be recovered. Review later identified hidden target GUID/segmentation/fixed-ID constraints stronger than the task prompt; separately versioned v2 accepts native semantic preservation without inventing real backend ID reuse. This does not retroactively establish that the historical attempt met every task requirement.

Post-run interpretation reviews retain raw verdicts/costs and explain why affected observations cannot support product comparisons: [events-outcome-review.json](events-outcome-review.json), [extensions-outcome-review.json](extensions-outcome-review.json), [packaging-outcome-review.json](packaging-outcome-review.json).

The product is `8d92e52`, reference `7fd6ef3`, and evaluator `dacbfcc`. Entry hashes, per-section seeded orders and paired observations are retained in the environment and batch manifests. All arms used Codex CLI 0.153.4, `gpt-6-astra`/high, equivalent shell/file tools, fresh workspaces and signed non-forwarding local fixtures. The new candidate guide includes `43bdff2`; earlier 74/312 runs used different guidance and remain separate.

These are three fresh repetitions per arm/task on fixed synthetic data. Small samples and variable model command choices limit generalization. Total API counts include identity and business requests; no backend request is silently subtracted. Per-turn token totals cannot establish the cost of a particular help response, output format, or individual command.

No held-out trials ran or held-out files changed. The coordinator’s earlier limited snippet exposure and affected-example quarantine remain documented in [the exposure record](../held-out-exposure.json). No live Feishu content, credentials, private paths or raw model traces are published.

See the [execution-environment note](execution-environment.json) for concurrent correctness-workload limits on latency attribution.

See [command-choice interpretation](trace-notes.md) for workflow, discovery, and reference-reading observations and their limits.

The Base regression and its observed command choices are reviewed in [the trace assessment](base-trace.md); no automatic tuning or rerun was applied.
