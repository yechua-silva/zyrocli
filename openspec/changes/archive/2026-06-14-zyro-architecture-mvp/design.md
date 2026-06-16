# Design: zyro-architecture-mvp

## 1. Component Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  cmd/zyrocli/                                                       │
│  ├── main.go        → cobra root, verbose flag                     │
│  ├── run.go         → scaffold + opencode launch                   │
│  ├── init.go        → handoff parse + scaffold                     │
│  └── doc.go         → zyrocli doc sync/search (NUEVO)             │
├─────────────────────────────────────────────────────────────────────┤
│  internal/                                                          │
│                                                                     │
│  ┌─ scheduler/ ──────────────────────────────────────────────────┐  │
│  │  scheduler.go    → Run(), RunPhase() — state machine          │  │
│  │  phase.go        → Phase, PhaseRunner interface, Config        │  │
│  │  approval.go     → PromptApproval() — BLOCKING gate           │  │
│  │  phase_stubs.go  → F1Runner..F4Runner (stubs → reales)        │  │
│  │  config.go       → LoadConfig() from handoff.yaml             │  │
│  │                                                               │  │
│  │  ┌─────────────────────────────────────────────────────────┐   │  │
│  │  │ REFACTOR: scheduler se convierte en HARNESS             │   │  │
│  │  │ - F1→F4 stubs se reemplazan por MacroPhase runners      │   │  │
│  │  │ - Nuevo: HarnessValidator que verifica output del agente │   │  │
│  │  │ - Nuevo: AgentBridge para communicate con OpenCode       │   │  │
│  │  └─────────────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ skilladvisor/ ───────────────────────────────────────────────┐  │
│  │  registry.go     → Registry.Load(dir) — YAML manifests       │  │
│  │  score.go        → ScoreSkill() + Recommend() — weighted     │  │
│  │  discover.go     → Discover() — skills.sh API + cache        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ context/ ────────────────────────────────────────────────────┐  │
│  │  bridge.go       → Bridge{Start/Stop/QueryDocs/Resolve}      │  │
│  │                   → os/exec MCP subprocess lifecycle          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ spec/ ───────────────────────────────────────────────────────┐  │
│  │  cio.go          → CIO structs (Contract/Interface/Behavior)  │  │
│  │  compile.go      → Compile() → Engram key emission           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ apply/ ──────────────────────────────────────────────────────┐  │
│  │  runner.go       → Runner.Run(tasks) — goroutine pool        │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ test/ ───────────────────────────────────────────────────────┐  │
│  │  contracts.go    → ContractTest.Given/When/Then executor     │  │
│  │  report.go       → Report — graphify diff output             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ doc/ ────────────────────────────────────────────────────────┐  │
│  │  index.go        → GenerateIndex(root) — walk .md files      │  │
│  │  search.go       → Search(req) — Engram-first search         │  │
│  │  sync.go         → Sync(source, target) — copy docs          │  │
│  │  export.go       → Export(format) — markdown/PDF             │  │
│  │  conventions.go  → LoadConventions(yaml)                     │  │
│  │  graphify.go     → UpdateGraph(root) — graphify refresh      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ handoff/ ────────────────────────────────────────────────────┐  │
│  │  payload.go      → Payload structs (COMPLETO)                │  │
│  │  parser.go       → Parse() YAML (COMPLETO)                   │  │
│  │  validate.go     → Validate() (COMPLETO)                     │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ scaffold/ ───────────────────────────────────────────────────┐  │
│  │  scaffold.go     → Run() (COMPLETO)                          │  │
│  │  renderer.go     → Template rendering (COMPLETO)             │  │
│  │  writer.go       → File writing (COMPLETO)                   │  │
│  │  state.go        → .zyro/state.json (COMPLETO)               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Dependency Graph (package-level)

```
cmd/zyrocli ──→ internal/scheduler
            ──→ internal/handoff
            ──→ internal/scaffold
            ──→ internal/doc (NUEVO)

internal/scheduler ──→ internal/handoff
                    ──→ internal/skilladvisor (via F1)
                    ──→ internal/context (via F2)
                    ──→ internal/spec (via F2)
                    ──→ internal/apply (via F3)
                    ──→ internal/test (via F3)

internal/skilladvisor ──→ stdlib net/http (skills.sh API)
                       ──→ gopkg.in/yaml.v3

internal/context ──→ os/exec (MCP subprocess)

internal/spec ──→ internal/skilladvisor (CIO → skill mapping)
                ──→ Engram (key emission)

internal/apply ──→ internal/spec (CIO contracts)
                ──→ internal/test (validation)

internal/test ──→ internal/spec (CIO contracts)
               ──→ graphify (diff)

internal/doc ──→ Engram (mem_save, mem_search, mem_get_observation)
              ──→ stdlib filepath/fs (index, sync)
              ──→ gopkg.in/yaml.v3 (conventions)
              ──→ graphify (update)
```

## 2. Macro 1 — Documentación Engram puro

**Decisión clave**: ELIMINAR openspec del pipeline. Todo se persiste en Engram con topic keys estructuradas. El orquestador genera archivos .md al final (doc-export) para que el humano tenga documentación legible.

### 2a. Topic Keys para CADA artifact

```yaml
# .zyro/conventions.yaml — Topic keys registry
topic_keys:
  # === Por proyecto ===
  project:
    context:        "zyro/{project}/context"
    investigation:  "zyro/{project}/investigation"
    planning:       "zyro/{project}/planning"
    doc_index:      "zyro/{project}/doc-index"
    architecture:   "zyro/{project}/architecture"
    changelog:      "zyro/{project}/changelog"

  # === Por cambio (SDD flow) ===
  change:
    explore:        "sdd/{change}/explore"
    proposal:       "sdd/{change}/proposal"
    spec:           "sdd/{change}/spec"
    design:         "sdd/{change}/design"
    tasks:          "sdd/{change}/tasks"
    apply_progress: "sdd/{change}/apply-progress"
    verify_report:  "sdd/{change}/verify-report"
    archive_report: "sdd/{change}/archive-report"

  # === Graphify ===
  graph:
    report:         "zyro/{project}/graph-report"
    last_diff:      "zyro/{project}/graph-diff"
```

**Ejemplo de uso**:

```
Proyecto: zyro/zyroagentcli/
├── context       → zyro/zyroagentcli/context          (stack, lenguaje, AGENT.md)
├── investigation → zyro/zyroagentcli/investigation    (resultado Macro 1)
├── planning      → zyro/zyroagentcli/planning         (features, timeline)
│
Cambio: sdd/scheduler-harness/
├── explore       → sdd/scheduler-harness/explore
├── proposal      → sdd/scheduler-harness/proposal
├── spec          → sdd/scheduler-harness/spec
├── design        → sdd/scheduler-harness/design
├── tasks         → sdd/scheduler-harness/tasks
├── apply         → sdd/scheduler-harness/apply-progress
├── verify        → sdd/scheduler-harness/verify-report
└── archive       → sdd/scheduler-harness/archive-report
```

### 2b. Formato estándar de entry en Engram

 TODOS los sub-agentes usan ESTE formato al guardar:

```markdown
## {Título descriptivo}
- **project**: {project}
- **change**: {change-name} (si aplica)
- **artifact**: {explore|proposal|spec|design|tasks|apply|verify|archive}
- **timestamp**: {ISO datetime}
- **status**: {draft|review|approved|archived}

### What
[qué se hizo en esta fase]

### Why
[por qué se hizo así, qué alternativas se descartaron]

### Where
[archivos/paths afectados]

### Decisions
[decisiones clave tomadas con rationale]

### Next
[siguiente paso recomendado]
```

**Regla de campo `artifact`**: Es OBLIGATORIO. Permite filtrar por tipo en `mem_search`. Ejemplo: `mem_search(query="", project="zyroagentcli", type="architecture")` devuelve todos los artifacts.

### 2c. Protocolo de búsqueda estructurada

```
SUB-AGENTE necesita buscar algo en Engram:

1. CONOCE topic_key exacta?
   → mem_get_observation(id)
   → Fast path, respuesta inmediata

2. NO conoce topic_key?
   → mem_search(query, project="zyroagentcli", type="architecture")
   → Usa keywords del contexto actual

3. mem_search devuelve vacío?
   → mem_context(project="zyroagentcli")
   → Busca en historial reciente de sesión

4. Todo falla?
   → PREGUNTAR AL HUMANO
   → "No encontré X en la memoria. ¿Dónde guardaste esto?"
```

**Filtros obligatorios**:
- `project` = SIEMPRE requerido (nunca buscar cross-project)
- `type` = Opcional pero recomendado (architecture|decision|bugfix|pattern)

### 2d. zyro-doc-search tool

```go
// internal/doc/search.go
package doc

import (
    "fmt"
    "strings"
)

type SearchRequest struct {
    TopicKey string // exact key (fast path) — opcional
    Query    string // NLP query (slow path) — opcional
    Type     string // artifact type filter — opcional
    Project  string // REQUIRED
}

type SearchResult struct {
    ID        int
    TopicKey  string
    Content   string
    Artifact  string
    Status    string
    Timestamp string
    Source    string // "topic_key" | "search" | "context"
}

// Search implements the structured search protocol.
// 1. Try topic_key → mem_get_observation
// 2. Fallback → mem_search con filtros
// 3. Fallback → mem_context
// 4. Return structured result or error
func (s *DocService) Search(req SearchRequest) (*SearchResult, error) {
    if req.Project == "" {
        return nil, fmt.Errorf("search: project is required")
    }

    // Fast path: exact topic_key
    if req.TopicKey != "" {
        result, err := s.engram.GetObservationByKey(req.TopicKey)
        if err == nil && result != nil {
            return &SearchResult{
                ID:        result.ID,
                TopicKey:  req.TopicKey,
                Content:   result.Content,
                Source:    "topic_key",
                Timestamp: result.Created,
            }, nil
        }
        // Fall through to search
    }

    // Slow path: mem_search with filters
    if req.Query != "" {
        results, err := s.engram.Search(req.Query, req.Project, req.Type)
        if err == nil && len(results) > 0 {
            best := results[0]
            return &SearchResult{
                ID:        best.ID,
                TopicKey:  best.TopicKey,
                Content:   best.Content,
                Source:    "search",
                Timestamp: best.Created,
            }, nil
        }
    }

    // Last resort: recent context
    ctxResults, err := s.engram.Context(req.Project)
    if err == nil && len(ctxResults) > 0 {
        best := ctxResults[0]
        return &SearchResult{
            ID:        best.ID,
            TopicKey:  best.TopicKey,
            Content:   best.Content,
            Source:    "context",
            Timestamp: best.Created,
        }, nil
    }

    return nil, fmt.Errorf("search: no results found for project=%s query=%s",
        req.Project, req.Query)
}

// BuildTopicKey construye una topic_key desde componentes
func BuildTopicKey(scope, project, artifact, change string) string {
    if scope == "project" {
        return fmt.Sprintf("zyro/%s/%s", project, artifact)
    }
    return fmt.Sprintf("sdd/%s/%s", change, artifact)
}
```

**Nota**: `DocService` es un wrapper delgado sobre las primitivas de Engram (`mem_save`, `mem_search`, `mem_get_observation`, `mem_context`). NO reimplementa Engram — lo orquesta.

### 2e. Eliminación de openspec

| What | Before | After |
|------|--------|-------|
| Propuesta | `openspec/changes/{change}/proposal.md` | `mem_save(topic_key: "sdd/{change}/proposal")` |
| Spec | `openspec/specs/{domain}/spec.md` | `mem_save(topic_key: "sdd/{change}/spec")` |
| Design | `openspec/changes/{change}/design.md` | `mem_save(topic_key: "sdd/{change}/design")` |
| Tasks | `openspec/changes/{change}/tasks.md` | `mem_save(topic_key: "sdd/{change}/tasks")` |
| Export | manual sync | `zyro-doc-export` genera .md desde Engram |

**Flujo corregido (Macro 3, PAR 1)**:

```
ANTES:
  sdd-propose → mem_save + openspec/changes/{change}/proposal.md

AHORA:
  zyro-sdd-propose → mem_save(
      topic_key: "sdd/{change}/proposal",
      title: "Proposal: {change}",
      type: "architecture",
      project: "{project}",
      content: "{formato estándar}"
  )
  → NO crea archivos en openspec/
```

## 3. Macro 3 — Sub-agentes con prefijo zyro-

### 3a. Mapeo de renombramiento

```
Nombre actual      →  Nuevo nombre            →  Skill path
sdd-explore        →  zyro-sdd-explore        →  skills/zyro-sdd-explore/SKILL.md
sdd-propose        →  zyro-sdd-propose        →  skills/zyro-sdd-propose/SKILL.md
sdd-spec           →  zyro-sdd-spec           →  skills/zyro-sdd-spec/SKILL.md
sdd-design         →  zyro-sdd-design         →  skills/zyro-sdd-design/SKILL.md
sdd-tasks          →  zyro-sdd-tasks          →  skills/zyro-sdd-tasks/SKILL.md
sdd-apply          →  zyro-sdd-implement      →  skills/zyro-sdd-implement/SKILL.md
sdd-verify         →  zyro-sdd-verify         →  skills/zyro-sdd-verify/SKILL.md
sdd-archive        →  zyro-sdd-archive        →  skills/zyro-sdd-archive/SKILL.md
```

### 3b. Implementación del wrapping

Cada `zyro-sdd-*` es un skill wrapper que:

1. **Inyecta** topic keys del proyecto desde `.zyro/conventions.yaml`
2. **Inyecta** reglas de búsqueda Engram (protocolo 2c)
3. **Inyecta** el proyecto y change-name como contexto
4. **Delega** al sub-agente SDD original
5. **Persiste** resultado con formato Engram estándar (2b)

**Ejemplo**: `zyro-sdd-explore/SKILL.md`

```markdown
---
name: zyro-sdd-explore
description: "Wrapper de sdd-explore con contexto Zyro. Explora código para un cambio."
---

## Pre-flight: Inyectar contexto

ANTES de delegar a sdd-explore, inyectá:

1. **Topic keys** del proyecto:
   - `zyro/{project}/context` — stack, lenguaje
   - `zyro/{project}/investigation` — research previa
   - `sdd/{change}/explore` — donde persistir resultado

2. **Formato Engram estándar** (ver §2b):
   - project, change, artifact, timestamp, status
   - Secciones: What, Why, Where, Decisions, Next

3. **Protocolo de búsqueda** (ver §2c):
   - Primero topic_key exacta
   - Si no, mem_search con project obligatorio
   - Si no, mem_context
   - Si nada, preguntar al humano

## Delegación

Delegar a sub-agente `sdd-explore` con:
- change-name: {change}
- project: {project}
- topic_key: sdd/{change}/explore
- format: engram-standard

## Post-flight: Persistir resultado

Cuando sdd-explore retorne:
1. Formatear resultado con template §2b
2. Llamar mem_save con topic_key correcta
3. Actualizar doc-index si existe
```

**Ejemplo**: `zyro-sdd-design/SKILL.md`

```markdown
---
name: zyro-sdd-design
description: "Wrapper de sdd-design con contexto Zyro. Diseña arquitectura para un cambio."
---

## Pre-flight

1. Buscar engram: `sdd/{change}/proposal` (debe existir)
2. Buscar engram: `sdd/{change}/spec` (debe existir)
3. Leer código real del proyecto
4. Inyectar topic_key: `sdd/{change}/design`

## Delegar a sdd-design

Sub-agente original con contexto inyectado.

## Post-flight

Persistir con formato Engram estándar:
- topic_key: sdd/{change}/design
- artifact: design
- status: draft
```

### 3c. Invocación desde el orquestador

```
ANTES (en AGENT.md / Zyro Agent):
  skill("sdd-explore")
  skill("sdd-propose")
  skill("sdd-spec")
  ...

AHORA:
  skill("zyro-sdd-explore")
  skill("zyro-sdd-propose")
  skill("zyro-sdd-spec")
  ...
```

El Zyro Agent NUNCA invoca `sdd-*` directamente. Siempre pasa por `zyro-sdd-*`.

## 4. Macro 4 — Graphify update después de archive

### 4a. Flujo actualizado

```
E4: Zyro Agent (orquestador)
  │
  ├─①─ work-unit commits
  │   → Cada task completada = 1 commit
  │   → Formato: "feat(scope): description" (conventional commits)
  │
  ├─②─ PR creation
  │   → Si >400 líneas → chained-pr skill
  │   → branch-pr skill para PR creation
  │   → PR description desde tasks.md
  │
  ├─③─ zyro-doc-index
  │   → internal/doc/index.GenerateIndex("docs/")
  │   → Produce: .zyro/doc-index.yaml
  │
  ├─④─ zyro-doc-sync
  │   → internal/doc/sync.Sync("docs/", ".zyro/docs/")
  │   → Actualiza doc-index
  │
  ├─⑤─ zyro-doc-export
  │   → internal/doc/export.Export("markdown")
  │   → Produce: ARCHITECTURE.md (si aplica)
  │   → Produce: CHANGELOG.md (si aplica)
  │   → ARCHITECTURE.md incluye enlace al último grafo:
  │     "## Graph\nVer [graph.json](.zyro/graph.json) | Último diff: {date}"
  │
  ├─⑥─ NUEVO: Graphify update
  │   → internal/doc/graphify.UpdateGraph(root)
  │   → Flujo:
  │     a) Ejecutar graphify sobre el proyecto
  │     b) Comparar con grafo anterior (diff)
  │     c) Si hay cambios estructurales significativos:
  │        - Actualizar GRAPH_REPORT.md
  │        - Actualizar graph.json
  │        - mem_save: "graphify updated - structural changes detected"
  │          topic_key: zyro/{project}/graph-diff
  │     d) Si NO hay cambios estructurales:
  │        - Skippear (ahorra tokens)
  │        - Solo log: "graphify: no structural changes, skipped"
  │
  └─⑦─ Engram final
       → mem_save(
           topic_key: "sdd/{change}/archive-report",
           title: "Archive: {change}",
           type: "architecture",
           project: "{project}",
           content: "{resumen del cambio completo}"
         )
```

### 4b. Graphify update — lógica de diff

```go
// internal/doc/graphify.go
package doc

type GraphDiff struct {
    NodesAdded    int
    NodesRemoved  int
    EdgesAdded    int
    EdgesRemoved  int
    Significant   bool // true si >10% cambio
}

func (s *DocService) UpdateGraph(root string) (*GraphDiff, error) {
    // 1. Cargar grafo anterior (si existe)
    prevGraph := s.loadPreviousGraph(root)

    // 2. Ejecutar graphify
    currentGraph, err := s.graphify.Analyze(root)
    if err != nil {
        return nil, fmt.Errorf("graphify update: %w", err)
    }

    // 3. Calcular diff
    diff := s.computeDiff(prevGraph, currentGraph)

    // 4. Decidir si actualizar
    if !diff.Significant {
        return diff, nil // skip — ahorra tokens
    }

    // 5. Persistir grafo actualizado
    s.writeGraphReport(root, currentGraph, diff)
    s.writeGraphJSON(root, currentGraph)

    // 6. Registrar en Engram
    s.engram.Save(EngramEntry{
        Title:    "graphify updated",
        TopicKey: fmt.Sprintf("zyro/%s/graph-diff", s.project),
        Type:     "architecture",
        Content:  fmt.Sprintf("Structural changes: +%d/-%d nodes, +%d/-%d edges", diff.NodesAdded, diff.NodesRemoved, diff.EdgesAdded, diff.EdgesRemoved),
    })

    return diff, nil
}

func (d *GraphDiff) IsSignificant() bool {
    totalChanges := d.NodesAdded + d.NodesRemoved + d.EdgesAdded + d.EdgesRemoved
    return totalChanges > 5 // umbral configurable
}
```

### 4c. ARCHITECTURE.md con enlace al grafo

```markdown
# Architecture: {project}

## Components
{componentes desde Engram}

## Data Flow
{flujo desde Engram}

## Graph
> Último análisis: {date}
> Cambios desde última revisión: {diff summary}
> Ver [graph.json](.zyro/graph.json) | [GRAPH_REPORT.md](.zyro/GRAPH_REPORT.md)
```

## 5. Orquestador como guía

### 5a. Protocolo de diálogo entre fases

El orquestador (Zyro Agent) NO es un simple "approve? (y/n)". Es un DIÁLOGO CON CONTEXTO:

```
ENTRE CADA FASE:

1. MOSTRAR RESULTADO
   "Fase completada: {fase}. Resumen: {resumen en 2-3 líneas}"

2. RECOMENDACIÓN
   "Basado en lo que investigué, sugeriría: {recomendación específica}"

3. ADVERTENCIA DE RIESGO
   "Esto podría ser complejo porque: {razón técnica}"

4. PREGUNTAR
   "¿Querés ajustar algo o continuamos?"

5. SOLO AVANZA si el humano dice sí
```

### 5b. Ejemplo concreto de diálogo

```
─── Fase: Investigación (Macro 1) completada ───

Resumen: Encontré 3 skills relevantes (go-testing, branch-pr, chained-pr).
El context bridge detectó que usás Go 1.22 con Cobra. No hay MCP server
disponible, fallback a GitMCP.

Recomendación: Sugiero empezar por el PR1 (skill-advisor + context bridge)
porque son dependencias de todo lo demás. Sin estos, el pipeline no puede
evaluar skills.

Riesgo: El context bridge depende de Neuledge Context CLI que podría no
estar instalado. El fallback a GitMCP funciona pero es más lento.

¿Querés ajustar algo o continuamos con planificación?

─── Humano: "Continuamos" ───

─── Fase: Planificación (Macro 2) completada ───

Resumen: 6 features atómicas, ordenadas por dependencias.
PR1 (skill+context) → PR2 (scheduler) → PR3 (SDD pairs) → PR4-6.

Recomendación: El PR3 (SDD pairs) es el más grande (~350 líneas).
Sugiero dividirlo en PR3a (explore+propose) y PR3b (spec+design+tasks)
para mantener diff ≤400.

Riesgo: El PR4 (Macro 1 investigation) tiene dependencia externa
(Context CLI). Si no está disponible, el flujo degradado funciona
pero pierde documentación automática.

¿Querés ajustar algo o continuamos con implementación?
```

### 5c. Harness de validación — cambios al scheduler

```go
// internal/scheduler/approval.go

type GuidedApproval struct {
    Phase      Phase
    Summary    string  // resumen de la fase
    Recommend  string  // recomendación del agente
    Risk       string  // advertencia de riesgo
    AskPrompt  string  // pregunta al humano
}

func (g *GuidedApproval) PromptApproval() (bool, error) {
    // 1. Mostrar resultado
    fmt.Printf("\n─── Fase: %s completada ───\n\n", g.Phase)
    fmt.Printf("Resumen: %s\n\n", g.Summary)

    // 2. Recomendación
    if g.Recommend != "" {
        fmt.Printf("Recomendación: %s\n\n", g.Recommend)
    }

    // 3. Riesgo
    if g.Risk != "" {
        fmt.Printf("Riesgo: %s\n\n", g.Risk)
    }

    // 4. Preguntar
    prompt := g.AskPrompt
    if prompt == "" {
        prompt = "¿Querés ajustar algo o continuamos?"
    }
    fmt.Printf("%s (s/n/detalle): ", prompt)

    // 5. Leer input
    var input string
    fmt.Scanln(&input)

    switch strings.ToLower(strings.TrimSpace(input)) {
    case "s", "si", "sí", "y", "yes":
        return true, nil
    case "n", "no":
        return false, nil
    case "d", "detalle":
        g.showDetails()
        return g.PromptApproval() // recursive retry
    default:
        fmt.Println("Respuesta no reconocida. Usá 's' (sí), 'n' (no), o 'd' (detalle).")
        return g.PromptApproval()
    }
}

func (g *GuidedApproval) showDetails() {
    // Muestra output completo del agente para esta fase
    fmt.Printf("\n─── Detalle de %s ───\n", g.Phase)
    fmt.Printf("%s\n\n", g.FullOutput)
}
```

### 5d. Integración con el flujo

```
E1: Zyro Agent
  ├─①─⑤  (investigación — sin cambio)
  ├─⑥─ GuidedApproval{
  │      Phase: "Macro 1",
  │      Summary: "Skills: 3, Docs: disponibles, Bridge: fallback",
  │      Recommend: "Empezar por PR1 (skill+context)",
  │      Risk: "Context CLI puede no estar instalado",
  │      AskPrompt: "¿Continuamos con planificación?",
  │    }
  └─⑦─ Si aprueba → Macro 2
       Si rechaza → abort

E2: Zyro Agent
  ├─①─④  (planificación — sin cambio)
  ├─⑤─ GuidedApproval{
  │      Phase: "Macro 2",
  │      Summary: "6 features, PR1→PR6",
  │      Recommend: "Dividir PR3 en PR3a+PR3b",
  │      Risk: "PR4 depende de Context CLI externo",
  │      AskPrompt: "¿Continuamos con implementación?",
  │    }
  └─⑥─ Si aprueba → Macro 3
       Si rechaza → abort
```

## 6. Scheduler Harness (Refactor)

### Decisión: Scheduler Go como Harness, NO como Orquestador

**Choice**: El scheduler se refactorea para ser un "harness" que:
1. Valida que cada fase del agente se completó correctamente
2. Bloquea sin validación humana (guiada, no solo y/n)
3. NO ejecuta la lógica de las fases (eso lo hace el agente)

**Alternatives rejected**:
- Scheduler como orquestador completo: Rígido, no puede investigar ni adaptarse
- Agente como único orquestador: Sin validación determinística, puede alucinar

**Rationale**: Híbrido. El agente tiene la flexibilidad para investigar, adaptar, tomar decisiones. El scheduler tiene la rigidez para verificar que el output del agente cumple el contrato. La guía (Ajuste 4) agrega contexto al approval para que el humano tome decisiones informadas.

### Nuevo Diseño de phase_stubs.go

```go
type MacroPhase struct {
    Name        Phase
    AgentFunc   func(ctx context.Context, cfg *Config) (*Result, error)
    Validator   func(result *Result) error
    Approval    *GuidedApproval // NUEVO: guía contextual
}

type F1InvestigationRunner struct {
    SkillAdvisor *skilladvisor.Registry
    Bridge       *context.Bridge
}

func (r *F1InvestigationRunner) Run(ctx context.Context, cfg *Config) (*Result, error) {
    payload, err := handoff.Parse("handoff.yaml")
    recs, err := r.SkillAdvisor.Recommend(query, 5)
    docs, err := r.Bridge.QueryDocs(libID, query)
    return &Result{
        Phase:   PhaseF1,
        Status:  StatusSuccess,
        Summary: fmt.Sprintf("Skills: %d, Docs: %d bytes", len(recs), len(docs)),
    }, nil
}
```

## 7. Doc Tools Integration

### .zyro/conventions.yaml completo

```yaml
# Topic keys para Engram
topic_keys:
  project:
    context:        "zyro/{project}/context"
    investigation:  "zyro/{project}/investigation"
    planning:       "zyro/{project}/planning"
    doc_index:      "zyro/{project}/doc-index"
    architecture:   "zyro/{project}/architecture"
    changelog:      "zyro/{project}/changelog"
  change:
    explore:        "sdd/{change}/explore"
    proposal:       "sdd/{change}/proposal"
    spec:           "sdd/{change}/spec"
    design:         "sdd/{change}/design"
    tasks:          "sdd/{change}/tasks"
    apply_progress: "sdd/{change}/apply-progress"
    verify_report:  "sdd/{change}/verify-report"
    archive_report: "sdd/{change}/archive-report"
  graph:
    report:         "zyro/{project}/graph-report"
    last_diff:      "zyro/{project}/graph-diff"

# Formato estándar de entry
entry_format:
  required_fields:
    - project
    - artifact
    - timestamp
    - status
  optional_fields:
    - change
  sections:
    - What
    - Why
    - Where
    - Decisions
    - Next

# Protocolo de búsqueda
search_protocol:
  fast_path: "topic_key → mem_get_observation"
  slow_path: "mem_search(query, project, type)"
  fallback:  "mem_context(project)"
  last_resort: "preguntar al humano"

# Graphify
graphify:
  auto_update: true
  significant_threshold: 5  # cambios para considerar significativo
  skip_if_no_change: true

# Conveniones de código
conventions:
  - type: code
    pattern: "**/*.go"
    rule: "Use table-driven tests"
    severity: must
  - type: doc
    pattern: "docs/**/*.md"
    rule: "Include ## Purpose section"
    severity: should
  - type: review
    pattern: "**/*.go"
    rule: "No hardcoded credentials"
    severity: must
```

### Flujo de doc tools (actualizado)

```
Post-Macro 4:
  zyro-doc-index
    → internal/doc/index.GenerateIndex("docs/")
    → Produce: .zyro/doc-index.yaml

  zyro-doc-search (on demand)
    → internal/doc/search.Search(req)
    → Protocolo: topic_key → mem_search → mem_context → preguntar

  zyro-doc-sync (post-Macro 4 + on demand)
    → internal/doc/sync.Sync("docs/", ".zyro/docs/")

  zyro-doc-export (post-Macro 4)
    → internal/doc/export.Export("markdown")
    → Produce: ARCHITECTURE.md (con enlace a graph.json), CHANGELOG.md

  NUEVO: zyro-graphify-update (post-archive)
    → internal/doc/graphify.UpdateGraph(root)
    → Si hay diff significativo: actualizar graph.json + GRAPH_REPORT.md
    → Si no: skip (ahorra tokens)
    → Siempre: mem_save en zyro/{project}/graph-diff
```

## 8. File Changes Summary (actualizado)

| File | Action | PR | Description |
|------|--------|----|---------|
| `internal/skilladvisor/registry.go` | Modify | 1 | Add Load(), SkillEntry with Tags/Publisher/Verified |
| `internal/skilladvisor/score.go` | Modify | 1 | Weighted scoring, Recommend(), ScoredSkill |
| `internal/skilladvisor/discover.go` | Modify | 1 | DiscoverClient with HTTP cache |
| `internal/context/bridge.go` | Modify | 1 | Full Bridge with Start/Stop/QueryDocs/Resolve |
| `internal/scheduler/phase_stubs.go` | Modify | 2 | F1→F4 real runners with GuidedApproval |
| `internal/scheduler/approval.go` | Modify | 2 | GuidedApproval dialog (resumen+recomendación+riesgo) |
| `internal/scheduler/scheduler.go` | Modify | 2 | HarnessValidator, AgentBridge |
| `internal/spec/compile.go` | Modify | 2 | CIO → Engram key emission |
| `internal/apply/runner.go` | Modify | 3 | Goroutine pool implementation |
| `internal/test/contracts.go` | Modify | 3 | Given/When/Then executor |
| `internal/test/report.go` | Modify | 3 | Graphify diff report |
| `internal/doc/index.go` | Create | 4 | GenerateIndex() |
| `internal/doc/search.go` | Create | 4 | Search() — Engram-first protocol |
| `internal/doc/sync.go` | Create | 4 | Sync() |
| `internal/doc/export.go` | Create | 4 | Export() — genera .md desde Engram |
| `internal/doc/conventions.go` | Create | 4 | LoadConventions() |
| `internal/doc/graphify.go` | Create | 4 | UpdateGraph() — graphify refresh post-archive |
| `cmd/zyrocli/doc.go` | Create | 4 | zyrocli doc sync/search |
| `.zyro/conventions.yaml` | Create | 1 | Topic keys, search protocol, graphify config |
| `skills/zyro-sdd-explore/SKILL.md` | Create | 3 | Wrapper: sdd-explore + Zyro context |
| `skills/zyro-sdd-propose/SKILL.md` | Create | 3 | Wrapper: sdd-propose + Zyro context |
| `skills/zyro-sdd-spec/SKILL.md` | Create | 3 | Wrapper: sdd-spec + Zyro context |
| `skills/zyro-sdd-design/SKILL.md` | Create | 3 | Wrapper: sdd-design + Zyro context |
| `skills/zyro-sdd-tasks/SKILL.md` | Create | 3 | Wrapper: sdd-tasks + Zyro context |
| `skills/zyro-sdd-implement/SKILL.md` | Create | 3 | Wrapper: sdd-apply + Zyro context |
| `skills/zyro-sdd-verify/SKILL.md` | Create | 3 | Wrapper: sdd-verify + Zyro context |
| `skills/zyro-sdd-archive/SKILL.md` | Create | 3 | Wrapper: sdd-archive + Zyro context |
| `AGENT.md` | Modify | 1 | Macro-fases, zyro-sdd-* invocations |
| `handoff.yaml` | Modify | 1 | Unify user_story (singular) |
| `go.mod` | Modify | 1 | Unify Go version |

## 9. Key Decisions & Tradeoffs (actualizado)

### Decision 1: Hybrid Orchestration

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Scheduler Go leads | Deterministic but rigid | ❌ |
| Agent leads | Flexible but can hallucinate | ❌ |
| **Hybrid** | **Best of both, integration complexity** | **✅** |

### Decision 2: Engram-only Documentation (ACTUALIZADO)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Engram + openspec | Double maintenance, confusing | ❌ |
| Openspec only | Structured but not cross-session | ❌ |
| **Engram pure + doc-export** | **Single source, export to .md for humans** | **✅** |

Engram es la fuente de verdad. Los .md se generan on-demand para humanos. NO hay sincronización bidireccional.

### Decision 3: C-I-O DSL

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Implement compile.go | Auto-generates OpenAPI/protobuf | ❌ |
| Eliminate C-I-O | Less code, loses formal spec | ❌ |
| **C-I-O as documentation** | **Structure without complexity** | **✅** |

### Decision 4: Skill Advisor

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Go native | Fast, deterministic, testable | ✅ base |
| Agent-based | Can understand context | ❌ alone |
| **Go base + agent refinement** | **Speed + intelligence** | **✅** |

### Decision 5: Context Bridge Protocol

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Direct HTTP to Context7 | Simple but tight coupling | ❌ |
| MCP subprocess via os/exec | Decoupled, testable | ✅ |
| Embedded Context client | No subprocess, but dependency | ❌ |

### Decision 6: zyro-sdd-* Wrapping (NUEVO)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Modificar skills SDD existentes | rompe compatibilidad con otros proyectos | ❌ |
| Skills zyro-* independientes | duplica lógica | ❌ |
| **Wrapper skills que inyectan contexto** | **no rompe SDD, agrega valor Zyro** | **✅** |

Los skills SDD existentes funcionan independientemente. Los wrappers zyro-* agregan:
- Topic keys del proyecto
- Protocolo de búsqueda Engram
- Formato estándar de persistencia

### Decision 7: Graphify post-archive (NUEVO)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Graphify en cada cambio | Muchos tokens, lento | ❌ |
| Graphify manual | Olvidable, inconsistente | ❌ |
| **Graphify post-archive con diff** | **Solo actualiza si hay cambios significativos** | **✅** |

### Decision 8: Orchestrator as Guide (NUEVO)

| Option | Tradeoff | Decision |
|--------|----------|----------|
| approve? (y/n) | Simple pero ciego | ❌ |
| Full auto sin approval | Rápido pero peligroso | ❌ |
| **Guided dialog** | **Más tokens, decisiones informadas** | **✅** |

El orquestador muestra: resumen + recomendación + riesgo + pregunta. El humano decide con contexto.

## 10. Migration / Rollout

No migration required. Los topic keys en Engram se crean on-demand cuando el flujo corre por primera vez. Los skills zyro-sdd-* se crean como archivos nuevos en `skills/`. El `.zyro/conventions.yaml` se genera en el primer `zyrocli init`.

## 11. Open Questions

- [ ] ¿El umbral de "cambio significativo" para graphify (5 cambios) es suficiente? ¿O necesita ser configurable por proyecto?
- [ ] ¿Los wrappers zyro-sdd-* deben incluir validación de pre-requisitos (ej: "¿existe proposal antes de spec?") o solo inyectar contexto?
- [ ] ¿El diálogo del orquestador debe ser serializable (guardar en Engram) para auditoría?
- [ ] ¿Qué pasa si el humano responde "detalle" — ¿cuánta información mostrar? ¿El output completo del sub-agente?
