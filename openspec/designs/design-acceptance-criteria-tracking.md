# Acceptance Criteria Tracking — Diseño Técnico

> **Documento**: Design Técnico  
> **Estado**: Draft  
> **Fase**: F2 (Design)  
> **Basado en**: Spec `spec-acceptance-criteria-tracking.md` | Código actual del pipeline SDD  

---

## Índice

1. [Resumen ejecutivo](#1-resumen-ejecutivo)
2. [Estado actual — Línea base](#2-estado-actual--línea-base)
3. [Arquitectura propuesta](#3-arquitectura-propuesta)
4. [Componentes de diseño](#4-componentes-de-diseño)
5. [Diagrama de datos](#5-diagrama-de-datos)
6. [Interfaces entre componentes](#6-interfaces-entre-componentes)
7. [Archivos a modificar](#7-archivos-a-modificar)
8. [Decisiones de diseño detalladas](#8-decisiones-de-diseño-detalladas)
9. [Flujo de datos paso a paso](#9-flujo-de-datos-paso-a-paso)
10. [Pruebas](#10-pruebas)
11. [Riesgos y mitigaciones](#11-riesgos-y-mitigaciones)
12. [Orden de implementación](#12-orden-de-implementación)

---

## 1. Resumen Ejecutivo

### Problema

Los acceptance criteria se definen en texto libre en propuestas y specs (markdown), pero **no viajan entre fases**. No se persisten en HelixDB, no se verifican automáticamente, y no hay un gate que impida avanzar si no se cumplieron.

### Solución

Crear un tipo `AcceptanceCriteria` first-class en el paquete `boomerang`, agregarlo a `TaskSpec`, `TaskRow`, `PhaseResult`, y `Result` del scheduler. Esto permite que los criteria fluyan desde F1 hasta F4 a través de:

1. **HelixDB** — persistencia en nodos Task
2. **PhaseResult** — retorno de Boomerang al scheduler
3. **Scheduler.Result** — entrada a ApprovalGate y writeHandoff
4. **Handoff payload** — serialización YAML para la siguiente fase

### Principios de diseño

- **Backward compatibility**: `omitempty` en todos los campos, slice vacío = auto-pass
- **Sin dependencias circulares**: `AcceptanceCriteria` en `boomerang`, `TaskRow` usa `[]map[string]any`
- **Evaluación progresiva**: primera versión = task success + output no vacío
- **Cero ruptura**: fases existentes sin criteria siguen igual

---

## 2. Estado actual — Línea base

### 2.1 Structs existentes (sin acceptance criteria)

| Struct | Paquete | Archivo | Estado actual |
|--------|---------|---------|---------------|
| `TaskSpec` | `boomerang` | `orchestrator.go:59-66` | 5 campos: ID, Name, Description, Agent, Tags, DependsOn |
| `TaskRow` | `helix` | `types.go:21-28` | 6 campos: ID, Name, Description, Phase, Status, CreatedAt |
| `PhaseResult` | `boomerang` | `orchestrator.go:23-35` | 12 campos, sin criteria |
| `Result` | `scheduler` | `phase.go:49-55` | 4 campos: Phase, Status, Summary, Error, MemoryContext |
| `UserStory` | `handoff` | `payload.go:24-27` | `Acceptance string` como texto libre |
| `Payload` | `handoff` | `payload.go:67-79` | Sin acceptance summary |

### 2.2 Flujo actual (sin criteria)

```
Spec (markdown)
  │
  ▼
Design (markdown) ──→ Task nodes (HelixDB, sin criteria)
  │
  ▼
Apply + Verify ──→ QualityStep (solo go build) ──→ verify-report.md
  │
  ▼
ApprovalGate (sin criteria check) ──→ Handoff (sin criteria)
```

### 2.3 Hallazgos del codebase

1. `saveTaskToHelix(taskID, name, agent, phase string)` — sin parámetro de criteria
2. `ApprovalGate(phase Phase, summary string)` — sin parámetro de criteria
3. `writeHandoff(phaseName string, result *Result, nextPhase Phase)` — sin criteria
4. `PhaseConfigV2` no tiene campo para pasar criteria entre fases
5. `MemoryStep` solo retorna texto de memoria causal, no criteria estructurados

---

## 3. Arquitectura Propuesta

### 3.1 Diagrama de flujo de Acceptance Criteria

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Acceptance Criteria Flow                             │
├──────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  F1: Spec                                                                    │
│   │  zyro-sdd-spec define criteria en Spec node (HelixDB)                    │
│   │  Ej: [AC-001: "El middleware debe rechazar requests sin JWT"]            │
│   ▼                                                                          │
│  F2: Design + Tasks                                                          │
│   │  zyro-sdd-tasks asigna criteria a Task nodes (HelixDB)                   │
│   │  Task.acceptance_criteria = [AC-001, AC-002]                             │
│   ▼                                                                          │
│  F3: Boomerang.RunPhase                                                      │
│   │                                                                          │
│   │  ┌─ MemoryStep ──► Recall criteria from HelixDB                          │
│   │  ├─ ThinkStep  ──► Inject criteria into TaskSpec                         │
│   │  ├─ DelegateStep ► Agents execute tasks with criteria context            │
│   │  ├─ QualityStep ─► evaluateCriteria() against delegate results           │
│   │  └─ SaveStep   ──► Persist updated criteria status to HelixDB            │
│   │                                                                          │
│   │  PhaseResult.AcceptanceCriteria = [...criteria con status actualizado]   │
│   ▼                                                                          │
│  Scheduler (scheduler.go)                                                    │
│   │                                                                          │
│   │  1. Extrae CriteriaSummary de PhaseResult                                 │
│   │  2. Pasa a ApprovalGate → bloquea si Failed > 0                          │
│   │  3. Pasa a writeHandoff → incluye tabla de criteria                      │
│   ▼                                                                          │
│  F4: Archive                                                                 │
│   │  ApprovalGate final verifica criteria                                     │
│   │  Handoff incluye resumen de criteria status                               │
│   │  Payload.AcceptanceStatus registra resumen global                         │
│   ▼                                                                          │
│  Done.                                                                       │
│                                                                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Decisiones arquitecturales clave

| Decisión | Opción elegida | Alternativa descartada |
|----------|---------------|----------------------|
| Dónde definir `AcceptanceCriteria` | Paquete `boomerang` (nuevo archivo `criteria.go`) | Paquete `helix` (crearía dependencia circular) |
| Cómo serializar a HelixDB | `[]map[string]any` en `TaskRow` | `[]AcceptanceCriteria` (importaría boomerang → helix) |
| Cómo pasa criteria entre fases | `PhaseResult.AcceptanceCriteria` + `Result.AcceptanceCriteria` | Query a HelixDB post-fase (más lento, menos preciso) |
| Cómo se actualiza status en HelixDB | MCP server + `complete_task` tool | sdd-verify escribe directo (problema de permisos) |
| Cómo se muestra en ApprovalGate | `CriteriaSummary` pasado como parámetro | Query inline en ApprovalGate (dependencia externa) |

---

## 4. Componentes de Diseño

### 4.1 `AcceptanceCriteria` struct

**Archivo**: `internal/boomerang/criteria.go` (NUEVO)

```go
package boomerang

// CriteriaStatus representa el estado de verificación de un acceptance criterion.
type CriteriaStatus string

const (
    CriteriaPending  CriteriaStatus = "pending"   // definido pero no evaluado
    CriteriaVerified CriteriaStatus = "verified"  // evaluación exitosa
    CriteriaFailed   CriteriaStatus = "failed"    // evaluación fallida
)

// AcceptanceCriteria representa un único criterion de aceptación.
// Viaja desde F1 (Spec) hasta F4 (Archive) a través del pipeline.
type AcceptanceCriteria struct {
    ID          string         `json:"id" yaml:"id"`
    Description string         `json:"description" yaml:"description"`
    Phase       string         `json:"phase" yaml:"phase"`                   // fase donde se definió
    Status      CriteriaStatus `json:"status" yaml:"status"`
    Source      string         `json:"source" yaml:"source"`                 // "proposal", "spec", "design"
    TaskID      string         `json:"task_id,omitempty" yaml:"task_id,omitempty"` // task que lo implementa
}

// CriteriaSummary proporciona un resumen agregado de acceptance criteria.
type CriteriaSummary struct {
    Total    int `json:"total"`
    Pending  int `json:"pending"`
    Verified int `json:"verified"`
    Failed   int `json:"failed"`
}

// NewCriteriaSummary computa un CriteriaSummary a partir de un slice de criteria.
func NewCriteriaSummary(criteria []AcceptanceCriteria) *CriteriaSummary { ... }
```

**Responsabilidad**: Representar y resumir acceptance criteria. Es el tipo canónico que viaja por todo el pipeline.

### 4.2 `TaskSpec` — Campo AcceptanceCriteria

**Archivo**: `internal/boomerang/orchestrator.go` (modificar)

```go
type TaskSpec struct {
    ID                 int                  `json:"id"`
    Name               string               `json:"name"`
    Description        string               `json:"description"`
    Agent              string               `json:"agent"`
    Tags               []string             `json:"tags,omitempty"`
    DependsOn          []int                `json:"depends_on,omitempty"`
    AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria,omitempty"` // ← NUEVO
}
```

**Responsabilidad**: Cada tarea puede llevar criteria asociados. Si el slice está vacío, no hay criteria que evaluar.

### 4.3 `PhaseResult` — Criteria fields

**Archivo**: `internal/boomerang/orchestrator.go` (modificar)

```go
type PhaseResult struct {
    // ... campos existentes ...
    AcceptanceCriteria []AcceptanceCriteria `json:"acceptance_criteria,omitempty"` // ← NUEVO
    CriteriaSummary    *CriteriaSummary      `json:"criteria_summary,omitempty"`    // ← NUEVO
}
```

**Responsabilidad**: Servir como vehículo de datos del Boomerang al scheduler. Los criteria viajan desde `runPhaseV2` hasta el scheduler a través de este struct.

### 4.4 `Result` (scheduler) — Criteria field

**Archivo**: `internal/scheduler/phase.go` (modificar)

```go
type Result struct {
    Phase         Phase
    Status        Status
    Summary       string
    Error         error
    MemoryContext string
    CriteriaSummary *boomerang.CriteriaSummary // ← NUEVO
}
```

**Responsabilidad**: El scheduler necesita criteria para pasarlos a `ApprovalGate` y `writeHandoff`. Este campo se puebla desde `PhaseResult.CriteriaSummary` cuando se usa Boomerang, o desde una query a HelixDB cuando se usa phase stub.

### 4.5 `TaskRow` — AcceptanceCriteria field

**Archivo**: `internal/db/helix/types.go` (modificar)

```go
type TaskRow struct {
    ID                 int              `json:"$id"`
    Name               string           `json:"name"`
    Description        string           `json:"description"`
    Phase              string           `json:"phase"`
    Status             string           `json:"status"`
    AcceptanceCriteria []map[string]any `json:"acceptance_criteria,omitempty"` // ← NUEVO
    CreatedAt          string           `json:"created_at"`
}
```

**Responsabilidad**: Persistencia en HelixDB. Usa `[]map[string]any` para evitar dependencia del paquete `helix` al `boomerang`.

### 4.6 `PhaseConfigV2` — AcceptanceCriteria field

**Archivo**: `internal/boomerang/phase_config.go` (modificar)

```go
type PhaseConfigV2 struct {
    // ... campos existentes ...
    AcceptanceCriteria []AcceptanceCriteria // ← NUEVO: criteria heredados de fase anterior
}
```

**Responsabilidad**: Transportar criteria entre la configuración de fase y el ciclo Boomerang. `MemoryStep` o `ThinkStep` pueblan este campo desde HelixDB.

### 4.7 `saveTaskToHelix` — Nuevo parámetro criteria

**Archivo**: `cmd/zyrocli/mcp_server.go` (modificar)

```go
func saveTaskToHelix(taskID, name, agent, phase string, criteria []AcceptanceCriteria) {
```

**Responsabilidad**: Persistir criteria en el nodo Task de HelixDB al crear la tarea.

### 4.8 `QualityStep.evaluateCriteria` — Nueva función

**Archivo**: `internal/boomerang/quality.go` (modificar)

```go
func (o *BoomerangOrchestrator) evaluateCriteria(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) bool
```

**Responsabilidad**: Evaluar cada acceptance criterion contra los resultados de las tareas delegadas. Si un criterion está "pending" y la tarea correspondiente fue exitosa con output → pasa a "verified". Si la tarea falló → pasa a "failed".

### 4.9 `ApprovalGate` — Nuevo parámetro criteria

**Archivo**: `internal/scheduler/approval.go` (modificar)

```go
func ApprovalGate(phase Phase, summary string, criteria *CriteriaSummary) (bool, error)
```

**Responsabilidad**: Mostrar resumen de criteria al humano. Bloquear si hay criteria "failed".

### 4.10 `writeHandoff` — Nuevo parámetro criteria

**Archivo**: `internal/scheduler/handoff.go` (modificar)

```go
func writeHandoff(phaseName string, result *Result, nextPhase Phase, criteria *CriteriaSummary) error
```

**Responsabilidad**: Incluir tabla de acceptance criteria status en el handoff markdown.

### 4.11 `Payload` — Nuevos campos

**Archivo**: `internal/handoff/payload.go` (modificar)

```go
// AcceptanceCriteriaInfo representa un acceptance criterion en el handoff payload.
type AcceptanceCriteriaInfo struct {
    ID          string `yaml:"id" json:"id"`
    Description string `yaml:"description" json:"description"`
    Status      string `yaml:"status" json:"status"`
}

// UserStory con nuevo campo Criteria
type UserStory struct {
    Story      string                  `yaml:"story,omitempty"`
    Acceptance string                  `yaml:"acceptance,omitempty"` // ← se mantiene por backward compat
    Criteria   []AcceptanceCriteriaInfo `yaml:"criteria,omitempty"`  // ← NUEVO
}

// AcceptanceSummary resumen global de criteria al final del pipeline.
type AcceptanceSummary struct {
    Total    int `yaml:"total"`
    Verified int `yaml:"verified"`
    Failed   int `yaml:"failed"`
    Pending  int `yaml:"pending"`
}

// Payload con nuevo campo
type Payload struct {
    // ... campos existentes ...
    AcceptanceStatus *AcceptanceSummary `yaml:"acceptance_status,omitempty"` // ← NUEVO
}
```

### 4.12 `generateDAGForPhase*` — Inyectar criteria en TaskSpec

**Archivo**: `internal/boomerang/think.go` (modificar)

Cada generador de DAG debe aceptar criteria opcionales y asignarlos a las TaskSpec correspondientes.

```go
func generateDAGForPhase3(criteria []AcceptanceCriteria) *TaskDAG {
    dag := &TaskDAG{ParallelGroups: [][]int{{0, 1}}}
    dag.Tasks = []TaskSpec{
        {
            ID: 1, Name: "implement",
            Description: "Implementar cambios según spec",
            Agent: "zyro-sdd-apply",
            Tags: []string{"implementation"},
            // Asignar criteria relevantes a esta tarea
            AcceptanceCriteria: filterByTask(criteria, "implement"),
        },
        {
            ID: 2, Name: "verify",
            Description: "Verificar implementación",
            Agent: "zyro-sdd-verify",
            Tags: []string{"verification"},
        },
    }
    return dag
}
```

---

## 5. Diagrama de Datos

### 5.1 Propagación del Acceptance Criteria entre fases

```
FASE           DATA STRUCT                     PERSISTENCIA
───            ──────────                      ────────────

F1 (Spec)      Spec node (HelixDB)             {
                 acceptance_criteria: []          "criteria": [
                   {ID, Description, Phase,         {"id": "AC-001",
                    Status: "pending", Source}        "description": "...",
                   ]                                 "status": "pending"
                 }                                 }
                                                 ↕
F2 (Tasks)     Task node (HelixDB)             {
                 acceptance_criteria: []          "acceptance_criteria": [
                   {ID, Description, Phase,         {"id": "AC-001",
                    Status: "pending", Source,        "status": "pending",
                    TaskID}                          "task_id": "implement-auth"
                   ]                               }
                 }
                                                 ↕
F3 (Boomerang) PhaseResult {                   (en memoria durante RunPhase)
                 AcceptanceCriteria: []           Después → SaveStep persiste
                 CriteriaSummary: {               status actualizado a HelixDB
                   Total, Pending,
                   Verified, Failed
                 }
               }
                                                 ↕
             scheduler.Result {                 (pasa a través de scheduler)
               CriteriaSummary: {...}
             }
                                                 ↕
F4 (Archive)   ApprovalGate: CriteriaSummary      No persiste nuevo nodo
               Handoff markdown: tabla status      (solo se escribe handoff.md)
               Handoff payload YAML:
                 user_story.criteria: []
                 acceptance_status: {...}
```

### 5.2 Mapeo AcceptanceCriteria → HelixDB

```
AcceptanceCriteria (Go struct boomerang)
  │
  │  serialización en mcp_server.go
  ▼
map[string]any {
  "id":          "AC-001",
  "description": "...",
  "phase":       "F1",
  "status":      "pending",
  "source":      "spec",
  "task_id":     "implement-auth"
}
  │
  │  CreateNode("Task", props)
  ▼
HelixDB Node (JSON almacenado en propiedades del nodo Task)
  task_id: "implement-auth"
  name: "Implementar auth"
  acceptance_criteria: [
    {id: "AC-001", description: "...", phase: "F1", status: "pending", ...}
  ]
```

---

## 6. Interfaces entre Componentes

| # | From | To | Contrato | Tipo de interfaz |
|---|------|----|----------|-----------------|
| I1 | `scheduler.Run()` | `boomerang.RunPhase()` | `PhaseConfig{Phase, TaskDesc} → (*PhaseResult, error)` | Llamada directa (mismo proceso) |
| I2 | `boomerang.runPhaseV2` | `boomerang.QualityStep()` | `(ctx, phase, dag, delegateResult) → (bool, error)` | Método interno |
| I3 | `boomerang.evaluateCriteria` | `DelegateResult.TaskResults` | `map[string]TaskResult` — lookup por task name | Map interno |
| I4 | `scheduler.Run()` | `scheduler.ApprovalGate()` | `(phase, summary, *CriteriaSummary) → (bool, error)` | Llamada directa |
| I5 | `scheduler.Run()` | `scheduler.writeHandoff()` | `(phaseName, *Result, Phase, *CriteriaSummary) → error` | Llamada directa |
| I6 | `mcp_server.go` | `helix.Client.CreateNode()` | `(ctx, "Task", props) → (int64, error)` | Llamada SDK |
| I7 | `mcp_server.go` | `helix.Client.UpdateNode()` | `(ctx, nodeID, props) → error` | Llamada SDK |
| I8 | `scheduler.Result` → `writeHandoff` | `CriteriaSummary` | Struct embebido en Result | Lectura de campo |
| I9 | `PhaseResult` → `scheduler.Run` | `CriteriaSummary` | Extraído de PhaseResult al construir scheduler.Result | Asignación manual |
| I10 | `sdd-verify` (subagente) | `helix.Client` | Query de Task nodes por phase + criteria status | MCP tool / SDK |

---

## 7. Archivos a Modificar

### 7.1 Lista completa

| # | Archivo | Tipo de cambio | Descripción |
|---|---------|---------------|-------------|
| 1 | `internal/boomerang/criteria.go` | **NUEVO** | Tipo `AcceptanceCriteria`, `CriteriaStatus`, `CriteriaSummary`, `NewCriteriaSummary()` |
| 2 | `internal/boomerang/orchestrator.go` | MODIFICAR | Agregar `AcceptanceCriteria []AcceptanceCriteria` a `TaskSpec` |
| 3 | `internal/boomerang/orchestrator.go` | MODIFICAR | Agregar `AcceptanceCriteria` + `CriteriaSummary` a `PhaseResult` |
| 4 | `internal/boomerang/quality.go` | MODIFICAR | Agregar `evaluateCriteria()` y llamarlo desde `QualityStep` |
| 5 | `internal/boomerang/phase_config.go` | MODIFICAR | Agregar `AcceptanceCriteria` a `PhaseConfigV2` |
| 6 | `internal/boomerang/think.go` | MODIFICAR | Pasar criteria opcionales a `generateDAGForPhase*()` |
| 7 | `internal/boomerang/memory.go` | MODIFICAR | (opcional) Recall de criteria desde HelixDB |
| 8 | `internal/boomerang/save.go` | MODIFICAR | Persistir criteria status actualizado en memoria causal |
| 9 | `internal/db/helix/types.go` | MODIFICAR | Agregar `AcceptanceCriteria []map[string]any` a `TaskRow` |
| 10 | `cmd/zyrocli/mcp_server.go` | MODIFICAR | Cambiar firma `saveTaskToHelix` para aceptar criteria |
| 11 | `cmd/zyrocli/mcp_server.go` | MODIFICAR | Serializar criteria a `[]map[string]any` en `saveTaskToHelix` |
| 12 | `internal/scheduler/phase.go` | MODIFICAR | Agregar `CriteriaSummary` a `Result` |
| 13 | `internal/scheduler/approval.go` | MODIFICAR | Nuevo parámetro `*CriteriaSummary` en `ApprovalGate` |
| 14 | `internal/scheduler/approval.go` | MODIFICAR | `CriteriaSummary` struct (mover acá o importar de boomerang) |
| 15 | `internal/scheduler/handoff.go` | MODIFICAR | Nuevo parámetro `*CriteriaSummary` en `writeHandoff` |
| 16 | `internal/scheduler/handoff.go` | MODIFICAR | Agregar tabla de criteria status al markdown |
| 17 | `internal/scheduler/scheduler.go` | MODIFICAR | Extraer criteria de boomerangResult y pasarlo a ApprovalGate/handoff |
| 18 | `internal/handoff/payload.go` | MODIFICAR | Agregar `AcceptanceCriteriaInfo`, `AcceptanceSummary`, campos nuevos |
| 19 | `.config/opencode/plugins/skills/sdd-verify/SKILL.md` | MODIFICAR | Leer criteria desde HelixDB, actualizar status |
| 20 | `internal/boomerang/boomerang_test.go` | MODIFICAR | Tests de criteria en QualityStep |
| 21 | `internal/boomerang/quality_test.go` | **NUEVO** | Tests de evaluateCriteria |
| 22 | `internal/scheduler/scheduler_test.go` | MODIFICAR | Tests con criteria en ApprovalGate/handoff |

### 7.2 Dependencias entre cambios

```
Paso 1: criteria.go (NUEVO)
   │
   ├─→ Paso 2: TaskRow (types.go) + helix prop
   ├─→ Paso 3: TaskSpec (orchestrator.go)
   │      │
   │      ├─→ Paso 4: phase_config.go (PhaseConfigV2)
   │      ├─→ Paso 5: think.go (inyectar criteria en DAG)
   │      └─→ Paso 6: quality.go (evaluateCriteria)
   │
   ├─→ Paso 7: mcp_server.go (saveTaskToHelix con criteria)
   │
   ├─→ Paso 8: scheduler/phase.go (Result.CriteriaSummary)
   │      │
   │      ├─→ Paso 9: approval.go (CriteriaSummary + bloqueo)
   │      ├─→ Paso 10: handoff.go (tabla en markdown)
   │      └─→ Paso 11: scheduler.go (extraer y pasar criteria)
   │
   ├─→ Paso 12: handoff/payload.go (AcceptanceCriteriaInfo)
   │
   └─→ Paso 13: sdd-verify/SKILL.md (consumir criteria)
```

---

## 8. Decisiones de Diseño Detalladas

### 8.1 ¿Dónde definir `CriteriaSummary`? ¿En `boomerang` o `scheduler`?

**Decisión**: En `boomerang` (junto a `AcceptanceCriteria`).

**Razonamiento**:
- `CriteriaSummary` se computa en `QualityStep` (paquete boomerang)
- Se necesita en `ApprovalGate` y `writeHandoff` (paquete scheduler)
- Si lo definimos en `scheduler`, `boomerang` no puede usarlo
- Si lo definimos en `boomerang`, `scheduler` lo importa (ya importa `boomerang`)

`scheduler` ya importa `boomerang` (line 7 de `phase.go`): `"github.com/secko/zyrocli/internal/boomerang"`. No hay dependencia circular.

### 8.2 ¿Cómo lleva criteria `scheduler.Result`?

**Decisión**: Campo `*boomerang.CriteriaSummary`.

```go
type Result struct {
    Phase          Phase
    Status         Status
    Summary        string
    Error          error
    MemoryContext  string
    CriteriaSummary *boomerang.CriteriaSummary  // nil si no hay criteria
}
```

**Alternativa descartada**: Usar `[]AcceptanceCriteria` directamente → más datos pero mayor acoplamiento. El scheduler solo necesita el resumen (totales). Para más detalle, el scheduler puede leer `PhaseResult` del Boomerang.

### 8.3 ¿Cómo se propaga criteria en modo phase stub (sin Boomerang)?

Los phase stubs (`F1Runner`, `F2Runner`, `F3Runner`, `F4Runner`) no usan Boomerang, por lo que no tienen `PhaseResult`.

**Solución**: Cada phase stub que quiera reportar criteria debe:
1. Leer criteria desde HelixDB (Task nodes) en su `Run()` si aplica
2. Poblarlos en `Result.CriteriaSummary` manualmente

Para la primera implementación, los phase stubs no reportan criteria (backward compat). Solo el pipeline Boomerang reporta criteria. Esto es aceptable porque:
- El pipeline Boomerang es el camino principal
- Los phase stubs son legacy
- Futuras iteraciones pueden agregar criteria a los stubs

### 8.4 ¿Cómo actualiza `sdd-verify` el status de criteria en HelixDB?

**Problema**: `sdd-verify` tiene permisos de escritura restringidos (según `spec-fix-subagent-permissions.md`).

**Solución**: El orquestador (Boomerang) es quien actualiza el status via `SaveStep`. `sdd-verify`:
1. Lee criteria de HelixDB vía `task_context` MCP tool
2. Evalúa criteria contra evidencia de implementación
3. **Reporta resultados al orquestador** (no escribe directo)
4. El orquestador persiste el status actualizado en `SaveStep`

Si no hay orquestador (modo phase stub), `sdd-verify` puede escribir a HelixDB si el permiso lo permite. Esto se resolverá en el spec de permisos.

### 8.5 ¿Cómo se serializa/deserializa criteria entre Go y HelixDB?

**Serialización** (Go → HelixDB) en `mcp_server.go`:
```go
func saveTaskToHelix(taskID, name, agent, phase string, criteria []boomerang.AcceptanceCriteria) {
    criteriaData := make([]map[string]any, len(criteria))
    for i, c := range criteria {
        criteriaData[i] = map[string]any{
            "id":          c.ID,
            "description": c.Description,
            "phase":       c.Phase,
            "status":      string(c.Status),
            "source":      c.Source,
            "task_id":     c.TaskID,
        }
    }
    props["acceptance_criteria"] = criteriaData
}
```

**Deserialización** (HelixDB → Go) en MemoryStep o en sdd-verify:
```go
func deserializeCriteria(raw []interface{}) []boomerang.AcceptanceCriteria {
    var result []boomerang.AcceptanceCriteria
    for _, item := range raw {
        m := item.(map[string]interface{})
        result = append(result, boomerang.AcceptanceCriteria{
            ID:          getString(m, "id"),
            Description: getString(m, "description"),
            Phase:       getString(m, "phase"),
            Status:      boomerang.CriteriaStatus(getString(m, "status")),
            Source:      getString(m, "source"),
            TaskID:      getString(m, "task_id"),
        })
    }
    return result
}
```

### 8.6 ¿Qué pasa si HelixDB no está disponible?

Si HelixDB no está disponible (como indica `mcp_server.go:90-93`):
1. `saveTaskToHelix` loggea warning y retorna sin error
2. Los criteria no se persisten, pero la task se ejecuta igual
3. `QualityStep` igual evalúa criteria en memoria
4. `ApprovalGate` muestra criteria con datos disponibles en memoria

**No es blocking**: El sistema funciona sin HelixDB, pero los criteria no sobreviven a un restart del MCP server.

### 8.7 `GuidedApproval` con criteria

`GuidedApproval.PromptApproval()` actualmente muestra:
```
─── Fase: F3 — Completada ───
Resumen: Boomerang: 2 tasks, quality=true, facts=1
```

El nuevo flujo con criteria:
```
─── Fase: F3 — Completada ───
Resumen: Boomerang: 2 tasks, quality=true, facts=1

### Acceptance Criteria
Total: 3 | ✅ Verified: 2 | ⏳ Pending: 0 | ❌ Failed: 1

❌ No se puede aprobar: 1 acceptance criteria fallaron.
```

El `GuidedApproval` debe incluir un método para agregar criteria:
```go
type GuidedApproval struct {
    Phase      Phase
    Summary    string
    Recommend  string
    Risk       string
    FullOutput string
    Criteria   *CriteriaSummary // ← NUEVO
}
```

---

## 9. Flujo de Datos Paso a Paso

### 9.1 F1: Spec define criteria

```
zyro-sdd-spec SKILL.md
  │
  │  1. Lee contexto de F0 (patrones, librerías)
  │  2. Genera Spec node en HelixDB con:
  │     - architecture, modules, dependencies, testing_strategy
  │     - acceptance_criteria: [{id, description, phase:"F1", status:"pending", source:"spec"}]
  │
  ▼
HelixDB: Spec node (project_id=N)
  properties.acceptance_criteria = [...]
```

### 9.2 F2: Tasks heredan criteria

```
zyro-sdd-tasks SKILL.md
  │
  │  1. Lee Spec node de HelixDB
  │  2. Extrae acceptance_criteria del Spec
  │  3. Asigna cada criterion a la Task correspondiente
  │  4. Crea Task nodes via dispatch_task → saveTaskToHelix con criteria
  │
  ▼
HelixDB: Task node (task_id="implement-auth")
  properties.acceptance_criteria = [
    {id: "AC-001", description: "...", status: "pending", task_id: "implement-auth"}
  ]
```

### 9.3 F3: Boomerang evalúa criteria

```
scheduler.Run("F3", ...)
  │
  ▼
boomerang.RunPhase(PhaseConfig{Phase: "F3"})
  │
  ├─ MemoryStep: recall criteria from HelixDB (via memory causal or helix query)
  │
  ├─ ThinkStep: inject criteria into TaskSpec
  │   dag.Tasks[0].AcceptanceCriteria = [{AC-001, AC-002, ...}]
  │
  ├─ DelegateStep: dispatch tasks → subagentes ejecutan
  │   delegateResult.TaskResults["implement"].Success = true
  │   delegateResult.TaskResults["implement"].Output = "...code changes..."
  │
  ├─ QualityStep:
  │   ├─ 1. go build ./... (OK)
  │   ├─ 2. Verificar task success (OK)
  │   └─ 3. evaluateCriteria():
  │       ├─ AC-001: task "implement" success + output → CriteriaVerified
  │       ├─ AC-002: task "implement" success + output → CriteriaVerified
  │       └─ Retorna true (todos verified)
  │
  ├─ SaveStep: persist criteria status to HelixDB
  │   Update Task node: acceptance_criteria[0].status = "verified"
  │
  └─ Retorna PhaseResult {
        AcceptanceCriteria: [AC-001 (verified), AC-002 (verified)],
        CriteriaSummary: {Total: 2, Verified: 2, Pending: 0, Failed: 0}
      }
```

### 9.4 Scheduler procesa criteria post-fase

```
scheduler.Run()
  │
  ├─ boomerangResult = boomerang.RunPhase(...)
  │   (tiene boomerangResult.CriteriaSummary)
  │
  ├─ Construye scheduler.Result:
  │   result = &Result{
  │     Phase: "F3",
  │     Status: StatusSuccess,
  │     Summary: "Boomerang: 2 tasks, quality=true",
  │     CriteriaSummary: boomerangResult.CriteriaSummary,  // ← NUEVO
  │   }
  │
  ├─ writeHandoff("F3", result, nextPhase, result.CriteriaSummary)
  │   → Genera handoff-F3.md con tabla de criteria
  │
  └─ ApprovalGate("F3", result.Summary, result.CriteriaSummary)
      → Muestra resumen, no bloquea (Verified=2, Failed=0)
```

### 9.5 Handoff incluye criteria

```
writeHandoff("F3", result, nextPhase)
  │
  ▼
Genera .zyro/handoffs/F3-handoff.md:

# Handoff — Fase F3

**Generado:** 2026-06-20T10:30:00Z
**Estado:** ✅ success

---

## Resumen

Boomerang: 2 tasks, quality=true, facts=1

## Acceptance Criteria

| Estado | Cantidad |
|--------|----------|
| ✅ Verified | 2 |
| ⏳ Pending | 0 |
| ❌ Failed | 0 |
| **Total** | **2** |

## Artefactos recientes
...
```

### 9.6 sdd-verify lee criteria de HelixDB

```
sdd-verify SKILL.md execution flow (paso 3.5):
  1. Query HelixDB: MATCH (t:Task {phase: "F3"}) RETURN t
  2. Extraer acceptance_criteria de cada Task node
  3. Para cada criterion con status "pending":
     a. Buscar evidencia en implementación (output de apply, tests)
     b. Si evidencia satisface el criterion → status = "verified"
     c. Si no → status = "failed"
  4. Reportar compliance matrix en verify-report.md

Compliance matrix:
  | ID | Description | Task | Status | Evidence |
  |----|-------------|------|--------|----------|
  | AC-001 | Reject requests without JWT | implement-auth | ✅ VERIFIED | TestAuthMiddleware fails without token |
```

---

## 10. Pruebas

### 10.1 Unitarias (nuevas)

| # | Nombre | Archivo | Descripción |
|---|--------|---------|-------------|
| UT1 | `TestNewCriteriaSummary` | `criteria_test.go` (NUEVO) | Crear `CriteriaSummary` de slice vacío → todos 0 |
| UT2 | `TestCriteriaSummaryCounts` | `criteria_test.go` (NUEVO) | Mezcla de pending/verified/failed → counts correctos |
| UT3 | `TestAcceptanceCriteriaJSON` | `criteria_test.go` (NUEVO) | Marshal/Unmarshal JSON de `AcceptanceCriteria` |
| UT4 | `TestAcceptanceCriteriaYAML` | `criteria_test.go` (NUEVO) | Marshal/Unmarshal YAML de `AcceptanceCriteria` |
| UT5 | `TestEvaluateCriteriaAllPass` | `quality_test.go` (NUEVO) | DAG con criteria pending, delegate results exitosos → todo verified |
| UT6 | `TestEvaluateCriteriaFail` | `quality_test.go` (NUEVO) | DAG con criteria, delegate result fallido → failed, retorna false |
| UT7 | `TestEvaluateCriteriaEmpty` | `quality_test.go` (NUEVO) | Slice vacío → retorna true |
| UT8 | `TestEvaluateCriteriaNoDAG` | `quality_test.go` (NUEVO) | DAG nil → retorna true |
| UT9 | `TestEvaluateCriteriaMixed` | `quality_test.go` (NUEVO) | Algunos verified, otros pending → retorna false |
| UT10 | `TestTaskSpecWithCriteria` | `orchestrator_test.go` | TaskSpec con criteria serializa JSON correcto |
| UT11 | `TestTaskSpecWithoutCriteria` | `orchestrator_test.go` | TaskSpec sin criteria → omitempty, no aparece en JSON |
| UT12 | `TestPhaseResultCriteria` | `orchestrator_test.go` | PhaseResult transporta criteria correctamente |
| UT13 | `TestApprovalGateBlocksFailed` | `approval_test.go` (NUEVO) | CriteriaSummary con Failed > 0 → retorna false |
| UT14 | `TestApprovalGateAllowsVerified` | `approval_test.go` (NUEVO) | CriteriaSummary con all verified → retorna true (misma entrada) |
| UT15 | `TestApprovalGateNoCriteria` | `approval_test.go` (NUEVO) | CriteriaSummary nil → comportamiento legacy |
| UT16 | `TestWriteHandoffWithCriteria` | `handoff_test.go` (NUEVO) | Markdown generado incluye tabla de criteria |
| UT17 | `TestWriteHandoffWithoutCriteria` | `handoff_test.go` (NUEVO) | Markdown generado sin tabla (nil criteria) |
| UT18 | `TestSaveTaskToHelixWithCriteria` | `mcp_server_test.go` (NUEVO) | Serialización criteria → map → JSON consistente |
| UT19 | `TestSaveTaskToHelixNoCriteria` | `mcp_server_test.go` (NUEVO) | Save sin criteria → nodo sin campo acceptance_criteria |
| UT20 | `TestDeserializeCriteria` | `quality_test.go` (NUEVO) | raw interface{} → AcceptanceCriteria correcto |
| UT21 | `TestPhaseConfigV2WithCriteria` | `phase_config_test.go` (NUEVO) | PhaseConfigV2 transporta criteria correctamente |

### 10.2 Integración (nuevas)

| # | Nombre | Descripción |
|---|--------|-------------|
| IT1 | `TestPipelineF3CriteriaEval` | Ejecutar Boomerang F3 con criteria, verificar que QualityStep los evalúa |
| IT2 | `TestPipelineF3CriteriaFail` | Ejecutar Boomerang F3 con criteria que fallan → PhaseResult.Success=false |
| IT3 | `TestCriteriaPersistenceHelixDB` | saveTaskToHelix con criteria → query task_context → criteria presentes |
| IT4 | `TestSchedulerResultWithCriteria` | scheduler.Run con Boomerang → Result.CriteriaSummary poblado |

### 10.3 Regresión

```bash
go build ./...
go test ./internal/boomerang/...
go test ./internal/scheduler/...
go test ./internal/handoff/...
go test ./internal/db/helix/...
```

---

## 11. Riesgos y Mitigaciones

| ID | Riesgo | P | I | Mitigación |
|----|--------|---|---|------------|
| R1 | **Dependencia circular**: boomerang → handoff → boomerang | B | A | `AcceptanceCriteria` en `boomerang`. `handoff/payload.go` define su propio `AcceptanceCriteriaInfo` (no importa boomerang). |
| R2 | **Backward compatibility**: fases existentes sin criteria dejan de funcionar | B | A | `omitempty` en todos los campos. Slice vacío = pass. Tests UT11, UT15, UT17, UT19 verifican esto. |
| R3 | **Criteria perdidos entre fases**: fase F2 no propaga criteria a F3 | M | A | PhaseConfigV2.AcceptanceCriteria transporta criteria. MemoryStep los recall desde HelixDB. |
| R4 | **HelixDB no disponible**: criteria no se persisten | M | M | Log de warning, no bloquea. Criteria se evalúan en memoria durante la fase activa. |
| R5 | **sdd-verify sin permisos de escritura**: no puede actualizar status | M | A | Boomerang.SaveStep persiste el status. sdd-verify solo lee. Resuelto en spec de permisos. |
| R6 | **Evaluación básica muy superficial**: solo task success + output | M | M | Es un primer paso. Documentado como mejora futura. No impide el flujo. |
| R7 | **Criteria duplicados**: mismo ID en múltiples tasks | B | B | ID único por criterion. Documentar que cada criterion se asigna a un solo task. |
| R8 | **Criteria en handoff.yaml muy grande** | B | B | Solo se incluye resumen (totales) en handoff.yaml. Detalle completo en HelixDB. |

---

## 12. Orden de Implementación

| Paso | Archivos | Depende de | Δ |
|------|----------|-----------|---|
| P1 | `internal/boomerang/criteria.go` (NUEVO) | — | +~60 loc |
| P2 | `internal/boomerang/orchestrator.go` | P1 | +3 lines (TaskSpec field) |
| P3 | `internal/boomerang/orchestrator.go` | P1 | +4 lines (PhaseResult fields) |
| P4 | `internal/db/helix/types.go` | — | +1 line (TaskRow field) |
| P5 | `internal/boomerang/phase_config.go` | P1 | +1 line (PhaseConfigV2 field) |
| P6 | `internal/boomerang/quality.go` | P1, P2 | +~60 loc (evaluateCriteria) |
| P7 | `internal/boomerang/think.go` | P5 | +~15 loc (inyectar criteria en DAG) |
| P8 | `cmd/zyrocli/mcp_server.go` | P1, P4 | +~25 loc (serialización) |
| P9 | `internal/scheduler/phase.go` | P1 | +1 line (Result field) |
| P10 | `internal/scheduler/approval.go` | P1 | +~30 loc (CriteriaSummary en ApprovalGate) |
| P11 | `internal/scheduler/handoff.go` | P1 | +~20 loc (tabla en handoff) |
| P12 | `internal/scheduler/scheduler.go` | P6, P9, P10, P11 | +~15 loc (extraer y pasar criteria) |
| P13 | `internal/handoff/payload.go` | — | +~25 loc (nuevos tipos) |
| P14 | `internal/boomerang/save.go` | P6 | +~10 loc (persistir criteria status) |
| P15 | `.config/opencode/plugins/skills/sdd-verify/SKILL.md` | — | +~20 lines (instrucciones) |
| P16 | Tests unitarios | P1-P14 | +~400 loc tests |
| P17 | Tests integración | P1-P14 | +~100 loc tests |

### Dependencias reales de implementación

```
P1 (criteria.go)
├── P2 (TaskSpec field)
├── P3 (PhaseResult field)
├── P5 (PhaseConfigV2 field)
├── P9 (Result field)
├── P10 (ApprovalGate criteria param)
├── P11 (Handoff criteria param)
└── P13 (Payload types, independiente)

P2 + P5 ──→ P6 (quality.go evaluateCriteria)
P1 + P4 ──→ P8 (mcp_server.go serialización)
P2 + P5 ──→ P7 (think.go inyectar)

P6 + P9 + P10 + P11 ──→ P12 (scheduler.go conexión)

P6 ──→ P14 (save.go persistir status)

P1-P14 ──→ P16 (tests unitarios)
P1-P14 ──→ P17 (tests integración)

P15 (sdd-verify) independiente de Go code
```

**Orden recomendado de implementación**:
1. P1 → P2 → P3 → P4 → P5 (tipos base)
2. P8 (MCP server serialización)
3. P6 → P7 (evaluación)
4. P9 → P10 → P11 → P12 (scheduler)
5. P13 → P14 (handoff + persistencia)
6. P15 (sdd-verify skill)
7. P16 → P17 (tests)

