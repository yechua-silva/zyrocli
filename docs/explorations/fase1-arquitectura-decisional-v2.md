# Exploration: ZyroAgentCLI Fase 1 — Arquitectura Decisional v2

> 2026-06-14 · SDD Explore Phase · Engram topic: `explore/zyroagentcli-fase1`

---

## Current State

### TUI (cmd/zyrocli/profile_tui.go — 340 loc)

- **Single-screen flat list**: 7 SDD phases (`sddPhases`) vs 5 hardcoded models (`modelPool`).
- **model struct**: holds `assignments []int` (index into modelPool per phase).
- **Navigation**: up/down moves cursor; left/right/tab cycles model for selected phase.
- **Output**: `Profile` struct `{Project, Models map[string]string, UpdatedAt}` → `~/.config/opencode/profiles/<project>.json`.
- **modelPool** is a flat `[]string` — NO provider grouping, NO abstraction boundary.

### CLI Registration (cmd/zyrocli/profile.go — 42 loc)

- `profile` parent → shows help. `profile tui` child → calls `runProfileTUI()`.
- Registered in `init()` via `rootCmd.AddCommand(profileCmd)` → `profileCmd.AddCommand(profileTUICmd)`.

### Tests (cmd/zyrocli/profile_tui_test.go — 549 loc)

- 19 tests covering: model lifecycle, cursor movement, model cycling, quit/confirm, window size, resolveProjectName, buildProfile, writeProfile, loadExistingProfile, view rendering.
- ALL reference old `sddPhases`, `modelPool`, and `Profile` struct — must be rewritten.

### go.mod

- `go 1.26.3`, `cobra v1.10.2`, `yaml.v3`, `bubbletea v1.3.10`, `lipgloss v1.1.0`.
- **No new dependencies needed** for Phase 1.

### internal/ (11 subdirs)

```
internal/
├── apply/          ← runner.go, types.go
├── context/        ← bridge.go, types.go
├── doc/            ← sync.go, search.go, index.go, graphify.go, export.go
├── handoff/        ← parser.go, payload.go, validate.go
├── investigation/  ← research.go, advisor.go
├── planning/       ← decomposer.go, scheduler.go
├── scaffold/       ← renderer.go, writer.go, scripts.go, state.go, scaffold.go
├── scheduler/      ← phase_stubs.go, macro_runner.go, config.go, phase.go, scheduler.go, approval.go
├── skilladvisor/   ← discover.go, pipeline.go, verify.go, query.go, types.go, score.go, registry.go
├── spec/           ← compile.go, cio.go
└── test/           ← report.go, contracts.go
```

- `internal/opencode/` does NOT exist yet — must be created.

### openspec/specs/ (14 specs)

- NO spec about "profile", "opencode", or "model selection".
- Relevant: `scheduler-engine/spec.md` defines 4-phase DAG (F1→F2→F3→F4) with HarnessValidator — aligns with new phase naming.
- `planning/spec.md` defines Decomposer + Scheduler — context for phase renaming.

### AGENT.md

- Line 126: `**Engram MCP**: mem_save y mem_search via MCP client para persistencia.` — not relevant to Phase 1.

### docs/plan-fase1.md (102 loc)

- The guiding document. Specifies: 2-step TUI, writes to `~/.config/opencode/opencode.json`, format uses `agents` section with `sdd-orchestrator-{profile}` and `sdd-apply-{profile}` entries.

---

## Affected Areas

| File | Action | Why |
|------|--------|-----|
| `cmd/zyrocli/profile_tui.go` | **Replace entirely** | New 2-step bubbletea model (provider→model), remove flat modelPool/sddPhases/Profile |
| `cmd/zyrocli/profile_tui_test.go` | **Replace entirely** | All tests reference old types — rewrite for 2-step flow |
| `cmd/zyrocli/profile.go` | **Modify** | Update help text, phase references. Command tree stays same. |
| `internal/opencode/models.go` | **Create new** | Curated `[]Provider` with `Name`, `Models []Model` — no HTTP, pure data |
| `internal/opencode/opencode.go` | **Create new** | `ReadConfig()`, `WriteConfig()` for `~/.config/opencode/opencode.json` |
| `go.mod` | **Unchanged** | All deps already present |

---

## Approaches

### 1. Refactor in place, extract data to internal/opencode/ ✅ RECOMMENDED

Keep TUI entry point in `cmd/zyrocli/` but refactor model to use `internal/opencode` types.

- **Pros**: Clean separation (data in `internal/`, UI in `cmd/`), testable opencode I/O independently, minimal profile.go changes, TUI remains where bubbletea deps are expected.
- **Cons**: TUI still in `package main` (not reusable), profile.go needs moderate rewrite.
- **Effort**: Medium (~350 lines new/changed across 5 files).

### 2. Move entire TUI to internal/opencode/

Move bubbletea model/view/update into `internal/opencode/tui.go`. `cmd/zyrocli/profile.go` becomes thin wrapper.

- **Pros**: Reusable TUI, cleaner `cmd/` layer.
- **Cons**: bubbletea + lipgloss become transitive deps of `internal/`, unusual for internal package.
- **Effort**: Medium-High (~450 lines, moving code + refactoring).

### 3. Separate provider screen via sub-commands ❌ REJECTED

`zyrocli profile provider` + `zyrocli profile model` as separate commands.

- **Pros**: Simpler per-command logic.
- **Cons**: Terrible UX (multiple commands), defeats interactive TUI purpose.
- **Effort**: Low. **Rejected outright.**

---

## Recommendation

**Approach 1 — Refactor in place, extract data to internal/opencode/.**

Rationale:
1. TUI belongs in `cmd/zyrocli/` — bubbletea is UI framework, not domain concern.
2. `internal/opencode/models.go` is pure-data (no HTTP, no IO beyond opencode.go) — easy to unit test.
3. `internal/opencode/opencode.go` handles config read/write — easy to unit test with temp files.
4. `profile_tui.go` becomes orchestrator: two screens (provider select → model select) per phase, then summary confirmation, then write.
5. `profile.go` needs only help text updates — command tree (`profile tui`) stays unchanged.

---

## Risks

1. **Phase count reduction (7→4) breaks existing profiles**: Old `profiles/*.json` files become incompatible. Mitigation: detect old directory, migrate or warn on first run.
2. **opencode.json overwrite**: Multiple users could overwrite shared config. Mitigation: `sdd-orchestrator-{profile}` naming with project-specific profile keys provides isolation.
3. **Provider list becomes stale**: Hardcoded model list in `internal/opencode/models.go` goes out of date. Mitigation: intentional (plan says "NO HTTP calls"). Future `zyrocli sync` command.
4. **sddPhases variable used in 18 locations** (2 files): Every reference must be ported to new 4-phase, 2-step model. Mechanical but must be thorough.

---

## Files to Create vs Modify

### New Files (~200 lines)

| File | Lines (est) | Content |
|------|-------------|---------|
| `internal/opencode/models.go` | ~80 | `Provider{Name, Models[]}`, `Model{Name, ID}`, curated list |
| `internal/opencode/opencode.go` | ~100 | `Config` struct, `Read()`, `Write()`, merge logic |
| `internal/opencode/models_test.go` | ~20 | Basic validation tests |

### Modified Files (~150 lines changed)

| File | Lines changed (est) | What changes |
|------|---------------------|--------------|
| `cmd/zyrocli/profile_tui.go` | ~340 (replace) | New 2-step model, remove flat pool |
| `cmd/zyrocli/profile_tui_test.go` | ~549 (replace) | All tests rewritten |
| `cmd/zyrocli/profile.go` | ~10 | Help text updates |

### Total: ~350 lines new, ~900 lines replaced

---

## Ready for Proposal

**Yes** — all code is read, plan exists in `docs/plan-fase1.md`, no blocking dependencies.
