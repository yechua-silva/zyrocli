# Fase 1 — Phase Skip Matrix: Tareas Atómicas

> **Fecha:** 17 Junio 2026  
> **Versión:** 1.0  
> **Spec:** [spec.md](./spec.md)  
> **Diseño:** [design.md](./design.md)  

---

## Resumen de Tareas

| ID | Nombre | Dependencias | LOC | Archivos |
|----|--------|-------------|-----|----------|
| **T1** | Definir tipos base | — | ~20 | `skip.go` |
| **T2** | Implementar DefaultPhaseMatrix + ShouldRun + ActiveSteps | T1 | ~40 | `skip.go` |
| **T3** | Implementar ValidateMatrix + AllSteps | T1 | ~30 | `skip.go` |
| **T4** | Modificar RunPhase() para usar skip matrix | T2 | ~35 | `orchestrator.go` |
| **T5** | Tests unitarios de matriz (ShouldRun, ActiveSteps, ValidateMatrix) | T3 | ~80 | `boomerang_test.go` |
| **T6** | Tests de integración (RunPhase con skip matrix) | T4 | ~80 | `boomerang_test.go` |
| **T7** | Tests de backward compatibility | T4 | ~40 | `boomerang_test.go` |

**Total estimado:** ~325 LOC, ~7 tareas atómicas.

---

## T1: Definir tipos base (Step, StepStatus, StepOutput)

### Descripción

Crear el archivo `internal/boomerang/skip.go` con los tipos enumerados `Step` y `StepStatus`, el struct `StepOutput`, y sus métodos `String()`.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/skip.go` | **Nuevo** | 1-45: tipos, constantes, String() |

### Código a implementar

```go
// package boomerang ... (package declaration)

// Step identifica cada paso del ciclo Boomerang.
type Step int

const (
    StepMemory   Step = iota // 0
    StepThink                // 1
    StepDelegate             // 2
    StepGit                  // 3
    StepQuality              // 4
    StepSave                 // 5
)

func (s Step) String() string {
    switch s {
    case StepMemory:   return "Memory"
    case StepThink:    return "Think"
    case StepDelegate: return "Delegate"
    case StepGit:      return "Git"
    case StepQuality:  return "Quality"
    case StepSave:     return "Save"
    default:           return "Unknown"
    }
}

type StepStatus int

const (
    StepPending StepStatus = iota
    StepRunning
    StepDone
    StepSkipped
    StepFailed
)

func (s StepStatus) String() string {
    switch s {
    case StepPending: return "pending"
    case StepRunning: return "running"
    case StepDone:    return "done"
    case StepSkipped: return "skipped"
    case StepFailed:  return "failed"
    default:          return "unknown"
    }
}

type StepOutput struct {
    Step     Step   `json:"step"`
    TaskName string `json:"task_name,omitempty"`
    Output   string `json:"output,omitempty"`
    Duration string `json:"duration,omitempty"`
}
```

### Dependencias

Ninguna. T1 es la tarea raíz.

### Criterios de aceptación

- [ ] `Step(String())` retorna `"Memory"` para `StepMemory`, `"Think"` para `StepThink`, etc.
- [ ] `Step(String())` retorna `"Unknown"` para valores inválidos (ej. `Step(-1)`)
- [ ] `StepStatus(String())` retorna `"pending"` para `StepPending`, `"done"` para `StepDone`, etc.
- [ ] `StepStatus(String())` retorna `"unknown"` para valores inválidos
- [ ] `StepOutput` struct compila con tags JSON
- [ ] `go build ./internal/boomerang/...` compila sin errores

### Estimación

- **LOC:** ~20
- **Tiempo:** ~15 minutos
- **Import:** solo `fmt` (no se necesita en String() de Step porque es switch)

### Subtareas

1.1. Declarar package + import "fmt"  
1.2. Definir `Step` type + const iota  
1.3. Implementar `Step.String()`  
1.4. Definir `StepStatus` type + const iota  
1.5. Implementar `StepStatus.String()`  
1.6. Definir `StepOutput` struct  

---

## T2: Implementar DefaultPhaseMatrix + ShouldRun + ActiveSteps

### Descripción

Agregar a `skip.go` el tipo `PhaseStepMatrix`, la función `DefaultPhaseMatrix()` con la matriz canónica, y los métodos `ShouldRun()` y `ActiveSteps()`.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/skip.go` | **Modificar** (agregar después de T1) | 46-95: tipos, matriz, métodos |

### Código a implementar

```go
type PhaseStepMatrix map[string][]Step

func DefaultPhaseMatrix() PhaseStepMatrix {
    return PhaseStepMatrix{
        "F0": {StepMemory, StepThink, StepDelegate, StepSave},
        "F1": {StepMemory, StepThink, StepDelegate, StepSave},
        "F2": {StepMemory, StepThink, StepDelegate, StepSave},
        "F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
        "F4": {StepMemory, StepDelegate, StepGit, StepSave},
    }
}

func (m PhaseStepMatrix) ShouldRun(phase string, step Step) bool {
    steps, ok := m[phase]
    if !ok {
        return true // default seguro
    }
    for _, s := range steps {
        if s == step {
            return true
        }
    }
    return false
}

func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
    steps, ok := m[phase]
    if !ok {
        return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
    }
    result := make([]Step, len(steps))
    copy(result, steps)
    return result
}
```

### Dependencias

- **T1** (necesita `Step` type y sus constantes)

### Criterios de aceptación

- [ ] `DefaultPhaseMatrix()` retorna las 5 fases F0-F4
- [ ] `ShouldRun("F0", StepGit)` = `false`
- [ ] `ShouldRun("F0", StepQuality)` = `false`
- [ ] `ShouldRun("F0", StepMemory)` = `true`
- [ ] `ShouldRun("F3", StepGit)` = `true`
- [ ] `ShouldRun("F3", StepQuality)` = `true`
- [ ] `ShouldRun("F4", StepThink)` = `false`
- [ ] `ShouldRun("F4", StepQuality)` = `false`
- [ ] `ShouldRun("F99", StepMemory)` = `true` (fase desconocida)
- [ ] `ActiveSteps("F0")` retorna 4 steps: `[Memory, Think, Delegate, Save]`
- [ ] `ActiveSteps("F3")` retorna 6 steps
- [ ] `ActiveSteps("F4")` retorna 4 steps: `[Memory, Delegate, Git, Save]`
- [ ] `ActiveSteps("F99")` retorna 6 steps (fase desconocida)
- [ ] `ActiveSteps()` retorna **copia** del slice (mutación de la copia no afecta la matriz)
- [ ] `go build ./internal/boomerang/...` compila sin errores

### Estimación

- **LOC:** ~40
- **Tiempo:** ~25 minutos

### Subtareas

2.1. Definir `PhaseStepMatrix` type  
2.2. Implementar `DefaultPhaseMatrix()` con matriz canónica  
2.3. Implementar `ShouldRun()` con default seguro  
2.4. Implementar `ActiveSteps()` con copia defensiva  

---

## T3: Implementar ValidateMatrix + AllSteps

### Descripción

Agregar a `skip.go` la función `AllSteps()` (helper) y `ValidateMatrix()` con todas las reglas de validación, incluyendo el tipo de error `ErrInvalidMatrix`.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/skip.go` | **Modificar** (agregar después de T2) | 96-130: AllSteps, ErrInvalidMatrix, ValidateMatrix |

### Código a implementar

```go
func AllSteps() []Step {
    return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
}

type ErrInvalidMatrix struct {
    Phase  string
    Reason string
}

func (e *ErrInvalidMatrix) Error() string {
    return fmt.Sprintf("skip matrix: phase %s: %s", e.Phase, e.Reason)
}

func ValidateMatrix(matrix PhaseStepMatrix) error {
    requiredPhases := []string{"F0", "F1", "F2", "F3", "F4"}
    maxStep := int(StepSave) // 5

    for _, phase := range requiredPhases {
        steps, ok := matrix[phase]
        if !ok {
            return &ErrInvalidMatrix{Phase: phase, Reason: "phase not defined in matrix"}
        }
        if len(steps) == 0 {
            return &ErrInvalidMatrix{Phase: phase, Reason: "must have at least one step"}
        }

        seen := make(map[Step]bool)
        for _, s := range steps {
            if int(s) < 0 || int(s) > maxStep {
                return &ErrInvalidMatrix{
                    Phase: phase, Reason: fmt.Sprintf("invalid step value %d", s),
                }
            }
            if seen[s] {
                return &ErrInvalidMatrix{
                    Phase: phase, Reason: fmt.Sprintf("duplicate step %s", s),
                }
            }
            seen[s] = true
        }

        hasStep := func(s Step) bool { return seen[s] }

        if phase == "F4" && !hasStep(StepSave) {
            return &ErrInvalidMatrix{Phase: phase, Reason: "F4 must include Save step"}
        }
        if phase == "F3" {
            if !hasStep(StepQuality) {
                return &ErrInvalidMatrix{Phase: phase, Reason: "F3 must include Quality step"}
            }
            if !hasStep(StepGit) {
                return &ErrInvalidMatrix{Phase: phase, Reason: "F3 must include Git step"}
            }
        }
    }
    return nil
}
```

### Dependencias

- **T1** (necesita `Step` type y constantes)
- **T2** (necesita `PhaseStepMatrix` type, aunque ValidateMatrix podría funcionar sin él)

### Criterios de aceptación

- [ ] `AllSteps()` retorna 6 steps en orden: Memory, Think, Delegate, Git, Quality, Save
- [ ] `ValidateMatrix(DefaultPhaseMatrix())` retorna `nil`
- [ ] `ValidateMatrix` detecta fase faltante → error
- [ ] `ValidateMatrix` detecta fase vacía → error
- [ ] `ValidateMatrix` detecta F4 sin Save → error
- [ ] `ValidateMatrix` detecta F3 sin Quality → error
- [ ] `ValidateMatrix` detecta F3 sin Git → error
- [ ] `ValidateMatrix` detecta steps duplicados → error
- [ ] `ValidateMatrix` detecta step con valor inválido (ej. `Step(99)`) → error
- [ ] `ErrInvalidMatrix.Error()` formatea correctamente: `"skip matrix: phase F3: F3 must include Quality step"`
- [ ] `go build ./internal/boomerang/...` compila sin errores

### Estimación

- **LOC:** ~30
- **Tiempo:** ~20 minutos

### Subtareas

3.1. Implementar `AllSteps()`  
3.2. Implementar `ErrInvalidMatrix` struct + `Error()` method  
3.3. Implementar `ValidateMatrix()` con todas las reglas  

---

## T4: Modificar RunPhase() para usar skip matrix

### Descripción

Modificar `BoomerangOrchestrator.RunPhase()` en `orchestrator.go` para que use `DefaultPhaseMatrix()` y consulte `ShouldRun()` antes de ejecutar cada paso.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/orchestrator.go` | **Modificar** `RunPhase()` | Líneas 110-194 (original) → reemplazar ~130 LOC |

### Cambios específicos

1. **Agregar** `matrix := DefaultPhaseMatrix()` al inicio de `RunPhase()`
2. **Mover declaraciones** de variables compartidas fuera de los bloques:
   ```go
   var memoryCtx string
   var dag *TaskDAG
   var delegateResult *DelegateResult
   var gitStatus string
   var qualityOK bool
   var saveResult *SaveResult
   var qualityRan bool
   ```
3. **Envolver cada paso** con `if matrix.ShouldRun(config.Phase, StepXXX) { ... }`
4. **Proteger** `StepDelegate` con `dag != nil` (si Think se salta, no hay DAG)
5. **Agregar flag** `qualityRan` para determinar `result.Success`
6. **Cambiar** `result.Success = result.QualityOK` por:
   ```go
   if qualityRan {
       result.Success = result.QualityOK
   } else {
       result.Success = true
   }
   ```

### Estado de variables por fase (post-ejecución)

| Variable | F0 | F1 | F2 | F3 | F4 |
|----------|----|----|----|----|----|
| `memoryCtx` | string | string | string | string | string |
| `dag` | *TaskDAG | *TaskDAG | *TaskDAG | *TaskDAG | nil (Think skip) |
| `delegateResult` | *DelegateResult | *DelegateResult | *DelegateResult | *DelegateResult | *DelegateResult |
| `gitStatus` | "" (skip) | "" (skip) | "" (skip) | string | string |
| `qualityOK` | false | false | false | bool | false |
| `saveResult` | *SaveResult | *SaveResult | *SaveResult | *SaveResult | *SaveResult |
| `qualityRan` | false | false | false | true | false |
| `result.Success` | true | true | true | result.QualityOK | true |

### Dependencias

- **T2** (necesita `DefaultPhaseMatrix()`, `ShouldRun()`)

### Criterios de aceptación

- [ ] Firma de `RunPhase()` se mantiene: `func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error)`
- [ ] `result.Success = true` cuando Quality se salta (F0, F1, F2, F4)
- [ ] `result.Success = result.QualityOK` cuando Quality se ejecuta (F3)
- [ ] `result.GitStatus = ""` cuando Git se salta (F0, F1, F2)
- [ ] `result.QualityOK = false` cuando Quality se salta
- [ ] `result.TasksPlanned = 0` cuando Think se salta (F4)
- [ ] `go build ./...` compila sin errores
- [ ] `go vet ./internal/boomerang/...` sin warnings

### Estimación

- **LOC:** ~35 modificados (~130 LOC total de la función)
- **Tiempo:** ~30 minutos

### Subtareas

4.1. Agregar `matrix := DefaultPhaseMatrix()` y mover variables al inicio  
4.2. Envolver MemoryStep en `if ShouldRun`  
4.3. Envolver ThinkStep en `if ShouldRun`  
4.4. Envolver DelegateStep en `if ShouldRun && dag != nil`  
4.5. Envolver GitStep en `if ShouldRun`  
4.6. Envolver QualityStep en `if ShouldRun` con flag `qualityRan`  
4.7. Envolver SaveStep en `if ShouldRun`  
4.8. Cambiar lógica de `result.Success`  

---

## T5: Tests unitarios de matriz (ShouldRun, ActiveSteps, ValidateMatrix)

### Descripción

Agregar tests unitarios en `boomerang_test.go` para verificar la matriz canónica, `ShouldRun()` en todas las combinaciones F0-F4 × 6 steps, `ActiveSteps()` con inmutabilidad, fases desconocidas, y `ValidateMatrix()` con casos válidos e inválidos.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/boomerang_test.go` | **Agregar** tests nuevos | Al final del archivo |

### Funciones de test a implementar

| Test | Propósito | Asserts |
|------|-----------|---------|
| `TestStepString` | Verificar Step.String() | 8 asserts |
| `TestStepStatusString` | Verificar StepStatus.String() | 7 asserts |
| `TestDefaultPhaseMatrix` | Verificar matriz canónica completa | ~25 asserts |
| `TestDefaultPhaseMatrixValidates` | Matriz default pasa ValidateMatrix | 1 assert |
| `TestShouldRunUnknownPhase` | Fase "F99" ejecuta todos los steps | 6 asserts |
| `TestActiveSteps` | ActiveSteps counts correctos | 4 asserts |
| `TestActiveStepsImmutable` | ActiveSteps retorna copia | 2 asserts |
| `TestValidateMatrix` (7 subtests) | Todos los casos de validación | ~15 asserts |

### Dependencias

- **T3** (necesita `ValidateMatrix` implementado)

### Criterios de aceptación

- [ ] Todos los tests unitarios pasan
- [ ] `go test -run 'TestStepString|TestStepStatusString|TestDefaultPhaseMatrix|TestDefaultPhaseMatrixValidates|TestShouldRunUnknownPhase|TestActiveSteps|TestActiveStepsImmutable|TestValidateMatrix' ./internal/boomerang/` pasa
- [ ] 90%+ cobertura de `skip.go`
- [ ] `go vet ./internal/boomerang/...` sin warnings

### Estimación

- **LOC:** ~80
- **Tiempo:** ~45 minutos

### Subtareas

5.1. `TestStepString` — 8 casos (6 válidos + 2 inválidos)  
5.2. `TestStepStatusString` — 7 casos (5 válidos + 2 inválidos)  
5.3. `TestDefaultPhaseMatrix` — validar F0, F1, F2, F3, F4 completa  
5.4. `TestDefaultPhaseMatrixValidates` — la default pasa ValidateMatrix  
5.5. `TestShouldRunUnknownPhase` — F99 debe ejecutar todo  
5.6. `TestActiveSteps` — verificar counts y valores  
5.7. `TestActiveStepsImmutable` — verificar copia defensiva  
5.8. `TestValidateMatrix` — 7 subtests (válida, faltante, vacía, F4 no Save, F3 no Quality, F3 no Git, duplicado, step inválido)  

---

## T6: Tests de integración (RunPhase con skip matrix)

### Descripción

Agregar tests de integración que ejecuten `RunPhase()` completo con diferentes fases y verifiquen que los steps correctos se ejecutan/saltan según la skip matrix.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/boomerang_test.go` | **Agregar** tests nuevos | Después de tests unitarios |

### Funciones de test a implementar

| Test | Propósito | Verificaciones clave |
|------|-----------|---------------------|
| `TestRunPhaseSkipMatrixF0` | F0: Git y Quality saltados | `GitStatus=""`, `QualityOK=false`, `Success=true` |
| `TestRunPhaseSkipMatrixF3` | F3: todos los steps ejecutados | `GitStatus!=""`, `QualityOK` evaluado |
| `TestRunPhaseSkipMatrixF4` | F4: Think y Quality saltados | `TasksPlanned=0`, `QualityOK=false`, `Success=true` |

### Dependencias

- **T4** (necesita `RunPhase()` modificado con skip matrix)

### Criterios de aceptación

- [ ] `TestRunPhaseSkipMatrixF0`: GitStatus vacío, QualityOK false, Success true
- [ ] `TestRunPhaseSkipMatrixF3`: GitStatus no vacío
- [ ] `TestRunPhaseSkipMatrixF4`: TasksPlanned = 0, QualityOK false, Success true
- [ ] `go test -run 'TestRunPhaseSkipMatrix' ./internal/boomerang/` pasa

### Estimación

- **LOC:** ~80
- **Tiempo:** ~40 minutos

### Subtareas

6.1. `TestRunPhaseSkipMatrixF0` — Fase F0 con asserts de GitStatus, QualityOK, Success  
6.2. `TestRunPhaseSkipMatrixF3` — Fase F3 con assert de GitStatus no vacío  
6.3. `TestRunPhaseSkipMatrixF4` — Fase F4 con asserts de TasksPlanned=0, no Quality, Success  

---

## T7: Tests de backward compatibility

### Descripción

Verificar que todos los tests existentes en `boomerang_test.go` siguen pasando sin cambios después de la modificación de `RunPhase()`. Agregar tests específicos que verifiquen que la firma y el comportamiento observable de `RunPhase()` no han cambiado.

### Archivos afectados

| Archivo | Cambio | Líneas |
|---------|--------|--------|
| `internal/boomerang/boomerang_test.go` | **Agregar** tests nuevos | Después de tests de integración |

### Funciones de test a implementar

| Test | Propósito |
|------|-----------|
| `TestRunPhaseLegacySignature` | Verificar que `RunPhase()` acepta `PhaseConfig` y retorna `(*PhaseResult, error)` |
| `TestRunPhaseLegacyF0Result` | Verificar que `RunPhase("F0")` retorna PhaseResult con Phase="F0" |
| `TestAllLegacyTestsPass` | Verificar que `TestNewBoomerangOrchestrator`, `TestMemoryStep`, `TestThinkStep`, `TestGitStep`, `TestDelegateStep`, `TestQualityStep`, `TestSaveStep` siguen pasando |

### Dependencias

- **T4** (necesita `RunPhase()` modificado)

### Criterios de aceptación

- [ ] Los tests existentes **no se modifican** (solo se agregan nuevos)
- [ ] `go test ./internal/boomerang/...` pasa completo (tests legacy + nuevos)
- [ ] La firma de `RunPhase()` es idéntica
- [ ] `go test -count=1 ./internal/boomerang/...` (sin caché) pasa

### Estimación

- **LOC:** ~40
- **Tiempo:** ~20 minutos

### Subtareas

7.1. `TestRunPhaseLegacySignature` — compilar invocación a RunPhase con PhaseConfig  
7.2. `TestRunPhaseLegacyF0Result` — verificar resultado básico  
7.3. Verificar manualmente que todos los tests existentes pasan  

---

## DAG de dependencias entre tareas

```
T1 (tipos base)
 ├── T2 (DefaultPhaseMatrix + métodos)
 │    └── T4 (RunPhase modificado)
 │         ├── T6 (tests integración)
 │         └── T7 (tests backward compat)
 └── T3 (ValidateMatrix + AllSteps)
      └── T5 (tests unitarios)
```

**Orden de implementación recomendado:** T1 → T2 → T3 → T4 → T5 → T6 → T7

---

## Resumen de validación final

```bash
# 1. Compilar
go build ./...

# 2. Vet
go vet ./internal/boomerang/...

# 3. Tests unitarios + integración + legacy
go test -v -count=1 ./internal/boomerang/...

# 4. Cobertura de skip.go
go test -coverprofile=coverage.out ./internal/boomerang/...
go tool cover -func=coverage.out | grep skip.go
# Esperado: 90%+

# 5. Verificar que no se rompió nada fuera del paquete
go test ./...
```
