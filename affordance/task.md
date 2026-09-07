# task

## +update
Use task GUIDs or task applinks for one or several replacements. The task display
number shown in the client is not the OpenAPI GUID.

### Tips
- Omitted fields remain unchanged. An explicit empty description clears it; an empty summary is invalid. `--data` retains supported nested fields and explicit null or empty values for the backend to interpret.
- Exact matching summary, description, due/start, completion, extra, recurrence, mode, milestone and custom completion fields skip the write. Members, custom fields and other workflow effects are sent explicitly.
- Each task is read before replacement using the existing `task:task:write` scope. This comparison does not prevent concurrent edits. [[tasks patch]] retains the direct API path for write-only scopes.
- Multiple tasks are sequential writes. A failure can follow earlier effects; inspect mutation receipts before deciding how to recover.

## +tasklist-create
Use when creating a list together with initial tasks. Every supplied task is
validated before the list is created; child creation can still partially fail.

### Tips
- Inspect both `created_tasks` and `failed_tasks`. Do not repeat the complete creation after a partial or unknown outcome.
- List editors and task assignees gain access or may receive notifications. Supply resolved identities within the user's authorization.

## +assign
Use resolved member IDs for explicit assignment changes. App IDs select app
members; other IDs use the declared user ID format.

### Tips
- Add runs before remove. A failed second operation does not undo the first, and assignment can notify members.

## +followers
Use for subscription changes; adding followers can notify them or expose task
content. Add runs before remove and is not rolled back after a later failure.

## +tasklist-members
Use without a membership action to read members. Setting membership computes an
ID difference; an unchanged set performs no writes.

### Tips
- An omitted or empty set is a read, not a clear. Use explicit removals to remove members. Existing roles are preserved for IDs already in the set.
- Add and remove may grant access or notify collaborators. A failure after an addition can leave partial effects.

## +reminder
Use to replace the reminder configuration or remove existing reminders. An equal
single reminder is left unchanged. Replacement removes the old reminder before
adding the new one; an interrupted replacement may leave no reminder.

## +reopen
Use for a completed task. The current completion timestamp is read first, and
an already open task with `completed_at:"0"` is left unchanged.

## +set-ancestor
Use for parent relationships. Omitting the ancestor deliberately detaches the
task; it is not an omitted-field update.

## +upload-attachment
Use for a single local attachment. Upload is an append effect; a matching filename
does not prove the existing attachment has the same bytes or suppress a write.
