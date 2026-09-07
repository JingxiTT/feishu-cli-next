# P1 shared-interface gate

The foundation is ready for parallel P2 implementation at `00f3e8e`. This accepts
the shared prerequisites and their scoped checks, not full release or live domain
parity. [Machine-readable evidence](foundation-evidence.json) records revisions,
commands, skips and limitations. Complete raw offline logs remain in `.tmp`.

The shared foundation reuses authentication, profiles, credentials, runtime,
FileIO, API clients, events, streams and extension contracts. Additions are
actual-tree discovery, compact/projection output, exclusive result artifacts,
durable mutation receipts and typed host outcomes, and one project skill.
[Output](../contracts/output.md), [receipts](../contracts/receipts.md),
[skill lifecycle](../contracts/skill.md), and [section ownership](OWNERSHIP.md)
define the interfaces consumed by domain owners.

## Acceptance evidence

- All 5,776 selected foundation unit test events passed. Nine explicit skips
  cover opt-in live/semantic probes, subprocess helpers and platform cases.
  Later binding, typed outcome, output, manifest and fixture changes passed
  their affected package regressions.
- The complete foundation process suite passed: 13 Python checks and 85 Go
  test events. Four nested tests intentionally exercise missing-credential skip
  behavior; their parent assertions pass. The final shared-fixture extraction
  separately passed all 16 foundation Go E2E tests, with no skips.
- Plugin and sidecar checks passed 456 test events without skips, plus the two
  public example builds and the authsidecar build. Plugin customer forks were
  built from `d4c705c`; later changes did not alter their SDK contracts.
- Vet, formatting, script tests, module tidy (no dependency diff), pinned
  golangci 2.1.6 and license policy passed. The committed quality gate passes
  using the actual next surface; it reports two non-failing bounded-list
  guidance warnings. Source-contract lint and its own tests also passed.
- Read-only live auth and own-user-info checks verified the current user. Bot
  verification was unavailable in that run; independent tenant consistency is
  not exposed by auth status. No content was written or delivered.
- [Balanced development evaluation](../../evaluation/evidence/foundation/REPORT.md)
  passed 18/18 compatible trials at `d873c0d`, with actual provider token usage.
  It includes the small status-task regression and earlier unknown-cost failed
  attempts. This is scoped foundation evidence, not a full-product claim.

Review found and resolved credential-ambiguous artifacts, missing typed mutation
outcomes for embedded hosts, bare-envelope metadata loss, and stale configured
user binding. Tests also exposed isolated-HOME assumptions, a nonexistent schema
fixture, auth enrichment counted as business writes, and canonical path spelling;
their fixes preserve product behavior and explicit request assertions.

## Remaining scope

All reference registrations remain present; the source ledger still tracks 22
intentional changed reference files for final hash-bound reconciliation. Domain
owners must verify supported content/types, no-op and concurrency semantics,
complete lifecycles, authorized live effects, and section development evaluations.
An absent tenant observation is never a verified tenant binding. Artifact
consumers must validate available binding and reject insufficient evidence where
their workflow requires it. Packaging, cross-section transfer, complete platform
tests and held-out evaluation remain required before release acceptance.

Independent process suites import `tests/nextfixture`. Their `TestMain` calls
`os.Exit(nextfixture.RunMain(m, "../../.."))` from `tests/next_sections/<section>`;
`NewFixture` returns a real-process fixture with separate identity-request counts
and a controlled business backend. The runner must remain credential-isolated.
