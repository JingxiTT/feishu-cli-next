# Full CLI evaluation matrix

Status: **complete**; 258/258 attempts recorded.

GPT-5.6 Luna / medium; native Codex CLI 0.153.4; original/full versus Next; three repetitions per task and arm; filesystem-only guidance.

New observed provider usage: **31,650,572 tokens**. Reused usage: **901,686 tokens** (not new spend). Unknown usage attempts: 6.

| Suite | Task | Original verified | Next verified | Token saving/success | Gate | Evidence |
| --- | --- | ---: | ---: | ---: | --- | --- |
| paired | paired-discovery | 3/3 | 3/3 | -11.9% | missed | New |
| paired | paired-contact | 3/3 | 3/3 | 8.8% | missed | New |
| paired | paired-docs-read | 3/3 | 3/3 | 39.2% | missed | New |
| paired | paired-docs-edit | 3/3 | 3/3 | 52.8% | met | New |
| paired | paired-markdown | 3/3 | 3/3 | 60.4% | met | New |
| paired | paired-sheets | 1/3 | 3/3 | Unavailable | unavailable | New |
| paired | paired-base | 1/3 | 3/3 | 81.9% | met | New |
| paired | paired-drive | 3/3 | 3/3 | 65.3% | met | New |
| docs-hard | docs-hard-rich-read | 3/3 | 3/3 | 38.2% | missed | Reused |
| docs-hard | docs-hard-focused-read | 3/3 | 3/3 | 31.1% | missed | Reused |
| docs-hard | docs-hard-multi-edit | 2/3; 1 inconclusive | 3/3 | Unavailable | unavailable | Reused |
| read | application-dev-01 | 1/3 | 2/3 | -71.3% | missed | New |
| read | apps-dev-01 | 3/3 | 3/3 | -92.0% | missed | New |
| read | im-dev-01 | 2/3 | 1/3 | -87.0% | missed | New |
| read | minutes-dev-01 | 3/3 | 3/3 | 12.3% | missed | New |
| read | note-dev-01 | 3/3 | 2/3 | -89.5% | missed | New |
| read | slides-dev-01 | 3/3 | 3/3 | -18.9% | missed | New |
| read | vc-dev-01 | 2/3 | 2/3 | 26.5% | missed | New |
| read | whiteboard-dev-01 | 3/3 | 3/3 | 30.8% | met | New |
| read | wiki-dev-01 | 1/3 | 0/3 | Unavailable | missed | New |
| read | calendar-dev-01 | 3/3 | 1/3 | -443.9% | missed | New |
| read | mail-dev-01 | 3/3 | 3/3 | 26.7% | missed | New |
| read | okr-dev-01 | 3/3 | 3/3 | -28.4% | missed | New |
| scenario | base-dev-01 | 3/3 | 3/3 | 60.2% | met | New |
| scenario | docs-dev-01 | 3/3 | 3/3 | 55.7% | met | New |
| scenario | drive-dev-01 | 3/3 | 3/3 | 62.7% | met | New |
| scenario | events-dev-01 | 1/3; 1 inconclusive | 3/3 | Unavailable | unavailable | New |
| scenario | extensions-dev-01 | 1/3 | 1/3 | Unavailable | unavailable | New |
| scenario | markdown-dev-01 | 3/3 | 3/3 | 50.8% | met | New |
| scenario | packaging-dev-01 | 3/3 | 3/3 | 36.5% | met | New |
| scenario | sheets-dev-01 | 3/3 | 1/3 | -142.4% | missed | New |
| scenario | task-dev-01 | 3/3 | 3/3 | -51.3% | missed | New |
| scaling | base-scale-dev-00100 | 0/3 | 3/3 | Unavailable | unavailable | New |
| scaling | base-scale-dev-01000 | 0/3 | 2/3 | Unavailable | unavailable | New |
| scaling | base-scale-dev-10000 | 0/3 | 3/3 | Unavailable | unavailable | New |
| scaling | sheets-size-dev-100 | 3/3 | 3/3 | 40.6% | met | New |
| scaling | sheets-size-dev-1000 | 3/3 | 1/3 | Unavailable | missed | New |
| scaling | sheets-size-dev-10000 | 3/3 | 2/3 | -8.8% | missed | New |
| business | business-dev-approval-flow | 3/3 | 3/3 | 30.2% | met | New |
| business | business-dev-mindnote-edit | 3/3 | 2/3 | -146.5% | missed | New |
| foundation | foundation-unconfigured-status | 3/3 | 3/3 | 5.6% | missed | New |
| foundation | foundation-document-discovery | 0/3 | 0/3 | Unavailable | unavailable | New |
| contact | contact-dev-01 | 3/3 | 3/3 | 54.3% | met | New |

## Whole-matrix accounting

| Arm | Verified / attempts | Inconclusive | Observed tokens | Complete usage |
| --- | ---: | ---: | ---: | --- |
| next | 107/129 | 0 | 14,954,750 | False |
| official | 102/129 | 2 | 17,597,508 | False |

Whole-matrix tokens-per-verified-success saving: **unavailable**.

Secondary raw token-spend reduction: **unavailable**. This compares all attempted work, without correcting for success; it cannot establish an efficiency gate or replace unavailable primary comparisons.

All failed, inconclusive and recovery costs remain included. Cached input is counted once in full. Missing accounting, inconclusive outcomes or zero verified successes make affected savings unavailable. Negative savings mean more tokens.

Docs require strictly more than 50% saving; other tasks require at least 30%, with observed success nonregression. Three repetitions do not establish statistical success parity. Historical reuse is noncontemporaneous; the provider model alias has no immutable snapshot.

Coverage includes every implemented development matrix: paired functions, harder Docs, domain reads, workflow scenarios, scaling, business, foundation and Contact. The development/held-out inventory stubs are not executable additional trials; held-out release acceptance and live-tenant parity remain separate.

[Exact results and evidence hashes](summary.json) · [Frozen inputs](environment.json) · [Execution plan](plan.json)

[Interpretation limits and qualifications](QUALIFICATIONS.md)
