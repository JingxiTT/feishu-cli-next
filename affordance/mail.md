# mail
> skill: lark-mail

## +thread-modify
Use known thread IDs when the intended label change or folder move covers whole conversations.

### Avoid when
- Changing specific messages → use [[+message-modify]].
- Deleting conversations → use [[+thread-trash]].

### Tips
- Submit known IDs in one call; the shortcut handles batching. Bot calls need an explicit mailbox address; user calls may use their own mailbox.
- Inspect both `success_thread_ids` and `failed_thread_ids`. A partial response or unknown mutation receipt needs reconciliation against the affected threads before retrying; do not replay successful batches.

### Skills
- lark-mail/references/lark-mail-thread-modify.md

## +thread-trash
Use known thread IDs only when deletion should cover entire conversations.

### Avoid when
- Deleting specific messages → use [[+message-trash]].

### Tips
- Preview the authorized deletion scope before executing with `--yes`. Submit known IDs in one call; the shortcut handles batching. Bot calls need an explicit mailbox address.
- Inspect both `success_thread_ids` and `failed_thread_ids`. A partial response or unknown mutation receipt needs reconciliation against the affected threads before retrying; do not replay successful batches.

### Skills
- lark-mail/references/lark-mail-thread-trash.md
