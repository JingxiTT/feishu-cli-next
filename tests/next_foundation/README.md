# Isolated section regressions

Run from the repository root. Python 3, Go, a populated Go module cache, and a
built CLI with full embedded metadata are prerequisites. `make build` restores verified frozen
metadata; only an empty Go dependency cache may need network access. The test runner never fetches metadata or
Go dependencies. Use the binary just built, not an installed personal CLI.

```sh
python3 scripts/next_test.py --harness scripts --report .tmp/next-tests/scripts.json
python3 scripts/next_test.py --harness evaluation --report .tmp/next-tests/evaluation.json
python3 scripts/next_test.py --section foundation --suite all --binary ./feishu-cli-next --report .tmp/next-tests/foundation.json
python3 scripts/next_test.py --section docs --suite all --binary ./feishu-cli-next --report .tmp/next-tests/docs.json
python3 scripts/next_test.py --section all --suite all --binary ./feishu-cli-next --report .tmp/next-tests/all.json
python3 scripts/next_test.py --integration sheets_base --binary ./feishu-cli-next --report .tmp/next-tests/sheets-base.json
```

`--section` selects the manifest's domain names, `foundation`, or `extensions`.
`--integration <name>` instead runs the complete matching
`tests/next_integration/<name>` package as a separate cross-domain gate; a missing
package is blocked. It does not run unrelated foundation or domain suites.
`events` maps to the reference `event` packages; `docs` maps to `shortcuts/doc`.
`--suite unit` and `--suite e2e` run separately. Each existing
`tests/next_sections/<section>` package runs in full alongside the inherited
regressions; it has no test-name filter. The `all` selection includes every
existing section package. Reports list missing section suites explicitly and
return blocked (exit 2) even if the available inherited regressions passed. `--timeout` bounds each process
(default 1,200 seconds, plus Go's 10-minute test timeout). Run each section's
E2E suite at its acceptance boundary. Packaging remains blocked until its
complete suite exists.

Every invocation creates and removes private HOME, config, state, cache, and
temporary directories. The child environment starts from an allowlist of tool
locations: developer CLI settings, credentials, live-test toggles, TEST_*
variables, Python startup paths, and inherited proxy settings are absent. Only
Go dependency/build caches are shared. Remote metadata, update/skill notifiers,
automatic toolchain downloads, and Go's saved environment file are disabled.
A reserved non-listening localhost HTTP(S) proxy prevents external calls by
clients honoring proxy settings, while local fixture servers still work. This
is not an OS network sandbox: tests using custom transports must remain synthetic.
No ordinary test may depend on the developer keychain or local account.

The runner prints one JSON summary and writes a JSON report with command exit
codes, pass/fail/timeout status, durations, sanitized output, and Go/Python unittest
pass/fail/skip counts. Child failures retain their exit code; timeout returns
124. Missing prerequisites and blocked live runs return 2. A skipped test is
recorded separately and is not evidence that its behavior passed.

The all-offline suite runs the upstream unit packages, foundation process
checks, the upstream root/config E2E packages, plugin E2E, domain tests
selected by the official CI's `DryRun|Regression` expression, and the complete
existing `tests/next_sections` packages. The report lists
every selected and excluded domain test. **This is the inherited deterministic
regression subset, not a complete section acceptance or full live suite.**
Unclassified tests, live scenarios, platform-specific builds, packaging, and
comparative evaluations require their own evidence. The Go foundation fixtures
build the production main with the existing `authsidecar` tag and route sentinel
auth requests to local HTTP handlers through `LARKSUITE_CLI_AUTH_PROXY`. They do
not forward requests to real API hosts. Python foundation checks use the normal
binary supplied with `--binary`; reports distinguish these commands. This fixture
evidence does not establish live backend or normal-build transport behavior. Plugin E2E builds from
`git archive HEAD`; commit the changes before relying on its results.

Live selection is explicit and remains blocked:

```sh
python3 scripts/next_test.py --section docs --suite live --allow-live --report .tmp/next-tests/docs-live.json
```

This writes `evidence: live`, `status: blocked`, and executes no tests. Merely
setting an opt-in flag or supplying a token does not establish the plan's
required account, tenant, private visibility, recipient identity, and cleanup
ownership. A fixture harness must verify these before enabling live execution.
Do not use `go test ./tests/...` or `make test` in a credentialed shell to work
around this gate: legacy live helpers probe the local account even without an
explicit token variable.

`make cross-domain-test` runs Sheets to Base independently. `make integration-test`
includes that gate and all section E2E suites. `make next-script-test` and
`make evaluation-test` run the isolated Python checks without requiring a CLI
binary. Controlled evaluation tests do not constitute real model comparisons.

`.github/workflows/next-offline.yml` runs the complete offline section, script,
evaluation harness, and cross-domain gates on native Linux, macOS, and Windows.
It preloads public Go dependencies before isolation and always uploads separate
machine reports plus stage outcomes, including failures and blocked prerequisites.
The workflow adds to the inherited CI gates. Its definition is not evidence that
a native platform passed: that requires a completed run on that platform. Offline
CI does not establish live API behavior, held-out model gains, signing, or release
publication readiness.
