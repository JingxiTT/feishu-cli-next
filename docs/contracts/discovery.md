# Literal command discovery

`discover [query...]` uses the registered visible command tree. Each
whitespace-separated term must appear as a case-insensitive substring somewhere
in the command path, aliases, short description, or public flag names and usage.
Terms need not appear in order. `*` permits every visible command. An empty query
lists top-level command groups. A known domain's `--help` is also a direct route.

Matching includes public local and inherited flags even without `--flags`.
Hidden commands/subtrees and hidden flag names/descriptions are excluded. Flag
default values are not search text. Results are ordered by where the terms
matched: every term in the command path first, then in the path, aliases, or
short description, then only in public local flags, then only in inherited flags.
Within a tier, exact path-segment matches come first, then shortcuts (leaves
starting with `+`), then fewer path segments, then shorter command paths, then
command path alphabetically. A term counts as a path or summary match only at the
start of a word, so `read` inside `spreadsheet` ranks as a weaker substring match.
An empty query or `*` lists results alphabetically. The order is deterministic,
so offset pagination stays stable. Matching adds no
inferred synonyms. A query with no matches returns an empty result with a
`meta.hint` that suggests fewer or shorter keywords.

`--flags` includes each public flag's name, type, default, required marker, and
optional `description` copied verbatim from its registered usage. It also
includes one compact `example` route when an authored command example exists or
required flags form an executable placeholder skeleton. An example is
illustrative, not a required workflow; command help and domain guidance retain
alternate routes and complex semantics. Descriptions
include accepted selector values and input conventions where the command defines
them, so a focused lookup can often replace a separate command-help read. Hidden
flags remain excluded and local definitions shadow inherited flags. An entry's
`flags` lists its local flags. On a page with two or more entries, inherited
flags that every entry shares, identical in name, type, default, required marker,
and description, are listed once in `meta.common_flags`; any other inherited flag
stays in its entry's `flags`. Each flag is recoverable from `flags` or
`common_flags`. The default output without `--flags` is unchanged, and
`common_flags` is omitted when the page has fewer than two entries or no shared
flags.

For the token-oriented route view, combine `--flags` with the existing
`--result-fields command,example`; the full result remains available when the
task needs complete flag descriptions, missing-option semantics, or alternate
workflows.

`--result-fields command,plan` opts into a command planning view without needing
`--flags`. Each returned entry's `plan` includes its description (the full long
description when present), usage with positional arguments, risk, identities,
public local and domain-inherited flags, an example when available, and complete
authored affordance guidance. Generated API methods additionally include the
same request-only envelope as `schema <method> --input-only`, preserving required
fields, carriers, enums, bounds, scopes and danger metadata. The response schema
is available through the full `schema` command. Root inherited flags are omitted
from plans and remain available through `discover <command> --flags`. A local or
domain override of a root flag remains in the plan. Pagination metadata is
unchanged; only the returned page is described. Plans require no identity lookup
or command execution. Existing discovery output is unchanged unless `plan` (or
a dotted path beneath it) is explicitly selected.

Local command registration uses the startup brand, and strict-mode discovery
uses provider identity-support metadata without verifying user or tenant identity.
Actual authorization and platform requests retain their normal credential and
identity verification paths.

A synthetic regression query such as `contact union_id` now reaches `+get-user`
through the supported values in its `--user-id-type` usage. This addresses a
search-surface gap identified while reviewing Contact evaluation friction; it
does not establish a reduction in agent calls or tokens. That requires fresh
trials with the changed binary and skill.
