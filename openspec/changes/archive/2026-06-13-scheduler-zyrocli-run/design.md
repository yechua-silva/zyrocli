# Design: scheduler-zyrocli-run

## Technical Approach

Sequential DAG executor with a `PhaseRunner` interface. Each phase is a struct implementing `Run(ctx, *Config) (*Result, error)`. The `Scheduler` walks phases in order (F1→F2→F3→F4), calling approval prompts between phases unless `--auto` is set. Config is loaded from `handoff.yaml` governance/limits sections via `handoff.Parse()`. All new code — no migration.

## Architecture Decisions

| Decision | Option A | Option B (chosen) | Tradeoff |
|----------|----------|-------------------|----------|
| Phase abstraction | `func(ctx, *Config) (*Result, error)` | `PhaseRunner` interface | Interface enables mocking and stateful phases; slightly more boilerplate |
| Phase identity | `int` enum | `string` constants (`PhaseF1`, etc.) | Strings are readable in logs; int is faster but opaque |
| Approval channel | Channel-based async | `bufio.Scanner` on stdin (sync) | Sync is simpler; async adds complexity unnecessary for stubs |
| Config source | Separate config file | Reuse `handoff.yaml` governance+limits | No new file to maintain; handoff is already the source of truth |
| Retry mechanism | Goroutine pool | Simple for-loop with counter | For-loop matches sequential DAG; goroutines premature for stubs |

## Data Flow

```
zyrocli run [--auto] [--phase F2]
  │
  ├─ handoff.Parse("handoff.yaml") → *Payload
  ├─ ConfigFromPayload(payload) → *Config
  ├─ NewScheduler(config, runners) → *Scheduler
  │
  └─ Scheduler.Run(ctx) or RunPhase(ctx, F2)
       │
       ├─ F1.Run(ctx, config) → *Result
       │    └─ handoff.Parse() + print summary + skilladvisor placeholder
       ├─ PromptApproval("F1", result) → bool  (skipped if --auto)
       │
       ├─ F2.Run(ctx, config) → *Result   (stub: banner + "not yet implemented")
       ├─ PromptApproval(...) → bool
       │
       ├─ F3.Run(...) → ... → F4.Run(...)
       │
       └─ []Result → print summary
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/scheduler/phase.go` | Create | `Phase` type, `PhaseRunner` interface, `Config`, `Result` structs |
| `internal/scheduler/scheduler.go` | Rewrite | `Scheduler` struct, `Run()`, `RunPhase()` — replaces current stub |
| `internal/scheduler/phase_stubs.go` | Create | `F1Runner`, `F2Runner`, `F3Runner`, `F4Runner` implementations |
| `internal/scheduler/approval.go` | Create | `PromptApproval(name, result, reader) (bool, error)` helper |
| `internal/scheduler/config.go` | Create | `ConfigFromPayload(*handoff.Payload) *Config` mapper |
| `internal/scheduler/scheduler_test.go` | Create | State machine transitions, approval flow, timeout tests |
| `cmd/zyrocli/run.go` | Create | Cobra `zyrocli run` subcommand with `--auto`, `--phase` flags |
| `cmd/zyrocli/run_test.go` | Create | CLI flag parsing, integration with scheduler |

## Interfaces / Contracts

```go
// phase.go
type Phase string

const (
    PhaseF1 Phase = "F1"
    PhaseF2 Phase = "F2"
    PhaseF3 Phase = "F3"
    PhaseF4 Phase = "F4"
)

type PhaseRunner interface {
    Run(ctx context.Context, cfg *Config) (*Result, error)
    Name() Phase
}

type Result struct {
    Phase  Phase
    Status ResultStatus // "success" | "failure" | "skipped"
    Output string       // human-readable summary
    Error  error        // non-nil when Status == "failure"
}

type Config struct {
    MaxTasks   int
    MaxLines   int
    ChainedPRs bool
    Approvals  []handoff.ApprovalPoint
    MaxLoops   int           // default 1 (no retry)
    Timeout    time.Duration // default 0 (no timeout)
}
```

```go
// scheduler.go
type Scheduler struct {
    phases  []PhaseRunner
    config  *Config
    auto    bool
}

func NewScheduler(cfg *Config, runners []PhaseRunner, auto bool) *Scheduler
func (s *Scheduler) Run(ctx context.Context) ([]Result, error)
func (s *Scheduler) RunPhase(ctx context.Context, phase Phase) (*Result, error)
```

```go
// approval.go
func PromptApproval(phase Phase, result *Result, reader io.Reader) (bool, error)
// Reads "y" or "n" from reader. Retries on invalid input.
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Phase transitions (all succeed, abort on failure) | Table-driven tests with mock `PhaseRunner` returning success/failure |
| Unit | Approval flow (approve, reject, invalid retry) | `PromptApproval` with `strings.NewReader` — no real stdin |
| Unit | ConfigFromPayload mapping + defaults | Table-driven: full payload, minimal payload, zero-values |
| Unit | RunPhase single-phase isolation | Mock runner, verify only requested phase executes |
| Unit | Timeout context cancellation | `context.WithTimeout` + slow mock runner |
| Integration | CLI flag parsing (`--auto`, `--phase`) | Cobra test harness like `init_test.go` pattern |
| Integration | Full F1→F4 sequence with stubs | Real `handoff.Parse()` + scheduler, verify output |

## Migration / Rollout

No migration required. All new code. The existing `internal/scheduler/scheduler.go` stub is replaced entirely — its only content is an unused `Scheduler` struct with a TODO comment.

## Open Questions

- [ ] Should `max_loops` and `timeout` be added to `handoff.yaml` governance schema, or kept as scheduler-only defaults? (Proposal assumes defaults; spec requires them configurable.)
- [ ] Should `F1Runner` call `handoff.Parse()` independently, or receive the already-parsed `*Payload` from the CLI layer? (Design assumes CLI passes `*Payload` to avoid double-parse.)
