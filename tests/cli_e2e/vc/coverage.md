# VC E2E Coverage

## Agent Calendar Lifecycle

| Status | Cmd | Coverage | Notes |
| --- | --- | --- | --- |
| Dry-run | `vc +meeting-join --action start` | `vc_agent_meeting_actions_dryrun_test.go::TestVCAgentMeetingActionsDryRun/start and join` | Verifies the bot join endpoint and `action=2`. |
| Dry-run | `vc +meeting-invite` | `vc_agent_meeting_actions_dryrun_test.go::TestVCAgentMeetingActionsDryRun/invite all suggested` | Verifies the Agent invite endpoint and long `meeting_id`. |
| Dry-run | `vc +meeting-end` | `vc_agent_meeting_actions_dryrun_test.go::TestVCAgentMeetingActionsDryRun/end meeting` | Verifies the bot end endpoint and that `--dry-run` bypasses the real-execution confirmation gate. |

## Live E2E Blocker

`vc +meeting-end` terminates a meeting for every participant. The shared E2E tenant does not provide an isolated Calendar fixture that can reliably create an Agent-enabled meeting where the test app bot becomes Host and has a distinct eligible invitee. Running against an existing meeting would be unsafe and is not self-cleaning.

## Required Fixture

1. A dedicated test app with the Agent Calendar feature enabled for its meeting owner.
2. A disposable app-calendar event that creates a VC meeting and makes that app bot Host.
3. A second user open_id scoped to the same test app for a `SELECTED` invite assertion.

## Substitute Evidence

On 2026-08-24, the gray-lane manual lifecycle created a bot-owned Calendar event, started it, sent an `ALL_SUGGESTED` invite, ended it as the Host bot, deleted the event, and verified that the active-meeting list was empty. The fixture above is required before promoting this sequence to CI live E2E.

## Next distribution section verification

The frozen VC implementation and tests are reused without removing commands or
flags. [The next process suite](../../next_sections/vc/vc_e2e_test.go) exercises
all 13 shortcuts and the generated `vc meeting get` method through the real
CLI against isolated synthetic HTTP fixtures. It independently checks request
bodies, recipients, bounded pagination, typed failures, receipt uncertainty,
artifact bytes and local overwrite behavior. Run the whole section with
`python3 scripts/next_test.py --section vc --suite all` from the repository root.

The historical manual live lifecycle above is not current-run authorization or
passing evidence. Current live verification remains blocked until a harness
verifies current account/tenant, private visibility before creation, run-owned
host meeting and cleanup, and self-only or unambiguously Ying Wu-only recipients
(including eligible ALL_SUGGESTED invitees). No unrelated existing meeting is a
permitted fixture. The separately selectable `TestVCLivePrivateLifecycle`
records this blocker.
