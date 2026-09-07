# Docs process scenarios

This package builds the production `authsidecar` CLI using `nextfixture.RunMain`.
Each CLI invocation uses a temporary config/work directory and a loopback synthetic
backend. No backend handler forwards to Feishu, no personal credentials are
loaded, and payloads are synthetic. Identity enrichment is provided separately by
the shared fixture. Run only in the isolation supplied by `scripts/next_test.py`.

Run the whole section boundary with the canonical binary and isolated runner:

```sh
make build
python3 scripts/next_test.py --section docs --suite all --report .tmp/docs-section.json
python3 scripts/next_test.py --section docs --suite live --allow-live --report .tmp/docs-live.json
```

The live command returns a blocked report and exit 2 until the verified fixture
harness exists; it dispatches no request. Complete process test output is captured
as Go JSON test events in the section report.

| Scenario | Process evidence |
| --- | --- |
| Create and fetch | File inputs; rich XML, Unicode, title escaping, lossless large integer/decimal reference-map input; revision zero; full export options; saved-result artifact preserves content and references |
| Update | All eight operations, append lowering to end insertion, range/block/source addressing, revision zero forwarding; changed XML overwrite and exact unchanged XML no-op |
| Bounded reads and conversions | Outline, range, section, keyword; depth/context/language inputs; explicit Markdown/IM Markdown residual conversion keeps reference sidecars and emits detail downgrade warnings; no read mutation receipts |
| Search and history | Folder-only search, page-size bound, continuation; history list, accepted revert and read-only task status |
| Local script | Draft initialization from presentation decision, writing reserved draft path, parsing local XML with assessment and profile |
| Media/resource plans | All seven Docs media/resource shortcuts plus whiteboard media download; explicit dimensions, cover offset, endpoint/method chains, zero output files in dry-run |
| Downloads | Preview, media, whiteboard and cover bytes preserved to files; existing files retained unless overwrite explicitly selected |
| No-op | Empty-cover clear does no PATCH; unchanged XML overwrite does no PUT and stores no receipts |
| Validation | Unsupported identity and read-scope rejection; invalid create/reference inputs, missing read selectors, negative context, incomplete update addressing, invalid history, conflicting parse/media/resource inputs all fail before a business request |
| Recovery | Lost update response is unknown, dispatched once; inspect/cleanup are local, fetch is read-only; service failure markers retain warnings/partial data without claiming success; bot create followed by rejected current-user permission grant returns a partial summary with both receipts |
| Cross-owner registration | `docs +whiteboard-update` Mermaid dry-run reaches its registered request plan; full implementation/evidence belongs to the Whiteboards owner |

`TestDocsLivePrivateLifecycle` is independently selectable with `-run
'^TestDocsLivePrivateLifecycle$'` and deliberately skips. No environment opt-in
can bypass missing independently verified account, tenant and private-creation
visibility evidence. The required live lifecycle is described in the skip reason.
The skip is blocked evidence, never a passed live verification. Parent section
acceptance also requires domain units, applicable frozen regressions, private
live fixtures and comparative tasks.

The asynchronous history fixture uses HTTP 200 with a pending task marker to
exercise conservative domain receipt interpretation. Synthetic success does not prove backend
conditional-write protection, snapshot consistency, resource fidelity or private
visibility. Full upload/orchestrated insert/write-resource paths have dry-run and
nearby domain unit coverage; their actual storage/backend behavior remains a live
fixture prerequisite.
