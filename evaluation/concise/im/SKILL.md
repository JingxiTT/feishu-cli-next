---
name: lark-im
version: 1.0.0
description: Use lark-cli for messages, chats, threads, resources, reactions, urgent notifications, interactive cards, flags, feed shortcuts, and feed groups.
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: lark-cli im --help
---

# IM

Read [lark-shared](../lark-shared/SKILL.md) first for authentication, global flags, permissions, and safety. Prefer a shortcut below; inspect its `--help` and linked reference for exact flags. Before a native command, run `lark-cli schema im.<resource>.<method>` and supply structured `--params` / `--data`; do not invent typed flags or payload fields.

## Identity, recipients, and links

Honor an explicitly requested `--as user` or `--as bot`. Otherwise infer the intended actor from context. User tokens require both granted scopes and the current user's access; bot tokens require bot scopes, membership, and application visibility/availability. A different identity is a different actor, not a permission workaround.

Sending, replying, editing, forwarding, inviting, recalling, and urgent notifications affect other people. Establish the user's approval of the intended action, destination, content, and identity before executing. Use already granted approval; resolve an ambiguous recipient or message first. A bot must be in the target chat where required. Use `--dry-run` to inspect requests when useful.

IDs: messages `om_`, chats `oc_`, threads `om_` or `omt_`, feed groups `ofg_`. Keep resource keys paired with their source message. Flags bookmark messages or feed cards; feed shortcuts pin chats in the sidebar; feed groups organize feed cards.

Prefer returned `message_app_link`, `chat_app_link` (joined conversations), or `share_link` (invitations). If constructing a chat AppLink, use `https://<applink_host>/client/chat/open?openChatId=<chat_id>`; neither `chatId` nor a `lark://` scheme is equivalent.

For chat management, read [chat identity](references/lark-im-chat-identity.md). A bot-created chat without an explicit owner belongs to the bot; an explicit user owner owns it. A user-created chat without an explicit owner belongs to that user. If ownership is unknown, establish it before an owner-only action. Bot manager status does not confer ownership. Membership failures require membership correction; owner/admin failures require an authorized owner/admin. Error `232024` concerns bot visibility; `232043` can reject chat creation with invisible invitees. An authorized user may add members after a chat is created without those invitees. Do not switch actors or retry blindly.

## Choose a shortcut

Commands below begin with `lark-cli im`. All linked references remain authoritative for parameters and response details.

| Shortcut | Use and important limits |
|---|---|
| [`+chat-create`](references/lark-im-chat-create.md) | Create private/public group or topic chat; user/bot, users/bots, optional owner and bot manager. |
| [`+chat-list`](references/lark-im-chat-list.md) | Joined chats, groups by default; user-only p2p via `--types=p2p,group`; sorting/pagination, user-only muted exclusion. |
| [`+chat-members-list`](references/lark-im-chat-members-list.md) | List members into separate `users` and `bots` buckets; filter member types and paginate. Truncation means incomplete membership. |
| [`+chat-messages-list`](references/lark-im-chat-messages-list.md) | History by `--chat-id` or `--user-id` resolving a private chat; time bounds, order, pagination. |
| [`+chat-search`](references/lark-im-chat-search.md) | Discover chats by query/member IDs and type; sorting/pagination, user-only muted exclusion. |
| [`+chat-update`](references/lark-im-chat-update.md) | Update chat name/description as user or bot, subject to chat permissions. |
| [`+message-read-users`](references/lark-im-message-read-status.md) | Readers of one message sent by the calling user/bot within seven days; caller must still be a member. |
| [`+messages-edit`](references/lark-im-messages-edit.md) | Replace a message sent by this bot; bot-only PUT for text/post, with explicit attachment replacement/clear controls. |
| [`+messages-mget`](references/lark-im-messages-mget.md) | Fetch 1–50 message IDs, render content, resolve names, expand threads. JSON output avoids table truncation; rich posts are rendered text. |
| [`+messages-read-status`](references/lark-im-message-read-status.md) | Current user's read status for 1–50 IDs; user-only. Inspect `invalid_message_ids` separately. |
| [`+messages-reply`](references/lark-im-messages-reply.md) | Reply to a message, optionally within a thread; text/post/media and explicit idempotency key. |
| [`+messages-resources-download`](references/lark-im-messages-resources-download.md) | Download matching message/resource IDs as `image` or `file`; audio/video use `file`, stickers unsupported. |
| [`+messages-search`](references/lark-im-messages-search.md) | Search keyword, sender, time, chats, or attachment type; bounded pages with message/chat enrichment. |
| [`+messages-send`](references/lark-im-messages-send.md) | Send to exactly one of `--chat-id` or `--user-id`; choose one content option. |
| [`+threads-messages-list`](references/lark-im-threads-messages-list.md) | Resolve `om_`/`omt_` thread and fetch ordered, paginated messages. |
| [`+flag-create`](references/lark-im-flag-create.md) | User-only bookmark; message by default, feed mode detects thread/topic vs ordinary feed item. |
| [`+flag-cancel`](references/lark-im-flag-cancel.md) | User-only removal; omitted feed type can invoke best-effort double cancellation. Inspect each outcome. |
| [`+flag-list`](references/lark-im-flag-list.md) | User-only flags with feed/thread enrichment; page limit defaults to 20, maximum 1000. |
| [`+feed-shortcut-create`](references/lark-im-feed-shortcut-create.md) | User-only sidebar pinning, chat IDs only, at most 10 per batch; head/tail position. Inspect partial results. |
| [`+feed-shortcut-remove`](references/lark-im-feed-shortcut-remove.md) | User-only, at most 10 chats per batch; already absent is idempotent. Inspect partial results. |
| [`+feed-shortcut-list`](references/lark-im-feed-shortcut-list.md) | User-only, one cursor page. Chat detail enrichment needs its scope; `--no-detail` skips it. |
| [`+feed-group-list`](references/lark-im-feed-group-list.md) | User-only paginated groups, including live and soft-deleted results. |
| [`+feed-group-list-item`](references/lark-im-feed-group-list-item.md) | User-only paginated cards within a group, with chat names. |
| [`+feed-group-query-item`](references/lark-im-feed-group-query-item.md) | User-only lookup of known card IDs within a group, with chat names. |

Do not treat a bounded page set as a complete collection. Preserve `has_more`, continuation tokens, truncation, invalid IDs, and per-item failures. Read-users supports page size 1–100 (default 100), `--page-all`, page limit 1–1000 (default 10), and page delay default 200 ms. Other commands have their own documented bounds.

## Message content and attachments

Read the send/reply/edit reference before composing complex content. Use `--text` for exact plain text, logs, code, or literal Markdown. Use `--markdown` for lightweight headings, lists, links, and summaries; it produces a normalized single `zh_cn` post, not full CommonMark. Use `--content` for exact post titles, locales, native rich content, or cards. Content options are mutually exclusive.

Send/reply `--attachment` takes repeatable bare file keys and merges a post attachment zone; do not combine it with `--content` already declaring files. Editing replaces the body: without attachment flags, existing attachments are preserved. `--set-attachments` supplies the complete replacement list; `--clear-attachments` sends `files:[]`. These are mutually exclusive, require post content, and conflict with `--content` already containing files. Include the full intended new body and retained attachments. Server metadata supplies attachment names.

Media flags accept existing keys, supported URLs, or working-directory-relative files; local absolute paths and escapes are rejected. Video requires a cover. `--audio` accepts Opus (`.opus` or Ogg Opus); convert MP3/WAV or send them with `--file`. Remote Markdown images may be downloaded/uploaded, and failures can omit an image with a warning. Local Markdown image paths are unsupported; pre-upload keys when needed. A standalone file message differs from a post attachment zone.

For text/post mentions, use the supported user mention markup or post `at` node; interactive cards use their own native mention syntax. Treat `all` as a notification to everyone. When forwarding document content, fetch with `--doc-format im-markdown` and pass the original to `--markdown`, retaining `<cite type="user">` markup. An explicit UUID is at most 50 characters and supports the documented one-hour send/reply deduplication window.

## Reading, enrichment, and resources

The four message-reading shortcuts (`+messages-mget`, `+chat-messages-list`, `+messages-search`, `+threads-messages-list`) attach reactions by default; `--no-reactions` skips this. Read [message enrichment](references/lark-im-message-enrichment.md) when reactions, edits, threads, or downloads matter. Prefer that enrichment over a separate reaction batch query for messages already being fetched. Standalone reaction operations and emoji values are in [reactions](references/lark-im-reactions.md).

Server names populate `sender_name`; i18n names and bot identifiers are preserved where supplied. Missing names fall back to IDs; do not add contact/bot-info queries or scopes. System messages may lack a sender. `update_time` appears for edited messages. Missing enrichment is not alone proof of an error: inspect warnings/error markers. Thread expansion is bounded to 50 replies per thread and 500 overall; distinguish `thread_has_more` from `thread_replies_error`.

`--download-resources` is opt-in on mget, chat history, and thread history. It writes into `./lark-im-resources`, excludes stickers, deduplicates matching message/key pairs, and isolates resource failures. Merged-forward resources use the container message ID. Without the flag, resource markers remain references, not downloaded bytes. Single-resource downloads allow paths inside the working directory, `/tmp`, or `~/files`; protected directories remain refused. Check `saved_path` and `size_bytes`.

Post folder attachments expand one level, at most 10 children; retain `has_more` and nested-folder depth hints. Folder objects are not downloadable, and automatic downloads do not include their children. Download an identified child explicitly using its matching resource key.

## Native API choices and restrictions

Inspect schemas for exact scopes and request fields. User sends need `im:message.send_as_user` and `im:message`; bot sends need `im:message:send_as_bot`. Resources use `im:resource`; reaction enrichment needs `im:message.reactions:read`. Read-users recommends `im:message:readonly`. Never assume an OAuth grant supplies bot permission or membership.

| Resource/method | Identity and decision |
|---|---|
| `chats.create`, `get`, `link`, `update` | Native create is bot-only; shortcut create supports both. Get/link require caller membership and same-tenant internal-chat rules. Restricted links require owner/admin. Update supports both. |
| `chat.members.create`, `delete` | Both identities, subject to membership/tenant restrictions. Bot-added users must be in app availability. Restricted additions require owner/admin or qualifying creator bot with `operate_as_owner`; removal requires authorized owner/admin/creator. Removal batches: at most 50 users or 5 bots. |
| `chat.user_setting.batch_query`, `batch_update` | User-only chat member; batch at most 10 chats. Ordinary mute and at-all notification settings are separate. |
| `chat.nickname.get`, `update`, `delete` | User-only self nickname. Empty get means unset; update must be nonempty and at most 300 bytes; delete clears it. |
| `chat.join_requests.list`, `handle` | User owner/admin. List page size 1–100; stop at `has_more=false` even if a token remains. Handle 1–50 requests; reconcile ordered success/failed/already-handled results, not just exit status. |
| `chat.managers.add_managers`, `delete_managers` | Both identities, owner-only. Maximum 10 managers (20 in superlarge groups); adding at most 5 bots; removal at most 50 users or 5 bots. |
| `chat.moderation.get`, `update` | Get requires membership/same tenant. Update requires owner or qualifying creator bot in the chat with `operate_as_owner`. |
| `messages.delete` | Recall as user/bot under sender/chat rules. Recalling another user's group message requires the bot's relevant owner/admin/creator authority; user private-chat recall also has app-availability restrictions. |
| `messages.forward`, `merge_forward`; `threads.forward` | Forward supports user/bot; merged forwarding is bot-only; thread forwarding supports both. Verify the new destination. |
| `messages.read_status`, `read_users` | Read status is user-only, at most 50 IDs with invalid IDs reported separately. Read-users covers the calling sender's own messages within seven days; still a chat member. Read the linked read-status reference; prefer shortcut pagination. |
| `messages.patch` | Update an app-sent interactive card as user/bot, within 14 days, serialized content at most 30 KB. Follow the card workflow below. |
| `messages.urgent_app`, `urgent_phone`, `urgent_sms` | Bot-only sender and chat member; explicit recipient delivery/notification effects. Inspect invalid recipients. |
| `reactions.create`, `list`, `delete`, `batch_query` | User/bot; create/list/delete require chat membership. Delete only this actor's reaction. List/batch query read; create/delete change reactions. Read the reactions reference before native operations. |
| `images.create`, `files.create` | Upload as user/bot; user upload requires `im:resource`. |
| `pins.create`, `delete`, `list` | User/bot, subject to chat permissions. |
| `feed.groups.create`, `update`, `delete`, `batch_query`, `batch_add_item`, `batch_remove_item` | User-only; read [feed groups](references/lark-im-feed-groups.md) before using structured payloads. Three other read methods exist only as the shortcuts above. |

For feed groups, prefer a normal group with explicit membership when chat IDs are known. Use rule groups only for automatic membership, with the smallest precise rule set; prefer positive conditions and `match_all` when appropriate. Known IDs favor `batch_query`/`+feed-group-query-item`; list only to discover unknown IDs. Use `feed_type:"chat"` and actual chat IDs. Rename with `update_fields:[1]`; rules use `[2]`. Consume live and soft-deleted lists; feed-group times are millisecond strings. Inspect `failed_items`. Read/write scopes are `im:feed_group_v1:read`/`:write`; item shortcuts also need `im:chat:read` for names.

For flags in feed mode, thread/topic feeds use item type 4 and ordinary `msg_thread` feeds use 11. Follow the flag references for detection and partial cancellation outcomes.

## Interactive cards and callbacks

Before sending, replying with, or patching an interactive card, **read and follow [the complete card creation workflow](references/card/lark-im-card-create.md)**. Do not handwrite or copy a payload in place of that workflow. It selects Card 2.0 by default, requires the matching schema, component references, and style checks, and defines validation retries and Card 1.0 fallback. Use the JSON produced by the workflow.

For button/form actions, read [card action replies](references/lark-im-card-action-reply.md) and its event prerequisite. Console callback enablement is a manual prerequisite; a running listener does not prove callbacks are configured. Callbacks are bot-only, need `im:message:readonly`, and include original `card_content` when available. Compact interactive event output is unsupported; use raw events and heed stderr hints.

Deduplicate by event ID. Callback tokens last 30 minutes and allow at most two updates; the SDK acknowledgment deadline is three seconds. Distinguish button actions, form `form_value`, and standalone input values. Decide whether a visual update is needed; delayed updates require a complete card, not a partial patch. Do not reconstruct a missing original card by guessing. Card 1.0 delayed updates must include the operator in `card.open_ids`; detect Card 2.0 from its schema. Follow the reference's listener/background authorization procedure before automatic handling.
