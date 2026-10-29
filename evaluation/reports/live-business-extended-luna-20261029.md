# Extended business evaluation: official CLI vs Next

Six additional business workflows, three repeats per CLI, using **Codex GPT-6 Luna (`codex/gpt-6-luna`), medium reasoning, through Magpie** against real Feishu. Each attempt used fresh private `[TEST]` resources and the same 480-second agent budget. Native readback independently checked complete outcomes and unrelated state preservation.

Native task checks: official **17/18**, Next **17/18**. All-attempt tokens per verified native task: official **454,324**, Next **317,858**; observed Next reduction **30.0%**. The recorded run-status metric below additionally retains a controller cleanup failure.

| Workflow | Official native task success | Next native task success | Official tokens/task success | Next tokens/task success | Next token reduction |
|---|---:|---:|---:|---:|---:|
| business-base-audit-large | 3/3 | 3/3 | 280,881 | 222,930 | 20.6% |
| business-base-conditional-close | 3/3 | 3/3 | 265,340 | 286,180 | -7.9% |
| business-budget-close | 3/3 | 3/3 | 907,600 | 809,070 | 10.9% |
| business-document-audit-large | 3/3 | 2/3 | 249,829 | 308,911 | -23.6% |
| business-sheet-export | 2/3 | 3/3 | 725,202 | 87,781 | 87.9% |
| business-sheet-intake | 3/3 | 3/3 | 387,384 | 189,292 | 51.1% |

Next used more tokens per verified task for conditional Base closing (7.9%) and the large-document audit (23.6%). The aggregate reduction does not imply improvement on every workflow.

Recorded end-to-end success: official **17/18**, Next **16/18**. Provider accounting complete for **36/36** attempts. All attempted token costs, including failures, are retained.

Total known tokens: official **7,723,510**, Next **5,403,579**. Tokens per recorded end-to-end success: official **454,324**, Next **337,724**. Observed Next reduction for recorded run successes: **25.7%**.

One Next settlement passed all independent task checks but its trusted controller cleanup timed out. The raw attempt remains failed (`live_cleanup_unconfirmed`) and its cost is included above. Repeated exact-ID metadata/ACL reads established the folder remained present before cleanup continued; final cleanup was confirmed. This is a controller lifecycle failure, recorded separately from the official export artifact failure. Native agent task checks therefore passed in 17/18 Next and 17/18 official trials. The assessment also derives task-only token cost per success, without changing raw flags.

Input includes cached input once; total tokens equal input plus output. This measures token volume, not money. Setup, calibration, verification and cleanup are controller work and excluded from agent token measurements.

The larger audit crosses 2,000 Base records; the document audit crosses 500 native blocks. Other cases check selective Base changes, quoted multiline Unicode CSV with exact codes/approval flags, raw Sheet types/formulas, and a settlement that changes Base, Sheet and document together.

One official export trial failed: ten numeric/boolean literal cells were emitted as formatted strings instead of raw JSON numbers/booleans (for example zero as "0" and false as "FALSE"). The remote workbook was unchanged. One Next document audit found all 107 exact matching IDs/codes and the correct total magnitude, but emitted total_amount as JSON string "1006" instead of numeric 1006; the native document was unchanged. Both task failures and their full provider token costs are retained; no model trial was replayed. See failure-analysis.json in the evidence directory.

All seven calibration records are retained. The initial CSV candidate applied the right values but caused a native column-clipping diagnostic to change. Before freezing, a fresh fixture widened its note column; the complete native-cell comparison then passed. No model trial was discarded or retried for this correction.

One additional fixture setup stopped before the model launched because Base returned revision 1 and then revision 2 while reading freshly seeded pages. The empty agent workspace and absent verifier configuration establish that the adapter was not reached. A later read-only snapshot matched all 205 seed rows; the six resources were cleaned, original markers retained, and the same unbilled schedule slot resumed with fresh fixtures. No mutation was replayed and the evaluator was unchanged. This setup interruption is separate from the 36 billed model attempts.

Cleanup: all **121 registered resources**, including attempt/calibration resources and the test parent were confirmed deleted by exact ID after owner/ACL checks. No unresolved mutation markers remain; temporary model authentication was removed. No new permission consent was requested.

Validation: `make evaluation-test` and `make script-test` passed. The evaluation suite ran 124 tests: 123 passed, one platform-specific native Codex isolation test skipped. No Go, command, flag, schema or shipped skill changes were made.

Three repeats per case are a small development sample. The workflow mix differs from the earlier four-case evaluation, so its aggregate is reported separately. Production adapter API-call/retry totals are unavailable. Task writes, Calendar/Mail, delivery, sharing and broad Drive listing were outside this experiment.

Evidence: [assessment](../../evidence/evaluation/live-business-extended-luna-20261029/assessment.json), [all raw attempts](../../evidence/evaluation/live-business-extended-luna-20261029/attempts.raw.json), [provenance](../../evidence/evaluation/live-business-extended-luna-20261029/provenance.json), and [frozen controller](../../evidence/evaluation/live-business-extended-luna-20261029/controller.tar.gz).
