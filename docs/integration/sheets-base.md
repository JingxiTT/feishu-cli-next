# Sheets → Base artifact integration

This P3 suite runs the real CLI for both sides of the handoff. The source file
comes from `sheets +data-export --save-result`; it is consumed unchanged by
`base +record-import --artifact @file --mapping @file --output`. The test does
not manufacture an import artifact or copy implementation functions into the
fixture. Shared `nextfixture` supplies an isolated account and tenant through a
non-forwarding authsidecar. No live content is created, modified or delivered.

## Scope and independent checks

`tests/next_integration/sheets_base/transfer_test.go` owns one in-memory source
and one destination, accessed only through their actual Sheets/Base API paths.
The fixture retains canonical backend state independently of CLI stdout and
report files. Source state includes native cell JSON, stable IDs, revision and
private visibility. Destination state starts with an unrelated fixture record
that must remain identical.

The complete transfer uses 10,000 rows × 8 columns:

- Unicode text with unique row keys, newlines and a checkmark.
- Numeric zero, `9007199254740991`, `-3.5` and `0.125` as exact JSON numbers.
- Both boolean values, explicit null, an omitted cell, and empty text.
- Leading-zero strings and literal text beginning `=`.

The test verifies all 80,000 cell positions and types against source state,
including JSON field presence. It checks 16 consecutive 625-row Sheets pages,
eight destination field-schema reads, and 50 Base batches of 200 records.
Every created record ID in the report must match the independently stored
record at the corresponding source row. The original destination record,
source state and exported artifact must remain unchanged.

Large cell matrices and record-ID collections remain in local files. Assertions
check that export stdout contains only the artifact reference and extent, and
import stdout omits source values and the 10,000 native record IDs. This is
process-output evidence, not an estimate of model tokens or a comparison with
the reference CLI.

## Failure scenarios

Unsupported formulas, date/time styles and a non-roundtrippable number
(`9007199254740993`) are placed in the last source row of a 401-row sheet. Each
case goes through a real Sheets export, then must fail Base validation before
any write. The exact unsupported number must survive export before the
consumer rejects its conversion. A truncated second source page cannot commit
an artifact or trigger destination reads/writes.

Five further process cases consume an actual exported artifact after changing the
effective tenant, user, app, environment, or user/bot identity. The artifact itself
is never edited. Each case requires typed `--artifact` validation before any Base
schema read or write, with no destination report or mutation receipt. Both the
source state and artifact bytes must remain identical.

For a 401-row handoff, batch one confirms 200 records. Batch two then exercises
four independent backend behaviors:

| Backend behavior | Independent destination prefix | Required CLI evidence |
| --- | ---: | --- |
| Only 37 additional records applied; response includes ignored fields | 237 rows | Unknown outcome, exit 22, unresolved response retained |
| HTTP 403 rejects batch two before application | 200 rows | Partial invocation outcome, exit 21; second receipt confirmed unapplied |
| All 200 additional records apply, then connection closes without a response | 400 rows | Unknown outcome, exit 22; no automatic replay |
| Batch two returns IDs already reported by batch one | 400 rows | Unknown outcome, exit 22; do not accept duplicate IDs or dispatch batch three |

Each failure report must retain exactly the first 200 confirmed IDs, one
completed batch and `first_unconfirmed_row: 200`. That row is a reconciliation
boundary, not authorization to replay. The test checks stdout, stderr and both
durable mutation receipts. It also checks actual stored rows and proves that
there is no third batch or retry.

## Running and evidence

```sh
python3 scripts/next_test.py --integration sheets_base \
  --binary .tmp/sheets/feishu-cli-next \
  --report .tmp/sheets-base/runner-report.json
```

The runner applies the repository's credential-isolated environment and builds
the actual authsidecar binary through `nextfixture.RunMain(m, "../../..")`.
The supplied normal binary is the runner's canonical-build prerequisite; it
does not replace the fresh process-fixture build.

The complete canonical run on 2026-09-07 passed all 11 test/subtest events, with
zero failures and zero skips. Machine-readable runner evidence
is `.tmp/sheets-base/runner-report.json`. The 10,000-row scenario independently
confirmed 80,000 cells, 16 source pages, eight schema reads and 50 destination
batches. Its exported artifact was 1,670,500 bytes; row contents and complete
record-ID lists stayed outside stdout. Scoped `go vet` also passed.

The cross-batch ID regression exposed an importer bug before the final run:
repeated IDs were accepted as complete and batch three was dispatched. The Base
importer now checks IDs across the entire import, while the shared mutation
journal classifies overlapping IDs within the same operation and destination
as unknown before finalizing the receipt. The final run verifies the confirmed
prefix, stopping behavior, stdout, stderr and durable receipts together.
It ran after both product dependencies were committed: Base importer
`f080e43` and shared receipt journal `d91e695`. The canonical integration runner
is provided by `312af93`.

The binding follow-up suite passed all 17 test/subtest events, with no failures
or skips, in `.tmp/sheets-base/binding-followup.json`. This initial run used the
shared working tree and is not an immutable release-candidate result.

## Limits

This is complete offline evidence for the supported primitive handoff, not full
P3 or release acceptance. The Base importer deliberately rejects native formula,
style, rich-content and identity conversions, including otherwise harmless
style metadata. Numeric values must round-trip through Base's double storage.
The primitive source fixture omits those unsupported fields; the suite does not
claim that arbitrary live Sheet-AI response shapes are interchangeable.

Neither endpoint supplies a snapshot or compare-and-swap lock. Backend live
privacy, live type/storage behavior, separately mapped cross-domain workflows and
comparative provider-token measurements remain separate gates. No skipped or
unperformed live case is counted as passing evidence here.
