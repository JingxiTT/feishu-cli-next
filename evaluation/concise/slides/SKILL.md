---
name: lark-slides
version: 1.0.0
description: Read, create, edit and manage native Lark/Feishu Slides, including individual pages, media, screenshots and history. Recognize /slides/ links by path and token, including doubao.com links. Use lark-doc for document content, lark-whiteboard for standalone document boards and lark-drive for ordinary files.
metadata:
  requires:
    bins: ["lark-cli"]
  cliHelp: "lark-cli slides --help"
---

# Native Slides

Use the existing deck and native elements for edits. A request to read or analyze a presentation does not authorize editing it. Keep the returned `xml_presentation_id`, `slide_id`, `revision_id`, media tokens and actual output paths for later operations.

Use explicit `--as user` by default. `lark-cli auth login --domain slides` authorizes the user workflow; shared authentication, scopes and global output options follow `lark-shared`. Use `--as bot` when the user requests application ownership/identity or the workflow explicitly requires it, and verify the bot can access the target. Permission failure is a reason to check the current identity and scopes, not to silently switch to bot. Bot creation may grant the current user access; inspect the result. Do not transfer ownership without explicit authorization.

`--presentation` accepts a presentation token, a `/slides/<token>` URL, or a `/wiki/<token>` URL. Route by URL path, including non-Feishu hosts such as doubao.com, rather than sending a Slides URL to a web fetcher. Shortcuts resolve wiki references and require `obj_type=slides`. For native API calls, resolve the wiki node using `wiki +node-get --node-token <wiki-url> --as <identity>`, then use its `obj_token`; keep the same identity for resolution and Slides operations.

Prefer workflow shortcuts when they cover the operation. Native `slides xml_presentations`, `slides xml_presentation.slide`, `slides xml_presentation.slide_image` and `slides xml_presentation.history` commands remain available for their supported parameters. Consult `schema slides.<resource>.<method>` for unfamiliar native request fields; command help/schema describe current flags. Do not guess XML or JSON field names from another command.

## Read, edit and manage pages

| Need | Command and decisions |
| --- | --- |
| Read the complete deck | `slides +xml-get --presentation <ref> --as user`; XML is in the JSON envelope. `--output <relative.xml>` saves XML and suppresses its content on stdout. |
| Read one page | Add `--slide-id <id>` or `--slide-number <1-based-number>` to `+xml-get`. Selectors are mutually exclusive. Native equivalent: `slides xml_presentation.slide get --params '{"xml_presentation_id":"<id>","slide_id":"<page>"}' --as user`. |
| Inspect raw XML | `+xml-get --raw` writes XML directly; it conflicts with `--output`, `--jq` and non-JSON formats. `--remove-attr-id` is available only for full-deck inspection, and removes the IDs needed for precise editing from the returned XML. |
| Create | `slides +create --title <title> --slide @page.xml --as user`; repeat `--slide` in page order, or use `--slides @pages.json` / `--slides -` for an array of complete page XML strings. Do not combine the two forms. At most 10 pages per create; for more, create the shell then add pages individually. |
| Add or insert | `slides +add-slide --presentation <ref> --slide @page.xml --as user`; add `--before-slide-id <id>` to insert before a known page. One page per call. |
| Delete | Read back and confirm the intended page ID, then `slides +delete-slide --presentation <ref> --slide-id <id> --as user`. Deletion is irreversible. |
| Change a few elements | `slides +replace-slide --presentation <ref> --slide-id <id> --parts @parts.json --as user`. Read the existing page first and change only the named blocks. Page ID and order remain unchanged. |
| Change a whole page, its background, or remove elements | `slides +update-slide --presentation <ref> --slide-id <id> --content @page.xml --as user`. Supply the complete target page: omitted elements are deleted. Page ID/order remain; retained elements need their original IDs to preserve native identity. Elements without IDs become new elements. For multiple pages, update each in place rather than create a replacement deck. |

Block parts use `{"action":"block_replace","block_id":"<id>","replacement":"<XML>"}` or `{"action":"block_insert","insertion":"<XML>","insert_before_block_id":"<optional-id>"}`. The shortcut injects the replacement root ID and missing shape `<content/>`. There is no field-level patch: replacing a coordinate still requires the whole block XML. Omit the insertion position to append. Up to 200 ordered parts may be mixed; `--parts` supports a file or stdin. The shortcut accepts structural operations; native replace exposes its own schema. Inspect `failed_part_index`, `failed_reason` and returned lint issues instead of treating an envelope alone as proof of completion.

The full-page root ID must match the selected page. The CLI preserves visible element IDs and strips carried note IDs as required by the backend. `--revision-id -1` means latest. A concrete revision uses that snapshot as the base and can discard newer edits; it is not a promise that another editor cannot change the deck. An unavailable revision can produce 3350002. Leave `--tid` empty for ordinary single-user operations; it is for an explicitly managed cross-request transaction.

## Images, screenshots and templates

`slides +media-upload --presentation <ref> --file ./image.png --as user` returns a `file_token` for `<img src="...">`. Images are limited to 20 MB; supported inputs include PNG, JPEG, GIF, BMP, TIFF and WebP. Slides image uploads do not support a chunked-upload workflow. Let the shortcut choose the correct native/imported presentation parent type.

Use `<img>`, not `<image>`, and a Drive media token, not an HTTP(S) image URL. A web image must be downloaded locally and uploaded first. Create/add/full-page update can upload deduplicated `<img src="@./path">` placeholders. Relative image paths resolve from the current working directory, not the XML file's directory. Keep inputs inside the working directory; use files or stdin for complex XML/JSON instead of fragile shell escaping. Preserve image aspect ratio and avoid covering existing elements.

`slides +media-download --file-token <token>` writes a local image, optionally with `--output <file>` or `--output-dir <directory>`; the default directory is `.lark-slides/media`. Direct-download permission denial can trigger the source-preview fallback. Use the returned path rather than guessing its filename.

`slides +screenshot --presentation <ref> --slide-number <n>` or `--slide-id <id>` saves rendered pages. Use one selector type and at most 10 pages per call; process larger sets in serial batches. `--output <relative-path>` selects one output; `--output-dir` handles batches. For a local XML preview use `--content @page.xml` without presentation/page selectors. The backend supplies PNG or JPEG; the CLI may correct an extension or avoid a filename collision. Always open the returned `output` / `screenshots[].path`, rather than the requested path or stdout Base64. Reuse one deck/task screenshot directory.

For a supplied PPTX template, import with `drive +import --file <template.pptx> --type slides --as user`, optionally `--name` / `--folder-token`. If not ready, follow the returned continuation or `drive +task_result --scenario import --ticket <ticket>`. Edit the imported native Slides directly. Preserve its layout, fonts, colors, images, charts, tables, layering and differences among page types; put new content into existing containers. Shorten text, adjust font size within the original style or reuse nearby native containers before adding small elements. Do not flatten the template into a background, cover it with a new layout, or rebuild an unrelated deck. Verify overflow and occlusion after editing.

## History and recovery

`slides +history-list --presentation <ref>` lists `entries`, with `--page-size 1..20` and `--page-token` continuation. Distinguish the operation's `history_version_id` from `revision_id`. To select by revision, gather matching entries, continuing until found and at most one additional page for adjacent candidates. Disambiguate with `edit_time` (UTC RFC3339) and the requested time; ask when candidates remain ambiguous. For a time-only restoration, select the nearest entry at or before that time unless the user requested absolute nearest.

`slides +history-revert --presentation <ref> --history-version-id <version>` starts an asynchronous task. Preserve its `task_id`; use `+history-revert-status --presentation <ref> --task-id <task>` for progress. While `running`, honor `poll_after_ms`, defaulting to 10 seconds; normally stop after five minutes and report the task as still running. Retry transient status reads at most three consecutive times, never restart the revert because a status read failed. Stop on `done`, `partial_failed` or `failed`; report `failed_block_tokens` when supplied. Read the deck after completion and do not automatically repeat a failed/partial revert.

After a creation or editing error, retain returned resource IDs and relevant payloads, then read current state before retrying. A presentation or uploaded images may already exist. For 3350001 inspect XML escaping, supported tags/attributes, current block IDs, shape content, image tokens and 960×540 bounds; for 3350002 check the target and revision. Correct only the affected content, preserving unrelated pages and native identities. Do not infer that a failed multi-step command had no effects.

## Native XML and authoring quality

These rules concern new decks and substantial rewrites, not read-only analysis. The matching official `lark-slides` reference bundle contains the authoritative `references/xml/slides_xml_schema_definition.xml`, its quick reference and helper scripts. Use that protocol for XML; CLI schemas own API fields.

Before creating or substantially rewriting, retain `.lark-slides/plan/<deck-or-task-id>/slide_plan.json`. Plan the goal, audience, theme, visual system, typography constraints, verification and ordered pages; each page needs its message, layout, visual focus, text density, speaker intent and asset needs. Small edits to existing blocks are exempt. Asset planning is metadata only, with a concrete `fallback_if_missing`; it must not itself require searching, downloading or uploading. Carry the plan into actual geometry, imagery and text quantities. Generate complete `<slide>` documents directly rather than parse and reserialize a full presentation into a create payload.

Use the native 960×540 canvas. `<slide>` contains only `<style>`, `<data>` and `<note>`; visible elements belong in `<data>`. Text uses `<content><p>…</p></content>` inside a shape. Escape XML text and attributes. Set `fontSize` explicitly, use `color` rather than `fontColor`, and use `lineSpacing="multiple:…"` or `"fixed:…"`. Large/long text needs `wrap="true" autoFit="normal-auto-fit"`. Prefer the default SML namespace and unprefixed tags.

A rectangle is a shape, not a container: text, icons and images are sibling elements positioned above it. Native charts cover bar/column/line/area/pie/doughnut/radar/combo families; use shapes and lines for other diagram forms. Omit `<chartLegend>` to hide the legend. Tables may be composed from rectangles and text or native `<table>`; there is no `shape type="table"`. Native tables need fixed width/height, and explicit column widths/row heights where preservation matters; cells contain fill, content and supported borders, not nested shapes/images/icons. Gradients use `<fill><fillColor color="linear-gradient(135deg, rgba(…) 0%, rgba(…) 100%)"/></fill>`. Consult the chart examples before authoring charts. Search and resolve IconPark names rather than guessing; supply contrasting, nontransparent fills. Do not use emoji anywhere in authored slides.

Prioritize quality. Use varied, substantial imagery without reusing one picture throughout. The cover's main visual must be an image, not a shape/icon construction. Generate images of concrete objects rather than drawing them from shapes and lines; image-generation prompts should end with “Do not include any text or color codes.” Avoid decorative rules beneath headings, meaningless strips and narrow page-edge bands. Every page needs meaningful visual structure, not only a title and bullets.

Unless the user provides a template/design, ordinary content pages use dense, balanced text/image card grids: typically 6–8 text cards containing 200–400 Chinese characters overall, plus 1–3 image cards, with data visualized rather than listed as prose. Special cover, divider, conclusion, quote and big-number pages may be less dense. Use square, borderless translucent cards, consistent restrained colors (at most three card colors per page), 40px page/content margins, at least 5px internal padding and 20–40px card gaps. Left-align body text; right-align table numbers. Prefer Source Han Sans, titles 28–36pt, body 12–14pt and notes 10–12pt; use 36–52pt for a principal metric. Maintain a consistent background, theme-specific primary/secondary/accent palette and recurring motif; avoid rainbow cards, low contrast and identical page layouts. Template preservation takes precedence over these default authoring choices.

Before submitting complete page XML, save it locally and run the matching bundle's `scripts/xml_lint.py --input <xml-file>`; `summary.error_count` must be zero and `release_ready` must not be false. After creation, major rewriting or full-page updates, read back the full XML, verify page count, expected elements and retained identities, and run the same lint. Warnings requiring screenshot review need actual rendered inspection, not an automatic increase in text density. Fix blank pages, broken images, clipping, obscured text, unreadable contrast and unresolved placeholders before delivery; record lesser remaining risks. For small edits, verify the affected page and surrounding structure. Never claim visual review without opening rendered output.

Keep the plan and a small resource/verification manifest; remove obsolete successful transient payloads, but retain failed/partial payloads until recovery is complete. Return the presentation link and a concise verification record distinguishing static checks from visual inspection.

Detailed references in the matching official bundle: `references/planning-layer.md`, `references/visual-planning.md`, `references/asset-planning.md`, `references/xml/xml-schema-quick-ref.md`, `references/xml/slides_chart_demo.xml`, `references/xml/iconpark.md`, `references/workflow/slides-editing.md`, `references/workflow/template-editing.md`, `references/workflow/error-handling.md`, `references/workflow/validation-xml.md`, and the corresponding command pages under `references/cli/`. Locate that installed bundle rather than invent a global path.
