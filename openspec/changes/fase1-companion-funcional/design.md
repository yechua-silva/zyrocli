# Design: Fase 1 — Companion Funcional

## Technical Approach

Refactor `zyrocli profile tui` from single-pool model cycling to a **2-step per-phase flow** (provider → model). Create `internal/opencode/` package for curated providers + `opencode.json` read/write. TUI iterates over 4 Zyro-SDD phases, each producing a `phaseAssignment`. On confirm, writes `agent` section to `~/.config/opencode/opencode.json`.

## Architecture Decisions

### Decision: Two-step TUI vs single-pool cycling

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Single pool (current) | Simple, but flat model list doesn't reflect provider structure | Rejected |
| Two-step (provider → model) | Matches real OpenCode config, requires state machine | **Chosen** |

**Rationale**: OpenCode's `opencode.json` requires `provider/model` format. A 2-step TUI naturally produces this compound identifier.

### Decision: Curated providers vs HTTP discovery

| Option | Tradeoff | Decision |
|--------|----------|----------|
| HTTP API calls | Always current, requires network, complex error handling | Rejected |
| Curated list in `models.go` | Offline, fast, predictable. Needs manual updates | **Chosen** |

**Rationale**: Proposal explicitly scopes out HTTP calls. Curated list covers known providers (deepseek, mimo, nvidia, anthropic, openai). Unknown providers from `opencode.json` are merged in at runtime.

### Decision: Read providers from opencode.json

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Hardcoded only | Simple, but ignores user-configured providers | Rejected |
| Merge curated + opencode.json | Full coverage, respects user config | **Chosen** |

**Rationale**: `ReadProviders()` reads `providers` section from `opencode.json`, then merges with `KnownProviders()` so user-added providers appear in TUI.

### Decision: Write to `agent` section (not `agents`)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `agents` (plural) | Current plan-fase1.md says this | Rejected |
| `agent` (singular) | Matches proposal spec and output format | **Chosen** |

**Rationale**: Proposal defines output as `"agent": {...}`. The plan-fase1.md uses `agents` but the proposal's concrete JSON example uses `agent`. Follow proposal.

## Data Flow

```
TUI init
  │
  ├─ phaseIdx=0 (zyro-sdd-explorer-stack)
  │   ├─ stateSelectProvider ──→ user picks provider
  │   ├─ stateSelectModel    ──→ user picks model
  │   └─ append assignment   ──→ phaseIdx++
  │
  ├─ phaseIdx=1 (zyro-sdd-planning)
  │   └─ ... (same cycle)
  │
  ├─ phaseIdx=2 (zyro-sdd-implement)
  │   └─ ...
  │
  ├─ phaseIdx=3 (zyro-sdd-verify)
  │   └─ ...
  │
  └─ stateSummary ──→ user confirms
       └─ WriteAgentConfig() ──→ opencode.json { "agent": {...} }
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/opencode/models.go` | Create | Provider/Model structs + `KnownProviders()` curated list |
| `internal/opencode/opencode.go` | Create | `ReadProviders()`, `WriteAgentConfig()`, `ResolveConfigPath()` |
| `internal/opencode/opencode_test.go` | Create | Unit tests for read/write/known |
| `cmd/zyrocli/profile_tui.go` | Replace | New 2-step bubbletea model with state machine |
| `cmd/zyrocli/profile_tui_test.go` | Replace | Tests for state transitions, navigation, selection |
| `cmd/zyrocli/profile.go` | Modify | Update help text for 2-step flow |

## Interfaces / Contracts

```go
// internal/opencode/models.go
type Provider struct {
    ID     string  `json:"id"`
    Name   string  `json:"name"`
    Models []Model `json:"models"`
}
type Model struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
func KnownProviders() []Provider

// internal/opencode/opencode.go
type AgentConfig struct {
    Model          string `json:"model"`
    Mode           string `json:"mode"`
    ReasoningEffort string `json:"reasoningEffort,omitempty"`
}
type OpenCodeConfig struct {
    Agent map[string]AgentConfig `json:"agent"`
}
func ReadProviders(path string) ([]Provider, error)
func WriteAgentConfig(path, profile string, configs map[string]AgentConfig) error
func ResolveConfigPath() string

// cmd/zyrocli/profile_tui.go — TUI state machine
type tuiState int
const (
    stateSelectProvider tuiState = iota
    stateSelectModel
    stateSummary
)
type phaseAssignment struct {
    Phase      string
    ProviderID string
    ModelID    string
}
type profileTuiModel struct {
    state       tuiState
    phases      []string
    phaseIdx    int
    providers   []opencode.Provider
    providerIdx int
    modelIdx    int
    assignments []phaseAssignment
    done, cancelled bool
    err         error
}
```

## Fases Zyro-SDD → OpenCode Agent Mapping

| Phase | Agent Name | Mode |
|-------|-----------|------|
| zyro-sdd-explorer-stack | sdd-explore-zyro | subagent |
| zyro-sdd-planning | sdd-orchestrator-zyro | primary |
| zyro-sdd-implement | sdd-implement-zyro | subagent |
| zyro-sdd-verify | sdd-verify-zyro | subagent |

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | `KnownProviders()` not empty, correct structure | Direct assertion |
| Unit | `ReadProviders()` with temp file, missing file, empty providers | Table-driven, `t.TempDir()` |
| Unit | `WriteAgentConfig()` roundtrip: write → read → verify | Temp dir, JSON unmarshal |
| Unit | TUI state transitions: provider→model→next phase | `tea.KeyMsg` simulation |
| Unit | TUI cursor bounds, quit/confirm keys | Key msg injection |
| Integration | Full TUI flow with mocked providers | `tea.NewProgram` + program.Send |

## Migration / Rollout

No migration required. Existing `profiles/` directory format is maintained as fallback via `writeProfile()`. New TUI writes to `opencode.json` in addition.

## Open Questions

- [ ] Should `ReadProviders` merge curated + file providers, or only read from file?
- [ ] What happens if `opencode.json` doesn't exist yet — create it or error?
- [ ] Should `reasoningEffort` be configurable per phase in TUI, or default to "high"?
