# Full-matrix interpretation limits

Read these qualifications alongside the [matrix report](REPORT.md). The completed domain-read subset contains 72 attempts across 12 tasks. Its retained verifier outcomes include strict fixture-route failures, so the success counts do not directly measure live CLI reliability or lost functionality. The [diagnostic evidence](read-regression-diagnostics.json) records all 72 rows, source hashes and bounded trace findings. No outcome or provider cost was changed.

| Domain-read outcome | Original | Next |
| --- | ---: | ---: |
| Verified / attempts | 30/36 | 26/36 |
| Correct typed answer, rejected request outside fixture | 1 | 9 |
| Typed answer mismatch | 5 | 1 |

All ten correct-answer request failures also included the required successful read groups. The read verifier checks the typed answer before the request ledger; any rejected or unmodeled request still fails the attempt after successful recovery. These remain failures in the frozen results, not newly credited successes or retrospectively classified inconclusive attempts. The machine category `misleading_success_claim` does not by itself establish a wrong final answer or deceptive intent.

The Wiki fixture lacks native document metadata and block responses needed by `docs +pull`. All three Next Wiki attempts tried that route and then recovered through supported reads; this demonstrates a fixture coverage limit, not a live `+pull` failure. Calendar, IM and VC also omit selected search/list alternatives. However, retained traces omit argument values and raw request bodies, so command availability does not prove every attempted selector was valid. The Note attempt may reflect mistaken domain routing rather than a valid alternative. For typed answer mismatches, the differing field and causal source are unavailable.

Every failed attempt and recovery cost remains included. Synthetic fixture rejections can add recovery work and lower verified-success counts, biasing tokens-per-success comparisons; the data cannot isolate that effect or allocate tokens precisely to each command. These findings support neither universal savings nor a claim that the CLI has lost the affected functions. Other matrix suites retain their own fixture and accounting limits. Historical condition 09 remains separately identified as reused evidence; exhaustive live parity and held-out release acceptance remain open.
