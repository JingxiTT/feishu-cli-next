# Security policy

Feishu CLI Next handles credentials and workspace data. Security fixes are
developed on `main`; this experimental fork does not maintain older releases.

Report vulnerabilities privately using GitHub's
[Report a vulnerability](https://github.com/Zhujingxi/feishu-cli-next/security/advisories/new)
form. Private vulnerability reporting is enabled for this repository. Do not
open a public issue containing an exploit, credentials, or private tenant data.

Include the affected revision and platform, the impact, and reproduction steps
using synthetic credentials and resources. Never include live tokens, app
secrets, or personal data. If a credential was exposed, revoke or rotate it
through its provider immediately; removing it from a file does not revoke it.

The maintainers will coordinate fixes and disclosure through the private report.
See the [2026-10-12 history review](docs/security/history-review-20261012.md) for
the scope and remaining limits of the repository's initial secret review.
