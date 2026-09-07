---
name: lark-whiteboard-concise
description: Read, export, create content for, and edit whiteboards embedded in Lark documents. Use for previews, native nodes, diagram source, SVG, and whiteboard updates; use Docs for document editing or creating embedded boards, and Sheets/Base for those resources.
---

Use `lark-cli whiteboard`. Prefer `--as user`; use `--as bot` when acting as the application with access to its resources. Read and edit permissions are distinct. An empty application result does not establish that the user's resource is empty. CLI JSON success uses `ok` and the process exit code, not a top-level API `code`; inspect errors on stderr. Never print secrets. Use relative workspace paths or stdin, and resolve authorization/scope failures through the existing auth workflow. Preserve authorization URLs verbatim and accompany them with an `auth qrcode` PNG when presenting them. For exit 10 confirmation gates, obtain explicit approval and append the returned confirmation flag to the original argv; do not silently bypass the gate.

Choose tools from the user's requested outcome. `lark-cli` performs remote operations; `npx -y @larksuite/whiteboard-cli@^0.2.13` generates, renders and converts local diagrams. Its version and icon-list commands are `-v` and `--icons`. Help and command references are available when syntax needs clarification.

## Addressing and reading

Use a supplied whiteboard token directly. For a document, `lark-cli docs +fetch --doc <URL-or-ID> --as user` exposes `<whiteboard token="…"/>`. Creating a board belongs to Docs: append `<whiteboard type="blank"></whiteboard>` using `docs +update --command append`, then take `block_token` from the returned `new_blocks` entry whose `block_type` is `whiteboard`.

`whiteboard +export --whiteboard-token <token> --output-type <type>` supports:

- `preview`: visual image; `--output` is required. The response MIME type determines PNG/JPEG extension; an extensionless output path avoids a mismatch.
- `svg`: vector snapshot, to stdout or `--output` file. It preserves appearance, not the original board's complete semantics.
- `source`: Mermaid/PlantUML source only when exactly one recognized diagram exists; missing or multiple sources are reported rather than guessed.
- `raw`: native OpenAPI nodes for structural inspection or small text/color changes. For editing and reimport, use `--output ./nodes.json`: the file is importable, whereas stdout adds an `{ok, identity, data}` envelope that `+update` cannot parse.

`--overwrite` on export permits replacing an existing local file. The hidden legacy `+query --output_as image|svg|code|raw` maps to these exports. Preserve native IDs and fields when inspecting raw nodes; do not infer editable structure from a rendered image.

## Updating and choosing an editing method

`whiteboard +update --whiteboard-token <token> --source @./diagram.mmd --input_format mermaid --as user` reads an explicit file. `--source -` reads stdin. Formats are `raw` (default), `mermaid`, `plantuml`, and `svg`; select one. The Docs compatibility command is `docs +whiteboard-update`.

Without `--overwrite`, new content is appended and existing nodes remain. With it, all existing content is deleted and recreated. Ensure the user's authorization covers that replacement and its potential losses; preview destructive requests with `--dry-run`. Do not describe overwrite, even from exported raw nodes or matching source, as a guaranteed identity-preserving edit. Supply one `--idempotent-token` of at least ten characters per logical update and retain it for any authorized retry; generating a new key on each retry can duplicate writes.

Respect source or format explicitly supplied by the user. Otherwise:

- A single exported Mermaid/PlantUML source can be edited and reimported in that language, with informed whole-board replacement authorization.
- Small existing text/color changes can use an exported raw file. Change only the intended fields; raw overwrite still rebuilds the board and needs that authorization. Do not author complex native OpenAPI JSON by hand.
- To add elements while retaining the original board, inspect its preview and raw coordinates, create the additions, and omit `--overwrite`.
- Structural/geometric edits can use SVG export/edit/reimport only with informed acceptance of semantic loss. SVG may lose hierarchy, node types and IDs, mind-map/table structure, connector bindings, containers, mentions, locks and comments. Retain the original SVG grouping, styling and definitions where possible; after moving a shape, adjust connected visual paths manually.

For SVG edits, establish informed authorization before editing and confirm the destructive overwrite request before applying it. Respect authorization already given. Try an editing path at most twice, then one alternative; if it still fails, report the concrete blocker instead of repeatedly switching approaches.

## Creating and rendering diagrams

Use Mermaid for mind maps, sequence/class diagrams, pie charts and Gantt charts, or when explicitly requested. It also supports state, ER and Git diagrams. Use pure Mermaid text rather than mixing it with DSL JSON. Quote labels with special characters. Simple textual flowcharts may use Mermaid; keep labels short and split overly long flows. Complex branching, nested cards and precise layouts need DSL or SVG.

Use DSL whenever content contains user mentions or images. For other diagrams, the official route favors SVG for GPT, Claude, Gemini, GLM, Doubao and Seed, and DSL for other model families; Doubao/Seed use DSL for fishbone, pyramid and flowchart work. Architectural, organization, swimlane, comparison, fishbone, bar/line, treemap, funnel, pyramid, flywheel and milestone diagrams need layouts suited to their relationships. Preserve the user's language and communicate meaningful visual hierarchy rather than filling a uniform grid with text.

Keep artifacts in the user's chosen directory, otherwise `./diagrams/YYYY-MM-DDTHHMMSS/`: `diagram.svg`, `diagram.mmd` or `diagram.json`, plus `diagram.png`. For computed geometry, use a CommonJS `diagram.gen.cjs` generator and run it with Node. Render before writing:

```sh
npx -y @larksuite/whiteboard-cli@^0.2.13 -i ./diagram.json -o ./diagram.png
npx -y @larksuite/whiteboard-cli@^0.2.13 -i ./diagram.svg -o ./diagram.png -f svg
npx -y @larksuite/whiteboard-cli@^0.2.13 -i ./diagram.svg -f svg --check
```

Inspect the PNG and check reports for missing content, overflow, overlap, contrast and connectors. Allow at most two correction rounds. If SVG rendering crashes, serious visual disorder persists, or text overflow remains after two rewrites, discard that SVG and recreate with DSL. Persistently broken DSL can fall back to Mermaid. A preview PNG validates the result; it is not the editable authoring source.

Write Mermaid, PlantUML or SVG directly with the matching `input_format`. Compile DSL to native nodes first:

```sh
npx -y @larksuite/whiteboard-cli@^0.2.13 -i ./diagram.json --to openapi --format json -o ./nodes.json
lark-cli whiteboard +update --whiteboard-token <token> --source @./nodes.json --input_format raw --idempotent-token <stable-logical-update-key> --as user
```

Add overwrite only when the intended replacement is authorized. Report the board token and actual outcome after writing.

## Format constraints that affect fidelity

DSL uses `{"version":2,"nodes":[...]}`. Frames contain children; other types include rect, ellipse, cylinder, diamond, triangle, trapezoid, text, stickyNote, image, icon, SVG and connector. Text may be a string or styled runs with content, emphasis, color, hyperlinks, lists and quotes. Use normal JSON escapes, including `\n`, without double escaping.

Choose horizontal/vertical Flex for structured groups, Dagre for linked topology, and `layout:"none"` for absolute positioning. Specify frame layout, gap and padding. Text-bearing nodes need `height:"fit-content"`; absolute frames need fixed dimensions. `fill-container` requires a suitable Flex ancestor; Flex ignores child x/y. Set `alignItems:"stretch"` for equal-height cards. Cylinders require fixed widths. Put connectors at the document's top-level nodes, with endpoints and anchors inside `connector`; never nest them in frame children. Give referenced frames visible styling so compilation does not remove them. Nested Dagre frames are opaque unless declared `isCluster:true`; put cluster edges on the root Dagre layout. Use consistent group colors, legible text and restrained gray connectors.

Direct SVG import recognizes geometry, paths, text, grouping and common transforms. Use real `<text>`/`<tspan>` for labels, allow approximately 1em per CJK character and 0.6em per Latin character, and prefer translate/rotate/scale over skew or arbitrary matrices. Patterns, clipping, masks and non-shadow filters can render incorrectly. Supported drop/inner shadows and simple objectBoundingBox gradients have narrower rules than general browser SVG; unsupported features may become uneditable images. For editing exported SVG, avoid filters and radial gradients too; linear gradients can degrade to a solid first-stop color. Preserve referenced defs/IDs. Local SVG previews may omit existing board images because of session access.

A DSL `type:"svg"` node is a separate image-blob representation: supply xmlns and viewBox, use it for geometry, and put text/images in sibling DSL nodes. Do not embed text, external URLs or foreignObject in that blob.

Resolve a mentioned person to a real `open_id` through Contact; never guess from a name. A mention is a text run with nonempty `content:"*"` and `mentionUserId`; hyperlinks and mentions require separate runs. Document mentions are unsupported.

Image nodes require media uploaded to the target whiteboard, not a URL or another resource's token. Download existing media when necessary, then use `docs +media-upload --file ./photo.jpg --parent-type whiteboard --parent-node <token>` and put the resulting media token in `image.src`. Verify real image bytes and subject matter, avoid random placeholder services, and respect image licensing. For a requested photo showcase, use distinct images with titles and consistent dimensions, normally a 3:2 ratio. Invalid, cross-board or expired tokens need reupload to the correct target. Images are added when requested, not inferred merely from a diagram topic.
