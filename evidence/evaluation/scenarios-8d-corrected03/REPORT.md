# Scenario development comparison

> **Validity amendment (2026-09-08):** The [native-fidelity audit](../native-fidelity-validity-audit-20260908.md) qualifies synthetic fidelity assumptions for Docs. Original verdicts and provider costs below remain historical; they do not establish equal live fidelity or live-fidelity efficiency. Original failure attributions and unknown costs remain unchanged.

Completed **27 attempts**, with **27 verified successes** and 0 inconclusive outcomes. Known provider usage totals **3,702,291 tokens**; 0 attempts have unavailable usage. Non-success-attempt known costs total **0 tokens** and are retained.

Provider input includes cached input in full; output and failed/recovery turns are counted. Token distributions come from the provider, not byte estimates. A group with unknown usage, inconclusive outcomes, or no successful outcome has no finite tokens-per-success result.

| Task | Arm | Success | Tokens/success | Complete-usage range | API requests |
| --- | --- | --- | ---: | ---: | ---: |
| docs-dev-01 | official | 3/3 | 161,107.00 | 90,024–230,112 | 32 |
| docs-dev-01 | official_concise | 3/3 | 76,104.33 | 76,086–76,117 | 24 |
| docs-dev-01 | next | 3/3 | 206,796.33 | 161,458–287,415 | 24 |
| extensions-dev-01 | official | 3/3 | 199,107.33 | 158,442–223,139 | 0 |
| extensions-dev-01 | official_concise | 3/3 | 305,044.00 | 220,138–405,941 | 0 |
| extensions-dev-01 | next | 3/3 | 94,401.67 | 80,588–104,224 | 0 |
| packaging-dev-01 | official | 3/3 | 58,075.33 | 57,863–58,188 | 0 |
| packaging-dev-01 | official_concise | 3/3 | 73,169.00 | 71,993–74,997 | 0 |
| packaging-dev-01 | next | 3/3 | 60,292.00 | 54,708–71,306 | 0 |

Next used more tokens than both comparison arms on Docs, fewer than both on Extensions, and slightly more than the original but fewer than the concise original on Packaging. These mixed results do not establish a uniform token advantage.

The frozen upstream tree has no matching Extensions skill. That arm therefore uses its actual extension/platform READMEs and the command example source as the declared source-matched guidance fallback; the concise arm summarizes that capability guidance. Packaging installs the selected distribution supplied to each arm, and the original/concise arms install identical upstream artifacts. These conditions compare the declared CLI-plus-guidance workflows, not text length alone.

The product is `8d92e52`, reference `7fd6ef3`, and evaluator `ea32e97`. Entry hashes, per-section seeded orders and paired observations are retained in the environment and batch manifests. All arms used Codex CLI 0.153.4, `gpt-6-astra`/high, equivalent shell/file tools, fresh workspaces and signed non-forwarding local fixtures. The new candidate guide includes `43bdff2`; earlier 74/312 runs used different guidance and remain separate.

These are three fresh repetitions per arm/task on fixed synthetic data. Small samples and variable model command choices limit generalization. Total API counts include identity and business requests; no backend request is silently subtracted. Per-turn token totals cannot establish the cost of a particular help response, output format, or individual command.

No held-out trials ran or held-out files changed. The coordinator’s earlier limited snippet exposure and affected-example quarantine remain documented in [the exposure record](../held-out-exposure.json). No live Feishu content, credentials, private paths or raw model traces are published.

See the [execution-environment note](execution-environment.json) for concurrent correctness-workload limits on latency attribution.

See [command-choice interpretation](trace-notes.md) for workflow, discovery, and reference-reading observations and their limits.

The independently recounted Docs regression is reviewed in [the source-free trace assessment](docs-trace.md).
