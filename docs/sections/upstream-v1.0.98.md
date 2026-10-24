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

## Security CI reconciliation

The first PR security job flagged 86 public source-file SHA-256 checksums in the
new inventory as generic API keys. Each checksum was recomputed from the exact
official v1.0.98 file. The user explicitly approved adding those exact original
commit/file/line fingerprints to `.gitleaksignore`; no path or scanner rule was
excluded. A generated dummy API key at the same inventory path remained detected.

The [security reconciliation](../../evidence/upstream-v1.0.98/security-reconciliation.json)
retains the reviewed list and local scanner, vulnerability, and license results.
Govulncheck found no called vulnerability; it noted one vulnerability in a
required module that this code does not call. The license check passed, with its
assembly inspection warning. Remote CI results must be assessed separately.
