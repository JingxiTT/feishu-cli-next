# Business development comparison

Completed **18 attempts**, with **18 verified successes** and 0 inconclusive outcomes. Known provider usage totals **2,741,785 tokens**; 0 attempts have unavailable usage. Non-success-attempt known costs total **0 tokens** and are retained.

Provider input includes cached input in full; output and failed/recovery turns are counted. Token distributions come from the provider, not byte estimates. A group with unknown usage, inconclusive outcomes, or no successful outcome has no finite tokens-per-success result.

| Task | Arm | Success | Tokens/success | Complete-usage range | API requests |
| --- | --- | --- | ---: | ---: | ---: |
| business-dev-approval-flow | official | 3/3 | 196,014.67 | 183,099–207,832 | 50 |
| business-dev-approval-flow | official_concise | 3/3 | 182,855.33 | 175,585–194,419 | 42 |
| business-dev-approval-flow | next | 3/3 | 120,166.00 | 118,937–120,841 | 24 |
| business-dev-mindnote-edit | official | 3/3 | 158,449.67 | 152,265–167,352 | 39 |
| business-dev-mindnote-edit | official_concise | 3/3 | 150,169.33 | 136,234–157,207 | 35 |
| business-dev-mindnote-edit | next | 3/3 | 106,273.33 | 99,907–117,124 | 18 |

Next used fewer tokens and fewer total API requests than both comparison arms on each of the two sampled tasks. These tasks cover approval creation/readback and an existing-node Mindnote edit with readback after an uncertain outcome. They do not establish an advantage for every generated method or a full-product improvement.

The frozen original arm uses its shared CLI skill; the concise-original arm uses the reviewed English shared guidance. Twelve separate controlled cases cover all seventeen native methods, while these two representative tasks receive model trials. The corrected fixture permits equivalent JSON serialization and uses source-backed native list fields; prompts, input meaning and mutation/idempotency semantics are unchanged.

The product is `8d92e52`, reference `7fd6ef3`, and evaluator `4b3abc6`. Entry hashes, per-section seeded orders and paired observations are retained in the environment and batch manifests. All arms used Codex CLI 0.153.4, `gpt-6-astra`/high, equivalent shell/file tools, fresh workspaces and signed non-forwarding local fixtures. The new candidate guide includes `43bdff2`; earlier 74/312 runs used different guidance and remain separate.

These are three fresh repetitions per arm/task on fixed synthetic data. Small samples and variable model command choices limit generalization. Total API counts include identity and business requests; no backend request is silently subtracted. Per-turn token totals cannot establish the cost of a particular help response, output format, or individual command.

No held-out trials ran or held-out files changed. The coordinator’s earlier limited snippet exposure and affected-example quarantine remain documented in [the exposure record](../held-out-exposure.json). No live Feishu content, credentials, private paths or raw model traces are published.

See the [execution-environment note](execution-environment.json) for concurrent correctness-workload limits on latency attribution.

See [command choices and uncertainty recovery](trace-notes.md) for the recorded Mindnote exit-22/readback sequence and attribution limits.
