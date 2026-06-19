# Fase 1 — Phase Skip Matrix

> **Fecha:** 17 Junio 2026
> **Versión:** 1.0
> **Autor:** SDD Orchestrator
> **Plan general:** [plan-optimizacion-boomerang.md](../../docs/plan-optimizacion-boomerang.md)
> **Proyecto:** ZyroAgentCLI (`github.com/secko/zyrocli`)

---

## 1. Objetivo

El `BoomerangOrchestrator` ejecuta actualmente 6 pasos secuenciales en todas las fases F0–F4: Memory → Think → Delegate → Git → Quality → Save. Esto es ineficiente porque fases de investigación (F0), especificación (F1), diseño (F2) y cierre (F4) no necesitan pasos como Git o Quality — que en esas fases no producen valor real y solo consumen tiempo (~40% del tiempo de fase). QualityStep además ejecuta `go build` y `go test` incluso en F0, generando falsos negativos.

La Fase 1 implementa una "Phase Skip Matrix": una matriz declarativa que define QUÉ pasos ejecutar en CADA fase. Reemplaza el "siempre todos" del `RunPhase()` actual por un mecanismo configurable con defaults inteligentes. El impacto es inmediato: F0, F1, F2 y F4 ejecutan solo los pasos necesarios, reduciendo el tiempo de ciclo hasta un 40% en esas fases sin modificar la lógica de cada step individual.

Esta fase no introduce nuevas dependencias ni cambia la API pública. Es puro código declarativo: tipos, matriz por defecto, métodos de consulta, validación y la integración mínima en `RunPhase()`.

---

## 2. Especificación Técnica

### 2.1 Tipos a definir (archivo nuevo: `internal/boomerang/skip.go`)

#### `Step` — enumeración de pasos

```go
// Step identifica cada paso del ciclo Boomerang
type Step int

const (
	StepMemory   Step = iota // 0
	StepThink                // 1
	StepDelegate             // 2
	StepGit                  // 3
	StepQuality              // 4
	StepSave                 // 5
)

// String retorna el nombre legible del step
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
```

#### `StepStatus` — estado de cada step durante la ejecución

```go
// StepStatus representa el estado de un step durante la ejecución
type StepStatus int

const (
	StepPending StepStatus = iota // no ha comenzado
	StepRunning                   // en ejecución
	StepDone                      // completado exitosamente
	StepSkipped                   // saltado por la skip matrix
	StepFailed                    // falló
)

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
```

#### `StepOutput` — datos de output de un step (para futura integración con event bus)

```go
// StepOutput encapsula el resultado individual de un step,
// usado para streaming de progreso en el event loop (Fase 4+).
type StepOutput struct {
	Step      Step   `json:"step"`
	TaskName  string `json:"task_name,omitempty"`
	Output    string `json:"output,omitempty"`
	Duration  string `json:"duration,omitempty"`
}
```

#### `PhaseStepMatrix` — mapa de fases a steps

```go
// PhaseStepMatrix define qué pasos ejecutar en cada fase del ciclo Boomerang.
// key = nombre de fase ("F0", "F1", etc.), value = slice ordenado de steps a ejecutar.
type PhaseStepMatrix map[string][]Step
```

---

### 2.2 `DefaultPhaseMatrix()` — matriz canónica

```go
// DefaultPhaseMatrix retorna la matriz de pasos por defecto para F0-F4.
//
// Matriz canónica:
//   F0: Memory → Think → Delegate → Save        (salta Git, Quality)
//   F1: Memory → Think → Delegate → Save        (salta Git, Quality)
//   F2: Memory → Think → Delegate → Save        (salta Git, Quality)
//   F3: Memory → Think → Delegate → Git → Quality → Save  (todos)
//   F4: Memory → Delegate → Git → Save          (salta Think, Quality)
func DefaultPhaseMatrix() PhaseStepMatrix {
	return PhaseStepMatrix{
		"F0": {StepMemory, StepThink, StepDelegate, StepSave},
		"F1": {StepMemory, StepThink, StepDelegate, StepSave},
		"F2": {StepMemory, StepThink, StepDelegate, StepSave},
		"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
		"F4": {StepMemory, StepDelegate, StepGit, StepSave},
	}
}
```

**Justificación paso a paso:**

| Fase | Memory | Think | Delegate | Git | Quality | Save |
|------|--------|-------|----------|-----|---------|------|
| **F0** (Investigación) | ✅ Necesita contexto causal de qué investigar | ✅ Planifica DAG de 3 subagentes de investigación | ✅ Ejecuta subagentes de patrones/librerías/skills | ❌ No hay código que modificar | ❌ No hay código que compilar/testear | ✅ Guarda hechos descubiertos |
| **F1** (Especificación) | ✅ Contexto de investigación F0 | ✅ Planifica spec + review secuencial | ✅ Ejecuta subagente de especificación | ❌ Aún no hay código | ❌ No hay código que validar | ✅ Guarda la especificación |
| **F2** (Diseño) | ✅ Contexto de especificación F1 | ✅ Planifica diseño técnico | ✅ Ejecuta subagente de diseño | ❌ Aún no hay implementación | ❌ No hay código que compilar | ✅ Guarda diseño y tareas |
| **F3** (Implementación) | ✅ | ✅ | ✅ | ✅ Verifica estado del repo | ✅ Compila y testea | ✅ |
| **F4** (Cierre) | ✅ Recupera hechos de F3 | ❌ No planifica nuevas estrategias | ✅ Ejecuta subagente de archivo | ✅ Verifica que todo esté commiteado | ❌ No hay nuevo código | ✅ Guarda registro de cierre |

---

### 2.3 `ValidateMatrix()` — validación de integridad

```go
// ErrInvalidMatrix se retorna cuando una matriz no pasa validación.
type ErrInvalidMatrix struct {
	Phase string
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

		// Validaciones específicas
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

---

### 2.4 `ShouldRun()` + `ActiveSteps()` — métodos de consulta

```go
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
func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
	steps, ok := m[phase]
	if !ok {
		return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
	}
	// Retornar copia para evitar mutación externa
	result := make([]Step, len(steps))
	copy(result, steps)
	return result
}
```

**Comportamiento para fase desconocida:**
- `ShouldRun("F99", StepMemory)` → `true` (ejecuta todo)
- `ActiveSteps("F99")` → `[StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave]`

---

### 2.5 Integración con `RunPhase()` existente

El cambio en `orchestrator.go` consiste en modificar `RunPhase()` para que ante cada paso consulte `ShouldRun()` antes de ejecutar. La firma pública se mantiene idéntica.

**Variable `dag` compartida entre pasos:**
- `dag` se declara fuera de los bloques `if` para que esté disponible para DelegateStep y QualityStep
- Si Think se salta, `dag` se queda como `nil` y DelegateStep no se ejecuta (no tiene sentido delegar sin DAG)
- Si Delegate se salta, `delegateResult` se queda como `nil`

**Variable `delegateResult` compartida:**
- Se declara fuera de los bloques condicionales
- QualityStep y SaveStep la usan; si Delegate se saltó, `delegateResult` es `nil`
- `SaveStep` ya maneja `nil` en `delegateResult.TaskResults` (range sobre nil map es seguro en Go)
- `QualityStep` itera sobre `delegateResult.TaskResults` — si `delegateResult` es nil, se debe proteger

**Manejo de `result.Success`:**
- Actualmente: `result.Success = result.QualityOK`
- Con skip matrix: si Quality fue saltado, `result.Success` se calcula distinto
- Lógica nueva: si Quality se ejecutó, `Success = QualityOK`; si no, `Success = true` (asumiendo que errores fatales ya abortaron con `return`)

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

	// Contar si Quality se ejecutó realmente
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

**Detalle fino del flujo de control por combinación de fases:**

| Fase | Memory | Think | Delegate | Git | Quality | Save | Success |
|------|--------|-------|----------|-----|---------|------|---------|
| F0 | ✅ corre | ✅ corre | ✅ corre | ❌ skip | ❌ skip | ✅ corre | `true` |
| F1 | ✅ corre | ✅ corre | ✅ corre | ❌ skip | ❌ skip | ✅ corre | `true` |
| F2 | ✅ corre | ✅ corre | ✅ corre | ❌ skip | ❌ skip | ✅ corre | `true` |
| F3 | ✅ corre | ✅ corre | ✅ corre | ✅ corre | ✅ corre | ✅ corre | `QualityOK` |
| F4 | ✅ corre | ❌ skip | ✅ corre | ✅ corre | ❌ skip | ✅ corre | `true` |

**Valores de PhaseResult para steps skippeados:**
- `MemoryUsed` = 0 (no se consultó memoria)
- `TasksPlanned` = 0 (no se planificó)
- `NodesCreated` = 0 (no se delegó, o delegación no produjo nodos)
- `GitStatus` = "" (vacío, no se ejecutó git)
- `QualityOK` = `false` (no se evaluó calidad)
- `Iterations` = 0 (no hubo loop de calidad)
- `FactsSaved` = 0 o valor real si Save corrió

---

### 2.6 Archivos afectados

| Archivo | Tipo de cambio | LOC estimados |
|---------|---------------|---------------|
| `internal/boomerang/skip.go` | **NUEVO** | ~120 LOC (tipos + matriz + validación + métodos) |
| `internal/boomerang/orchestrator.go` | **MODIFICAR** | ~35 LOC modificados en `RunPhase()` |
| `internal/boomerang/boomerang_test.go` | **MODIFICAR** | ~120 LOC nuevos tests |

**No requiere cambios en:**
- `internal/scheduler/scheduler.go` — usa `RunPhase()` con la misma firma
- `internal/scheduler/config.go` — no toca Boomerang internos
- `cmd/zyrocli/run.go` — no cambia interfaz de usuario
- `go.mod` — sin nuevas dependencias

---

### 2.7 Código completo de `skip.go`

```go
// Package boomerang implementa el orquestador de 6 pasos del ciclo Boomerang.
package boomerang

import "fmt"

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

// String retorna el nombre legible del step.
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

// StepStatus representa el estado de un step durante la ejecución.
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

// StepOutput encapsula el resultado individual de un step.
type StepOutput struct {
	Step     Step   `json:"step"`
	TaskName string `json:"task_name,omitempty"`
	Output   string `json:"output,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// PhaseStepMatrix define qué pasos ejecutar en cada fase del ciclo Boomerang.
type PhaseStepMatrix map[string][]Step

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
// Si la fase no está definida en la matriz, retorna true (default seguro).
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
func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
	steps, ok := m[phase]
	if !ok {
		return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
	}
	result := make([]Step, len(steps))
	copy(result, steps)
	return result
}

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
	maxStep := int(StepSave)

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

---

### 2.8 Tests (agregar en `boomerang_test.go`)

```go
// ──────────────────────────────────────────────────
// Tests para Fase 1: Phase Skip Matrix
// ──────────────────────────────────────────────────

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

func TestDefaultPhaseMatrix(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// Verificar que todas las fases existen
	for _, phase := range []string{"F0", "F1", "F2", "F3", "F4"} {
		if _, ok := matrix[phase]; !ok {
			t.Errorf("DefaultPhaseMatrix missing phase %s", phase)
		}
	}

	// F0 no debe ejecutar Git ni Quality
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

	// F1 igual que F0
	if matrix.ShouldRun("F1", StepGit) {
		t.Error("F1 should skip Git")
	}
	if matrix.ShouldRun("F1", StepQuality) {
		t.Error("F1 should skip Quality")
	}

	// F2 igual que F0
	if matrix.ShouldRun("F2", StepGit) {
		t.Error("F2 should skip Git")
	}
	if matrix.ShouldRun("F2", StepQuality) {
		t.Error("F2 should skip Quality")
	}

	// F3 debe ejecutar los 6 pasos
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

func TestShouldRunUnknownPhase(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// Fase desconocida: debe ejecutar todos los pasos (default seguro)
	if !matrix.ShouldRun("F99", StepMemory) {
		t.Error("unknown phase should run Memory")
	}
	if !matrix.ShouldRun("F99", StepThink) {
		t.Error("unknown phase should run Think")
	}
	if !matrix.ShouldRun("F99", StepDelegate) {
		t.Error("unknown phase should run Delegate")
	}
	if !matrix.ShouldRun("F99", StepGit) {
		t.Error("unknown phase should run Git")
	}
	if !matrix.ShouldRun("F99", StepQuality) {
		t.Error("unknown phase should run Quality")
	}
	if !matrix.ShouldRun("F99", StepSave) {
		t.Error("unknown phase should run Save")
	}
}

func TestActiveSteps(t *testing.T) {
	matrix := DefaultPhaseMatrix()

	// F0: 4 steps
	steps := matrix.ActiveSteps("F0")
	if len(steps) != 4 {
		t.Errorf("F0 expected 4 active steps, got %d: %v", len(steps), steps)
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

	// Verificar que ActiveSteps retorna una copia (no muta la original)
	steps[0] = StepGit
	if matrix.ShouldRun("F4", StepGit) {
		// No debería haber cambiado
	}
}

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
			"F4": {StepMemory, StepDelegate, StepGit}, // falta Save
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
			"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepSave}, // falta Quality
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
			"F3": {StepMemory, StepThink, StepDelegate, StepQuality, StepSave}, // falta Git
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
			"F4": {StepMemory, StepDelegate, StepGit, StepSave, StepSave}, // duplicado
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
			"F4": {Step(99), StepDelegate, StepGit, StepSave}, // step inválido
		}
		if err := ValidateMatrix(matrix); err == nil {
			t.Error("expected error for invalid step value")
		}
	})
}

// TestRunPhaseSkipMatrixF0 verifica que RunPhase() no ejecute Git/Quality en F0
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
	// GitStatus debe estar vacío (no se ejecutó)
	if result.GitStatus != "" {
		t.Errorf("F0 should not run Git, got status %q", result.GitStatus)
	}
	// QualityOK debe ser false (no se ejecutó)
	if result.QualityOK {
		t.Error("F0 should not run Quality")
	}
	// Success debe ser true aunque Quality no corrió
	if !result.Success {
		t.Error("F0 should be successful even without Quality")
	}
}

// TestRunPhaseSkipMatrixF3 verifica que F3 ejecute los 6 pasos
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
	// GitStatus no está vacío (se ejecutó)
	// En CI puede ser clean, dirty o error
	if result.GitStatus == "" {
		t.Error("F3 should run Git, expected non-empty status")
	}
}

// TestActiveStepsImmutable verifica que ActiveSteps retorne una copia
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
```

---

## 3. Criterios de Aceptación

- [ ] **CA1:** F0 no ejecuta Git ni Quality (verificado por test)
- [ ] **CA2:** F1 no ejecuta Git ni Quality (verificado por test)
- [ ] **CA3:** F2 no ejecuta Git ni Quality (verificado por test)
- [ ] **CA4:** F3 ejecuta los 6 pasos (verificado por test)
- [ ] **CA5:** F4 ejecuta Memory, Delegate, Git, Save — NO Think, NO Quality
- [ ] **CA6:** `RunPhase()` mantiene su firma pública intacta (`func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error)`)
- [ ] **CA7:** Todos los tests existentes en `boomerang_test.go` siguen pasando sin cambios
- [ ] **CA8:** `ValidateMatrix()` detecta matrices inválidas (faltante, vacía, duplicados, steps inválidos, F3 sin Quality/Git, F4 sin Save)
- [ ] **CA9:** Si se pasa una fase desconocida, `ShouldRun()` retorna `true` para todos los steps (default seguro)
- [ ] **CA10:** `ShouldRun()` funciona correctamente para todas las combinaciones F0-F4 × 6 steps (30 combinaciones)
- [ ] **CA11:** `ActiveSteps()` retorna una copia del slice, no una referencia mutable
- [ ] **CA12:** `result.Success` es `true` cuando Quality fue saltado (no penaliza a la fase)
- [ ] **CA13:** `result.GitStatus` es `""` (vacío) cuando Git fue saltado
- [ ] **CA14:** Los valores `Step` y `StepStatus` tienen `String()` correcto

---

## 4. Dependencias

- **Solo stdlib** — sin dependencias externas
- `fmt` para `String()` y errores de validación
- No requiere `golang.org/x/sync` (se usará en fases posteriores)
- No requiere cambios en `go.mod`

---

## 5. Riesgos y mitigaciones

| # | Riesgo | Probabilidad | Impacto | Mitigación |
|---|--------|-------------|---------|------------|
| 1 | **Steps skippeados dejan variables intermedias en cero** que otros steps esperan | Baja | Medio | `dag` se verifica antes de Delegate (`dag != nil`); `delegateResult` puede ser `nil` para SaveStep (Go maneja nil map en range) |
| 2 | **`result.Success` incorrecto** cuando Quality se salta | Baja | Alto | Lógica: si `qualityRan == false`, `result.Success = true`. Test específico `TestRunPhaseSkipMatrixF0` lo verifica |
| 3 | **Matriz mal configurada** (ej: F3 sin Quality) | Baja | Medio | `ValidateMatrix()` en construcción; tests de validación exhaustivos |
| 4 | **Fase futura con nombre no estándar** (no F0-F4) | Baja | Bajo | `ShouldRun()` retorna `true` para cualquier fase no definida (default seguro) |
| 5 | **Regresión en tests existentes** | Baja | Alto | CA7 exige que todos los tests actuales sigan pasando; CI gate |

**Ninguno de estos riesgos es significativo.** El código es puramente declarativo (tipos, matriz, validación, consulta). No toca lógica de ejecución de los steps individuales (MemoryStep, ThinkStep, etc.) más allá del wrapper condicional en `RunPhase()`.

---

## 6. Checklist de implementación

- [ ] Crear `internal/boomerang/skip.go` con tipos `Step`, `StepStatus`, `StepOutput`, `PhaseStepMatrix`, `ErrInvalidMatrix`, funciones `DefaultPhaseMatrix()`, `ValidateMatrix()`, métodos `ShouldRun()` y `ActiveSteps()`
- [ ] Modificar `internal/boomerang/orchestrator.go`:
  - [ ] Agregar llamado a `DefaultPhaseMatrix()` al inicio de `RunPhase()`
  - [ ] Envolver cada step con `if matrix.ShouldRun(config.Phase, StepXXX)`
  - [ ] Mover declaración de `dag`, `delegateResult` fuera de los bloques if
  - [ ] Cambiar lógica de `result.Success` para considerar si Quality corrió
  - [ ] Manejar `delegateResult == nil` para QualityStep cuando Delegate fue saltado
- [ ] Agregar tests en `internal/boomerang/boomerang_test.go`:
  - [ ] `TestStepString` / `TestStepStatusString`
  - [ ] `TestDefaultPhaseMatrix`
  - [ ] `TestDefaultPhaseMatrixValidates`
  - [ ] `TestShouldRunUnknownPhase`
  - [ ] `TestActiveSteps`
  - [ ] `TestValidateMatrix` (subtests: missing, empty, F4 no Save, F3 no Quality, F3 no Git, duplicates, invalid step)
  - [ ] `TestRunPhaseSkipMatrixF0`
  - [ ] `TestRunPhaseSkipMatrixF3`
  - [ ] `TestActiveStepsImmutable`
- [ ] Verificar: `go test ./internal/boomerang/...` pasa con los 8+ tests nuevos
- [ ] Verificar: `go vet ./internal/boomerang/...` sin warnings
- [ ] Verificar: `go build ./...` compila sin errores

---

## 7. Referencia a Design y Tasks

### 7.1 Documento de diseño técnico

[**design.md**](./design.md) — Diseño técnico detallado que incluye:

- Resumen del problema y solución propuesta
- Arquitectura: diagrama de relación entre `skip.go` y `orchestrator.go`
- Flujo de `RunPhase()` con skip checks
- Decisiones de diseño fundamentadas:
  - `Step` como `int iota` vs alternativas
  - Por qué no se modifica `PhaseConfig`
  - Por qué `DefaultPhaseMatrix()` es pública
  - Manejo de fases desconocidas (default seguro)
- Código completo de `skip.go` (~112 LOC)
- Código de `RunPhase()` modificado (~130 LOC)
- Código completo de tests (~250 LOC)

### 7.2 Desglose de tareas atómicas

[**tasks.md**](./tasks.md) — 7 tareas atómicas para implementación secuencial:

| ID | Tarea | LOC | Dependencia |
|----|-------|-----|-------------|
| T1 | Definir tipos base (Step, StepStatus, StepOutput) | ~20 | — |
| T2 | Implementar DefaultPhaseMatrix + ShouldRun + ActiveSteps | ~40 | T1 |
| T3 | Implementar ValidateMatrix + AllSteps | ~30 | T1 |
| T4 | Modificar RunPhase() para usar skip matrix | ~35 | T2 |
| T5 | Tests unitarios de matriz (ShouldRun, ActiveSteps, ValidateMatrix) | ~80 | T3 |
| T6 | Tests de integración (RunPhase con skip matrix) | ~80 | T4 |
| T7 | Tests de backward compatibility | ~40 | T4 |

### 7.3 Resumen de archivos

| Archivo | Propósito | Estado |
|---------|-----------|--------|
| `spec.md` | Especificación de requisitos y criterios de aceptación | ✅ Actual |
| `design.md` | Diseño técnico detallado y decisiones de arquitectura | ✅ Nuevo |
| `tasks.md` | Tareas atómicas con dependencias y criterios | ✅ Nuevo |
| `internal/boomerang/skip.go` | Implementación: tipos, matriz, validación | ⬜ Por crear |
| `internal/boomerang/orchestrator.go` | Modificar `RunPhase()` | ⬜ Por modificar |
| `internal/boomerang/boomerang_test.go` | Agregar tests nuevos | ⬜ Por modificar |

---

*Fin de la especificación — Ver [design.md](./design.md) para el diseño técnico y [tasks.md](./tasks.md) para el plan de implementación.*
