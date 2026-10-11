# Secret history review: 2026-10-12

Gitleaks 8.30.1 scanned all local Git refs using the repository's existing
`.gitleaks.toml`, at working-tree HEAD
`fc16b9beb43e8bce109c1bcd42e3eeb4404fefad`. There were 477 reachable commits.
The scan returned exit code 1 and 4,830 findings; it did **not** pass cleanly.
An earlier scan covered 476 commits and returned 4,551 findings. Counts include
repeated values across commits and refs.

Each finding was classified against its historical source blob, including JSON
field context and the definitions of synthetic test fixtures. The private raw
report is not distributed. No finding values are reproduced here.

| Classification | Findings | Files | Evidence |
| --- | ---: | ---: | --- |
| SHA-256 integrity metadata | 4,401 | 20 | Exactly 64 lowercase hexadecimal characters in source inventories, artifact hashes, build/catalog digests, or fixture references. |
| Deterministic offline fixtures | 289 | 8 | Synthetic document, wiki, spreadsheet, minute, or tokenizer values defined by offline tests and their evaluation contracts. |
| Public catalog application IDs | 72 | 2 | OpenAPI example `app_id` fields in committed `im` and `task` service snapshots; application IDs are identifiers, not application secrets. |
| Pagination placeholder | 17 | 1 | An IM reference example explicitly ends in `xxxx`. |
| Published upstream resource examples | 51 | 2 | Two distinct resource IDs in Base app-block and Approval document examples match public upstream examples at the revisions below; neither is an authentication credential. |

The last category appeared in
`skills/lark-base/references/lark-base-app-block-data-config.md` and
`skills/lark-approval/references/lark-approval-instance-form-control-parameters.md`.
Both exact values were compared in memory with the same files in public
`larksuite/cli` at frozen comparison commit
[`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`](https://github.com/larksuite/cli/tree/7fd6ef3c07182257ce776cdc5a614e122d5bd4b3)
and tag `v1.0.97`, which resolves to commit
[`72579c80027c863ca51d5f9affda72a70ab0d8a6`](https://github.com/larksuite/cli/tree/72579c80027c863ca51d5f9affda72a70ab0d8a6).
Both revisions contain the same examples. The respective source-file SHA-256
digests are `13b65987b18bea1c83fe5a93fa8c67ea1e28ccee2f6ea878d471c7214a9e24ac`
and `8367dcd3328f3a1acaaead392caf72b390cdd6c603f4840a2c6c15c25c23dbf6`.

The current examples now use explicit `_EXAMPLE_TOKEN` placeholders; the
published upstream values remain in Git history. These findings are disposed of
as upstream documentation examples, not leaked authentication credentials.
This review did not access the resources or establish whether they exist.
No confirmed authentication credential was found by this scan, which does not
prove the history contains no sensitive data.

Five reference documents also used ambiguous open-user IDs, including short
numeric examples. These now use `ou_EXAMPLE_TOKEN`. The documentation-token and
skill-format checks pass after those edits. The existing token check recognizes
known Lark prefixes; it does not recognize every arbitrary resource identifier.

No history was rewritten, generated catalog edited, or scanner allowlist
broadened. Reproduce the history scan locally with a private report destination:

```bash
umask 077
gitleaks git --log-opts=--all --report-format=json \
  --report-path=/tmp/feishu-history-review.json --redact
```

Expect the classified historical findings to remain. Investigate new findings
against source context rather than suppressing whole directories or token
formats. Keep reports private and remove any unredacted temporary report after
triage. The scan covers local refs and Gitleaks' configured rules; it does not
cover un-fetched or unreachable objects, tenant contents, external artifacts, or
every possible personal-data pattern.
