# Second token-efficiency pass

This is a new balanced condition after the immutable
[first improvement results](../../evidence/evaluation/token-efficiency-02/REPORT.md): four of
eight functions met the combined gate, and one candidate document edit failed.
The second pass improves existing command guidance for precise document edits,
single known Base records, validated Sheets exports, and Drive transfers.
Original commands, formats, skills, error semantics and verification stay available.

The comparison and accounting rules in [condition 02](token-efficiency-02.md)
apply unchanged: the same eight task prompts, three repetitions and three arms
(72 attempts), original `7fd6ef3`, previous Next `7d28655`, and a newly frozen
candidate. Use GPT-5.6 Luna with medium reasoning through native Codex CLI
0.153.4, fresh contexts, serial randomized ordering, and the existing timeouts.
Both baselines rerun against the same evaluator as the candidate.

The primary gate remains at least **50% fewer all-attempt tokens per verified
success than the initial frozen previous Next, separately for every function**,
with no observed success regression against either baseline. Report original
CLI comparisons separately. All failures and recovery costs count; unknown
usage, inconclusive coverage and zero successes retain the limitations declared
in condition 02. Never pool functions to declare this target achieved.

Native observations may expand Docs fixture coverage and add finite diagnostic
reasons before freezing this condition. Correctness checks stay authoritative;
unobserved combinations remain unsupported. This does not rescore condition 02.
Record evaluator, product, binaries, guidance, manifest and observation hashes
and pass isolated/native controls plus independent readiness review before
launch. Do not modify the candidate or evaluator during the batch.

Retain all outcomes, including missed targets. Three repetitions are development
evidence, not statistical proof or comprehensive live functionality coverage.
