# Isolated section regressions

Run from the repository root. Python 3, Go, a populated Go module cache, and a
built CLI with full embedded metadata are prerequisites. `make build` prepares
metadata; it may need network access. The test runner never fetches metadata or
Go dependencies. Use the binary just built, not an installed personal CLI.

```sh
python3 scripts/next_test_test.py
python3 scripts/next_test.py --section foundation --suite all --binary ./feishu-cli-next --report .tmp/next-tests/foundation.json
python3 scripts/next_test.py --section docs --suite all --binary ./feishu-cli-next --report .tmp/next-tests/docs.json
python3 scripts/next_test.py --section all --suite all --binary ./feishu-cli-next --report .tmp/next-tests/all.json
```

`--section` selects the manifest's domain names, `foundation`, or `extensions`.
`events` maps to the reference `event` packages; `docs` maps to `shortcuts/doc`.
`--suite unit` and `--suite e2e` run separately. `--timeout` bounds each process
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
codes, pass/fail/timeout status, durations, sanitized output, and Go test
pass/fail/skip counts. Child failures retain their exit code; timeout returns
124. Missing prerequisites and blocked live runs return 2. A skipped test is
recorded separately and is not evidence that its behavior passed.

The all-offline suite runs the upstream unit packages, foundation process
checks, the upstream root/config E2E packages, plugin E2E, and domain tests
selected by the official CI's `DryRun|Regression` expression. The report lists
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
