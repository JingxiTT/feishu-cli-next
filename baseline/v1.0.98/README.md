# Official v1.0.98 sync baseline

The fork integrated official release `v1.0.98`, commit
`ac81f5056ab88f37d581d25916f3dab85abfca76`, on 2026-10-25.
The v1.0.97 snapshot, original baseline, and retained evidence are unchanged.

The official release adds device-bound authentication (DPoP), the local
`config set dpop disabled|preferred|required` policy, URL rewriting extensions,
and manifest-based distribution updates. Its task and Apps guidance is included.
The user approved the required upstream dependencies and Base behavior changes:
record reads now default to NDJSON artifacts, Markdown output is removed, and
NDJSON list/search reads fetch one page of up to 2,000 records. A caller must
continue explicitly with `next_offset` when `has_more` is true.

The fork retains compact JSON, mutation receipts, request identity propagation,
exclusive artifact publication, missing-pagination rejection, config transaction
locks, and stable app selection during tenant-token refresh. The new config
setter participates in the same transaction lock. Fork package identity and
previous dependency upgrades are retained.

The clean official checkout registers 942 command nodes, 538 shortcuts,
251 generated methods, and 25 event keys. The fork registers 957 nodes and
543 shortcuts for each configured brand. All official command nodes, shortcuts,
existing flags, event keys, and brand guards are retained. These are registration
counts, not live verification results.

`coverage.json`, `sources.json`, and `registrations.json.gz` were captured with
`next_inventory.py freeze` from a clean official checkout. The API catalog is
byte-identical to v1.0.97; `meta_data.json.gz` retains that exact comparison
input. Configured brand registrations were executed on Linux/amd64 using
Go 1.26.10, isolated synthetic credentials, remote metadata disabled, and
unreachable external proxies. No business command or live tenant request ran.
Darwin and Windows profiles are source-derived; native captures were not run.
The only Darwin registration difference is the supported
`config keychain-downgrade` write risk in its platform implementation.

The approved `next-interface-mapping.json.gz` retains all 5,770 existing fork
mapping rows unchanged and adds exactly six rows: the three shared result flags
on upstream's new `config set` command, for both brands. It binds the new reference
hash and an immutable candidate revision. It cannot waive removals, changed
existing flags, or weakened brand guards. Source differences still require
separate review; registration parity establishes no live or efficiency claim.

```sh
python3 scripts/next_inventory.py gate --target . \
  --reference-dir baseline/v1.0.98 \
  --mapping baseline/v1.0.98/next-interface-mapping.json.gz \
  --out .tmp/upstream-sync/capabilities.json
```

The sync verification record is added under `evidence/upstream-v1.0.98/` after
local checks complete. Historical measurements describe their original revisions.
