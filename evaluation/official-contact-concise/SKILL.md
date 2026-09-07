---
name: lark-contact-concise
description: Resolve people and visible bots with Lark CLI contact commands.
---

Use `lark-cli contact --help` and command `--help` for exact flags.

Choose identity explicitly. With user identity, search people by name/email with `contact +search-user --query NAME --as user`, or resolve known open IDs with `contact +search-user --user-ids ID --as user`. With bot identity use `contact +get-user --user-id ID --as bot`. To inspect yourself as user, use `contact +get-user --as user`. `+get-user --user-id-type` accepts open_id, union_id, or user_id; `+search-user` takes open IDs.

A display name is ambiguous: match the exact ID, or ask the user to select before a consequential follow-up. For known IDs needed only by another command, use them directly without contact discovery.

Search supports CSV IDs (maximum100), CSV `--queries`, `--lang`, and `--page-size`1–30. Optional filters: `--has-chatted`, `--exclude-external-users`, `--has-enterprise-email`, `--left-organization`; omit false filters. If `has_more` is true, narrow the query or filters; there is no automatic pagination. Use `contact +search-bot --query NAME --as user` for visible bots; it searches keywords and does not support direct ID lookup. If person versus bot is unclear, check both.

For personal status/signature, inspect `lark-cli schema contact.user_profiles.batch_query`, then call `contact user_profiles batch_query --as user --params '{"user_id_type":"open_id"}' --data '{"user_ids":["ID"],"query_option":{"include_personal_status":true,"include_description":true}}'`.

Permission denied/41050 depends on identity visibility. Cross-tenant profiles may have empty fields; preserve missing values. Contact does not send messages, arrange meetings, or enumerate organization trees; use the relevant domain or generated API when needed.
