---
name: zyro-sdd-propose
description: "F2: Crea propuestas de cambio con intento, alcance y enfoque"
---
# F2: Propuesta de Cambio — Intento, Alcance y Enfoque

## ⚠️ REGLAS
- **NO implementes código.** Solo proponés — el diseño y la ejecución vienen después.
- **NO edites archivos del proyecto.**
- **NO corras npm/pip/go mod. NO instales nada.**
- **NO leas archivos del proyecto.** Solo leés los inputs especificados abajo.
- **NO ENTREVISTES AL USUARIO.** Sintetizá solo lo que ya está resuelto de fases anteriores.
- **No incluyas código en la propuesta** — describí el enfoque, no la implementación.

## ACCIÓN

### Inputs que recibís (ya resueltos en PRE-F0 + F0 + F1)

1. `openspec/specs/<feature>/spec.md` → especificación técnica (PRD)
2. `openspec/alignment.md` → resumen de alineación (PRE-F0)
3. `openspec/domain-model.md` → modelo de dominio (PRE-F0)
4. `CONTEXT.md` → glosario de dominio
5. `docs/adr/` → decisiones de arquitectura tomadas

### Proceso

1. **Leé los inputs** — entendé el contexto sin preguntar nada al usuario
2. **Identificá el intento** — qué problema resuelve este cambio, por qué ahora
3. **Definí el alcance** — qué entra y qué queda fuera explícitamente
4. **Describí el enfoque** — cómo se va a implementar, sin código, a nivel de módulos y flujos
5. **Evaluá riesgos** — trade-offs, impacto en otras áreas, deuda técnica que se genera
6. **Estimá esfuerzo** — small/medium/large con criterios claros

### Output esperado

Escribí en `openspec/proposals/<feature>/proposal.md`:

```markdown
# Propuesta: <título>

## Intento
Por qué existe este cambio, qué problema resuelve, qué duele hoy.

## Alcance
### Incluye
- Lista de capacidades que entran

### Excluye
- Lista de capacidades que quedan fuera (explícito)

## Enfoque
Descripción de cómo se aborda, qué módulos se afectan, flujo de datos.
Sin código — solo arquitectura y decisiones.

## Riesgos
- Trade-offs identificados
- Impacto en otras áreas
- Deuda técnica que se genera

## Esfuerzo estimado
[small/medium/large] — criterio: archivos afectados, complejidad, riesgos.

## Criterios de éxito
Lista verificable: cómo saber si el cambio funcionó.
```

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-sdd-propose",
  task_id: "<task-id>",
  summary: "Resumen breve de lo que se completó",
  read: false
})`
