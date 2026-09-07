# Build-local authorization recovery

The next distribution renders contextual authorization commands as
`feishu-cli-next auth login`; legacy builds and factories without a configured
command name retain their existing `lark-cli` bytes. The build selects this
fixed name, independently of argv or user content.

The invocation's explicit profile remains one shell-quoted `--profile` value in
both authorization-start and device-code-resume commands. Local no-wait and
timeout messages consume the same immutable context as typed error recovery.

Only contextual, code-owned authorization fragments are rendered this way.
Surrounding messages, unannotated user text, API payloads, and raw-passthrough
errors are unchanged. Presentation clones typed errors and preserves the source
error, causes, scope metadata, and concealed-command behavior.

Coverage includes legacy byte golden tests, shell/Markdown quoting, independent
next and legacy command trees, no-wait JSON and timeout messages, untouched
unannotated text, and raw-passthrough identity.
