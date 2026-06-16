# Proposal: Bridge Orchestration-Agent Pipeline

## Intent

`cmd/zyrocli/run.go` ejecuta scaffold + doc.Sync + `exec.Command("opencode")` — pero **nunca** llama al scheduler, ResearchEngine, o Advisor. El agente LLM recibe un AGENT.md informativo sin contexto pre-computado ni flujo procedural, forzándolo a preguntar todo al humano.

Solución: que `zyrocli run` ejecute investigación REAL (ResearchEngine + Advisor) **antes** de abrir OpenCode, persista resultados a `.zyro/investigation-report.md`, y genere un AGENT.md dinámico con contexto pre-computado.

## Scope

### In Scope
- ResearchEngine se ejecuta en `run.go` antes de scaffold/opencode (3 fuentes concurrentes con timeout)
- Advisor genera recomendaciones de stack, patrones, skills
- Resultados persistidos a `.zyro/investigation-report.md` + Engram
- AGENT.md.tmpl recibe nuevos campos: `InvestigationReport`, `Advisory`, `ProceduralFlow`
- Template incluye flujo procedural obligatorio (no solo informativo)
- GuidedApproval opcional post-investigación antes de abrir OpenCode
- Scheduler.F1 harness integrado como pre-flight (no como reemplazo del flujo completo)

### Out of Scope
- Pipeline scheduler completo F1-F4 (futuro PR)
- GuidedApproval entre fases dentro de OpenCode (el agente gestiona eso internamente)
- Interfaz gráfica para investigación
- Web fetch callback para páginas autenticadas

## Capabilities

### New Capabilities
- `pre-flight-investigation`: ResearchEngine + Advisor ejecutados en `run.go` con persistencia a `.zyro/investigation-report.md`
- `agent-context-enrichment`: AGENT.md generado con investigación pre-computada + flujo procedural

### Modified Capabilities
- `zyrocli-run`: incorpora pre-investigación + scheduler harness F1 + GuidedApproval antes de `exec.Command("opencode")`
- `investigation`: sus tipos `Report` y `Advisory` se serializan a `.zyro/investigation-report.md` + se inyectan en el template pipeline

## Approach

**Pre-investigación en run.go**. Flujo nuevo:

```
run.go:
  1. Parse handoff.yaml (existente)
  2. ResearchEngine.Run(ctx) — 3 fuentes concurrentes con 30s timeout
  3. Advisor.Analyze(report) → Advisory
  4. Skill advisor rankea skills (existente, pero ahora recibe contexto)
  5. Guardar .zyro/investigation-report.md (Report.Markdown() + Advisory.Markdown())
  6. MemSave a Engram topic "sdd/{project}/investigation"
  7. Optional: GuidedApproval (resumen + recomendar continuar)
  8. Scaffold.Run(cfg) — AGENT.md recibe InvestigationReport + Advisory (existente + nuevo)
  9. doc.Sync() (existente)
  10. exec.Command("opencode") (existente)
```

Template AGENT.md.tmpl gana sección procedural:
```
## Flujo obligatorio
1. Leer .zyro/investigation-report.md (contexto pre-computado)
2. Identificar skills a instalar de la recomendación
3. Ejecutar fase SDD-1 (proposal) basado en la investigación
4. Reportar al humano antes de cada transición
```

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/zyrocli/run.go` | Modified | Agregar pre-investigación + scheduler harness antes de opencode |
| `internal/scaffold/templates/go-project/AGENT.md.tmpl` | Modified | Nuevos campos + flujo procedural |
| `internal/scaffold/scaffold.go` | Modified | `Config` gana campos `InvestigationReport` y `Advisory` |
| `internal/scaffold/renderer.go` | Modified | Template recibe nuevos campos de Config |
| `internal/investigation/` | No change | Ya existe, solo se integra |
| `.zyro/investigation-report.md` | New | Archivo generado por run.go |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| ResearchEngine timeout (30s) bloquea el flujo | Medium | Context with timeout; si falla, log warning y continúa sin reporte |
| Web fetch lento o fallido | High | PartialFailure ya manejado: errores por fuente, no fatal |
| Template pipeline se rompe con nuevos campos | Low | Config embebido con valores zero (empty string) como fallback |
| GuidedApproval bloquea en CI | Low | Solo si stdin es interactivo; detectar con `isatty` o flag `--yes` |

## Rollback Plan

1. Revertir cambios en `run.go` → vuelve al bootstrap directo
2. Revertir template → AGENT.md vuelve a versión informativa
3. `.zyro/investigation-report.md` no se crea si se revierte run.go
4. Engram entry se marca obsoleto por topic_key (no se borra)

## Dependencies

- `internal/investigation/research.go` y `advisor.go` — ya implementados
- `internal/scheduler/approval.go` — GuidedApproval ya implementado
- `golang.org/x/term` — para detectar `isatty` (stdin interactivo)
- `context` package con timeout de 30s por fuente

## Success Criteria

- [ ] `zyrocli run` ejecuta ResearchEngine + Advisor antes de scaffold
- [ ] `.zyro/investigation-report.md` existe después de `zyrocli run`
- [ ] AGENT.md incluye sección "Flujo obligatorio" procedural
- [ ] AGENT.md incluye resumen de investigación pre-computada
- [ ] GuidedApproval muestra resumen de investigación y pide confirmación
- [ ] Sin stdin interactivo (`--yes` o pipe), GuidedApproval se salta automáticamente
- [ ] `go test ./internal/investigation/...` pasa sin cambios (regression)
- [ ] `go test ./internal/scaffold/...` pasa con nuevos campos
