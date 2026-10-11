# Official v1.0.97 sync baseline

The fork integrated official release `v1.0.97`, commit
`72579c80027c863ca51d5f9affda72a70ab0d8a6`, on 2026-10-11. This is 53
upstream commits after the original `7fd6ef3` reference. The historical parent
reference directory and its evaluation evidence remain unchanged.

The [optimization comparison](OPTIMIZATION.md) records changed capabilities,
shared-core fixes, local timing/context measurements and verification limits.

The official snapshot contains 941 command nodes, 538 shortcuts, 251 generated
methods, and 25 event keys. The fork contains 956 nodes and 543 shortcuts and
retains every official node, shortcut, event, and existing command flag contract
for both configured brands. The exact fork additions and metadata differences
are recorded in `next-interface-mapping.json.gz`. The existing capability gate
still rejects removals, changed existing flags, weakened brand guards, and
unmapped differences; its enforcement was not changed for this upgrade.

`coverage.json`, `sources.json`, and `registrations.json.gz` were captured from a
clean temporary checkout of the official commit using `next_inventory.py freeze`.
The metadata document is assembled from the exact service JSON in the official
committed catalog, ordered by its manifest. `meta_data.json.gz` preserves that
comparison input; product builds embed the original manifest and service files.
`configured-brand-context.json` records the separate configured capture. No CLI
operation or live API request executes during these captures.

Darwin registrations were executed locally. Linux and Windows reference files
are source-derived: they differ only in `config keychain-downgrade`'s empty risk
annotation in the shared `!darwin` implementation. Both implementation files
match the official commit. Native Linux/Windows capture and execution remain
unverified; the platform profiles explicitly report that limitation.

Run the current capability gate with:

```sh
python3 scripts/next_inventory.py gate --target . \
  --reference-dir baseline/v1.0.97 \
  --mapping baseline/v1.0.97/next-interface-mapping.json.gz \
  --out .tmp/upstream-sync/capabilities.json
```

The upgrade preserves the fork's compact JSON, request-only schema views,
exclusive result artifacts, mutation receipts, read-only Sheets tool
classification, non-secret startup brand selection, and shortened skill routing.
Discovery now requests the complete command tree from upstream's target-aware
builder. Base exports accept upstream's button and unknown field types while
preserving source values and rejecting colliding column names.

Upstream intentionally changed some historical defaults, aliases, and schemas,
including Base list pagination, Wiki lookup hints, and Sheets chart options.
Those changes belong to the official release baseline; the original frozen gate
will continue to report them when explicitly selected. Historical token savings
and live results do not establish performance or live parity for this revision.
