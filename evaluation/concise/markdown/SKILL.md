---
name: lark-markdown
version: 1.2.2
description: "Create, read, patch, overwrite and compare Markdown files stored in Feishu Drive or Wiki. Not for purely local Markdown work or native document conversion; use Drive/Docs for imports and Drive for file management."
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli markdown --help"
---

# Markdown files

Read the [shared authentication, permission and safety guidance](../lark-shared/SKILL.md). Prefer `--as user` for personal Drive files; complete `auth login` before first user access. Use `--as bot` for app-owned automation with suitable target permissions. Keep the selected identity: a bot failure usually concerns app scopes or target ACLs, while a user failure concerns user authorization or ACLs. Do not alternate identities blindly.

These commands keep `.md` source files as ordinary Drive files. Native online `docx` conversion belongs to other domains: the frozen Markdown guide routes local file import to [Drive](../lark-drive/SKILL.md), `drive +import --type docx`; [Docs creation](../lark-doc/references/lark-doc-create.md) also supports a Markdown body via `docs +create --doc-format markdown --content @./draft.md`. Rename, move, delete, search, permissions and comments belong to Drive. Version discovery is `drive +version-history`; use its `version`, not its `tag`.

Prefer the matching shortcut when available. Commands have `--dry-run` to inspect their requests. Content flags accept literal strings, `@file`, or `-` for stdin; `--file` specifically accepts a local `.md` path. Names, including a local input filename, must explicitly end in `.md`. Zero-byte creates/overwrites and a patch that empties the entire file are unsupported.

## Create and overwrite

`markdown +create` requires exactly one of `--content` and `--file`. Inline/file-reference/stdin content requires `--name`; local `--file` defaults to its basename. Omit both destination flags for the caller's Drive root. Use `--folder-token` only for a Drive folder, or `--wiki-token` only for a Wiki node; they are mutually exclusive. Full URLs are accepted and normalized. Resolve a document URL to a Wiki node using `wiki +node-get` when necessary rather than guessing token types. A Wiki-hosted file may lack an independent file URL; use the actual returned access URL when supplied.

`markdown +overwrite --file-token <TOKEN>` uploads a new version using exactly one content source. `--name` wins; otherwise local file input uses its basename, and inline content retains the remote filename, falling back to `<file_token>.md` when metadata lacks a name. It returns file identity, filename, byte size and the new version. Small files use `upload_all`; files larger than 20 MiB use prepare/parts/finish.

Create/overwrite require upload and metadata access (`drive:file:upload`, `drive:drive.metadata:readonly`). Bot creation attempts to grant the current CLI user `full_access`: inspect `permission_grant.status` (`granted`, `skipped`, `failed`). A skipped grant lacks an available current user ID; a failed grant does not mean file creation failed. Follow its message/hint and retain the created file identity. This is not owner transfer; obtain separate authorization for owner transfer.

For missing scope, permission denial, missing target, quota or version-limit errors, stop and follow the hint. Correct the resource type/target, authorization or quota rather than resubmitting unchanged. The frozen guidance permits only bounded backoff for rate limits, server errors or temporary network failures; retain the log/request ID when failures persist.

## Fetch and compare

`markdown +fetch --file-token <TOKEN>` returns source text and byte size. `--output <PATH_OR_DIRECTORY>` saves it locally; a directory uses the remote filename. Existing output requires explicit `--overwrite`. Download access is `drive:file:download`.

`markdown +diff --file-token <TOKEN>` compares remote versions or a local draft without changing either side:

- `--from-version <VERSION> [--to-version <VERSION>]`: omitted target means latest.
- `--file ./draft.md [--from-version <VERSION>]`: omitted base means latest remote; `--to-version` cannot accompany a file.
- `--context-lines` is nonnegative, default 3; JSON and pretty are supported. Each compared payload is limited to 10 MiB.

JSON retains the changed predicate, mode, version strings, source/target labels, added/deleted counts, hunks and full unified diff; local comparisons also identify the local file. Labels are explanatory, not API identifiers. Pretty renders the diff; unchanged content returns `No differences.` A missing final newline matters.

## Patch

`markdown +patch --file-token <TOKEN> --pattern <TEXT> --content <REPLACEMENT>` performs one full-file literal all-match replacement. Both inputs are explicit; pattern must be nonempty and replacement may be empty. Only one pattern/replacement pair is supported per invocation. `--regex` uses Go RE2 and expands `$1` or `${name}` in replacement text; use `$$` for a literal dollar sign. Escape regex metacharacters when matching punctuation literally.

Patch downloads the whole source, replaces locally, then uploads a whole-file overwrite. It needs download, upload and metadata access. It is not an atomic server patch: concurrent edits between download and upload can be overwritten. Zero matches succeeds without a new upload. Output reports `updated`, mode, match count, version and before/after byte sizes, not exact match locations. An empty resulting file is rejected. Multiple replacements need separate calls or an explicitly prepared full overwrite.

## Native document import and export

For native document work, continue with the [Docs skill](../lark-doc/SKILL.md) and its [Markdown format](../lark-doc/references/lark-doc-md.md), [create](../lark-doc/references/lark-doc-create.md) and [fetch](../lark-doc/references/lark-doc-fetch.md) references. XML is the rich-edit default; Markdown is suitable for explicit Markdown import. `docs +create` accepts `--title`, `--content`, `--doc-format xml|markdown`, optional `--reference-map`, and one of `--parent-token` or `--parent-position`. A title alone is valid. Do not mix complete XML and Markdown documents; documented XML extensions inside Markdown are supported.

`docs +fetch --doc <TOKEN_OR_URL>` accepts `--doc-format xml|markdown|im-markdown`; IM Markdown is a read format for messaging, never a create/update format. Detail `simple` is for reading, `with-ids` includes block references, and `full` retains styles and edit metadata. Preserve returned content and reference-map sidecars together. For a whole document omit scope; outline, section, range and keyword provide partial reads, which must not be treated as complete tables/documents. A fragment/excerpt represents a selection. Inspect returned warnings even on success, because content may have been downgraded or partly unsupported; `tips` alone does not mean failure.

Keep export escaping intact. In literal Markdown text escape backslash, backtick, asterisk, underscore, brackets, dollar, tilde and `<`; escape heading/list/quote markers at line starts and pipes inside table cells. Code spans/fences treat their contents literally; math retains LaTeX semantics. XML tags within Markdown create rich nodes, so escape `<` when the tag itself is intended as text. Use file/stdin input or a quoted heredoc to avoid shell expansion; relative `@file` paths must remain within the working directory. `@@` represents literal leading `@`; Docs `--pattern` does not support `@file`.

Images support URL syntax and local `![alt](@./image.png)` (angle-bracket the destination when its path has spaces); attachments use a documented XML source tag. Base64 data URLs are unsupported: decode to a local file first. Non-Markdown blocks such as callouts, grids, rich references, drawings, embedded sheets and synchronized blocks require documented XML extensions; do not claim a plain Markdown representation preserves every native property. Remote downloads must satisfy the shared trusted-URL and address/redirect checks.
