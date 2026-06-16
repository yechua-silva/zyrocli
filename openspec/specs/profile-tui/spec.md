# Profile TUI Specification

## Purpose

Interactive 2-step terminal UI (provider → model per phase) that replaces the old flat model-pool TUI. Writes selections to `~/.config/opencode/opencode.json` section `agent` in Zyro-SDD profile format.

## Requirements

### Requirement: 2-Step Provider→Model Selection

The TUI MUST present two steps per phase: (1) list providers from `internal/opencode.ReadProviders()`, user picks one; (2) list models for the selected provider, user picks one.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Step 1: provider list | TUI starts | View renders | Provider names are listed with ↑/↓ navigation |
| Step 2: model list | User presses Enter on a provider | View updates | Model names for that provider are shown |
| Go back | User is in step 2 (model list) and presses Esc | View updates | Returns to step 1 (provider list) |

### Requirement: Zyro-SDD Phase Names

The TUI MUST use Zyro-SDD phase names (replacing the 7 old SDD phases): `zyro-sdd-explorer-stack`, `zyro-sdd-planning`, `zyro-sdd-implement`, `zyro-sdd-verify`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| 4 phases shown | TUI is running | Counting visible phases | Exactly 4 phases are displayed |
| Phase naming convention | TUI renders | View output | Each phase name starts with "zyro-sdd-" |

### Requirement: Provider Source

The TUI MUST source providers from `internal/opencode.ReadProviders()` (curated + JSON merge). If the JSON file is missing, MUST fall back to curated list without error.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| JSON available | `opencode.json` has `providers` | Providers shown | JSON providers appear alongside curated |
| No JSON | File does not exist | Providers shown | Only `KnownProviders` are listed |
| Empty JSON | JSON has no `providers` key | Providers shown | Only `KnownProviders` are listed |

### Requirement: Unknown Provider Models

If a selected provider has an empty model list (e.g., `nvidia` with no static models), the TUI MUST show "unknown models — enter manually" and allow free-text input.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Empty model list | User selects `nvidia` (no curated models) | Model step renders | Shows "unknown models — enter manually" with text input |
| Manual entry accepted | User types a model name | User presses Enter | Custom model name is accepted as selection |

### Requirement: Confirm and Write

After all 4 phases are assigned, the TUI MUST show a summary table and wait for Enter to confirm. On confirm, MUST call `internal/opencode.WriteAgentConfig` with `profileName="zyro"` and entries named `sdd-{role}-zyro`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| Summary displayed | All 4 phases have provider/model pairs | TUI enters summary | Table shows phase, provider, and model for each |
| Confirm writes | User presses Enter on summary | `WriteAgentConfig` called | `agent.sdd-orchestrator-zyro` written to opencode.json |
| Cancel | User presses Esc on summary | TUI exits | No write occurs, returns message "Profile cancelled" |

### Requirement: Output Naming Convention

Written config entries MUST follow the pattern `sdd-{role}-zyro`: `sdd-orchestrator-zyro`, `sdd-explore-zyro`, `sdd-planning-zyro`, `sdd-implement-zyro`, `sdd-verify-zyro`.

| Scenario | GIVEN | WHEN | THEN |
|----------|-------|------|------|
| All entries written | User confirms | Inspecting `agent` section | Keys match `sdd-*-zyro` pattern |

## Removed Requirements

- Flat 1-step `modelPool` cycling with ←/→ keys — replaced by 2-step provider→model
- Output to `profiles/{project}.json` — replaced by `opencode.json` section `agent`
- 7 SDD phases (sdd-explore, sdd-onboard, sdd-spec, sdd-design, sdd-propose, sdd-apply, sdd-verify) — replaced by 4 Zyro-SDD phases
