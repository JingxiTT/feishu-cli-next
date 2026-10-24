# Documentation index

Start with the row that matches what you are doing.

| You want to | Read |
|-------------|------|
| Install and use the CLI | [README](../README.md), then [upstream authentication and configuration](../README.upstream.md#authentication) |
| Change the code | [Contributing](contributing.md), then [AGENTS.md](../AGENTS.md) for the binding rules |
| Run tests, with or without a Feishu account | [Testing](testing.md) |
| Build, install, or package a distribution | [Distribution](distribution.md) |
| Run or read an agent token comparison | [Evaluation](../evaluation/README.md) |

## Contracts

Read the owning contract before changing the behavior it covers.

| Contract | Covers |
|----------|--------|
| [Output](contracts/output.md) | Compact JSON, `--result-fields`, `--save-result`, artifacts |
| [Receipts](contracts/receipts.md) | Mutation attempts, outcomes, reconciliation |
| [Discovery](contracts/discovery.md) | `discover` search and flag descriptions |
| [Recovery](contracts/recovery.md) | Recovery text on failures |
| [Skill](contracts/skill.md) | Entry skill content and `guide` install lifecycle |
| [Errors](../errs/ERROR_CONTRACT.md) | Typed error envelopes (upstream-owned) |
| [Affordance](../affordance/README.md) | Per-command guidance format (upstream-owned) |
| [Source guards](../lint/README.md) | Lint rules and waivers (upstream-owned) |

## Maintaining the fork

| Topic | Read |
|-------|------|
| Upstream files the fork edits | [Upstream footprint](upstream-footprint.md) (generated) |
| Capability gate and interface mapping | [baseline/CI.md](../baseline/CI.md), [v1.0.98 baseline](../baseline/v1.0.98/README.md) |
| Sealed evidence | [evidence/README.md](../evidence/README.md) |

## Records

These describe how the fork was built and what was verified at a given
revision. They are kept for provenance; current behavior is defined by the code,
tests, and contracts above.

| Record | Holds |
|--------|-------|
| [Sections](sections/OWNERSHIP.md) | Section ownership and the reports for the sections that [testing](testing.md) links |
| [Integration](integration/sheets-base.md) | Sheets to Base artifact handoff suite |
| [Security](security/history-review-20261012.md) | Secret history review |

Section reports quote the commands their runs used. For a real-account run
today, use `make live-test SECTION=<section>` as described in
[testing](testing.md); `next_test.py --suite live` stays blocked.
