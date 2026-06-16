# Scheduler Engine Specification

## Purpose

Define the scheduler harness that executes a 4-phase DAG (F1→F2→F3→F4) with HarnessValidator transition enforcement, GuidedApproval gates, and governance-aware execution for the zyrocli SDD pipeline.

## Requirements

### Requirement: PhaseRunner Interface

<!-- delta:start -->
The scheduler MUST define a `PhaseRunner` interface with a method `Run(ctx, *Config) (*Result, error)`. Each phase MUST implement this interface. The `Result` MUST expose phase name, status (success/failure/skipped/abort), optional error, and `Skills []ValidatedSkill` propagated from F1 for downstream phases.
<!-- delta:end -->

<details>
<summary>Scenarios</summary>

- **Happy path**: GIVEN a phase that implements `PhaseRunner`; WHEN `Run` completes successfully; THEN `Result.Status` is `success`.
- **Phase failure**: GIVEN a phase that returns an error; WHEN `Run` fails; THEN `Result.Status` is `failure` and the error wraps the cause.
</details>

### Requirement: Sequential DAG Execution with HarnessValidator

The scheduler MUST execute phases in strict order: F1 → F2 → F3 → F4. Each phase MUST complete before the next begins. The `HarnessValidator` MUST enforce sequential transitions and require human approval for each step. If a phase returns failure, the scheduler MUST abort the remaining phases.

<details>
<summary>Scenarios</summary>

- **Full sequence**: GIVEN all phases succeed; WHEN the scheduler runs; THEN each phase executes in order and all results are returned.
- **Abort on failure**: GIVEN F2 fails; WHEN the scheduler reaches F2; THEN F3 and F4 are skipped; the result contains the partial execution.
- **Blocked transition**: GIVEN human approval is denied; WHEN the scheduler receives "n"; THEN HarnessValidator blocks the next phase and the scheduler aborts.
</details>

### Requirement: GuidedApproval Gates

The scheduler MUST prompt for human approval after each phase using `GuidedApproval`. The dialog MUST display: phase name, summary, recommendation, and risk warning. Accepted responses MUST be "s" (proceed), "n" (abort), or "d" (detail). The "d" option MUST show full agent output and re-prompt. There is no `--auto` mode.

<details>
<summary>Scenarios</summary>

- **Guided approval**: GIVEN a phase completes with success; WHEN the user enters "s"; THEN execution proceeds to the next phase.
- **Reject halts**: GIVEN the user answers "n" at a prompt; WHEN the scheduler receives the response; THEN execution stops and remaining phases are skipped.
- **Detail mode**: GIVEN the user enters "d"; WHEN the scheduler reads the response; THEN full agent output is displayed and the prompt recurs.
- **Invalid input retry**: GIVEN the user enters "maybe"; WHEN the scheduler reads the response; THEN it re-prompts until "s", "n", or "d" is entered.
</details>

### Requirement: Config-Driven Governance

The scheduler MUST load governance config from `handoff.yaml` via `LoadConfig(path)` including `governance.mode`, `limits.max_loops`, and `limits.phase_timeout`. If `phase_timeout` is absent, MUST default to 10 minutes. If `max_loops` is 0, MUST default to 5. If a phase exceeds its timeout, the scheduler MUST abort with a timeout error.

<details>
<summary>Scenarios</summary>

- **Default config**: GIVEN handoff.yaml with no `limits.phase_timeout` or `limits.max_loops`; WHEN `LoadConfig()` is called; THEN `PhaseTimeout` is 10 minutes and `MaxLoops` is 5.
- **Timeout abort**: GIVEN a phase exceeds its configured timeout; WHEN the deadline elapses; THEN the scheduler returns a timeout error and aborts remaining phases.
- **Retry on failure**: GIVEN `max_loops: 3`; WHEN a phase fails; THEN the scheduler retries up to 2 more times before reporting failure.
</details>

<!-- delta:start -->
### Requirement: F1 Phase — Unified Skill Discovery with DiscoverAndRank

The F1 phase MUST call `handoff.Parse()` on `handoff.yaml`. If parse fails, MUST return failure. On success, F1 MUST invoke `skilladvisor.DiscoverAndRank(payload, n)` which runs the unified 6-layer validation pipeline (API skills.sh + local registry + merge + ValidateAndScore). F1 MUST store `Result.Skills []ValidatedSkill` so downstream phases (F2+) can consume ranked, validated skills. F1 is implemented as a `MacroPhaseRunner`.
<!-- delta:end -->

<details>
<summary>Scenarios</summary>

- **Valid handoff with validated skills**: GIVEN a valid `handoff.yaml` in CWD; WHEN F1 runs; THEN it parses the file, calls `DiscoverAndRank`, and `Result.Skills` contains validated entries with scores.
- **Skills propagated to F2**: GIVEN F1 produces `Result.Skills` with 3 entries; WHEN F2 runs; THEN `Result.Skills` is accessible in the scheduler context for downstream decisions.
- **Missing file**: GIVEN no `handoff.yaml` in CWD; WHEN F1 runs; THEN it returns an error wrapping "handoff.yaml not found".
- **Invalid YAML**: GIVEN a malformed `handoff.yaml`; WHEN F1 runs; THEN `handoff.Parse()` returns an error and F1 returns failure.
</details>

### Requirement: F2–F4 — Real Phase Implementations

Phases F2, F3, and F4 MUST execute real pipeline logic via `MacroPhaseRunner` instances. F2 MUST run CIO compile from the handoff context. F3 and F4 MUST provide phase-appropriate summaries with governance context. Each phase MUST accept the scheduler Config and context cancellation.

<details>
<summary>Scenarios</summary>

- **F2 compiles CIO**: GIVEN the scheduler reaches F2; WHEN F2 runs; THEN it creates a CIO from handoff context and compiles it to Engram entries.
- **F3 implementation summary**: GIVEN the scheduler reaches F3; WHEN F3 runs; THEN it returns a summary with governance mode and max tasks.
- **Context cancellation during F3**: GIVEN a context that is cancelled during F3 execution; WHEN the deadline elapses; THEN the phase returns with timeout status.
</details>

### Requirement: MacroPhaseRunner

The scheduler MUST provide a `MacroPhaseRunner` type that implements `PhaseRunner`. It MUST wrap an agent function and an optional validator. The runner MUST support context cancellation with configurable timeout.

<details>
<summary>Scenarios</summary>

- **Success with validator**: GIVEN a runner with a successful function and a validator; WHEN `Run` completes; THEN both function and validator succeed.
- **Validator rejects**: GIVEN a runner with a validator that checks success; WHEN the function returns failure; THEN the validator returns an error.
- **Context cancel**: GIVEN a cancelled context; WHEN `Run` is called; THEN it returns a timeout error.
</details>
