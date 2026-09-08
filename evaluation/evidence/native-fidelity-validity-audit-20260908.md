# Development evaluation validity amendment — 2026-09-08

New private live controls contradict backend behavior assumed by the Docs, Sheets and Base scaling fixtures. The recorded provider costs and synthetic verdicts remain historical observations; these conditions must not support claims of equal live fidelity or cost per successfully preserved live outcome. This amendment does not retroactively assign individual live failures or replace any raw result.

Scope: development fixtures, listed retained batches and sanitized command choices only. No held-out content, raw private artifacts, new model trials, live calls or evaluator/product semantic changes were used in this audit. Live observations below were supplied by the section owners and coordinator; in-memory fixture probes were performed independently. These live controls are not a live comparison of both historical binaries.

## Docs: plain replacement inherits formatting only in the fixture

The [Docs fixture](../scenarios/docs.py) parses plain XML content without child tags as non-explicit markup. Its `_replace` branch retains the matching prefix/suffix of the old native runs and replaces only differing characters. Consequently, replacing `Old status` with `New status` preserves the old bold/link on `status`. The independent verifier checks that resulting mock state correctly, but cannot establish that this backend transition is real. The [contract](../contracts/docs.json) and [tests](../../tests/next_evaluation/test_scenario_docs.py) encode the same assumption. Both historical fixture versions contain it.

The owner’s live control instead replaced the entire matching span: plain `New status` returned success with no warnings, but native readback lost bold and link. A separate lifecycle using explicit `New <a href="<same-link>"><b>status</b></a>` preserved native bold, the same link URL, Unicode and unrelated blocks. Both resources were independently confirmed deleted. This agrees with the source forwarder’s lack of a plain-content style-inheritance promise.

Fresh in-memory fixture probes reproduce the mismatch: plain `New status` verifies; `<span>New status</span>` loses styles and fails verification; explicit bold/link XML verifies. These are mock controls, not additional live evidence.

Retained corrected Docs traces show three concise-arm direct update commands and one official-arm direct update; the other five use generic API compositions. Earlier conditions also contain both forms. The sanitizer discarded replacement values, operation values and API bodies. A direct update is compatible with the problematic strategy but does not prove its use; a generic API choice does not establish that native typed PATCH was used. No individual historical verdict can be reconstructed reliably. The observed Next token regression remains a historical synthetic result, not evidence of a live fidelity advantage for either arm.

## Sheets: one render mode does not establish typed values plus formulas

Both [small Sheets](../scenarios/sheets.py) and [size Sheets](../scaling/sheets.py) copy formula fields into `raw_value` responses. Their `formula` mode changes the formula cell’s value to formula text while retaining ordinary numeric, boolean and empty-string types. Their final verifiers require both typed ordinary values and explicit formulas. The [small contract](../contracts/sheets.json) explicitly describes this rendering assumption.

The owner/coordinator’s same-range native control found a different separation: `raw_value` preserved numeric zero, boolean false, empty text and leading-zero text, but the formula cell contained its numeric result and no formula. Formula rendering exposed the explicit formula while ordinary zero/false became display strings and the empty cell became an empty object. The raw artifact hash was `c7a14a637eba98ecf61c1f9e6ffad0760b7351db30e37291daae17f3d8226b77`; the formula-read hash was `0a60e9161803eb97f36f9d01095041ba5d89b3a4899e3df0ab4d5c89f6a72276`. Raw files were not opened by this audit.

In-memory probes confirm both synthetic renderers retain formulas in raw mode and ordinary scalar types in formula mode. Thus the limitation affects both the original `+cells-get --include …` strategy and Next export, not only Next. Retained Next traces sometimes identify only the `sheets` prefix because the frozen command vocabulary lacks newer leaves; render arguments and output payloads are unavailable. A future paired-read implementation would have different work and consistency requirements. Historical token/API counts cannot be advertised as measurements of that implementation.

## Base: empty text is stored unchanged only by the synthetic create model

The [Base size generator](../scaling/base.py) includes a text-typed empty string in every row. Its create handler copies submitted values directly into mock records; its verifier requires exact empty-string preservation. The owner’s native Base-v3 batch-create/readback control instead returned `null` for submitted empty text while preserving Unicode, zero, false and `1.25`. Both synthetic live resources were deleted. See the sanitized [live evidence](../../docs/sections/base-private-live-evidence.json), committed at `a6c570e`.

This directly undermines the size condition’s assumed create/readback fidelity for every size. The [small Base task](../scenarios/base.py) likewise requires and stores empty text unchanged, but uses PATCH/batch-update. The live control did not exercise updates. Its empty-text fidelity is therefore unvalidated and at risk; the create observation alone does not prove update normalization. The Base import trials start from supplied local data, so they do not depend on the Sheets renderer, but their own empty-text assumption independently prevents a validated live round-trip claim.

## Historical records remain unchanged

“Verified” below means the original synthetic verifier result, not a newly validated live outcome. Known tokens include cached input in full and all known unsuccessful attempt costs. Conditions are listed separately, not pooled into a performance score.

| Retained condition | Starts / originally verified | Known provider tokens | Unknown totals |
|---|---:|---:|---:|
| [Docs initial](scenarios-8d92e5/docs/batch-01.json) | 1 / 1 | 75,555 | 0 |
| [Docs schema02](scenarios-8d-schema02/docs/batch-01.json) | 9 / 8 | 1,620,033 | 0 |
| [Docs corrected03](scenarios-8d-corrected03/docs/batch-01.json) | 9 / 9 | 1,332,023 | 0 |
| [Sheets small](scenarios-8d-schema02/sheets/batch-01.json) | 9 / 9 | 1,539,955 | 0 |
| [Sheets size archive04](scaling-8d-archive04/sheets/batch-01.json) | 27 / 26 | 5,531,972 | 1 |
| [Base small](scenarios-8d-schema02/base/batch-01.json) | 9 / 9 | 927,306 | 0 |
| [Base size archive04](scaling-8d-archive04/base/batch-01.json) | 27 / 25 | 4,248,697 | 0 |

Earlier Base preparations also remain: [initial schema failures](scenarios-8d92e5/base/batch-01.json), nine starts with unknown usage; [size schema02](scaling-8d-schema02/base/batch-01.json), seven starts, one synthetic verification and 1,258,267 known tokens; [size fixture03](scaling-8d-fixture03/base/batch-01.json), two starts, no verification, 237,389 known tokens and one unknown total. Their previously diagnosed preparation/coverage failures must not be reassigned to the new native-fidelity findings. Canceled-before-model slots are not extra trials.

Affected reports, the size plot and aggregate assessment need a linked validity notice. Plot points remain actual provider costs, while their success markers denote the historical synthetic checks. Withhold live same-fidelity efficiency and scaling conclusions for the affected conditions; retain all costs, unknowns, regressions and original machine-readable fields. The small Base update question should remain explicitly unresolved. These findings do not by themselves invalidate unrelated domain conditions, Docs read-only retrieval, Wiki resolution or Markdown creation.

Any future correction should first match source-backed native representations with symmetric controlled tests, preserve the task’s intended fidelity requirements, and receive a new fixture/product condition. Changing empty text to null or dropping formulas merely to pass would change the requested semantics. No paid reruns or semantic corrections are authorized by this audit itself.
