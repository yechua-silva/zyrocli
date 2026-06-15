# Validación de Arquitectura — ZyroAgentCLI

**Fecha**: 2026-06-14
**Estado**: Pre-SDD — validación antes del cambio grande

---

## Consistencia

### Hallazgos críticos

**1. AGENT.md vs. Arquitectura nueva — dualidad de fases**

El `AGENT.md` (el contrato vivo del proyecto) describe **4 macro-fases (F1→F4)** con un scheduler Go que orquesta. La propuesta nueva describe **5 macro-fases (0→4)** donde el pipeline vive en el agente (OpenCode), no en Go. Esto es una **contradicción directa**:

- `AGENT.md` línea ~45: "Flujo de 4 macro-fases (scheduler state machine)"
- Propuesta nueva: "Macro 3: Ejecución por pares (Zyro Agent + SDD skills)"

**Problema**: Si el agente orquesta, el scheduler Go (`internal/scheduler/`) se vuelve redundante. Pero el AGENT.md lo documenta como parte integral. Hay que decidir: ¿quién lidera?

**2. Artifact store: Engram vs. openspec — confusión residual**

La propuesta dice "Artifact store: Engram (NO openspec)". Pero:
- `openspec/config.yaml` existe y tiene reglas configuradas
- `openspec/specs/` tiene 5 specs (handoff-parser, project-scaffold, python-tools, scheduler-engine, zyrocli-run)
- `openspec/changes/` tiene 1 cambio archived + 1 activo

**Problema**: No se puede decir "NO openspec" y tener openspec configurado. Si Engram es el store, ¿qué pasa con los specs existentes? ¿Se migran? ¿Se archivan? ¿Se ignoran?

**3. Skill advisor: Go nativo vs. skill del agente**

El `AGENT.md` dice explícitamente: "Skill Advisor: Corre en paralelo con agentes Python. `internal/skilladvisor/score.go` hace ScoreSkill determinístico". La propuesta pregunta si debería ser Go nativo o skill del agente.

**Problema**: El AGENT.md ya respondió esta pregunta. Si cambiamos, hay que actualizar AGENT.md. Si no cambiamos, hay que implementar la lógica Go real (que hoy es stub).

**4. C-I-O DSL: relevancia incierta**

`internal/spec/cio.go` tiene structs definidas pero `compile.go` es stub. La propuesta pregunta si sigue siendo relevante. Pero el `AGENT.md` lo describe como parte de F2 (Especificación): "C-I-O DSL: Convierte HU del handoff.yaml a contratos".

**Problema**: Si eliminamos C-I-O, F2 pierde su mecanismo de especificación. Si lo mantenemos, hay que implementar compile.go. No hay tercera opción documentada.

**5. handoff.yaml — user_stories (plural) vs. user_story (singular)**

El `handoff.yaml` real del proyecto usa `user_story` (singular). La propuesta nueva dice `user_stories` (plural). El struct Go en `payload.go` también usa `UserStory` (singular).

**Problema**: Inconsistencia de nomenclatura que puede causar bugs de parsing si no se alinea.

### Hallazgos menores

- `zyro-skill-overrides.yaml` tiene un custom skill con path hardcoded `/path/to/custom-skill/SKILL.md` — claramente un placeholder.
- `go.mod` declara `go 1.26.3` pero el handoff.yaml dice `go_version: "1.26"` y AGENT.md dice "Go 1.22+". Hay 3 versiones diferentes.
- La propuesta menciona "zyro-doc-sync → genera ARCHITECTURE.md + CHANGELOG.md" pero no hay ningún mecanismo de generación automática de docs definido.

---

## Dependencias y orden

### Orden recomendado de implementación

```
FASE 0: Resolución de decisiones pendientes ( Blocking )
├── Decidir: ¿Scheduler Go o agente lidera?
├── Decidir: ¿Engram o openspec como artifact store?
├── Decidir: ¿C-I-O se implementa o se elimina?
├── Decidir: ¿Skill advisor Go o agente?
└── Actualizar AGENT.md con decisiones

FASE 1: Fundamentos Go ( sin dependencias externas )
├── internal/handoff/ — COMPLETO ✅ (29 tests, 96.6% coverage)
├── internal/scaffold/ — COMPLETO ✅ (tests passing)
├── internal/scheduler/ — COMPLETO ✅ (tests passing, stubs funcionales)
└── cmd/zyrocli/ — COMPLETO ✅ (run, init, tests)

FASE 2: Componentes stub → implementación real
├── internal/skilladvisor/ — implementar lógica real
│   ├── registry.go → carga YAML real
│   ├── score.go → weighted scoring
│   └── discover.go → skills.sh API client
├── internal/spec/ — decidir y ejecutar
│   └── Si C-I-O: implementar compile.go
│   └── Si no: eliminar paquete, actualizar AGENT.md
└── internal/context/ — implementar bridge.go
    └── os/exec para MCP server, JSON-RPC framing

FASE 3: Skills del agente (requiere FASE 2)
├── zyro-doc-export (SKILL.md)
├── zyro-doc-search (SKILL.md + Go function)
├── zyro-sdd-pairs (orquestador de pares)
└── .zyro/conventions.yaml

FASE 4: Integración y delivery
├── zyro-doc-index (Go function)
├── zyro-doc-sync (CLI command)
├── work-unit commits integration
├── branch-pr integration
└── chained-pr integration

FASE 5: Verificación y cierre
├── Contract testing (internal/test/)
├── Graphify periódico
├── Engram archive final
└── AGENT.md vivo — sincronización
```

### Dependencias críticas

| Componente | Depende de | Bloquea |
|------------|-----------|---------|
| Skill advisor real | Decisión Go vs. agente | Macro 1 investigación |
| Context bridge | Decisión C-I-O relevance | Macro 1 investigación |
| C-I-O compile | Decisión de mantener/eliminar | Macro 2 planificación |
| zyro-sdd-pairs | SDD skills existentes | Macro 3 ejecución |
| doc tools | Engram como store | Macro 4 delivery |
| Contract testing | C-I-O o alternativa | Macro 3 verificación |

---

## Riesgos

### Riesgo 1: Dualidad de control (CRÍTICO)

**Descripción**: El scheduler Go y el agente OpenCode compiten por ser el orquestador. Si ambos intentan liderar el pipeline, se genera caos.

**Impacto**: El proyecto puede quedar con dos sistemas paralelos que no se comunican.

**Mitigación**: Decidir ANTES de implementar. Recomendación: el agente lidera, el scheduler Go se convierte en utilidad (helper para validación de estado, no orquestador).

### Riesgo 2: Scope creep del SDD grande (ALTO)

**Descripción**: La propuesta incluye 15+ componentes nuevos. Sin priorización clara, todo se vuelve bloqueante.

**Impacto**: El proyecto nunca termina porque siempre falta "el siguiente componente".

**Mitigación**: Implementar por MVP. El mínimo viable es: handoff parser ✅ + scaffold ✅ + agente abre OpenCode ✅. Todo lo demás es iterativo.

### Riesgo 3: Openspec zombie (MEDIO)

**Descripción**: Se dice "NO openspec" pero los specs existentes no se migran ni archivan. Quedan como artefactos muertos.

**Impacto**: Confusión futura sobre dónde está la verdad.

**Mitigación**: Archivar explícitamente los specs de openspec a Engram ANTES de declarar Engram como store.

### Riesgo 4: Skill advisor sin lógica real (MEDIO)

**Descripción**: `internal/skilladvisor/` tiene structs y interfaces pero cero lógica real. `ScoreSkill()` es un dot product trivial que no considera pesos.

**Impacto**: Las recomendaciones de skills serán triviales o incorrectas.

**Mitigación**: Implementar scoring real con pesos por categoría (language +10, framework +20, etc. como documenta AGENT.md).

### Riesgo 5: Context bridge sin definición clara (MEDIO)

**Descripción**: `internal/context/bridge.go` es un stub. No hay definición de qué MCP server se arranca, cómo se comunica, ni qué datos devuelve.

**Impacto**: La "investigación" de Macro 1 queda como concepto sin implementación.

**Mitigación**: Definir contrato de bridge primero: input (librerie detectadas), output (docs relevantes), protocolo (JSON-RPC over stdio).

### Riesgo 6: Go version mismatch (BAJO)

**Descripción**: Tres versiones diferentes de Go declaradas en diferentes archivos.

**Impacto**: Confusión al buildlear, posibles incompatibilidades.

**Mitigigation**: Unificar a una sola versión en go.mod, handoff.yaml, y AGENT.md.

---

## Huecos

### 1. No hay definición de "investigación"

Macro 1 dice "INVESTIGA con: Context MCP, GitMCP, Web fetch, Playwright". Pero:
- ¿Quién ejecuta estas herramientas? ¿El agente directamente o una función Go?
- ¿Los resultados se guardan en Engram o se pasan como contexto al agente?
- ¿Hay rate limiting? ¿Timeouts?

### 2. No hay flujo de error

Las macro fases describen el happy path. ¿Qué pasa cuando:
- Context MCP server no responde?
- Skills.sh API está caída?
- El agente OpenCode se cierra inesperadamente?
- Engram no está disponible?

### 3. No hay métricas de éxito

¿Cómo se mide que la arquitectura funciona? No hay:
- Tiempo promedio por fase
- Tasa de aprobación humana
- Número de iteraciones por feature
- Cobertura de tests

### 4. No hay rollbacks

Si el agente toma una mala decisión en Macro 1 (recomienda stack incorrecto), ¿cómo se revierte? No hay mecanismo de undo.

### 5. Doc tools sin especificación

`zyro-doc-index`, `zyro-doc-search`, `zyro-doc-sync`, `zyro-doc-export` se mencionan pero no hay:
- Formato de `.zyro/doc-index.yaml`
- Protocolo de búsqueda de `zyro-doc-search`
- Qué genera exactamente `zyro-doc-sync`

### 6. Chained PRs sin métrica

La propuesta dice "Chained PRs si >400 líneas". ¿Quién mide? ¿El agente? ¿El scheduler? ¿Hay threshold configurable?

---

## Decisiones pendientes

### Decisión 1: ¿Quién lidera el pipeline? (BLOCKING)

**Opción A**: Scheduler Go (estado actual del AGENT.md)
- Pros: Determinístico, testeable, no depende de LLM
- Contra: Rígido, no puede adaptarse a contextos dinámicos

**Opción B**: Agente OpenCode (propuesta nueva)
- Pros: Flexible, puede investigar, adaptarse, tomar decisiones
- Contra: No determinístico, costoso en tokens, puede alucinar

**Opción C**: Híbrido — Scheduler Go valida estado, agente ejecuta lógica
- Pros: Lo mejor de ambos mundos
- Contra: Complejidad de integración

**Recomendación**: Opción C. El scheduler Go se convierte en "state validator" que verifica que el agente completó cada fase correctamente. El agente lidera la ejecución.

### Decisión 2: ¿Artifact store final? (BLOCKING)

**Opción A**: Engram (propuesta)
- Pros: Persistente, cross-session, ya integrado
- Contra: Pierde la estructura de specs de openspec

**Opción B**: Openspec (estado actual)
- Pros: Estructura clara, ya configurado
- Contra: No es cross-session, no tiene MCP

**Opción C**: Engram primario + openspec como export format
- Pros: Engram para trabajo, openspec para documentación
- Contra: Doble mantenimiento

**Recomendación**: Opción C. Engram para el trabajo del agente, openspec como formato de export para humanos.

### Decisión 3: ¿C-I-O se implementa? (RELEVANTE)

**Opción A**: Implementar compile.go
- Pros: Genera OpenAPI/protobuf automáticamente
- Contra: Complejidad, puede no ser necesario para CLI tools

**Opción B**: Eliminar C-I-O
- Pros: Menos código, más simple
- Contra: Pierde la especificación formal

**Opción C**: C-I-O como documentación, no como DSL
- Pros: Mantiene la estructura sin la complejidad de compilación
- Contra: No genera output automático

**Recomendación**: Opción C. C-I-O como estructura de documentación para el agente, no como DSL compilable.

### Decisión 4: ¿Skill advisor Go o agente? (RELEVANTE)

**Opción A**: Go nativo (estado del AGENT.md)
- Pros: Rápido, determinístico, testeable
- Contra: Lógica limitada, no puede "entender" contexto

**Opción B**: Skill del agente
- Pros: Puede entender contexto, adaptar recomendaciones
- Contra: Lento, costoso en tokens, no determinístico

**Recomendación**: Go nativo para scoring base + agente para refinamiento. El Go hace el trabajo pesado, el agente ajusta.

---

## Recomendación final

### Estado: AJUSTAR — la propuesta es sólida pero tiene inconsistencias que resolver

### Acciones inmediatas (antes de arrancar SDD):

1. **Actualizar AGENT.md** con las nuevas macro-fases (0→4) y decidir quién lidera
2. **Unificar nomenclatura**: `user_story` (singular) en toda la cadena
3. **Unificar Go version**: una sola versión en todos los archivos
4. **Archivar openspec existente** a Engram antes de declarar Engram como store
5. **Definir contrato de context bridge**: input/output/protocolo
6. **Decidir C-I-O**: implementar, eliminar, o documentar

### Para el SDD grande:

Dividir en 3 chained PRs:

**PR 1: Fundamentos** (~200 líneas)
- Actualizar AGENT.md
- Unificar versiones
- Archivar openspec
- Actualizar handoff.yaml si se cambia user_stories

**PR 2: Componentes core** (~350 líneas)
- Skill advisor: implementar lógica real
- Context bridge: implementar con MCP
- C-I-O: decidir y ejecutar

**PR 3: Skills y delivery** (~300 líneas)
- zyro-doc-export
- zyro-doc-search
- zyro-sdd-pairs
- .zyro/conventions.yaml

Cada PR es mergeable independientemente y no rompe el flujo existente.

---

## Archivos referenciados

- `AGENT.md` — contrato vivo del proyecto (NECESITA ACTUALIZACIÓN)
- `cmd/zyrocli/run.go` — flujo principal (COMPLETO)
- `cmd/zyrocli/init.go` — comando init (COMPLETO)
- `internal/handoff/payload.go` — structs del contrato (COMPLETO)
- `internal/scaffold/scaffold.go` — scaffolding (COMPLETO)
- `internal/scheduler/scheduler.go` — state machine (COMPLETO, decidir rol)
- `internal/skilladvisor/` — stubs (NECESITA IMPLEMENTACIÓN)
- `internal/spec/cio.go` — structs (DECIDIR RELEVANCIA)
- `internal/context/bridge.go` — stub (NECESITA IMPLEMENTACIÓN)
- `internal/apply/runner.go` — stub (depende de C-I-O)
- `internal/test/` — stubs (depende de C-I-O)
- `openspec/config.yaml` — configuración actual (ARCHIVAR O MANTENER)
- `zyro-skill-overrides.yaml` — tiene placeholder hardcoded
