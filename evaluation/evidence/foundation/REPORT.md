# Foundation development evaluation

The compatible Codex batch passed **18/18 independent outcome checks**: two
offline foundation tasks, three arms, and three fresh trials per task/arm.
These results cover unconfigured authentication inspection and document-command
discovery only. They do not establish full-domain or release performance.

## Measured comparison

All arms used `gpt-6-astra`, reasoning `high`, the app-bundled native
`codex-cli 0.153.4`, equivalent shell/file tools, a 180-second task budget,
fresh context, and shuffled order with seed `20260907`. The next binary reported
`d873c0d` with reference `larksuite/cli@7fd6ef3`; the retained official binary
reported `0c1ac75`. All binary, complete selected guidance, task, and adapter
hashes are recorded in the accompanying manifests.

| Arm | Verified successes | Total tokens | Mean tokens per success | Median tokens per attempt | Median latency |
| --- | ---: | ---: | ---: | ---: | ---: |
| Official CLI + matching official shared skill | 6/6 | 478,520 | 79,753.33 | 74,356.5 | 37.66 s |
| Official CLI + concise English guidance | 6/6 | 226,141 | 37,690.17 | 37,681 | 24.03 s |
| Next CLI + concise English skill | 6/6 | 182,674 | 30,445.67 | 30,442.5 | 21.55 s |

| Task and arm | Successes | Mean tokens | Token range | Sample standard deviation |
| --- | ---: | ---: | ---: | ---: |
| Unconfigured status — official | 3/3 | 85,549 | 57,875–99,422 | 23,966.41 |
| Unconfigured status — concise official | 3/3 | 23,844.33 | 23,837–23,858 | 11.85 |
| Unconfigured status — next | 3/3 | 24,019.33 | 24,012–24,031 | 10.21 |
| Document discovery — official | 3/3 | 73,957.67 | 73,160–75,073 | 995.28 |
| Document discovery — concise official | 3/3 | 51,536 | 51,504–51,576 | 36.66 |
| Document discovery — next | 3/3 | 36,872 | 36,854–36,891 | 18.52 |

The next arm used fewer tokens on document discovery than either baseline.
Unconfigured-status inspection used **175 more mean tokens** than the concise
official arm, and its median latency was 20.46 seconds versus 18.23 seconds.
Retain this small regression; these two tasks do not support a universal
improvement claim. The official status trials also show considerable variability.
Three trials per task are an initial sample, not proof of statistical significance.

All comparable trials succeeded, so the six matched-success pairs for each
baseline cover the complete compatible batch. Pair-level counts are included in
`batch-02.json`; no failed compatible trial was dropped from the denominator.

## Accounting and correctness

| Arm | Input tokens, including cache | Cached input subset | Output tokens |
| --- | ---: | ---: | ---: |
| Official | 476,588 | 351,872 | 1,932 |
| Concise official | 225,255 | 143,104 | 886 |
| Next | 181,956 | 94,336 | 718 |

Counts come from Codex's provider-reported `turn.completed.usage`. Each span
aggregates a Codex user turn and its internal model/tool loop. Cached input is
included in total input, never discounted to zero. CLI bytes are not used as
model-token measurements. Codex does not disclose the tokenizer version here,
so the manifest says `provider-undisclosed`.

The independent verifier reran the actual selected CLI, checked factual answers
against criteria defined before evaluation, and checked that no Feishu profile
or credential file was created. Four additional controlled checks confirmed
the verifier against both real binaries and both tasks; these are explicitly
separate from model evaluation. The model had an empty Feishu configuration,
metadata/update fetching disabled, and shell network access disabled. No live
Feishu operation was part of these tasks.

API-call and retry counters are unavailable and remain `null`. Raw model/tool
traces were intentionally not retained. This protects private content but limits
analysis of repeated discovery, reference reading, recovery, and autonomous
choices; do not infer a particular command sequence from the token counts.
Only fixed result categories, numeric measurements, sanitized settings, and
hashes are published. This is trusted offline evaluation, not an adversarial
sandbox or proof of all possible host-policy isolation.

## Failed prerequisite attempts remain visible

`batch-01.json` preserves the first 18 balanced attempts using the installed
`codex-cli 0.146.0`. All failed without reported token usage or a verified answer.
Two separately labeled diagnostic attempts also had no reported usage. The
second diagnostic observed HTTP 400 `invalid_request_error`:

> The 'gpt-6-astra' model requires a newer version of Codex.

The model was kept unchanged. A newer executable already bundled with the
desktop app, version `0.153.4`, was used for the complete second balanced batch.
No global CLI installation or user configuration was changed. The task manifest,
CLI binaries, and selected guidance hashes matched across both conditions.

There are **887,335 known model tokens** in the compatible batch and **20 earlier
attempts with unknown usage**. Unknown means unreported, not free. Overall
tokens per success across both conditions and diagnostics is therefore `null`.
The measured table above is explicitly conditional on the compatible executable;
it must not be presented as a finite all-attempt cost across the whole exercise.

## Reproduction and limits

Use `config-batch-02.template.json`, replacing placeholder paths with the pinned
native executable, Python, repository, and existing Codex auth home. The adapter
lets Codex consume its login internally; no credential contents belong in the
configuration or evidence. Then run from the repository:

```sh
python3 evaluation/harness.py --tasks evaluation/tasks.foundation.json \
  --config /ABSOLUTE/local-config.json --repeats 3 --seed 20260907 \
  --output /ABSOLUTE/new-results.json
```

The official shared skill is Chinese and mandates more reference reading.
Arm 1 versus 3 is a complete product comparison, not a language-specific claim.
The concise official arm narrows the guidance difference, but differing domain
detail and the absence of retained traces prevent clean interface-only causal
attribution. A provider model alias can also change even when its requested
name is pinned. No held-out task was executed, and broader setup, authenticated
operations, live backend behavior, and release evaluation remain outstanding.
