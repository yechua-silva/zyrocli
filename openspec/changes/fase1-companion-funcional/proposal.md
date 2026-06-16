# Proposal: Fase 1 — Companion Funcional

## Intent

ZyroCLI debe funcionar como companion/configurador de OpenCode (como gentle-ai). El TUI actual usa un pool hardcoded de modelos. Necesita refactorizarse a flujo 2-pasos (provider → modelo) que lea providers desde `opencode.json` y escriba agent configs también allí.

## Scope

### In Scope
- Refactor `zyrocli profile tui` a selector 2-pasos por fase Zyro-SDD
- Crear `internal/opencode/` (lista curada providers + Reader/Writer de opencode.json)
- Output a `~/.config/opencode/opencode.json` sección `agent`
- Tests completos del nuevo package y TUI

### Out of Scope
- HTTP calls a APIs externas para descubrir modelos
- HelixDB integration (Fase 2)
- `zyrocli sync`, `zyrocli absorb`
- Modificar `main.go`, `internal/scheduler/`, `internal/handoff/`, `go.mod`

## Capabilities

### New Capabilities
- `opencode-config`: Package `internal/opencode/` con structs Provider/Model, lista curada, y Reader/Writer de `opencode.json`
- `profile-tui`: TUI bubbletea 2-pasos que selecciona provider → modelo para cada fase Zyro-SDD

### Modified Capabilities
None — no existing spec requirements change at behavioral level.

## Approach

1. `internal/opencode/models.go` define structs + `KnownProviders` map (curado, sin HTTP)
2. `internal/opencode/opencode.go`: `ReadProviders()` desde sección `providers` de opencode.json; `WriteAgentConfig()` escribe en sección `agent`
3. TUI: Paso 1 lista providers (leídos de opencode.json + curados); Paso 2 lista modelos del provider seleccionado; se repite por cada fase Zyro-SDD
4. Confirmación escribe `opencode.json` sección `agent` + mantiene perfiles en `profiles/` como fallback

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/opencode/models.go` | New | Structs + lista curada providers/modelos |
| `internal/opencode/opencode.go` | New | Reader/Writer de opencode.json |
| `internal/opencode/opencode_test.go` | New | Tests del package |
| `cmd/zyrocli/profile.go` | Modified | Help text, referencias nuevo flujo |
| `cmd/zyrocli/profile_tui.go` | Replaced | Nuevo modelo bubbletea 2-pasos |
| `cmd/zyrocli/profile_tui_test.go` | Replaced | Tests nuevo TUI |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Formato `agent` vs `agents` cambia en futura versión OpenCode | Low | Leer formato actual dinámicamente |
| Provider sin modelos curados en `models.go` | Med | Mostrar "unknown models" + permitir entrada manual |
| Ruptura perfiles existentes en `profiles/` | Low | Mantener `writeProfile()` como fallback |

## Rollback Plan

1. `git checkout HEAD -- cmd/zyrocli/profile.go cmd/zyrocli/profile_tui.go cmd/zyrocli/profile_tui_test.go`
2. `git rm -r internal/opencode/`
3. Verificar `go build ./...` y `go test ./...`

## Dependencies

- bubbletea v1.3.10 (ya en go.mod)
- lipgloss v1.1.0 (ya en go.mod)
- `~/.config/opencode/opencode.json` con sección `providers`

## Success Criteria

- [ ] `internal/opencode/` compila y lee providers desde opencode.json
- [ ] TUI 2-pasos permite seleccionar provider → modelo por fase Zyro-SDD
- [ ] `go test ./...` pasa sin errores
- [ ] `go build ./...` compila correctamente
- [ ] Output se escribe en `opencode.json` sección `agent`
