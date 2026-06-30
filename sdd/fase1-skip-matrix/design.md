# Fase 1 — Phase Skip Matrix: Diseño Técnico

> **Fecha:** 17 Junio 2026  
> **Versión:** 1.0  
> **Spec:** [spec.md](./spec.md)  
> **Tareas:** [tasks.md](./tasks.md)  
> **Plan general:** [plan-optimizacion-boomerang.md](../../docs/plan-optimizacion-boomerang.md)

---

## 1. Resumen

### 1.1 Qué se implementa

Phase Skip Matrix: una matriz declarativa `map[string][]Step` que define QUÉ pasos del ciclo Boomerang ejecutar en CADA fase (F0–F4). Reemplaza el actual "siempre ejecutar los 6 pasos" por un mecanismo configurable con defaults inteligentes.

### 1.2 Por qué

El `BoomerangOrchestrator.RunPhase()` ejecuta actualmente 6 pasos secuenciales **en todas las fases** sin distinción: Memory → Think → Delegate → Git → Quality → Save. Esto derrocha ~40% del tiempo en fases que no necesitan Git ni Quality:

| Fase | Propósito | Pasos necesarios | Pasos innecesarios |
|------|-----------|-----------------|-------------------|
| **F0** | Investigación | Memory, Think, Delegate, Save | Git, Quality |
| **F1** | Especificación | Memory, Think, Delegate, Save | Git, Quality |
| **F2** | Diseño | Memory, Think, Delegate, Save | Git, Quality |
| **F3** | Implementación | Los 6 pasos | Ninguno |
| **F4** | Cierre | Memory, Delegate, Git, Save | Think, Quality |

Peor aún: `QualityStep` ejecuta `go build` y `go test` incluso en F0, generando falsos negativos (no hay código que compilar) y añadiendo latencia sin valor.

### 1.3 Impacto esperado

- **Reducción de tiempo en F0, F1, F2, F4:** ~40% (al saltar 2 de 6 pasos)
- **Sin falsos negativos:** Quality solo se ejecuta en F3, donde tiene sentido
- **Backward compatible:** la firma de `RunPhase()` no cambia
- **0 nuevas dependencias:** solo stdlib (`fmt`)
- **Puro código declarativo:** tipos, matriz por defecto, métodos de consulta, validación

---

## 2. Arquitectura

### 2.1 Diagrama de relación: `skip.go` ↔ `orchestrator.go`

```
┌─────────────────────────────────────────────────────────────────────┐
│                    skip.go (NUEVO)                                   │
│                                                                     │
│  Step (iota enum)         StepStatus (iota enum)   StepOutput       │
│  ┌──────────────┐         ┌───────────────┐       ┌──────────────┐ │
│  │ StepMemory   │         │ StepPending   │       │ Step    Step │ │
│  │ StepThink    │         │ StepRunning   │       │ TaskName     │ │
│  │ StepDelegate │         │ StepDone      │       │ Output       │ │
│  │ StepGit      │         │ StepSkipped   │       │ Duration     │ │
│  │ StepQuality  │         │ StepFailed    │       └──────────────┘ │
│  │ StepSave     │         └───────────────┘                        │
│  └──────┬───────┘                                                  │
│         │                                                          │
│         ▼                                                          │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  PhaseStepMatrix          map[string][]Step               │      │
│  │  ┌──────────────────────────────────────────────────────┐│      │
│  │  │ DefaultPhaseMatrix() → F0..F4 default matrix         ││      │
│  │  │ ShouldRun(phase, step) → bool                        ││      │
│  │  │ ActiveSteps(phase) → []Step                          ││      │
│  │  │ ValidateMatrix(matrix) → error                       ││      │
│  │  │ AllSteps() → []Step                                  ││      │
│  │  └──────────────────────────────────────────────────────┘│      │
│  └──────────────────────────────────────────────────────────┘      │
│                              │                                      │
│                              │ consulta                             │
│                              ▼                                      │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  ErrInvalidMatrix        error personalizado              │      │
│  │  ┌──────────────────────────────────────────────────────┐│      │
│  │  │ Phase:  string   ("F3")                              ││      │
│  │  │ Reason: string   ("F3 must include Quality step")    ││      │
│  │  └──────────────────────────────────────────────────────┘│      │
│  └──────────────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────┘
         │
         │ ShouldRun(phase, step)
         ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    orchestrator.go (MODIFICADO)                      │
│                                                                     │
│  RunPhase(ctx, config PhaseConfig) (*PhaseResult, error) {          │
│    matrix := DefaultPhaseMatrix()                                   │
│                                                                     │
│    if matrix.ShouldRun(config.Phase, StepMemory)   { ... }          │
│    if matrix.ShouldRun(config.Phase, StepThink)    { ... }          │
│    if matrix.ShouldRun(config.Phase, StepDelegate) { ... }          │
│    if matrix.ShouldRun(config.Phase, StepGit)      { ... }          │
│    if matrix.ShouldRun(config.Phase, StepQuality)  { ... }          │
│    if matrix.ShouldRun(config.Phase, StepSave)     { ... }          │
│                                                                     │
│    result.Success = qualityRan ? result.QualityOK : true            │
│  }                                                                  │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Flujo de `RunPhase()` con skip checks

```
RunPhase(ctx, config)
  │
  ├─ matrix := DefaultPhaseMatrix()
  │
  ├─ ¿ShouldRun(phase, StepMemory)?
  │    ├─ Sí → memoryCtx, err := o.MemoryStep(...)
  │    └─ No → memoryCtx = ""
  │
  ├─ ¿ShouldRun(phase, StepThink)?
  │    ├─ Sí → dag, err := o.ThinkStep(...)
  │    └─ No → dag = nil
  │
  ├─ ¿ShouldRun(phase, StepDelegate) AND dag != nil?
  │    ├─ Sí → delegateResult, err := o.DelegateStep(...)
  │    └─ No → delegateResult = nil
  │
  ├─ ¿ShouldRun(phase, StepGit)?
  │    ├─ Sí → gitStatus, err := o.GitStep(...)
  │    └─ No → gitStatus = ""
  │
  ├─ ¿ShouldRun(phase, StepQuality)?
  │    ├─ Sí → qualityOK = retry loop { o.QualityStep(...) }
  │    │       qualityRan = true
  │    └─ No → qualityOK = false, qualityRan = false
  │
  ├─ ¿ShouldRun(phase, StepSave)?
  │    ├─ Sí → saveResult, err := o.SaveStep(...)
  │    └─ No → saveResult = nil
  │
  ├─ result.Success = qualityRan ? result.QualityOK : true
  │
  └─ return result, nil
```

### 2.3 Matriz canónica (default)

```
F0: [Memory] → [Think] → [Delegate] → [Save]          ← salta Git, Quality
F1: [Memory] → [Think] → [Delegate] → [Save]          ← salta Git, Quality
F2: [Memory] → [Think] → [Delegate] → [Save]          ← salta Git, Quality
F3: [Memory] → [Think] → [Delegate] → [Git] → [Quality] → [Save]  ← todos
F4: [Memory] → [Delegate] → [Git] → [Save]            ← salta Think, Quality
```

---

## 3. Decisiones de Diseño

### 3.1 `Step` como `int iota` (no `string`, no `const` string)

| Opción | Ventajas | Desventajas | Decisión |
|--------|----------|-------------|----------|
| `type Step int` con `iota` | Eficiente (4 bytes), switch/match rápido, `iota` garantiza unicidad, cero valor por defecto | Requiere `String()` para debugging | ✅ **Elegido** |
| `type Step string` con strings planos | Debugging directo sin `String()` | Comparación más costosa, no hay enumeración garantizada, propenso a typos | ❌ |
| `const ( StepMemory = iota )` sin tipo | Simple | Sin type safety, puede mezclarse con otros ints | ❌ |
| `const ( StepMemory = 1 << iota )` bitset | Permite combinar pasos con OR binario | Complejidad innecesaria, no necesitamos combinar pasos | ❌ |

**Razonamiento:**

- `iota` garantiza valores secuenciales desde 0. Esto permite que `Step(String())` sobre array funcione en principio, pero elegimos switch para seguridad.
- `Step(0)` = `StepPending` no existe; `StepMemory` es 0, lo cual es natural como primer paso.
- El tipo `Step` es **comparable directamente como int**: `if step == StepMemory { ... }`. Esto es más rápido que comparar strings.
- `String()` añadimos con switch (no lookup de array) para evitar panic con valores inválidos:

```go
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
```

**Nota sobre array vs switch:** El array `[...]string{"Memory","Think",...}[s]` es O(1) vs O(n) del switch, pero **paniquea si `s` está fuera de rango**. Preferimos seguridad sobre micro-optimización aquí porque `String()` se usa para debugging/logging, no en hot paths.

### 3.2 `StepStatus` como `int iota` (análogo a Step)

Misma lógica que `Step`: eficiencia, comparación rápida, `String()` con switch para seguridad.

**Orden del estado** refleja el ciclo de vida natural:
```
StepPending (0) → StepRunning (1) → StepDone (2)
                                  → StepSkipped (3)
                                  → StepFailed (4)
```

### 3.3 ¿Por qué NO modificar `PhaseConfig`?

**Decisión: No tocar `PhaseConfig`.** La matriz es un detalle de implementación interna.

```go
// PhaseConfig NO cambia — backward compatibility total
type PhaseConfig struct {
    Phase       string
    TaskDesc    string
    ProjectID   string
    MemoryLimit int
    Iterations  int
    Timeout     time.Duration
}
```

**Razones:**

1. **API pública:** `PhaseConfig` es parte de la interfaz pública. `Scheduler` lo usa directamente. Modificarlo forzaría cambios en `cmd/`, `internal/scheduler/`, y tests.

2. **Separación de concerns:** La matriz de skip es una decisión interna del orquestador. El llamante no necesita saber qué pasos se ejecutan — solo quiere ejecutar una fase.

3. **Default correcto por defecto:** La matriz canónica es la correcta para el 99% de los casos. No necesitamos que cada llamante configure los pasos.

4. **Fase 2 del plan introduce `PhaseConfigV2`:** El plan de optimización (sección 8.3) ya contempla `PhaseConfigV2` como un tipo nuevo para quienes necesiten override explícito. Eso será en una fase posterior.

### 3.4 ¿Por qué `DefaultPhaseMatrix()` es pública?

```go
func DefaultPhaseMatrix() PhaseStepMatrix { ... }
```

Pública por dos razones:

1. **Tests:** Los tests unitarios necesitan verificar la matriz canónica:
   ```go
   func TestDefaultPhaseMatrix(t *testing.T) {
       matrix := DefaultPhaseMatrix()
       if matrix.ShouldRun("F0", StepGit) {
           t.Error("F0 should skip Git")
       }
   }
   ```

2. **Override por configuración (futuro):** Cuando `PhaseConfigV2` permita inyectar una matriz personalizada, el llamante puede partir de `DefaultPhaseMatrix()` y modificarla:
   ```go
   m := DefaultPhaseMatrix()
   m["F0"] = append(m["F0"], StepQuality) // override
   config := PhaseConfigV2{SkipMatrix: m}
   ```

**No hay riesgo de seguridad:** `DefaultPhaseMatrix()` retorna un `map[string][]Step` nuevo cada vez que se invoca. No hay estado compartido ni mutabilidad entre llamadas.

### 3.5 Manejo de fases desconocidas (default seguro)

Para fases que no están en la matriz (ej. "F99", "F5", "testing"), adoptamos el principio de **default seguro**:

| Método | Fase conocida | Fase desconocida |
|--------|--------------|------------------|
| `ShouldRun("F0", StepMemory)` | `true` | — |
| `ShouldRun("F99", StepMemory)` | — | `true` ✅ |
| `ShouldRun("F99", StepGit)` | — | `true` ✅ |
| `ActiveSteps("F0")` | `[Memory, Think, Delegate, Save]` | — |
| `ActiveSteps("F99")` | — | Todos los 6 steps ✅ |

**Razonamiento:** Es mejor ejecutar un paso innecesario (falso positivo) que saltar un paso necesario (falso negativo). Una fase desconocida probablemente es una fase futura o experimental; ejecutar todos los pasos es la opción más segura.

**Implementación:**
```go
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
```

### 3.6 `ValidateMatrix()` — garantía de integridad

La validación asegura que la matriz nunca tenga configuraciones inconsistentes:

| Regla | Ejemplo de violación | Por qué |
|-------|---------------------|---------|
| F0-F4 definidas | Falta "F1" | Fase sin definición tomaría default seguro, ocultando error |
| Al menos 1 step por fase | F0: `[]` | Matriz vacía no tiene sentido |
| F4 debe incluir Save | F4: `[Memory, Delegate, Git]` | Cierre debe guardar |
| F3 debe incluir Quality | F3: `[Memory, Think, Delegate, Git, Save]` | Implementación sin validación |
| F3 debe incluir Git | F3: `[Memory, Think, Delegate, Quality, Save]` | Implementación sin control de versión |
| Sin duplicados | F4: `[Memory, Save, Save]` | Paso duplicado es ruido |
| Steps en rango (0-5) | F4: `[Memory, Step(99)]` | Step inválido rompería lógica |

### 3.7 `AllSteps()` — helper para completitud

```go
func AllSteps() []Step {
    return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
}
```

Función auxiliar que retorna los 6 pasos en orden. Se usa internamente en:
- `ActiveSteps()` para fases desconocidas
- Tests que necesitan iterar sobre todos los pasos

### 3.8 `result.Success` con Quality saltado

**Problema:** Actualmente `result.Success = result.QualityOK`. Si Quality se salta, `QualityOK` es `false`, entonces `Success = false` aunque la fase se haya completado sin errores.

**Solución:** Flag `qualityRan` que indica si Quality se ejecutó realmente:
```go
qualityRan := false

// ... dentro del bloque if de Quality:
if matrix.ShouldRun(config.Phase, StepQuality) {
    qualityRan = true
    // ... ejecutar QualityStep ...
}

// Al final:
if qualityRan {
    result.Success = result.QualityOK
} else {
    result.Success = true // sin Quality, asumimos éxito
}
```

**Excepción:** Si un step retorna error fatal (Memory, Think, Git), la función retorna con `return nil, err` antes de llegar a `result.Success`. Solo errores no-fatales (Delegate, que retorna `result, nil` con `result.Error`) pueden dejar `Success` en un estado mixto.

### 3.9 Protección de `delegateResult == nil` en QualityStep

Si Delegate se salta (fases sin delegación), `delegateResult` es `nil`. `QualityStep` itera sobre `delegateResult.TaskResults`:

```go
// quality.go actual:
for _, tr := range delegateResult.TaskResults { // panic si delegateResult es nil
```

**Solución:** No cambiamos `quality.go` en esta fase. Si Delegate se salta, Quality también se salta (excepto en matrices inválidas). La matriz canónica garantiza que Delegate y Quality siempre coexistan o se salten juntos (ver tabla 2.3).

**Protección adicional en el retry loop de Quality:**
```go
if matrix.ShouldRun(config.Phase, StepQuality) {
    qualityRan = true
    for i := 0; i < o.maxIterations; i++ {
        qualityOK, err = o.QualityStep(ctx, config.Phase, dag, delegateResult)
        // delegateResult puede ser nil si Delegate se saltó
        // QualityStep itera sobre delegateResult.TaskResults.
        // Si delegateResult es nil, el range sobre un nil map es seguro en Go,
        // pero delegateResult.TaskResults itera sobre un map vacío.
    }
}
```

**Nota:** `range` sobre un `nil map` en Go itera cero veces (no panic). `range` sobre un `nil slice` también itera cero veces. `DelegateResult.TaskResults` es un `map[string]TaskResult` — si `delegateResult` es `nil`, acceder a `delegateResult.TaskResults` sí paniquea. Por eso, en la práctica, Quality y Delegate deben coexistir en la matriz.

---

## 4. Código Detallado

### 4.1 `skip.go` — código completo (112 LOC)

```go
// Package boomerang implementa el orquestador de 6 pasos del ciclo Boomerang.
package boomerang

import "fmt"

// ──────────────────────────────────────────────────────────────
// Step: enumeración de pasos del ciclo Boomerang
// ──────────────────────────────────────────────────────────────

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

// String retorna el nombre legible del step para debugging/logging.
func (s Step) String() string {
	switch s {
	case StepMemory:
		return "Memory"
	case StepThink:
		return "Think"
	case StepDelegate:
		return "Delegate"
	case StepGit:
		return "Git"
	case StepQuality:
		return "Quality"
	case StepSave:
		return "Save"
	default:
		return "Unknown"
	}
}

// ──────────────────────────────────────────────────────────────
// StepStatus: estado de cada step durante la ejecución
// ──────────────────────────────────────────────────────────────

// StepStatus representa el estado de un step durante la ejecución.
type StepStatus int

const (
	StepPending StepStatus = iota // 0 — no ha comenzado
	StepRunning                   // 1 — en ejecución
	StepDone                      // 2 — completado exitosamente
	StepSkipped                   // 3 — saltado por la skip matrix
	StepFailed                    // 4 — falló
)

// String retorna la representación legible del estado.
func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "pending"
	case StepRunning:
		return "running"
	case StepDone:
		return "done"
	case StepSkipped:
		return "skipped"
	case StepFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ──────────────────────────────────────────────────────────────
// StepOutput: resultado individual de un step (para event bus futuro)
// ──────────────────────────────────────────────────────────────

// StepOutput encapsula el resultado individual de un step.
// Se usará para streaming de progreso en Fase 4+ (Event Bus).
type StepOutput struct {
	Step     Step   `json:"step"`
	TaskName string `json:"task_name,omitempty"`
	Output   string `json:"output,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// ──────────────────────────────────────────────────────────────
// PhaseStepMatrix: matriz declarativa de pasos por fase
// ──────────────────────────────────────────────────────────────

// PhaseStepMatrix define qué pasos ejecutar en cada fase del ciclo Boomerang.
// key = nombre de fase ("F0".."F4"), value = slice ordenado de steps.
type PhaseStepMatrix map[string][]Step

// AllSteps retorna todos los steps en orden de ejecución.
// Útil para fases desconocidas (default seguro) y para tests.
func AllSteps() []Step {
	return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
}

// DefaultPhaseMatrix retorna la matriz de pasos por defecto para F0-F4.
//
// Matriz canónica:
//
//	F0: Memory → Think → Delegate → Save        (salta Git, Quality)
//	F1: Memory → Think → Delegate → Save        (salta Git, Quality)
//	F2: Memory → Think → Delegate → Save        (salta Git, Quality)
//	F3: Memory → Think → Delegate → Git → Quality → Save  (todos)
//	F4: Memory → Delegate → Git → Save          (salta Think, Quality)
func DefaultPhaseMatrix() PhaseStepMatrix {
	return PhaseStepMatrix{
		"F0": {StepMemory, StepThink, StepDelegate, StepSave},
		"F1": {StepMemory, StepThink, StepDelegate, StepSave},
		"F2": {StepMemory, StepThink, StepDelegate, StepSave},
		"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
		"F4": {StepMemory, StepDelegate, StepGit, StepSave},
	}
}

// ShouldRun verifica si un step debe ejecutarse en la fase dada.
// Si la fase no está definida en la matriz, retorna true (default seguro:
// ejecutar todos los pasos para fases desconocidas).
func (m PhaseStepMatrix) ShouldRun(phase string, step Step) bool {
	steps, ok := m[phase]
	if !ok {
		return true // default seguro: ejecutar todo
	}
	for _, s := range steps {
		if s == step {
			return true
		}
	}
	return false
}

// ActiveSteps retorna los steps activos para una fase, en orden de ejecución.
// Si la fase no está definida, retorna los 6 steps (default seguro).
// Retorna una copia del slice para evitar mutación externa de la matriz.
func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
	steps, ok := m[phase]
	if !ok {
		return AllSteps()
	}
	// Retornar copia para inmutabilidad
	result := make([]Step, len(steps))
	copy(result, steps)
	return result
}

// ──────────────────────────────────────────────────────────────
// ErrInvalidMatrix: error personalizado para validación
// ──────────────────────────────────────────────────────────────

// ErrInvalidMatrix se retorna cuando una matriz no pasa validación.
type ErrInvalidMatrix struct {
	Phase  string
	Reason string
}

func (e *ErrInvalidMatrix) Error() string {
	return fmt.Sprintf("skip matrix: phase %s: %s", e.Phase, e.Reason)
}

// ValidateMatrix valida que una PhaseStepMatrix cumpla con las reglas:
//   - Todas las fases F0-F4 tienen al menos un step
//   - F4 debe incluir Save
//   - F3 debe incluir Quality + Git
//   - No hay steps duplicados dentro de una fase
//   - No hay steps con valores fuera de rango (0-5)
func ValidateMatrix(matrix PhaseStepMatrix) error {
	requiredPhases := []string{"F0", "F1", "F2", "F3", "F4"}
	maxStep := int(StepSave) // 5

	for _, phase := range requiredPhases {
		steps, ok := matrix[phase]
		if !ok {
			return &ErrInvalidMatrix{
				Phase:  phase,
				Reason: "phase not defined in matrix",
			}
		}
		if len(steps) == 0 {
			return &ErrInvalidMatrix{
				Phase:  phase,
				Reason: "must have at least one step",
			}
		}

		seen := make(map[Step]bool)
		for _, s := range steps {
			if int(s) < 0 || int(s) > maxStep {
				return &ErrInvalidMatrix{
					Phase:  phase,
					Reason: fmt.Sprintf("invalid step value %d", s),
				}
			}
			if seen[s] {
				return &ErrInvalidMatrix{
					Phase:  phase,
					Reason: fmt.Sprintf("duplicate step %s", s),
				}
			}
			seen[s] = true
		}

		// Validaciones específicas por fase
		hasStep := func(s Step) bool { return seen[s] }

		if phase == "F4" && !hasStep(StepSave) {
			return &ErrInvalidMatrix{
				Phase:  phase,
				Reason: "F4 must include Save step",
			}
		}

		if phase == "F3" {
			if !hasStep(StepQuality) {
				return &ErrInvalidMatrix{
					Phase:  phase,
					Reason: "F3 must include Quality step",
				}
			}
			if !hasStep(StepGit) {
				return &ErrInvalidMatrix{
					Phase:  phase,
					Reason: "F3 must include Git step",
				}
			}
		}
	}

	return nil
}
```

### 4.2 `orchestrator.go` — `RunPhase()` modificado

**Archivo:** `internal/boomerang/orchestrator.go`  
**Función:** `RunPhase()` (líneas 110-194 original → reemplazar)

El cambio consiste en:
1. Agregar `matrix := DefaultPhaseMatrix()` al inicio
2. Envolver cada paso en `if matrix.ShouldRun(config.Phase, StepXXX)`
3. Mover variables compartidas (`dag`, `delegateResult`, `gitStatus`, `qualityOK`, `saveResult`) fuera de los bloques if
4. Agregar flag `qualityRan` para determinar `result.Success`
5. Cambiar `result.Success = result.QualityOK` → lógica condicional

```go
// RunPhase ejecuta el ciclo Boomerang usando la Phase Skip Matrix.
// Firma idéntica — backward compatibility total.
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
	start := time.Now()
	result := &PhaseResult{Phase: config.Phase}
	matrix := DefaultPhaseMatrix()

	// Variables compartidas entre pasos
	var memoryCtx string
	var dag *TaskDAG
	var delegateResult *DelegateResult
	var gitStatus string
	var qualityOK bool
	var saveResult *SaveResult

	// Indicador: ¿Quality se ejecutó realmente?
	qualityRan := false

	// ──────────────────────────────────────────────
	// Paso 1: MEMORY
	// ──────────────────────────────────────────────
	if matrix.ShouldRun(config.Phase, StepMemory) {
		var err error
		memoryCtx, err = o.MemoryStep(ctx, config.Phase, config.TaskDesc)
		if err != nil {
			return nil, err
		}
		result.MemoryUsed = len(memoryCtx)
	}

	// Estimar tokens (solo si Memory corrió)
	var withoutTokens, withTokens int64
	if memoryCtx != "" {
		withoutTokens = tokens.Count("Execute phase " + config.Phase + ": " + config.TaskDesc + ". Codebase context: ~3000 chars baseline.")
		withTokens = tokens.Count(memoryCtx)
	}

	// ──────────────────────────────────────────────
	// Paso 2: THINK
	// ──────────────────────────────────────────────
	if matrix.ShouldRun(config.Phase, StepThink) {
		var err error
		dag, err = o.ThinkStep(ctx, config.Phase, memoryCtx)
		if err != nil {
			return nil, err
		}
		result.TasksPlanned = len(dag.Tasks)
	}

	// ──────────────────────────────────────────────
	// Paso 3: DELEGATE
	// ──────────────────────────────────────────────
	// Solo delegar si hay DAG (Think corrió exitosamente)
	if matrix.ShouldRun(config.Phase, StepDelegate) && dag != nil {
		var err error
		delegateResult, err = o.DelegateStep(ctx, dag, config.Phase)
		if err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result, nil
		}
		if delegateResult != nil {
			result.NodesCreated = delegateResult.NodesCreated
		}
	}

	// ──────────────────────────────────────────────
	// Paso 4: GIT
	// ──────────────────────────────────────────────
	if matrix.ShouldRun(config.Phase, StepGit) {
		var err error
		gitStatus, err = o.GitStep(ctx)
		if err != nil {
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result, nil
		}
		result.GitStatus = gitStatus
	}

	// ──────────────────────────────────────────────
	// Paso 5: QUALITY (con retry loop)
	// ──────────────────────────────────────────────
	if matrix.ShouldRun(config.Phase, StepQuality) {
		qualityRan = true
		for i := 0; i < o.maxIterations; i++ {
			var err error
			qualityOK, err = o.QualityStep(ctx, config.Phase, dag, delegateResult)
			if err == nil && qualityOK {
				result.QualityOK = true
				result.Iterations = i + 1
				break
			}
			if i < o.maxIterations-1 && dag != nil {
				// Redelegar tareas fallidas
				delegateResult, _ = o.DelegateStep(ctx, dag, config.Phase)
			}
		}
	}

	// ──────────────────────────────────────────────
	// Paso 6: SAVE
	// ──────────────────────────────────────────────
	if matrix.ShouldRun(config.Phase, StepSave) {
		var err error
		saveResult, err = o.SaveStep(ctx, config.Phase, delegateResult, nil)
		if err == nil && saveResult != nil {
			result.FactsSaved = saveResult.FactsSaved
		}
	}

	// ──────────────────────────────────────────────
	// Mediciones y resultado final
	// ──────────────────────────────────────────────
	if o.measurementCallback != nil {
		o.measurementCallback(Measurement{
			Phase:            config.Phase,
			TaskDescription:  config.TaskDesc,
			WithoutBoomerang: withoutTokens,
			WithBoomerang:    withTokens,
			OutputTokens:     0,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		})
	}

	// Éxito: si Quality corrió, usamos su resultado; si no, asumimos éxito
	if qualityRan {
		result.Success = result.QualityOK
	} else {
		result.Success = true
	}

	result.Duration = time.Since(start)

	return result, nil
}
```

### 4.3 Tests — código completo para `boomerang_test.go`

Los siguientes tests se **agregan** al archivo `internal/boomerang/boomerang_test.go`. Los tests existentes (`TestNewBoomerangOrchestrator`, `TestMemoryStep`, `TestThinkStep`, etc.) no se modifican.

```go
// ══════════════════════════════════════════════════════════════
// Tests para Fase 1: Phase Skip Matrix
// ══════════════════════════════════════════════════════════════

// --- Tests unitarios de tipos base ---------------------------------

func TestStepString(t *testing.T) {
	tests := []struct {
		step Step
		want string
	}{
		{StepMemory, "Memory"},
		{StepThink, "Think"},
		{StepDelegate, "Delegate"},
		{StepGit, "Git"},
		{StepQuality, "Quality"},
		{StepSave, "Save"},
		{Step(-1), "Unknown"},
		{Step(99), "Unknown"},
	}
	for _, tc := range tests {
		if got := tc.step.String(); got != tc.want {
			t.Errorf("Step(%d).String() = %q, want %q", tc.step, got, tc.want)
		}
	}
}

func TestStepStatusString(t *testing.T) {
	tests := []struct {
		status StepStatus
		want   string
	}{
		{StepPending, "pending"},
		{StepRunning, "running"},
		{StepDone, "done"},
		{StepSkipped, "skipped"},
		{StepFailed, "failed"},
		{StepStatus(99), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.status.String(); got != tc.want {
			t.Errorf("StepStatus(%d).String() = %q, want %q", tc.status, got, tc.want)
		}
	}
}

// --- Tests de DefaultPhaseMatrix -----------------------------------

func TestDefaultPhaseMatrix(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// Verificar que todas las fases existen
	for _, phase := range []string{"F0", "F1", "F2", "F3", "F4"} {
		if _, ok := matrix[phase]; !ok {
			t.Errorf("DefaultPhaseMatrix missing phase %s", phase)
		}
	}

	// F0: NO Git, NO Quality
	if matrix.ShouldRun("F0", StepGit) {
		t.Error("F0 should skip Git")
	}
	if matrix.ShouldRun("F0", StepQuality) {
		t.Error("F0 should skip Quality")
	}
	if !matrix.ShouldRun("F0", StepMemory) {
		t.Error("F0 should run Memory")
	}
	if !matrix.ShouldRun("F0", StepThink) {
		t.Error("F0 should run Think")
	}
	if !matrix.ShouldRun("F0", StepDelegate) {
		t.Error("F0 should run Delegate")
	}
	if !matrix.ShouldRun("F0", StepSave) {
		t.Error("F0 should run Save")
	}

	// F1: igual que F0 — NO Git, NO Quality
	if matrix.ShouldRun("F1", StepGit) {
		t.Error("F1 should skip Git")
	}
	if matrix.ShouldRun("F1", StepQuality) {
		t.Error("F1 should skip Quality")
	}

	// F2: igual que F0 — NO Git, NO Quality
	if matrix.ShouldRun("F2", StepGit) {
		t.Error("F2 should skip Git")
	}
	if matrix.ShouldRun("F2", StepQuality) {
		t.Error("F2 should skip Quality")
	}

	// F3: debe ejecutar los 6 pasos
	for _, s := range []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave} {
		if !matrix.ShouldRun("F3", s) {
			t.Errorf("F3 should run step %s", s)
		}
	}

	// F4: NO Think, NO Quality
	if matrix.ShouldRun("F4", StepThink) {
		t.Error("F4 should skip Think")
	}
	if matrix.ShouldRun("F4", StepQuality) {
		t.Error("F4 should skip Quality")
	}
	if !matrix.ShouldRun("F4", StepMemory) {
		t.Error("F4 should run Memory")
	}
	if !matrix.ShouldRun("F4", StepDelegate) {
		t.Error("F4 should run Delegate")
	}
	if !matrix.ShouldRun("F4", StepGit) {
		t.Error("F4 should run Git")
	}
	if !matrix.ShouldRun("F4", StepSave) {
		t.Error("F4 should run Save")
	}
}

func TestDefaultPhaseMatrixValidates(t *testing.T) {
	matrix := DefaultPhaseMatrix()
	if err := ValidateMatrix(matrix); err != nil {
		t.Fatalf("DefaultPhaseMatrix should be valid: %v", err)
	}
}

// --- Tests de ShouldRun con fases desconocidas -----------------------

func TestShouldRunUnknownPhase(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// Fase desconocida: debe ejecutar todos los pasos (default seguro)
	for _, s := range []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave} {
		if !matrix.ShouldRun("F99", s) {
			t.Errorf("unknown phase should run step %s", s)
		}
	}
}

// --- Tests de ActiveSteps -------------------------------------------

func TestActiveSteps(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// F0: 4 steps
	steps := matrix.ActiveSteps("F0")
	if len(steps) != 4 {
		t.Errorf("F0 expected 4 active steps, got %d: %v", len(steps), steps)
	}

	// F3: 6 steps
	steps = matrix.ActiveSteps("F3")
	if len(steps) != 6 {
		t.Errorf("F3 expected 6 active steps, got %d: %v", len(steps), steps)
	}

	// F4: 4 steps (Memory, Delegate, Git, Save)
	steps = matrix.ActiveSteps("F4")
	if len(steps) != 4 {
		t.Errorf("F4 expected 4 active steps, got %d: %v", len(steps), steps)
	}

	// Fase desconocida: 6 steps
	steps = matrix.ActiveSteps("F99")
	if len(steps) != 6 {
		t.Errorf("F99 expected 6 active steps, got %d: %v", len(steps), steps)
	}
}

func TestActiveStepsImmutable(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	original := matrix.ActiveSteps("F0")
	modified := matrix.ActiveSteps("F0")

	// Modificar la copia
	modified[0] = StepGit

	// El original no debe haber cambiado
	if original[0] != StepMemory {
		t.Error("ActiveSteps should return a copy, original was mutated")
	}
}

// --- Tests de ValidateMatrix ----------------------------------------

func TestValidateMatrix(t *testing.T) {
	t.Run("valid default matrix", func(t *testing.T) {
		matrix := DefaultPhaseMatrix()
		if err := ValidateMatrix(matrix); err != nil {
			t.Errorf("default matrix should be valid: %v", err)
		}
	})

	t.Run("missing phase", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for missing F1, F2")
		} else {
			t.Logf("got expected error: %v", err)
		}
	})

	t.Run("empty steps", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for empty F0 steps")
		}
	})

	t.Run("F4 missing Save", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory, StepThink, StepDelegate, StepSave},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for F4 missing Save")
		}
	})

	t.Run("F3 missing Quality", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory, StepThink, StepDelegate, StepSave},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for F3 missing Quality")
		}
	})

	t.Run("F3 missing Git", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory, StepThink, StepDelegate, StepSave},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepQuality, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for F3 missing Git")
		}
	})

	t.Run("duplicate step in phase", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory, StepThink, StepDelegate, StepSave},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
			"F4": {StepMemory, StepDelegate, StepGit, StepSave, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for duplicate step in F4")
		}
	})

	t.Run("invalid step value", func(t *testing.T) {
		matrix := PhaseStepMatrix{
			"F0": {StepMemory, StepThink, StepDelegate, StepSave},
			"F1": {StepMemory, StepThink, StepDelegate, StepSave},
			"F2": {StepMemory, StepThink, StepDelegate, StepSave},
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
			"F4": {Step(99), StepDelegate, StepGit, StepSave},
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for invalid step value")
		}
	})
}

// --- Tests de integración: RunPhase con skip matrix ------------------

func TestRunPhaseSkipMatrixF0(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F0",
		TaskDesc: "test skip matrix F0",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F0" {
		t.Errorf("expected F0, got %s", result.Phase)
	}

	// GitStatus debe estar vacío (no se ejecutó Git)
	if result.GitStatus != "" {
		t.Errorf("F0 should not run Git, got status %q", result.GitStatus)
	}
	// QualityOK debe ser false (no se ejecutó Quality)
	if result.QualityOK {
		t.Error("F0 should not run Quality")
	}
	// Success debe ser true aunque Quality no corrió
	if !result.Success {
		t.Error("F0 should be successful even without Quality")
	}
}

func TestRunPhaseSkipMatrixF3(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F3",
		TaskDesc: "test skip matrix F3",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F3" {
		t.Errorf("expected F3, got %s", result.Phase)
	}
	// GitStatus no está vacío (se ejecutó Git en F3)
	if result.GitStatus == "" {
		t.Error("F3 should run Git, expected non-empty status")
	}
}

func TestRunPhaseSkipMatrixF4(t *testing.T) {
	o := NewBoomerangOrchestrator(&mockStore{}, mockBoundariLoader, nil)
	ctx := context.Background()

	config := PhaseConfig{
		Phase:    "F4",
		TaskDesc: "test skip matrix F4",
	}

	result, err := o.RunPhase(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phase != "F4" {
		t.Errorf("expected F4, got %s", result.Phase)
	}

	// F4 no ejecuta Think → TasksPlanned debe ser 0
	if result.TasksPlanned != 0 {
		t.Errorf("F4 should not run Think, got TasksPlanned=%d", result.TasksPlanned)
	}
	// F4 no ejecuta Quality → QualityOK debe ser false
	if result.QualityOK {
		t.Error("F4 should not run Quality")
	}
	// Success debe ser true
	if !result.Success {
		t.Error("F4 should be successful even without Quality")
	}
}
```

---

## 5. Archivos Afectados

| Archivo | Tipo de cambio | LOC |
|---------|---------------|-----|
| `internal/boomerang/skip.go` | **Nuevo** | ~112 LOC |
| `internal/boomerang/orchestrator.go` | **Modificar** `RunPhase()` | ~35 LOC modificados (~130 LOC total función) |
| `internal/boomerang/boomerang_test.go` | **Agregar** tests | ~250 LOC nuevos |

**No requieren cambios:** `delegate.go`, `memory.go`, `think.go`, `git.go`, `quality.go`, `save.go` — ningún step individual se modifica. Solo su wrapper condicional en `RunPhase()`.

---

## 6. Checklist de Implementación

- [ ] Crear `internal/boomerang/skip.go` con tipos y funciones
- [ ] Modificar `RunPhase()` en `orchestrator.go`
- [ ] Agregar tests unitarios en `boomerang_test.go`
- [ ] `go build ./...` compila sin errores
- [ ] `go vet ./internal/boomerang/...` sin warnings
- [ ] `go test ./internal/boomerang/...` pasa (tests nuevos + existentes)
- [ ] Verificar `result.Success = true` cuando Quality se salta
- [ ] Verificar `result.GitStatus = ""` cuando Git se salta
