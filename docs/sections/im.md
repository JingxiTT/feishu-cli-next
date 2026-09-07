# IM section evidence

Status: implementation and complete offline section gate passed. Live
backend guarantees and comparative evaluation remain open, not passing evidence.

## Ownership and reference

Own `shortcuts/im/`, `affordance/im.md`, `tests/cli_e2e/im/`,
`tests/next_sections/im/`, and this report. Shared runtime, generated service
assembly, output/artifact/receipt contracts belong to the coordinator; all IM
events belong to the events owner. Reference sibling is unchanged at
`7fd6ef3c07182257ce776cdc5a614e122d5bd4b3`. Before changes, recursive source
comparison found only the accepted synthetic-network fixture adjustment in
`helpers_network_test.go`. Every existing implementation is retained; no raw
forwarding substitute or product recipient restriction was introduced.

The frozen inventory contains 84 IM tree entries: 24 shortcuts, 45 generated
API methods and 15 domain/resource nodes. API contracts come from
`docs/reference/meta_data.json.gz`, registrations from
`docs/reference/registrations.json.gz`; all flag, identity, risk and schema
metadata is retained except the intentional additive changes below.

## Intentional changes

- `+messages-edit --skip-unchanged` requires a successful read of exactly the
  requested, undeleted message and compares the complete text/post JSON value.
  JSON object key order and whitespace do not matter; language, content order,
  formatting, attachments, null/empty/omitted distinctions and numeric strings
  do. The flag is opt-in because read permission is additional to editing.
  Failed/unusable reads stop before PUT. Equal content emits `unchanged:true`
  without a message write or receipt. Different content preserves the original
  replacement payload and normal mutation evidence.
- `+chat-update --skip-unchanged` compares only explicitly supplied name/description
  fields after a successful GET. Omitted response fields stop the update.
  `--description ""` explicitly clears the description; omission preserves it.
- These are point-in-time checks without remote compare-and-swap. Another client
  can edit after the read. Raw generated replacement/card APIs retain their
  original exact endpoint semantics and do not promise no-op detection.
- Message sends, replies, forwards, flags, feed insertion, invitations and urgent
  notifications are effects, not replacements. Repeated calls remain intentional
  writes; only user-supplied backend idempotency keys are forwarded. No guessed
  send deduplication or unknown-outcome replay was added.
- Existing Markdown/media resolution runs before content comparison and may
  upload assets. Message no-op evidence does not promise no asset upload.
  Omitted attachment zone retains the original server preservation behavior;
  explicit clear sends `files:[]`. A clear-only edit retains the reference's
  empty post-body semantics; it is not a body-preserving attachment patch.

## Complete mapping and acceptance scenarios

All shortcuts retain user/bot restrictions, rich content conversion, ID checking,
FileIO path policy, scopes, dry-run and existing uncommon flags. Shared discovery
and output controls apply automatically. Existing units cover aliases, paging,
search filters, sender names, muted chats, attachments, multipart/media/network
limits, converted cards/folders/merged messages, reactions and expanded threads.

| Capability group | Preserved commands and behavior | Process evidence |
| --- | --- | --- |
| Message composition | `+messages-send`, `+messages-reply`: text, localized post, markdown, image/file/audio/video and cover, interactive, share_chat/share_user; mentions, attachments and explicit UUID | all 9 send types, repeated sends, threaded reply UUID/content, recipient/conflict validation, unknown send without retry |
| Replacement | `+messages-edit`: text/post, attachment set/clear; `+chat-update`: name/description | exact rich/text no-op, changed body, explicit clear, failed/malformed preflight, dry-run, write-only compatibility |
| Message reads | `+messages-mget`, `+chat-messages-list`, `+threads-messages-list`, `+messages-search`, `+messages-read-status`, `+message-read-users` | process routes/empty boundaries plus existing full conversion/enrichment/paging units; read POST commands do not create mutation receipts |
| Chats | `+chat-create`, `+chat-list`, `+chat-members-list`, `+chat-search` | private/topic request, invitation/owner IDs, creation share-link follow-up, exact bounded pagination and full artifact preservation; member/search filters |
| Personal flags | `+flag-create`, `+flag-cancel`, `+flag-list` | request identity, read-only list and two-layer partial cancellation preserving every result |
| Feed shortcuts/groups | `+feed-shortcut-create`, `+feed-shortcut-remove`, `+feed-shortcut-list`, `+feed-group-list`, `+feed-group-list-item`, `+feed-group-query-item` | head/tail identity, writes/read markers; group cursor/time/enrichment units retained |
| File resources | `+messages-resources-download`; images/files create and files folder | multipart uploaded bytes and fields; range-first downloaded bytes/output file; existing bounded continuation, cleanup/path/content tests |
| Generated lifecycle | 45 methods below | 43 static synthetic JSON request cases + 2 multipart byte cases through real CLI; params/body/identity/method/path/output observed independently |
| Partial outcomes | urgent targets, pending membership, batch user settings, flag two-layer cleanup | failed recipient payloads remain visible with projection; batch/unknown conservatism and multiple request receipts |
| Events | receive message, card action, native catalog | delegated to events owner; existing `events/im` regressions included by section runner |

The static generated request cases use only synthetic IDs/content and exercise
all methods through the actual service metadata and CLI process. They verify
wire contracts and preserved effects; they do not emulate all backend business
rules or prove tenant visibility, actual invitations, notifications or indexing.

| Generated command | Method and endpoint |
| --- | --- |
| `im chat.join_requests handle` | `POST /open-apis/im/v1/chats/{chat_id}/membership_applications/handle` |
| `im chat.join_requests list` | `GET /open-apis/im/v1/chats/{chat_id}/membership_applications` |
| `im chat.managers add_managers` | `POST /open-apis/im/v1/chats/{chat_id}/managers/add_managers` |
| `im chat.managers delete_managers` | `POST /open-apis/im/v1/chats/{chat_id}/managers/delete_managers` |
| `im chat.members bots` | `GET /open-apis/im/v1/chats/{chat_id}/members/bots` |
| `im chat.members create` | `POST /open-apis/im/v1/chats/{chat_id}/members` |
| `im chat.members delete` | `DELETE /open-apis/im/v1/chats/{chat_id}/members` |
| `im chat.members get` | `GET /open-apis/im/v1/chats/{chat_id}/members` |
| `im chat.moderation get` | `GET /open-apis/im/v1/chats/{chat_id}/moderation` |
| `im chat.moderation update` | `PUT /open-apis/im/v1/chats/{chat_id}/moderation` |
| `im chat.nickname delete` | `DELETE /open-apis/im/v1/chats/{chat_id}/nickname` |
| `im chat.nickname get` | `GET /open-apis/im/v1/chats/{chat_id}/nickname` |
| `im chat.nickname update` | `PUT /open-apis/im/v1/chats/{chat_id}/nickname` |
| `im chat.user_setting batch_query` | `POST /open-apis/im/v1/chat_user_setting/batch_query` |
| `im chat.user_setting batch_update` | `POST /open-apis/im/v1/chat_user_setting/batch_update` |
| `im chats create` | `POST /open-apis/im/v1/chats` |
| `im chats get` | `GET /open-apis/im/v1/chats/{chat_id}` |
| `im chats link` | `POST /open-apis/im/v1/chats/{chat_id}/link` |
| `im chats update` | `PUT /open-apis/im/v1/chats/{chat_id}` |
| `im feed.groups batch_add_item` | `POST /open-apis/im/v1/groups/{feed_group_id}/batch_add_item` |
| `im feed.groups batch_query` | `POST /open-apis/im/v1/groups/batch_query` |
| `im feed.groups batch_remove_item` | `POST /open-apis/im/v1/groups/{feed_group_id}/batch_remove_item` |
| `im feed.groups create` | `POST /open-apis/im/v1/groups` |
| `im feed.groups delete` | `DELETE /open-apis/im/v1/groups/{feed_group_id}` |
| `im feed.groups update` | `PUT /open-apis/im/v1/groups/{feed_group_id}` |
| `im files create` | `POST /open-apis/im/v1/files` |
| `im files folder` | `GET /open-apis/im/v1/files/{file_key}/folder` |
| `im images create` | `POST /open-apis/im/v1/images` |
| `im messages delete` | `DELETE /open-apis/im/v1/messages/{message_id}` |
| `im messages forward` | `POST /open-apis/im/v1/messages/{message_id}/forward` |
| `im messages merge_forward` | `POST /open-apis/im/v1/messages/merge_forward` |
| `im messages patch` | `PATCH /open-apis/im/v1/messages/{message_id}` |
| `im messages read_status` | `POST /open-apis/im/v1/messages/read_status` |
| `im messages read_users` | `GET /open-apis/im/v1/messages/{message_id}/read_users` |
| `im messages urgent_app` | `PATCH /open-apis/im/v1/messages/{message_id}/urgent_app` |
| `im messages urgent_phone` | `PATCH /open-apis/im/v1/messages/{message_id}/urgent_phone` |
| `im messages urgent_sms` | `PATCH /open-apis/im/v1/messages/{message_id}/urgent_sms` |
| `im pins create` | `POST /open-apis/im/v1/pins` |
| `im pins delete` | `DELETE /open-apis/im/v1/pins/{message_id}` |
| `im pins list` | `GET /open-apis/im/v1/pins` |
| `im reactions batch_query` | `POST /open-apis/im/v1/messages/reactions/batch_query` |
| `im reactions create` | `POST /open-apis/im/v1/messages/{message_id}/reactions` |
| `im reactions delete` | `DELETE /open-apis/im/v1/messages/{message_id}/reactions/{reaction_id}` |
| `im reactions list` | `GET /open-apis/im/v1/messages/{message_id}/reactions` |
| `im threads forward` | `POST /open-apis/im/v1/threads/{thread_id}/forward` |

## Validation and remaining evidence

Shared fixes consumed: `6bec646` preserves no-op markers under projection;
`cfb5afa` conservatively classifies endpoint-specific partial recipient and
pending membership evidence. Neither fabricates successful delivery counts.

Section command: `python3 scripts/next_test.py --section im --suite all --report .tmp/im-all.json`. Process-only during development uses `isolated_env`
and `go test -json -count=1 ./tests/next_sections/im`. The real authsidecar build
runs with separate synthetic account/tenant and local backend for every test.
No test forwards a request to Feishu. Legacy live workflows are not selected.

Live scenario `TestLiveIMBoundary` is separately selectable and explicitly
blocked. To execute actual lifecycle guarantees, first verify effective current
account and tenant, private creation visibility, and the unambiguous authorized
Ying Wu recipient where needed. No group deliveries or other recipients are
permitted during testing. Existing legacy chat creation helpers leave chats and
lack the necessary identity/visibility proof, so they must not run with personal
credentials. Search indexing, live permissions, true notification delivery,
backend idempotency, rich read-after-write normalization and cleanup remain
unverified. This boundary is testing policy only, not shipped behavior.

Comparative development tasks are scheduled by the coordinator. No held-out
workload was accessed and no token-efficiency claim is made here.

Final section command above passed: 1,294 unit checks, 45 legacy deterministic
E2E checks, and 107 new process checks, with one explicitly blocked live scenario.
These are Go test/subtest events, not counts of distinct backend guarantees.
Raw machine-readable evidence is `.tmp/im-all.json`; the reproducible runner
records exact commands, results, exclusions and skips. Affordance/service/schema
checks passed (275 events), as did scoped `go vet ./shortcuts/im/...
./tests/next_sections/im` and owned-file `git diff --check`.

The one reference deterministic regression change replaces a whitespace-dependent
JSON assertion with explicit field existence and boolean false, preserving the
contract under compact output. Full repository formatting/lint/license/release
gates remain coordinator-owned. Live and comparative work above remain open.
