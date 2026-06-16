# Proposal: Skill Validation Pipeline

## Intent

La fase 0 tiene 2 caminos desconectados para discovery de skills (run.go usa solo registry local, scheduler F1 usa solo API skills.sh) y ninguna aplica validación real. Necesitamos un pipeline único con 6 capas secuenciales que unifique ambos orígenes y rechace skills inseguros.

## Scope

### In Scope
- Pipeline unificado DiscoverAndRank (firma: `func(handoff.Payload) ([]ValidatedSkill, error)`)
- `BuildDiscoveryQuery()` — enriquece query con language+framework+keywords desde handoff.yaml
- `detectFramework()` — infiere framework desde lenguaje y directorio
- `ValidateAndScore()` — 6 capas secuenciales con hard block
- Publisher whitelist (NVIDIA, Anthropic, Microsoft, Google, Meta, etc.)
- SocketAlerts > 0 → REJECT (hard block), no soft penalty
- Merge resultados de API skills.sh + registry local
- Propagar `[]ValidatedSkill` en `Result` para F2+ en scheduler
- run.go usa `DiscoverAndRank` en vez de `RecommendFromHandoff`
- Tests de integración para el pipeline completo

### Out of Scope
- Interfaz gráfica para recomendaciones
- Cambios en la API de skills.sh
- Persistencia de recomendaciones entre sesiones
- UI/UX para mostrar validación

## Capabilities

### New Capabilities
None — this change refactors existing capabilities.

### Modified Capabilities
- `skill-advisor` — Major: añade `BuildDiscoveryQuery`, `detectFramework`, `ValidateAndScore` con hard block, publisher whitelist `VerifyPublisher`, merge local+API, `DiscoverAndRank` como entry point unificado. Tipos nuevos: `DiscoveryQuery`, `ValidatedSkill`.
- `scheduler-engine` — F1 usa `DiscoverAndRank`, propaga `Skills []ValidatedSkill` en `Result` a F2+.
- `zyrocli-run` — run.go usa `DiscoverAndRank` (incorpora API skills.sh y validación).

## Approach

Pipeline unificado en `internal/skilladvisor/`:

```
handoff.Payload → BuildDiscoveryQuery() → [API Discover() + Registry.Recommend()]
                                              ↓
                                       ValidateAndScore()
                                        1. SocketAlerts > 0 → REJECT
                                        2. Publisher ∉ whitelist → penalty -50
                                        3. Language mismatch → -10
                                        4. Framework mismatch → -20
                                        5. Project type mismatch → -30
                                        6. ScoreSkillWeighted() ranking
                                              ↓
                                       Sort + top-N → []ValidatedSkill
```

`F1AgentFunc` llama `DiscoverAndRank()` y guarda `Result.Skills`. `run.go` migra de `RecommendFromHandoff` → `DiscoverAndRank()`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/skilladvisor/query.go` | **New** | BuildDiscoveryQuery, detectFramework, extractKeywords |
| `internal/skilladvisor/verify.go` | **New** | Publisher whitelist + VerifyPublisher |
| `internal/skilladvisor/types.go` | Modified | DiscoveryQuery, ValidatedSkill, ValidationError |
| `internal/skilladvisor/discover.go` | Modified | Discover acepta DiscoveryQuery |
| `internal/skilladvisor/score.go` | Modified | ValidateAndScore con hard block + whitelist |
| `internal/skilladvisor/registry.go` | Modified | Merge local+API, RecommendFromHandoff deprecated |
| `internal/scheduler/macro_runner.go` | Modified | F1 usa DiscoverAndRank, Result.Skills propagado |
| `internal/scheduler/phase.go` | Modified | Skills []ValidatedSkill en Result |
| `cmd/zyrocli/run.go` | Modified | usa DiscoverAndRank |
| `internal/skilladvisor/skilladvisor_test.go` | Modified | Tests integración pipeline completo |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| API skills.sh cambia formato respuesta | Low | Deserialización tolerante + test con fixture |
| Whitelist desactualizada | Low | Constante en verify.go, fácil de extender |
| Hard block rechaza skills útiles con alerts | Low | SocketAlerts mal reportados → log warning + bypass flag |

## Effort Estimate

| File | Tipo | Estimación |
|------|------|------------|
| `query.go` | New | ~60 loc |
| `verify.go` | New | ~40 loc |
| `types.go` | Mod | +30 loc |
| `discover.go` | Mod | ~20 loc |
| `score.go` | Mod | ~80 loc |
| `registry.go` | Mod | ~30 loc |
| `macro_runner.go` | Mod | ~25 loc |
| `phase.go` | Mod | ~10 loc |
| `run.go` | Mod | ~15 loc |
| Tests | Mod | ~120 loc |
| **Total** | | **~430 loc** |

## Rollback Plan

1. Revert run.go → vuelve a `RecommendFromHandoff` (registry local)
2. Revert macro_runner.go → F1 vuelve a Discover() plano sin validación
3. Revert phase.go → quita Skills de Result
4. Eliminar query.go y verify.go
5. `go test ./internal/...` debe pasar

## Dependencies

- `internal/handoff/payload.go` — Payload.Project.Language ya existe (Framework agregado)

## Success Criteria

- [ ] `go test ./internal/skilladvisor/...` pasa con tests de integración
- [ ] `go test ./internal/scheduler/...` pasa (F1 propaga skills)
- [ ] `go test ./cmd/zyrocli/...` pasa (run.go usa pipeline unificado)
- [ ] SocketAlerts > 0 → `ValidationError` con `HardBlocked` = true
- [ ] Publisher NVIDIA, Anthropic, etc. → pasa whitelist; publisher desconocido → penalizado
- [ ] F1 produce `Result.Skills` con al menos 1 entry (no vacío)
- [ ] Ambos caminos (run.go + scheduler) usan `DiscoverAndRank`
