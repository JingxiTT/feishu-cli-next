# Generated business API section

Status: the complete offline section boundary passes. Live verification and
comparative agent trials remain blocked/pending; this section is not release
accepted.

## Scope and ownership

This supplement owns the 17 frozen generated methods under `approval`,
`attendance`, and `mindnotes`, plus their eight routing nodes. The original
`baseline/coverage.json` classified these as foundation because their Cobra
domain annotation was empty. The frozen capture is unchanged; the coordinator's
supplemental reconciliation ledger assigns the business process suite.

Reference: sibling `feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`, frozen
`registrations.json.gz`, `meta_data.json.gz`, and `sources.json`. The sibling has
no approval, attendance, or mindnotes shortcut directories/registrations. These
are intentionally one-to-one supported OpenAPI commands assembled by
`cmd/service`; adding endpoint wrapper shortcuts would duplicate the reference
surface. Existing typed parameter flags, JSON body/file input, and raw `--params`
escape hatch are retained. Frozen metadata is unchanged; coordinated shared service corrections are
listed below.

Owned files: `tests/next_sections/business/business_e2e_test.go` and this report.
The coordinator owns shared fixes, runner/CI registration, and inventory. A
coordinator-delegated follow-up (`1058824`) also changed the owning generic service
read classification/tests and the corresponding receipt contract for three
confirmed Mail POST reads; the Mail owner supplies their process coverage.
Dependencies: accepted P1 `ce68188`, generic service registry/schema/parameter
binder, input FileIO, sidecar fixture, result artifacts, and mutation receipts.

## Complete command mapping

Every row has an explicit independent HTTP method/path, all documented request
body fields and typed URL/query fields, `--data @file` for body methods, identity,
scope rejection, lost response, domain rejection, validation, and dry-run process
test. Schema checks recursively compare all frozen input/output field names and
types, input requirements, and all enum options, including nested objects/arrays.
Response required markers are not emitted by the inherited public output schema.

| Public command | Wire operation | Extra contract coverage |
| --- | --- | --- |
| `approval approvals get` | GET `/approval/v4/approvals/{approval_code}/detail` | locale, definition/form schema, read artifact |
| `approval approvals search` | POST `/approval/v4/approvals/search_launchable` | keyword/locale, integer page size, explicit body cursor continuation |
| `approval instances get` | GET `/approval/v4/instances/detail` | required query instance code, locale, all ID-type options in schema, read artifact |
| `approval instances create` | POST `/approval/v4/instances/initiate` | escaped JSON-string form, uuid, node approver and cc arrays |
| `approval instances cancel` | POST `/approval/v4/instances/recall` | instance code; explicit high-risk confirmation |
| `approval instances cc` | POST `/approval/v4/instances/add_cc` | recipient array, empty comment, ID type |
| `approval instances initiated` | GET `/approval/v4/instances/initiated` | every filter, leading-zero timestamps, bounded/all pages, second-page failure, artifact |
| `approval tasks approve` | POST `/approval/v4/tasks/pass` | leading-zero task ID, escaped form, Unicode/multiline comment |
| `approval tasks reject` | POST `/approval/v4/tasks/refuse` | instance/task IDs and comment |
| `approval tasks transfer` | POST `/approval/v4/tasks/forward` | target ID preserved with matching typed ID-type query |
| `approval tasks query` | GET `/approval/v4/tasks` | every filter/topic, bounded/all pages, second-page failure, artifact |
| `approval tasks add_sign` | POST `/approval/v4/tasks/add_sign` | integer add-sign/approval-method enums including value 3, recipient list |
| `approval tasks rollback` | POST `/approval/v4/tasks/rollback` | multiple node IDs including `START` |
| `approval tasks remind` | POST `/approval/v4/instances/remind` | multiple task IDs and comment; notification action remains explicit |
| `attendance user_tasks query` | POST `/attendance/v1/user_tasks/query` | both identities, employee ID type, explicit false/true query booleans, numeric dates, false overtime flag, partial result lists, raw/typed overlay |
| `mindnotes nodes create` | POST `/mindnote/v1/mindnotes/{mindnote_id}/nodes` | both identities, client token, node/parent IDs, completion boolean, highlight, images, all four text/note element types, styles, readback and explicit replay |
| `mindnotes nodes list` | GET `/mindnote/v1/mindnotes/{mindnote_id}/nodes` | user identity, ID type, full nested content readback and artifact |

Wire paths above omit the common `/open-apis` prefix. Routing nodes are
`approval`, `approval approvals`, `approval instances`, `approval tasks`,
`attendance`, `attendance user_tasks`, `mindnotes`, and `mindnotes nodes`.
All eight help routes and discovery reachability of all 17 leaves are executed.

## Behavior and limits

- Approval methods accept user identity only. Attendance query and mindnotes
  create accept both bot and user; mindnotes list accepts user only. Process
  tests reject unsupported bot identities before business dispatch.
- Frozen approval search and attendance query catalog/schema risks remain
  `write`, as do legacy command annotations. Agent-mode command annotations
  identify their verified read semantics and enable `--save-result`; exact
  request markers avoid mutation receipts. Their artifacts preserve full native
  results, partial employee arrays, continuation and verified user/tenant binding
  despite projected stdout. Arbitrary raw API POST calls retain conservative
  mutation classification. Approval's nine
  high-risk operations require `--yes`, including cancellation, cc, initiation,
  signing, approval, rejection, reminder, rollback, and transfer.
- All bodies remain schema-guided JSON through `--data`, including `@file`.
  None of these 17 methods declares a multipart file field. Image tokens and
  attachment-like rich objects are JSON values, not local upload operations.
  Referenced mindnote `plain_text`, `text_style`, `link`, `mention_user`, and
  `mention_doc` definitions are absent from the frozen catalog. They remain
  arbitrary objects and are forwarded without normalization; fixture members
  demonstrate preservation, not undocumented backend field acceptance.
- Body cursors for approval search remain explicit. Auto pagination applies to
  the documented URL `page_token` methods (initiated instances and task query).
  Tests inspect each request cursor and ensure a bounded result retains its
  continuation and a second-page error does not report successful completion.
- Read artifacts preserve full nested content, escaped forms, identity, and
  continuation despite projected stdout. Large numeric results, leading-zero
  strings, null, empty strings, false, and Unicode are distinct.
- Lost mutating POST responses and arbitrary domain errors are unknown, exit
  22, with one observed request and no automatic replay. Mutation scope rejection
  is confirmed unapplied, exit 20. GET transport may reconnect once after a lost
  response. GETs and the two domain-known generated POST reads retain typed read
  errors without mutation receipts.
- Mindnote pure-create acknowledgement requires one unique nonempty created ID
  for every requested node and no unrecognized response fields. Missing, short,
  duplicate, empty, extra, or wrongly typed IDs remain unknown. Requests with
  any existing `node_id` remain unknown even when IDs appear complete: the
  response describes created nodes and cannot establish update coverage.
  Identical mindnote create/update input is still an explicit request. The CLI
  does not claim a no-op or concurrency guarantee. `client_token` and approval
  `uuid` are preserved; fixture readback is not proof of backend idempotency,
  atomicity, collision behavior, notification delivery, or conditional writes.
- Attendance's successful response can contain `invalid_user_ids` and
  `unauthorized_user_ids`. Those native partial coverage fields remain visible;
  a response acknowledgement cannot imply every requested employee was returned.

## Executable evidence

The suite uses `nextfixture.RunMain(m, "../../..")` to build the production
`authsidecar` binary and `nextfixture.NewFixture` for isolated config/workspace
and loopback synthetic backends. Request handlers independently inspect real
paths, methods, query values and request bodies. They never forward requests.
The test parent uses `scripts/next_test.py`'s `isolated_env`, excludes personal
credentials, and sets an unreachable external proxy. All fixture data is synthetic.

Full section commands:

```sh
python3 scripts/next_test.py --section business --suite all --report .tmp/next-tests/business.json
python3 scripts/next_test.py --section business --suite live --allow-live --report .tmp/next-tests/business-live.json
```

Current complete section execution (2026-09-07): canonical runner registration
`6226e3c`, receipt correction `041a9cd`, verified POST read-artifact correction
`ef9a2c8`, pagination corrections `6056d73`/`94e4afc`, and the exact Mail read
classification follow-up `1058824`. The authsidecar build is
performed by `TestMain` for the process suite.

| Isolated command | Result |
| --- | --- |
| `python3 scripts/next_test.py --section business --suite all --report .tmp/next-tests/business.json` | PASS: shared units 551 pass records, zero failures, one skip; process suite 189 pass records, zero failures, 17 blocked live skips |
| `python3 scripts/next_test.py --section business --suite live --allow-live --report .tmp/next-tests/business-live.json` | BLOCKED, exit 2: no verified authorized fixture harness |

The unit package set is `cmd/service`, `internal/apicatalog`, `internal/registry`,
`internal/schema`, `internal/client`, and `internal/nextattempt`. Registry's
`TestColdStart_NoEmbedded_SyncFetch` skips because the canonical embedded catalog
is present; the no-embedded build scenario is not passing evidence from this run.
The 17 process skips are exactly the named `TestBusinessLive/<method>` cases.
They are not passing live evidence. Go counts include parent tests and subtests,
not only independent scenarios.

Machine-readable full command output is in `.tmp/next-tests/business.json` and
`.tmp/next-tests/business-live.json`. All four later-page faults and all four
missing/repeated continuation cases now pass without saving successful artifacts.
Ten mindnote completeness/update uncertainty cases pass. Independent approval
create→approve/reject/cancel→readback retains exact forms, history, comments,
leading-zero identifiers, and a 64-bit attachment size. The shared generic unit
selection includes inherited reference regressions; no legacy domain E2E package
exists for these three services.
The shared Mail follow-up also passed 487 focused service/cmdutil test records
and `go vet ./cmd/service`. Broader root CI/static checks and comparative
evaluations remain coordinator-owned and were not run here.

The live selector is deliberately blocked until authorized fixtures exist.
`TestBusinessLive` contains a separately named blocked scenario for each of the
17 leaves and never accesses personal configuration. These skips are not passes.

## Shared corrections verified offline

The coordinator fixed generic pagination so later API/transport failures return
nonzero typed errors instead of a merged success. The business suite checks both
approval list methods for later scope rejection and lost response, with no
successful result artifact. Missing and cyclic continuation tokens also return
`internal/invalid_response` and stop before another cyclic request.

The shared receipt owner implemented conservative mindnote created-ID evidence
and exact read-only markers for generated attendance query and approval search.
The business suite verifies full saved artifacts and no receipt files for both
known POST reads, preserves native partial attendance arrays, and separately
checks ten mindnote response completeness/update uncertainty cases. It does not
infer applied item counts for mixed updates or employee coverage from an API
success envelope.

The coordinator-requested frozen-catalog audit examined 251 generated methods,
including 106 POSTs and 92 POSTs with non-read risk. Three additional native Mail
reads were confirmed from exact endpoint, body/response and readonly-scope
contracts: `user_mailbox.message.batch_get`, `multi_entity.search`, and
`user_mailbox.search`. The shared helper corrects only their exact service/ID/
HTTP/path tuples and binds each validated concrete decoded path; neighboring
mutations and raw API calls remain independently classified. Mail's full suite
verifies the corresponding process behavior. IM chat-link generation and Slides
image rendering were not reclassified because their generated-link/artifact
effects need separate domain interpretation. Drive user-event subscription
remains a mutation despite its readonly scope. No name-only or scope-only
classification rule was introduced.

## Outstanding acceptance

No live request was made. Approval lifecycle actions can notify/assign or expose
content to workflow participants. There is no verified private, run-owned
workflow with current account/tenant binding, unambiguously resolved Ying Wu-only
other participants, and complete cleanup. Attendance lacks a verified
current-account employee fixture. Mindnotes lacks a verified private, run-owned
mindnote with established visibility before creation/update. Reads that depend
on those fixtures also remain unverified.

Required live semantics, backend rich-text field validation, notification
outcomes, atomicity/idempotency, and live cleanup remain open. Comparative
development tasks and all three-arm model/token measurements are owned by the
coordinator/evaluator and were not run here. Held-out tasks were not accessed.
