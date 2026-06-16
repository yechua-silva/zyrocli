# Delta for Scheduler Engine

## MODIFIED Requirements

### Requirement: PhaseRunner Interface

The scheduler MUST define a `PhaseRunner` interface with a method `Run(ctx, *Config) (*Result, error)`. Each phase MUST implement this interface. The `Result` MUST expose phase name, status (success/failure/skipped/abort), and optional error.
(Previously: Result had approval_status (approved/rejected/pending))

#### Scenario: Happy path
- GIVEN a phase that implements `PhaseRunner`
- WHEN `Run` completes successfully
- THEN `Result.Status` is `success`

#### Scenario: Phase failure
- GIVEN a phase that returns an error
- WHEN `Run` fails
- THEN `Result.Status` is `failure` and the error wraps the cause

### Requirement: Sequential DAG Execution with Human Validation

The scheduler MUST execute phases in strict order: F1 → F2 → F3 → F4. Each phase MUST complete before the next begins. If a phase returns failure, the scheduler MUST abort remaining phases. After each phase, the scheduler MUST prompt for human approval — there is no `--auto` mode. The scheduler MUST act as a **validation harness**: it orchestrates real phase implementations (not stubs) and enforces governance rules.
(Previously: Same DAG execution but phases were stubs with F2-F4 placeholder banners)

#### Scenario: Full sequence
- GIVEN all phases succeed
- WHEN the scheduler runs
- THEN each phase executes in order and all results are returned

#### Scenario: Abort on failure
- GIVEN F2 fails
- WHEN the scheduler reaches F2
- THEN F3 and F4 are skipped; the result contains the partial execution

#### Scenario: Approval required after each phase
- GIVEN a phase completes with success
- WHEN the phase returns
- THEN the user is prompted for approval before the next phase starts

### Requirement: Config-Driven Governance

The scheduler MUST load governance config from `handoff.yaml` via `LoadConfig(path)` including `governance.mode`, `limits.max_loops`, and `limits.phase_timeout`. If `phase_timeout` is absent, MUST default to 10 minutes. If `max_loops` is 0, MUST default to 5.
(Previously: max_loops defaulted to 1, max_loop was per-phase retry; validation was simpler)

#### Scenario: Default timeout
- GIVEN handoff.yaml with no `limits.phase_timeout`
- WHEN `LoadConfig("handoff.yaml")` is called
- THEN `PhaseTimeout` is 10 minutes

#### Scenario: Parsed timeout
- GIVEN handoff.yaml with `limits.phase_timeout: "5m"`
- WHEN `LoadConfig("handoff.yaml")` is called
- THEN `PhaseTimeout` is 5 minutes

#### Scenario: Default loops
- GIVEN handoff.yaml with no `limits.max_loops`
- WHEN `LoadConfig("handoff.yaml")` is called
- THEN `MaxLoops` is 5

#### Scenario: Retry on failure
- GIVEN `max_loops: 3` in config
- WHEN a phase fails
- THEN the scheduler retries up to 2 more times before reporting failure

### Requirement: F1 Phase — Real Skill Advisor Integration

The F1 phase MUST call `handoff.Parse()` on `handoff.yaml`. If parse fails, MUST return failure. On success, F1 MUST invoke the Skill Advisor to score available skills against the handoff context and print recommendations. F1 is no longer a stub.
(Previously: F1 called a skilladvisor placeholder and printed "not yet implemented")

#### Scenario: Valid handoff with recommendations
- GIVEN a valid `handoff.yaml` and registered skills
- WHEN F1 runs
- THEN it parses the file, queries the Skill Advisor, and prints recommended skills

#### Scenario: Missing file
- GIVEN no `handoff.yaml` in CWD
- WHEN F1 runs
- THEN it returns an error wrapping "handoff.yaml not found"

### Requirement: F2–F4 — Real Phase Implementations

Phases F2, F3, and F4 MUST execute real pipeline logic (spec writing, implementation, delivery) rather than printing "not yet implemented". Each phase MUST accept the scheduler Config and context cancellation.
(Previously: F2–F4 printed "not yet implemented" and returned success)

#### Scenario: F2 runs spec phase
- GIVEN the scheduler reaches F2
- WHEN F2 runs
- THEN it executes the spec-writing macro (not a stub)

#### Scenario: Context cancellation during F3
- GIVEN a context that is cancelled during F3 execution
- WHEN the deadline elapses
- THEN the phase returns with timeout status

### Requirement: HarnessValidator — Transition Enforcement

The scheduler MUST include a `HarnessValidator` type that enforces sequential phase transitions with human approval. The validator MUST block any transition where `approved=false`. It MUST track `CurrentPhase()` and provide `NextPhase()` based on the ordered phase list.

#### Scenario: Blocked transition
- GIVEN a HarnessValidator at F1
- WHEN `ValidateTransition(F1, F2, false)` is called
- THEN an error is returned indicating the transition is blocked

#### Scenario: Valid transition
- GIVEN a HarnessValidator at F1
- WHEN `ValidateTransition(F1, F2, true)` is called
- THEN no error is returned

#### Scenario: Next phase calculation
- GIVEN a HarnessValidator at F1
- WHEN `NextPhase()` is called
- THEN it returns F2

#### Scenario: Unknown phase
- GIVEN a HarnessValidator with an unrecognized phase
- WHEN `ValidateTransition("F5", "F6", true)` is called
- THEN an error is returned indicating the phase is unknown

### Requirement: GuidedApproval — Contextual Dialog

The scheduler MUST replace `PromptApproval` with a `GuidedApproval` type that displays a structured dialog: phase name, summary, recommendation, risk, and the prompt with options `s (sí)`, `n (no)`, `d (detalle)`. The "d" option must display the full agent output and re-prompt. A backward-compatible `PromptApproval(phase, summary)` standalone function MUST still exist.

#### Scenario: Standard approval
- GIVEN a GuidedApproval with Phase, Summary, Recommend, and Risk
- WHEN `PromptApproval()` is called with input "s"
- THEN it returns true

#### Scenario: Detail mode
- GIVEN a GuidedApproval with FullOutput set
- WHEN the user enters "d"
- THEN the full output is printed and the user is re-prompted

#### Scenario: Rejection
- GIVEN a GuidedApproval
- WHEN the user enters "n"
- THEN it returns false

#### Scenario: Invalid input retry
- GIVEN a GuidedApproval
- WHEN the user enters an unrecognized response
- THEN it prints an error message and re-prompts

### Requirement: MacroPhaseRunner — Configurable Phase Runner

The scheduler MUST provide a `MacroPhaseRunner` type that wraps an agent function and optional validator. The runner MUST support context cancellation and configurable timeouts. It MUST implement `PhaseRunner`.

#### Scenario: Successful execution
- GIVEN a MacroPhaseRunner with a successful agent function
- WHEN `Run(ctx, cfg)` is called
- THEN the result status is `success`

#### Scenario: Validator rejects failure
- GIVEN a MacroPhaseRunner with a validator that checks success
- WHEN the agent function returns failure
- THEN the validator returns an error

#### Scenario: Context cancellation
- GIVEN a MacroPhaseRunner with a cancelled context
- WHEN `Run(ctx, cfg)` is called
- THEN it returns a timeout error

### Requirement: F1-F4 Real Implementations via MacroPhaseRunner

Phases F1 through F4 MUST be implemented as `MacroPhaseRunner` instances with real agent functions. F1 MUST parse handoff.yaml and call the Skill Advisor. F2 MUST run CIO compile. F3 and F4 MUST provide phase-appropriate summaries with governance context.

#### Scenario: F1 calls Skill Advisor
- GIVEN a valid handoff.yaml in CWD
- WHEN F1AgentFunc is called
- THEN it parses the handoff and queries the Skill Advisor
