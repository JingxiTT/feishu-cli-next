# Executable read-only fixture data v1

This is deliberately a small signed HTTP response table, not a general API emulator.
A section owner may normalize their reviewed contract into `<section>.json`.
Do not change the original contract, common Python adapter or another section.
No model runs or product edits. Do not stage/commit; evaluator integrates.

The JSON document has these fields:

- `schema_version`:1; `section`; `source_contract`: `evaluation/contracts/<section>.json`.
- `task`: the harness task `{id,section,prompt,fixture:{scenario,answer_schema},expected:{answer,required_reads,unordered_answer_paths}}`.
- `task.fixture` contains only scenario name and a model-facing output TYPE schema.
  Never include response data, expected answers, schema const/enum values that reveal
  answers, hidden native state or backend instructions. Keep exact values only in
  `task.expected`. `required_reads` is a list of alternative route-ID groups: at
  least one successful route in EACH group must be observed independently.
  `unordered_answer_paths` optionally lists top-level answer array keys whose order
  is not part of the task (default all list order significant).
- `routes`: list of `{id,method,path,request,response}`. Paths are exact strings.
  `method` may include POST only for an existing read endpoint. No mutable cases.
  `request` is optional; accepted keys are `query` (required key → string or list
  of alternative single values), `optional_query` (if present must equal an allowed
  value), `body_contains` (recursive subset object; list values match exactly).
  Use multiple route variants with the same `id` to represent equivalent read forms.
  Missing body selectors must not silently return the target of a different read.
  Unknown query keys are allowed unless a handler needs them for fidelity; do not
  use this static table when a query changes content/pagination and cannot be
  represented faithfully with a small list of variants.
- `response`: `{http_status:200,json:<full API envelope>}` or
  `{http_status:200,text:<content>,content_type:<mime>}`. Use actual tested public
  shapes. API response fields must not invent capabilities.
- `examples`: list of actual CLI argv arrays excluding binary; at least one proven
  path for each supported strategy. Examples are controlled tests, not model prompt.
- `limitations`: list explaining bounded fixture scope and any excluded variants.

Include the existing user_info identity GET response if required. A standard
synthetic app/default user binding is supplied equally by the signed sidecar;
both user and bot reads may be used unless the public API requires one identity.
No outbound forwarding exists. Unlisted requests are rejected and ledgered. The
independent verifier checks a typed exact answer plus required successful signed
reads and absence of rejected requests/config changes. It never trusts an agent
receipt. Models can choose help/schema/skills/raw APIs/commands and shell freely.

If this static interface cannot faithfully express your task, report the required
small extension instead of weakening semantics. Do not guess a query response or
mark a contract executable without evaluator-controlled binary tests.

Supported small selector extensions (add only where the public API needs them):
`request.identities` is an allowed list of `user`/`bot`; `forbidden_query_keys`
and `forbidden_body_keys` reject present top-level keys; `optional_body` maps a
present body key to a list of allowed scalar values. `query_keys` optionally lists
all permitted query keys (extra keys then reject). These are validation predicates,
not response transformation rules. A missing body is treated as an empty object.
