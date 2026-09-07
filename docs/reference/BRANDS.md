# Configured brand reconciliation

The original frozen `registrations.json.gz` records an unrestricted bootstrap
assembly. Its exporter set the brand environment but omitted the production
entry point's environment credential-provider import and explicit
`WithStartupBrand` option. With no resolved account in that assembly, the
reference intentionally left brand-restricted services visible. Both original
brand entries therefore have the same 938-node broad capability inventory.
They remain frozen; they are not configured Feishu/Lark runtime evidence.

The exporter now imports the same environment provider as `main.go`, passes the
startup brand explicitly, and uses an isolated synthetic app account. It disables
remote metadata and directs external proxies to an unreachable loopback port.
No command is dispatched. Legacy export still compiles against the unchanged
reference, without a reference to the next-only `WithAgentMode` API.

[Configured registrations](configured-brand-registrations.json.gz) and
[capture context](configured-brand-context.json) supplement the frozen files.
Capture requires the exact frozen reference revision and byte-identical frozen
source files, and refuses a purported configured Lark capture if the complete
Apps deny subtree is missing.

| Reference assembly | Nodes | Hidden nodes | Apps behavior |
| --- | ---: | ---: | --- |
| Configured Feishu | 938 | 11 | Available |
| Configured Lark | 938 | 108 | All 98 Apps nodes hidden and guarded; flag parsing disabled |

All generated API methods, shortcuts and event schemas remain in the compiled
inventory. A hidden brand-restricted command is retained as a deny surface;
it is not represented as supported on that brand. Guard metadata now includes
`DisableFlagParsing` as well as hidden status.

Reproduce the supplement without modifying the reference checkout:

```sh
python3 scripts/next_inventory.py capture-brands \
  --reference ../feishu-cli-review \
  --out docs/reference/configured-brand-registrations.json.gz \
  > docs/reference/configured-brand-context.json

python3 scripts/next_inventory.py check --surface next --target . \
  --reference-registrations docs/reference/configured-brand-registrations.json.gz
```

Use an immutable target clone for a stable checkpoint. The checker continues to
report source and intentional contract differences as `review_required`; the
supplement is not an approval mechanism. A check against the original bootstrap
file now reports configured Lark guard changes rather than silently treating
the bootstrap tree as a configured account.

[Candidate capture](p3-next-post-mail-reconciliation.json) uses immutable commit
`1058824` and matches both configured brand guards. It retains all 938 reference
nodes, 535 shortcuts, 251 pinned generated methods and 25 event keys, with no
missing or changed existing flag contracts. Candidate counts are 951 nodes and
538 shortcuts. Thirty deliberate metadata paths per brand remain for review,
including the three additional Mail query risk annotations; source differences
remain separate.

The [Mail mapping](p3-post-mail-interface-mapping.json) binds exact native IDs,
POST templates and frozen risk hashes to the next-only corrections. Independent
recount of its owner evidence confirms 1,643 Mail unit, 14 inherited selected E2E
and 215 process passes, with one explicit live skip. Arbitrary raw POST and
neighboring mutations retain their original treatment. No live permission,
platform or token-efficiency guarantee follows from this inventory correction.
