# Official v1.0.98 sync verification

The fork integrates official `v1.0.98` (`ac81f5056ab88f37d581d25916f3dab85abfca76`).
The [baseline](../../baseline/v1.0.98/README.md) describes the release changes,
user-approved Base behavior changes, dependency additions, and exact mapping.

Local Linux/amd64 validation passed: `make test`, final race-enabled unit tests,
vet, formatting, the all-sections offline suite, capability and quality gates,
sidecar tests, and lint tests/checks. The quality gate emitted nonblocking
flag-naming and skill-routing diagnostics. The capability gate retains all 942
official command nodes, 538 shortcuts, 251 generated methods, and 25 event keys
for both brands, with no changed existing flags or weakened brand guards.

[The sealed verification record](../../evidence/upstream-v1.0.98/verification.json)
links the full reports and logs and records earlier failed attempts separately.
The fork-specific identity checks cover propagation of DPoP token bindings into
artifact and receipt identity reads. Config concurrency, exclusive artifacts,
explicit Base continuation, and missing pagination remain covered.

These results are offline. Live tenant acceptance, native Darwin/Windows
execution, source-semantic equivalence, and efficiency measurements are not
established. Historical baselines and retained evidence remain unchanged.
