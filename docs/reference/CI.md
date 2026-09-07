# CI capability reconciliation

Run the real next-mode command registration guard after the pinned metadata build:

```sh
python3 scripts/next_inventory.py gate --target . --out .tmp/next-capability-gate.json
```

The guard exports both configured Feishu and Lark surfaces with the existing temporary-copy exporter and the frozen metadata. It executes no business commands, uses a synthetic environment account, and disables remote metadata and outbound proxy access. No live credentials are needed. The normal Go dependency cache must already be populated.

Exit 0 means interface reconciliation passed; exit 1 means a capability mismatch; exit 2 means a handled collection or mapping error. With a writable output path, the requested JSON report records these outcomes, including handled exporter errors. Malformed local mapping structures and output-write failures may terminate without that report. Successful collection retains the full original report, every changed reference source hash, and its original `review_required` status. `capability_gate.status` is a separate result. This guard grants no source-semantic, native-platform, model-evaluation, or live verification approval.

The guard rejects missing reference commands, shortcuts, event keys, source files, raw API, changed existing command flags, changed brand guards, mixed tracked source snapshots, and unrecognized interface changes. It also rejects removal or modification of previously mapped next additions. Both configured brands and the pinned reference/metadata hashes are required.

`next-interface-mapping.json.gz` is an exact value mapping from immutable candidate `10588248080a91412b0718eb0aa921cff6a90f10`, using the configured-brand reference. It contains 60 changed metadata leaves (30 per brand) and 5,682 added dictionary contracts. Every changed leaf binds both its reference and candidate canonical JSON SHA256; every addition binds its complete new value, including new command flag/schema subtrees. There are no prefix exceptions. The many inherited flag entries are compressed data, not separate hand-written policy rules.

The mapped changes are documented in `p3-post-mail-interface-mapping.json`, `P1.md`, and `P3.md`: 13 new command nodes and three shortcut schemas per brand; explicit profile/result output controls; the ten opt-in skip-unchanged shortcuts and their conditional scopes; three validation declarations; and five exact generated POST read annotations. The reference catalog risk remains unchanged. The mapping capture was independently checked against the existing immutable reconciliation's complete difference and addition lists.

An intentional interface change requires reviewing its exact old/new registration values and updating this mapping at an immutable source revision, with an explanation and focused evidence. A passing gate must never be obtained by accepting all paths from a failing current report. Source changes still need their separate exact-hash semantic review ledger, even when this interface guard passes. The original frozen snapshot and configured-brand supplement remain untouched.

Controlled tests:

```sh
python3 -m unittest discover -s scripts -p 'next_inventory_test.py'
```

These cover exact value matching, changed new-subtree definitions, missing capabilities, existing flags and brand guards that cannot be waived by a mapping, wrong capture context, mixed snapshots, source-review separation, and JSON report/exit behavior on success, rejection, and collection errors.

The local actual-process checkpoint `ci-capability-checkpoint.json.gz` passed on immutable product revision `8d92e5253aa9d0582314180d51cdec1b10778897` with both configured brands, 951 candidate command nodes and 538 shortcuts per brand, all 938 reference nodes and 535 shortcuts retained, and 25 event keys retained. Its 5,742 interface deltas exactly match the mapping. This is local registration execution, not execution of the remote CI matrix. The preceding shared-checkout run correctly rejected a mixed tracked snapshot; no interface exception was added for that rejection. The complete inventory/review-aggregation test selection passed 26 tests (18 inventory and eight review aggregation).
