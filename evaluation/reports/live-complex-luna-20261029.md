# Complex live workflows: official CLI vs Next

> Current scope update: IM/group-chat evaluation cases, fixture code and dispatcher
> routes were removed at the user’s request after these measurements. IM status
> below describes the recorded run; sealed evidence and results are unchanged.

**GPT-6 Luna (`codex/gpt-6-luna`), medium reasoning, through Magpie.** Three repeats per CLI across three workflows, **18 measured attempts**, the same frozen production binaries/file guidance as prior runs, and 480 seconds per attempt. Fresh private `[TEST]` resources and independent native Feishu readback; no simulator.

Observed success: official **8/9**, Next **8/9**. Complete known tokens: official **4,430,828**, Next **3,939,480**. All-attempt tokens per successful run: official **553,854**, Next **492,435**. Observed Next reduction: **11.1%**.

| Workflow | Official success | Next success | Official tokens/success | Next tokens/success | Next reduction |
|---|---:|---:|---:|---:|---:|
| Base conflict reconciliation | 3/3 | 3/3 | 382,805 | 409,080 | -6.9% |
| Long-document closeout | 2/3 | 2/3 | 1,008,528 | 971,838 | 3.6% |
| Versioned FX journal | 3/3 | 3/3 | 421,786 | 256,189 | 39.3% |

The Sheet case resolves 34 shuffled journal entries into 24 event IDs: latest revisions, duplicate copies, voids, missing codes and original boolean approvals, with per-event decimal FX rounding before aggregation. It checks numeric closing amounts, exact report classifications, all cell styles/formulas and untouched guard cells.

The Base case reconciles 79 versioned/duplicated adjustment rows against all 205 original records. Original-value conflicts take precedence over approval gates; null clears notes, and zero/negative amounts remain numeric. It checks every final row, stable IDs, all skipped data and the full-table total.

The document case selects approved positive R42 invoices across 520 invoice paragraphs plus rich-text guards, updates status in place and appends exactly one computed summary. It checks every native block ID, content, order, rich formatting and link, including continuation pages beyond 500 blocks.

Observed failures: next / business-document-closeout: Report selected six zero-amount invoices despite the strictly-positive condition: 113 reported versus 107 required. The sum still equalled 1006, so checking totals alone would miss the error. official / business-document-closeout: Report selected 6 zero-amount invoices despite the strictly-positive condition: 113 reported versus 107 required. The sum still equalled 1006, so checking totals alone would miss the error.
For the failed reports, post-edit content preservation was not assessed because the frozen checker stops at report mismatches. Cleanup still independently verified ownership and private ACL before exact-ID deletion.

**Verifier correction:** the frozen checker rejected document results because Feishu split a text run into adjacent runs with identical formatting. Complete independent post-model readback establishes identical text, style/link metadata, native IDs, unrelated blocks and root order. `readback-reassessment.json` applies the same narrowly corrected rule to all document trials, without rerunning models or tenant actions. The table uses those native task outcomes; all original raw results and costs remain unchanged. Raw accepted runs were official 7/9 and Next 7/9. The two actual zero-amount selection failures remain failures.

**IM and group tests are implemented, but live evaluation is blocked.** The configured peer was independently reverified. Feishu rejected the first DM seed message with `230027` (user authorization); zero messages were created. The bounded TEST-group search found a candidate containing an unconfigured bot, so it was not read or messaged. No group was created and no new scopes were requested. Recipient, tenant and chat identifiers remain in ignored owner-only local config. These blocked cases have no model usage or measured success and are excluded from the 18-trial sample.

Four earlier adapter attempts failed deterministically before provider subprocess launch because the first evaluator snapshot omitted the required command registry. They remain in `adapter-attempts.raw.json` with their original null accounting, and all fixtures were cleaned. The corrected snapshot includes and validates that public registry. Its 18-trial results are reported separately; no overall efficiency ratio pools known and unknown accounting.

Calibration is retained separately: Sheets and Docs passed initially; the Base calibration used an incorrect HTTP method, received a native invalid-route rejection, and had its entire table independently confirmed unchanged before a corrected POST passed. The rejection was reconciled as confirmed unapplied; no unknown mutation was replayed.

Cleanup confirmed deletion of **51 registered resources**, including all calibration/adapter-failure/trial objects and the test parent. No reconciliation/in-flight markers remain. Temporary model authentication was removed. All-attempt costs in the corrected sample include failures; cached input is counted once inside input tokens. Controller work is excluded.

Checks: `make evaluation-test`, `make script-test`, layout/seal, diff and private-value scans. One platform-specific evaluator check remains skipped. No Go, CLI surface, dependency or shipped-skill change was made.

This is a small descriptive development sample. Keep it separate from previous workload mixes. It adds three measured live workflow types (19 total across the recorded contact/business/collaboration suites); two IM/group types remain blocked. These results do not establish full CLI parity, complete business coverage, API retry rates or monetary savings.

Evidence: [assessment](../../evidence/evaluation/live-complex-luna-20261029/assessment.json), [measured attempts](../../evidence/evaluation/live-complex-luna-20261029/attempts.raw.json), [adapter failures](../../evidence/evaluation/live-complex-luna-20261029/adapter-attempts.raw.json), [calibration](../../evidence/evaluation/live-complex-luna-20261029/calibration.json), [native reassessment](../../evidence/evaluation/live-complex-luna-20261029/readback-reassessment.json), [blocked IM](../../evidence/evaluation/live-complex-luna-20261029/blocked-im.json), and [frozen evaluator](../../evidence/evaluation/live-complex-luna-20261029/controller.tar.gz).
