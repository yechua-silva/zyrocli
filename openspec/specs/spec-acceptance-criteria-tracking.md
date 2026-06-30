# Acceptance Criteria Tracking — Especificación Técnica

> **Documento**: Spec Técnica
> **Estado**: Borrador
> **Fase**: F1 (Spec)
> **Basado en**: Investigación del codebase, análisis de brecha entre fases SDD

---

## 1. Propósito

Cerrar la brecha de conectividad de **acceptance criteria** entre las fases del pipeline SDD (Propose → Spec → Design → Tasks → Apply → Verify → Archive).

### Problema actual

Los acceptance criteria se definen en texto libre dentro de propuestas y specs (markdown), pero:

1. **No viajan entre fases** — se definen en F0/F1 y se pierden en F2/F3/F4.
2. **No se persisten en HelixDB** — el nodo Task no tiene campo `acceptance_criteria`, por lo que no son queryables ni trazables.
3. **No se verifican automáticamente** — `QualityStep` solo ejecuta `go build ./...`, no evalúa si los criteria se cumplieron.
4. **`sdd-verify` genera compliance matrix manual** — el reporte es markdown estático en `openspec/changes/<change>/verify-report.md`, sin conexión al pipeline ni a HelixDB.
5. **No hay gate entre fases** — ApprovalGate no verifica criteria de la fase anterior antes de permitir el avance.

El problema raíz: **se crea algo en una fase y se desconecta en la siguiente. No da error, no avisa, solo muere.**

### Lo que ya existe (hallazgos)

| # | Hallazgo | Archivo | Línea |
|---|----------|---------|-------|
| 1 | `UserStory.Acceptance` existe como string libre | `internal/handoff/payload.go` | 26 |
| 2 | `TaskSpec` NO tiene `AcceptanceCriteria` | `internal/boomerang/orchestrator.go` | 59-66 |
| 3 | `TaskRow` en HelixDB NO tiene `acceptance_criteria` | `internal/db/helix/types.go` | 21-28 |
| 4 | MCP server crea Tasks sin `acceptance_criteria` | `cmd/zyrocli/mcp_server.go` | 564-570 |
| 5 | `QualityStep` solo verifica `go build` + task success | `internal/boomerang/quality.go` | 13-29 |
| 6 | `sdd-verify` genera compliance matrix estática | `.config/opencode/plugins/skills/sdd-verify/SKILL.md` | — |
| 7 | Handoff se escribe desde `scheduler/handoff.go` | `internal/scheduler/handoff.go` | 47-121 |
| 8 | Approval gate existe pero no evalúa criteria | `internal/scheduler/approval.go` | 97-117 |

---

## 2. Arquitectura

### 2.1 Flujo del Acceptance Criteria a través de las fases

```
Proposal (markdown)
   │  Acceptance criteria como texto libre
   ▼
F1: Spec
   │  → Agente zyro-sdd-spec extrae criteria del spec markdown
   │  → Persiste criteria como parte del nodo Spec en HelixDB
   ▼
F2: Design + Tasks
   │  → Agente zyro-sdd-design mapea criteria a módulos/componentes
   │  → Agente zyro-sdd-tasks asigna criteria a tareas específicas
   │  → MCP server persiste acceptance_criteria en cada nodo Task
   ▼
F3: Apply + Verify
   │  → Boomerang lleva criteria en TaskSpec.AcceptanceCriteria
   │  → QualityStep evalúa criteria (no solo go build)
   │  → sdd-verify lee criteria desde HelixDB
   │  → sdd-verify actualiza status (verified/failed) en HelixDB
   ▼
F4: Archive
   │  → ApprovalGate verifica que todos los criteria estén "verified"
   │  → Handoff incluye resumen de criteria status
   │  → Pipeline solo avanza si criteria de fase anterior se cumplieron
   ▼
Done
```

### 2.2 Diagrama de datos

```
┌─────────────────────────────────────────────────────────────────────┐
│                        AcceptanceCriteria                           │
│  {ID, Description, Phase, Status, Source, TaskID}                   │
└──────────────────────────┬──────────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┬────────────────┐
          │                │                │                │
          ▼                ▼                ▼                ▼
   ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐
   │ Payload    │  │ TaskSpec   │  │ TaskRow    │  │ Handoff    │
   │ (handoff)  │  │ (boomerang)│  │ (HelixDB)  │  │ (scheduler)│
   │ UserStory  │  │ +Criteria  │  │ +criteria  │  │ +criteria  │
   │ +criteria[]│  │            │  │            │  │ status     │
   └────────────┘  └────────────┘  └────────────┘  └────────────┘
```

### 2.3 Módulos afectados

| Módulo | Rol actual | Cambio |
|--------|-----------|--------|
| `internal/boomerang/orchestrator.go` | Define `TaskSpec` | Agregar `AcceptanceCriteria []AcceptanceCriteria` |
| `internal/boomerang/quality.go` | `QualityStep`: solo go build | Evaluar criteria contra delegate results |
| `internal/boomerang/think.go` | Genera DAG por fase | Inyectar criteria en TaskSpec desde handoff/memoria |
| `internal/boomerang/delegate.go` | Despacha tareas | (sin cambio directo, criteria viajan en TaskSpec) |
| `internal/db/helix/types.go` | Define `TaskRow` | Agregar campo `AcceptanceCriteria` |
| `cmd/zyrocli/mcp_server.go` | `saveTaskToHelix()` | Persistir acceptance_criteria en nodo Task |
| `internal/scheduler/approval.go` | `ApprovalGate` | Verificar criteria status antes de aprobar |
| `internal/scheduler/handoff.go` | `writeHandoff()` | Incluir resumen de criteria status |
| `internal/scheduler/scheduler.go` | `Run()` | Inyectar verificación de criteria entre fases |
| `internal/scheduler/phase_stubs.go` | Phase runners (F1-F4) | Extraer/verificar criteria en cada fase |
| `internal/handoff/payload.go` | Define `Payload`, `UserStory` | Agregar `AcceptanceCriteria` struct y field |
| `.config/opencode/plugins/skills/sdd-verify/SKILL.md` | Verificación | Leer criteria desde HelixDB en vez de markdown |

---

## 3. Especificación Detallada

### 3.1 Tipo `AcceptanceCriteria` — Struct first-class

**Ubicación**: Nuevo archivo `internal/boomerang/criteria.go` (paquete `boomerang`)

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
    Phase       string         `json:"phase" yaml:"phase"`           // fase donde se definió
    Status      CriteriaStatus `json:"status" yaml:"status"`
    Source      string         `json:"source" yaml:"source"`         // "proposal", "spec", "design", "task"
    TaskID      string         `json:"task_id,omitempty" yaml:"task_id,omitempty"` // task que lo implementa
}
```

**Razonamiento**: Se ubica en `internal/boomerang/` porque:
- Es el módulo que orquesta el ciclo de vida de las tareas.
- `TaskSpec` ya está en este paquete — los criteria son parte de la especificación de una tarea.
- `QualityStep` que evaluará los criteria está en el mismo paquete.
- Evita dependencias circulares con `handoff` o `scheduler`.

---

### 3.2 `TaskSpec` — Agregar `AcceptanceCriteria`

**Archivo**: `internal/boomerang/orchestrator.go` líneas 59-66

**Cambio**: Agregar campo `AcceptanceCriteria` a la struct `TaskSpec`.

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

**Comportamiento**:
- `omitempty` garantiza backward compatibility: fases existentes sin criteria siguen funcionando con slice vacío.
- Si `AcceptanceCriteria` está vacío, `QualityStep` salta la evaluación de criteria (no es un error).
- Los criteria se asignan en `ThinkStep` cuando se genera el DAG para cada fase.

---

### 3.3 `ThinkStep` — Inyectar criteria en el DAG

**Archivo**: `internal/boomerang/think.go`

**Cambio**: Cada generador de DAG (`generateDAGForPhase1`, etc.) debe aceptar criteria desde el contexto de memoria.

**Mecanismo**:
- `MemoryStep` ya recupera contexto de HelixDB antes de `ThinkStep`.
- Si hay acceptance criteria en el contexto de memoria (recuperados de nodos Spec o Tasks previos), se inyectan en los `TaskSpec` correspondientes.
- Si no hay criteria (fases existentes), el slice queda vacío — compatible.

**Firma modificada** (opcional — puede ser un field en `PhaseConfig`):

```go
type PhaseConfigV2 struct {
    // ... campos existentes
    AcceptanceCriteria []AcceptanceCriteria // ← NUEVO: criteria heredados de fase anterior
}
```

**Flujo**:
1. `MemoryStep` recupera criteria de fase anterior desde HelixDB.
2. `PhaseConfigV2.AcceptanceCriteria` se puebla con los criteria recuperados.
3. `ThinkStep` asigna los criteria a las `TaskSpec` del DAG según corresponda.

---

### 3.4 `TaskRow` — Agregar `AcceptanceCriteria` en HelixDB

**Archivo**: `internal/db/helix/types.go` líneas 21-28

**Cambio**: Agregar campo `AcceptanceCriteria` a `TaskRow`.

```go
type TaskRow struct {
    ID                 int                    `json:"$id"`
    Name               string                 `json:"name"`
    Description        string                 `json:"description"`
    Phase              string                 `json:"phase"`
    Status             string                 `json:"status"`
    AcceptanceCriteria []map[string]any       `json:"acceptance_criteria,omitempty"` // ← NUEVO
    CreatedAt          string                 `json:"created_at"`
}
```

**Formato en HelixDB** (JSON almacenado en el nodo Task):
```json
{
  "$id": 42,
  "name": "implement-auth",
  "description": "Implementar middleware de autenticación",
  "phase": "F3",
  "status": "running",
  "acceptance_criteria": [
    {
      "id": "AC-001",
      "description": "El middleware debe rechazar requests sin token JWT",
      "phase": "F1",
      "status": "pending",
      "source": "spec",
      "task_id": "implement-auth"
    },
    {
      "id": "AC-002",
      "description": "El middleware debe retornar 401 con mensaje descriptivo",
      "phase": "F1",
      "status": "pending",
      "source": "spec",
      "task_id": "implement-auth"
    }
  ]
}
```

**Nota**: Se usa `[]map[string]any` en lugar de `[]AcceptanceCriteria` tipado para no crear dependencia del paquete `boomerang` desde `helix`. La serialización/deserialización se hace en el MCP server.

---

### 3.5 MCP Server — Persistir criteria al crear nodos Task

**Archivo**: `cmd/zyrocli/mcp_server.go` líneas 558-577, función `saveTaskToHelix()`

**Cambio**: Agregar `acceptance_criteria` a las properties del nodo Task.

```go
func saveTaskToHelix(taskID, name, agent, phase string, criteria []boomerang.AcceptanceCriteria) {
    if helixClient == nil {
        log.Printf("[mcp] HelixDB no disponible, saltando persistencia de Task %s", taskID)
        return
    }

    // Serializar criteria a []map[string]any para HelixDB
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

    props := map[string]interface{}{
        "task_id":             taskID,
        "name":                name,
        "agent":               agent,
        "phase":               phase,
        "status":              "running",
        "acceptance_criteria": criteriaData,  // ← NUEVO
    }
    nodeID, err := helixClient.CreateNode(context.Background(), "Task", props)
    // ...
}
```

**Llamadas afectadas**: Actualizar todas las invocaciones de `saveTaskToHelix` para pasar el slice de criteria (puede ser `nil`).

---

### 3.6 `QualityStep` — Evaluar acceptance criteria

**Archivo**: `internal/boomerang/quality.go`

**Cambio**: Agregar evaluación de acceptance criteria como step adicional dentro de `QualityStep`.

```go
func (o *BoomerangOrchestrator) QualityStep(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) (bool, error) {
    // 1. Verificar que compile (para fases de implementación)
    if phase == "F3" {
        if err := exec.CommandContext(ctx, "go", "build", "./...").Run(); err != nil {
            return false, err
        }
    }

    // 2. Verificar que todas las tareas hayan sido exitosas
    for _, tr := range delegateResult.TaskResults {
        if !tr.Success {
            return false, nil
        }
    }

    // 3. Evaluar acceptance criteria (NUEVO)
    criteriaOK := o.evaluateCriteria(ctx, phase, dag, delegateResult)
    if !criteriaOK {
        return false, nil  // QualityStep falla si algún criteria no se cumple
    }

    return true, nil
}

// evaluateCriteria evalúa los acceptance criteria de las tareas de la fase.
// Retorna true si todos los criteria están "verified" o no hay criteria que evaluar.
func (o *BoomerangOrchestrator) evaluateCriteria(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) bool {
    if dag == nil {
        return true  // sin DAG, no hay criteria que evaluar
    }

    allVerified := true
    for _, task := range dag.Tasks {
        for i := range task.AcceptanceCriteria {
            c := &task.AcceptanceCriteria[i]
            if c.Status == CriteriaPending {
                // Evaluar el criterion contra el output de la tarea
                tr, exists := delegateResult.TaskResults[task.Name]
                if !exists {
                    c.Status = CriteriaFailed
                    allVerified = false
                    continue
                }
                if tr.Success && tr.Output != "" {
                    c.Status = CriteriaVerified  // Evaluación básica: tarea exitosa con output
                } else {
                    c.Status = CriteriaFailed
                    allVerified = false
                }
            } else if c.Status == CriteriaFailed {
                allVerified = false
            }
            // CriteriaVerified no necesita re-evaluación
        }
    }
    return allVerified
}
```

**Nota**: La evaluación básica (task success + output no vacío) es un primer paso. En fases futuras se puede refinar con:
- Pattern matching en el output contra la descripción del criterion.
- Ejecución de tests específicos por criterion.
- Validación por LLM del output contra el criterion.

---

### 3.7 Approval Gate — Verificar criteria antes de approval

**Archivo**: `internal/scheduler/approval.go`

**Cambio**: `ApprovalGate` recibe el resumen de criteria status y lo muestra al humano.

```go
// CriteriaSummary proporciona un resumen del estado de los acceptance criteria.
type CriteriaSummary struct {
    Total    int `json:"total"`
    Pending  int `json:"pending"`
    Verified int `json:"verified"`
    Failed   int `json:"failed"`
}

// ApprovalGate procesa aprobación mostrando criteria status.
func ApprovalGate(phase Phase, summary string, criteria *CriteriaSummary) (bool, error) {
    // Construir resumen de criteria
    criteriaBlock := ""
    if criteria != nil && criteria.Total > 0 {
        criteriaBlock = fmt.Sprintf(
            "\n### Acceptance Criteria\nTotal: %d | ✅ Verified: %d | ⏳ Pending: %d | ❌ Failed: %d\n",
            criteria.Total, criteria.Verified, criteria.Pending, criteria.Failed,
        )
        // Bloquear si hay failed criteria
        if criteria.Failed > 0 {
            fmt.Printf("❌ No se puede aprobar: %d acceptance criteria fallaron.\n", criteria.Failed)
            return false, nil
        }
    }

    // ... resto del diálogo existente, inyectando criteriaBlock en el mensaje
}
```

**Cambio en `scheduler.go`**: La llamada a `ApprovalGate` (línea 119) debe pasar el `CriteriaSummary` de la fase completada:

```go
// Antes:
approved, err := ApprovalGate(result.Phase, result.Summary)

// Después:
var criteriaSummary *CriteriaSummary
if boomerangResult != nil {
    criteriaSummary = extractCriteriaSummary(boomerangResult, dag)
}
approved, err := ApprovalGate(result.Phase, result.Summary, criteriaSummary)
```

---

### 3.8 Handoff — Incluir criteria status

**Archivo**: `internal/scheduler/handoff.go` función `writeHandoff()` (líneas 47-121)

**Cambio**: Agregar sección de acceptance criteria status al handoff.

```go
func writeHandoff(phaseName string, result *Result, nextPhase Phase, criteria *CriteriaSummary) error {
    // ... código existente ...

    // Bloque de acceptance criteria
    criteriaBlock := ""
    if criteria != nil && criteria.Total > 0 {
        criteriaBlock = fmt.Sprintf(`
## Acceptance Criteria

| Estado | Cantidad |
|--------|----------|
| ✅ Verified | %d |
| ⏳ Pending | %d |
| ❌ Failed | %d |
| **Total** | **%d** |
`,
            criteria.Verified, criteria.Pending, criteria.Failed, criteria.Total,
        )
    }

    content := fmt.Sprintf(`# Handoff — Fase %s

**Generado:** %s
**Estado:** %s %s

---

## Resumen

%s
%s

## Artefactos recientes

%s

## Siguiente fase sugerida

%s
`,
        phaseName,
        time.Now().Format(time.RFC3339),
        statusEmoji, result.Status,
        result.Summary,
        criteriaBlock,  // ← NUEVO
        artifactsBlock,
        nextPhaseStr,
    )
    // ...
}
```

**Nota**: La función `writeHandoff` se llama desde `scheduler.go` línea 109. Se debe actualizar la llamada para pasar el `CriteriaSummary`.

---

### 3.9 Handoff Payload — Agregar AcceptanceCriteria

**Archivo**: `internal/handoff/payload.go`

**Cambio 1**: Agregar struct `AcceptanceCriteriaInfo` y campo en `UserStory`.

```go
// AcceptanceCriteriaInfo representa un acceptance criterion en el handoff payload.
type AcceptanceCriteriaInfo struct {
    ID          string `yaml:"id" json:"id"`
    Description string `yaml:"description" json:"description"`
    Status      string `yaml:"status" json:"status"`
}

// UserStory captures a single user story for the change.
type UserStory struct {
    Story      string                  `yaml:"story,omitempty"`
    Acceptance string                  `yaml:"acceptance,omitempty"` // ← se mantiene por backward compat
    Criteria   []AcceptanceCriteriaInfo `yaml:"criteria,omitempty"`  // ← NUEVO: structured criteria
}
```

**Cambio 2**: Agregar campo `AcceptanceStatus` en el Payload principal para resumen global.

```go
type Payload struct {
    // ... campos existentes
    AcceptanceStatus *AcceptanceSummary `yaml:"acceptance_status,omitempty"` // ← NUEVO
}

// AcceptanceSummary resumen global de criteria al final del pipeline.
type AcceptanceSummary struct {
    Total    int    `yaml:"total"`
    Verified int    `yaml:"verified"`
    Failed   int    `yaml:"failed"`
    Pending  int    `yaml:"pending"`
}
```

---

### 3.10 MCP Tools — Query criteria desde HelixDB

Para que `sdd-verify` pueda leer criteria desde HelixDB, las MCP tools (Python) deben soportar queries por campo `acceptance_criteria`.

**Archivo**: `~/.config/zyrocli/mcp-tools/search_code.py` o una nueva tool `search_criteria.py`

**Opción A (recomendada)**: Extender `search_facts` para también buscar en nodos Task por acceptance_criteria.

**Opción B**: Crear una nueva tool MCP `search_criteria`:

```python
# search_criteria.py (NUEVO)
@mcp.tool()
async def search_criteria(task_id: str = None, status: str = None, phase: str = None) -> str:
    """Search acceptance criteria across Task nodes in HelixDB.
    
    Args:
        task_id: Optional Task ID to filter by
        status: Optional status filter (pending, verified, failed)
        phase: Optional phase filter
        
    Returns:
        JSON string with matching criteria records
    """
    query = "MATCH (t:Task) RETURN t.acceptance_criteria as criteria, t.name as task_name, t.task_id as task_id"
    # ... implementation
```

**Nota**: Esta implementación depende del schema de HelixDB y la capacidad de hacer queries estructuradas. Alternativamente, se puede usar `task_context` para obtener el Task node completo con sus criteria.

---

### 3.11 SDD-Verify Skill — Consumir criteria desde HelixDB

**Archivo**: `.config/opencode/plugins/skills/sdd-verify/SKILL.md`

**Cambio**: El skill debe leer acceptance criteria desde HelixDB (Task nodes) en lugar de depender solo del markdown de verify-report.

**Agregar al execution flow** (entre pasos actuales):

```
3.5. Acceptance Criteria Resolution:
   a. Query HelixDB for Task nodes with phase==current_phase
   b. Extract acceptance_criteria from each Task node
   c. For each criterion with status "pending":
      - Evaluate against implementation evidence
      - If satisfied → update to "verified"
      - If not satisfied → update to "failed"
   d. Persist updated criteria status back to HelixDB
   e. Include criteria compliance in verification report
```

**Cambio en el reporte**: La compliance matrix debe incluir criteria status desde HelixDB:

```
### Acceptance Criteria Compliance
| ID | Description | Task | Status | Evidence |
|----|-------------|------|--------|----------|
| AC-001 | Reject requests without JWT | implement-auth | ✅ VERIFIED | TestAuthMiddleware fails without token |
| AC-002 | Return 401 with message | implement-auth | ❌ FAILED | Returns 403 instead of 401 |
```

---

### 3.12 Pipeline Gate — Verificar criteria antes de avanzar

**Archivo**: `internal/scheduler/scheduler.go` en `Run()` (línea 119)

**Cambio**: Antes de llamar a `ApprovalGate`, verificar que todos los criteria de la fase completada estén en estado "verified".

```go
// Después de ejecutar la fase y antes de approval:

// Verificar acceptance criteria
var criteriaSummary *CriteriaSummary
if boomerangResult != nil && dag != nil {
    criteriaSummary = computeCriteriaSummary(dag)
    if criteriaSummary.Failed > 0 {
        fmt.Printf("\n❌ %d acceptance criteria fallaron en fase %s. No se puede avanzar.\n",
            criteriaSummary.Failed, phaseName)
        results = append(results, &Result{
            Phase: phaseName,
            Status: StatusFail,
            Summary: fmt.Sprintf("%d acceptance criteria failed", criteriaSummary.Failed),
        })
        return results, fmt.Errorf("criteria check: %d failed in phase %s",
            criteriaSummary.Failed, phaseName)
    }
}
```

---

## 4. Conexión con otros specs existentes

### 4.1 `spec-fix-mcp-tools-bugs.md`

**Relación**: Las MCP tools de búsqueda deben soportar queries de acceptance criteria.

**Dependencias**:
- `search_facts` necesita poder buscar en nodos Task por `acceptance_criteria` field.
- `text_search` (helix_client.py) debe soportar `property` configurable (Bug #1 resuelto en ese spec).
- `task_context` debe retornar el campo `acceptance_criteria` en los nodos Task.

**Acción**: Verificar que `task_context.py` incluya `acceptance_criteria` en las properties retornadas para nodos Task.

### 4.2 `spec-fix-subagent-permissions.md`

**Relación**: `sdd-verify` necesita permisos para:
1. **Leer** nodos Task desde HelixDB (read: allow — ya tiene).
2. **Escribir** updates a nodos Task (para actualizar criteria status).

**Cambio requerido**: El spec de permisos debe considerar que `sdd-verify` necesita `write: allow` para actualizar `acceptance_criteria.status` en HelixDB. 

**Estado actual según spec-fix-subagent-permissions.md**:
- `sdd-verify` pasa de `write: allow` → `write: deny` (tabla post-cambio, agente #9).
- **Problema**: Si `sdd-verify` no puede escribir en HelixDB, no puede actualizar el status de criteria.

**Solución**: `sdd-verify` debe ser incluido en la lista de agentes que pueden escribir solo a HelixDB (no a archivos del proyecto). Se puede lograr mediante:
1. **Opción A**: Mantener `write: deny` pero permitir `save_to_helix` vía tool rule en Boundari policy.
2. **Opción B**: Crear una tool MCP específica `update_criteria_status` que no requiera write general.
3. **Opción C**: Que `sdd-verify` delegue la escritura de criteria al orquestador.

**Recomendación**: Opción A es la más alineada con la arquitectura existente (Boundari policy se integra en `spec-activate-boundary-enforcement.md`).

### 4.3 `spec-activate-boundary-enforcement.md`

**Relación**: QualityStep Boundari enforcement debe incluir acceptance criteria como parte de la verificación.

**Cambio sugerido**: Agregar una tool rule `evaluate_criteria` en las Boundari policies de fases F3 y F4.

```yaml
# En phase3-boundari.yaml (agregar)
- name: "evaluate_criteria"
  action: allow
```

Además, el `Enforcer` en `QualityStep` debe verificar que la evaluación de criteria está permitida por la política antes de ejecutarla.

---

## 5. Criterios de Éxito

### 5.1 Funcionales

- [ ] **CE1**: Existe el tipo `AcceptanceCriteria` struct con campos `ID`, `Description`, `Phase`, `Status`, `Source`, `TaskID`.
- [ ] **CE2**: `TaskSpec` tiene campo `AcceptanceCriteria []AcceptanceCriteria` con `omitempty`.
- [ ] **CE3**: `TaskRow` en HelixDB tiene campo `acceptance_criteria` en el JSON del nodo.
- [ ] **CE4**: `saveTaskToHelix()` en MCP server persiste `acceptance_criteria` al crear nodos Task.
- [ ] **CE5**: `QualityStep` evalúa acceptance criteria además de `go build` y task success.
- [ ] **CE6**: Si un acceptance criterion está en estado "pending" y la tarea falla, pasa a "failed".
- [ ] **CE7**: `ApprovalGate` muestra resumen de criteria status y bloquea si hay criteria "failed".
- [ ] **CE8**: `writeHandoff()` incluye tabla de acceptance criteria status.
- [ ] **CE9**: El handoff payload (`handoff.yaml`) incluye `AcceptanceCriteriaInfo` en `UserStory`.
- [ ] **CE10**: `sdd-verify` puede leer acceptance criteria desde HelixDB y actualizar su status.
- [ ] **CE11**: Pipeline no avanza a la siguiente fase si criteria de la fase anterior no están "verified".

### 5.2 No funcionales

- [ ] **CN1**: Backward compatible: fases existentes sin acceptance criteria funcionan sin cambios.
- [ ] **CN2**: Slice vacío de criteria = auto-pass en QualityStep (no bloquea).
- [ ] **CN3**: Criteria son queryables desde HelixDB via `task_context` o `search_facts`.
- [ ] **CN4**: Status de criteria se actualiza sin romper nodes existentes (patch, no replace).
- [ ] **CN5**: `go build ./...` compila sin errores.
- [ ] **CN6**: `go test ./...` pasa sin errores (tests existentes + nuevos).

---

## 6. Pruebas

### 6.1 Pruebas unitarias

| # | Prueba | Archivo | Descripción |
|---|--------|---------|-------------|
| T1 | `AcceptanceCriteria` struct se crea correctamente | `criteria_test.go` (nuevo) | Verificar que los campos se setean y serializan correctamente |
| T2 | `TaskSpec` con criteria se serializa a JSON | `orchestrator_test.go` | `json.Marshal` de TaskSpec con criteria produce el formato esperado |
| T3 | `evaluateCriteria` con todos los criteria verified | `quality_test.go` | DAG con criteria en estado pending, delegate results exitosos → todos a verified |
| T4 | `evaluateCriteria` con criteria que falla | `quality_test.go` | DAG con criteria, delegate result fallido → criterion a failed, retorna false |
| T5 | `evaluateCriteria` con criteria vacío | `quality_test.go` | Slice vacío → retorna true sin evaluar |
| T6 | `evaluateCriteria` sin DAG | `quality_test.go` | DAG nil → retorna true |
| T7 | `ApprovalGate` bloquea con failed criteria | `approval_test.go` | CriteriaSummary con Failed > 0 → retorna false |
| T8 | `ApprovalGate` permite con all verified | `approval_test.go` | CriteriaSummary con all verified → retorna true (misma entrada de usuario) |
| T9 | `ApprovalGate` funciona sin criteria | `approval_test.go` | CriteriaSummary nil → comportamiento normal (no bloquea) |
| T10 | Serialización/deserialización de criteria en HelixDB | `mcp_server_test.go` | criteria → map → JSON → back es consistente |
| T11 | `saveTaskToHelix` con criteria nil | `mcp_server_test.go` | No se pasa acceptance_criteria → nodo se crea sin el campo |
| T12 | `writeHandoff` con criteria summary | `handoff_test.go` | El markdown generado incluye la tabla de criteria |

### 6.2 Pruebas de integración

| # | Prueba | Descripción |
|---|--------|-------------|
| I1 | Pipeline F1→F2 con criteria | Crear spec con criteria, verificar que persisten a F2 vía HelixDB |
| I2 | Pipeline F2→F3 con criteria | Tasks tienen criteria desde diseño, QualityStep los evalúa |
| I3 | Pipeline completo con criteria | F1→F2→F3→F4 con criteria definidos, todos verificados |
| I4 | Pipeline sin criteria (backward compat) | Pipeline legacy, sin acceptance criteria → funciona sin cambios |
| I5 | Criteria persisted in HelixDB | `saveTaskToHelix` → query `task_context` → criteria presentes |

### 6.3 Prueba de regresión

```bash
# Verificar compilación
go build ./...

# Verificar tests existentes no rotos
go test ./internal/boomerang/...
go test ./internal/scheduler/...
go test ./internal/handoff/...
go test ./internal/db/helix/...
```

---

## 7. Riesgos y Mitigaciones

| ID | Riesgo | Probabilidad | Impacto | Mitigación |
|----|--------|-------------|---------|------------|
| R1 | **Backward compatibility rota**: fases existentes sin criteria dejan de funcionar | Baja | Alto | `omitempty` en JSON/YAML. `nil`/slice vacío = auto-pass en evaluación. Tests T5, T6, T11, I4 verifican esto. |
| R2 | **Dependencia circular**: `boomerang` importa `helix` o viceversa | Media | Alto | `AcceptanceCriteria` se define en `boomerang`. `TaskRow` usa `[]map[string]any`. La conversión ocurre solo en el MCP server (`cmd/zyrocli/`), que no crea dependencia circular. |
| R3 | **sdd-verify sin write permission**: no puede actualizar status de criteria en HelixDB | Media | Alto | Opción A del spec de permisos: mantener `write: deny` pero permitir `save_to_helix` via Boundari. O que el orquestador actualice criteria en nombre de sdd-verify. |
| R4 | **Evaluación básica es muy superficial**: solo checkea task success + output | Media | Medio | La evaluación es un primer paso. Se documenta como mejora futura: pattern matching, tests específicos, validación LLM. Mientras tanto, un task exitoso con output es un proxy razonable. |
| R5 | **Criteria en HelixDB no se actualizan si el MCP server no está disponible** | Baja | Medio | Log de warning no bloqueante. El status de criteria se puede actualizar en el siguiente ciclo de QualityStep cuando HelixDB esté disponible. |
| R6 | **Criteria duplicados**: el mismo criterion aparece en múltiples tasks | Media | Baja | ID único por criterion. Si el mismo criterion aparece en múltiples tasks, se evaluará en cada uno. El resumen global puede contar un criterion múltiples veces. Mitigación: documentar que cada criterion debe asignarse a un solo task. |
| R7 | **Handoff markdown muy largo** con muchos criteria | Baja | Baja | La tabla en handoff tiene formato markdown compacto. Para >20 criteria, considerar solo mostrar resumen (✅/❌/⏳) y referenciar a HelixDB para detalle. |

---

## 8. Orden de Implementación Sugerido

| Paso | Archivos | Depende de | Descripción |
|------|----------|-----------|-------------|
| 1 | `internal/boomerang/criteria.go` (NUEVO) | — | Crear tipo `AcceptanceCriteria` y `CriteriaStatus` |
| 2 | `internal/boomerang/orchestrator.go` | Paso 1 | Agregar `AcceptanceCriteria` a `TaskSpec` |
| 3 | `internal/db/helix/types.go` | — | Agregar `AcceptanceCriteria` a `TaskRow` |
| 4 | `cmd/zyrocli/mcp_server.go` | Pasos 1, 3 | Persistir criteria en `saveTaskToHelix()` |
| 5 | `internal/boomerang/quality.go` | Pasos 1, 2 | Agregar `evaluateCriteria()` a `QualityStep` |
| 6 | `internal/boomerang/think.go` | Paso 2 | Inyectar criteria desde PhaseConfigV2 en DAG |
| 7 | `internal/scheduler/approval.go` | Paso 1 | Agregar `CriteriaSummary` y bloqueo en `ApprovalGate` |
| 8 | `internal/scheduler/scheduler.go` | Pasos 5, 7 | Conectar evaluación de criteria antes de approval |
| 9 | `internal/scheduler/handoff.go` | Paso 1 | Incluir criteria status en `writeHandoff()` |
| 10 | `internal/handoff/payload.go` | Paso 1 | Agregar `AcceptanceCriteriaInfo` y `AcceptanceSummary` |
| 11 | `.config/opencode/plugins/skills/sdd-verify/` | Paso 3 | Actualizar skill para leer criteria desde HelixDB |
| 12 | Tests unitarios | Pasos 1-11 | T1 a T12 |
| 13 | Test de integración | Paso 12 | I1 a I5 |

---

## 9. Notas Técnicas para Implementación

### 9.1 Patrones existentes en HelixDB

El patrón para persistir datos estructurados en nodos HelixDB ya existe:
- `CodeNode` usa `path`, `language`, `summary` como properties (ver `spec-fix-mcp-tools-bugs.md`).
- `Fact` usa `content`, `source` como properties.
- `Task` ya tiene `task_id`, `name`, `agent`, `phase`, `status`.

### 9.2 Consideraciones de serialización

- HelixDB almacena properties como JSON. `[]map[string]any` es el formato más portable.
- La serialización de `AcceptanceCriteria` → `map[string]any` se hace en el MCP server.
- La deserialización (lectura) se hace en `sdd-verify` y en `boomerang` (via MemoryStep).

### 9.3 Skills disponibles

- `zyro-sdd-apply`: Implementa cambios siguiendo specs (usará este spec para implementar).
- `zyro-sdd-verify`: Verifica implementación (debe leer criteria desde HelixDB).
- `typescript-advanced-types`: No aplica (el proyecto es Go).

### 9.4 Patrones en HelixDB

Buscar en HelixDB (post-F0) patrones de "acceptance criteria tracking" o "test case management" que puedan informar decisiones de diseño.
