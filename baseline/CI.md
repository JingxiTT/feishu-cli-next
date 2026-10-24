# CI capability reconciliation

Run the real next-mode command registration guard after the pinned metadata build.
CI and AGENTS.md select the v1.0.97 baseline explicitly; without
`--reference-dir` and `--mapping` the guard falls back to the original frozen
reference described in the historical notes below.

```sh
python3 scripts/next_inventory.py gate --target . \
  --reference-dir baseline/v1.0.97 \
  --mapping baseline/v1.0.97/next-interface-mapping.json.gz \
  --out .tmp/capability-gate.json
```

The guard exports both configured Feishu and Lark surfaces with the existing temporary-copy exporter and the frozen metadata. It selects an exact reference hash from `configured-brand-platforms.json` for the actual Go target/host OS; only Darwin, Linux, and Windows are supported, and a target/host OS mismatch fails closed. It executes no business commands, uses a synthetic environment account, and disables remote metadata and outbound proxy access. No live credentials are needed. The normal Go dependency cache must already be populated.

Exit 0 means interface reconciliation passed; exit 1 means a capability mismatch; exit 2 means a handled collection or mapping error. With a writable output path, the requested JSON report records these outcomes, including handled exporter errors. Malformed local mapping structures and output-write failures may terminate without that report. Successful collection retains the full original report, every changed reference source hash, and its original `review_required` status. `capability_gate.status` is a separate result. This guard grants no source-semantic, native-platform, model-evaluation, or live verification approval.

The guard rejects missing reference commands, shortcuts, event keys, source files, raw API, changed existing command flags, changed brand guards, mixed tracked source snapshots, and unrecognized interface changes. It also rejects removal or modification of previously mapped next additions. Both configured brands and the pinned reference/metadata hashes are required.

`next-interface-mapping.json.gz` originally captured immutable candidate `10588248080a91412b0718eb0aa921cff6a90f10`, using the configured-brand reference: 60 changed metadata leaves (30 per brand) and 5,682 added dictionary contracts. Those 5,742 rows remain unchanged. The current mapping adds ten exact registrations captured at `b1cddcb82bbeba4220fad57b07de1d1b03dc4617`: the opt-in `schema --input-only` flag and the existing `docs +pull`/`docs +push` command and shortcut schemas, for both brands. [The addition review](../evidence/reference/reviews/agent-efficiency-interface-additions.json) retains their complete values, checksums, rationale and focused test pointers. Every changed leaf binds both its reference and candidate canonical JSON SHA256; every addition binds its complete new value, including new command flag/schema subtrees. There are no prefix exceptions. The many inherited flag entries are compressed data, not separate hand-written policy rules.

The mapped changes are documented in `p3-post-mail-interface-mapping.json`, `P1.md`, and `P3.md`: 13 new command nodes and three shortcut schemas per brand; explicit profile/result output controls; the ten opt-in skip-unchanged shortcuts and their conditional scopes; three validation declarations; and five exact generated POST read annotations. The reference catalog risk remains unchanged. The mapping capture was independently checked against the existing immutable reconciliation's complete difference and addition lists.

An intentional interface change requires reviewing its exact old/new registration values and updating this mapping at an immutable source revision, with an explanation and focused evidence. A passing gate must never be obtained by accepting all paths from a failing current report. Source changes still need their separate exact-hash semantic review ledger, even when this interface guard passes. The original frozen snapshot and configured-brand supplement remain untouched.

Controlled tests:

```sh
python3 -m unittest discover -s scripts -p 'next_inventory_test.py'
```

These cover exact value matching, changed new-subtree definitions, missing capabilities, existing flags and brand guards that cannot be waived by a mapping, wrong capture context, mixed snapshots, source-review separation, and JSON report/exit behavior on success, rejection, and collection errors.

The local actual-process checkpoint `ci-capability-checkpoint.json.gz` passed on immutable product revision `8d92e5253aa9d0582314180d51cdec1b10778897` with both configured brands, 951 candidate command nodes and 538 shortcuts per brand, all 938 reference nodes and 535 shortcuts retained, and 25 event keys retained. Its 5,742 interface deltas exactly match the mapping. This is local registration execution, not execution of the remote CI matrix. The preceding shared-checkout run correctly rejected a mixed tracked snapshot; no interface exception was added for that rejection. The complete inventory/review-aggregation test selection passed 26 tests (18 inventory and eight review aggregation).

The Linux-only `.github/workflows/next-offline.yml` runs this
actual guard as the required `capabilities` stage after dependency preloading
and the frozen metadata build. It uses the exporter's temporary-copy and
synthetic-account isolation directly, without invoking business commands. The
step retains the guard's nonzero exit, so mismatches and collection failures fail
the job. The always-written `gate.json` records its step outcome, and the always
uploaded `.tmp/next-ci/` artifact includes `capabilities.json` when the guard
could write it. The source report's `review_required` status remains unchanged;
only `capability_gate.status` determines interface reconciliation success.
A workflow definition is not evidence of a completed native CI run.

## Platform reference boundary

The original configured reference was executed on Darwin. On Linux, its `config keychain-downgrade` risk annotation produced a false mismatch: Darwin registers a supported write operation, while the unchanged `!darwin` file registers an unsupported stub with an empty risk annotation. Both files remain byte-identical to frozen reference `7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`.

`configured-brand-registrations-linux.json.gz` is a separate actual Linux/arm64 reference capture, using the frozen metadata and reference source in the same offline test image as candidate `8ce0688b303eb646e1b79a329002dddcb6403be4`. Its exhaustive difference from the original Darwin reference is exactly those two risk values, one per configured brand. The Linux candidate comparison contains the original 5,742 mapped interface deltas, and the unchanged mapping predicate passes when given the Linux reference's exact hash. See `configured-brand-context-linux.json`, `configured-brand-linux-evidence.json`, and the full `ci-capability-linux-checkpoint.json.gz` report. The original capture, original checkpoint, and all 5,742 intentional mapping rows remain unchanged.

The Windows profile currently reuses the Linux registration baseline based on the same `!darwin` command source. It is explicitly **source-derived; native Windows reference capture is not run**. A Windows candidate passing this guard establishes comparison against that baseline only. Full Windows acceptance still requires an actual Windows reference capture and candidate comparison; neither Linux execution nor shared source proves it. Unknown OSes have no fallback. Target/host OS and architecture are recorded separately in `build_context`; OS matching does not assert physical native architecture.

The final committed-selector CLI was then executed on Linux/arm64 against unchanged candidate `8ce0688`, using helper `cb3a750`. It exited 0 in 23.127 seconds, retained all 5,742 approved deltas, and kept the full checker's `review_required` source status. `ci-capability-linux-gate.json.gz` preserves that complete final report; `ci-capability-linux-gate-evidence.json` binds the helper/profile hashes, command outcome, raw logs, and identical before/after candidate tree and binary hashes. This closure replaces neither the earlier failed setup diagnostics nor their historical reports.
