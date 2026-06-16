# Tasks: scheduler-zyrocli-run

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–600 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (types+config) → PR 2 (scheduler+stubs) → PR 3 (CLI+tests) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Types, config, handoff schema | PR 1 | Foundation — phase.go, config.go, payload.go changes; tests included |
| 2 | Scheduler engine + phase stubs | PR 2 | Depends on PR 1; scheduler.go, approval.go, phase_stubs.go |
| 3 | CLI command + all tests | PR 3 | Depends on PR 2; run.go, run_test.go, scheduler_test.go |

## Phase 1: Schema & Types (Foundation)

- [x] 1.1 Add `MaxLoops int` and `PhaseTimeout string` to `internal/handoff/payload.go` Limits struct with yaml tags
- [x] 1.2 Create `internal/scheduler/phase.go` — Phase type (string constants F1–F4), PhaseRunner interface (`Run(ctx, *Config) (*Result, error)` + `Name() Phase`), Config, Result, ResultStatus types
- [x] 1.3 Create `internal/scheduler/config.go` — `LoadConfig(path string) (*Config, error)` reads handoff.yaml, maps Limits to Config, defaults: max_loops=5, phase_timeout=10m
- [x] 1.4 Add test cases in `internal/handoff/payload_test.go` for MaxLoops and PhaseTimeout parsing
  - [x] 1.4a Create `internal/scheduler/config_test.go` — valid config, defaults, custom values, invalid timeout, file not found

## Phase 2: Scheduler Engine (Core)

- [x] 2.1 Rewrite `internal/scheduler/scheduler.go` — Scheduler struct (phases, config, auto), New(), Run() sequential DAG, RunPhase() single-phase
- [x] 2.2 Create `internal/scheduler/approval.go` — `PromptApproval(phase Phase, summary string) (bool, error)` with bufio.NewReader, accepts y/yes/s/si/n/no
- [x] 2.3 Create `internal/scheduler/phase_stubs.go` — F1Runner calls handoff.Parse(), prints contract summary; F2–F4Runner print "not yet implemented"; each respects context.WithTimeout
- [x] 2.4 Implement abort logic: if status==FAIL → abort remaining; if !auto → PromptApproval → reject aborts

## Phase 3: CLI Integration (Wiring)

- [x] 3.1 Create `cmd/zyrocli/run.go` — cobra runCmd with `--auto` (bool) and `--phase` (string) flags
- [x] 3.2 Register runCmd as subcommand of rootCmd
- [x] 3.3 Error if handoff.yaml missing: "handoff.yaml not found — run 'zyrocli init <file>' first"
- [x] 3.4 Validate --phase flag: error "invalid phase: F5 (valid: F1, F2, F3, F4)" for bad values
- [x] 3.5 Handle --auto + --phase combination: single phase without prompting

## Phase 4: Testing (Verification)

- [x] 4.1 Create `internal/scheduler/scheduler_test.go` — all succeed, abort on failure, abort on runner error, phase timeout, timeout skips subsequent phases
- [x] 4.2 Test PromptApproval with strings.Reader: approve ("y", "yes", "s", "si"), reject ("n", "no"), invalid ("maybe", "xyz") retries
- [x] 4.3 Test auto mode: full run with auto=true passes without prompts
- [x] 4.4 Test RunPhase isolation: only requested phase executes; unknown phase returns error
- [x] 4.5 Create `cmd/zyrocli/run_test.go` — flag parsing (--auto, --phase, combination), missing handoff.yaml error
- [x] 4.6 Test --phase F5 produces descriptive error message
