# Proposal: scheduler-zyrocli-run

## Intent

Build the scheduler state machine backbone and `zyrocli run` command. Without these, `zyrocli` is a static parser — it cannot execute the 4-phase SDD pipeline (F1→F2→F3→F4) with human-validation gates as AGENT.md describes.

## Scope

### In Scope
- Scheduler DAG: 4 sequential nodes, `PhaseRunner` interface
- `zyrocli run` — interactive mode, `--auto` (skip gates), `--phase F2` (single phase)
- Phase stubs: F1 calls `handoff.Parse()`, F2–F4 print "not yet implemented"
- Approval points: human confirmation after each phase unless `--auto`
- Max loops per phase + governance config from `handoff.yaml`
- Unit tests: state machine transitions, approval flow, CLI flag parsing

### Out of Scope
- Real F1 logic (skill advisor, Python exploration) — future
- Real F2–F4 logic (CI-O spec, apply runner, archive) — future

## Capabilities

### New Capabilities
- `scheduler-engine`: Phase state machine DAG with approval gates, max-loop enforcement, governance-aware execution, and `zyrocli run` CLI

### Modified Capabilities
- None — `handoff-parser` unchanged (F1 stub consumes but doesn't alter it)

## Approach

1. **Scheduler**: ordered `[]PhaseRunner`, each with `Run(ctx, *Config) (*Result, error)`. Config from `handoff.yaml` governance/limits.
2. **Loop**: walk phases sequentially. Run → if gates required AND not `--auto` → prompt → proceed or abort.
3. **Approval**: `PromptApproval(name, result) (bool, error)` — stdin read, gated by `--auto`.
4. **F1 stub**: call `handoff.Parse()`, print summary, placeholder skilladvisor call.
5. **F2–F4 stubs**: phase banner + "not yet implemented".
6. **CLI**: Cobra subcommand on root. Flags: `--auto` (bool), `--phase` (string).

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/scheduler/scheduler.go` | Modified | Stub → DAG executor |
| `internal/scheduler/phase.go` | New | `PhaseRunner`, `Config`, `Result` |
| `internal/scheduler/scheduler_test.go` | New | State machine + approval tests |
| `cmd/zyrocli/run.go` | New | `zyrocli run` subcommand |
| `cmd/zyrocli/run_test.go` | New | CLI flag + integration tests |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Stdin prompt vs phase I/O conflict | Low | Dedicated approval channel |
| handoff.yaml missing from CWD | Medium | Clear error: "run 'zyrocli init <file>' first" |

## Rollback Plan

- **Bad state machine**: `git revert` — phases are stubs, no data loss
- **CLI conflict**: Remove `run.go`, revert `main.go`
- **Partial write**: No persistent state — scheduler is in-memory

## Dependencies

- `github.com/secko/zyrocli/internal/handoff` (existing)
- Stdlib only: `context`, `fmt`, `bufio`, `os`

## Success Criteria

- [ ] `zyrocli run` executes F1→pause→F2→F3→F4 in sequence
- [ ] `zyrocli run --phase F3` runs only F3 stub
- [ ] `zyrocli run --auto` runs all phases without prompting
- [ ] Missing handoff.yaml errors clearly
- [ ] `go test ./internal/scheduler/...` passes with 70%+ coverage
- [ ] `go build ./cmd/zyrocli/...` compiles clean
