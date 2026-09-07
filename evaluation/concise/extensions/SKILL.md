---
name: lark-extensions
description: Use or embed a lark-cli host distribution with public command contributions, credential and transport providers, file backends or platform plugins. Use when working with a compiled host integration; ordinary Feishu content tasks belong to their domain guides.
---

No dedicated extension skill ships in the frozen official distribution. This
English guidance condenses its `extension/README.md`,
`extension/platform/README.md` and public command example documentation. The
original source documentation remains available; this is a disclosed documentation
fallback, not a translation of a nonexistent official skill.

Public Go extensions compile into the host binary. Platform plugins register at
startup through an imported package; they share process memory and lifecycle.
There is no dynamic `.so` loading, separate RPC service or subprocess sandbox.
The wrapper runs the CLI with `cmd.Execute` or `cmd.ExecuteWithOptions`.
`config plugins show` reports installed hooks and policies in distributions that
retain that operator command. Use the selected host's command help/schema when
its contributed commands or restrictions are unfamiliar.

Typed business commands use `extension/command.Define` and are contributed through
`cmd.WithCommandSets`. Preserve declared domain, risk, identity and required or
conditional scopes. Normalize/Validate are input-only stages; network effects
belong in Execute. Return a declared result or typed error. Supported dry-runs
describe effects without executing them. Use declared file/stdin inputs for large
or multiline values and preserve IDs, Unicode, empty values and JSON types.
Pagination limits and continuation metadata determine completeness.

Credential providers can handle, skip or explicitly block resolution. Bot and
user identities are different authorization contexts; do not switch identity or
run login to bypass a host policy. Transport providers can intercept requests,
with optional request-class scoping and abort capability. Keep tokens and secrets
out of output. File backends are host-owned: respect their validated paths and
supported operations, including whether exclusive saves are available.

Observers report command execution; wrappers may abort it. Lifecycle hooks handle
startup/shutdown. Restrict rules constrain command domains, paths, identities and
risk, with deny taking precedence. Respect typed failure envelopes and the active
policy; do not assume hidden commands remain callable. High-risk confirmation
requires the user's actual authorization. Success data goes to stdout; typed
errors, diagnostics and audit notices go to stderr. Check exit status and result
fields, preserving false, zero, null, empty string and omitted values distinctly.

Hosts explicitly wire their own embedded skills and command affordance content.
Skill overlays can select, remove or replace content; dependencies and reference
remaps must resolve in the shipped tree. Go embedding cannot import a dependency
module's assets automatically. Declared invalid skill composition fails closed;
ordinary hook failure behavior follows its declared fail-open/fail-closed policy.
