# Luna medium: same-task token evaluation

Completed **72/72 attempts** using `gpt-5.6-luna`, reasoning `medium`, native Codex CLI `0.153.4`. **54 verified successes**, **7 inconclusive fixture outcomes**, **11 failed verifications**, and **0 attempts with incomplete usage**.

Observed provider usage totals **8,442,585 tokens**, including **3,454,343 tokens on unsuccessful or inconclusive attempts**. This comprises 8,357,840 input tokens and 84,745 output tokens. Input includes 6,382,336 cached tokens, counted once. Unknown usage is never treated as zero.

This is a small development pilot: eight different functions, each compared on the same task across three arms and three repetitions. Signed synthetic backends isolate Feishu state; these are real model calls, not live Feishu workloads. Interpret each function separately.

The clearest improvement was **Drive**: Next completed 3/3 roundtrips with **51.1% fewer tokens** than concise guidance, which also completed 3/3. Against official/full, Next used **58.7% fewer tokens on the two matched successful trials**; its all-attempt tokens/success was 68.3% lower after retaining the baseline failure.

Against official/full, Next used **33.0% more tokens for Discovery**, **47.9% more for Contact**, and **91.5% more for Docs read**, with 3/3 successes per arm on each task. Docs edit (−2.9%) and Base (+2.9%) had small differences that this three-repeat pilot does not establish as stable improvements or regressions.

**Sheets:** Next verified 3/3 exports; neither baseline produced a verified export, and one baseline attempt was inconclusive. A comparative token-saving percentage is unavailable. **Markdown:** no attempts were verified and six were inconclusive, so the fixture needs broader native readback coverage before that function can support an efficiency conclusion.

Concise guidance alone did not consistently reduce total task tokens: Discovery fell from 41,521 to 22,835 tokens/success, while Docs read rose from 32,208 to 56,973, with all three attempts successful in each comparison. On Contact, concise guidance was cheaper than Next on matched successes; its worse all-attempt tokens/success includes a failed attempt.

## Per-function results

Tokens/success includes every attempt cost in that task/arm. It is unavailable with zero successes, incomplete accounting, or fixture-inconclusive outcomes. Median/range describe attempts with complete usage, including failures.

| Task | Arm | Success / attempts | Inconclusive | Known tokens | Tokens / success | Median [min–max] | API calls | Median runner seconds |
| --- | --- | ---: | ---: | ---: | ---: | --- | ---: | ---: |
| base | Next + entry skill | 3/3 | 0 | 359,570 | 119,857 | 117,238 [92,929–149,403] | 15 | 43.6 |
| base | Official + full skill | 3/3 | 0 | 349,436 | 116,479 | 136,654 [43,402–169,380] | 22 | 38.8 |
| base | Official + concise skill | 3/3 | 0 | 385,184 | 128,395 | 129,791 [95,874–159,519] | 27 | 42.1 |
| contact | Next + entry skill | 3/3 | 0 | 209,549 | 69,850 | 69,423 [61,393–78,733] | 12 | 37.9 |
| contact | Official + full skill | 3/3 | 0 | 141,725 | 47,242 | 47,775 [44,436–49,514] | 15 | 29.9 |
| contact | Official + concise skill | 2/3 | 0 | 152,924 | 76,462 | 42,909 [40,784–69,231] | 27 | 30.5 |
| discovery | Next + entry skill | 3/3 | 0 | 165,661 | 55,220 | 58,305 [47,442–59,914] | 3 | 35.5 |
| discovery | Official + full skill | 3/3 | 0 | 124,562 | 41,521 | 31,650 [29,261–63,651] | 9 | 29.4 |
| discovery | Official + concise skill | 3/3 | 0 | 68,505 | 22,835 | 18,280 [7,918–42,307] | 6 | 27.6 |
| docs-edit | Next + entry skill | 3/3 | 0 | 428,918 | 142,973 | 122,017 [117,602–189,299] | 18 | 47.4 |
| docs-edit | Official + full skill | 3/3 | 0 | 441,924 | 147,308 | 147,319 [132,523–162,082] | 38 | 53.7 |
| docs-edit | Official + concise skill | 2/3 | 0 | 545,468 | 272,734 | 181,597 [168,032–195,839] | 39 | 58.8 |
| docs-read | Next + entry skill | 3/3 | 0 | 185,038 | 61,679 | 55,733 [47,827–81,478] | 6 | 39.2 |
| docs-read | Official + full skill | 3/3 | 0 | 96,625 | 32,208 | 32,228 [31,979–32,418] | 6 | 25.0 |
| docs-read | Official + concise skill | 3/3 | 0 | 170,918 | 56,973 | 60,477 [44,611–65,830] | 11 | 29.7 |
| drive | Next + entry skill | 3/3 | 0 | 256,162 | 85,387 | 89,606 [67,276–99,280] | 21 | 37.0 |
| drive | Official + full skill | 2/3 | 0 | 538,019 | 269,010 | 184,269 [134,392–219,358] | 26 | 45.1 |
| drive | Official + concise skill | 3/3 | 0 | 523,625 | 174,542 | 129,043 [127,085–267,497] | 34 | 47.3 |
| markdown | Next + entry skill | 0/3 | 3 | 361,700 | — | 129,000 [101,043–131,657] | 14 | 47.8 |
| markdown | Official + full skill | 0/3 | 0 | 829,551 | — | 226,938 [226,355–376,258] | 40 | 73.0 |
| markdown | Official + concise skill | 0/3 | 3 | 488,163 | — | 161,299 [125,534–201,330] | 18 | 48.2 |
| sheets | Next + entry skill | 3/3 | 0 | 216,084 | 72,028 | 69,293 [59,895–86,896] | 9 | 31.8 |
| sheets | Official + full skill | 0/3 | 1 | 768,273 | — | 269,493 [195,559–303,221] | 19 | 55.3 |
| sheets | Official + concise skill | 0/3 | 0 | 635,001 | — | 224,468 [183,188–227,345] | 11 | 72.4 |

## Matched successful attempts

These compare the same task and repetition only where both arms succeeded with complete usage. They exclude failures and therefore supplement, rather than replace, tokens/success.

| Task | Baseline | Matched pairs | Baseline tokens | Next tokens | Next change |
| --- | --- | ---: | ---: | ---: | ---: |
| base | Official + full skill | 3 | 349,436 | 359,570 | +2.9% |
| contact | Official + full skill | 3 | 141,725 | 209,549 | +47.9% |
| discovery | Official + full skill | 3 | 124,562 | 165,661 | +33.0% |
| docs-edit | Official + full skill | 3 | 441,924 | 428,918 | -2.9% |
| docs-read | Official + full skill | 3 | 96,625 | 185,038 | +91.5% |
| drive | Official + full skill | 2 | 403,627 | 166,556 | -58.7% |
| markdown | Official + full skill | 0 | — | — | — |
| sheets | Official + full skill | 0 | — | — | — |
| base | Official + concise skill | 3 | 385,184 | 359,570 | -6.6% |
| contact | Official + concise skill | 2 | 83,693 | 140,126 | +67.4% |
| discovery | Official + concise skill | 3 | 68,505 | 165,661 | +141.8% |
| docs-edit | Official + concise skill | 2 | 377,436 | 239,619 | -36.5% |
| docs-read | Official + concise skill | 3 | 170,918 | 185,038 | +8.3% |
| drive | Official + concise skill | 3 | 523,625 | 256,162 | -51.1% |
| markdown | Official + concise skill | 0 | — | — | — |
| sheets | Official + concise skill | 0 | — | — | — |

## Observed workflow overhead

On Discovery, median shell tool executions were 5 for Next, 2 for official/full and 1 for official/concise. On Contact they were 9, 4 and 4. The Next Contact traces include 14 discover invocations across three trials. These observations are consistent with extra discovery work, but the retained usage does not isolate tokens for individual commands. [Contact/Discovery traces](contact/batch-01.json).

On Base, Next made 15 fixture API calls versus 22 for official/full across three successful trials, while using 2.90% more tokens. API request count alone therefore does not measure agent token efficiency. [Base traces](base/batch-01.json).

Next also invoked skills read during Markdown. Injecting only its new entry skill therefore did not eliminate on-demand guidance loading. The sanitized trace omits skill arguments, so it cannot identify which specific guide was read. [Markdown traces](markdown/batch-01.json); summary.json retains public command counts by task and arm.


## Failure and accounting evidence

| Task | Arm | Categories (attempt counts; may overlap) |
| --- | --- | --- |
| contact | Official + concise skill | misleading_success_claim: 1; outcome_failure: 1 |
| docs-edit | Official + concise skill | misleading_success_claim: 1; outcome_failure: 1 |
| drive | Official + full skill | misleading_success_claim: 1; outcome_failure: 1 |
| markdown | Next + entry skill | fixture_unsupported: 3 |
| markdown | Official + full skill | misleading_success_claim: 3; outcome_failure: 3 |
| markdown | Official + concise skill | fixture_unsupported: 3 |
| sheets | Official + full skill | fixture_unsupported: 1; misleading_success_claim: 2; outcome_failure: 2 |
| sheets | Official + concise skill | misleading_success_claim: 3; outcome_failure: 3 |

## Conditions and limits

- All three arms use the same task, initial state, independent result checks, model, effort, tools and time limits: 180 seconds for the agent, 210 seconds outside it. Each attempt starts a fresh context. Trials are serial and randomized within section using recorded seeds.
- Official and Official-concise use the identical reference binary. Their difference isolates guidance within this pilot. Next versus either baseline combines CLI and guidance effects. Three repeats do not establish statistical significance.
- API calls count agent-side signed fixture requests, including bootstrap/metadata when issued; they are not real Feishu network calls. Discovery verifier-only requests are excluded.
- Failure categories are harness classifications. request_outside_fixture alone does not establish whether a product defect, model mistake or unmodeled valid route caused rejection. misleading_success_claim means a completed agent answer was not verified, not a judgment of intent. Runtime scoring used independent state/artifact verifiers; the later audit recomputes retained scores and accounting, and cannot replay discarded workspaces or recover omitted payloads.
- Counts cover only the measured agent attempts; fixture development and reviewer conversations are outside the experiment. Fresh context does not mean an empty provider prompt cache.
- These small fixed workloads do not establish large-document, large-table, large-file or live-tenant performance.
- Provider input tokens include cached input in full; output tokens are added once. Cache counts are reported separately in summary.json. No byte-based token estimates or cached-token discounts are substituted for usage.
- The product builds, API catalog, entry guidance and evaluator are frozen by revision and SHA-256 in environment.json. The requested model name may be a provider alias; no immutable model snapshot is recorded. Codex consumes existing authentication internally; no credential contents, private tenant content or raw model transcripts are published.
- Preflight: 292 isolated harness tests (271 passed, 21 optional skips), 17 final launcher tests, and separate two-binary task controls. The independent readiness review approved evaluator 596373b before launch.
- Native sandbox controls verified signed local fixture access and denied external sockets through both binaries before trials. Synthetic task controls covered both binaries. Fixtures are bounded approximations, not complete Feishu implementations.
- Sheets default/formatted rendering and additional inspection tools lack a verified native fixture contract. Such strategies are inconclusive when classified as unsupported. Explicit raw and formula responses come from separate reads at the same revision. Base positive mutation uses numeric zero and explicit JSON null; empty-string UPDATE is unverified.
- Controlled Contact checks found that Next raw POST users/basic_batch exits 22 after a successful read; it remains a failed alternative. Shortcut, generated profile and raw GET controls pass both binaries.
- Replacing a formatted Docs span with plain text drops its styles; explicit rich content or edits outside that styled span can preserve them. Markdown conversion covers the tested heading, bold, link, table, Unicode and code subset; Markdown-format and scoped fetch readbacks are unmodeled. Drive measures a 4 KiB roundtrip, not large-file scaling.
- Discovery validates a returned read-only command after the agent exits; verifier-only API requests are not model API calls. Old exposed examples remain quarantined, and this pilot does not claim full blinding or held-out generalization. Historical evidence is unchanged.

## Reproducibility

[Frozen environment](environment.json), [machine-readable summary](summary.json), [independent result audit](result-review.json), and per-section checkpoints retain all attempted trial results, usage diagnostics and sanitized command choices. The manifest is evaluation/tasks.paired.json at the evaluator revision. No failed run was removed or overwritten.
