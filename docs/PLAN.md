# feishu-cli-next token-efficiency plan

> **Record.** This is the plan the fork was built against. Current rules are in
> [AGENTS.md](../AGENTS.md) and [contributing](contributing.md). The
> [testing authorization boundaries](#authorization-boundaries-for-all-testing)
> below still apply to every test.

## Objective

Build a Feishu/Lark CLI named `feishu-cli-next` and one concise English agent skill that help capable shell/file agents complete real Feishu tasks with less unnecessary context and more freedom to choose an effective approach.

Optimize the complete task: discovering the right command, supplying inputs, reading results, handling errors, and confirming the outcome. A shorter response is useful only if it does not require extra calls or hide information needed to complete the task correctly.

Evaluate total agent input/output tokens alongside task success, output quality, recovery, and the ability to solve unfamiliar tasks compared with the selected official CLI and its matching skill. Aim to reduce wasted tokens without narrowing agent capability. Set no fixed percentage improvement, token quota, or mandatory reduction for every task. Data integrity and honest mutation outcomes remain correctness requirements; latency and API request counts provide additional evidence.

Implement every existing Feishu CLI capability in `feishu-cli-next`, using the sibling `feishu-cli-review` source tree as the reference. Full functional coverage and improved agent usability are joint goals. Implementation agents determine the detailed command inventory and design from the code during implementation; the examples in this plan do not limit scope.

The 2026-10-07 continuation requests speed and token improvements across every
original area and function. Compare individual capabilities against the complete
inventory, including lifecycle commands, events, host integrations and platform
operations. Report unmeasured functions and regressions explicitly; neither a
sampled schema reduction nor an area-average saving establishes a whole-CLI
lead. Preserve the reference contracts and keep local introspection measurements
separate from verified workflow/provider-token evidence.

This document defines the plan; reference code and its tests define the capabilities to preserve. All proposed capabilities remain unverified until their acceptance tests and evaluations pass.

## Product boundaries

- Inspect `feishu-cli-review`, record its source revision and any local changes, and capture the service metadata and build configuration used for the reference inventory. Reuse upstream authentication, runtime, API clients, and domain implementations where appropriate; retain license notices and record source/distribution identity in the new build. Keep the reference checkout unchanged.
- Ship one binary and one progressively disclosed English skill. Preserve user content and language preferences.
- Keep domain operations deterministic and use the host agent for reasoning. Preserve existing event, streaming, and runtime capabilities found in the reference. Do not add unrelated infrastructure such as a second LLM or filesystem mounting merely to optimize token usage.
- Preserve the reference CLI’s shortcut, generated service-command, and raw API capabilities in the new binary. A compatibility/raw surface can retain original contracts, but raw forwarding alone does not demonstrate completion of the new agent-facing interface. Reuse shared implementations; do not duplicate every operation merely to rename it.
- Use measurements to prioritize implementation order and interface improvements, not to exclude existing functions. Every reference capability remains in scope regardless of estimated frequency or token savings.
- Inventory reference platforms, architectures, build variants, installation modes, and host integrations. Preserve applicable support and document/test platform-specific behavior; do not silently narrow it to make implementation easier.

## Skill and tool design principles

Assume the agent can reason, plan, use a shell, and choose an appropriate workflow. The skill supplies Feishu-specific knowledge and discoverable capabilities. It should not dictate a universal sequence for getting work done. Development gates, subagent coordination, and testing procedures in this plan govern implementation; do not copy them into the shipped skill as instructions for end users’ agents.

- Describe what the CLI can do, when its capabilities are useful, and the non-obvious semantics that affect decisions. Keep descriptions precise and examples brief; avoid generic agent advice, exhaustive command lists, and repeated API schemas.
- Offer useful defaults with a clear reason, while allowing the agent to choose direct commands, composed operations, files, scripts, or raw access according to the task and the user's preferences. Examples illustrate possibilities, not required routes.
- Do not require routine planning, help lookup, schema inspection, export, dry-run, confirmation, or repeated readback for every task. State a prerequisite only when a particular operation actually needs it. Existing user authorization remains valid within its scope.
- Reserve strict guidance for concrete constraints such as ambiguous recipients, irreversible effects, unsupported conversions, or unsafe retries. Explain the relevant consequence concisely. Enforce deterministic validation and backend constraints in code where possible instead of making the agent repeat a checklist.
- Keep the entry skill as short as useful without a fixed token budget. Load additional help or references only when needed; avoid splitting trivial guidance across files that create more discovery work. Keep each fact in one maintained place.
- Preserve composability and user choice. A convenient command can perform repetitive mechanics internally, but should expose its effects and useful results so the agent can combine it with other tools. Do not force a single preferred workflow or hide capabilities merely to make benchmarks cheaper.

These choices follow the principles of task-calibrated guidance and progressive disclosure in [Agent Skills authoring guidance](https://agentskills.io/skill-creation/best-practices), and meaningful, focused interfaces in [Anthropic’s tool-design guidance](https://www.anthropic.com/engineering/writing-tools-for-agents). The requirements needed to implement this plan are stated here; the links provide optional rationale.

## Token-saving priorities

| Source of wasted tokens | Planned improvement | Evidence required |
| --- | --- | --- |
| Large skill catalogs and help trees | One short entry skill and operation-specific help | Lower discovery tokens without lower command-selection accuracy |
| Repeated schema lookup and API payload assembly | Typed commands for frequent complete operations | Fewer agent turns and fewer input tokens for the same verified effect |
| Verbose responses and irrelevant fields | Compact defaults and useful field projections | Lower result tokens while retaining enough information to finish the task |
| Large documents and tables copied through the model | Bounded reads and direct file-to-file artifact handoff | Large content stays outside model context and arrives intact |
| Repeated resource/account resolution | Accept supported URLs/IDs and reuse validated resource identity within an operation | Fewer lookup turns without ambiguous targeting |
| Trial-and-error failures | Local validation and concise actionable errors | Lower recovery tokens and fewer failed attempts |
| Excessive mutation workflows | Direct precise operations and software-owned no-op detection | Lower end-to-end tokens than an equivalent multi-step workflow |

## Full functional scope

The target is all functionality exposed by the reference CLI, including command families, subcommands, flags, input/output modes, supported resource and field types, event workflows, authentication/configuration, installation/lifecycle operations, and public extension or host integration contracts. Functional parity concerns user-visible behavior and public contracts; internal helper functions may be reused or reorganized.

Implementation agents must inspect the source during implementation rather than treat a hand-written feature list as exhaustive. Start with `cmd/`, `shortcuts/`, `events/`, `extension/`, `internal/registry/`, the skill/help sources, build definitions, and existing tests in `feishu-cli-review`. Inspect actual registration and generated service metadata, including conditional/platform-dependent surfaces; directory names alone are not a complete inventory.

- Build a coverage manifest mapping each reference capability to its source/command identity, new CLI entry point, shared or domain owner, relevant contracts, tests, and verification status.
- Include Docs, Sheets, Base, Tasks, Messages, Calendar, Drive, Wiki, Contacts, Mail, Meetings/Minutes, Notes, Slides, Whiteboards, OKR, applications, events, and every other capability discovered in the reference. These are examples for finding owners, not an allowlist.
- Preserve less common operations and supported options/types as well as frequent workflows. Do not reduce Docs to plain text, Messages to text send, or Sheets/Base to a small type subset if the reference supports more.
- Retain useful high-level operations, generated API commands, and raw access. Apply shared discovery/input/output improvements across the whole surface, with domain-specific improvements where they help. Commands can reuse existing implementations and metadata while remaining discoverable and usable through the new CLI.
- Record any renamed or reorganized command with a clear capability mapping. Preserve intended effects, permissions, identities, resource fidelity, and access to full results; document intentional interface differences.
- Classify missing implementation, unresolved semantics, unavailable test access, and backend limitations separately. A blocked or unverified capability remains outstanding; it is not silently dropped or counted as complete.
- Intermediate builds may expose verified completed sections, but a partial build does not satisfy the full implementation goal. Implementation agents may choose design and sequencing; they may not shrink functional scope based on predicted demand or benchmark convenience.

Freeze the reference snapshot for an auditable completion check. Newly discovered capabilities in that snapshot must be added to the manifest. Later upstream changes can be reconciled separately; full coverage does not mean inventing APIs or behaviors absent from the reference.

## Shared contracts

### Discovery and outputs

- The entry skill explains capabilities, useful defaults, non-obvious domain constraints, and where to find optional targeted help. Let the agent decide when additional guidance is needed; do not require a help lookup before every command.
- Generate capability and help metadata from command definitions. Advertise only implemented, verified capabilities; avoid duplicating API documentation in the skill.
- Default agent output is concise but informative. Offer predictable structured JSON, selectable fields/detail, and raw access so the agent can obtain the information its task needs. Keep names and parameter conventions consistent across domains; avoid overlapping commands with unclear distinctions.
- Define stable result fields for status, resource identity, concise result data, pagination/completeness, artifact references, and recovery references where applicable. Do not emit unused fields by default.
- Preserve decision-critical facts: errors, missing scopes, truncation, partial results, unknown mutation outcomes, and unsupported types must remain visible.
- Set bounded read defaults with explicit pagination, filtering, projections, and artifact output. Allow the agent to request more detail within backend/resource limits. Explain limits and continuations; never silently discard requested information or imply a partial result is complete.
- Continuation information must retain query/resource context. Do not claim snapshot-consistent pagination unless verified.
- Success results go to stdout; diagnostics go to stderr. Specify and test distinct exit behavior for rejected, partial, and unknown outcomes.

### Inputs and resource identity

- Accept typed flags for small inputs and explicit file/stdin input for substantive text or data. Normal operations should not require agents to author API payloads, receipts, or baseline metadata; raw access remains available when custom payloads are appropriate.
- Validate sizes, required fields, supported types, and ambiguity before mutation. Define literal text, omitted fields, empty values, and explicit clearing separately.
- Resolve supported URLs and IDs to an unambiguous resource. Do not silently choose a recipient or resource by an ambiguous display name.
- Bind operations and reusable artifacts to the effective Feishu/Lark environment, tenant, account, and user/app identity. Revalidate binding before use; explain mismatches concisely.
- Keep credentials out of artifacts, results, logs, and receipts. Read-only doctor reports configuration health without printing credential contents or modifying installation state.

### Artifacts and typed data

- Support direct backend-to-file export and file-to-backend input for substantial data, while retaining convenient inline results for small tasks. Return artifact references with schema, counts, completeness, and relevant limitations so the agent can choose whether to inspect or pass them onward.
- Choose a documented typed representation for each supported workflow. Use CSV only where generated metadata and explicit conversion rules preserve the supported semantics.
- Define null, empty string, omitted value, formula, literal text beginning with `=`, leading-zero identifiers, numeric precision, dates, and time zones. Reject ambiguous conversions.
- Identify records/cells by native IDs or explicit stable addressing. Never infer deletion from omitted rows, columns, or cells.
- Inventory and preserve all fields, formulas, and rich content supported by the reference. Document conversion rules before implementing writes. If an optimized format cannot express a supported value, provide a lossless representation or equivalent operation; do not drop the capability. Reject genuinely unsupported or ambiguous conversions explicitly.
- Version metadata; use restrictive local permissions, atomic updates, and locks where state is shared. Define retention and explicit cleanup for artifacts/receipts, and fail safely on incompatible or corrupted state.

### Mutations and recovery

- Validate the complete bounded operation before the first write. Define recipient, notification, assignment, append, patch, and clear effects in the relevant command contract.
- Record durable attempt evidence before issuing a mutation. If required evidence cannot be persisted, stop before writing.
- Classify effects as confirmed applied, confirmed unapplied, or unknown, with supporting evidence. Do not infer failure from a timeout or success from a local journal.
- Retry mutations only when verified backend idempotency or reconciliation makes the retry safe. Bound retries and API rate-limit recovery.
- Handle cancellation, partial batches, accepted-but-lost responses, and remote success followed by local state-save failure. Provide concise receipt inspection and read-only reconciliation where useful.
- Verify no-op behavior in software. Unchanged supported content causes zero backend mutations.
- Probe conditional-write behavior instead of trusting revision field names. Local locks and readback do not prevent external concurrent edits. Without suitable backend protection, document the remaining race and implement honest behavior consistent with the actual backend contract. An unresolved safe implementation remains tracked work rather than a reason to remove the capability from scope.

## Parallel development after the foundation

Complete authentication and all shared foundation work in P1 before starting parallel domain implementation. Pass the foundation section tests and establish stable interfaces for authentication, account/resource binding, transport, inputs, outputs/errors, artifacts, mutation receipts, and test fixtures.

After that gate, use subagents to develop all independent capability sections from the inventory concurrently, including but not limited to Docs, Sheets, Base, Tasks, and Messages. Split independent functions within a domain among additional subagents when interfaces and file ownership are clear. Prefer useful independent work over a fixed agent count; do not serialize unrelated domains.

- The coordinating agent owns the complete coverage manifest, section assignment, shared interfaces, command registration, entry-skill assembly, integration, and release acceptance. Domain subagents consume the shared foundation instead of creating competing implementations.
- Give every subagent a bounded assignment, explicit file/module ownership, interface contracts, and completion criteria. Tell each subagent it is not alone in the codebase, must preserve others' edits, and must coordinate overlapping changes.
- Each implementation assignment starts by inspecting the relevant reference code, registrations, metadata, and tests to establish full section coverage. It includes unit tests and relevant operation help, plus written E2E scenarios for the workflows it contributes to. Assign one owner to assemble each section’s executable E2E suite. Report changed files, unit-test results, E2E test files, and unresolved assumptions when handing work back; execute the assembled E2E suite at section completion.
- Route shared-contract changes through the coordinator and notify affected subagents before integration. Use isolated worktrees where concurrent edits would otherwise conflict.
- Integrate completed functions into their owning section as they arrive. Run workflow/live/token evaluations only when that whole section is complete; function handoffs are not separate end-to-end testing gates.
- Each domain can reach its section test gate independently. The Sheets-to-Base handoff waits for both complete sections; it does not block their independent implementation or domain tests.
- Keep concurrent live tests on distinct authorized fixtures and account for shared API limits. Keep comparative token measurements isolated from competing evaluation workloads.

Dependency sequence: P0 inventory/baseline → P1 authentication and foundation → P2 capability sections in parallel → P3 full coverage and cross-section integration → P4 release. The coordinator continues integration and shared work while subagents implement independent functions.

## Testing cadence

This rule applies to every implementation section and work package, including baseline tooling, authentication, shared infrastructure, discovery/skills, artifacts, every domain, packaging, and future additions. Docs, Sheets, and Base are examples, not an exhaustive list. Before starting a section, define its scope, test owner, complete acceptance suite, and dependencies.

- Complete authentication and the shared foundation first, then implement all independently owned capability sections in parallel. Define each section's supported scope before starting it; retain a separate completion and testing gate for every section.
- Write and run focused unit tests alongside each small function or change. Cover input validation, transformations, request construction, output contracts, no-op detection, and failure-state transitions using deterministic fixtures and mock transports.
- Do not run separate integration, end-to-end, live, manual workflow, or agent token evaluations for each small subfunction. Prepare those scenarios during implementation, then run them together after the section's implementation, help, error handling, and unit tests are complete.
- When any section’s implementation is finished, run its complete unit, integration, and written E2E suites, relevant regressions in completed sections, authorized live verification where backend behavior is involved, and comparative token evaluation for its agent-facing workflows. Run all applicable scenarios in the section, not only the latest changed function or a few happy-path examples. Record any non-applicable check with a concrete reason; skipped or blocked required tests do not count as passing.
- A section is complete only after its required suite passes and results are recorded. Do not postpone its E2E tests until the final release or wait for unrelated parallel sections to finish. Cross-section tests run as soon as all participating sections are ready and must pass before accepting the combined workflow.
- Resolve section-level failures and rerun the affected scenarios before accepting the section. Do not repeatedly rerun unrelated domains after small fixes; shared-contract changes require regression checks in affected completed sections at the next section boundary.
- Keep held-out tasks untouched until final release evaluation. Section evaluations use development tasks and the same token-accounting/comparison rules as the final evaluation.
- Run the complete cross-domain suite and held-out evaluation after all release sections are finished, in addition to the tests already run at every section boundary. Continuous CI runs unit tests and static checks; broader section suites run at section completion and release gates.

### Written E2E tests

- Write executable, version-controlled E2E tests during implementation for every section. Scenario notes, manual transcripts, unit tests, and agent benchmarks do not replace the E2E suite.
- Exercise the built CLI through its real process entry point and public commands. For installation, authentication, skills, and evaluation tooling, exercise the corresponding complete user or harness flow. Assert exit codes, stdout/stderr contracts, artifacts, and independently observed effects rather than trusting a success message.
- Cover each advertised workflow and its applicable failure/recovery paths: invalid or ambiguous input, identity/scope mismatch, pagination/truncation, no-op, partial effects, interruption, and unknown outcomes. Include cross-section artifact handoffs and preservation checks where applicable.
- Provide deterministic offline E2E tests using isolated local state and a controlled backend for repeatable fault injection. Write separately selectable live E2E tests for backend-dependent guarantees; label the evidence separately and never treat offline success as proof of live behavior.
- Each test owns setup, expected results, and cleanup, including cleanup on failure. Keep fixtures isolated for parallel execution, bound execution time, and preserve sanitized failure evidence.
- Provide documented test commands for each section and the full suite, with explicit prerequisites and machine-readable pass/fail/skip results. Wire these commands into section-completion and release CI jobs. Do not execute E2E after each small subfunction; run the completed section’s suite at its acceptance boundary.

## Evaluation protocol

### Comparison arms

Use the same pinned backend, task fixtures, model, reasoning settings, tools, permissions, and task budgets:

1. Official CLI with its matching official skill.
2. Official CLI with concise English guidance covering equivalent task capabilities.
3. Optimized `feishu-cli-next` CLI with its concise English skill.

Arm 1 versus 3 measures the complete product effect. Arm 2 versus 3 helps isolate interface improvements from shorter guidance. Keep guidance coverage comparable; document differences that prevent a clean attribution. Do not make language-specific claims without a separate equivalent-content language comparison.

### Task suite and measurements

- Define representative development and held-out tasks across every agent-facing section in the coverage inventory before tuning it. Include unfamiliar combinations and user-requested approaches. Comparative task sampling measures usability and efficiency; it does not replace full capability/contract coverage or justify omitting commands.
- Include simple reads/writes, ambiguous targets, missing scopes, bounded pagination, Unicode, empty values, a large artifact transfer, and a recoverable failure. Keep destructive fault injection in deterministic fixtures.
- Run repeated independent trials per task per arm from equivalent fresh states; choose and record a balanced sample plan sufficient to assess variability. Randomize run order and use fresh agent context for primary measurements; report warm-session measurements separately.
- Record total model input/output tokens across every turn, including injected skills, help, tool results, errors, and recovery. Record cached input separately without treating it as zero tokens. Report tokenizer/model versions and the harness accounting method.
- Score outcomes and user-intent fulfillment, not adherence to a prescribed command sequence. Accept alternative valid strategies and useful composition with other tools. Verify success against expected content and actual effects, independently of the agent's final claim. Count wrong recipients, corrupted types/content, unintended changes, and misleading outcome reports as failures.
- Inspect traces for unnecessary instruction-following, repeated discovery, missing detail, and useful autonomous choices. Compare skill-assisted behavior with CLI help alone when it can reveal whether guidance adds value. Report success rate, token distributions, latency, API calls, retries, and failure categories per task and domain. Include failed-run token costs so early failures cannot appear efficient.
- Report tokens per successful completion as total tokens spent across all attempts divided by verified successes. Also report matched-success comparisons, where both arms succeed, to expose task-mix effects. If there are no successes, do not report a finite efficiency score.
- Sanitize traces and publish reproducible fixture and environment manifests. Keep private user content and credentials out of evaluation artifacts.

### Evidence-based release review

Define task success and correctness criteria before evaluation. Use measured results to understand tradeoffs and guide changes, without fixed improvement percentages or token ceilings.

- Report task-level and domain-level token changes, success, quality, recovery, and variability across comparison arms. Include regressions, failed-run costs, and inconclusive results rather than reducing the decision to one aggregate score.
- Assess whether the CLI and skill remove unnecessary agent work while retaining useful flexibility and capability. A workflow that uses more tokens to achieve a better or previously unsupported result may be worthwhile; explain the tradeoff.
- Review unexpected regressions and unnecessary complexity. Prefer evidence-backed simplification, clearer descriptions, or better defaults over adding universal instructions that constrain every task.
- Require full reference-capability reconciliation and the complete section and release correctness suites to pass. Unintended-recipient writes, silent corruption, hidden truncation, and unsafe duplicate retries are failures to resolve, regardless of token savings.
- Publish only performance claims supported by the measurements. If results are too variable, repeat balanced trials or report uncertainty. Keep held-out tasks separate from tuning and version any changes to the evaluation.

Release acceptance combines full functional coverage, passing correctness/E2E tests, and a documented assessment of practical usefulness, token efficiency, and agent capability. It does not depend on hitting a numeric improvement target.

## Work packages

Every package and capability section must deliver written integration/E2E tests and run its complete section suite before acceptance. Implementation agents derive detailed tasks and additional sections from reference-code inspection; the plan does not prescribe a limited command list.

### P0 — Inventory the reference and prepare evaluation

Deliverables:

- Capture the reference revision, local changes, service metadata, platform/build variants, license requirements, and reproducible build setup.
- Inspect registrations, schemas, runtime/extension surfaces, and tests to create the complete capability manifest. Assign each capability to a section, including shared infrastructure and non-domain operations.
- Prepare baseline tasks, independent outcome checks, token accounting, comparison arms, and an executable evaluation harness. Let section owners refine scenarios after inspecting their code.
- Prepare inventory comparison checks that detect missing commands, flags, schemas, types, and supported behaviors. Record intentional interface mappings explicitly.

Section completion tests: run the inventory/harness unit, integration, and E2E suite. Verify that the inventory captures generated and conditional surfaces, and that a controlled harness run checks task loading, process execution, scoring, token accounting, failure reporting, and result generation. Keep product held-out tasks unused.

Acceptance: the reference snapshot and initial manifest are reproducible; capability owners and source evidence are recorded; harness and inventory checks pass. Refine newly discovered details during implementation without discarding coverage.

### P1 — Complete authentication and all shared foundations

Depends on P0.

Deliverables:

- Implement or reuse all reference authentication, identity/account/environment handling, configuration, and shared runtime behavior needed by the capability sections.
- Establish transport, input/file/stdin, output/error, artifact, receipt, event/stream, and extension interfaces as required by the inventory. Preserve compatibility contracts while providing concise agent-facing discovery and output controls.
- Complete command registration, generated service loading, schema/help infrastructure, and the concise agent-trusting skill framework. Document ownership and interfaces before delegation.
- Write focused unit tests during implementation and executable integration/E2E tests for the complete foundation workflows.

Section completion tests: run the full foundation suite, including authentication/configuration lifecycles, identity and permission failures, stream/output contracts, artifacts, persistence failures, discovery, and relevant host integration behavior. Compare complete agent-facing setup/discovery tasks.

Acceptance: all identified shared prerequisites and authentication contracts pass; shared interfaces are ready for parallel implementation. Do not start domain subagents against an incomplete foundation.

### P2 — Implement every capability section in parallel

Depends on P1 acceptance. Create one owned section for every domain or coherent capability group identified in P0; split independent functions among subagents as useful.

Each section owner will:

- Inspect the reference implementation, command registration, generated metadata, field/type support, help, and tests. Complete its manifest entries and identify cross-section dependencies before choosing implementation details.
- Implement every mapped capability through reuse or adaptation, including uncommon options, reads/writes, resource operations, events, and recovery paths that belong to the section. Do not stop at a representative subset.
- Apply shared token-efficient discovery, input/output, and artifact conventions while preserving full detail, composability, and agent choice. Keep mechanical procedures in code and skill guidance concise.
- Write and run unit tests as functions are developed. Write the section’s integration and E2E suites during development, then run them after the whole section is implemented.
- Record functional parity, deliberate interface mappings, test results, evaluation results, and unresolved issues. A command merely present in help or accessible only through an unreviewed raw forwarding path is not sufficient evidence.

Section completion tests: run all unit, integration, and E2E tests for the section, applicable reference regressions, authorized live verification, and comparative agent tasks. Verify its entire capability mapping and relevant error/recovery behavior, not only the functions named as examples here.

Coverage examples to expand from the source:

| Section | Verification emphasis |
| --- | --- |
| Docs and rich content | Every supported read/edit/resource operation, native identity and formatting preservation, ambiguity, no-op, concurrency limitations, and interruption |
| Sheets and Base | All supported data/schema operations and types, formulas, resource addressing, pagination, batches, permissions, and large artifact transfer |
| Tasks, Messages, Calendar, Mail, and other actions | Complete supported lifecycles, identities, recipients, assignment/notification effects, attachments or rich inputs where supported, and mutation outcomes |
| Drive, Wiki, and other resource domains | Supported navigation, search, transfers, links, permissions, and identity preservation |
| Events and streaming | Registration, input/output, long-running behavior, cancellation, reconnection/replay semantics, and relevant event types |
| Extensions, lifecycle, and other shared surfaces | Public contracts, host integration, installation/configuration/update behavior, and platform/build variants |

Acceptance: the section’s full mapped functionality is implemented and its required suite passes. Efficiency and agent-capability tradeoffs are documented without numeric improvement targets. Unresolved capabilities keep the section open; unrelated sections can continue independently.

### P3 — Reconcile full coverage and cross-section behavior

Depends on completed participating P2 sections; begin each cross-section test as soon as its dependencies are ready.

Deliverables:

- Compare the assembled CLI against the reference inventory and reconcile every command/capability, option, schema/type, event, public integration, and build/platform mapping. Have implementation agents inspect unexplained differences instead of assuming a smaller interface is acceptable.
- Complete all cross-section workflows, including large artifact handoff such as Sheets-to-Base, resource resolution, shared identities, and recovery across operation boundaries.
- Verify a 10,000-row transfer preserves supported values/types outside model context; examine token behavior across data sizes without imposing a percentage target.
- Resolve integration conflicts and gaps, confirm discovery reaches every capability, and keep skill/help metadata aligned with the executable surface.

Section completion tests: run complete cross-section integration/E2E suites and relevant regressions. Require manifest reconciliation with no omitted or unverified required capabilities; do not count blocked tests as passed.

Acceptance: full reference functionality is accounted for, implemented, and verified; combined workflows preserve content, identity, effects, and useful agent choice.

### P4 — Evaluate and package the full release

Depends on acceptance of every inventory section and P3.

Deliverables:

- Run the complete unit/integration/E2E suite and held-out comparative evaluation across all sections. Publish sanitized evidence, tradeoffs, and backend limitations.
- Configure continuous unit/static checks and section-completion/release jobs for broader suites and capability reconciliation.
- Preserve reference installation and platform capabilities under the new product identity; produce checksummed artifacts and project-scoped skill install/update/uninstall support.
- Write and run packaging E2E tests for supported platforms/build variants, clean setup, collisions, upgrades, uninstall, and preservation of unrelated profiles, skills, and files.
- Provide concise setup instructions, comprehensive on-demand capability discovery, migration mappings, and accurate behavior/recovery documentation.

Acceptance: full functional coverage and all required tests pass; the evidence-based review assesses token efficiency and agent capability without fixed improvement targets. Partial milestones remain incomplete toward this release objective.

## Authorization boundaries for all testing

These are testing-only authorization boundaries. They apply to every test, including unit, integration, E2E, live, manual, smoke, regression, agent evaluation, test setup, and cleanup, whether run by the coordinating agent, a subagent, a script, or CI. Offline tests use synthetic data and must not accidentally perform live actions.

Do not encode these test-specific recipient or visibility restrictions as product limitations, runtime allowlists, or instructions in the shipped skill. The CLI must retain full functionality for users acting within their own authorization. During testing:

- Do not send messages, documents, files, invitations, notifications, shares, or any other content to anyone in Feishu except **Ying Wu**. This includes indirect delivery through mentions, assignments, subscriptions, group chats, permission changes, or automated workflow effects.
- Ying Wu is the only authorized other recipient or collaborator. Resolve that person's identity unambiguously before any delivery or access grant; a matching display name alone is insufficient. Do not send to groups or destinations that include anyone else.
- The user authorizes freely creating documents, Sheets, Base resources, and other Feishu objects for testing when they are visible only to the currently logged-in account. Routine creation and use within this boundary do not require repeated approval.
- Private resources must remain restricted to that account unless intentionally shared with the authorized exception, Ying Wu. Do not enable public, link-wide, organization-wide, inherited group, or other-user access. Verify destination and creation visibility before writing; creating broadly visible content and restricting it afterward does not satisfy this boundary.
- Bind every live run to the effective current account and tenant. Use isolated, clearly labeled resources created for testing; do not edit or delete unrelated pre-existing resources. Clean up only resources created by the run and preserve any the user wants to keep.
- If a feature necessarily exposes content or notifies anyone other than Ying Wu, do not perform that live action. Continue with offline fixtures and record the live case as blocked under this boundary. All test runners and subagents inherit this restriction and cannot broaden test authorization.

Write and run unit tests using synthetic fixtures and mock transports during implementation. Write live E2E tests alongside the implementation, then batch their execution into the completed section’s full suite; do not run live tests after each small function. The private-resource authorization above provides the permitted fixture scope; verify actual account, identity, and visibility before using it. Ordinary CI must not depend on personal credentials.

If the required privacy or recipient restrictions cannot be verified, keep the affected live test blocked and continue independent offline work. Blocked or skipped live tests are not passing evidence, and live-dependent capabilities remain unverified until their required checks can run within these boundaries.

## Completion and ongoing development

Completion means every capability in the captured Feishu CLI reference is available in `feishu-cli-next`, with complete section tests, written E2E coverage, full inventory reconciliation, and an evidence-based review of token efficiency and agent capability. The goal is not limited to frequently used domains, a private alpha subset, or the examples listed in this document.

Implementation agents choose detailed designs, discover missing inventory entries, and organize work during implementation. Prioritization controls order and parallelism, not whether a reference capability will be delivered. Later upstream changes and genuinely new capabilities receive the same inventory, implementation, testing, and evaluation treatment.
