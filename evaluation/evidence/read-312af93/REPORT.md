# Nine-section development comparison

The frozen batch completed **75/81 independently verified successes**, using **8,419,135 provider tokens** including cached input in full. The six failed Wiki attempts consumed **1,146,338 tokens**; none were dropped. The first eight sections passed 9/9 each; Wiki passed 3/9.

Candidate `312af93`, reference `7fd6ef3`, evaluator `cc745f5`, Codex CLI `0.153.4`, model `gpt-6-astra`, reasoning `high`. Each section has three fresh trials per arm, a seeded balanced order, an identical signed non-forwarding synthetic backend, and shell/file tools. These are small development samples, not live backend or held-out results.

## Tokens per verified success

Costs include failed attempts. “Unavailable” means zero verified successes. Values are rounded; exact per-attempt distributions, usage/cache components, timings and traces remain in each section’s `batch-01.json`.

| Section | Original | Concise original | Next | Verified successes (original / concise / next) |
|---|---:|---:|---:|---|
| application | 23,864 | 24,052 | 50,032 | 3/3 / 3/3 / 3/3 |
| apps | 127,839 | 83,564 | 97,030 | 3/3 / 3/3 / 3/3 |
| im | 284,703 | 213,348 | 87,958 | 3/3 / 3/3 / 3/3 |
| minutes | 126,736 | 41,429 | 63,117 | 3/3 / 3/3 / 3/3 |
| note | 188,952 | 50,549 | 113,471 | 3/3 / 3/3 / 3/3 |
| slides | 214,026 | 29,362 | 66,575 | 3/3 / 3/3 / 3/3 |
| vc | 121,205 | 32,479 | 51,016 | 3/3 / 3/3 / 3/3 |
| whiteboard | 82,471 | 27,157 | 49,669 | 3/3 / 3/3 / 3/3 |
| wiki | 621,505 | 299,638 | Unavailable | 1/3 / 2/3 / 0/3 |

Across the eight fully successful sections, Next uses fewer tokens than the original arm in seven, but fewer than concise original guidance only in IM. Application is worse against both baselines. This does not support a general product improvement claim. Prior Foundation and Contact evidence remains separate; Contact’s earlier regression is not overwritten.

## Observed discovery and command choices

Application: every concise run executes the list command directly (one shell execution, two API calls). Every Next run uses discovery, leaf help, then the same list command (three shell executions, four API calls). Token ranges do not overlap: concise 24,037–24,059; Next 49,519–50,293.

Note: both arms read details, fetch a transcript artifact and read the artifact. Next additionally performs two discovery calls and leaf help for both operations in every run, then a skill-read or API-help call. Next uses eight shell executions and ten API calls in every run, versus three to five executions and five to six API calls for concise guidance. Token ranges: concise 50,318–50,771; Next 112,665–114,887. The artifact is part of both approaches; its existence alone does not explain the difference.

Slides: concise guidance reaches `slides +xml-get` directly in each run (one execution, two API calls). Next uses root help or discovery, domain help, leaf help, then the same read (four executions, five API calls). Token ranges: concise 29,360–29,366; Next 65,694–67,046.

These repeated patterns identify extra discovery/context exposure. Provider usage is aggregated per user turn, so the evidence cannot assign tokens to a particular help output, prompt component or API call. Query values, output content and byte sizes were deliberately not retained. Three observations do not establish a population variance estimate or causal decomposition. The original compatibility entries for Note/Minutes/VC route to the larger meeting skill; their additional reference reading is a guidance-condition difference, not a CLI-output comparison.

A separate controlled product check identified eager identity reads during local help/discovery in this older candidate. That concrete issue was fixed after this batch’s freeze; the new version needs its own balanced measurement. These traces do not justify a universal command sequence or task-specific guidance. A possible follow-up is making existing optional discovery flag metadata sufficient for accurate argument selection; exact query causality cannot be reconstructed from sanitized traces.

## Wiki attribution limit

Wiki results are original 1/3, concise original 2/3, Next 0/3. All failed traces contain a failed raw `api` call; several later content reads exit successfully. The frozen verifier records only pass/fail, while exact API method/path, payload and the first failed invariant were discarded. A malformed request, a legitimate unmodeled native-document route, and a final-answer mismatch cannot be distinguished retrospectively. The observed verification failures and costs remain in the dataset; they are **not attributed to a product bug**. No finite Next tokens-per-success value is reported.

The runtime supports Wiki resolution and DocsAI XML/Markdown fetch, not every native document read strategy. Any demonstrated fixture gap requires a new version and a full balanced Wiki rerun, preserving these attempts. Future evaluator `f846268` adds only allowlisted route IDs, methods/statuses and fixed verifier reasons. The old raw batch’s accounting flag does not resolve this attribution limitation; the assessment here supersedes a whole-product performance interpretation.

## Provenance and privacy limits

Application has no frozen matching skill and uses its unchanged official affordance fallback, disclosed in environment.json. Both official arms have the same original reference bundle available. The model has only synthetic data and an empty isolated Feishu configuration; no live forwarding or user credential content was retained. Independent verifiers run locally without a second model. Retry counts are unavailable; total API counts include identity/bootstrap reads and are not equivalent to business operations.

Held-out files were neither executed nor modified. The coordinator reported accidental limited search snippets during later scaling review; affected examples are quarantined pending independent blind replacement after product freeze. See `../held-out-exposure.json`; complete coordinator blinding is not claimed.

A later [mechanism probe](../wiki-diagnostic-74/REPORT.md) demonstrated a missing
native Docx raw-content route in the fixture. It does not establish the cause of
every earlier frozen failure. Original attempts and costs remain unchanged; a
corrected, separately versioned balanced Wiki follow-up is required.
