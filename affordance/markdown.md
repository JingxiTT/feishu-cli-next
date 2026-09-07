# markdown

## +create
Use for a Markdown source file that should remain a `.md` file in Drive or Wiki.

### Avoid when
- Creating a native rich document: use [[docs +create]]. Markdown files do not convert source text into native document blocks.

## +fetch
### Tips
- JSON content preserves UTF-8 text, escaping and line endings. Use a local output file to retain the downloaded bytes without putting the content in context.

## +diff
### Tips
- Version-to-version and remote-to-local comparisons are read-only. A missing final newline is a meaningful difference.

## +patch
### Tips
- Replacements affect every match. RE2 replacement text expands capture references such as `${1}`; literal mode preserves dollar signs.
- Identical resulting bytes return `updated:false`, including when matches exist, without uploading a new version.
- Fetch and overwrite are separate requests. Concurrent edits by another client can be overwritten; there is no conditional version check.

## +overwrite
### Tips
- This explicitly uploads a new version, including identical content. The existing upload and metadata scopes suffice; it does not add a download prerequisite to compare content.
- The local file name is used with file input unless a name is supplied. Inline content preserves the existing remote name unless a name is supplied.
- There is no conditional version check against concurrent edits.
