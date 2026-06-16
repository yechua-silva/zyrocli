# Tasks: Fase 1 — Companion Funcional

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 750–1000 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 → PR 2 → PR 3 |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | `internal/opencode/` package (models + reader/writer + tests) | PR 1 | base: main; ~270 lines; self-contained |
| 2 | TUI rewrite (2-step state machine + tests) | PR 2 | base: main after PR 1; ~700 lines; core deliverable |
| 3 | profile.go wiring + integration verification | PR 3 | base: main after PR 2; ~30 lines; final integration |

## Phase 1: Foundation — `internal/opencode/` Package

- [x] 1.1 Create `internal/opencode/models.go`: Define `Provider` (ID, Name, Models) and `Model` (ID, Name) structs. Implement `KnownProviders()` returning curated map with 8 providers: opencode-go, opencode, google, groq, openrouter, cerebras, nvidia (empty models), anthropic. Match spec providers exactly.
- [x] 1.2 Create `internal/opencode/opencode.go`: Implement `ResolveConfigPath()` returning `~/.config/opencode/opencode.json`. Implement `ReadProviders(path)`: parse `providers` section, merge with `KnownProviders()` (JSON overrides curated by ID). Missing file returns curated list only.
- [x] 1.3 Add to `internal/opencode/opencode.go`: Implement `ReadAgentConfigs(path)` returning `map[string]AgentConfig` from `agent` section. Implement `WriteAgentConfig(path, profileName, configs)`: read existing JSON, overwrite `agent.{profileName}`, preserve all other keys. Create file if missing (safe default behaviour).
- [x] 1.4 Create `internal/opencode/opencode_test.go`: 14 tests covering KnownProviders, ReadProviders (missing file, valid file, provider override, no providers section), WriteAgentConfig (creates file, preserves sections, format, file not exist), ReadAgentConfigs, and path resolution. Uses `t.TempDir()` for all file ops.

## Phase 2: TUI Rewrite — 2-Step State Machine

- [ ] 2.1 Rewrite `cmd/zyrocli/profile_tui.go`: Remove `sddPhases` (7 phases), `modelPool`, `Profile` struct, `handoffProject`, `buildProfile`, `writeProfile`, `loadExistingProfile`, `profilePath`, `profileDir`, `ensureProfileDir`. Keep only `resolveProjectName()` and lipgloss styles (update colors if needed).
- [ ] 2.2 Define new types in `profile_tui.go`: `tuiState` enum (stateSelectProvider, stateSelectModel, stateSummary), `phaseAssignment` struct (Phase, ProviderID, ModelID), `profileTuiModel` struct with state machine fields (state, phases, phaseIdx, providers, providerIdx, modelIdx, assignments, done, cancelled, err).
- [ ] 2.3 Implement TUI `Update()` with state machine: stateSelectProvider — ↑/↓ navigate providers, Enter selects → stateSelectModel, Esc → quit. stateSelectModel — ↑/↓ navigate models, Enter selects → append assignment + next phase (or stateSummary if last), Esc → back to stateSelectModel. stateSummary — Enter confirms, Esc cancels. Use `internal/opencode.ReadProviders(opencode.ResolveConfigPath())` for provider list.
- [ ] 2.4 Implement TUI `View()`: Render title "zyro-model — Asignar modelos por fase SDD". stateSelectProvider: show current phase name + provider list with cursor. stateSelectModel: show selected provider + model list. stateSummary: table with Phase | Provider | Model for all 4 assignments. Show help keys per state.
- [ ] 2.5 Update `runProfileTUI()`: Call `internal/opencode.ReadProviders()` to populate model. On confirm, build `map[string]AgentConfig` from assignments using `sdd-{role}-zyro` naming pattern. Call `internal/opencode.WriteAgentConfig(path, "zyro", configs)`. Keep `writeProfile()` as fallback for backward compat with `profiles/` directory.
- [ ] 2.6 Rewrite `cmd/zyrocli/profile_tui_test.go`: Test `newModel` initializes correctly with providers from `KnownProviders`. Test state transitions: provider→model→next phase→summary. Test cursor bounds (does not exceed provider/model count). Test quit/confirm keys. Test Esc in stateSelectModel returns to stateSelectProvider. Test `View()` renders phase names, provider names, model names. Use `tea.KeyMsg` simulation. Target 15–20 test functions.

## Phase 3: Integration — Wiring + Cleanup

- [ ] 3.1 Update `cmd/zyrocli/profile.go`: Change help text to describe 2-step flow (provider → model per phase). Update `Long` description: mention output goes to `opencode.json` section `agent`. Remove references to `profiles/` directory.
- [ ] 3.2 Verify end-to-end: `go build ./...` compiles. `go test ./internal/opencode/...` passes. `go test ./cmd/zyrocli/...` passes. `go vet ./...` clean.
- [ ] 3.3 Remove dead code: delete unused `Profile` struct, `modelPool`, `sddPhases` (7-phase list), `buildProfile`, `loadExistingProfile`, `profileDir`, `ensureProfileDir` if no longer referenced. Keep `resolveProjectName()`.
