## Briefing técnico para el agente — Absorción de skills Pocock en ZyroCLI

### Contexto
Se van a absorber dos skills de Matt Pocock directamente en agentes existentes de ZyroCLI. No se crean agentes nuevos ni skills paralelas. Se modifican los prompts de zyro-sdd-explore y zyro-sdd-spec.
Fuentes primarias que el agente DEBE leer antes de modificar nada:

- grill-with-docs SKILL.md real: https://github.com/mattpocock/skills/blob/main/skills/engineering/grill-with-docs/SKILL.md
- to-prd descripción completa: https://www.aihero.dev/skills-to-prd.md
- Artículo explicativo de grill-with-docs: https://www.aihero.dev/grill-with-docs.md
- Repo completo de skills: https://github.com/mattpocock/skills

### Cambio 1 — zyro-sdd-explore absorbe /grill-with-docs
Archivo a modificar: .claude/skills/zyro-sdd-explore.md (o donde viva el prompt del agente)
Comportamiento a añadir al inicio del agente, antes de explorar código:
#### Paso 1 — READ DOCS (nuevo, va primero)
El agente debe buscar y leer en este orden:
```
openspec/           → specs y PRDs anteriores del proyecto
docs/adr/           → decisiones de arquitectura previas
CONTEXT.md          → glosario de lenguaje compartido (puede no existir)
AGENT.md           → personalidad y reglas del proyecto
.zyro/config.yaml   → configuración del proyecto
```

#### Paso 2 — INTERVIEW (nuevo, va después de READ DOCS)
Con lo leído, el agente entrevista al usuario con estas reglas exactas (tomadas de la SKILL.md de Pocock):

- Hace una pregunta a la vez, espera respuesta antes de continuar
- Si una pregunta puede responderse explorando el codebase, explora en vez de preguntar
- Cuando el usuario usa un término que conflictúa con CONTEXT.md: lo señala inmediatamente. Ejemplo: "Tu glosario define 'task' como X, pero pareces referirte a Y — ¿cuál es el canónico?"
- Cuando el usuario usa lenguaje vago o sobrecargado: propone término canónico. - Ejemplo: "Dices 'job' — ¿te refieres al Task del TaskBoard MCP o a una operación de HelixDB?"
- Cuando se discuten relaciones entre entidades: stress-test con escenarios concretos

#### Paso 3 — OUTPUT (modificado)
Actualmente sdd-explore produce un reporte de exploración. Con el cambio, produce tres archivos:

```
openspec/exploration-summary.md   → lo que existe en código (igual que antes)
CONTEXT.md                        → glosario actualizado/creado con términos resueltos
docs/adr/NNNN-<decision>.md       → solo si surgió una decisión no-obvia e irreversible 
```

### Cambio 2 — zyro-sdd-spec absorbe /to-prd
Archivo a modificar: .claude/skills/zyro-sdd-spec.md
Comportamiento crítico de /to-prd a preservar:

`El agente NO re-entrevista al usuario. Sintetiza lo que ya está resuelto del paso anterior (grill-with-docs / sdd-explore).`

El agente lee como input: openspec/exploration-summary.md + CONTEXT.md + docs/adr/ y produce el spec en este formato PRD (reemplaza el formato técnico actual):
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

## Decisiones de testing
Qué se testea y cómo. Qué queda fuera del scope de tests.

## Fuera de scope
Qué NO incluye esta entrega. Explícito.

## Módulos afectados
Sketch de qué módulos se crean o modifican.
Buscar oportunidades de "deep modules": mucho comportamiento detrás de interfaz pequeña y testeable.

## Notas técnicas
(Para el agente implementador — zyro-sdd-apply)
Referencias a patrones en HelixDB, librerías validadas en F0, skills disponibles.
```
Concepto "deep modules" que el agente debe aplicar activamente: una función/módulo "profundo" oculta complejidad real detrás de una interfaz pequeña y estable. Es testeable por esa interfaz. En el contexto de ZyroCLI: EngramStore es un deep module (10 métodos, complejidad vector+BM25+RRF oculta). El agente debe buscar oportunidades similares en cada PRD.


### Cambio 3 un prestapa antes de fase 0, un fase de contexto y entrevista
con un flujo como esto:
```
    [NUEVO] Pre-F0 "Alineación"
    zyrocli align
        → /grill-me         (entrevista al usuario)
        → /domain-model     (output: alignment.md + domain-model.md)
            ↓ alimenta ↓
    [EXISTENTE] F0 — Investigación
        zyro-phase-0-patterns   → ahora sabe qué buscar
        zyro-phase-0-libraries  → ídem
        zyro-skills-find/audit  → ídem
        + /handoff al finalizar
```

- `/grill-me` → Gap real. No tienes nada pre-F0 que entreviste al usuario antes de investigar. F0 investiga patrones/librerías pero arranca sin saber bien el WHAT. Iría como comando zyrocli align antes de todo el pipeline.
- `/domain-model` → También falta. zyro-phase-0-patterns busca patrones pero no genera un modelo de dominio estructurado. El output de esto sería el input ideal para F0, así los subagentes de patrones y librerías buscarían con contexto real.
- `/triage` → No tienes equivalente. El Task Board MCP tiene list_tasks / cancel_task pero no tiene concepto de "ordenar el backlog caótico entre sesiones". Útil cuando tienes tareas acumuladas sin priorizar.
- `/improve-codebase-architecture` → No existe nada parecido. zyro-sdd-verify chequea correctitud contra specs, requesting-code-review es más sobre estilo. Esta skill busca refactors que mejoran navegabilidad para agentes futuros — post-F4 o cross-project.