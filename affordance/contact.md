# contact
> skill: lark-contact

## +search-user
Search by keyword or email, fetch richer profiles, or resolve several known open_ids with --user-ids under user identity. Each match returns an open_id and p2p_chat_id to chain into follow-ups. For one known user’s basic profile, [[+get-user]] also supports user identity.

### Skills
- lark-contact/references/lark-contact-search-user.md

### Avoid when
- Running as a bot — this shortcut is user-only; use [[+get-user]] instead (it supports bot identity)
- You only need users' personal status for ids you already hold → use [[user_profiles batch_query]]

### Examples

**Find a user by name**
```bash
lark-cli contact +search-user --query "alice" --as user
```

**Fetch known users by open_id (me = yourself)**
```bash
lark-cli contact +search-user --user-ids "ou_3a8b****6a7b,me" --as user
```

## +search-bot
Search bots (apps) by keyword. Pass `--query` or `--queries`; use `--chat-ids` to search within specific chats.

### Skills
- lark-contact/references/lark-contact-search-bot.md

### Avoid when
- Looking for a person rather than a bot → use [[+search-user]]
- Running as a bot — this shortcut is user-only

### Tips
- `has_more=true` means the search is incomplete; refine the keyword or search scope instead of paginating

### Examples

**Find bots by keyword**
```bash
lark-cli contact +search-bot --query "会议助手" --as user
```

**Search inside one chat**
```bash
lark-cli contact +search-bot --query "助手" --chat-ids "oc_3a8b****6a7b" --as user
```

**Find bots you've chatted with**
```bash
lark-cli contact +search-bot --query "助手" --has-chatted --as user
```

**Search several bot keywords in one call**
```bash
lark-cli contact +search-bot --queries "会议助手,日报助手,审批助手" --as user
```

## +get-user
Use one known user’s basic profile under user or bot identity. Omit --user-id to read yourself under user identity.

### Skills
- lark-contact/references/lark-contact-get-user.md

### Avoid when
- You don't have the user's id yet, or want to match by name/keyword → use [[+search-user]]
- Need richer fields such as email or departments, or several known open_ids under user identity → use [[+search-user]] --user-ids

### Tips
- Self lookup (omit --user-id) needs user identity; a bot must pass --user-id
- --user-id-type must match the id you pass (default open_id)

## user_profiles batch_query
Bulk-fetch personal status and signature for user ids you already have.

### Avoid when
- Need more than status/signature (name, dept, email), or don't have the open_id yet → use [[+search-user]]

### Tips
- Off by default — set include_personal_status / include_description to true under query_option
- ids in user_ids must match --user-id-type (default open_id)

### Examples

**Bulk-query status and signature**
```bash
lark-cli contact user_profiles batch_query --data '{"user_ids":["ou_3a8b****6a7b"],"query_option":{"include_personal_status":true,"include_description":true}}'
```
