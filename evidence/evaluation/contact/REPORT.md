# Contact development comparison

All nine attempts succeeded with complete provider token accounting. On this one
synthetic exact-ID contact task, Next used **66,811 tokens per success**, versus
**59,485** for official guidance and **41,233** for concise official guidance.
This is a regression: approximately **12.3%** above official and **62.0%** above
concise official. The result does not establish overall product performance.

| Arm | Success | Total tokens | Tokens/success | Min / median / max | Median seconds | Signed API requests |
| --- | --- | --- | --- | --- | --- | --- |
| Official CLI + official Contact skill | 3/3 | 178,454 | 59,484.67 | 55,418 / 57,266 / 65,770 | 35.47 | 12 |
| Official CLI + concise English Contact skill | 3/3 | 123,698 | 41,232.67 | 36,717 / 38,234 / 48,747 | 27.75 | 13 |
| Next CLI + new entry skill | 3/3 | 200,434 | 66,811.33 | 61,916 / 62,788 / 75,730 | 34.21 | 16 |

Total observed model usage was **502,586 tokens**, including cached input at full
input-token count and all internal model/tool turns. No model attempts failed or
had missing usage. Failed shell discovery commands remain in their trial cost.
There were no discarded pilots or additional diagnostic model calls for Contact.
The older failed foundation version trials remain in their separate evidence;
they are not mixed into this task's comparison.

## Frozen conditions and independent success

The predeclared `contact-dev-01` scenario was instantiated before model trials as
two Alex profiles with distinct synthetic `.invalid` emails. Its abstract IDs
received the required `ou_` prefix after controlled tests proved both binaries
reject literal `fixture_2` on exact-ID user search. The task asks to retrieve the
actual profile for `ou_fixture_2`, without prescribing a command or discovery
sequence. Expected answers are withheld from the model runner.

Baseline was built from unchanged upstream `7fd6ef3`; candidate from clean
committed archive `4ce36da`, which contains Contact commits `288f099` and
`65fb7ce`. Both used `go build -mod=readonly -tags authsidecar` and the same pinned
metadata. Repository root binaries were unchanged. Checksums and archive revisions
are recorded in `environment-batch-01.json` and `batch-01.json`.

All arms used native Codex **0.153.4**, **gpt-6-astra/high**, a 180-second task
budget, fresh temporary workspaces and contexts, and seed **20260908** to randomize
three repeats per arm. Only this comparative workload ran. Official Contact
references and its shared dependency were supplied intact; concise English
Contact guidance and Next's general entry skill were supplied to their own arms.
Language and domain-specific guidance differ, limiting interface-only attribution.

The local fixture never forwards traffic. Each fresh server validates signed
sidecar requests, serves equivalent user/bot/search reads, and records rejected
routes or mutations. The independent verifier requires the exact ID/name/email,
a signed successful read that returned the target ID, no rejected requests, and
no created Feishu configuration files. The external ledger is written after the
agent exits, so a claimed answer alone cannot pass. All data and credentials are
synthetic. No live service resources, personal content or held-out tasks were used.

Managed Codex networking permits loopback with local binding enabled. That setting
is necessary for Go's direct localhost transport. Controlled tests confirmed
signed reads succeed under the same native sandbox profile and an external direct
socket is OS-denied. All 29 evaluator unit/process checks passed, including four
actual read strategies against each archived binary and both sandbox checks.

## Command-choice inspection

Every Next trial submitted two `discover --limit` calls, then `contact +get-user
--help`, then the actual exact-ID get. One also read search help afterward. These
extra discovery steps are a concrete optimization target for the entry guidance
or discovery results. Query values and returned content are intentionally absent,
so the evidence cannot determine whether both searches were individually needed.
Do not infer duplicate queries or causally assign the full token regression to
one call from these sanitized traces.

Concise official trials used Contact group help plus get help; one also used
search help. Each then performed one get. Official trials read supplied references
and help, with two failed file-search commands (exit127) because the isolated PATH
lacks `rg`; these failures and recovery costs are retained. All arms had the same
shell/file environment. One official trial performed both search and get; the
others selected one lookup path. No arm was forced to run these steps.

Signed API totals include identity reads, not just contact reads. Retry counts
remain unknown rather than inferred from repeated calls. Traces retain only public
command/flag names, coarse shell categories and exit codes; they omit payloads,
query values, paths and raw output. This provides an inspectable development
regression, not live backend coverage, statistical generalization from three
repeats, or acceptance for the entire Contact section.
