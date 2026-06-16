# Tasks: Inicializar módulo Go y estructura base del proyecto

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 200-250 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Complete Go scaffold with all 17 files | PR 1 | Base branch; includes go.mod, cmd/, internal stubs, configs, verification |

## Phase 1: Foundation

- [x] 1.1 Run `go mod init github.com/secko/zyrocli` in project root
- [x] 1.2 Run `go get github.com/spf13/cobra gopkg.in/yaml.v3`
- [x] 1.3 Create `.gitignore` with Go standard patterns (bin/, vendor/, *.exe, .env, IDE files)

## Phase 2: Entry Point

- [x] 2.1 Create `cmd/zyrocli/main.go` with Cobra root command, `Use: "zyrocli"`, `--verbose` persistent flag, and `Execute()` function
- [x] 2.2 Ensure `main.go` imports only Cobra (no internal packages yet)

## Phase 3: Internal Stubs

- [x] 3.1 Create `internal/scheduler/scheduler.go` with exported `Scheduler` struct and TODO comment
- [x] 3.2 Create `internal/handoff/payload.go` with exported `Payload` struct and TODO comment
- [x] 3.3 Create `internal/skilladvisor/registry.go` with exported `Registry` struct and TODO comment
- [x] 3.4 Create `internal/skilladvisor/score.go` with exported `ScoreSkill()` function stub
- [x] 3.5 Create `internal/skilladvisor/discover.go` with exported `Discover()` function stub
- [x] 3.6 Create `internal/spec/cio.go` with exported `CIO` struct and TODO comment
- [x] 3.7 Create `internal/spec/compile.go` with exported `Compile()` function stub
- [x] 3.8 Create `internal/context/bridge.go` with exported `Bridge` struct and TODO comment
- [x] 3.9 Create `internal/apply/runner.go` with exported `Runner` struct and TODO comment
- [x] 3.10 Create `internal/test/contracts.go` with exported `ContractTest` struct
- [x] 3.11 Create `internal/test/report.go` with exported `Report` struct

## Phase 4: Configs

- [x] 4.1 Create `handoff.yaml` with example fields matching AGENT.md contract (version, source, project, phases, skills)
- [x] 4.2 Create `zyro-skill-overrides.yaml` with example overrides structure

## Phase 5: Verification

- [x] 5.1 Run `go vet ./...` and ensure zero diagnostics
- [x] 5.2 Run `go build ./...` and verify binary builds successfully
- [x] 5.3 Test `./zyrocli --help` shows root command usage
