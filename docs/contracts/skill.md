# Single entry skill and project lifecycle

`agent-skills/feishu-cli-next/SKILL.md` is the concise English entry skill.
`guide` and `guide show` print its embedded bytes as Markdown on stdout without
requiring authentication. Existing `skills list/read` commands retain the
upstream documentation for compatibility and optional domain detail. The entry
skill routes Base formula and lookup field guidance through executable
`feishu-cli-next skills read lark-base references/lark-base-field-formula.md`
and `feishu-cli-next skills read lark-base references/lark-base-field-lookup.md`.
These complete English references remain embedded in the binary; the installed
entry does not depend on repository-relative files or a second skill install.

`guide install|update|uninstall --project PATH` manages only
`PATH/.agents/skills/feishu-cli-next/SKILL.md`. The default project is the current
directory; an explicit project must already exist. This is host-local,
CLI-managed configuration, independent of extension document FileIO providers.
No command installs globally or changes profiles, other skills, or extra files.

Lifecycle output is JSON `{ok, action, path, changed}`. Repeating an installation
of the same version, an update to the same version, or uninstalling an absent
skill succeeds with `changed:false`. Installing over a different managed version
requires `update`. Updating an absent skill fails. Typed validation errors report
collisions; typed file I/O errors preserve the original cause.

The installed file has a versioned HTML footer containing the SHA-256 checksum
of all preceding embedded bytes. This is local ownership/change detection, not
a signature. A single file holds both the content and checksum so no separate
manifest can become inconsistent. Updates and uninstall refuse unmarked,
malformed, or edited files. Preserve local edits elsewhere before resolving a
collision; there is intentionally no force-overwrite option.

Install publishes a fully written file exclusively; update uses a synced
same-directory temporary file and atomic rename. Managed files and locks use
mode 0600 and newly created directories use 0700 on Unix; Windows follows its
native permissions. Symlinks and non-directory components beneath the project,
and non-regular skill targets, are rejected. The project itself may be a symlink
and is resolved to its existing directory. Operations are local filesystem
operations and do not claim protection against an adversarial process replacing
directory components concurrently.

An exclusive `.install.lock` in the skill directory serializes lifecycle
commands. Normal completion removes it. If a process crashes, inspect the
installation and remove that exact lock after confirming no lifecycle command
is running. Uninstall removes only the intact managed skill file; directories
and unrelated files remain. No automatic deletion of modified content occurs.

Validation: `go test ./internal/nextskill ./cmd/guide`; real process coverage is
`python3 scripts/next_test.py --section foundation --suite offline --binary /absolute/path/feishu-cli-next`.
The foundation runner supplies isolated HOME/config and the explicit binary;
the guide lifecycle needs no credentials or live Feishu resources.
