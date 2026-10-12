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

Read [the testing authorization boundaries](PLAN.md#authorization-boundaries-for-all-testing)
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
