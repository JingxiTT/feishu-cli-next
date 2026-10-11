# Read development comparison

Completed **27 attempts**, with **27 verified successes** and 0 inconclusive outcomes. Known provider usage totals **3,647,948 tokens**; 0 attempts have unavailable usage. Failed-attempt known costs total **0 tokens** and are retained.

Next used fewer tokens per successful task than both baselines on each of these three workloads. This does not establish a full-product improvement or attribute the difference to the guide revision alone.

Provider input includes cached input in full; output and failed/recovery turns are counted. Token distributions come from the provider, not byte estimates. A group with unknown usage or no successful outcome has no finite tokens-per-success result.

| Task | Arm | Success | Tokens/success | All-attempt range | API requests |
| --- | --- | --- | ---: | ---: | ---: |
| calendar-dev-01 | official | 3/3 | 153,198.67 | 118,167–196,348 | 15 |
| calendar-dev-01 | official_concise | 3/3 | 146,588.67 | 92,894–178,965 | 19 |
| calendar-dev-01 | next | 3/3 | 98,273.33 | 92,663–109,253 | 12 |
| mail-dev-01 | official | 3/3 | 258,457.00 | 209,403–330,088 | 24 |
| mail-dev-01 | official_concise | 3/3 | 116,976.33 | 73,940–138,495 | 12 |
| mail-dev-01 | next | 3/3 | 68,579.00 | 68,575–68,586 | 9 |
| okr-dev-01 | official | 3/3 | 169,457.00 | 162,109–175,581 | 30 |
| okr-dev-01 | official_concise | 3/3 | 120,259.00 | 90,926–135,249 | 31 |
| okr-dev-01 | next | 3/3 | 84,193.67 | 83,975–84,313 | 24 |

The product is `8d92e52`, reference `7fd6ef3`, and evaluator `117d76a`. Entry hashes, per-section seeded orders and paired observations are retained in the environment and batch manifests. All arms used Codex CLI0.153.4, `gpt-6-astra`/high, equivalent shell/file tools, fresh workspaces and signed non-forwarding local fixtures. The new candidate guide includes `43bdff2`; earlier74/312 runs used different guidance and remain separate.

These are three fresh repetitions per arm/task on fixed synthetic data. Small samples and variable model command choices limit generalization. Total API counts include identity and business requests; no backend request is silently subtracted. Per-turn token totals cannot establish the cost of a particular help response, output format, or individual command.

No held-out trials ran or files changed. The coordinator’s earlier limited snippet exposure and affected-example quarantine remain documented in [the exposure record](../held-out-exposure.json). No live Feishu content, credentials, private paths or raw model traces are published.
