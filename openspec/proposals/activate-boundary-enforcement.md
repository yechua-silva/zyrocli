# Propuesta: Activar Boundary Enforcement en el runtime + PRE-F0 al sistema de políticas

## Intento

El sistema **Boundari** (módulo `internal/boundari/`) fue diseñado como un **enforcement layer** para controlar qué herramientas puede invocar cada fase del pipeline SDD, implementando budgets, reglas por tool y auditoría. Sin embargo, el código revela un **gap de integración crítico**:

1. **Boundari existe pero no se ejecuta.** El `BoomerangOrchestrator` recibe un `boundariLoader` en el constructor y lo almacena, pero **nunca lo invoca** en `runPhaseV2()`. No se crea un `Enforcer`, no se llama `CheckTool()` en ningún step, no se loggean eventos de auditoría. El sistema está totalmente desconectado del ciclo de ejecución.

2. **PRE-F0 no tiene política Boundari.** Existen archivos YAML para F0–F4, pero no existe `phasePRE-F0-boundari.yaml`. El `loader.go` construye el nombre de archivo como `phase{PRE-F0}-boundari.yaml`, falla al no encontrarlo, y cae en `LoadDefaultPolicy` que no tiene un case específico para PRE-F0 (usa el default genérico de solo lectura).

3. **El mensaje del pipeline está desactualizado.** `cmd/zyrocli/run.go` muestra "F0 → F1 → F2 → F3 → F4" cuando el pipeline real ya incluye PRE-F0 como primera fase.

Este cambio resuelve los tres problemas, cerrando la brecha entre la implementación de Boundari y su activación en el runtime.

---

## Alcance

### Incluye

1. **Conectar Boundari en `BoomerangOrchestrator.runPhaseV2()`**
   - Crear un `Enforcer` al inicio de cada fase, cargando la política correspondiente vía `boundariLoader`.
   - Ejecutar `CheckTool()` antes de cada tool call en `DelegateStep` y `SaveStep`.
   - Loggear `AuditEvent` por cada decisión de enforcement (allow/deny).
   - Verificar `IsBudgetExceeded()` al inicio de cada step, abortando la fase si se excedió el budget.
   - Al terminar la fase, persistir el audit log a disco (`boundari/SaveAuditLog`).

2. **Crear política YAML para PRE-F0**
   - Archivo: `internal/boundari/phasePRE-F0-boundari.yaml`
   - Reglas basadas en el skill `zyro-pre-f0`:
     - Tools permitidas: `read_file`, `search_code`, `search_skills`, `task_context`, `web_search`, `web_fetch`, `glob`, `grep`, `save_to_helix` (para guardar facts)
     - Tools denegadas: `write_file`, `edit_file`, `execute_command` (PRE-F0 no debe modificar archivos de proyecto; solo produce `.md` en `openspec/`)
     - Budget acorde: ~30 tool calls, 300s runtime
   - Agregar case `"PRE-F0"` en `LoadDefaultPolicy` como fallback

3. **Actualizar mensaje de pipeline en `cmd/zyrocli/run.go`**
   - Cambiar `Short` de "Execute 5-phase SDD pipeline (F0→F1→F2→F3→F4)" a "Execute 6-phase SDD pipeline (PRE-F0→F0→F1→F2→F3→F4)"
   - Cambiar mensaje de `run.Rune` línea 116 de "F0 → F1 → F2 → F3 → F4" a "PRE-F0 → F0 → F1 → F2 → F3 → F4"
   - Actualizar flag help de `--phase` para listar PRE-F0

### Excluye

- NO se modifica la lógica de los steps individuales (MemoryStep, ThinkStep, etc.) — solo se añade el enforcement wrapper.
- NO se cambia el modelo de datos de Policy/ToolRule/Budget (ya es correcto).
- NO se agrega approval mode interactivo (el `ActionApproval` ya existe pero queda para otro cambio).
- NO se modifica el test suite existente (solo se agregan tests nuevos si aplica).
- NO se toca la skip matrix ni el DAG de tareas.
- NO se implementa UI para visualizar audit logs (solo persistencia a disco).
- NO se modifican las políticas YAML existentes de F0–F4.

---

## Enfoque técnico

### Módulos afectados

| Módulo | Archivo | Cambio |
|--------|---------|--------|
| **Boomerang** | `internal/boomerang/orchestrator.go` | Conectar `boundariLoader` + `Enforcer` en `runPhaseV2()` |
| **Boomerang** | `internal/boomerang/delegate.go` | Agregar `CheckTool()` antes de cada `DispatchTask()` |
| **Boomerang** | `internal/boomerang/save.go` | Agregar `CheckTool()` antes de cada `SaveFact()` |
| **Boundari** | `internal/boundari/phasePRE-F0-boundari.yaml` | **Nuevo archivo** con política de PRE-F0 |
| **Boundari** | `internal/boundari/loader.go` | Agregar case `"PRE-F0"` en `LoadDefaultPolicy()` |
| **CLI** | `cmd/zyrocli/run.go` | Actualizar mensajes de pipeline |

### Flujo modificado en `runPhaseV2()`

```
inicio de runPhaseV2:
  policy, err := o.boundariLoader(config.Phase)
  si err → fallback a LoadDefaultPolicy(config.Phase)
  enforcer := boundari.NewEnforcer(policy)

  para cada step en steps:
    si enforcer.IsBudgetExceeded() → abortar fase con error
    ejecutar step normalmente
      dentro de DelegateStep:
        para cada task en dag.Tasks:
          result := enforcer.CheckTool("dispatch_task", ...)
          si !result.Allowed → loggear AuditEvent + saltar tarea
          boundari.LogAudit(...)
      dentro de SaveStep:
        para cada fact a guardar:
          result := enforcer.CheckTool("save_to_helix", ...)
          si !result.Allowed → loggear AuditEvent + saltar fact
          boundari.LogAudit(...)

  al finalizar fase:
    boundari.SaveAuditLog("audit/boomerang-<phase>-<timestamp>.jsonl")
```

### Política PRE-F0 propuesta

Basada en las reglas del skill `zyro-pre-f0`:
- **Solo lectura** del codebase (explorar, entender)
- **Entrevista** al usuario (vía OpenCode, no requiere tools externas)
- **Escritura de archivos `.md`** (alignment.md, domain-model.md, CONTEXT.md)
- **Guardado en HelixDB** (facts de alineación)
- **Sin ejecución de comandos**, sin modificar código fuente

### Consideración sobre `write_file`

PRE-F0 necesita poder escribir archivos `.md` en `openspec/` y `CONTEXT.md`. Hay dos opciones:
- **Opción A (recomendada):** Permitir `write_file` globalmente en PRE-F0, dado que es una fase de alineación previa a cualquier cambio real y los únicos archivos que escribe son documentación. Es seguro porque PRE-F0 no tiene acceso a modificar código fuente (no hay tareas de implementación).
- **Opción B:** Usar `conditions` en `ToolRule` para restringir `write_file` solo a patrones `openspec/*.md` y `CONTEXT.md`. Más granular pero requiere implementar evaluación de conditions en el Enforcer.

Se adopta **Opción A** por simplicidad y porque el riesgo es bajo: PRE-F0 corre antes de cualquier cambio y su output es puramente documental.

---

## Riesgos

| Riesgo | Impacto | Mitigación |
|--------|---------|------------|
| **Falso positivo: Boundari deniega una tool legítima** | Una fase se aborta incorrectamente | El `boundariLoader` ya tiene fallback a `LoadDefaultPolicy`. Además, el error de enforcement se loggea y la fase reporta `error` sin panickear. |
| **PRE-F0 se queda sin budget** | La alineación queda incompleta si hay muchas preguntas | Budget de 30 tool calls / 300s es conservador. Si es insuficiente, se ajusta el YAML sin cambiar código. |
| **Audit logs ocupan espacio en disco** | Crecimiento de archivos JSONL | Se escribe un archivo por fase con timestamp. En la práctica son kilobytes. Se puede agregar rotación en el futuro. |
| **Enforcer agregado en medio del ciclo** | Steps existentes no esperaban el enforcement | El Enforcer se inyecta en `runPhaseV2()` sin cambiar la firma de los steps. Los steps reciben el enforcer como parámetro adicional o se pasa vía contexto. |
| **La condición `strings.TrimPrefix("PRE-F0", "F")` no produce "PRE-F0"** | Wait — sí produce "PRE-F0" porque el prefijo "F" no está al inicio. | Correcto. `strings.TrimPrefix("PRE-F0", "F")` → `"PRE-F0"` porque la string no empieza con "F". El filename generado será `phasePRE-F0-boundari.yaml`. |

---

## Esfuerzo estimado

**Small** (— 3 archivos modificados, 1 archivo nuevo)

Criterios:
- **Archivos afectados:** 4 (orchestrator.go, delegate.go, save.go, run.go, loader.go + 1 nuevo YAML)
- **Complejidad:** Baja. La API de Boundari ya está completa (Enforcer, CheckTool, AuditEvent). Solo hay que conectarla.
- **Riesgo:** Bajo. El cambio es aditivo, no modifica lógica existente. Los fallos de enforcement se loggean y la fase reporta error controladamente.
- **Tests existentes:** Cubren `Enforcer`, `CheckTool`, `LoadPolicy`, `LoadDefaultPolicy` — sirven como red de seguridad.
- **Estimación:** ~2–4 horas de implementación + 1 hora de testing manual.

---

## Criterios de éxito

- [ ] `BoomerangOrchestrator.runPhaseV2()` crea un `Enforcer` y llama `CheckTool()` antes de cada tool en DelegateStep y SaveStep.
- [ ] Al exceder budget, la fase se aborta con error controlado (no panic).
- [ ] Audit events se persisten a disco al finalizar cada fase.
- [ ] `internal/boundari/phasePRE-F0-boundari.yaml` existe y es válido (`ValidatePolicy` pasa).
- [ ] `LoadDefaultPolicy("PRE-F0")` retorna una política con tools de solo-lectura + `write_file` permitido.
- [ ] `cmd/zyrocli/run.go` muestra "PRE-F0 → F0 → F1 → F2 → F3 → F4" en todos los mensajes.
- [ ] `go test ./internal/boundari/...` y `go test ./internal/boomerang/...` siguen pasando.
- [ ] El flag `--phase PRE-F0` funciona correctamente.

---

## Evidencia de hallazgos

### Hallazgo 1: Boundari no se ejecuta en runPhaseV2()

```go
// orchestrator.go:89-95 — boundariLoader se almacena
type BoomerangOrchestrator struct {
    memoryStore    memory.EngramStore
    boundariLoader func(string) (*boundari.Policy, error)  // ← existe
    taskManager    *TaskManager
    ...
}

// orchestrator.go:121-240 — runPhaseV2() NUNCA usa boundariLoader
func (o *BoomerangOrchestrator) runPhaseV2(...) {
    // No hay: policy, err := o.boundariLoader(config.Phase)
    // No hay: enforcer := boundari.NewEnforcer(policy)
    // No hay: enforcer.CheckTool(...)
    // No hay: boundari.LogAudit(...)
    // No hay: boundari.SaveAuditLog(...)
    ...
}
```

### Hallazgo 2: Falta archivo phasePRE-F0-boundari.yaml

```go
// loader.go:15-16 — el loader construye el filename
phaseNum := strings.TrimPrefix(phase, "F")   // "PRE-F0" → "PRE-F0" (no cambia)
filename := fmt.Sprintf("phase%s-boundari.yaml", phaseNum) // "phasePRE-F0-boundari.yaml"

// Archivos existentes:
// - phase0-boundari.yaml  (F0)
// - phase1-boundari.yaml  (F1)
// - phase2-boundari.yaml  (F2)
// - phase3-boundari.yaml  (F3)
// - phase4-boundari.yaml  (F4)
// - phasePRE-F0-boundari.yaml ← NO EXISTE
```

### Hallazgo 3: LoadDefaultPolicy no tiene case PRE-F0

```go
// loader.go:40 — switch sin case para "PRE-F0"
switch phase {
case "F0":
    ...  // solo lectura
case "F3":
    ...  // permisiva
default:
    // PRE-F0 cae aquí → genérico de solo lectura (ok por ahora)
    // Pero write_file también se deniega en el default
}
```

### Hallazgo 4: Mensaje desactualizado en run.go

```go
// run.go:21
Short: "Execute SDD pipeline (F0→F1→F2→F3→F4)",  // ← falta PRE-F0

// run.go:116
cmd.Println("▶ Iniciando el proceso de desarrollo (F0 → F1 → F2 → F3 → F4)")  // ← falta PRE-F0

// run.go:143
runCmd.Flags().StringVarP(&runPhase, "phase", "p", "", "run a single phase only (F0, F1, F2, F3, F4)")  // ← falta PRE-F0
```

### Hallazgo 5: PRE-F0 ya está completamente integrado en el pipeline

```go
// phase.go:14
PhasePREF0 Phase = "PRE-F0"

// phase_stubs.go:16-82 — PREF0Runner implementado con Run() y Name()
// skip.go:76 — PRE-F0 en la DefaultPhaseMatrix
// think.go:13-14 — ThinkStep con case "PRE-F0"
// run.go:82 — PREF0Runner en el slice de runners
```
