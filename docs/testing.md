# Tests with and without a Feishu account

| Part | Where it runs | Feishu account | Entry point |
| --- | --- | --- | --- |
| Account-free | GitHub Actions on PRs/main; also locally | None | `make offline-test` (same checks as `make test`) |
| Real-account | Manually on the user's machine only | Local authenticated account and verified fixtures | `make live-test SECTION=<section>` |

GitHub's required `results` and `offline` checks keep their existing coverage.
Build, race tests, coverage, lint, capability reconciliation, mock/dry-run E2E,
controlled evaluation and cross-domain checks belong to the account-free part.
Other automation can use GitHub, coverage or model-review credentials; none
requires a Feishu account. Real-account tests have no GitHub Actions runner,
schedule, repository secrets or required merge check.

## Manual real-account tests

Read [the testing authorization boundaries](#authorization-boundaries-for-all-testing)
first. Authenticate locally, independently verify the effective account/tenant,
and prepare only the private, run-owned fixtures described by the section guide.
Keep tokens, fixture manifests and raw evidence outside version control. The
runner uses the local authentication environment/configuration and does not
create an account or establish fixture authorization for you.

Build once with `make build`, then explicitly select one scenario:

```sh
# After local login and independently verifying these identity values:
export NEXT_CONTACT_LIVE=1
export NEXT_CONTACT_LIVE_ACCOUNT_ID='<verified-current-account-open-id>'
export NEXT_CONTACT_LIVE_TENANT_KEY='<verified-current-tenant-key>'
make live-test SECTION=contact
```

| Section | Selected test | Fixture and opt-in instructions |
| --- | --- | --- |
| `contact` | Current-account reads; no writes | [Contact](sections/contact.md) |
| `drive` | Private file lifecycle | [Drive](sections/drive.md) |
| `sheets` | Private user lifecycle | [Sheets](sections/sheets.md) |
| `docs` | Private native document lifecycle | [Docs](sections/docs.md) |
| `markdown` | Private Markdown document lifecycle | [Markdown](sections/markdown.md) |
| `base` | Original strict preservation probe | [Base](sections/base.md) |
| `base-supported` | Separate supported-input lifecycle | [Base](sections/base.md) |

The runner requires each test's existing opt-in and fixture inputs. The tests
verify account binding and resource privacy before dependent writes; cleanup
stays with the owning test. The Base selections retain their distinct input
contracts and results. Do not run the entire inherited `tests/cli_e2e/...` tree
with real credentials: those legacy workflows are not verified private fixtures.

`.tmp/next-tests/live.json` records only the selected scenario's status, command,
counts and coverage limit, without raw API output or credentials. Exit 0 means
the selected test passed; exit 2 means blocked (including skipped/no tests);
other nonzero exits mean failure. Unsupported sections, including `all`, remain
blocked rather than appearing covered. Missing credentials can produce a test
failure or skip; neither counts as live success. Inspect the section's private
recovery evidence and reconcile unknown mutation outcomes before any rerun.
Raw test diagnostics are saved separately in a unique owner-only `.log` file
beside the report. They may contain private data: keep them local and never
commit or upload them. The command prints that path without printing its contents.

Direct invocation supports `--binary` and `--report`:

```sh
python3 scripts/next_live_test.py --section contact --allow-live \
  --binary ./feishu-cli-next --report .tmp/next-tests/contact-live.json
```

The local runner refuses CI execution. The existing isolated
`next_test.py --suite live` remains blocked; use this dedicated local runner for
the listed account-bound scenarios. Offline success and one live scenario do
not establish complete live parity or release readiness. The separate
`make live-skills-test` exercises remote skill downloads and needs no Feishu
account; it is not part of real-account testing.

## Authorization boundaries for all testing

These are testing-only authorization boundaries. They apply to every test, including unit, integration, E2E, live, manual, smoke, regression, agent evaluation, test setup, and cleanup, whether run by the coordinating agent, a subagent, a script, or CI. Offline tests use synthetic data and must not accidentally perform live actions.

Do not encode these test-specific recipient or visibility restrictions as product limitations, runtime allowlists, or instructions in the shipped skill. The CLI must retain full functionality for users acting within their own authorization. During testing:

- Do not send messages, documents, files, invitations, notifications, shares, or any other content to anyone in Feishu except **Ying Wu**. This includes indirect delivery through mentions, assignments, subscriptions, group chats, permission changes, or automated workflow effects.
- Ying Wu is the only authorized other recipient or collaborator. Resolve that person's identity unambiguously before any delivery or access grant; a matching display name alone is insufficient. Do not send to groups or destinations that include anyone else.
- The user authorizes freely creating documents, Sheets, Base resources, and other Feishu objects for testing when they are visible only to the currently logged-in account. Routine creation and use within this boundary do not require repeated approval.
- Private resources must remain restricted to that account unless intentionally shared with the authorized exception, Ying Wu. Do not enable public, link-wide, organization-wide, inherited group, or other-user access. Verify destination and creation visibility before writing; creating broadly visible content and restricting it afterward does not satisfy this boundary.
- Bind every live run to the effective current account and tenant. Use isolated, clearly labeled resources created for testing; do not edit or delete unrelated pre-existing resources. Clean up only resources created by the run and preserve any the user wants to keep.
- If a feature necessarily exposes content or notifies anyone other than Ying Wu, do not perform that live action. Continue with offline fixtures and record the live case as blocked under this boundary. All test runners and subagents inherit this restriction and cannot broaden test authorization.

Write and run unit tests using synthetic fixtures and mock transports during implementation. Write live E2E tests alongside the implementation, then batch their execution into the completed section’s full suite; do not run live tests after each small function. The private-resource authorization above provides the permitted fixture scope; verify actual account, identity, and visibility before using it. Ordinary CI must not depend on personal credentials.

If the required privacy or recipient restrictions cannot be verified, keep the affected live test blocked and continue independent offline work. Blocked or skipped live tests are not passing evidence, and live-dependent capabilities remain unverified until their required checks can run within these boundaries.
