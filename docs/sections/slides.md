# Slides P2 section

Status: offline section gate passed; required live guarantees and coordinator comparative development trials remain open. This is not full section acceptance.

## Ownership and reference

Owns `shortcuts/slides/`, `tests/cli_e2e/slides/`, `tests/next_sections/slides/`, matching affordance and this report. Shared runtime/output/auth/receipts and root assembly belong to foundation/coordinator; wiki target resolution and Drive media/permission behavior are cross-section dependencies. No Slides event projections are registered in the frozen source.

Inspected sibling `../feishu-cli-review` at `7fd6ef3`, the frozen `coverage.json`, `registrations.json.gz`, `meta_data.json.gz`, `sources.json`, all Slides implementations and tests, and the SML schema/quick reference. Reuses the complete implementation; 30 command-tree nodes comprise 14 shortcuts (including hidden `+update`), 11 generated endpoint leaves, and 5 routing nodes. No raw endpoint replaces a workflow.

## Complete frozen mapping

Each source command maps to the same path under `feishu-cli-next`. Shared execution/output flags retain their original contracts and gain the foundation result controls. The table includes every domain-specific frozen flag and its type. Canonical schemas remain in the frozen catalog, including nested request/response types.

| Command | Frozen domain flags | Evidence owner |
| --- | --- | --- |
| `slides` | routing/shared only | frozen registration/schema inspection |
| `slides +add-slide` | `--before-slide-id` string, `--no-lint` bool, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--revision-id` int, `--slide` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +create` | `--no-lint` bool, `--slide` stringArray, `--slides` string, `--title` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +delete-slide` | `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--revision-id` int, `--slide-id` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +history-list` | `--page-size` int, `--page-token` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`) | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +history-revert` | `--history-version-id` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`) | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +history-revert-status` | `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--task-id` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +media-download` | `--file-token` string, `--output` string, `--output-dir` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +media-upload` | `--file` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`) | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +replace-pages` | `--continue-on-error` bool, `--no-lint` bool, `--pages` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--validate-only` bool | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +replace-slide` | `--no-lint` bool, `--parts` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--revision-id` int, `--slide-id` string, `--tid` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +screenshot` | `--content` string, `--output` string, `--output-dir` string, `--output-name` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--slide` string, `--slide-id` stringSlice (aliases `--slide-ids`, `--slides`), `--slide-number` intSlice (aliases `--slide-numbers`) | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +update` | `--content` string (aliases `--xml`, `--slide-xml`, `--slide-content`, `--content-xml`), `--no-lint` bool, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--revision-id` int, `--slide-id` string, `--tid` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +update-slide` | `--content` string (aliases `--xml`, `--slide-xml`, `--slide-content`, `--content-xml`), `--no-lint` bool, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--revision-id` int, `--slide-id` string, `--tid` string | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides +xml-get` | `--output` string, `--presentation` string (aliases `--presentation-id`, `--presentation-token`, `--token`, `--presentation_id`, `--xml-presentation-id`, `--url`), `--raw` bool, `--remove-attr-id` bool, `--revision-id` int, `--slide-id` string, `--slide-number` int | inherited domain units + legacy dry-run E2E + next workflow process suite |
| `slides xml_presentation.history` | routing/shared only | frozen registration/schema inspection |
| `slides xml_presentation.history list` | `--page-size` int, `--page-token` string, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.history revert` | `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.history revert_status` | `--task-id` string, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide` | routing/shared only | frozen registration/schema inspection |
| `slides xml_presentation.slide create` | `--idempotency-key` string, `--revision-id` int, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide delete` | `--idempotency-key` string, `--revision-id` int, `--slide-id` string, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide get` | `--revision-id` int, `--slide-id` string, `--slide-number` int, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide replace` | `--revision-id` int, `--slide-id` string, `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide_image` | routing/shared only | frozen registration/schema inspection |
| `slides xml_presentation.slide_image list` | `--xml-presentation-id` string | next generated endpoint process suite |
| `slides xml_presentation.slide_image render` | routing/shared only | next generated endpoint process suite |
| `slides xml_presentations` | routing/shared only | frozen registration/schema inspection |
| `slides xml_presentations create` | routing/shared only | next generated endpoint process suite |
| `slides xml_presentations get` | `--remove-attr-id` bool, `--revision-id` int, `--xml-presentation-id` string | next generated endpoint process suite |

## Content and behavior acceptance

- XML remains an opaque full-content payload after existing targeted root-ID injection, missing shape content insertion, note-ID stripping, and image placeholder expansion. Presentation themes; slide styles/backgrounds/notes; text runs and Unicode/whitespace; shape/line/polyline/image/icon/table/chart/embed/SVG elements, gradients and namespaces remain supported. Units pin token-level rewriting and namespace/CDATA preservation; process tests preserve complete rich page XML and verify native visible IDs. Rendering correctness of all SML types remains live-dependent.
- All presentation reference aliases, native/imported office tokens, Slides URLs, wiki resolution and non-Slides rejection are preserved. Media upload keeps native `slide_file` versus `office_slide_file`, 20 MB limit, placeholder deduplication and progress reporting. Download keeps permission-triggered source-preview fallback and local file collision handling. Screenshot selectors, 10-page cap, raw render/list and PNG/JPEG remain unchanged.
- Existing +update-slide and hidden +update keep page ID/order and replace the complete supplied page; omitted elements are deletion by the existing explicit full-page contract. New opt-in `--skip-unchanged` requires conditional read scope only. A current page read must match both returned slide ID and XML root ID. A nonnegative revision must also be explicitly returned. Exact complete XML comparison ignores only the existing note-ID normalization. Unchanged results include `unchanged:true` and observed revision without a mutation. Whitespace/text/element IDs remain significant. Default write-only calls retain their authorization contract.
- Pinned revision, explicit tid and local-image placeholder operations retain writes even with `--skip-unchanged`; these express historical/transaction/upload effects. Comparison is a point-in-time observation, not a concurrency lock. Element insert/replace, add/delete, history revert and +replace-pages retain their distinct effects; page recreation changes page and element IDs and is never silently treated as an in-place no-op.
- Structural parts retain strict validation, canonical/compatibility aliases, empty replacement semantics, 200-part limit, lint default/opt-out and advisory/failure data. Generic replace retains backend str_replace support even though the workflow shortcut deliberately accepts structural actions only.
- History pagination exposes has_more/page_token unchanged, preserves numeric history IDs as strings and supports explicit continuation. Async revert responses are unknown until status evidence; lost responses and partial part failures retain receipt evidence with no automatic replay. Multi-page replacement creates before deletion, threads returned revisions, reports continue-on-error partial results and permits validate-only without mutation. When default stop-on-error follows observed creation/replacement effects, partial stdout now retains all observed per-page results (old/new IDs, status and summary) while preserving the typed API error. This additive failure output survives the shared mutation wrapper’s removal of retry hints.

## Written process suite

`slides_e2e_test.go` covers full-page artifact/read/edit/no-op/raw/file workflows, create/add/delete/recreate ordering, validation, history continuation/revert/status, structural partial/scope/lost-response failures and no replay, screenshot render/list plus upload/direct-download/permission-preview bytes, bot/user wiki routing/refusal and bounded history pagination. `slides_api_e2e_test.go` covers all 11 catalog endpoints through real processes with exact method/path/query/body and Unicode/empty/null output assertions, including idempotency parameters and generic str_replace.

The fixtures build the production authsidecar CLI through `nextfixture.RunMain`, use separate private config/work directories and synthetic local backends, and never forward a request. Backend state/request assertions are independent from command success output. Existing 481 reference unit test events and full deterministic legacy selectors remain applicable.

`TestGeneratedRenderNativeImageEvidenceLimit` verifies the generated `xml_presentation.slide_image render` command with native PNG/Base64 responses. A small valid image returns exit 0 and `confirmed_applied`; a valid 960×541 uncompressed PNG whose response exceeds the shared 1 MiB receipt evidence cap returns exit 22 and `unknown` / `response_too_large`. In both cases, one POST sends the exact content-only request and stdout preserves the complete image bytes and native metadata; the oversized result has `ok:false` because acknowledgement evidence is bounded, not because image bytes were lost. Receipt inspection makes no backend request, and neither case replays the render.

The generated command retains the frozen write classification. The native contract and inherited `+screenshot` preview behavior do not establish server-side persistence semantics; this test makes no such claim. This is the accepted shared receipt capacity limit, not a render-specific product failure. The frozen native method is `services[name=slides].resources[xml_presentation.slide_image].methods[render]` in `docs/reference/meta_data.json.gz`: its content-only POST returns `slide_image.format` and `slide_image.data` with optional page metadata. Focused process checks passed both cases through `isolated_env`; `.tmp/slides-render-regression.log` records the execution.

## Execution and outstanding verification

Run `python3 scripts/next_test.py --section slides --suite all --report .tmp/slides-section.json` for the complete offline section boundary. The runner isolates HOME/config/credentials/network and selects all next section tests plus legacy DryRun|Regression cases. Machine-readable pass/fail/skip totals and exclusions are recorded in the report.

Final offline execution: **486 unit test events, 56 inherited dry-run events, 30 next process events passed; zero failures**. One next live scenario skipped explicitly. Eight inherited live workflows were excluded by the deterministic selector and remain unverified. The process fixture builds the real authsidecar binary. `make build` passed the canonical metadata-restored normal build; scoped `go vet ./shortcuts/slides/... ./tests/next_sections/slides` via `isolated_env` passed; owned Go formatting and whitespace checks passed.

The live gate command `python3 scripts/next_test.py --section slides --suite live --allow-live --report .tmp/slides-live.json` returned **blocked, exit 2**, without issuing a live request. Complete raw failure/recheck logs remain in `.tmp/slides-*.json`, including the initial rich-output and native failed-part regressions. Final process behavior depends on shared rich-output fix `91d167d` and native mutation classifiers `cfb5afa` / `1bcdb68`.

Root-wide `make unit-test`, `make vet`, `make fmt-check`, quality/source/license gates, module reconciliation and release cross-section tests remain coordinator-owned; this section did not run or claim those shared gates. OS-level process-kill journaling is covered by foundation evidence; Slides process fault injection verifies a lost response after dispatch and no automatic replay, not a new process-kill harness.

`TestLivePrivateSlideLifecycle` is separately selectable and currently explicitly BLOCKED: no harness verifies effective current account/tenant and private creation visibility before content write, with run-owned cleanup. Once available, create a private owned deck; verify SML types/ID/order/render/media/history effects by readback; clean up only run-owned resources. Any access delivery to another person requires unambiguous Ying Wu. Legacy live helpers are not run with personal credentials, and this skip is not a pass.

Required current-backend guarantees (private creation, permission scope behavior, native rendering/SML normalization, imported media preview, revision concurrency and history timing) remain unverified. Comparative development tasks are scheduled by the coordinator; no held-out tasks were used. No performance claim or full section acceptance is made from offline tests.

## Source reuse evidence

| Frozen sibling file | SHA-256 |
| --- | --- |
| `shortcuts/slides/helpers.go` | `d3b2a8035143decd8f6558f606e5d594c80f2a81ccbcef7ce374b695a67f65bb` |
| `shortcuts/slides/presentation_flag.go` | `3d08d6db0bf6f63735685c4e4f3e9af469cc5cf7c9ae3a3a9daef10181924494` |
| `shortcuts/slides/shortcuts.go` | `71c04f483c9ca979413f39b0fdb67a1e81c21df7443e6d0c52e6437acdda582c` |
| `shortcuts/slides/slides_add_slide.go` | `dd05b6be28549124d5a7cc9e9496845e6f61743d3dbf1bc71c0014c10e33ca1a` |
| `shortcuts/slides/slides_create.go` | `a06b416ba89463b6b4b4baa778c295c54a675c458ae72568fd777e0c68044172` |
| `shortcuts/slides/slides_delete_slide.go` | `80844ff9bc58a84e0aa0b147f63f45a2265074cbcdc8d92c1bf24cf97f95da63` |
| `shortcuts/slides/slides_errors.go` | `b66fbe19af1b611e555c60205fb49f5c86c91b604726436db6dcf56af60abdbe` |
| `shortcuts/slides/slides_history.go` | `a4b4c81c6730ed23eb158cac9ad682b97c6a0c5e9b6ad818685223855aa1a367` |
| `shortcuts/slides/slides_lint_error.go` | `6e28f264866db8cca11bece122fe0826c6449f2674be82644a7a81f325051d31` |
| `shortcuts/slides/slides_lint_param.go` | `aecef9b00d460d0af8e2bfe0c734c75cef96ccec8e658d2fb7f1f9ebb9e4095c` |
| `shortcuts/slides/slides_media_download.go` | `2ef641e9c8c62b57b2d4d81723216507002663b7cf52b77d3649e2ec3a2d10f3` |
| `shortcuts/slides/slides_media_upload.go` | `2b4d717d2725bd1069cf6abfb8d42719a0a455e7dd04970f04c28666ef8a9ceb` |
| `shortcuts/slides/slides_replace_pages.go` | `74fb2e1d2941a896f8738326fe67dffa1050a4b199b519890daf5a22192e6f08` |
| `shortcuts/slides/slides_replace_slide.go` | `0be64b4d208a4f099fff599f2a0779d6decb83de74b533256a5408e56d6e4e3c` |
| `shortcuts/slides/slides_screenshot.go` | `ae704ec6748e02eaf31a35d6853eb7dcd6bde7c71db7595d944add712566a9c5` |
| `shortcuts/slides/slides_shared.go` | `bdcfa072e15e4c5bd8a017686ac0a86430747b86ec8fda998a34adb22ad7d8b9` |
| `shortcuts/slides/slides_update_slide.go` | `4aabccc659138e9c9621dfdc747f2a88a76b858dced9f87943ad285b7dd9ab62` |
| `shortcuts/slides/slides_xml_get.go` | `a07ccd84f140fe6c381a38475562fef72e4fd1b8f912e9c89f00dba7eebb2e17` |

## Changed files and commits

- `3109878`: optional full-page no-op and target-bound comparison units.
- `d6e61a9`: require actual revision evidence for no-op results.
- `3795fb5`: preserve old/new page recovery IDs after delete failure.
- `efb7134`: structured partial recovery output preserves observed effects when replacement stops.
- Final section commit: exact dry-run step numbering, full process suite, this mapping and corrected legacy coverage scope. No generated catalog, shared runtime or event files are edited by this owner.
