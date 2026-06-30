# Plan de Optimización: BoomerangOrchestrator

> **Fecha:** 17 Junio 2026
> **Versión:** 1.0
> **Contexto:** ZyroAgentCLI — `BoomerangOrchestrator` + `Scheduler` SDD (F0–F4)
> **Propósito:** Phase Skip Matrix + Async Event Loop para delegación responsiva

---

## Tabla de Contenidos

1. [Resumen Ejecutivo](#1-resumen-ejecutivo)
2. [Problemas Identificados](#2-problemas-identificados)
3. [Arquitectura Propuesta: BoomerangAsync](#3-arquitectura-propuesta-boomerangasync)
4. [Phase Skip Matrix](#4-phase-skip-matrix)
5. [Async Event Loop](#5-async-event-loop)
6. [Cambios Específicos por Archivo](#6-cambios-específicos-por-archivo)
7. [Nuevos Archivos a Crear](#7-nuevos-archivos-a-crear)
8. [Estrategia de Implementación](#8-estrategia-de-implementación)
9. [Backward Compatibility](#9-backward-compatibility)
10. [Riesgos y Mitigaciones](#10-riesgos-y-mitigaciones)
11. [Diagrama General de la Solución](#11-diagrama-general-de-la-solución)

---

## 1. Resumen Ejecutivo

### 1.1 Estado Actual

El `BoomerangOrchestrator` es un orquestador de 6 pasos secuenciales y bloqueantes:

```
Memory → Think → Delegate → Git → Quality → Save
```

**Problemas críticos detectados:**

| # | Problema | Impacto | Archivo/Líneas |
|---|----------|---------|-----------------|
| 1 | **Sin phase-skip**: los 6 pasos se ejecutan en TODAS las fases (F0–F4) | ~40% tiempo perdido en fases de investigación/diseño | `orchestrator.go:111-194` |
| 2 | **DelegateStep bloqueante**: `cmd.Output()` + `wg.Wait()` congelan el orquestador | No se puede recibir input del usuario mientras los subagentes trabajan | `delegate.go:39-63` |
| 3 | **Sin event loop**: no hay canales para notificaciones, progreso, ni cancelación | UX pobre, sin visibilidad de progreso, sin cancelación limpia | `orchestrator.go` (ausencia total) |
| 4 | **PhaseConfig rígido**: no permite configurar qué pasos ejecutar ni modo async | Extensibilidad limitada, cada cambio requiere modificar `RunPhase()` | `orchestrator.go:12-20` |
| 5 | **QualityStep sin contexto por fase**: ejecuta `go build` y `go test` incluso en F0 | Falso positivo/negativo en fases no-implementación | `quality.go:11-34` |

### 1.2 Solución Propuesta

Arquitectura **BoomerangAsync** basada en 3 pilares:

| Pilar | Patrón | Implementación |
|-------|--------|----------------|
| **Phase Skip Matrix** | Matriz declarativa `[Phase][Step]bool` | `PhaseStepMatrix` + `PhaseConfigV2` |
| **Async Event Loop** | `select{}` sobre múltiples channels | `EventBus` + `RunPhaseAsync()` |
| **Delegate no-bloqueante** | `cmd.Start()` + `cmd.StdoutPipe()` + channels | `DelegateStepAsync()` con `errgroup.Group` |

### 1.3 Beneficios Esperados

- **40%** reducción de tiempo en F0, F1, F2, F4 saltando pasos innecesarios
- **Responsividad total**: el usuario puede enviar input mientras los subagentes trabajan
- **Progreso streaming**: notificaciones en tiempo real de cada subagente
- **Cancelación granular**: cancelar subagentes individuales sin matar el proceso
- **Fase completa**: el event loop permanece vivo escuchando `userInput`, `subagentDone`, `approval`, `ctx.Done()`
- **0 nuevas dependencias externas** — todo se implementa con stdlib + `errgroup` (golang.org/x/sync)

---

## 2. Problemas Identificados

### 2.1 `orchestrator.go:111-194` — RunPhase() monolítico

```go
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
    // Paso 1: MEMORY — siempre se ejecuta (L117)
    memoryCtx, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)

    // Paso 2: THINK — siempre se ejecuta (L130)
    dag, err := o.ThinkStep(ctx, config.Phase, memoryCtx)

    // Paso 3: DELEGATE — siempre se ejecuta, BLOQUEANTE (L138)
    delegateResult, err := o.DelegateStep(ctx, dag, config.Phase)

    // Paso 4: GIT — siempre se ejecuta (L148)
    gitStatus, err := o.GitStep(ctx)

    // Paso 5: QUALITY — siempre se ejecuta con loop de retry (L158-169)
    for i := 0; i < o.maxIterations; i++ {
        qualityOK, err := o.QualityStep(ctx, config.Phase, dag, delegateResult)
    }

    // Paso 6: SAVE — siempre se ejecuta (L173)
    saveResult, err := o.SaveStep(ctx, config.Phase, delegateResult, nil)
}
```

**Problemas:**
- No hay switch por fase → todos los pasos siempre se ejecutan
- Si `MemoryStep` falla, no hay fallback (contexto vacío sería aceptable)
- `DelegateStep` bloquea hasta que todos los subagentes terminan
- `QualityStep` en F0 intenta `go build` aunque no haya código
- No hay manera de cancelar una fase en medio de la ejecución

### 2.2 `delegate.go:39-63` — DelegateStep bloqueante con falso paralelismo

```go
cmd := exec.CommandContext(ctx, "opencode", "subagent", t.Agent, ...)
output, err := cmd.Output()  // BLOQUEA la goroutine (L45)

// ...

wg.Wait()  // BLOQUEA al orquestador hasta que TODAS terminan (L63)
```

Aunque usa goroutines, el patrón `wg.Wait()` congela al orquestador:
- No puede recibir input del usuario mientras los subagentes trabajan
- No puede notificar progreso parcial
- No puede cancelar un subagente específico
- Si un subagente se cuelga, no hay timeout granular por tarea

### 2.3 `orchestrator.go:12-20` — PhaseConfig insuficiente

```go
type PhaseConfig struct {
    Phase       string
    TaskDesc    string
    ProjectID   string
    MemoryLimit int
    Iterations  int
    Timeout     time.Duration
    // ❌ No hay: Steps, AsyncMode, SkipMatrix, Parallelism
}
```

Sin capacidad de configurar:
- Qué pasos ejecutar (steps)
- Modo async vs síncrono
- Máximo de subagentes concurrentes
- Matriz de skip por fase

### 2.4 `quality.go:11-34` — QualityStep monolítico

```go
if phase == "F3" {
    if err := exec.CommandContext(ctx, "go build ./...").Run(); err != nil {
        return false, err  // En F0, F1, F2, F4 esto no tiene sentido
    }
}
if phase == "F3" {
    if err := exec.CommandContext(ctx, "go test ./...").Run(); err != nil {
        return false, err
    }
}
```

- `go build` en F0 falla porque no hay código generado aún
- `go test` en F2 falla por la misma razón
- No hay validación específica por fase (ej: en F1 validar especificación YAML)

### 2.5 `scheduler.go:50-72` — Integración Boomerang sin async

```go
if s.config.Boomerang != nil {
    boomerangResult, boomerErr := s.config.Boomerang.RunPhase(phaseCtx, ...)
    // ↑ Bloquea hasta que toda la fase termina
}
// ...
approved, err := ApprovalGate(result.Phase, result.Summary)
// ↑ Bloquea esperando input del usuario — pero sin event loop,
//   no hay manera de interrumpir la approval gate si el usuario
//   quiere cancelar desde otro canal
```

El Scheduler actual:
- Ejecuta cada fase de forma bloqueante
- Hace approval gate bloqueante entre fases
- No tiene canales para eventos de progreso

### 2.6 `phase_stubs.go` — Phase runners sin Boomerang (ejecución secuencial)

Cada runner (F0Runner–F4Runner) usa `exec.CommandContext` de forma bloqueante. Los runners que no usan Boomerang son igualmente sincrónicos y no reportan progreso hasta que terminan por completo.

---

## 3. Arquitectura Propuesta: BoomerangAsync

### 3.1 Visión General

```
┌──────────────────────────────────────────────────────────────────┐
│                    BoomerangAsyncOrchestrator                     │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │                    Event Loop Central                       │  │
│  │                    select {                                 │  │
│  │                      case event  := <-o.bus                 │  │
│  │                      case input   := <-o.userInput          │  │
│  │                      case <-ctx.Done()                       │  │
│  │                      case <-time.After(...)                  │  │
│  │                    }                                         │  │
│  └──────────┬─────────────────────────────────────────────────┘  │
│             │                                                    │
│    ┌────────┼────────┬────────┬────────┬────────┬────────┐       │
│    ▼        ▼        ▼        ▼        ▼        ▼        ▼       │
│  Memory   Think   Delegate   Git    Quality   Save   UserInput   │
│  Step     Step    Step      Step    Step     Step    Step        │
│                                                                    │
│  Fase: [F0│F1│F2│F3│F4]  ←  Phase Skip Matrix decide qué steps  │
└──────────────────────────────────────────────────────────────────┘
         │
         ▼  eventos asíncronos (StepEvent chan)
┌──────────────────────────────────────────────────────────────────┐
│           CLI / TUI / OpenCode Bridge                            │
│  - Muestra progreso en tiempo real                               │
│  - Recibe input del usuario mientras orquesta                    │
│  - Muestra approval gate sin bloqueo                             │
└──────────────────────────────────────────────────────────────────┘
```

### 3.2 Tipos Fundamentales

```go
// Step identifica cada paso del ciclo Boomerang (nuevo)
type Step int

const (
    StepMemory   Step = iota
    StepThink
    StepDelegate
    StepGit
    StepQuality
    StepSave
)

// StepStatus representa el estado de un step
type StepStatus string

const (
    StepPending   StepStatus = "pending"
    StepRunning   StepStatus = "running"
    StepDone      StepStatus = "done"
    StepSkipped   StepStatus = "skipped"
    StepFailed    StepStatus = "failed"
)
```

### 3.3 Fases de Implementación

| Fase | Nombre | Cambios principales |
|------|--------|---------------------|
| **F1** | Phase Skip Matrix | Agregar `PhaseStepMatrix`, `ShouldRun()`, modificar `RunPhase()` |
| **F2** | PhaseConfigV2 | Nuevo `PhaseConfigV2` con `Steps []Step`, convivencia con `PhaseConfig` |
| **F3** | DelegateStep Async | `DelegateStepAsync()` con `cmd.Start()` + `errgroup`, streaming de progreso |
| **F4** | Event Bus + Event Loop | Nuevo `EventBus`, `RunPhaseAsync()` con event loop central |
| **F5** | Approval Gate Async | Approval gate no-bloqueante con canales de aprobación |
| **F6** | Scheduler Async | `RunAsync()` en scheduler, integración con evento de progreso |
| **F7** | Legacy Cleanup | Deprecar `RunPhase()` síncrono, mantener para backward compat |

---

## 4. Phase Skip Matrix

### 4.1 Concepto

Matriz declarativa que determina qué pasos ejecutar en cada fase. Reemplaza el "siempre todos" actual.

### 4.2 Matriz por Defecto

| Fase | Memory | Think | Delegate | Git | Quality | Save | Ahorro |
|------|--------|-------|----------|-----|---------|------|--------|
| **F0** | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ~40% |
| **F1** | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ~40% |
| **F2** | ✅ | ✅ | ✅ | ❌ | ❌ | ✅ | ~40% |
| **F3** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | 0% |
| **F4** | ✅ | ❌ | ✅ | ✅ | ❌ | ✅ | ~35% |

### 4.3 Justificación Paso a Paso

**F0 (Investigación):**
- **Memory** ✅: Necesita contexto causal de qué investigar
- **Think** ✅: Planifica DAG de 3 subagentes de investigación
- **Delegate** ✅: Ejecuta subagentes de patrones/librerías/skills
- **Git** ❌: No hay código que modificar, es investigación pura
- **Quality** ❌: No hay código que compilar o testear
- **Save** ✅: Guarda hechos descubiertos en memoria causal

**F1 (Especificación):**
- **Memory** ✅: Contexto de investigación F0
- **Think** ✅: Planifica spec + review secuencial
- **Delegate** ✅: Ejecuta subagente de especificación
- **Git** ❌: Aún no hay código
- **Quality** ❌: No hay código que validar
- **Save** ✅: Guarda la especificación como hechos

**F2 (Diseño):**
- **Memory** ✅: Contexto de especificación F1
- **Think** ✅: Planifica diseño técnico
- **Delegate** ✅: Ejecuta subagente de diseño
- **Git** ❌: Aún no hay implementación
- **Quality** ❌: No hay código que compilar
- **Save** ✅: Guarda diseño y tareas

**F3 (Implementación):**
- **Todos** ✅: Fase completa — necesita validación de código

**F4 (Cierre):**
- **Memory** ✅: Necesita recuperar hechos de F3 para auditarlos/archivarlos correctamente
- **Think** ❌: F4 no planifica nuevas estrategias, solo ejecuta el build final y guarda
- **Delegate** ✅: Ejecuta subagente de archivo
- **Git** ✅: Verifica que todo esté commiteado
- **Quality** ❌: No hay nuevo código que validar
- **Save** ✅: Guarda registro de cierre

### 4.4 Código: PhaseStepMatrix

```go
// skip.go
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
        return true // por defecto: ejecutar todos
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
    return steps
}
```

### 4.5 PhaseConfigV2

```go
// FailurePolicy define cómo manejar fallos de subagentes durante Delegate
type FailurePolicy int

const (
    FailurePolicyFailFast         FailurePolicy = iota // Cancelar todos si uno falla (default)
    FailurePolicyContinueOnError                       // Recolectar errores sin abortar
)

// PhaseConfigV2 configura una fase con soporte de skip matrix y modo async
type PhaseConfigV2 struct {
    Phase          string
    TaskDesc       string
    ProjectID      string
    Steps          []Step           // nil = usar SkipMatrix
    Timeout        time.Duration
    Parallelism    int              // subagentes concurrentes máx. (default 3)
    AsyncMode      bool             // true = event loop, false = síncrono legacy
    SkipMatrix     PhaseStepMatrix  // nil = usar DefaultPhaseMatrix()
    FailurePolicy  FailurePolicy    // FailFast (default) o ContinueOnError
}
```

### 4.6 Integración con RunPhase existente

Modificar `RunPhase()` para usar skip matrix:

```go
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
    // Construir PhaseConfigV2 a partir de PhaseConfig (backward compat)
    v2 := PhaseConfigV2{
        Phase:     config.Phase,
        TaskDesc:  config.TaskDesc,
        Timeout:   config.Timeout,
        AsyncMode: false,
    }
    return o.runPhaseV2(ctx, v2)
}

func (o *BoomerangOrchestrator) runPhaseV2(ctx context.Context, config PhaseConfigV2) (*PhaseResult, error) {
    start := time.Now()
    result := &PhaseResult{Phase: config.Phase}

    // Determinar steps a ejecutar
    matrix := config.SkipMatrix
    if matrix == nil {
        matrix = DefaultPhaseMatrix()
    }
    steps := config.Steps
    if steps == nil {
        steps = matrix.ActiveSteps(config.Phase)
    }

    for _, step := range steps {
        switch step {
        case StepMemory:
            memoryCtx, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)
            if err != nil { return nil, err }
            result.MemoryUsed = len(memoryCtx)

        case StepThink:
            dag, err := o.ThinkStep(ctx, config.Phase, "") // memoryCtx desde phaseState
            if err != nil { return nil, err }
            result.TasksPlanned = len(dag.Tasks)

        case StepDelegate:
            delegateResult, err := o.DelegateStep(ctx, dag, config.Phase)
            // ...
        // ... etc
        }
    }

    result.Duration = time.Since(start)
    result.Success = true
    return result, nil
}
```

---

## 5. Async Event Loop

### 5.1 Diagrama de Flujo

```
┌──────────────────┐     ┌──────────────────┐     ┌──────────────────┐
│  RunPhaseAsync() │     │  Event Loop       │     │  Consumidor      │
│  (goroutine)     │────►│  (select)         │────►│  (CLI/TUI)       │
└──────────────────┘     └──────────────────┘     └──────────────────┘
         │                       │
         │  events chan          │  events chan
         ▼                       ▼
  ┌──────────────┐      ┌──────────────┐
  │ StepMemory   │      │ userInput    │
  │ StepThink    │      │ approval     │
  │ StepDelegate─┼──────┤ subagentDone │
  │   ├─task1    │      │ subagentErr  │
  │   ├─task2    │      │ ctx.Done()   │
  │   └─task3    │      │ timeout      │
  │ StepGit      │      └──────────────┘
  │ StepQuality  │
  │ StepSave     │
  └──────────────┘
```

### 5.2 Tipos de Eventos

```go
// events.go
type EventType int

const (
    EventStepStarted      EventType = iota  // Step comenzó
    EventStepCompleted                       // Step completado exitosamente
    EventStepSkipped                         // Step saltado (skip matrix)
    EventStepFailed                          // Step falló
    EventSubagentStarted                     // Subagente comenzó
    EventSubagentOutput                      // Línea de output del subagente
    EventSubagentCompleted                   // Subagente terminó
    EventSubagentFailed                      // Subagente falló
    EventUserInput                           // Input del usuario recibido
    EventApprovalRequired                    // Se necesita aprobación para avanzar
    EventApprovalGranted                     // Usuario aprobó
    EventApprovalDenied                      // Usuario rechazó
    EventPhaseStarted                        // Fase comenzó
    EventPhaseCompleted                      // Fase completada
    EventPhaseFailed                         // Fase falló
)
```

### 5.3 Event Bus

```go
// eventbus.go
type StepEvent struct {
    Type    EventType
    Phase   string
    Step    Step
    Status  StepStatus
    Data    interface{}  // TaskSpec, TaskResult, PhaseResult, etc.
    Error   error
    Time    time.Time
}

type EventBus struct {
    subscribers map[EventType][]chan StepEvent
    mu          sync.RWMutex
}

func NewEventBus(buffer int) *EventBus {
    return &EventBus{
        subscribers: make(map[EventType][]chan StepEvent),
    }
}

func (b *EventBus) Subscribe(eventType EventType, buffer int) chan StepEvent {
    ch := make(chan StepEvent, buffer)
    b.mu.Lock()
    b.subscribers[eventType] = append(b.subscribers[eventType], ch)
    b.mu.Unlock()
    return ch
}

func (b *EventBus) Publish(event StepEvent) {
    b.mu.RLock()
    channels := b.subscribers[event.Type]
    b.mu.RUnlock()
    for _, ch := range channels {
        select {
        case ch <- event:
        default: // non-blocking: drop slow consumers
        }
    }
}
```

### 5.4 Event Loop Central

```go
// async_orchestrator.go
type BoomerangAsyncOrchestrator struct {
    // Embed el orquestador base para reutilizar steps
    *BoomerangOrchestrator

    // Channels del event loop
    bus         EventBus
    userInput   chan string
    approvalReq chan ApprovalRequest

    // Estado
    phaseState  *PhaseState
    activeTasks map[string]context.CancelFunc
    mu          sync.RWMutex
}
```

### 5.5 RunPhaseAsync() — Event Loop

```go
func (o *BoomerangAsyncOrchestrator) RunPhaseAsync(
    ctx context.Context,
    config PhaseConfigV2,
) <-chan StepEvent {
    events := make(chan StepEvent, 100)
    phaseCtx, cancel := context.WithCancel(ctx)

    go func() {
        defer close(events)
        defer cancel()

        // Determinar steps
        matrix := config.SkipMatrix
        if matrix == nil {
            matrix = DefaultPhaseMatrix()
        }
        steps := config.Steps
        if steps == nil {
            steps = matrix.ActiveSteps(config.Phase)
        }

        // Publicar inicio de fase
        events <- StepEvent{
            Type:  EventPhaseStarted,
            Phase: config.Phase,
            Time:  time.Now(),
        }

        for _, step := range steps {
            // Check cancelación antes de cada step
            select {
            case <-phaseCtx.Done():
                events <- StepEvent{
                    Type: EventStepFailed, Phase: config.Phase,
                    Step: step, Status: StepFailed, Error: phaseCtx.Err(),
                }
                return
            default:
            }

            // Emitir inicio de step
            events <- StepEvent{
                Type: EventStepStarted, Phase: config.Phase,
                Step: step, Status: StepRunning,
            }

            // Ejecutar step (puede emitir eventos internos)
            err := o.executeStepAsync(phaseCtx, step, config, events)

            if err != nil {
                events <- StepEvent{
                    Type: EventStepFailed, Phase: config.Phase,
                    Step: step, Status: StepFailed, Error: err,
                }
                // Steps críticos abortan la fase
                if step == StepDelegate || step == StepQuality {
                    events <- StepEvent{
                        Type: EventPhaseFailed, Phase: config.Phase,
                        Error: err,
                    }
                    return
                }
            } else {
                events <- StepEvent{
                    Type: EventStepCompleted, Phase: config.Phase,
                    Step: step, Status: StepDone,
                }
            }
        }

        // Fase completada
        events <- StepEvent{
            Type: EventPhaseCompleted, Phase: config.Phase,
            Time: time.Now(),
        }
    }()

    return events
}
```

### 5.6 DelegateStepAsync — No Bloqueante

```go
func (o *BoomerangAsyncOrchestrator) executeStepAsync(
    ctx context.Context,
    step Step,
    config PhaseConfigV2,
    events chan<- StepEvent,
) error {
    switch step {
    case StepMemory:
        memoryCtx, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)
        if err == nil {
            o.phaseState.MemoryContext = memoryCtx
        }
        return err

    case StepThink:
        dag, err := o.ThinkStep(ctx, config.Phase, o.phaseState.MemoryContext)
        if err == nil {
            o.phaseState.DAG = dag
        }
        return err

    case StepDelegate:
        return o.delegateStepAsync(ctx, config, events)

    case StepGit:
        status, err := o.GitStep(ctx)
        if err == nil {
            o.phaseState.GitStatus = status
        }
        return err

    case StepQuality:
        return o.qualityStepAsync(ctx, config, events)

    case StepSave:
        result, err := o.SaveStep(ctx, config.Phase, o.phaseState.DelegateResult, nil)
        if err == nil {
            o.phaseState.SaveResult = result
        }
        return err

    default:
        return fmt.Errorf("boomerang: unknown step %v", step)
    }
}

func (o *BoomerangAsyncOrchestrator) delegateStepAsync(
    ctx context.Context,
    config PhaseConfigV2,
    events chan<- StepEvent,
) error {
    dag := o.phaseState.DAG
    if dag == nil {
        return fmt.Errorf("boomerang: no DAG available for delegate step")
    }

    parallelism := config.Parallelism
    if parallelism <= 0 {
        parallelism = 3
    }

    // Usamos errgroup para manejo de errores y cancelación
    g, gCtx := errgroup.WithContext(ctx)
    g.SetLimit(parallelism)

    var mu sync.Mutex
    taskResults := make(map[string]TaskResult)
    nodesCreated := 0
    var taskErrors []error

    for _, group := range dag.ParallelGroups {
        // Verificar cancelación entre grupos
        select {
        case <-gCtx.Done():
            return gCtx.Err()
        default:
        }

        for _, taskIdx := range group {
            if taskIdx >= len(dag.Tasks) {
                continue
            }
            task := dag.Tasks[taskIdx]

            g.Go(func() error {
                // Crear contexto cancelable para esta tarea
                taskCtx, taskCancel := context.WithCancel(gCtx)

                // Registrar tarea activa (para cancelación externa)
                o.mu.Lock()
                o.activeTasks[task.Name] = taskCancel
                o.mu.Unlock()
                defer func() {
                    o.mu.Lock()
                    delete(o.activeTasks, task.Name)
                    o.mu.Unlock()
                }()

                // Notificar inicio de subagente
                events <- StepEvent{
                    Type: EventSubagentStarted, Phase: config.Phase,
                    Step: StepDelegate, Data: task,
                }

                // Lanzar proceso no bloqueante
                cmd := exec.CommandContext(taskCtx, "opencode",
                    "subagent", task.Agent,
                    "--param", fmt.Sprintf("task=%s", task.Name),
                    "--param", fmt.Sprintf("phase=%s", config.Phase),
                )

                // Pipe de stdout para streaming
                stdout, err := cmd.StdoutPipe()
                if err != nil {
                    return fmt.Errorf("%s: stdout pipe: %w", task.Name, err)
                }
                cmd.Stderr = cmd.Stdout // merge stderr → stdout

                if err := cmd.Start(); err != nil {
                    events <- StepEvent{
                        Type: EventSubagentFailed, Phase: config.Phase,
                        Step: StepDelegate, Error: err, Data: task,
                    }
                    return fmt.Errorf("%s: start: %w", task.Name, err)
                }

                // Leer output línea por línea (streaming)
                scanner := bufio.NewScanner(stdout)
                var fullOutput strings.Builder
                scannerDone := make(chan struct{})
                go func() {
                    defer close(scannerDone)
                    for scanner.Scan() {
                        line := scanner.Text()
                        fullOutput.WriteString(line + "\n")
                        select {
                        case events <- StepEvent{
                            Type: EventSubagentOutput, Phase: config.Phase,
                            Step: StepDelegate, Data: StepOutput{
                                Task: task.Name,
                                Line: line,
                            },
                        }:
                        case <-gCtx.Done():
                            _ = cmd.Process.Kill()
                            return
                        }
                    }
                }()
                select {
                case <-scannerDone:
                    // scanner terminó normalmente
                case <-gCtx.Done():
                    _ = cmd.Process.Kill()
                    return gCtx.Err()
                }

                // Esperar a que termine el proceso
                if err := cmd.Wait(); err != nil {
                    events <- StepEvent{
                        Type: EventSubagentFailed, Phase: config.Phase,
                        Step: StepDelegate, Error: err, Data: task,
                    }
                    if config.FailurePolicy == FailurePolicyContinueOnError {
                        mu.Lock()
                        taskErrors = append(taskErrors, fmt.Errorf("%s: wait: %w", task.Name, err))
                        mu.Unlock()
                        return nil // no propagar el error, continuar con otras tareas
                    }
                    return fmt.Errorf("%s: wait: %w", task.Name, err)
                }

                // Agregar resultado
                tr := TaskResult{
                    TaskName: task.Name,
                    Success:  true,
                    Output:   fullOutput.String(),
                    Nodes:    1,
                }

                mu.Lock()
                taskResults[task.Name] = tr
                nodesCreated += tr.Nodes
                mu.Unlock()

                events <- StepEvent{
                    Type: EventSubagentCompleted, Phase: config.Phase,
                    Step: StepDelegate, Data: tr,
                }
                return nil
            })
        }

        // Esperar grupo — si un grupo falla, errgroup cancela los demás
        if err := g.Wait(); err != nil {
            return err
        }
    }

    // En modo ContinueOnError, registrar errores sin abortar la fase
    if len(taskErrors) > 0 {
        o.phaseState.Errors = append(o.phaseState.Errors, taskErrors...)
    }

    // Guardar resultados en el estado de fase
    o.phaseState.DelegateResult = &DelegateResult{
        NodesCreated: nodesCreated,
        TaskResults:  taskResults,
    }
    return nil
}
```

### 5.7 QualityStep con validación por fase

```go
func (o *BoomerangAsyncOrchestrator) qualityStepAsync(
    ctx context.Context,
    config PhaseConfigV2,
    events chan<- StepEvent,
) error {
    // Validación específica por fase
    switch config.Phase {
    case "F0":
        // Validar que se encontraron patrones y librerías
        return o.validatePhase0(ctx, config)
    case "F1":
        // Validar que la especificación esté completa
        return o.validatePhase1(ctx, config)
    case "F2":
        // Validar que el diseño y tareas existan
        return o.validatePhase2(ctx, config)
    case "F3":
        // Compilar y testear
        return o.validatePhase3(ctx, config, events)
    case "F4":
        // Validar que no haya tareas pendientes
        return o.validatePhase4(ctx, config)
    default:
        return fmt.Errorf("boomerang: unknown phase %s for quality", config.Phase)
    }
}
```

### 5.8 Approval Gate No Bloqueante

```go
type ApprovalRequest struct {
    Phase   Phase
    Summary string
    Result  chan ApprovalResponse
}

type ApprovalResponse struct {
    Approved bool
    Feedback string
}

// En el event loop del scheduler:
func (s *Scheduler) RunAsync(ctx context.Context) (<-chan scheduler.Event, error) {
    events := make(chan scheduler.Event, 100)

    go func() {
        defer close(events)

        for _, phase := range s.phases {
            // ... ejecutar fase async ...

            // Emitir evento de approval requerido
            approvalCh := make(chan ApprovalResponse, 1)
            events <- scheduler.Event{
                Type:  EventNeedsApproval,
                Phase: phase.Name(),
                Data:  ApprovalRequest{
                    Phase:   phase.Name(),
                    Summary: result.Summary,
                    Result:  approvalCh,
                },
            }

            // Esperar aprobación sin bloquear el event loop
            select {
            case resp := <-approvalCh:
                if !resp.Approved {
                    events <- scheduler.Event{
                        Type: EventPhaseAborted,
                        Phase: phase.Name(),
                    }
                    return
                }
            case <-ctx.Done():
                return
            }
        }
    }()

    return events, nil
}
```

---

## 6. Cambios Específicos por Archivo

### 6.1 `internal/boomerang/orchestrator.go`

**Tipo de cambio:** Modificar

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 12-20 | **Reemplazar** `PhaseConfig` | Agregar campos opcionales `Steps`, `Parallelism`, `AsyncMode` (o crear `PhaseConfigV2` en nuevo archivo) |
| 111-194 | **Modificar** `RunPhase()` | Usar `DefaultPhaseMatrix().ShouldRun()` para cada step |
| 89-108 | **Agregar** constructor `NewBoomerangAsyncOrchestrator()` | Nuevo constructor con canales y event bus |
| — | **Agregar** `runPhaseV2()` | Método interno que usa `PhaseStepMatrix` para decidir pasos |

**Código nuevo en RunPhase():**

```go
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
    start := time.Now()
    result := &PhaseResult{Phase: config.Phase}
    matrix := DefaultPhaseMatrix()

    // Paso 1: MEMORY (solo si ShouldRun)
    if matrix.ShouldRun(config.Phase, StepMemory) {
        memoryCtx, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)
        if err != nil { return nil, err }
        result.MemoryUsed = len(memoryCtx)
    }

    // Paso 2: THINK (solo si ShouldRun)
    var dag *TaskDAG
    if matrix.ShouldRun(config.Phase, StepThink) {
        dag, err := o.ThinkStep(ctx, config.Phase, "")
        if err != nil { return nil, err }
        result.TasksPlanned = len(dag.Tasks)
    }

    // ... (similar para los demás pasos)
}
```

### 6.2 `internal/boomerang/delegate.go`

**Tipo de cambio:** Agregar (no modificar el existente)

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| — | **Agregar** `DelegateStepAsync()` | Nueva función async con `cmd.Start()` + `errgroup.Group` |
| — | **Agregar** tipos auxiliares | `StepOutput`, `TaskProgress` |
| 13-67 | **No modificar** | El `DelegateStep()` existente se mantiene para backward compat |

**Nuevo tipo auxiliar:**

```go
type StepOutput struct {
    Task string
    Line string
}
```

### 6.3 `internal/boomerang/think.go`

**Tipo de cambio:** Mínimo (o ninguno)

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 11-26 | **Evaluar** si necesita caché de DAG | Por ahora no, el switch F0-F4 es suficiente |

### 6.4 `internal/boomerang/memory.go`

**Tipo de cambio:** Mínimo

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 15-46 | **Evaluar** agregar variante async | Por ahora no necesario, MemoryStep es rápido (< 100ms) |

### 6.5 `internal/boomerang/git.go`

**Tipo de cambio:** Mínimo

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 20-32 | **Evaluar** variante no bloqueante | Por ahora no necesario, GitStep es rápido |

### 6.6 `internal/boomerang/quality.go`

**Tipo de cambio:** Reestructurar

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 11-34 | **Reemplazar** con validación por fase | `qualityStepAsync()` que switchea por fase |
| 11-34 | **Agregar** `validatePhase0()` a `validatePhase4()` | Validación específica para cada fase |

### 6.7 `internal/boomerang/save.go`

**Tipo de cambio:** Mínimo

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| 13-45 | **Evaluar** variante batch | Por ahora ok, SaveStep es rápido |

### 6.8 `internal/scheduler/scheduler.go`

**Tipo de cambio:** Agregar

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| — | **Agregar** `RunAsync()` | Nueva función que retorna `<chan scheduler.Event>` |
| 29-124 | **Modificar** `Run()` para aceptar modo async | Si `Boomerang.AsyncMode`, usar `RunPhaseAsync()` |

### 6.9 `internal/scheduler/approval.go`

**Tipo de cambio:** Mínimo

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| — | **Agregar** `ApprovalGateAsync()` | Variante no bloqueante con channel |
| 95-117 | **No modificar** | `ApprovalGate()` se mantiene para backward compat |

### 6.10 `internal/boomerang/boomerang_test.go`

**Tipo de cambio:** Agregar tests

| Líneas | Cambio | Descripción |
|--------|--------|-------------|
| — | **Agregar** `TestPhaseSkipMatrix` | Verificar que `ShouldRun()` retorne los valores correctos |
| — | **Agregar** `TestDelegateStepAsync` | Verificar delegación async con mock |
| — | **Agregar** `TestEventBus` | Verificar publish/subscribe |
| — | **Agregar** `TestRunPhaseAsync` | Verificar event loop completo |

---

## 7. Nuevos Archivos a Crear

### 7.1 `internal/boomerang/skip.go`

**Propósito:** Phase Skip Matrix y tipos Step/StepStatus.

**Contenido:**
```go
package boomerang

// Step identifica cada paso del ciclo Boomerang
type Step int

const (
    StepMemory   Step = iota
    StepThink
    StepDelegate
    StepGit
    StepQuality
    StepSave
)

func (s Step) String() string {
    return [...]string{"Memory", "Think", "Delegate", "Git", "Quality", "Save"}[s]
}

// StepStatus representa el estado de un step
type StepStatus string

const (
    StepPending   StepStatus = "pending"
    StepRunning   StepStatus = "running"
    StepDone      StepStatus = "done"
    StepSkipped   StepStatus = "skipped"
    StepFailed    StepStatus = "failed"
)

// PhaseStepMatrix define qué pasos ejecutar en cada fase
type PhaseStepMatrix map[string][]Step

// DefaultPhaseMatrix retorna la matriz de pasos por defecto
func DefaultPhaseMatrix() PhaseStepMatrix {
    return PhaseStepMatrix{
        "F0": {StepMemory, StepThink, StepDelegate, StepSave},
        "F1": {StepMemory, StepThink, StepDelegate, StepSave},
        "F2": {StepMemory, StepThink, StepDelegate, StepSave},
        "F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
        "F4": {StepMemory, StepDelegate, StepGit, StepSave},
    }
}

// ShouldRun verifica si un paso debe ejecutarse en una fase
func (m PhaseStepMatrix) ShouldRun(phase string, step Step) bool {
    steps, ok := m[phase]
    if !ok {
        return true
    }
    for _, s := range steps {
        if s == step {
            return true
        }
    }
    return false
}

// ActiveSteps retorna los pasos activos para una fase
func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
    steps, ok := m[phase]
    if !ok {
        return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
    }
    return steps
}
```

### 7.2 `internal/boomerang/events.go`

**Propósito:** Definiciones de tipos de eventos, StepEvent, EventBus.

**Contenido:** (ver sección 5.3 arriba)

### 7.3 `internal/boomerang/phase_config.go`

**Propósito:** PhaseConfigV2 con soporte de steps y async.

**Contenido:**
```go
package boomerang

import "time"

// FailurePolicy define cómo manejar fallos de subagentes durante Delegate
type FailurePolicy int

const (
    FailurePolicyFailFast         FailurePolicy = iota // Cancelar todos si uno falla (default)
    FailurePolicyContinueOnError                       // Recolectar errores sin abortar
)

// PhaseConfigV2 configura una fase con soporte de skip matrix y modo async
type PhaseConfigV2 struct {
    Phase          string
    TaskDesc       string
    ProjectID      string
    MemoryLimit    int
    Steps          []Step           // nil = usar SkipMatrix.ActiveSteps()
    Iterations     int
    Timeout        time.Duration
    Parallelism    int              // subagentes concurrentes (default 3)
    AsyncMode      bool             // true = event loop, false = síncrono legacy
    SkipMatrix     PhaseStepMatrix  // nil = usar DefaultPhaseMatrix()
    FailurePolicy  FailurePolicy    // FailFast (default) o ContinueOnError
}

// ToV2 convierte PhaseConfig legacy a PhaseConfigV2
func (c PhaseConfig) ToV2() PhaseConfigV2 {
    return PhaseConfigV2{
        Phase:       c.Phase,
        TaskDesc:    c.TaskDesc,
        ProjectID:   c.ProjectID,
        MemoryLimit: c.MemoryLimit,
        Iterations:  c.Iterations,
        Timeout:     c.Timeout,
        Parallelism: 3,
        AsyncMode:   false,
    }
}
```

### 7.4 `internal/boomerang/async_orchestrator.go`

**Propósito:** BoomerangAsyncOrchestrator con RunPhaseAsync().

**Contenido:** (ver secciones 5.4–5.6 arriba)

### 7.5 `internal/boomerang/state.go`

**Propósito:** Estado mutable de una fase (PhaseState).

```go
package boomerang

// PhaseState mantiene el estado mutable de una fase en ejecución
type PhaseState struct {
    mu sync.RWMutex

    Phase          string
    MemoryContext  string
    DAG            *TaskDAG
    DelegateResult *DelegateResult
    GitStatus      string
    SaveResult     *SaveResult
    Errors         []error
}

// NewPhaseState crea un nuevo estado de fase
func NewPhaseState(phase string) *PhaseState {
    return &PhaseState{
        Phase:  phase,
        Errors: make([]error, 0),
    }
}
```

### 7.6 `internal/boomerang/async_orchestrator_test.go`

**Propósito:** Tests para el nuevo orquestador async.

Contenido esperado: tests para `EventBus`, `PhaseStepMatrix`, `RunPhaseAsync()`, `delegateStepAsync()`.

### 7.7 `internal/scheduler/scheduler_events.go` (opcional)

**Propósito:** Eventos del scheduler a nivel superior, integrando BoomerangAsync.

---

## 8. Estrategia de Implementación

### 8.1 Orden de Implementación (7 fases)

| Fase | Prioridad | Depende de | Esfuerzo | Riesgo |
|------|-----------|------------|----------|--------|
| **F1: Phase Skip Matrix** | P0 | Ninguna | 1 día | Bajo |
| **F2: PhaseConfigV2** | P0 | F1 | 0.5 día | Bajo |
| **F3: DelegateStep Async** | P1 | F2 | 2 días | Medio |
| **F4: Event Bus + Event Loop** | P1 | F2, F3 | 3 días | Medio |
| **F5: Approval Gate Async** | P2 | F4 | 1 día | Medio |
| **F6: Scheduler Async** | P2 | F4, F5 | 2 días | Alto |
| **F7: Legacy Cleanup** | P3 | F1-F6 | 1 día | Bajo |

### 8.2 Fase 1: Phase Skip Matrix (P0)

**Archivos:**
- Crear: `internal/boomerang/skip.go`
- Modificar: `internal/boomerang/orchestrator.go`

**Test:**
```go
func TestDefaultPhaseMatrix(t *testing.T) {
    matrix := DefaultPhaseMatrix()

    // F0 no debe ejecutar Git ni Quality
    if matrix.ShouldRun("F0", StepGit) {
        t.Error("F0 should skip Git")
    }
    if matrix.ShouldRun("F0", StepQuality) {
        t.Error("F0 should skip Quality")
    }

    // F0 debe ejecutar Memory, Think, Delegate, Save
    if !matrix.ShouldRun("F0", StepMemory) {
        t.Error("F0 should run Memory")
    }

    // F3 debe ejecutar todos
    if !matrix.ShouldRun("F3", StepGit) {
        t.Error("F3 should run Git")
    }
    if !matrix.ShouldRun("F3", StepQuality) {
        t.Error("F3 should run Quality")
    }
}
```

### 8.3 Fase 2: PhaseConfigV2 (P0)

**Archivos:**
- Crear: `internal/boomerang/phase_config.go`
- Modificar: `internal/boomerang/orchestrator.go` (agregar `RunPhaseV2()`)

**Test:**
```go
func TestPhaseConfigV2(t *testing.T) {
    legacy := PhaseConfig{Phase: "F0", TaskDesc: "test"}
    v2 := legacy.ToV2()
    if v2.Phase != "F0" {
        t.Error("conversion failed")
    }
}
```

### 8.4 Fase 3: DelegateStep Async (P1)

**Archivos:**
- Modificar: `internal/boomerang/delegate.go` (agregar `DelegateStepAsync()`)

**Test:**
```go
func TestDelegateStepAsync(t *testing.T) {
    // Usar mock que simula subagentes
    o := NewBoomerangAsyncOrchestrator(&mockStore{}, mockBoundariLoader, nil)
    ctx := context.Background()

    dag := &TaskDAG{
        ParallelGroups: [][]int{{0, 1}},
        Tasks: []TaskSpec{
            {ID: 1, Name: "task1", Agent: "mock-agent"},
            {ID: 2, Name: "task2", Agent: "mock-agent"},
        },
    }

    events := make(chan StepEvent, 100)
    err := o.delegateStepAsync(ctx, PhaseConfigV2{Phase: "F0", Parallelism: 2}, events)
    if err != nil {
        t.Fatal(err)
    }
    // Verificar eventos de progreso
    close(events)
    for event := range events {
        t.Logf("Event: %v", event.Type)
    }
}
```

### 8.5 Fase 4: Event Bus + Event Loop (P1)

**Archivos:**
- Crear: `internal/boomerang/events.go`
- Crear: `internal/boomerang/async_orchestrator.go`
- Crear: `internal/boomerang/state.go`

**Test:**
```go
func TestEventBus(t *testing.T) {
    bus := NewEventBus(10)
    ch := bus.Subscribe(EventSubagentStarted, 5)

    bus.Publish(StepEvent{
        Type: EventSubagentStarted,
        Data: TaskSpec{Name: "test"},
    })

    select {
    case event := <-ch:
        if event.Type != EventSubagentStarted {
            t.Error("wrong event type")
        }
    case <-time.After(time.Second):
        t.Error("timeout waiting for event")
    }
}
```

### 8.6 Fase 5: Approval Gate Async (P2)

**Archivos:**
- Modificar: `internal/scheduler/approval.go`

### 8.7 Fase 6: Scheduler Async (P2)

**Archivos:**
- Modificar: `internal/scheduler/scheduler.go`
- Crear: `internal/scheduler/scheduler_events.go` (opcional)

### 8.8 Fase 7: Legacy Cleanup (P3)

**Archivos:**
- Marcar: `BoomerangOrchestrator.RunPhase()` como deprecated
- Tests: verificar que todo siga funcionando

### 8.9 Cómo Testear

| Nivel | Qué probar | Cómo |
|-------|------------|------|
| **Unitario** | `PhaseStepMatrix.ShouldRun()` | Tests directos por fase/step |
| **Unitario** | `EventBus.Publish()` / `Subscribe()` | Tests de publicación suscripción |
| **Unitario** | `PhaseConfig.ToV2()` | Conversión legacy → v2 |
| **Integración** | `delegateStepAsync()` | Mock de `exec.Command` + ver eventos |
| **Integración** | `RunPhaseAsync()` | Fase completa con DAG mock + ver eventos |
| **Integración** | Scheduler `RunAsync()` | Pipeline F0-F4 con approvals mock |
| **Regresión** | `RunPhase()` legacy | Asegurar que no se rompió |
| **Regresión** | Tests existentes en `boomerang_test.go` | `go test ./internal/boomerang/...` |

---

## 9. Backward Compatibility

### 9.1 Principios

1. **No romper API pública existente**
   - `BoomerangOrchestrator.RunPhase()` se mantiene idéntico
   - `PhaseConfig` se mantiene (se agrega `PhaseConfigV2` como alternativa)
   - `DelegateStep()` se mantiene (se agrega `DelegateStepAsync()` como alternativa)
   - `ApprovalGate()` se mantiene (se agrega `ApprovalGateAsync()`)
   - `Scheduler.Run()` se mantiene (se agrega `Scheduler.RunAsync()`)

2. **Todas las adiciones son opt-in**
   - `PhaseConfigV2` es un tipo nuevo, no modifica `PhaseConfig`
   - `RunPhaseAsync()` es un método nuevo, no modifica `RunPhase()`
   - Los cambios en `RunPhase()` son internos (usa skip matrix por defecto)

3. **Tests existentes deben seguir pasando**
   - `go test ./internal/boomerang/...` debe pasar sin cambios
   - `go test ./internal/scheduler/...` debe pasar sin cambios

### 9.2 Matriz de Compatibilidad

| API actual | Status | Alternativa nueva |
|------------|--------|-------------------|
| `PhaseConfig` | ✅ Mantiene | `PhaseConfigV2` para nuevos desarrollos |
| `RunPhase(ctx, PhaseConfig)` | ✅ Mantiene | `RunPhaseAsync(ctx, PhaseConfigV2)` |
| `DelegateStep(ctx, dag, phase)` | ✅ Mantiene | `DelegateStepAsync(ctx, dag, phase, events)` |
| `QualityStep(ctx, phase, dag, dr)` | ✅ Mantiene | `qualityStepAsync(ctx, config, events)` |
| `Scheduler.Run(ctx)` | ✅ Mantiene | `Scheduler.RunAsync(ctx)` |
| `ApprovalGate(phase, summary)` | ✅ Mantiene | `ApprovalGateAsync(phase, summary) chan` |

### 9.3 Migración

1. Todo código cliente existente (`cmd/`, tests, otros paquetes) **no requiere cambios**
2. El scheduler puede optar por `RunAsync()` cuando esté listo
3. Los phase runners pueden optar por async cuando convenga

---

## 10. Riesgos y Mitigaciones

| # | Riesgo | Probabilidad | Impacto | Mitigación |
|---|--------|-------------|---------|------------|
| 1 | **Goroutine leak** en delegateStepAsync si ctx se cancela | Media | Alto | Usar `errgroup.WithContext` que auto-cancela; `defer` para limpiar `activeTasks` |
| 2 | **Race condition** en `phaseState` compartido | Media | Alto | Proteger con `sync.Mutex`; o pasar estado por valor en eventos |
| 3 | **Event bus bloqueante** si consumidor es lento | Baja | Medio | Usar `select default` en Publish para drop de eventos no críticos |
| 4 | **Mala configuración de Skip Matrix** (ej: F4 sin Save) | Baja | Alto | Tests exhaustivos de la matriz por defecto; validación en `ActiveSteps()` |
| 5 | **cmd.Start() + pipe** puede dejar procesos huérfanos | Media | Alto | `cmd.Cancel` y `cmd.WaitDelay` (Go 1.20+); cleanup de procesos en `defer` |
| 6 | **DelegateStepAsync más complejo** que el original | Alta | Medio | Mantener `DelegateStep()` legacy para referencia; documentar bien el nuevo |
| 7 | **Scheduler.RunAsync()** cambia el contrato de retorno | Media | Medio | Retornar `<-chan Event` en lugar de `[]*Result`; mantener `Run()` legacy |
| 8 | **Dependencia de `errgroup`** (golang.org/x/sync) | Baja | Bajo | Ya es un paquete extremadamente estable del equipo de Go |

### 10.1 Plan de Mitigación Detallado

**Para riesgo 1 (goroutine leak):**
```go
// Patrón: context + errgroup + cleanup tracking
g, gCtx := errgroup.WithContext(ctx)
activeTasks := make(map[string]context.CancelFunc)
var mu sync.Mutex

// Cleanup asegurado
defer func() {
    mu.Lock()
    for name, cancel := range activeTasks {
        cancel()  // Asegurar cancelación de todo
        delete(activeTasks, name)
    }
    mu.Unlock()
}()

g.Go(func() error {
    taskCtx, taskCancel := context.WithCancel(gCtx)
    mu.Lock()
    activeTasks[task.Name] = taskCancel
    mu.Unlock()
    defer func() {
        mu.Lock()
        delete(activeTasks, task.Name)
        mu.Unlock()
        taskCancel()
    }()
    // ... ejecutar tarea ...
})
```

**Para riesgo 5 (procesos huérfanos):**
```go
cmd := exec.CommandContext(taskCtx, "opencode", args...)
cmd.Cancel = func() error {
    return cmd.Process.Signal(syscall.SIGINT)
}
cmd.WaitDelay = 30 * time.Second
```

**Para riesgo 4 (skip matrix incorrecta):**
```go
func ValidateMatrix(matrix PhaseStepMatrix) error {
    required := []string{"F0", "F1", "F2", "F3", "F4"}
    for _, phase := range required {
        steps := matrix[phase]
        if len(steps) == 0 {
            return fmt.Errorf("skip matrix: phase %s has no steps", phase)
        }
    }
    // F4 debe incluir Save
    if !matrix.ShouldRun("F4", StepSave) {
        return fmt.Errorf("skip matrix: F4 must include Save")
    }
    return nil
}
```

---

## 11. Diagrama General de la Solución

### 11.1 Diagrama de Paquetes

```
┌────────────────────────────────────────────────────────────┐
│                    internal/boomerang/                      │
│                                                             │
│  ┌──────────┐  ┌─────────┐  ┌─────────┐  ┌──────────┐     │
│  │ skip.go  │  │events.go│  │state.go │  │phase_    │     │
│  │ (matrix) │  │ (bus)   │  │(state)  │  │config.go │     │
│  └──────────┘  └─────────┘  └─────────┘  └──────────┘     │
│                                                             │
│  ┌────────────────────────────────────────────────────┐    │
│  │  async_orchestrator.go (BoomerangAsyncOrchestrator) │    │
│  │  - RunPhaseAsync()  → event loop                     │    │
│  │  - delegateStepAsync() → errgroup + cmd.Start()      │    │
│  │  - qualityStepAsync() → validación por fase          │    │
│  └────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌────────────────┐ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┐    │
│  │ orchestrator.go │ (sin cambios en API pública)       │    │
│  │ delegate.go     │ (se agregan funciones nuevas)      │    │
│  │ think.go        │ (sin cambios)                      │    │
│  │ memory.go       │ (sin cambios)                      │    │
│  │ git.go          │ (sin cambios)                      │    │
│  │ quality.go      │ (se agrega qualityStepAsync)       │    │
│  │ save.go         │ (sin cambios)                      │    │
│  └────────────────┘ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ┘    │
└────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌────────────────────────────────────────────────────────────┐
│                   internal/scheduler/                       │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐ │
│  │ scheduler.go │  │ approval.go  │  │ scheduler_events  │ │
│  │ + RunAsync() │  │ + approval   │  │ (opcional)        │ │
│  │              │  │   async      │  │                   │ │
│  └──────────────┘  └──────────────┘  └───────────────────┘ │
└────────────────────────────────────────────────────────────┘
```

### 11.2 Diagrama de Flujo de Eventos (secuencia)

```
Usuario         BoomerangAsync          Subagente 1        Subagente 2
  │                    │                     │                   │
  │  RunPhaseAsync()   │                     │                   │
  │───────────────────►│                     │                   │
  │                    │  EventPhaseStarted  │                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  StepMemory (rápido)│                   │
  │                    │  EventStepCompleted │                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  StepThink (rápido) │                   │
  │                    │  EventStepCompleted │                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  StepDelegate       │                   │
  │                    │  EventSubagentStarted                   │
  │                    │──(task1 started)───►│                   │
  │                    │  EventSubagentStarted                   │
  │                    │──(task2 started)───►│                   │
  │                    │                     │                   │
  │  ─── usuario escribe ───────────────────►│                   │
  │  mientras espera ──┘                     │                   │
  │                    │                     │                   │
  │  EventUserInput    │                     │                   │
  │◄───────────────────┤                     │                   │
  │                    │                     │                   │
  │                    │  EventSubagentOutput│                   │
  │                    │◄────────────────────┤                   │
  │                    │  EventSubagentOutput│                   │
  │                    │◄────────────────────┤                   │
  │                    │                     │                   │
  │                    │  (subagentes       │                   │
  │                    │   trabajan en       │                   │
  │                    │   paralelo)         │                   │
  │                    │                     │                   │
  │                    │  EventSubagentCompleted                │
  │                    │◄────────────────────┤                   │
  │                    │  EventSubagentCompleted                │
  │                    │◄────────────────────┤                   │
  │                    │                     │                   │
  │                    │  StepQuality (skipped en F0)           │
  │                    │  EventStepSkipped   │                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  StepSave           │                   │
  │                    │  EventStepCompleted │                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  EventPhaseCompleted│                   │
  │                    │──(evento a CLI)────►│                   │
  │                    │                     │                   │
  │                    │  EventApprovalReq   │                   │
  │                    │──(pide aprobación)─►│                   │
  │  Usuario aprueba   │                     │                   │
  │───────────────────►│                     │                   │
  │                    │  EventApprovalGranted                   │
  │                    │──(next phase)──────►│                   │
```

---

## Apéndice A: Comparativa de Implementaciones

| Aspecto | Actual (RunPhase) | Propuesto (RunPhaseAsync) |
|---------|------------------|--------------------------|
| **Pasos ejecutados** | Siempre 6 | Según skip matrix (3-6) |
| **Delegate** | `cmd.Output()` + `wg.Wait()` | `cmd.Start()` + `errgroup.Group` |
| **Streaming output** | ❌ No | ✅ Sí (línea por línea) |
| **Input de usuario** | ❌ No | ✅ Sí (userInput chan) |
| **Cancelación granular** | ❌ Solo ctx global | ✅ Por tarea (CancelFunc) |
| **Approval gate** | Bloqueante | No bloqueante (channel) |
| **Progreso** | ❌ No | ✅ Eventos en tiempo real |
| **Tiempo F0** | ~100% | ~60% (salta Git + Quality) |
| **Dependencias nuevas** | 0 | 0 (solo errgroup, ya golang.org/x/sync) |

## Apéndice B: Tiempos Estimados de Implementación

| Fase | Archivos | LOC estimados | Tiempo |
|------|----------|---------------|--------|
| F1: Skip Matrix | 1 nuevo + 1 mod | ~80 LOC | 1 día |
| F2: PhaseConfigV2 | 1 nuevo + 1 mod | ~50 LOC | 0.5 día |
| F3: Delegate Async | 1 mod (añadir) | ~150 LOC | 2 días |
| F4: Event Bus + Loop | 3 nuevos + 1 mod | ~250 LOC | 3 días |
| F5: Approval Async | 1 mod | ~50 LOC | 1 día |
| F6: Scheduler Async | 1 mod + (1 opcional) | ~120 LOC | 2 días |
| F7: Legacy Cleanup | 1 mod | ~30 LOC | 0.5 día |
| **Tests** | 2 nuevos + ampliar existentes | ~200 LOC | (incluido arriba) |
| **Total** | ~9 archivos | ~930 LOC | ~10 días |

---

*Fin del plan de optimización*

