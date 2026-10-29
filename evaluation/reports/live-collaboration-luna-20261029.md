# Live collaboration: official CLI vs Next

> Current scope update: IM/group-chat evaluation cases, fixture code and dispatcher
> routes were removed at the user’s request after these measurements. IM status
> below describes the recorded run; sealed evidence and results are unchanged.

**Codex GPT-6 Luna (`codex/gpt-6-luna`), medium reasoning, through Magpie**, with frozen production binaries, file-delivered guidance, fresh private `[TEST]` resources and 480 seconds per attempt. Three repeats per CLI for five workflows: **30 provider-accounted attempts**. All actual effects are checked through real Feishu native readback; there is no API simulator.

Observed successes: official **14/15**, Next **15/15**. Known provider tokens: official **4,498,862**, Next **3,190,787**. All-attempt tokens per successful recorded run: official **321,347**, Next **212,719**. Observed Next reduction: **33.8%**.

| Workflow | Official success | Next success | Official tokens/success | Next tokens/success | Next reduction |
|---|---:|---:|---:|---:|---:|
| Create rich document | 3/3 | 3/3 | 173,396 | 144,336 | 16.8% |
| Create SVG drawing in whiteboard | 2/3 | 3/3 | 749,250 | 285,949 | 61.8% |
| Upload Unicode CSV and binary file | 3/3 | 3/3 | 245,728 | 221,292 | 9.9% |
| Upload two same-named images | 3/3 | 3/3 | 226,555 | 186,033 | 17.9% |
| Create Mermaid whiteboard | 3/3 | 3/3 | 354,443 | 225,985 | 36.2% |

Observed failures: official / collaboration-drawing-create: One whiteboard contained two identical copies of the drawing: 16 native nodes instead of the required 8. Four shapes, six text nodes, four connectors and two vector diamond fallbacks were present.

Document creation checks the native heading, bold link, Unicode, leading zeros and table. Image uploads use two different PNGs with identical basenames, checking exact downloaded bytes and display sizes. File uploads check CSV/binary bytes, Unicode filenames and native view/file relationships. Mermaid checks stored source and participants/connectors/branch structure. The SVG drawing checks editable shapes/text and connector geometry, including its vector diamond fallback. Every append also checks all previous native block IDs/content/styles.

**IM coverage is added but live execution is blocked.** `collaboration-im-read` covers selected messages and `collaboration-group-read` covers complete bounded history. Both stop before any tenant or model call because no explicitly designated private TEST chat fixture exists. Group creation is outside the user-granted permissions. The 12 potential IM trials are unbilled and excluded from success denominators; no existing chats were enumerated, read or messaged. [Fixture format](../LIVE_COLLABORATION.md).

Input tokens include cached input once; total is input plus output. All failed attempts and their costs remain included. These are token-volume measurements, not monetary savings. Setup, calibration, independent verification and cleanup are controller work and excluded from agent usage. Eight calibration records, including three initial fixture-assumption failures and fresh corrected runs, are retained separately.

Cleanup: **77 registered document/folder IDs** confirmed deleted, including the test parent; no pending mutation or in-flight markers remain. Temporary model authentication was removed. Native document deletion covers the containing attachments/boards; no unrelated files were listed or modified.

Validation: `make evaluation-test`, `make script-test`, layout checks and staged diff checks passed. One platform-specific native Codex isolation check is skipped on this host. No Go, command/flag/schema, shipped skill or dependency changes were made.

Three repeats are a small development sample. The new workload mix is kept separate from earlier business comparisons. Uploads are document attachments; this report does not claim Drive placement or IM resource-delivery coverage. Drawing checks are structural and semantic, not a subjective visual-design score. API-call/retry totals are unavailable from the production adapter.

Evidence: [assessment](../../evidence/evaluation/live-collaboration-luna-20261029/assessment.json), [all raw attempts](../../evidence/evaluation/live-collaboration-luna-20261029/attempts.raw.json), [blocked IM](../../evidence/evaluation/live-collaboration-luna-20261029/blocked-im.json), [calibration](../../evidence/evaluation/live-collaboration-luna-20261029/calibration.json), [failures](../../evidence/evaluation/live-collaboration-luna-20261029/failure-analysis.json), and [frozen evaluator](../../evidence/evaluation/live-collaboration-luna-20261029/controller.tar.gz).
