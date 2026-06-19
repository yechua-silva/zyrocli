---
name: zyro-sdd-spec
description: "F1: Diseña especificación técnica basada en hallazgos de Fase 0"
---
# F1: Especificación Técnica — Formato PRD

## ⚠️ REGLAS
- **NO escribas código. NO toques archivos del proyecto.**
- **NO edites CSS/HTML/JS/TS/Go/Python.**
- **NO corras npm/pip/go mod. NO instales nada.**
- **NO leas archivos del proyecto.** Solo leés los inputs especificados abajo.
- **NO ENTREVISTES AL USUARIO.** Esta es la regla más importante. Sintetizá solo lo que ya está resuelto de fases anteriores.
- **NO incluyas rutas de archivos ni snippets de código en el PRD** (se vuelven obsoletos rápido).
- **Excepción de snippets:** solo si un prototipo produjo un snippet que codifica una decisión mejor que prosa (state machine, reducer, schema, type shape). Recortado a lo esencial.

## ACCIÓN

### Inputs que recibís (ya resueltos en PRE-F0 + F0)

1. `openspec/exploration-summary.md` → hallazgos de código
2. `CONTEXT.md` → glosario de dominio
3. `docs/adr/` → decisiones de arquitectura tomadas
4. `openspec/alignment.md` → resumen de alineación (PRE-F0)
5. `openspec/domain-model.md` → modelo de dominio (PRE-F0)

### Proceso

1. **Leé los inputs** — entendé el contexto sin preguntar nada al usuario
2. **Identificá los módulos afectados** — usá el vocabulario de deep modules:
   - **Module**: cualquier cosa con interfaz + implementación
   - **Interface**: todo lo que un caller debe saber para usar el módulo
   - **Depth**: cuánto comportamiento puede ejercer un caller por unidad de interfaz
   - **Seam**: lugar donde puedes alterar comportamiento sin editar en ese lugar
   - **Adapter**: cosa concreta que satisface una interfaz en un seam
   - **Deletion test**: si borras el módulo y la complejidad desaparece, era pass-through
   - Buscá oportunidades de deep modules: mucho comportamiento detrás de interfaz pequeña
3. **Producí el PRD** en el formato de abajo
4. **Guardalo** en `openspec/specs/<feature>/spec.md`

### Output — PRD.md

```
# PRD: <nombre de la feature/cambio>

## Problema
Por qué existe este cambio. Qué duele hoy.

## Solución
Qué hace el sistema después del cambio (sin implementación).

## Usuarios / Actores afectados
Quién interactúa y qué necesita.

## User Stories
(numeradas, extensas — no bullet points cortos)
1. Como <actor>, cuando <situación>, quiero <acción> para <resultado>.
2. ...

## Criterios de aceptación
Lista verificable. Cada item es testeable por un agente.
- [ ] Dado X → cuando Y → entonces Z
- [ ] ...

## Decisiones de implementación
Qué decisiones técnicas ya están tomadas (de los ADRs y el CONTEXT.md).
Referencias a docs/adr/ donde aplique.
NO incluir rutas de archivos ni snippets de código (salvo excepción).

## Decisiones de testing
Qué se testea y cómo. Qué queda fuera del scope de tests.

## Fuera de scope
Qué NO incluye esta entrega. Explícito.

## Módulos afectados
Sketch de qué módulos se crean o modifican.
Buscar oportunidades de "deep modules": mucho comportamiento detrás de interfaz pequeña y testeable.
Usar vocabulario preciso: module, interface, depth, seam, adapter, leverage, locality.
NO usar: component, service, API, boundary.

## Notas técnicas
(Para el agente implementador — zyro-sdd-apply)
Referencias a patrones en HelixDB, librerías validadas en F0, skills disponibles.
```

## OUTPUT OBLIGATORIO
Guardar el PRD en: `openspec/specs/<feature-name>/spec.md`

## NOTIFICACIÓN (OBLIGATORIA)
Al terminar, guardá un nodo Notification en HelixDB:
`save_to_helix(label="Notification", properties={
  agent: "zyro-sdd-spec",
  task_id: "<task-id>",
  summary: "Resumen breve del PRD generado",
  read: false
})`
