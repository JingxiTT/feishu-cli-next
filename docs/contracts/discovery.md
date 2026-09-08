# Literal command discovery

`discover [query...]` uses the registered visible command tree. Each
whitespace-separated term must appear as a case-insensitive substring somewhere
in the command path, aliases, short description, or public flag names and usage.
Terms need not appear in order. `*` permits every visible command. An empty query
lists top-level command groups. A known domain's `--help` is also a direct route.

Matching includes public local and inherited flags even without `--flags`.
Hidden commands/subtrees and hidden flag names/descriptions are excluded. Flag
default values are not search text. Results remain alphabetical by command path,
with stable offset pagination; matching adds no rankings or inferred synonyms.

`--flags` includes each public flag's name, type, default, required marker, and
optional `description` copied verbatim from its registered usage. Descriptions
include accepted selector values and input conventions where the command defines
them, so a focused lookup can often replace a separate command-help read. Hidden
flags remain excluded and local definitions shadow inherited flags. The default
output without `--flags` is unchanged. Narrow the query and `--limit` before
requesting flags; a broad listing repeats inherited flag descriptions. Command
help and domain skills remain available for examples and workflow guidance.

Local command registration uses the startup brand, and strict-mode discovery
uses provider identity-support metadata without verifying user or tenant identity.
Actual authorization and platform requests retain their normal credential and
identity verification paths.

A synthetic regression query such as `contact union_id` now reaches `+get-user`
through the supported values in its `--user-id-type` usage. This addresses a
search-surface gap identified while reviewing Contact evaluation friction; it
does not establish a reduction in agent calls or tokens. That requires fresh
trials with the changed binary and skill.
