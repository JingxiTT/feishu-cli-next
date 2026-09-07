# Corrected Docs development trace review

All nine trials verified the requested outcome. Next used 206,796.33 provider tokens per success, compared with 161,107.00 for official guidance and 76,104.33 for concise official guidance. These are three fresh trials per arm; input includes cached tokens in full.

| Arm | Trial | Total tokens | Shell calls | Help choices | Discover choices | Skill reads | Raw API choices excluding help | Nonzero shell exits | API requests |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| official | 0 | 90024 | 6 | 0 | 0 | 0 | 0 | 0 | 6 |
| official | 1 | 230112 | 13 | 5 | 0 | 0 | 2 | 2 | 14 |
| official | 2 | 163185 | 11 | 3 | 0 | 0 | 2 | 1 | 12 |
| official_concise | 0 | 76086 | 5 | 2 | 0 | 0 | 0 | 0 | 8 |
| official_concise | 1 | 76117 | 5 | 2 | 0 | 0 | 0 | 0 | 8 |
| official_concise | 2 | 76110 | 5 | 2 | 0 | 0 | 0 | 0 | 8 |
| next | 0 | 171516 | 12 | 5 | 2 | 1 | 3 | 1 | 8 |
| next | 1 | 161458 | 12 | 5 | 2 | 1 | 3 | 1 | 8 |
| next | 2 | 287415 | 18 | 5 | 2 | 2 | 4 | 2 | 8 |

Concise official guidance produced the same five-call workflow in each trial: two targeted help calls, fetch, update, then fetch. Official guidance varied: one trial used the update shortcut directly, while two used raw API composition after additional discovery. Each official trial also made three local file-read calls.

Next used five help choices, two discover choices, and at least one skill read in every trial before completing through raw API composition. Each trial began with an exit-2 help attempt. The highest-token Next trial also had an exit-2 API attempt, additional file/skill reads, and local script processing. Its API request count nevertheless matched the other Next trials and all concise-official trials. The traces therefore associate the regression with additional agent discovery and processing rather than a higher total API request count against the concise baseline.

This is descriptive evidence, not per-command token attribution. Sanitization retains command families and flag names, but removes arguments, resource content and detailed API request traces. It cannot establish the exact failed syntax, the motivation for each discovery step, or the number of tokens caused by a particular command. Shell calls may contain multiple choices. Retry counts are unavailable; nonzero exits must not be relabeled as mutation retries.

Raw API composition is a valid strategy and all final results passed the same independent outcome check. No product or guidance change, trial exclusion, or historical rescoring is justified by this small comparison alone. The observed regression remains part of the release assessment.

Source: [completed Docs batch](docs/batch-01.json). Its SHA-256 is `d02de263853f331585a9d57e5cada9eae91ef3c006fc66e71d077f42d6ec438a`. Coordinator independently recounted all nine trials, complete input/output usage, group distributions and matched-success pairs. This review contains no live or held-out evidence.
