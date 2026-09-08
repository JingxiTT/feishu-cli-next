# Qualification: ambient skill catalogs in conditions 05 and 06

Conditions [05](token-efficiency-05/REPORT.md) and [06](token-efficiency-06/REPORT.md) retain their original attempts, provider usage, verifier outcomes and gates. A later native control found that their harness did not fully suppress the public skill catalog from `CODEX_HOME`. This qualifies attribution of their results to the selected CLI guidance alone; it does not justify subtracting tokens or rescoring outcomes.

## Confirmed mechanism

Both frozen runners built disabled-skill entries from each `SKILL.md` file's **parent directory**. Native Codex CLI `0.153.4` requires the **entry-file path** to suppress its catalog entry. The same `configuration()` function serves both original/full and Next arms. The selected arm's guidance is explicitly appended once to the task prompt; it is not separately registered for automatic skill loading.

The [native control evidence](codex-skill-isolation-native-20260909.json) uses the exact Codex binary hash recorded in condition 06. It mirrored nine public `SKILL.md` files into fresh `HOME` and `CODEX_HOME` directories and inspected native `debug prompt-input` output with all network denied:

| Control | Advertised public skill entries | Other observations |
| --- | ---: | --- |
| Parent-directory disable paths, matching the frozen bug | 8 | Skills instructions present |
| Exact `SKILL.md` disable paths | 0 | Skills instructions absent |

The eight advertised entries were hatch-pet, imagegen, kimi-webbridge, openai-docs, playwright, plugin-creator, skill-creator and skill-installer. The ninth mirrored entry, review-agent, was not advertised. Real-home `.agents` skill paths and an arbitrary workspace guidance skill were absent in both controls. This is a catalog-visibility finding, not evidence that eight complete skill bodies were injected or read.

No model call was made. The probe copied only public skill entry files, read/copied no configuration or credentials, retained no raw prompts and removed its temporary directories. It used the shared skill settings while omitting managed proxy listeners so network access could be denied entirely; it was not a replay of a paid `codex exec` attempt.

## What remains unknown

The discarded historical prompts and file-read paths cannot be recovered from the retained traces. The control establishes the native behavior of the frozen configuration and the present mirrored catalog; it does not establish the exact contents of every historical prompt. Both arms used the affected configuration, but that does not prove an equal token or behavioral effect.

There is no measured billed-token amount attributable to these catalog entries, and no proof that they caused extra file reads, help loops, success changes or either missed Docs target. File-read trace events do not identify the files. Real-home `.agents` leakage was not observed in the native controls; the finding concerns the separate `CODEX_HOME` catalog.

A fresh comparison after validating the isolation correction is needed to measure its effect. Such a condition must retain its own baselines and costs; it must not pool or relabel conditions 05/06 as isolated confirmation. Their published reports, environments, checkpoints and stage decisions remain unchanged.

## Sources

- [Native evidence](codex-skill-isolation-native-20260909.json), SHA-256 `9bec5e1877e429eb2d8f3180f2009ec9ca84fea0a51c5602749728510c5a1dc8`.
- [Frozen condition-05 runner](https://github.com/Zhujingxi/feishu-cli-next/blob/75e57af771a58a9929e03f6221f3e2c1480ca96b/evaluation/codex_runner.py#L98) and [environment](token-efficiency-05/environment.json).
- [Frozen condition-06 runner](https://github.com/Zhujingxi/feishu-cli-next/blob/dd0fc7a7e62b29790c567959a6daf9fa5ec55c20/evaluation/codex_runner.py#L192) and [environment](token-efficiency-06/environment.json).
