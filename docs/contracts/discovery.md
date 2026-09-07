# Literal command discovery

`discover [query...]` uses the registered visible command tree. Each
whitespace-separated term must appear as a case-insensitive substring somewhere
in the command path, aliases, short description, or public flag names and usage.
Terms need not appear in order. `*` permits every visible command. An empty query
lists top-level command groups. A known domain's `--help` is also a direct route.

Matching includes public local and inherited flags even without `--flags`.
Hidden commands/subtrees and hidden flag names/descriptions are excluded. Flag
default values are not search text. `--flags` retains its existing optional
output shape; matching adds no output fields, rankings, or inferred synonyms.

A synthetic regression query such as `contact union_id` now reaches `+get-user`
through the supported values in its `--user-id-type` usage. This addresses a
search-surface gap identified while reviewing Contact evaluation friction; it
does not establish a reduction in agent calls or tokens. That requires fresh
trials with the changed binary and skill.
