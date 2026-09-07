---
name: lark-okr-concise
description: Read and manage Feishu OKR cycles, objectives, key results, alignments, quantitative indicators, progress records and comments. Use Task for to-dos, Calendar for scheduling, and API discovery for performance reviews.
---

# OKR

Use the supplied resource IDs when available. Otherwise resolve the current user
with `contact +get-user`, or another person with `contact +search-user`, then use
`okr +cycle-list --user-id <user_id>` and `okr +cycle-detail --cycle-id <cycle_id>`.
A name alone is not a stable identity. IDs are positive int64 decimal strings;
keep their exact digits. Use `--as user` for the user's OKRs unless another
identity was requested; bot access to others' OKRs requires the appropriate
permissions. Keep identity and ID type consistent across a workflow. Read
`lark-shared` for authentication, permission handling and global safety rules;
check exact flags and scopes through help/schema when needed.

Cycle-list returns one page; its `--time-range` is a local filter on that page.
Continue with the returned `page_token` when `has_more` is true. Current cycles
must overlap the current time; status alone does not prove a cycle is current.
Cycle-detail traverses objectives and their KRs. For an already known objective,
`okr objectives get --objective-id <id>` reads it and
`okr objective.key_results list --objective-id <id>` lists its KRs. Native lists
support pagination; do not present a bounded page as the complete collection.

Distinguish score, quantitative indicators and written progress. `score` is an
assessment from 0 to 1 with at most one decimal; use `+patch --score` only when
scoring is explicitly requested. Numeric progress usually belongs to an indicator;
written updates are progress records. Read indicators using
`objective.indicators list --objective-id <id>` or
`key_result.indicators list --key-result-id <id>`. These return a single
`indicator` object, including its unit, current and target values. Do not infer
completion percentage by reading a score or assume every indicator uses percent.

```bash
lark-cli okr +progress-list --target-type key_result --target-id <key_result_id> --as user
lark-cli okr +progress-get --progress-id <progress_id> --as user
```

Progress-list reads one v2 page of the target's history; it accepts target types
`objective` and `key_result`, user/department ID types, page size 1–100 and an
opaque page token. Progress-get reads a specific v1 record. Shortcut results use
`progress_rate.percent` as a number and status `normal`, `overdue` or `done`;
raw v2 fields are `progress_percent` and `progress_status`. Missing values and
numeric values are distinct. History records do not by themselves establish an
undocumented latest-record ordering. Content may be a serialized ContentBlock
JSON string. `simple` converts content to semi-plain text; `richtext` preserves
supported native content for detailed work.

For one objective or KR use `+create --level objective|key-result`; supply a cycle
for an objective and an existing objective for a KR. For a group use
`+batch-create --cycle-id <id> --input <json-or-file>` with objective text,
optional notes/mentions/category and nested KRs. It attempts rollback on failure;
inspect any remaining IDs and errors before recovery. Add a category only when
requested or tenant validation requires one: `okr categories list` with user
ownership identifies enabled choices. `+patch` changes supplied content, notes,
score or deadline; notes apply only to objectives. Do not replace omitted fields.

`+reorder` fills a complete order from requested positions. `+weight` normalizes
specified and remaining weights. Native position/weight replacement requires the
entire relevant collection, with weights summing to one. Both objective and KR
levels are available. `+indicator-update --level ... --id ... --value ...` updates
only current value. For units, target values, initial values or calculation mode,
read the indicator and use `indicators patch` with its actual ID. Indicator
status and current value can be changed only in their respective manual modes.
Objectives cannot change start value, target value or unit. KRs with inherited
records cannot change target value or unit; objective and KR calculation modes
differ.

Use `+progress-create` for a new written update, `+progress-update` for an existing
record, and `+progress-delete` only for an intended irreversible deletion. Content
is required; percentage and status flags must be supplied together when used.
Numbers can be negative or exceed 100 where the indicator semantics allow it;
interpret them using the unit. Creation supports source title and URL. Use
`+upload-image` with its actual target before referencing the returned token in
rich content. Supported input is simple JSON with text/mentions or native
ContentBlock JSON via `--style richtext`; file/stdin inputs avoid shell escaping.
Native content includes paragraphs, lists, formatting/colors, mentions, document
links and galleries. Progress layout allows one image per gallery block; use
separate galleries for multiple images.

For comments, `+comment-list` reads a page for a cycle, objective, KR or progress;
threads may span pages. `+comment-detail` aggregates the cycle's targets and
threads; `+comment-get` reads a comment ID. Writes require user identity.
`+comment-create` supports cycle/progress comments and objective/KR selections:
use selected text, `--select-all`, or an existing comment ID to reply. A reply's
`--ref-comment-id` is a comment ID, not `selection.id`. `+comment-patch` changes
only nonempty content, preserving target, selection and references.
`+comment-solve` and `+comment-reopen` change comment/thread status; solving is
reversible and is not deletion. `+comment-delete` permanently removes one comment.

Alignment reads and creation use `objective.alignments`; inspect or delete an
alignment through `alignments`. Source and target cannot be the same objective,
their cycles must overlap, and source editing permission is required. Objective,
KR, progress, comment and alignment deletion is destructive: confirm the intended
target and authorization, preview with `--dry-run`, and use `--yes` when required.
Mentions and comments can notify people; preserve the user's intended recipients.
A transport failure after a write does not prove nothing changed; inspect backend
state and rollback evidence before retrying.

This is concise English coverage of the frozen official `lark-okr` skill and its
entity, content, cycle, creation, patch, order/weight, indicator, progress, image,
comment and alignment references. Exact schemas remain available through CLI help
and schema discovery; no extra product restrictions are introduced here.
