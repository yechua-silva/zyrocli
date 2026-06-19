---
name: zyro-sdd-explore
description: "Explora codebase y requerimientos antes de un cambio"
---
# F0: Exploración de Código con Grill-with-Docs

## ⚠️ REGLAS
- **NO modifiques archivos.** Solo leés, investigás y entrevistás.
- **NO corras bash excepto para grep, find, o leer archivos.**
- **NO guardes en HelixDB.** Tu output son archivos markdown.
- **Una pregunta a la vez.** Espera respuesta antes de continuar.
- **Si una pregunta puede responderse explorando el codebase, explora en vez de preguntar.**

## ACCIÓN

### Paso 1 — READ DOCS (siempre primero)

Buscá y leé los siguientes archivos/documentos en este orden exacto:

1. `openspec/` → specs y PRDs anteriores del proyecto
2. `docs/adr/` → decisiones de arquitectura previas
3. `CONTEXT.md` → glosario de lenguaje compartido (puede no existir)
4. `AGENT.md` → personalidad y reglas del proyecto
5. `.zyro/config.yaml` → configuración del proyecto (puede no existir — seguí sin él)

Si `CONTEXT.md` no existe, seguí sin él. Si `docs/adr/` no existe, seguí sin ADRs. Si `.zyro/config.yaml` no existe, seguí sin él.

### Paso 2 — INTERVIEW (después de READ DOCS)

Con lo leído, entrevistá al usuario para entender qué investigar. Reglas exactas:

- **Una pregunta a la vez** — nunca dos o más. El usuario se confunde.
- **Provide your recommended answer** — antes de preguntar, da tu respuesta sugerida: "Creo que deberíamos investigar X porque Y. ¿Estás de acuerdo?"
- **Codebase over interview** — si una pregunta se puede responder explorando el codebase, explora en vez de preguntar. Ej: "Déjame revisar el código para ver cómo se maneja eso."
- **Call out conflicts immediately** — cuando el usuario usa un término que conflictúa con CONTEXT.md, señálalo inmediatamente.
  Ej: "Tu glosario define 'task' como 'unidad de trabajo atómica', pero pareces referirte a 'task' como 'issue de GitHub'. ¿Cuál es el canónico?"
- **Sharpen vague language** — cuando el usuario usa lenguaje vago o sobrecargado, propone término canónico.
  Ej: "Dices 'job' — ¿te refieres al Task del TaskBoard MCP o a una operación de HelixDB?"
- **Stress-test with scenarios** — cuando se discuten relaciones entre entidades, inventá escenarios concretos que prueben bordes.
  Ej: "Si decís que Order → Invoice es 1:1, ¿qué pasa cuando un pedido tiene múltiples envíos parciales?"

### Paso 3 — OUTPUT (archivos)

Producí estos archivos:

```
openspec/exploration-summary.md   → lo que existe en código (hallazgos, estructura, patrones)
CONTEXT.md                        → glosario actualizado/creado con términos resueltos
docs/adr/NNNN-<decision>.md       → solo si surgió una decisión no-obvia e irreversible
```

Criterios para ADR (los 3 deben cumplirse):
- (a) Cara de revertir
- (b) Sorprendente sin contexto previo
- (c) Resultado de un trade-off real

Si falta cualquiera, NO generes ADR.

## OUTPUT
Resumen estructurado con:
- Archivos leídos: [paths]
- Patrones encontrados: [nombres]
- Decisiones detectadas: [decisiones]
- Archivos generados: [paths]
- Recomendación para el orquestador

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-sdd-explore",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
