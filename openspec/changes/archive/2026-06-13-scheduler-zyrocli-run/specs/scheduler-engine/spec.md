# Scheduler Engine Specification

## Purpose

Define the scheduler state machine that executes a 4-phase DAG (F1→F2→F3→F4) with approval gates, max-loop enforcement, and governance-aware execution for the zyrocli SDD pipeline.

## Requirements

### Requirement: PhaseRunner Interface

The scheduler MUST define a `PhaseRunner` interface with a method `Run(ctx, *Config) (*Result, error)`. Each phase MUST implement this interface. The `Result` MUST expose phase name, status (success/failure/skipped), and approval status (approved/rejected/pending).

<details>
<summary>Scenarios</summary>

- **Happy path**: GIVEN a phase that implements `PhaseRunner`; WHEN `Run` completes successfully; THEN `Result.Status` is `success` and `Result.ApprovalStatus` is `pending`.
- **Phase failure**: GIVEN a phase that returns an error; WHEN `Run` fails; THEN `Result.Status` is `failure` and the error wraps the cause.
</details>

### Requirement: Sequential DAG Execution

The scheduler MUST execute phases in strict order: F1 → F2 → F3 → F4. Each phase MUST complete before the next begins. If a phase returns failure, the scheduler MUST abort the remaining phases.

<details>
<summary>Scenarios</summary>

- **Full sequence**: GIVEN all phases succeed; WHEN the scheduler runs; THEN each phase executes in order and all results are returned.
- **Abort on failure**: GIVEN F2 fails; WHEN the scheduler reaches F2; THEN F3 and F4 are skipped; the result contains the partial execution.
</details>

### Requirement: Approval Gates

The scheduler MUST prompt for human approval after each phase unless `--auto` mode is active. The approval prompt MUST block until stdin provides a response. Accepted responses MUST be "y" (proceed) or "n" (abort). Any other input MUST re-prompt.

<details>
<summary>Scenarios</summary>

- **Interactive approval**: GIVEN interactive mode; WHEN F1 completes with success; THEN the user is prompted "[Phase] Continue? [y/n]:" and execution proceeds only on "y".
- **Reject halts**: GIVEN the user answers "n" at a prompt; WHEN the scheduler receives the response; THEN execution stops and remaining phases are skipped.
- **Invalid input retry**: GIVEN the user enters "maybe"; WHEN the scheduler reads the response; THEN it re-prompts until "y" or "n" is entered.
</details>

### Requirement: Config-Driven Governance

The scheduler MUST load governance config from `handoff.yaml` including `max_loops` (per-phase retry limit) and `timeout` (per-phase max duration). If `max_loops` is 0 or absent, the scheduler MUST default to 1 (no retry). If a phase exceeds its timeout, the scheduler MUST abort with a timeout error.

<details>
<summary>Scenarios</summary>

- **Default loops**: GIVEN handoff.yaml with no `scheduler.max_loops`; WHEN a phase runs; THEN it executes exactly once with no retry.
- **Retry on failure**: GIVEN `max_loops: 3`; WHEN a phase fails; THEN the scheduler retries up to 2 more times before reporting failure.
- **Timeout abort**: GIVEN a phase exceeds its configured timeout; WHEN the deadline elapses; THEN the scheduler returns a timeout error and aborts remaining phases.
</details>

### Requirement: F1 Phase Stub

The F1 phase MUST call `handoff.Parse()` on `handoff.yaml`. If parse fails, the phase MUST return failure. On success, F1 MUST print a contract summary (project name, phases count) and call a placeholder `skilladvisor` invocation. F1 MUST return approved/rejected status.

<details>
<summary>Scenarios</summary>

- **Valid handoff**: GIVEN a valid `handoff.yaml` in CWD; WHEN F1 runs; THEN it parses the file, prints summary, calls skilladvisor placeholder, and returns `Result{Status: success}`.
- **Missing file**: GIVEN no `handoff.yaml` in CWD; WHEN F1 runs; THEN it returns an error wrapping "handoff.yaml not found".
- **Invalid YAML**: GIVEN a malformed `handoff.yaml`; WHEN F1 runs; THEN `handoff.Parse()` returns an error and F1 returns failure.
</details>

### Requirement: F2–F4 Stubs

Phases F2, F3, and F4 MUST print a phase banner (e.g., "=== Phase F2: Design ==="), print "not yet implemented", and return success. They MUST accept the scheduler Config but MUST NOT execute any real pipeline logic.

<details>
<summary>Scenarios</summary>

- **Stub execution**: GIVEN the scheduler runs past F1; WHEN it reaches F2; THEN the banner is printed, "not yet implemented" is displayed, and the phase reports success.
</details>
