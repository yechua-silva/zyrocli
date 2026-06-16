# Proposal: Agent Procedural Flow

## Intent

Cuando el usuario dice "quiero hacer X" en OpenCode, el Zyro Agent pregunta al
humano de inmediato porque el `AGENT.md` es una lista estática de skills y fases,
sin instrucciones procedurales. El agente tiene tools (`webfetch`, `context7`,
`bash`) pero no sabe cuándo usarlas.

## Scope

### In Scope
- Modificar `AGENT.md.tmpl` para incluir un flujo procedural obligatorio de 7 pasos
- Incluir en el flujo: investigación con webfetch + context7, Skill Advisor, validación humana
- Actualizar el spec `project-scaffold` si sus requirements sobre AGENT.md cambian

### Out of Scope
- NO tocar `run.go`, `internal/scheduler/`, `internal/investigation/`
- NO meter pre-investigación en el CLI
- NO modificar el pipeline Go existente

## Capabilities

### New Capabilities
None — no se introduce una nueva capability; se modifica un template existente.

### Modified Capabilities
- `project-scaffold`: los requirements sobre AGENT.md (`AGENT.md Reflects Macro Fases 1-4`,
  `AGENT.md Enforcement Table`) se extienden para incluir el flujo procedural de 7 pasos
  como parte del contrato de scaffold.

## Approach

**Approach 1 (seleccionado)**: Modificar únicamente `AGENT.md.tmpl`. El template
incluirá una sección `## Flujo de inicio obligatorio` que el agente LLM debe
seguir textualmente cuando recibe una instrucción del usuario. El flujo fuerza
investigación (webfetch + context7) antes de preguntar, y finaliza con
recomendaciones informadas + validación humana.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/scaffold/templates/go-project/AGENT.md.tmpl` | Modified | Agregar sección de flujo procedural obligatorio |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| LLM ignora instrucciones si el prompt es muy largo | Medium | Diseñar el flujo conciso con bullets + tablas, no párrafos |
| webfetch falla sin internet | Low | El paso 3 debe ser best-effort: si falla, continúa con lo que tiene |

## Rollback Plan

`git revert` del cambio en `AGENT.md.tmpl`. El cambio toca 1 archivo — revert es
inmediato y sin efectos secundarios.

## Dependencies

- Ninguna. El template se renderiza en el scaffold existente sin cambios en Go.

## Success Criteria

- [ ] `AGENT.md.tmpl` renderizado tiene sección `## Flujo de inicio obligatorio`
- [ ] El flujo incluye pasos explícitos para `webfetch` y `context7` antes de preguntar
- [ ] El flujo termina con "¿Querés ajustar algo o continuamos?" y no con código automático
