# Investigación: Patrones de Orquestación Asíncrona para Sistemas Multi-Agente

> Fecha: 17 Junio 2026
> Contexto: ZyroAgentCLI — BoomerangOrchestrator + Scheduler SDD (F0-F4)
> Propósito: Diseñar un event loop asíncrono con phase-skip matrix y exec.Command no bloqueante

---

## Tabla de Contenidos

1. [Resumen Ejecutivo](#1-resumen-ejecutivo)
2. [Análisis del Código Actual](#2-análisis-del-código-actual)
3. [Patrón: Goroutines + Channels para Subprocesos Asíncronos](#3-patrón-goroutines--channels-para-subprocesos-asíncronos)
4. [Patrón: Event Loop / Reactor en Go](#4-patrón-event-loop--reactor-en-go)
5. [Patrón: exec.Command No-Bloqueante con Concurrencia](#5-patrón-execcommand-no-bloqueante-con-concurrencia)
6. [Patrón: Phase Skip Matrix](#6-patrón-phase-skip-matrix)
7. [Orquestadores Multi-Agente: LangGraph](#7-orquestadores-multi-agente-langgraph)
8. [Orquestadores Multi-Agente: CrewAI](#8-orquestadores-multi-agente-crewai)
9. [Orquestadores Multi-Agente: AutoGen (Microsoft)](#9-orquestadores-multi-agente-autogen-microsoft)
10. [Comparativa y Lecciones para ZyroAgentCLI](#10-comparativa-y-lecciones-para-zyroagentcli)
11. [Propuesta de Arquitectura para BoomerangAsync](#11-propuesta-de-arquitectura-para-boomerangasync)
12. [Referencias y URLs](#12-referencias-y-urls)

---

## 1. Resumen Ejecutivo

### Problema Identificado

El `BoomerangOrchestrator` ejecuta un ciclo de 6 pasos **sincrónico y monolítico**:

```
Memory → Think → Delegate → Git → Quality → Save
```

Tres problemas críticos:
1. **Sin phase-skip**: TODOS los 6 pasos en TODAS las fases (F0-F4)
2. **exec.Command bloqueante**: `DelegateStep` lanza `opencode subagent` y espera. Mientras tanto, el orquestador no puede recibir mensajes del usuario ni responder.
3. **Sin event loop**: No hay canal para notificaciones de progreso, cancelación, o interleaving de input del usuario.

### Solución Propuesta (BoomerangAsync)

Arquitectura basada en 3 patrones fundamentales:

| Patrón | Propósito | Implementación en Go |
|--------|-----------|---------------------|
| **Event Loop + Channel Bus** | Loop principal que recibe eventos de subprocesos, input del usuario, y timeouts | `select` sobre múltiples channels |
| **Goroutine Pool** | Ejecutar `exec.Command` sin bloquear el event loop | `go func()` + `cmd.Start()` + channel de resultado |
| **Phase Skip Matrix** | Determinar qué pasos ejecutar según la fase | Tabla declarativa `[Phase][Step]bool` |

### Beneficios Esperados

- **Responsividad**: El orquestador puede recibir input del usuario mientras los subagentes trabajan
- **Progreso visible**: El usuario ve qué subagentes están activos y su estado
- **Cancelación limpia**: Se puede cancelar una delegación en curso sin matar el proceso
- **Eficiencia**: Fases que no necesitan Git o Quality no pierden tiempo en esos pasos
- **Paralelismo real**: Múltiples subagentes en paralelo con fan-in de resultados

---

## 2. Análisis del Código Actual

### 2.1 BoomerangOrchestrator.RunPhase() — El Problema Central

**Archivo**: `internal/boomerang/orchestrator.go` (líneas 111-194)

```go
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
    // Paso 1: MEMORY — siempre se ejecuta
    memoryCtx, err := o.MemoryStep(ctx, ...)

    // Paso 2: THINK — siempre se ejecuta
    dag, err := o.ThinkStep(ctx, ...)

    // Paso 3: DELEGATE — siempre se ejecuta, BLOQUEANTE
    delegateResult, err := o.DelegateStep(ctx, dag, ...)

    // Paso 4: GIT — siempre se ejecuta
    gitStatus, err := o.GitStep(ctx)

    // Paso 5: QUALITY — siempre se ejecuta (con loop de retry)
    for i := 0; i < o.maxIterations; i++ {
        qualityOK, err := o.QualityStep(ctx, ...)
    }

    // Paso 6: SAVE — siempre se ejecuta
    saveResult, err := o.SaveStep(ctx, ...)
}
```

**Problema**: No hay `switch` por fase, no hay skip, no hay concurrencia.

### 2.2 DelegateStep() — Bloqueante con Falso Paralelismo

**Archivo**: `internal/boomerang/delegate.go` (líneas 13-67)

```go
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string) (*DelegateResult, error) {
    for _, group := range dag.ParallelGroups {
        var wg sync.WaitGroup
        for _, taskIdx := range group {
            wg.Add(1)
            go func(t TaskSpec) {
                defer wg.Done()
                cmd := exec.CommandContext(ctx, "opencode", "subagent", t.Agent, ...)
                output, err := cmd.Output()  // ¡BLOQUEA la goroutine hasta que termina!
            }(task)
        }
        wg.Wait()  // ¡BLOQUEA hasta que TODAS las tareas del grupo terminan!
    }
}
```

**Problema**: Aunque usa goroutines, el `wg.Wait()` bloquea al orquestador. No hay manera de:
- Recibir notificaciones de progreso
- Interleaving de input del usuario
- Cancelar una tarea específica
- Hacer streaming de output parcial

### 2.3 PhaseConfig — Sin Skip Configuration

```go
type PhaseConfig struct {
    Phase       string
    TaskDesc    string
    ProjectID   string
    MemoryLimit int
    Iterations  int
    Timeout     time.Duration
    // ❌ No hay: SkipSteps, AsyncMode, EventChan, ProgressChan
}
```

---

## 3. Patrón: Goroutines + Channels para Subprocesos Asíncronos

### 3.1 El Patrón Pipeline (Go Blog, Sameer Ajmani, 2014)

**URL**: https://go.dev/blog/pipelines

**Concepto**: Un pipeline es una serie de *stages* conectados por channels. Cada stage es un grupo de goroutines ejecutando la misma función.

**Etapas**:
1. **Source/Producer**: genera valores y los envía por un channel
2. **Middle stages**: reciben, transforman, y reenvían
3. **Sink/Consumer**: recibe y consume los resultados finales

**Fan-Out, Fan-In**:
- *Fan-Out*: múltiples goroutines leen del mismo channel (distribución de trabajo)
- *Fan-In*: múltiples goroutines escriben al mismo channel, multiplexado con `select`

**Cancelación explícita**:
```go
done := make(chan struct{})
defer close(done)  // Broadcast a todos los stages

// Cada stage usa select para escuchar done:
select {
case out <- result:
case <-done:
    return  // Cancelación limpia
}
```

### 3.2 Cómo aplica a ZyroAgentCLI

**DelegateStep con fan-out**:
```go
func (o *BoomerangOrchestrator) DelegateStepAsync(ctx context.Context, dag *TaskDAG) <-chan TaskProgress {
    progress := make(chan TaskProgress)
    
    for _, task := range dag.Tasks {
        go func(t TaskSpec) {
            cmd := exec.CommandContext(ctx, "opencode", "subagent", t.Agent)
            // Enviar progreso
            progress <- TaskProgress{Task: t.Name, Status: "started"}
            output, err := cmd.Output()
            progress <- TaskProgress{Task: t.Name, Status: "done", Output: output, Err: err}
        }(task)
    }
    
    return progress  // El orquestador itera sobre este channel
}
```

**Fan-In en el orquestador**:
```go
func (o *BoomerangAsyncOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) {
    progress := o.DelegateStepAsync(ctx, dag)
    
    for p := range progress {
        o.respondToUser(p)  // No bloqueante
        select {
        case userInput := <-o.userInputChan:
            o.handleUserInput(userInput)
        default:
            // Continuar
        }
    }
}
```

---

## 4. Patrón: Event Loop / Reactor en Go

### 4.1 El Patrón Select como Event Loop

**URL**: https://go.dev/talks/2012/concurrency.slide (Rob Pike, Google I/O 2012)

**Concepto**: El `select` en Go es inherentemente un event loop. No necesita librerías externas.

```go
for {
    select {
    case msg := <-inboundChan:
        handleMessage(msg)
    case progress := <-progressChan:
        updateUI(progress)
    case <-done:
        return
    case <-time.After(5 * time.Second):
        heartbeat()
    default:
        // No hay eventos, podemos hacer work-stealing o idle
    }
}
```

### 4.2 Variante: Reactor con Channel Bus

**Concepto**: Un channel central `EventBus` por donde pasan todos los eventos del sistema. Los handlers se registran para tipos de eventos específicos.

```go
type Event struct {
    Type    EventType
    Payload interface{}
}

type EventBus chan Event

// El event loop central
func (o *Orchestrator) EventLoop(ctx context.Context) {
    for {
        select {
        case event := <-o.bus:
            o.dispatch(event)
        case input := <-o.userInput:
            o.bus <- Event{Type: EventUserInput, Payload: input}
        case <-ctx.Done():
            return
        }
    }
}

func (o *Orchestrator) dispatch(event Event) {
    switch event.Type {
    case EventTaskStarted:
        o.onTaskStarted(event.Payload.(TaskSpec))
    case EventTaskCompleted:
        o.onTaskCompleted(event.Payload.(TaskResult))
    case EventUserInput:
        o.onUserInput(event.Payload.(string))
    case EventPhaseComplete:
        o.onPhaseComplete(event.Payload.(PhaseResult))
    }
}
```

### 4.3 Cómo aplica a ZyroAgentCLI

El `BoomerangAsyncOrchestrator` tendría:

1. **Un event loop central** que reemplaza a `RunPhase()`
2. **Múltiples channels**: uno por step (memoryChan, thinkChan, delegateChan, etc.)
3. **Un channel de input del usuario** para interleaving
4. **Un channel de progreso** para notificar al frontend/CLI

---

## 5. Patrón: exec.Command No-Bloqueante con Concurrencia

### 5.1 El Problema de cmd.Output()

`cmd.Output()` es bloqueante porque espera a que el proceso termine y capture todo el stdout. Lo mismo para `cmd.Run()`.

### 5.2 Solución: cmd.Start() + Channels

**Referencia**: Go stdlib `os/exec` package

```go
func runCommandAsync(ctx context.Context, name string, args ...string) (<-chan string, <-chan error) {
    output := make(chan string, 100)  // Buffered para no bloquear
    errs := make(chan error, 1)
    
    cmd := exec.CommandContext(ctx, name, args...)
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()
    
    go func() {
        defer close(output)
        defer close(errs)
        
        if err := cmd.Start(); err != nil {
            errs <- err
            return
        }
        
        // Streaming de output línea por línea
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            select {
            case output <- scanner.Text():
            case <-ctx.Done():
                cmd.Process.Kill()
                return
            }
        }
        
        if err := cmd.Wait(); err != nil {
            errs <- err
        }
    }()
    
    return output, errs
}
```

### 5.3 Patrón: Subprocess Manager con Pool

```go
type SubprocessManager struct {
    active     map[string]context.CancelFunc
    mu         sync.RWMutex
    maxWorkers int
}

func (sm *SubprocessManager) StartTask(ctx context.Context, task TaskSpec, progress chan<- TaskProgress) {
    if sm.countActive() >= sm.maxWorkers {
        progress <- TaskProgress{Task: task.Name, Status: "queued"}
        return  // Se queda en cola
    }
    
    taskCtx, cancel := context.WithCancel(ctx)
    sm.mu.Lock()
    sm.active[task.Name] = cancel
    sm.mu.Unlock()
    
    go func() {
        defer func() {
            sm.mu.Lock()
            delete(sm.active, task.Name)
            sm.mu.Unlock()
        }()
        
        progress <- TaskProgress{Task: task.Name, Status: "running"}
        cmd := exec.CommandContext(taskCtx, "opencode", "subagent", task.Agent)
        output, err := cmd.Output()
        
        if err != nil {
            progress <- TaskProgress{Task: task.Name, Status: "failed", Error: err}
        } else {
            progress <- TaskProgress{Task: task.Name, Status: "done", Output: string(output)}
        }
    }()
}

func (sm *SubprocessManager) CancelTask(name string) bool {
    sm.mu.RLock()
    cancel, ok := sm.active[name]
    sm.mu.RUnlock()
    if ok {
        cancel()
        return true
    }
    return false
}
```

### 5.4 Cómo aplica a ZyroAgentCLI

El `DelegateStep` actual:

```go
// Antes (bloqueante):
cmd := exec.CommandContext(ctx, "opencode", "subagent", t.Agent, ...)
output, err := cmd.Output()

// Después (asíncrono con streaming):
delegateInstance := NewDelegateManager(maxConcurrent)
progress := delegateInstance.RunTasks(ctx, dag)
// El event loop puede procesar progreso mientras las tareas corren
```

---

## 6. Patrón: Phase Skip Matrix

### 6.1 Concepto

Una matriz declarativa `[Phase][Step]bool` que determina qué pasos ejecutar en cada fase. Esencialmente una tabla de verdad:

| Fase | Memory | Think | Delegate | Git | Quality | Save |
|------|--------|-------|----------|-----|---------|------|
| F0   | ✅     | ✅    | ✅       | ❌  | ❌      | ✅   |
| F1   | ✅     | ✅    | ✅       | ❌  | ❌      | ✅   |
| F2   | ✅     | ✅    | ✅       | ❌  | ❌      | ✅   |
| F3   | ✅     | ✅    | ✅       | ✅  | ✅      | ✅   |
| F4   | ❌     | ✅    | ✅       | ✅  | ❌      | ✅   |

### 6.2 Implementación

```go
// PhaseStep determina qué step ejecutar en cada fase
type Step int

const (
    StepMemory   Step = iota
    StepThink
    StepDelegate
    StepGit
    StepQuality
    StepSave
)

// PhaseSkipMatrix define qué steps ejecutar por fase
type PhaseSkipMatrix map[string][]Step

var defaultSkipMatrix = PhaseSkipMatrix{
    "F0": {StepMemory, StepThink, StepDelegate, StepSave},
    "F1": {StepMemory, StepThink, StepDelegate, StepSave},
    "F2": {StepMemory, StepThink, StepDelegate, StepSave},
    "F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
    "F4": {StepThink, StepDelegate, StepGit, StepSave},
}

func (m PhaseSkipMatrix) ShouldRun(phase string, step Step) bool {
    for _, s := range m[phase] {
        if s == step {
            return true
        }
    }
    return false
}
```

### 6.3 Variante: Configurable por Fase (extensible)

```go
type PhaseConfigV2 struct {
    Phase       string
    Steps       []Step           // Steps a ejecutar (si nil, todos)
    AsyncMode   bool             // Modo asíncrono (event loop) vs síncrono
    Timeout     time.Duration
    Parallelism int              // Subagentes concurrentes máximos
}
```

### 6.4 Cómo aplica a ZyroAgentCLI

**Análisis de cada fase**:

| Fase | Característica | Steps necesarios | Steps a saltar |
|------|---------------|------------------|----------------|
| **F0** | Investigación (3 subagentes paralelos) | Memory, Think, Delegate, Save | Git (no hay código), Quality (no hay código que validar) |
| **F1** | Especificación técnica | Memory, Think, Delegate, Save | Git, Quality |
| **F2** | Diseño técnico + tareas | Memory, Think, Delegate, Save | Git, Quality |
| **F3** | Implementación + verificación | Memory, Think, Delegate, Git, Quality, Save | Ninguno |
| **F4** | Archivo y cierre | Think, Delegate, Git, Save | Memory (no necesita contexto previo), Quality |

**Ahorro estimado**: ~40% de tiempo en F0 saltando Git y Quality.

---

## 7. Orquestadores Multi-Agente: LangGraph

### 7.1 Arquitectura

**URL**: https://langchain-ai.github.io/langgraph/

LangGraph es un framework de orquestación de agentes basado en **grafos de estados** (StateGraph). A diferencia de pipelines lineales, LangGraph modela el flujo como un grafo dirigido con:

- **Nodos**: funciones que ejecutan lógica (un agente, una herramienta, un paso)
- **Aristas**: transiciones condicionales o no condicionales entre nodos
- **Estado**: objeto compartido que se pasa entre nodos (State)

### 7.2 Patrones Clave de LangGraph

#### StateGraph (Máquina de Estados)
```python
from langgraph.graph import StateGraph

graph = StateGraph(AgentState)

graph.add_node("agent", call_agent)
graph.add_node("tools", call_tool)

graph.add_edge("agent", "tools")
graph.add_conditional_edges("tools", router, {
    "continue": "agent",
    "end": END
})
```

#### Flujo Controlado por Eventos
LangGraph usa un event loop interno (similar a Reactor) que:
1. Ejecuta un nodo
2. Recibe el nuevo estado
3. Evalúa las aristas del nodo actual
4. Decide el siguiente nodo
5. Repite hasta llegar a END

#### Checkpointing y Persistencia
LangGraph guarda checkpoints después de cada paso del grafo. Esto permite:
- **Resumir** ejecuciones interrumpidas
- **Human-in-the-loop**: pausar en puntos específicos para aprobación humana
- **Time travel**: volver a estados anteriores

### 7.3 Lecciones para ZyroAgentCLI

1. **Grafo de estados > Pipeline lineal**: El BoomerangOrchestrator debería evolucionar de un pipeline fijo a un grafo de estados configurable.
2. **Human-in-the-loop con pause/resume**: LangGraph interrumpe el grafo en nodos específicos (por ejemplo, antes de ejecutar una herramienta peligrosa). ZyroCLI podría pausar entre fases SDD.
3. **Checkpointing**: Guardar el estado después de cada paso Boomerang permitiría reanudar una fase si el proceso muere.
4. **Ruteo condicional**: En vez de ejecutar Quality solo al final, se podría rutear condicionalmente a "reparación" si falla.

---

## 8. Orquestadores Multi-Agente: CrewAI

### 8.1 Arquitectura

**URL**: https://docs.crewai.com/core-concepts/Flow/

CrewAI organiza agentes en **Crews** (equipos) con **Processos** de ejecución:

- **Sequential**: cada agente ejecuta su tarea en orden
- **Hierarchical**: un agente manager asigna tareas y revisa resultados
- **Hybrid**: combinación de ambos

### 8.2 Patrón Flow (Event-Driven)

CrewAI introdujo el concepto de **Flows** como una alternativa a los procesos lineales:

```python
from crewai.flow.flow import Flow, listen, start, router

class MyFlow(Flow):
    @start()
    def begin(self):
        # Estado inicial
        self.state.counter = 0

    @listen(begin)
    def process(self):
        # Escucha el evento de begin()
        self.state.counter += 1

    @router(process)
    def decide(self):
        if self.state.counter < 3:
            return "process"  # Loop back
        return "end"
```

### 8.3 Patrones Clave de CrewAI

- **`@start`, `@listen`, `@router`**: Decoradores que definen el grafo de eventos
- **Estado compartido**: `self.state` es un objeto Pydantic que fluye entre steps
- **Paralelismo automático**: Tareas independientes se ejecutan en paralelo
- **Callback de progreso**: CrewAI emite eventos de progreso que se pueden escuchar

### 8.4 Lecciones para ZyroAgentCLI

1. **Decoradores como DSL**: El patrón `@listen` de CrewAI sugiere que ZyroCLI podría tener un DSL declarativo para definir pipelines.
2. **Estado tipado**: CrewAI usa Pydantic para el state. En Go podríamos usar structs con validación.
3. **Router condicional**: El `@router` de CrewAI es equivalente a las aristas condicionales de LangGraph. ZyroCLI podría tener QualityStep como router que decide si redelegar o continuar.

---

## 9. Orquestadores Multi-Agente: AutoGen (Microsoft)

### 9.1 Arquitectura

**URL**: https://microsoft.github.io/autogen/

AutoGen (v0.4+) fue reestructurado con una arquitectura basada en **eventos**:

- **Agent**: unidad básica que puede enviar y recibir mensajes
- **Messaging**: los agentes se comunican mediante mensajes asíncronos
- **Orchestration**: manejo de contexto, turnos, y control de flujo
- **GroupChat**: múltiples agentes conversan bajo un manager que orquesta turnos

### 9.2 Patrón GroupChat con Round-Robin Manager

AutoGen popularizó el patrón **GroupChatManager** donde:
1. Un agente habla → emite un mensaje
2. El manager decide quién habla después
3. El siguiente agente recibe el contexto completo
4. Repite hasta que se alcanza un criterio de terminación

### 9.3 Lecciones para ZyroAgentCLI

1. **Mensajería asíncrona**: La comunicación entre agentes debe ser basada en mensajes, no en llamadas bloqueantes.
2. **Manager como scheduler**: El GroupChatManager de AutoGen es conceptualmente similar al Scheduler de ZyroCLI — ambos controlan qué agente ejecuta y cuándo.
3. **Contexto compartido**: El mensaje que pasa entre agentes incluye el historial completo. En ZyroCLI, esto sería el `memoryCtx` que fluye entre steps.

---

## 10. Comparativa y Lecciones para ZyroAgentCLI

### 10.1 Tabla Comparativa

| Aspecto | LangGraph | CrewAI | AutoGen | ZyroCLI (actual) |
|---------|-----------|--------|---------|------------------|
| **Modelo** | Grafo de estados | Pipeline + Flow | Mensajería asíncrona | Pipeline lineal |
| **Paralelismo** | Nodos paralelos | Tareas independientes | Agentes concurrentes | Solo en DelegateStep |
| **Event Loop** | Interno (StateGraph) | Event-driven (Flows) | Messaging bus | ❌ Ninguno |
| **HITL** | Checkpoints + interrupt | Human input tool | GroupChat manager | Approval gates entre fases |
| **Estado** | State object (tipado) | State Pydantic | Messages list | PhaseResult struct |
| **Persistencia** | Checkpointing | No nativo | No nativo | Engram (solo facts) |
| **Skip condicional** | Aristas condicionales | Router decorator | Termination conditions | ❌ No existe |

### 10.2 Lecciones Específicas para ZyroAgentCLI

| Lección | De | Aplicación |
|---------|----|-----------|
| **Grafo > Pipeline** | LangGraph | Reemplazar RunPhase() secuencial por un grafo de steps |
| **Event Loop** | Todos | Select sobre múltiples channels para eventos asíncronos |
| **Skip condicional** | LangGraph/CrewAI | Phase Skip Matrix + aristas condicionales |
| **Human-in-the-loop** | LangGraph | Pausar el event loop cuando se necesita aprobación |
| **Estado tipado** | CrewAI | PhaseState como struct con validación |
| **Checkpointing** | LangGraph | Guardar estado después de cada step para reanudar |
| **Parallel fan-out** | Go Pipeline | DelegateStep con goroutine pool + channel de progreso |
| **Broadcast cancelación** | Go Pipeline | `close(done)` para cancelar todos los subprocesos |

---

## 11. Propuesta de Arquitectura para BoomerangAsync

### 11.1 Nuevos Tipos

```go
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

// StepStatus representa el estado de un step
type StepStatus string
const (
    StepPending   StepStatus = "pending"
    StepRunning   StepStatus = "running"
    StepDone      StepStatus = "done"
    StepSkipped   StepStatus = "skipped"
    StepFailed    StepStatus = "failed"
)

// StepEvent es un evento emitido por un step
type StepEvent struct {
    Step   Step
    Status StepStatus
    Data   interface{} // Datos contextuales del step
    Error  error
}

// PhaseSkipMatrix define qué steps ejecutar por fase
type PhaseSkipMatrix map[string][]Step

// PhaseConfigV2 configura una fase con soporte asíncrono
type PhaseConfigV2 struct {
    Phase        string
    TaskDesc     string
    ProjectID    string
    Steps        []Step           // nil = usar skip matrix
    Timeout      time.Duration
    Parallelism  int              // Subagentes concurrentes
    AsyncMode    bool             // true = event loop, false = síncrono legacy
    SkipMatrix   PhaseSkipMatrix
}
```

### 11.2 Event Loop Asíncrono

```go
type BoomerangAsyncOrchestrator struct {
    // Core
    bus          chan StepEvent      // Event bus central
    userInput    chan string         // Input del usuario
    done         chan struct{}       // Señal de cancelación global
    
    // Dependencias
    memoryStore  memory.EngramStore
    skipMatrix   PhaseSkipMatrix
    
    // Estado
    phaseState   *PhaseState         // Estado mutable de la fase actual
    activeTasks  map[string]context.CancelFunc
    mu           sync.RWMutex
}

func (o *BoomerangAsyncOrchestrator) RunPhaseAsync(ctx context.Context, config PhaseConfigV2) <-chan StepEvent {
    events := make(chan StepEvent, 100)
    
    go func() {
        defer close(events)
        
        phaseCtx, cancel := context.WithCancel(ctx)
        defer cancel()
        
        // Determinar qué steps ejecutar
        steps := config.Steps
        if steps == nil {
            steps = config.SkipMatrix[config.Phase]
        }
        
        for _, step := range steps {
            select {
            case <-phaseCtx.Done():
                events <- StepEvent{Step: step, Status: StepFailed, Error: phaseCtx.Err()}
                return
            default:
            }
            
            // Emitir evento de inicio
            events <- StepEvent{Step: step, Status: StepRunning}
            
            // Ejecutar step (podría ser bloqueante o lanzar goroutine)
            err := o.executeStep(phaseCtx, step, config, events)
            
            if err != nil {
                events <- StepEvent{Step: step, Status: StepFailed, Error: err}
                // ¿Continuar o abortar? Depende del step
                if step == StepDelegate || step == StepQuality {
                    return // Steps críticos: abortar
                }
            } else {
                events <- StepEvent{Step: step, Status: StepDone}
            }
        }
    }()
    
    return events
}

// executeStep delega a la función correspondiente
func (o *BoomerangAsyncOrchestrator) executeStep(ctx context.Context, step Step, config PhaseConfigV2, events chan<- StepEvent) error {
    switch step {
    case StepMemory:
        return o.memoryStepAsync(ctx, config)
    case StepThink:
        return o.thinkStepAsync(ctx, config)
    case StepDelegate:
        return o.delegateStepAsync(ctx, config, events) // events para progreso intermedio
    case StepGit:
        return o.gitStepAsync(ctx)
    case StepQuality:
        return o.qualityStepAsync(ctx, config, events)
    case StepSave:
        return o.saveStepAsync(ctx, config)
    }
    return nil
}
```

### 11.3 DelegateStep No-Bloqueante

```go
func (o *BoomerangAsyncOrchestrator) delegateStepAsync(ctx context.Context, config PhaseConfigV2, events chan<- StepEvent) error {
    dag := o.phaseState.DAG
    
    // Semáforo para limitar concurrencia
    sem := make(chan struct{}, config.Parallelism)
    var wg sync.WaitGroup
    var mu sync.Mutex
    errors := make([]error, 0)
    
    for _, group := range dag.ParallelGroups {
        for _, taskIdx := range group {
            if taskIdx >= len(dag.Tasks) {
                continue
            }
            task := dag.Tasks[taskIdx]
            
            select {
            case sem <- struct{}{}: // Adquirir slot
            case <-ctx.Done():
                return ctx.Err()
            }
            
            wg.Add(1)
            go func(t TaskSpec) {
                defer wg.Done()
                defer func() { <-sem }() // Liberar slot
                
                stepCtx, cancel := context.WithCancel(ctx)
                defer cancel()
                
                // Registrar tarea activa (para cancelación)
                o.mu.Lock()
                o.activeTasks[t.Name] = cancel
                o.mu.Unlock()
                defer func() {
                    o.mu.Lock()
                    delete(o.activeTasks, t.Name)
                    o.mu.Unlock()
                }()
                
                // Notificar inicio
                events <- StepEvent{Step: StepDelegate, Status: StepRunning, Data: t}
                
                // Lanzar subproceso
                cmd := exec.CommandContext(stepCtx, "opencode", "subagent", t.Agent,
                    "--param", fmt.Sprintf("task=%s", t.Name),
                    "--param", fmt.Sprintf("phase=%s", config.Phase),
                )
                
                // Streaming de output (opcional)
                stdout, _ := cmd.StdoutPipe()
                cmd.Stderr = cmd.Stdout  // Merge stderr → stdout
                
                if err := cmd.Start(); err != nil {
                    events <- StepEvent{Step: StepDelegate, Status: StepFailed, Error: err, Data: t}
                    mu.Lock()
                    errors = append(errors, err)
                    mu.Unlock()
                    return
                }
                
                // Leer output línea por línea (no bloquea otras tareas)
                scanner := bufio.NewScanner(stdout)
                for scanner.Scan() {
                    line := scanner.Text()
                    events <- StepEvent{
                        Step: StepDelegate,
                        Status: StepRunning,
                        Data: StepOutput{Task: t.Name, Line: line},
                    }
                }
                
                if err := cmd.Wait(); err != nil {
                    events <- StepEvent{Step: StepDelegate, Status: StepFailed, Error: err, Data: t}
                    mu.Lock()
                    errors = append(errors, err)
                    mu.Unlock()
                    return
                }
                
                events <- StepEvent{Step: StepDelegate, Status: StepDone, Data: t}
            }(task)
        }
        
        // Esperar grupo paralelo sin bloquear el event loop
        done := make(chan struct{})
        go func() {
            wg.Wait()
            close(done)
        }()
        
        select {
        case <-done:
        case <-ctx.Done():
            return ctx.Err()
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("delegate: %d tasks failed", len(errors))
    }
    return nil
}
```

### 11.4 Integración con el Scheduler Existente

El `Scheduler` actual en `internal/scheduler/scheduler.go` se modificaría para:

```go
func (s *Scheduler) RunAsync(ctx context.Context) (<-chan scheduler.Event, error) {
    events := make(chan scheduler.Event, 100)
    
    go func() {
        defer close(events)
        
        for _, phase := range s.phases {
            phaseConfig := boomerang.PhaseConfigV2{
                Phase:      string(phase.Name()),
                TaskDesc:   s.config.Module,
                AsyncMode:  true,
                SkipMatrix: boomerang.DefaultSkipMatrix(),
                Parallelism: 3,
            }
            
            phaseEvents := s.config.Boomerang.RunPhaseAsync(ctx, phaseConfig)
            for event := range phaseEvents {
                events <- scheduler.Event{
                    Type:  scheduler.EventPhaseProgress,
                    Phase: phase.Name(),
                    Data:  event,
                }
            }
            
            // Approval gate (no bloqueante si AsyncMode)
            if s.config.Boomerang.IsAsyncMode() {
                events <- scheduler.Event{
                    Type: scheduler.EventNeedsApproval,
                    Phase: phase.Name(),
                }
                // Esperar aprobación sin bloquear
                select {
                case approval := <-s.approvalChan:
                    if !approval.Approved {
                        events <- scheduler.Event{
                            Type: scheduler.EventPhaseAborted,
                            Phase: phase.Name(),
                        }
                        return
                    }
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return events, nil
}
```

### 11.5 Diagrama de Arquitectura Propuesta

```
┌─────────────────────────────────────────────────────────┐
│                    User Input                           │
│                    (stdin / chat)                       │
└─────────┬───────────────────────────────────────────────┘
          │ userInputChan
          ▼
┌─────────────────────────────────────────────────────────┐
│              BoomerangAsyncOrchestrator                 │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Event Loop (select)                  │   │
│  │                                                   │   │
│  │  ┌──────┐  ┌──────┐  ┌──────────┐  ┌──────────┐ │   │
│  │  │Memory│→│Think │→│Delegate │→│Git     │ │   │
│  │  │Step  │  │Step  │  │Step     │  │Step    │ │   │
│  │  └──────┘  └──────┘  └──────────┘  └──────────┘ │   │
│  │                               │                  │   │
│  │                    ┌──────────┘                  │   │
│  │                    ▼                             │   │
│  │              ┌──────────┐  ┌──────┐              │   │
│  │              │Quality   │→│Save  │              │   │
│  │              │Step      │  │Step  │              │   │
│  │              └──────────┘  └──────┘              │   │
│  └──────────────────────────────────────────────────┘   │
│                                                         │
│  Phase Skip Matrix:                                      │
│  ┌─────┬────┬─────┬─────┬───┬───────┬────┐             │
│  │Phase│Mem │Think│Del  │Git│Quality│Save│             │
│  ├─────┼────┼─────┼─────┼───┼───────┼────┤             │
│  │F0   │ ✅ │ ✅  │ ✅  │ ❌│ ❌    │ ✅ │             │
│  │F3   │ ✅ │ ✅  │ ✅  │ ✅│ ✅    │ ✅ │             │
│  └─────┴────┴─────┴─────┴───┴───────┴────┘             │
└─────────────────────────────────────────────────────────┘
          │ events (StepEvent channel)
          ▼
┌─────────────────────────────────────────────────────────┐
│              CLI / TUI / OpenCode Bridge                │
│  - Muestra progreso en tiempo real                      │
│  - Recibe input del usuario (cancelar, priorizar)       │
│  - Notifica cuando se necesita aprobación               │
└─────────────────────────────────────────────────────────┘
```

### 11.6 Plan de Migración (Fases)

| Fase | Cambio | Impacto |
|------|--------|---------|
| **1. Phase Skip Matrix** | Agregar skip matrix + `ShouldRun()` | No rompe API existente |
| **2. PhaseConfigV2** | Agregar `Steps []Step` opcional | Compatible hacia atrás |
| **3. DelegateStep Async** | Versión paralela con channel de progreso | Nueva función, legacy sigue |
| **4. Event Loop** | Nuevo `RunPhaseAsync()` | Conviviente con `RunPhase()` |
| **5. Scheduler Async** | `RunAsync()` en scheduler | Opcional, no obligatorio |
| **6. Legacy cleanup** | Deprecar `RunPhase()` síncrono | Cuando todo funcione en async |

---

## 12. Referencias y URLs

### Concurrencia en Go

| Recurso | URL |
|---------|-----|
| Go Concurrency Patterns: Pipelines & Cancellation | https://go.dev/blog/pipelines |
| Go Concurrency Patterns: Context | https://go.dev/blog/context |
| Go Concurrency Patterns (Rob Pike, Google I/O 2012) | https://go.dev/talks/2012/concurrency.slide |
| Advanced Go Concurrency Patterns | https://go.dev/blog/advanced-go-concurrency-patterns |
| Concurrency in Go (Katherine Cox-Buday, O'Reilly) | https://www.oreilly.com/library/view/concurrency-in-go/9781491941294/ |
| Go Concurrency Patterns (video) | https://www.youtube.com/watch?v=f6kdp27TYZs |
| Concurrency is not Parallelism | https://go.dev/s/concurrency-is-not-parallelism |

### Orquestación Multi-Agente

| Recurso | URL |
|---------|-----|
| LangGraph — StateGraph Architecture | https://langchain-ai.github.io/langgraph/ |
| LangGraph — Low Level Concepts | https://langchain-ai.github.io/langgraph/concepts/low_level/ |
| LangGraph — High Level Concepts | https://langchain-ai.github.io/langgraph/concepts/high_level/ |
| CrewAI — Flows (Event-Driven) | https://docs.crewai.com/core-concepts/Flow/ |
| CrewAI — Architecture | https://docs.crewai.com/concepts/ |
| AutoGen (Microsoft) | https://microsoft.github.io/autogen/ |
| AutoGen — Core Architecture | https://microsoft.github.io/autogen/stable/user-guide/core-user-guide/architecture.html |

### Código Base de ZyroAgentCLI (Analizado)

| Archivo | Propósito |
|---------|-----------|
| `internal/boomerang/orchestrator.go` | RunPhase(): ciclo de 6 pasos síncrono |
| `internal/boomerang/delegate.go` | DelegateStep(): exec.Command bloqueante con wg.Wait() |
| `internal/boomerang/think.go` | ThinkStep(): DAG por fase (F0-F4) |
| `internal/boomerang/memory.go` | MemoryStep(): consulta EngramStore |
| `internal/boomerang/git.go` | GitStep(): git status --porcelain |
| `internal/boomerang/quality.go` | QualityStep(): go build + go test |
| `internal/boomerang/save.go` | SaveStep(): persiste facts en Engram |
| `internal/boomerang/boomerang_test.go` | Tests con mockStore |
| `internal/scheduler/scheduler.go` | Scheduler con approval gates |
| `internal/scheduler/phase.go` | PhaseRunner interface, Config, HarnessValidator |
| `internal/scheduler/phase_stubs.go` | F0Runner-F4Runner (cada uno con exec.Command bloqueante) |
| `docs/architecture-v2.md` | Arquitectura general v2 |
| `sdd/bridge-orchestration-agent/proposal.md` | Propuesta pre-investigación en run.go |

---

## Apéndice: Fragmentos de Código Clave

### A. Phase Skip Matrix — Implementación Completa

```go
// boomerang/skip.go
package boomerang

// Step representa un paso del ciclo Boomerang
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

// PhaseStepMatrix define qué pasos ejecutar en cada fase
type PhaseStepMatrix map[string][]Step

// DefaultPhaseMatrix retorna la matriz por defecto
func DefaultPhaseMatrix() PhaseStepMatrix {
    return PhaseStepMatrix{
        "F0": {StepMemory, StepThink, StepDelegate, StepSave},
        "F1": {StepMemory, StepThink, StepDelegate, StepSave},
        "F2": {StepMemory, StepThink, StepDelegate, StepSave},
        "F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
        "F4": {StepThink, StepDelegate, StepGit, StepSave},
    }
}

// ShouldRun verifica si un paso debe ejecutarse en una fase
func (m PhaseStepMatrix) ShouldRun(phase string, step Step) bool {
    steps, ok := m[phase]
    if !ok {
        return true // Por defecto, ejecutar todos
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

### B. Channel Bus Central con Select Multiplexado

```go
// boomerang/eventbus.go
package boomerang

import "context"

// EventType categoriza los eventos del bus
type EventType int

const (
    EventStepStarted   EventType = iota
    EventStepCompleted
    EventStepSkipped
    EventStepFailed
    EventTaskStarted
    EventTaskOutput
    EventTaskCompleted
    EventTaskFailed
    EventUserInput
    EventApprovalNeeded
    EventApprovalGranted
    EventApprovalDenied
    EventPhaseCompleted
)

// Event es un evento en el bus
type Event struct {
    Type    EventType
    Phase   string
    Step    Step
    Data    interface{}
    Error   error
}

// EventBus es el canal central de eventos
type EventBus chan Event

// NewEventBus crea un bus con buffer
func NewEventBus(buffer int) EventBus {
    return make(chan Event, buffer)
}

// EventLoop procesa eventos del bus hasta que ctx se cancela
func EventLoop(ctx context.Context, bus EventBus, handlers map[EventType]func(Event)) {
    for {
        select {
        case event := <-bus:
            if handler, ok := handlers[event.Type]; ok {
                handler(event)
            }
        case <-ctx.Done():
            return
        }
    }
}
```

---

*Fin del documento de investigación*
