# Activate Boundary Enforcement — Especificación Técnica

## Propósito

Conectar el módulo **Boundari** (`internal/boundari/`) al ciclo de ejecución del **BoomerangOrchestrator** para que las políticas de enforcement (budget, tool rules, auditoría) se apliquen en tiempo real durante cada fase del pipeline SDD. Esto cierra el gap identificado en Fase 0: el `boundariLoader` se inyecta en el constructor pero nunca se invoca en `runPhaseV2()`.

Además, se crea la política YAML para la fase **PRE-F0** (que ya existe como runner y fase en el pipeline pero no tiene archivo boundari) y se actualizan los mensajes de pipeline en `cmd/zyrocli/run.go`.

## Arquitectura actual

### Estado antes del cambio

```
BoomerangOrchestrator {
  boundariLoader: func(string) (*Policy, error)  ← SE ALMACENA
  taskManager:    *TaskManager
  memoryStore:    EngramStore
}

runPhaseV2() {
  // NO usa boundariLoader
  // NO crea Enforcer
  // NO llama CheckTool()
  // NO loggea audit events
  // NO persiste audit log
  for step in steps {
    switch step {
      case StepDelegate: DelegateStep()   ← sin enforcement
      case StepSave:     SaveStep()       ← sin enforcement
    }
  }
}
```

### El gap

1. **`boundariLoader` existe pero está desconectado.** El campo `boundariLoader` se pasa en `NewBoomerangOrchestrator()` (línea 100 de `orchestrator.go`) y se almacena (línea 106), pero `runPhaseV2()` nunca lo llama.
2. **No existe `phasePRE-F0-boundari.yaml`.** El loader construye el filename como `phase{PRE-F0}-boundari.yaml`, falla al buscar, y `LoadDefaultPolicy()` no tiene un `case "PRE-F0"` — cae en el default genérico que deniega `write_file`, lo cual es incorrecto para PRE-F0 que necesita escribir `.md`.
3. **Los mensajes del pipeline están desactualizados.** `run.go` muestra "F0 → F1 → F2 → F3 → F4" cuando el pipeline real ya incluye PRE-F0 como primera fase.

## Arquitectura propuesta

### Flujo de `runPhaseV2()` con Boundari activado

```
runPhaseV2(ctx, config):
  |
  |-- 1. policy, err := o.boundariLoader(config.Phase)
  |    err → policy = boundari.LoadDefaultPolicy(config.Phase)
  |
  |-- 2. enforcer := boundari.NewEnforcer(policy)
  |
  |-- 3. boundari.ClearAuditLog()  ← limpia eventos de fases anteriores
  |
  |-- 4. Para cada step en steps:
  |     |
  |     |-- 4a. ¿IsBudgetExceeded()? → abortar fase con error controlado
  |     |
  |     |-- 4b. Ejecutar step (switch):
  |     |     - MemoryStep:  sin enforcement (solo lectura)
  |     |     - ThinkStep:   sin enforcement (solo genera DAG en memoria)
  |     |     - DelegateStep: pasa enforcer, CheckTool("dispatch_task") por tarea
  |     |     - GitStep:     sin enforcement (comando local controlado)
  |     |     - QualityStep: sin enforcement directo, repite DelegateStep con enforcer
  |     |     - SaveStep:    pasa enforcer, CheckTool("save_to_helix") por fact
  |
  |-- 5. Al finalizar: boundari.SaveAuditLog("audit/boomerang-{phase}-{ts}.jsonl")
  |
  v
PhaseResult
```

### Diagrama de responsabilidades

```
┌─────────────────────────────────────────────────────────┐
│                   runPhaseV2()                          │
│                                                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────┐ │
│  │ Memory   │  │ Think    │  │ Delegate │  │ Save   │ │
│  │ Step     │  │ Step     │  │ Step     │  │ Step   │ │
│  └──────────┘  └──────────┘  └──────────┘  └────────┘ │
│       │             │             │             │       │
│       │             │             │             │       │
│       ▼             ▼             ▼             ▼       │
│  ┌──────────────────────────────────────────────┐       │
│  │           boundari.Enforcer                  │       │
│  │  ┌──────────┐  ┌──────────┐  ┌────────────┐ │       │
│  │  │CheckTool │  │IsBudget  │  │ auditLogger│ │       │
│  │  │          │  │Exceeded  │  │            │ │       │
│  │  └──────────┘  └──────────┘  └────────────┘ │       │
│  └──────────────────────────────────────────────┘       │
│                                                         │
│  ┌──────────────────────────────────────────────┐       │
│  │         boundari.Policy (cargada)             │       │
│  │  Datos: Budget, []ToolRule                    │       │
│  │  Fuente: YAML o LoadDefaultPolicy fallback    │       │
│  └──────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────┘
```

## Especificación detallada

### Archivo 1: `internal/boundari/phasePRE-F0-boundari.yaml` (NUEVO)

**Ubicación:** `internal/boundari/phasePRE-F0-boundari.yaml`

**Contenido:**

```yaml
version: "1.0"
phase: "PRE-F0"
description: "Alineación de dominio — solo lectura + escritura de documentos .md. No ejecutar comandos."
budget:
  max_tool_calls: 30
  max_runtime_seconds: 300
tools:
  - name: "read_file"
    action: allow
  - name: "search_code"
    action: allow
  - name: "search_skills"
    action: allow
  - name: "task_context"
    action: allow
  - name: "web_search"
    action: allow
  - name: "web_fetch"
    action: allow
  - name: "glob"
    action: allow
  - name: "grep"
    action: allow
  - name: "write_file"
    action: allow
    # Opción A: write_file global. PRE-F0 solo escribe .md en openspec/ y CONTEXT.md.
    # El riesgo es bajo porque PRE-F0 corre antes de cualquier modificación de código.
  - name: "save_to_helix"
    action: allow
  - name: "dispatch_task"
    action: allow
  - name: "edit_file"
    action: deny
  - name: "execute_command"
    action: deny
```

**Decisiones de diseño:**
- `write_file: allow` sin restricciones (Opción A de la propuesta). PRE-F0 solo produce archivos `.md` de documentación. No hay tareas de implementación en esta fase.
- `dispatch_task: allow` necesario porque la skip matrix de PRE-F0 incluye `StepDelegate`.
- Budget conservador (30 calls / 300s) porque PRE-F0 es una fase corta de alineación.

---

### Archivo 2: `internal/boundari/loader.go` — Modificar `LoadDefaultPolicy()`

**Cambio:** Agregar `case "PRE-F0"` en el switch de `LoadDefaultPolicy()`.

```go
func LoadDefaultPolicy(phase string) *Policy {
    p := &Policy{
        Version: "1.0",
        Phase:   phase,
        Budget:  Budget{MaxToolCalls: 50, MaxRuntimeSecs: 300},
    }
    switch phase {
    case "PRE-F0":  // ← NUEVO
        p.Description = "Alineación de dominio — lectura + .md (fallback)"
        p.Budget = Budget{MaxToolCalls: 30, MaxRuntimeSecs: 300}
        p.Tools = []ToolRule{
            {Name: "read_file", Action: ActionAllow},
            {Name: "search_code", Action: ActionAllow},
            {Name: "search_skills", Action: ActionAllow},
            {Name: "task_context", Action: ActionAllow},
            {Name: "web_search", Action: ActionAllow},
            {Name: "web_fetch", Action: ActionAllow},
            {Name: "glob", Action: ActionAllow},
            {Name: "grep", Action: ActionAllow},
            {Name: "write_file", Action: ActionAllow},
            {Name: "save_to_helix", Action: ActionAllow},
            {Name: "dispatch_task", Action: ActionAllow},
            {Name: "edit_file", Action: ActionDeny},
            {Name: "execute_command", Action: ActionDeny},
        }
    case "F0":
        // ... existente, PERO añadir dispatch_task y save_to_helix:
        // {Name: "dispatch_task", Action: ActionAllow},
        // {Name: "save_to_helix", Action: ActionAllow},
    case "F3":
        // ... existente, PERO añadir:
        // {Name: "dispatch_task", Action: ActionAllow},
        // {Name: "save_to_helix", Action: ActionAllow},
    default:
        // ... existente, PERO añadir:
        // {Name: "dispatch_task", Action: ActionAllow},
        // {Name: "save_to_helix", Action: ActionAllow},
    }
    return p
}
```

**Nota sobre YAMLs existentes (F0–F4):** Las políticas YAML actuales (`phase0-boundari.yaml` a `phase4-boundari.yaml`) no contienen las entradas `dispatch_task` ni `save_to_helix` porque fueron escritas pensando en tools de agente, no del orquestador. Para que `CheckTool("dispatch_task")` no deniegue la delegación en fases existentes, se AGREGAN estas entradas a los YAMLs como tool rules adicionales. Esto NO modifica reglas existentes — solo añade las que faltan para el orquestador. Ver "Riesgos" para la alternativa.

---

### Archivo 3: `internal/boomerang/orchestrator.go` — Conectar Boundari en `runPhaseV2()`

**3a. Crear Enforcer al inicio de `runPhaseV2()`**

Después de la declaración de variables compartidas (línea 141), antes del loop de steps (línea 145):

```
1. Llamar o.boundariLoader(config.Phase)
2. Si error → policy = boundari.LoadDefaultPolicy(config.Phase)
3. enforcer := boundari.NewEnforcer(policy)
4. boundari.ClearAuditLog()  // limpiar eventos de fases anteriores
```

**3b. Budget check antes de cada step**

Al inicio de cada `case` en el switch de steps, agregar:

```
if enforcer.IsBudgetExceeded() {
    return nil, fmt.Errorf("boundari: budget exceeded for phase %s", config.Phase)
}
```

Esto aplica a: `StepMemory`, `StepThink`, `StepDelegate`, `StepGit`, `StepQuality`, `StepSave`.

**3c. Pasar enforcer a DelegateStep y SaveStep**

Modificar las llamadas:

- Línea 167: `o.DelegateStep(ctx, dag, config.Phase, enforcer)`
- Línea 198 (dentro de QualityStep retry): `o.DelegateStep(ctx, dag, config.Phase, enforcer)`
- Línea 207: `o.SaveStep(ctx, config.Phase, delegateResult, nil, enforcer)`

**3d. Persistir audit log al finalizar la fase**

Después del switch de steps (línea 213), antes del cálculo de tokens (línea 216):

```
// Persistir audit log
auditDir := "audit"
auditFile := fmt.Sprintf("boomerang-%s-%d.jsonl", config.Phase, time.Now().Unix())
auditPath := filepath.Join(auditDir, auditFile)
if err := boundari.SaveAuditLog(auditPath); err != nil {
    // warning no bloqueante — la fase ya terminó
    fmt.Fprintf(os.Stderr, "⚠ boundari: error saving audit log: %v\n", err)
}
```

**3e. Import adicionales necesarios**

Agregar a los imports de `orchestrator.go`:
- `"fmt"` (si no está)
- `"os"` 
- `"path/filepath"` (si no está)
- `"github.com/secko/zyrocli/internal/boundari"` (ya existe)

---

### Archivo 4: `internal/boomerang/delegate.go` — Agregar `CheckTool` antes de `DispatchTask`

**4a. Cambiar firma**

```go
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string, enforcer *boundari.Enforcer) (*DelegateResult, error) {
```

**4b. Si enforcer es nil, saltar enforcement (backward compatibility)**

```go
if enforcer == nil {
    // No enforcement — usar un default que permite todo
    // (para tests que llaman DelegateStep directamente sin enforcer)
}
```

**4c. CheckTool antes de cada DispatchTask**

Dentro del `for _, task := range dag.Tasks` loop, antes de `o.taskManager.DispatchTask(...)`:

```
result := enforcer.CheckTool("dispatch_task", map[string]any{
    "task_name": task.Name,
    "agent":     task.Agent,
    "phase":     phase,
})
boundari.LogAudit(boundari.AuditEvent{
    Phase:      phase,
    Tool:       "dispatch_task",
    Allowed:    result.Allowed,
    Reason:     result.Reason,
    DurationMs: 0,
})
if !result.Allowed {
    tr := TaskResult{
        TaskName: task.Name,
        Success:  false,
        Output:   fmt.Sprintf("denied by boundari: %s", result.Reason),
    }
    result.TaskResults[task.Name] = tr
    // NO incrementar NodesCreated
    continue  // saltar esta tarea
}
```

**4d. Consideración sobre budget**

`CheckTool` internamente ya contabiliza `usage.ToolCalls`. Cada tarea despachada = 1 tool call contra el budget.

---

### Archivo 5: `internal/boomerang/save.go` — Agregar `CheckTool` antes de `SaveFact`

**5a. Cambiar firma**

```go
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte, enforcer *boundari.Enforcer) (*SaveResult, error) {
```

**5b. Si enforcer es nil, saltar enforcement**

```go
if enforcer == nil {
    // backward compatibility para tests directos
}
```

**5c. CheckTool antes de cada SaveFact**

Dentro del `for _, tr := range delegateResult.TaskResults` loop, antes de `o.memoryStore.SaveFact(...)`:

```
result := enforcer.CheckTool("save_to_helix", map[string]any{
    "task_name": tr.TaskName,
    "phase":     phase,
})
boundari.LogAudit(boundari.AuditEvent{
    Phase:      phase,
    Tool:       "save_to_helix",
    Allowed:    result.Allowed,
    Reason:     result.Reason,
    DurationMs: 0,
})
if !result.Allowed {
    continue  // saltar este fact
}
```

---

### Archivo 6: `cmd/zyrocli/run.go` — Actualizar mensajes de pipeline

**6a. Línea 21 — `Short`**

Cambiar de:
```go
Short: "Execute SDD pipeline (F0→F1→F2→F3→F4)",
```
a:
```go
Short: "Execute SDD pipeline (PRE-F0→F0→F1→F2→F3→F4)",
```

**6b. Líneas 22-27 — `Long`**

Cambiar de:
```go
Long: `Execute the 5-phase SDD pipeline (F0→F1→F2→F3→F4) sequentially
...
Flags:
  --phase F0   Run a single phase only (F0, F1, F2, F3, or F4)`,
```
a:
```go
Long: `Execute the 6-phase SDD pipeline (PRE-F0→F0→F1→F2→F3→F4) sequentially
...
Flags:
  --phase F0   Run a single phase only (PRE-F0, F0, F1, F2, F3, or F4)`,
```

**6c. Línea 116 — Mensaje del pipeline**

Cambiar de:
```go
cmd.Println("▶ Iniciando el proceso de desarrollo (F0 → F1 → F2 → F3 → F4)")
```
a:
```go
cmd.Println("▶ Iniciando el proceso de desarrollo (PRE-F0 → F0 → F1 → F2 → F3 → F4)")
```

**6d. Línea 143 — Flag help de `--phase`**

Cambiar de:
```go
runCmd.Flags().StringVarP(&runPhase, "phase", "p", "", "run a single phase only (F0, F1, F2, F3, F4)")
```
a:
```go
runCmd.Flags().StringVarP(&runPhase, "phase", "p", "", "run a single phase only (PRE-F0, F0, F1, F2, F3, F4)")
```

**Nota:** La validación de fase en línea 98-107 ya lista correctamente `PRE-F0, F0, F1, F2, F3, F4`. No necesita cambios.

---

### Archivo 7: YAMLs existentes (F0–F4) — Agregar tool rules faltantes

Aunque la propuesta dice "NO modificar YAMLs existentes", es necesario AGREGAR dos tool rules que no existían antes porque el enforcement no estaba activo. No se modifica ninguna regla existente.

| Archivo | Tool rule a agregar |
|---------|-------------------|
| `phase0-boundari.yaml` | `{name: "dispatch_task", action: allow}` |
| `phase1-boundari.yaml` | `{name: "dispatch_task", action: allow}`, `{name: "save_to_helix", action: allow}` |
| `phase2-boundari.yaml` | `{name: "dispatch_task", action: allow}` (save_to_helix ya está como deny) |
| `phase3-boundari.yaml` | `{name: "dispatch_task", action: allow}`, `{name: "save_to_helix", action: allow}` |
| `phase4-boundari.yaml` | `{name: "dispatch_task", action: allow}` (save_to_helix ya está como allow) |

**Razonamiento:** Sin estas entradas, `CheckTool("dispatch_task")` retornaría `denied` para TODAS las fases existentes porque la tool no está listada (default deny en el enforcer). Esto rompería la ejecución de F0–F4. La alternativa es cambiar la semántica del enforcer a "default allow" para tools no listadas, pero eso debilita el modelo de seguridad.

---

### Resumen de cambios por archivo

| Archivo | Tipo de cambio | Líneas aproximadas |
|---------|---------------|-------------------|
| `internal/boundari/phasePRE-F0-boundari.yaml` | **NUEVO** | 28 líneas |
| `internal/boundari/loader.go` | Modificar `LoadDefaultPolicy()`: +12 líneas (case PRE-F0) + agregar dispatch_task/save_to_helix a otros cases | +15 líneas |
| `internal/boomerang/orchestrator.go` | Modificar `runPhaseV2()`: crear enforcer, budget checks, pasar enforcer, save audit log | +25 líneas |
| `internal/boomerang/delegate.go` | Modificar firma + agregar CheckTool loop | +20 líneas |
| `internal/boomerang/save.go` | Modificar firma + agregar CheckTool loop | +18 líneas |
| `cmd/zyrocli/run.go` | Actualizar 4 strings de mensajes | 4 cambios de 1 línea c/u |
| `phase*-boundari.yaml` (F0–F4) | Agregar dispatch_task y save_to_helix | 1–2 líneas c/u |

## Criterios de éxito

### Criterios funcionales

- [ ] **C1:** `BoomerangOrchestrator.runPhaseV2()` crea un `Enforcer` al inicio usando `o.boundariLoader(config.Phase)`.
- [ ] **C2:** Si `boundariLoader` falla, se usa `LoadDefaultPolicy(config.Phase)` como fallback.
- [ ] **C3:** `CheckTool("dispatch_task")` se ejecuta antes de cada `DispatchTask()` en `DelegateStep`. Si denegado, la tarea se salta y se loggea un `AuditEvent`.
- [ ] **C4:** `CheckTool("save_to_helix")` se ejecuta antes de cada `SaveFact()` en `SaveStep`. Si denegado, el fact se salta y se loggea un `AuditEvent`.
- [ ] **C5:** `IsBudgetExceeded()` se verifica al inicio de cada step. Si excedido, la fase se aborta con error controlado (no panic).
- [ ] **C6:** Los audit events se persisten a disco (`audit/boomerang-{phase}-{ts}.jsonl`) al finalizar cada fase.
- [ ] **C7:** El audit log se limpia (`ClearAuditLog()`) al iniciar cada fase para evitar acumulación entre fases.
- [ ] **C8:** `internal/boundari/phasePRE-F0-boundari.yaml` existe, es válido (`ValidatePolicy` pasa), y contiene las tools: `read_file`, `search_code`, `search_skills`, `task_context`, `web_search`, `web_fetch`, `glob`, `grep`, `write_file` (allow), `save_to_helix` (allow), `dispatch_task` (allow), `edit_file` (deny), `execute_command` (deny).
- [ ] **C9:** `LoadDefaultPolicy("PRE-F0")` retorna una política con las mismas tools que el YAML (budget 30/300).
- [ ] **C10:** `cmd/zyrocli/run.go` muestra "PRE-F0 → F0 → F1 → F2 → F3 → F4" en `Short`, `Long`, mensaje de pipeline y flag `--phase`.
- [ ] **C11:** El flag `--phase PRE-F0` funciona correctamente (ya validado en `AllPhases`).
- [ ] **C12:** `DelegateStep` y `SaveStep` aceptan un parámetro `*boundari.Enforcer` como último argumento. Si es `nil`, el enforcement se salta (backward compatibility para tests directos).

### Criterios de no-regresión

- [ ] **N1:** `go test ./internal/boundari/...` pasa sin errores.
- [ ] **N2:** `go test ./internal/boomerang/...` pasa sin errores.
- [ ] **N3:** `go build ./...` compila sin errores.
- [ ] **N4:** El pipeline completo (`zyrocli run`) arranca y muestra el mensaje actualizado.

## Pruebas

### Tests existentes que se modifican

| Test | Cambio necesario |
|------|-----------------|
| `TestDelegateStep` (boomerang_test.go:118) | Actualizar llamada a `DelegateStep` con `enforcer=nil` (backward compat). Verificar que el resultado sigue siendo correcto. |
| `TestSaveStep` (boomerang_test.go:158) | Actualizar llamada a `SaveStep` con `enforcer=nil`. Verificar FactsSaved=2. |
| `TestRunPhase` (boomerang_test.go:179) | NO requiere cambios porque `RunPhase` → `runPhaseV2` crea el enforcer internamente usando `mockBoundariLoader`. |
| `TestRunPhaseV2WithCustomSteps` (boomerang_test.go:659) | NO requiere cambios. |
| `TestQualityStep` (boomerang_test.go:138) | NO requiere cambios (QualityStep no recibe enforcer directamente). |
| `TestAllPoliciesLoad` (boundari_test.go:145) | Agregar `"PRE-F0"` al slice de phases para verificar que el nuevo YAML carga correctamente. |
| `TestLoadDefaultPolicy` (boundari_test.go:26) | Agregar subtest para `"PRE-F0"` que verifique `write_file=allow`. |

### Nuevos tests

No se requieren tests nuevos obligatorios, pero se recomiendan:

1. **Test que el Enforcer se crea correctamente en runPhaseV2:**
   - Mock `boundariLoader` que retorna una política conocida
   - Ejecutar `runPhaseV2` con fase "PRE-F0"
   - Verificar que se llama `CheckTool` (observable via audit log)
   - Verificar que audit log se persiste

2. **Test que CheckTool deniega en SaveStep con política restrictiva:**
   - Crear `SaveStep` con un enforcer que deniega `save_to_helix`
   - Verificar que `FactsSaved` es 0

3. **Test de budget excedido:**
   - Crear enforcer con budget=0
   - Verificar que `IsBudgetExceeded()` retorna true
   - Verificar que `runPhaseV2` aborta con error

### Prueba de integración manual

```bash
# Probar que la política PRE-F0 carga correctamente
cd internal/boundari && go test -run TestAllPoliciesLoad

# Probar que el orquestador crea el enforcer
cd internal/boomerang && go test -run TestRunPhase

# Probar --phase PRE-F0 desde CLI (simulado)
cd /tmp/test-project && zyrocli run --phase PRE-F0
```

## Riesgos

| # | Riesgo | Impacto | Probabilidad | Mitigación |
|---|--------|---------|-------------|------------|
| R1 | **YAMLs F0–F4 sin `dispatch_task`** | `CheckTool("dispatch_task")` deniega todas las tareas en fases existentes. Pipeline se rompe. | **Alta** — los YAMLs actuales no tienen esta tool. | **Solución A (recomendada):** Agregar `{name: "dispatch_task", action: allow}` a todos los YAMLs F0–F4. **Solución B:** Modificar `CheckTool` para que tools no listadas sean "allow by default" — pero esto debilita la seguridad. Se adopta Solución A. |
| R2 | **F0 YAML tiene `save_to_helix: deny`** | `CheckTool("save_to_helix")` deniega los saves en F0. `FactsSaved` sería 0. | Media — el YAML de F0 explicitamente deniega. | Decisión de diseño: F0 fue diseñada como "solo lectura". Si se quiere que F0 guarde facts, cambiar el YAML a `allow`. Si no, el comportamiento es correcto (F0 no persiste nada). |
| R3 | **Audit log singleton (`defaultAuditLogger`) no es thread-safe** | Si dos fases corren concurrentemente, los audit events se mezclan. | Baja — actualmente las fases son secuenciales. | `ClearAuditLog()` al inicio de cada fase + `SaveAuditLog()` al final. Si se implementa paralelismo en el futuro, migrar a un logger por instancia de Enforcer. |
| R4 | **Budget excedido aborta la fase abruptamente** | Si una fase está cerca del límite, el último tool call puede exceder el budget y cancelar todo. | Media — el budget es un límite duro. | El error es controlado (no panic). El mensaje indica claramente que el budget se excedió. El usuario puede re-ejecutar la fase con un budget mayor ajustando el YAML. |
| R5 | **`strings.TrimPrefix("PRE-F0", "F")`** | ¿Produce "PRE-F0" o algo inesperado? | Muy baja — `TrimPrefix` solo elimina el prefijo si está al inicio. "PRE-F0" no empieza con "F", así que retorna "PRE-F0". El filename generado es `phasePRE-F0-boundari.yaml`. | Verificado: el filename es correcto. |
| R6 | **Fase PRE-F0 sin Boomerang en scheduler** | `PREF0Runner.Run()` (phase_stubs.go) abre OpenCode directamente, no usa `BoomerangOrchestrator`. El enforcement nunca se ejecuta para PRE-F0. | **Alta** — el scheduler ejecuta `PREF0Runner.Run()` que no pasa por `runPhaseV2()`. | Verificar `scheduler.go` línea 61-78: cuando `cfg.Boomerang != nil`, se usa `Boomerang.RunPhase()` que SÍ pasa por `runPhaseV2()`. Si el scheduler está configurado con `cfg.Boomerang`, PRE-F0 sí tiene enforcement. Si no, no. Esto se verifica en la integración. |

## Notas técnicas

### Dependencias

- El módulo `internal/boundari` ya está importado en `internal/boomerang/orchestrator.go` (línea 7).
- La interfaz `EngramStore` del paquete `memory` ya está importada.
- `time.Now()` y `fmt.Sprintf` están disponibles en `orchestrator.go`.
- `path/filepath` y `os` pueden necesitar ser agregados a los imports de `orchestrator.go`.

### Orden de implementación sugerido

1. Crear `internal/boundari/phasePRE-F0-boundari.yaml` (archivo nuevo, cero riesgo de regresión)
2. Modificar `internal/boundari/loader.go` (agregar case PRE-F0 + dispatch_task/save_to_helix en otros cases)
3. Modificar `internal/boomerang/delegate.go` (firma + CheckTool)
4. Modificar `internal/boomerang/save.go` (firma + CheckTool)
5. Modificar `internal/boomerang/orchestrator.go` (enforcer en runPhaseV2)
6. Modificar `cmd/zyrocli/run.go` (mensajes)
7. Modificar tests existentes para reflejar nuevas firmas
8. `go build ./...` y `go test ./...` para verificar

### Referencias

- Propuesta: `openspec/proposals/activate-boundary-enforcement.md`
- Código Boundari: `internal/boundari/enforcer.go`, `internal/boundari/loader.go`, `internal/boundari/types.go`
- Orquestador: `internal/boomerang/orchestrator.go`
- Steps: `internal/boomerang/delegate.go`, `internal/boomerang/save.go`
- CLI: `cmd/zyrocli/run.go`
- Políticas existentes: `internal/boundari/phase{0..4}-boundari.yaml`
