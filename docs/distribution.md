# Local distribution

The npm package and executable are `feishu-cli-next`. This derivative never
installs or publishes `@larksuite/cli`, and retains the upstream MIT license.
The npm package is private and publishing workflows are disabled pending P4.
A public repository, owned npm name, release credentials, trusted publishing,
and signing/notarization identities must be accepted before enabling the
reference-equivalent automated public release channel. Apple signing is optional
for source builds and unsigned local archives.
The native `update` command currently reports an unconfigured release channel.

Build the current host with `./build.sh` or `make build`. Python 3 and Go are
required; frozen metadata is restored without network. Dependencies must already
be cached for offline builds. `python3 scripts/next_package.py --output .tmp/release`
builds all seven OS/architecture pairs, each in normal and `authsidecar` variants,
and writes SHA-256 checksums plus a local npm wrapper. An existing output directory
is refused. `--target linux/amd64 --variant normal` selects a smaller build.

Archives contain the executable, license, changelog, README and this document.
Windows archives are ZIP; other platforms use tar.gz. Both variants retain the
same executable name and differ in the archive suffix (`-authsidecar`). Use the
sidecar archive explicitly for host-managed credential isolation. npm installs
the normal variant. The supported pairs are macOS amd64/arm64, Linux
amd64/arm64/riscv64, and Windows amd64/arm64; riscv64 is Linux-only.

For local npm setup, first pack the generated wrapper:

```sh
npm pack --ignore-scripts ./.tmp/release/npm --pack-destination .tmp/release
FEISHU_CLI_NEXT_ARTIFACT_DIR="$PWD/.tmp/release" npm install --offline --prefix ./cli-install ./.tmp/release/feishu-cli-next-0.1.0-beta.1.tgz
./cli-install/node_modules/.bin/feishu-cli-next --version
```

The installer checks the archive against checksums inside the npm package before
extracting it; supplying an artifact directory cannot override those checksums.
Public downloads are unavailable until the release channel is configured. npm
owns package binary install/upgrade/uninstall. Upgrade by installing the new
local tarball using the same prefix. `npm uninstall --offline --prefix ./cli-install
feishu-cli-next` removes that package; configuration and project skills are
separate. Avoid npm `--force`, which can override npm's existing-link checks.

For a direct archive installation, verify `checksums.txt`, extract into a fresh
version directory, and invoke that executable. Select the new version directory
to upgrade; remove only the directory you created to uninstall. Existing binaries
need not be overwritten. Alternatively use `make install PREFIX=/your/prefix`, `make update-install
PREFIX=/your/prefix`, and `make uninstall PREFIX=/your/prefix`. These targets
manage only the executable and its adjacent `.feishu-cli-next.sha256` ownership
file (with `.exe` on Windows). They refuse unrelated, symlinked or edited files.
Install repeats are no-ops; changing a managed version requires update-install.
A local exclusive lock serializes operations. A crash can leave that exact lock
or an incomplete binary/checksum pair: inspect both files before recovery and
remove the lock only when no installer is running. Incomplete or edited states
fail closed. This checksum is local ownership detection, not a signature or
protection against another process concurrently replacing directory components.

`feishu-cli-next guide install --project /existing/project` installs the one
embedded entry skill. Update with `guide update` and remove with `guide uninstall`
using the same project. The npm wrapper's `install --project` maps to `guide install`.
Installation does not configure apps or authorize accounts. Continue explicitly
with `config init --new` and `auth login` when required; their native language,
profile and identity options remain available. Existing global upstream skills
remain usable through `skills list/read`, and are never rewritten by setup.

The [skill lifecycle contract](contracts/skill.md) defines collisions, ownership,
unchanged content, symlinks and cleanup. npm uninstall deliberately preserves
project skills, profiles, configuration and unrelated files. Build success for a
foreign architecture does not establish runtime behavior. Acceptance requires
applicable OS and lifecycle evidence, not physical hardware per architecture.
Both macOS amd64 variants passed startup and project guide smoke under Rosetta on
macOS arm64; Linux/Windows OS-specific behavior remains unverified. The retained
automated public release channel also requires macOS signing/notarization
verification. See the [packaging evidence](sections/packaging.md) for exact scope.
