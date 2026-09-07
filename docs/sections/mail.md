# Mail section

Status: complete offline section suite passed; required live and comparative
acceptance remain open.
Reference: unchanged sibling `../feishu-cli-review` at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`; frozen registrations,
service catalog and source inventory in `docs/reference/`.

## Ownership and complete mapping

Mail owns `shortcuts/mail/`, matching command affordance (none in the frozen source), inherited
`tests/cli_e2e/mail/`, and `tests/next_sections/mail/`. Shared runtime,
result artifacts, receipts and generated API dispatch remain foundation-owned.
Event projection/stream routing is coordinated with Events; Mail retains the
existing `+watch` implementation. Drive uploads, IM sharing, calendar MIME and
signature profile lookup are cross-section dependencies.

All 102 frozen command nodes remain mapped without renaming: 29 workflow
shortcuts, 57 catalog API methods and 16 domain/resource groups. Every frozen
node is exercised through real-process help, including its visible options.
The authsidecar test build intentionally hides local `--profile`; normal-build
profile semantics belong to foundation validation.

| Workflow shortcuts | Preserved behavior and primary unit evidence |
| --- | --- |
| `+message`, `+messages`, `+thread` | HTML/plain/metadata bodies, recipient and inline/regular attachment metadata, requested order, missing IDs, draft identities, thread date order and spam/trash option; `mail_messages_test.go`, `mail_shortcut_test.go`, `helpers_test.go` |
| `+triage`, `+watch` | Search/list filters, system/custom folders and labels, bounded result count and continuation, payload formats, watch filtering/output/fetch failures and shutdown; `mail_triage_test.go`, `mail_watch_test.go` |
| `+message-modify`, `+message-trash` | 20-ID batching, normalization, custom folder/label validation, no requested change, partial batch failures and high-risk confirmation; `mail_message_manage_test.go` |
| `+rule-list`, `+rule-get`, `+rule-create`, `+rule-update`, `+rule-delete`, `+rule-enable`, `+rule-disable`, `+rule-reorder` | Complete condition/action grammar, typed JSON and files, unknown-field preservation, order validation, state no-op, readback failure evidence and confirmation; `mail_rules_test.go` |
| `+send`, `+reply`, `+reply-all`, `+forward`, `+draft-create`, `+draft-edit`, `+draft-send` | Mailbox/from priority, display names and repeated To/Cc/Bcc, draft-first versus explicit send, body files, HTML lint, MIME/inline/large attachments, signatures, templates, quote/reply references, calendar and read receipt headers, scheduled compose send, sequential send abort/partial effects; compose/edit/send tests and `draft/`, `emlbuilder/`, `ics/`, `filecheck/` suites |
| `+send-receipt`, `+decline-receipt`, `+share-to-chat` | Receipt-request eligibility and destinations, MDN content, mailbox identity, message/thread addressing and explicit receiver type; corresponding shortcut tests |
| `+signature`, `+template-create`, `+template-update`, `+lint-html` | Signature language/rendered variables, typed template fields, recipient clears, attachments/CIDs, HTML/body file handling, native regions and lint detail selection; signature/template/lint tests and `signature/`, `lint/` suites |

The generated methods below use their existing catalog schemas and generic
service machinery. `catalog_requests.json` contains synthetic typed required
inputs derived from the frozen catalog. `TestMailCatalogRequests` dispatches
all 57 through the actual CLI and independently asserts method, path, body and
lossless response values. This establishes request plumbing, not live backend
semantics or every optional field combination; the unchanged catalog and
reference tests remain part of the mapping.

| Resource (after `mail`) | Methods |
| --- | --- |
| `multi_entity` | `search` |
| `user_mailbox.drafts` | `cancel_scheduled_send`, `create`, `delete`, `get`, `list`, `send`, `update` |
| `user_mailbox.event` | `subscribe`, `subscription`, `unsubscribe` |
| `user_mailbox.folders` | `create`, `delete`, `get`, `list`, `patch` |
| `user_mailbox.labels` | `create`, `delete`, `get`, `list`, `patch` |
| `user_mailbox.mail_contacts` | `create`, `delete`, `list`, `patch` |
| `user_mailbox.message.attachments` | `download_url` |
| `user_mailbox.messages` | `batch_get`, `batch_modify`, `batch_trash`, `get`, `list`, `modify`, `send_status`, `trash` |
| `user_mailbox.rules` | `create`, `delete`, `list`, `reorder`, `update` |
| `user_mailbox.sent_messages` | `get_recall_detail`, `recall` |
| `user_mailbox.settings` | `send_as` |
| `user_mailbox.template.attachments` | `download_url` |
| `user_mailbox.templates` | `create`, `delete`, `get`, `list`, `update` |
| `user_mailbox.threads` | `batch_modify`, `batch_trash`, `get`, `list`, `modify`, `trash` |
| `user_mailboxes` | `accessible_mailboxes`, `profile`, `search` |

## Changes and semantics

Existing complete domain implementations are reused. Draft replacements now
compare the original and patched MIME using the same serializer before PUT.
Identical supported replacement results return `no_op:true` and do not write.
Unchanged plain/HTML bodies keep their original MIME entity instead of marking
it dirty, and a matching decoded Unicode subject retains its encoded header. This comparison avoids treating base64 padding or line endings in
the backend transport representation as user changes. It is deliberately
conservative: append, uploads, attachment removals, signatures and calendar
operations keep their explicit effects. Changed content still uses the existing
MIME-safe pipeline and preserves untouched attachments and custom headers.

Template updates similarly skip PUT when their complete typed representation
is unchanged and no attachments were uploaded. Rule replacement/state no-op is
reused from the reference. None of these checks provides optimistic locking;
concurrent writers can overwrite each other after the read.

Mail reads inherit compact lossless JSON, projection and full result artifacts.
A 21-ID process fixture verifies two bounded batch reads, stable order, a missing
ID, rich HTML retained in the artifact and no receipt on the POST read. Triage
exposes its returned count, mailbox and continuation rather than claiming a
complete mailbox snapshot. Generic artifacts are not draft/template patch files.

The generated `user_mailbox.messages batch_get`, `multi_entity search`, and
`user_mailboxes search` methods are also reads. Their frozen metadata declares
POST/write despite read-only native requests and scopes. The shared generated
dispatcher corrects only their exact service/method/path tuples
in next mode, marks the concrete mailbox request as read-only, and preserves
the frozen metadata and legacy command risk. Ordinary reads exit successfully
without mutation receipts, and `--save-result` preserves the full response.
Neighboring `batch_modify` and raw `api POST` retain conservative mutation
classification. `generated_read_e2e_test.go` proves both user and bot identities,
a substituted mailbox address, projected output/full typed artifacts, both
user-only search methods and their filters/pagination, no read receipts, and
receipts for the two negative controls.

Send operations are never deduplicated from equal content. Process fault
injection checks one dispatch for accepted, scope-rejected, malformed and lost
responses, plus partial send and account-level abort. Shared receipts describe
API acknowledgement, not downstream delivery. Asynchronous/batch uncertainty
retains the foundation's conservative classification and never authorizes replay.

## Acceptance scenarios and commands

Run the complete section boundary with:

```bash
python3 scripts/next_test.py --section mail --suite all --report .tmp/mail-generated-read-boundary.json
```

This strips developer credentials, isolates config/home, prevents ordinary
external HTTP, runs every Mail unit/subpackage, selects inherited deterministic
`DryRun|Regression` E2E, then builds the real authsidecar process suite against
non-forwarding local backends. The JSON report records exact commands and
pass/fail/skip events. All 29 shortcuts have written request/local process
scenarios; discovery alone is not counted as behavior verification.

The process suite additionally covers draft inspect/no-op/MIME preservation,
rule/template omission preservation, identity/invalid input, bounded reads and
artifact handoff, unknown outcomes and partial effects. Unit tests retain the
broader uncommon input/type and reference MIME golden matrix.

## Recorded offline evidence

Final boundary command above passed on 2026-09-07. Machine-readable execution
and selection evidence is in `.tmp/mail-generated-read-boundary.json`
(regenerate with that command). This includes the generated POST read correction.

| Suite | Passing test events | Failures | Skips |
| --- | ---: | ---: | ---: |
| All Mail units and MIME/ICS/signature/lint subpackages | 1643 | 0 | 0 |
| Inherited deterministic process regressions | 14 | 0 | 0 |
| Next process suite (all 102 nodes, 57 APIs and 29 shortcut surfaces) | 215 | 0 | 1 |

Counts include Go subtests. The next process skip is the explicitly blocked
live fixture. The runner deliberately excludes the three inherited live tests
`TestMail_DraftLifecycleWorkflowAsUser`, `TestMail_DraftSendWorkflowAsUser`, and
`TestMail_SendWorkflowAsUser`; none was run with personal credentials.

A later controlled development evaluation exposed generated `batch_get`
returning data with exit 22 and an unknown mutation receipt. The new public
process regression reproduced both identities and the separate artifact
preflight rejection (exit 2) before the fix; `.tmp/mail-generated-read-before.json`
records 7 failing and 3 passing test events, including parent subtests.
Both native searches also reproduced exit 22 before the fix;
`.tmp/mail-search-reads-before.json` records their 3 failing test events.
The previous catalog dispatch test accepted exit 22 for asynchronous mutations
and therefore did not establish the read classification of every POST method.
The dedicated regression closes that gap without relaxing mutation outcomes.

`go vet ./shortcuts/mail/...` passed in `isolated_env`; changed Go files pass
`gofmt -l`, and owned paths pass `git diff --check`. A full no-index comparison
against the sibling Mail directory confirms all other reference implementations
are unchanged; expected diff exit 1 reports only the owned changes listed here.

Changed implementation: `draft/patch.go`, `mail_draft_edit.go`,
`mail_template_update.go`, `mail_watch.go`. Nearby regression tests:
`draft/patch_noop_test.go`, `mail_draft_noop_test.go`. Written process suite:
`tests/next_sections/mail/mail_e2e_test.go`, `generated_read_e2e_test.go`, and
the frozen synthetic `catalog_requests.json`; this section report records mapping
and acceptance. The generated read correction is implemented in the
foundation-owned `cmd/service/request_effect.go` and `cmd/service/service.go`.

## Outstanding acceptance

- Live mailbox lifecycle is separately selectable as
  `TestMailLivePrivateMailboxLifecycle` and explicitly skipped until a verified
  fixture binds the effective account and tenant, current-account private draft
  visibility and self-only recipients (including Cc/Bcc, reply-all, calendar
  attendees and receipt/share targets). Other-recipient delivery is restricted
  to unambiguously verified Ying Wu. No live credentials or legacy live
  workflows were used. A skip is not passing live evidence.
- Live send/delivery/scheduling/recall, mailbox permissions, download/upload,
  template/rule backend behavior and WebSocket reconnection/shutdown require
  that harness and run-owned cleanup. Synthetic route success does not verify
  those backend guarantees. No unrelated resource is edited or deleted.
- Shared stream result artifacts are unavailable; watch's existing NDJSON and
  output-directory contracts remain. Watch subscription lifecycle calls use the shared exact-request mutation marker;
  a process setup-failure scenario verifies subscribe and unsubscribe receipts
  without treating event reads as writes.
- Comparative development evaluation is scheduled by the coordinator. No
  held-out task was used, and no token-efficiency claim is made.
- Broad release quality gates and normal-build reconciliation are coordinator
  responsibilities; this report does not claim their completion.
