# Tasks: ZyroAgentCLI — Arquitectura Completa MVP

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~2000 (across 6 PRs) |
| 400-line budget risk | High (mitigated: 6 PRs ≤400 each) |
| Chained PRs recommended | Yes |
| Chain strategy | stacked-to-main |
| Delivery strategy | ask-on-risk |

## PR1: Foundation — Skill Advisor + Context MCP Bridge **(✓ COMPLETED)**

### Phase 1.1: Skill Advisor Core

- [x] 1.1.1 Create `internal/skilladvisor/types.go` — Skill, ScoreResult, Layer interfaces
- [x] 1.1.2 Create `internal/skilladvisor/score.go` — ScoreSkillWeighted() weighted scoring (language 10, framework 20, project_type 30, verified 50, socket 15)
- [x] 1.1.3 Create `internal/skilladvisor/discover.go` — DiscoverClient with HTTP cache + TTL (1h default)
- [x] 1.1.4 Create `internal/skilladvisor/registry.go` — Load() reads YAML files from directory

### Phase 1.2: Context MCP Bridge

- [x] 1.2.1 Create `internal/context/types.go` — Bridge interface, QueryResult, LibraryID types
- [x] 1.2.2 Create `internal/context/bridge.go` — Bridge.Start() spawns `context serve --libs` via os/exec
- [x] 1.2.3 Implement Bridge.QueryDocs() — sends JSON-RPC request, parses response
- [x] 1.2.4 Implement Bridge.ResolveLibraryID() — resolves package name to Context7 library ID
- [x] 1.2.5 Implement Bridge.Stop() — SIGTERM → 5s grace → SIGKILL fallback

### Phase 1.3: Tests

- [x] 1.3.1 Create `internal/skilladvisor/skilladvisor_test.go` — 24 tests (ScoreSkill, weights, registry, cache)
- [x] 1.3.2 Create `internal/context/bridge_test.go` — 11 tests (JSON-RPC, pipe mock, lifecycle)
- [x] 1.3.3 `go test ./internal/skilladvisor/... ./internal/context/...` — ALL PASS

### Phase 1.4: Specs

- [x] 1.4.1 Update `openspec/specs/skill-advisor/spec.md` — reflect weighted scoring design
- [x] 1.4.2 Update `openspec/specs/context-mcp-bridge/spec.md` — reflect JSON-RPC protocol
- [x] 1.4.3 Create delta specs in `openspec/changes/zyro-architecture-mvp/specs/`

---

## PR2: CIO DSL Compile + Scheduler Harness (~400 lines) **(✓ COMPLETED)**

### Phase 2.1: CIO DSL Compile

- [x] 2.1.1 Add `EngramEntry` type + `ToMarkdown()` to existing `cio.go` — CIO now serializes as markdown for Engram (EngramKey type lives in `compile.go` as `EngramEntry`)
- [x] 2.1.2 Rewrite `internal/spec/compile.go` — Compile() walks CIO struct, emits Engram topic keys per phase (format: `sdd/{change}/cio-{component}`)
- [x] 2.1.3 Skipped per design decision — Design Decision 3 explicitly rejects OpenAPI/protobuf: "C-I-O as documentation (Structure without complexity)"
- [x] 2.1.4 Create `internal/spec/compile_test.go` — 8 tests (nil CIO, full CIO, empty CIO, zero-value safety, stable topic key, markdown serialization all-sections, zero-value no-panic)

### Phase 2.2: Scheduler Harness Refactor

- [x] 2.2.1 Refactor `internal/scheduler/scheduler.go` — add `HarnessValidator` type with `ValidateTransition()`, `CurrentPhase()`, `NextPhase()`, `SetCurrent()`; integrate into `Run()` with approval gates
- [x] 2.2.2 Rewrite `internal/scheduler/approval.go` — replace `PromptApproval` with `GuidedApproval` type (structured dialog: resumen + recomendación + riesgo + "s/n/d")
- [x] 2.2.3 Implement approval "d" (detail) mode — show full agent output and re-prompt
- [x] 2.2.4 Create `internal/scheduler/macro_runner.go` — `MacroPhaseRunner` type with agent function + optional validator; F1-F4 real implementations (F1: skilladvisor, F2: CIO compile, F3: governance summary, F4: delivery summary)
- [x] 2.2.5 Implement transition blocking via `HarnessValidator.ValidateTransition()` — blocks with error when `approved=false`

## PR3: Apply Runner + Contract Testing (~300 lines) **(✓ COMPLETED)**

### Phase 3.1: Apply Runner

- [x] 3.1.1 Create `internal/apply/types.go` — Task, Result, PoolConfig types
- [x] 3.1.2 Create `internal/apply/runner.go` — TaskRunner with goroutine pool (N workers, buffered channel)
- [x] 3.1.3 Implement Run(tasks) — fan-out tasks to pool, collect Results via channel
- [x] 3.1.4 Implement timeout per task — configurable via PoolConfig.TaskTimeout

### Phase 3.2: Contract Testing

- [x] 3.2.1 Create `internal/test/contracts.go` — ContractExecutor runs given/when/then against fixtures
- [x] 3.2.2 Create `internal/test/report.go` — GraphifyDiff compares contract results with previous graph state
- [x] 3.2.3 Implement diff threshold — only flag if >5 structural changes detected

### Phase 3.3: Tests

- [x] 3.3.1 Create `internal/apply/runner_test.go`
- [x] 3.3.2 Run `go test ./internal/apply/... ./internal/test/...` — all pass

## PR4: Investigation + Planning + SDD Wrappers **(✓ COMPLETED)**

### Phase 4.1: Investigation

- [x] PR4-TASK-001: Create `internal/investigation/research.go` — ResearchEngine that orchestrates Context MCP + GitMCP + web fetch concurrently (DocQueryFn, WebFetchFn, Git analysis via CLI)
- [x] PR4-TASK-002: Create `internal/investigation/advisor.go` — Advisor with stack recommendations (pro/contra), architectural patterns, skills-by-layer (excludes already-installed), MVP approach based on handoff boundaries
- [x] PR4-TASK-003: 22 tests for investigation package (ResearchEngine empty/doc/error/multi-concurrent, Report.Markdown/HasData, Advisor GoCLI/empty/installed-skills/noMVP, Advisory.Markdown)
- [x] PR4-TASK-004: Delta spec investigation — `openspec/changes/zyro-architecture-mvp/specs/investigation/spec.md` with R-INV-004/005/006, deprecates old doc path

### Phase 4.2: Planning

- [x] PR4-TASK-005: Create `internal/planning/decomposer.go` — Decomposer breaks down user_stories → atomic Feature values (handles bullet lists, numbered lists, "and" conjunctions, single sentences)
- [x] PR4-TASK-006: Create `internal/planning/scheduler.go` — Scheduler with Kahn's algorithm topological sort, phase grouping, priority assignment, circular dependency detection
- [x] PR4-TASK-007: 21 tests for planning package (Decomposer empty/single/bullet/numbered/and/cleaning, Scheduler empty/single/topological/parallel/unknown-dep/phase-limit, ValidateNoCircularDeps, Schedule.Markdown)
- [x] PR4-TASK-008: Delta spec planning — `openspec/changes/zyro-architecture-mvp/specs/planning/spec.md` with R-PLN-004/005/006/007

### Phase 4.3: zyro-sdd Wrappers

- [x] PR4-TASK-009: Create skill `~/.config/opencode/skills/zyro-sdd-explore/SKILL.md` — wrapper that injects topic keys from .zyro/conventions.yaml + Engram search protocol + standard entry format, then delegates to sdd-explore
- [x] PR4-TASK-010: Create skill `~/.config/opencode/skills/zyro-sdd-propose/SKILL.md` — wrapper with pre-flight prerequisite check (explore must exist), topic key injection, Engram format, then delegates to sdd-propose
- [x] PR4-TASK-011: Integration tests for wrapper pattern — `internal/investigation/integration_test.go` simulates full research→advise pipeline, partial failure recovery, and the engram entry formatting that the SKILL.md wrappers produce
- [x] PR4-TASK-012: Create `.zyro/conventions.yaml` — topic keys registry (project/change/graph scopes), standard entry format, search protocol (fast/slow/fallback/last-resort), graphify config, code/doc/review conventions

### Verification
- [x] `go test ./...` — all 11 packages pass (37+ tests across investigation/planning)
- [x] `go vet ./...` — no issues
- [x] `go build ./...` — binary builds

## PR5: Delivery + Doc Tools (~350 lines) **(✓ COMPLETED)**

### doc-tools (internal/doc/)

- [x] PR5-TASK-001: Crear `internal/doc/index.go` — GenerateIndex() lee conventions.yaml + descubre cambios activos, genera .zyro/doc-index.yaml
- [x] PR5-TASK-002: Crear `internal/doc/search.go` — SearchIndex() con protocolo: topic_key exacta → query texto → filtro tipo/change → sin resultados
- [x] PR5-TASK-003: Crear `internal/doc/sync.go` — Sync() orquesta: GenerateIndex → SaveIndex → Export → UpdateGraph
- [x] PR5-TASK-004: Crear `internal/doc/graphify.go` — UpdateGraph() compara conteo de entradas con estado previo, persiste solo si diff ≥5
- [x] PR5-TASK-005: Crear templates embebidos `ARCHITECTURE.md.tmpl` y `CHANGELOG.md.tmpl` — renderizados con `//go:embed`
- [x] PR5-TASK-006: 23 tests para doc-tools (GenerateIndex, SaveIndex/LoadIndex, SearchIndex, Sync, UpdateGraph, Export)
- [x] PR5-TASK-007: Delta spec doc-tools — `openspec/changes/zyro-architecture-mvp/specs/doc-tools/spec.md` con R-DOC-004 a R-DOC-009 + 9 scenarios

### CLI

- [x] PR5-TASK-008: Crear `cmd/zyrocli/doc.go` — comando `zyrocli doc sync` con flag `--dir`

### Wrappers zyro-sdd-* restantes

- [x] PR5-TASK-009: Crear skill `zyro-sdd-spec` (SKILL.md) — prerequisite: proposal exists, inyecta topic keys + search protocol + delta spec conventions
- [x] PR5-TASK-010: Crear skill `zyro-sdd-design` (SKILL.md) — prerequisite: spec exists, inyecta topic keys + design conventions
- [x] PR5-TASK-011: Crear skill `zyro-sdd-tasks` (SKILL.md) — prerequisite: design exists, inyecta topic keys + task planning conventions + workload forecast
- [x] PR5-TASK-012: Crear skill `zyro-sdd-implement` (SKILL.md) — prerequisite: tasks exists, inyecta topic keys + apply conventions + doc sync post-flight

### Integration

- [x] F4AgentFunc actualizado — ahora ejecuta doc.Sync() en lugar de resumen placeholder
- [x] `go test ./...` — all 12 packages pass
- [x] `go build ./...` — binary builds

## PR6: Final Adjustments (~250 lines) — **(✓ COMPLETED)**

### handoff adjustments:
- [x] PR6-TASK-001: Agregar campos `Capabilities` y `Dependencies` a handoff payload (para trazabilidad C-I-O)
- [x] PR6-TASK-002: Actualizar parser.go para parsear los nuevos campos
- [x] PR6-TASK-003: Tests + delta spec handoff-parser

### scaffold adjustments:
- [x] PR6-TASK-004: Actualizar scaffold para generar `.zyro/conventions.yaml` en proyectos nuevos
- [x] PR6-TASK-005: Actualizar scaffold templates (AGENT.md refleja nueva arquitectura)
- [x] PR6-TASK-006: Tests + delta spec project-scaffold

### run.go adjustments:
- [x] PR6-TASK-007: Asegurar que `zyrocli run` integra doc.Sync() post-scaffold
- [x] PR6-TASK-008: Actualizar mensajes y help text
- [x] PR6-TASK-009: Tests + delta spec zyrocli-run

### final integration:
- [x] PR6-TASK-010: Crear `zyro-sdd-verify` + `zyro-sdd-archive` wrappers (skills)
- [x] PR6-TASK-011: Full integration test: `go test ./...` debe pasar completo
