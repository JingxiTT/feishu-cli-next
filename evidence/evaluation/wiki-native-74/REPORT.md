# Corrected Wiki development comparison

All 9 fresh trials passed independent resolution, content, signed-read and unchanged-state checks. Total provider usage was **1,595,048 tokens**, including cached input in full; no missing usage or excluded attempts. These are actual provider counts, not CLI output bytes.

| Arm | Success | Mean tokens/success | Token range | Total API requests |
| --- | --- | ---: | ---: | ---: |
| official | 3/3 | 212,169.33 | 142,430–265,345 | 27 |
| official_concise | 3/3 | 207,725.67 | 187,647–218,209 | 30 |
| next | 3/3 | 111,787.67 | 111,595–111,900 | 12 |

The candidate used fewer tokens in this small task sample than either baseline. Three repetitions and one fixed synthetic document do not establish broad efficiency or statistical significance. API totals include identity requests; token counts aggregate the whole model turn and cannot isolate the cost of an individual help call or response.

The reference is `7fd6ef3`; the candidate is `74b6b53`; the evaluator and corrected fixture are `c875148`. All arms used Codex CLI 0.153.4, `gpt-6-astra` with high reasoning, identical shell/file tools, isolated fresh workspaces and signed non-forwarding loopback fixtures. The seeded order is 20260918. The candidate uses its original 74 guide, without the later `43bdff2` wording reorder. Binary and entry hashes are retained in the batch and environment manifests.

The correction adds the source-backed native Docx raw-content GET as an equivalent read strategy. The prompt, output schema and expected answer are unchanged. Controlled validation covered both clean binaries, both identities and omitted/example-zero language selectors before models ran. Sanitized route traces now show the native read succeeding. This resolves the demonstrated fixture coverage gap; it does not retrospectively explain every earlier failure.

The [original 81-trial batch](../read-312af93/REPORT.md), including six failed Wiki attempts costing 1,146,338 tokens, is retained unchanged. The [six-attempt mechanism probe](../wiki-diagnostic-74/REPORT.md) retains a further 1,091,185 tokens and separately identifies three slots cancelled before any model started. Neither earlier series is replaced or pooled with this corrected comparison.

No held-out trials ran or files changed. The coordinator’s limited earlier held-out snippet exposure is recorded in [the quarantine record](../held-out-exposure.json); affected examples require independent replacement after product freeze. No live Feishu content or credentials are present in published evidence.
