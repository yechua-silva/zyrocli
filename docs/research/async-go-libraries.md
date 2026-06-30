# Async Go Libraries & Patterns for BoomerangOrchestrator

> Research date: 2026-06-17
> Context: Refactoring `BoomerangOrchestrator` in ZyroAgentCLI from synchronous `exec.CommandContext()` to async subagent delegation.
> Goals: non-blocking orchestration, notification on completion, phase-gated approval flow.

---

## 1. Go Concurrency Primitives — Core Patterns

### 1.1 Goroutines + Channels (fundamental)

**Source:** https://go.dev/blog/pipelines (Go Concurrency Patterns: Pipelines & cancellation)

**Pattern:** Pipeline stages connected by channels, fan-out/fan-in, done-channel broadcast for cancellation.

```go
// Fan-out pattern: distribute work across N workers
tasks := make(chan TaskSpec)
results := make(chan TaskResult)

// Start N workers
var wg sync.WaitGroup
for i := 0; i < numWorkers; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for task := range tasks {
            result := executeTask(task)
            results <- result
        }
    }()
}

// Close results when all workers done
go func() {
    wg.Wait()
    close(results)
}()
```

**Key insight for Boomerang:** Replace `go func() { ... wg.Done() }` with channel-based communication so the orchestrator reads results as they arrive instead of blocking on `wg.Wait()`.

**Pros:**
- Zero dependencies, idiomatic Go
- Channel close broadcasts to all readers (cancellation signal)
- `select` enables non-blocking multi-wait

**Cons:**
- Manual error propagation (no built-in error aggregation)
- No panic recovery
- Easy to leak goroutines if channels aren't properly closed

**URLs:**
- https://go.dev/blog/pipelines
- https://go.dev/blog/context

### 1.2 `context` Package — Cancellation & Deadlines

**Source:** https://go.dev/blog/context

**Patterns:**
- `context.WithCancel(ctx)` → returns cancel func
- `context.WithTimeout(ctx, duration)` → auto-cancel after timeout
- `context.WithDeadline(ctx, time)` → cancel at specific time
- Value propagation for request-scoped data

```go
ctx, cancel := context.WithTimeout(parentCtx, 5*time.Minute)
defer cancel()

// In a worker:
select {
case result := <-workerCh:
    return result
case <-ctx.Done():
    return nil, ctx.Err() // DeadlineExceeded or Canceled
}
```

**For Boomerang:** Each subagent gets a derived context. If the user cancels or a phase times out, all subagents are killed automatically via `cmd.Cancel` (set by `CommandContext`).

**Pros:**
- Standard library, no dependencies
- Tree-structured cancellation (parent cancels → all children cancel)
- Deadline propagation across API boundaries

**Cons:**
- Value propagation via `WithValue` is type-unsafe (interface{})
- No built-in way to wait for goroutines to finish after cancellation

**URLs:**
- https://pkg.go.dev/context
- https://go.dev/blog/context

---

## 2. Goroutine Sync Primitives (stdlib vs libs)

### 2.1 `sync.WaitGroup` (stdlib)

**What:** Classic counter-based goroutine synchronization.

**For Boomerang:** Currently used in `DelegateStep`. Blocks until all goroutines in a ParallelGroup finish.

```go
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Add(1)
    go func(t TaskSpec) {
        defer wg.Done()
        // ... execute ...
    }(task)
}
wg.Wait() // BLOCKS — this is the problem
```

**Pros:** Simple, zero dependencies.

**Cons:** No error propagation, no context awareness, blocks the caller.

### 2.2 `errgroup.Group` (golang.org/x/sync/errgroup)

**Source:** https://pkg.go.dev/golang.org/x/sync/errgroup

**What:** Sync.WaitGroup + error propagation + context cancellation.

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(5) // max 5 concurrent goroutines

for _, task := range tasks {
    task := task
    g.Go(func() error {
        return executeSubagent(ctx, task)
    })
}

// Wait returns the first error (cancels context for others)
if err := g.Wait(); err != nil {
    return nil, err
}
```

**For Boomerang:** Replace `sync.WaitGroup` + manual error handling. `errgroup.WithContext` auto-cancels all goroutines on first error — perfect for "fail fast" scenarios.

**Pros:**
- Error propagation + context cancellation built-in
- `SetLimit` for bounding concurrency
- 30K+ importers, extremely stable

**Cons:**
- No panic recovery
- Only returns the *first* error (can use `WithFirstError` in newer versions)
- No streaming results (blocks until all done)

**URLs:**
- https://pkg.go.dev/golang.org/x/sync/errgroup
- https://pkg.go.dev/golang.org/x/sync@v0.21.0/errgroup

### 2.3 `sourcegraph/conc` — Structured Concurrency

**Source:** https://github.com/sourcegraph/conc

**What:** Better structured concurrency: panic-safe WaitGroup, pools, streams, iterators.

```go
import "github.com/sourcegraph/conc/pool"

// Bounded pool with error handling + context
p := pool.New().
    WithMaxGoroutines(10).
    WithErrors().
    WithContext(ctx)

for _, task := range tasks {
    task := task
    p.Go(func(ctx context.Context) error {
        return executeSubagent(ctx, task)
    })
}
err := p.Wait() // first error, panics propagated
```

**Stream pattern** (ordered results from concurrent work):

```go
s := stream.New().WithMaxGoroutines(10)
for _, task := range tasks {
    task := task
    s.Go(func() stream.Callback {
        result := executeSubagent(context.Background(), task)
        return func() { results = append(results, result) }
    })
}
s.Wait()
```

**For Boomerang:** Best fit for replacing `DelegateStep`. The `pool.Pool` with context + errors maps directly to parallel group execution. The `stream.Stream` could be used for ordered task results.

**Pros:**
- Panic-safe (catches and propagates panics with stack traces)
- Rich pool types (ErrorPool, ContextPool, ResultPool)
- Stream for ordered concurrent processing
- Clean API, minimal boilerplate

**Cons:**
- External dependency (pre-1.0 at v0.3.0)
- Additional layer of abstraction
- Less flexible than raw channels for complex patterns

**URLs:**
- https://github.com/sourcegraph/conc
- https://pkg.go.dev/github.com/sourcegraph/conc
- https://about.sourcegraph.com/blog/building-conc-better-structured-concurrency-for-go

---

## 3. Async Subprocess Management (`os/exec`)

### 3.1 `Start()` + `Wait()` Pattern — Non-blocking process launch

**Source:** https://pkg.go.dev/os/exec

**The current problem in `delegate.go`:**
```go
output, err := cmd.Output() // BLOCKS until complete
```

**The async solution:**
```go
cmd := exec.CommandContext(ctx, "opencode", args...)

// Set up stdout pipe for non-blocking reads
stdout, _ := cmd.StdoutPipe()
stderr, _ := cmd.StderrPipe()

// Start (non-blocking)
if err := cmd.Start(); err != nil {
    return nil, err
}

// Read output in goroutines while process runs
go func() {
    output, _ := io.ReadAll(stdout)
    resultCh <- TaskResult{Output: string(output)}
}()

go func() {
    errOutput, _ := io.ReadAll(stderr)
    if len(errOutput) > 0 {
        errCh <- string(errOutput)
    }
}()

// Wait in background
go func() {
    err := cmd.Wait()
    if err != nil {
        errCh <- err.Error()
    }
    close(doneCh)
}()
```

### 3.2 StdoutPipe + Streaming Reads

**Pattern:** Read subprocess output line-by-line as it's produced:

```go
cmd := exec.CommandContext(ctx, "opencode", args...)
stdout, _ := cmd.StdoutPipe()
cmd.Start()

scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    line := scanner.Text()
    // Process line in real-time (e.g., send to UI channel)
    uiCh <- line
}
cmd.Wait()
```

**For Boomerang:** This enables streaming subagent output to the user while the agent works, maintaining interactivity.

### 3.3 `Cmd.Cancel` and `WaitDelay` — Graceful Shutdown (Go 1.20+)

```go
cmd := exec.CommandContext(ctx, "opencode", args...)
cmd.Cancel = func() error {
    // Custom cancellation: send SIGINT first, then SIGKILL after delay
    return cmd.Process.Signal(syscall.SIGINT)
}
cmd.WaitDelay = 30 * time.Second // Kill after 30s if not responding
```

**For Boomerang:** Graceful subagent shutdown. If a phase is cancelled, send SIGINT to subagents first, then SIGKILL after WaitDelay.

**Pros:** Standard library, non-blocking via pipes and goroutines.

**Cons:** Pipe management is manual; watching both stdout and stderr requires two goroutines.

**URLs:**
- https://pkg.go.dev/os/exec#Cmd.Start
- https://pkg.go.dev/os/exec#Cmd.StdoutPipe
- https://pkg.go.dev/os/exec#Cmd.Cancel

---

## 4. Event Bus / Pub-Sub Patterns

### 4.1 Channel-Based Event Bus (no deps)

```go
type EventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
}

func (b *EventBus) Subscribe(topic string, ch chan Event) {
    b.mu.Lock()
    b.subscribers[topic] = append(b.subscribers[topic], ch)
    b.mu.Unlock()
}

func (b *EventBus) Publish(topic string, event Event) {
    b.mu.RLock()
    for _, ch := range b.subscribers[topic] {
        select {
        case ch <- event:
        default: // non-blocking, drop slow consumers
        }
    }
    b.mu.RUnlock()
}
```

**For Boomerang:** Publish events when:
- Subagent starts → `"subagent.started"`
- Subagent finishes → `"subagent.completed"`
- Phase changes → `"phase.changed"`
- Error occurs → `"system.error"`

The orchestrator subscribes to these events to drive the state machine without blocking.

### 4.2 `watcher`/`notifier` Pattern

```go
type SubagentNotifier struct {
    listeners map[string][]func(SubagentEvent)
    mu        sync.Mutex
}

func (n *SubagentNotifier) On(event string, fn func(SubagentEvent)) {
    n.mu.Lock()
    n.listeners[event] = append(n.listeners[event], fn)
    n.mu.Unlock()
}

func (n *SubagentNotifier) Emit(event string, data SubagentEvent) {
    n.mu.Lock()
    fns := append([]func(SubagentEvent){}, n.listeners[event]...)
    n.mu.Unlock()
    for _, fn := range fns {
        fn(data) // synchronous — could fan out to goroutines
    }
}
```

**URLs (conceptual):**
- https://eli.thegreenplace.net/2020/building-an-event-bus-in-go/ (404, but pattern is well-known)
- https://pkg.go.dev/github.com/asaskevich/EventBus (popular Go event bus)
- https://github.com/tevino/abool (atomic bools for state flags)

---

## 5. Workflow / Orchestration Libraries

### 5.1 Temporal.io Go SDK

**Source:** https://github.com/temporalio/temporal

**What:** Durable execution platform — workflows survive process restarts.

```go
func MyWorkflow(ctx workflow.Context, input string) error {
    // Activities execute in separate processes, with retries
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
        StartToCloseTimeout: 5 * time.Minute,
        HeartbeatTimeout:     30 * time.Second,
        RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 3},
    })

    var result1 string
    err := workflow.ExecuteActivity(ctx, SubagentActivity, "task1").Get(ctx, &result1)
    if err != nil {
        return err
    }

    // Parallel activities with futures
    var f1, f2 workflow.Future
    f1 = workflow.ExecuteActivity(ctx, SubagentActivity, "task2")
    f2 = workflow.ExecuteActivity(ctx, SubagentActivity, "task3")
    f1.Get(ctx, nil)
    f2.Get(ctx, nil)

    return nil
}
```

**For Boomerang:** Could model the 6-phase Boomerang cycle as a Temporal workflow with activities for each step. Built-in retries, timeouts, and visibility.

**Pros:**
- Production-grade, used at Stripe, Netflix, Snap
- Durable — survives crashes
- Built-in retries, timeouts, signal handling
- Web UI for monitoring workflow execution

**Cons:**
- Requires running Temporal Server (heavyweight)
- Significant architectural shift
- Overkill for local CLI tool
- Network latency to server

**URLs:**
- https://github.com/temporalio/temporal (21K stars)
- https://docs.temporal.io/dev-guide/go
- https://github.com/temporalio/samples-go

### 5.2 Cadence (Uber)

**Source:** https://github.com/uber/cadence

**What:** Temporal's predecessor (both forked from same internal tool at Uber).

**Similar to Temporal** but older. Not recommended for new projects — Temporal is the active fork with more features.

### 5.3 Dagger (Dagger.io)

**Source:** https://dagger.io

**What:** CI/CD orchestration SDK that runs pipelines in containers.

```go
import "dagger.io/dagger"

func (m *Module) Build(ctx context.Context) (string, error) {
    return dag.Container().
        From("golang:1.26").
        WithDirectory("/src", dag.Host().Directory(".")).
        WithWorkdir("/src").
        WithExec([]string{"go", "build", "-o", "output"}).
        Stdout(ctx)
}
```

**For Boomerang:** Dagger modules could encapsulate subagent execution in containers. Useful if subagents need isolated environments.

**Pros:** Container-native, reproducible, composable modules (Daggerverse).

**Cons:** 
- Designed for CI/CD, not general orchestration
- Container overhead for each subagent
- Different paradigm than process execution

**URLs:**
- https://dagger.io
- https://daggerverse.dev/
- https://docs.dagger.io

### 5.4 `looplab/fsm` — Finite State Machine

**Source:** https://github.com/looplab/fsm

**What:** Declarative finite state machine for Go.

```go
phaseFSM := fsm.NewFSM(
    "idle",
    fsm.Events{
        {Name: "start_phase", Src: []string{"idle"}, Dst: "running"},
        {Name: "approve",     Src: []string{"running"}, Dst: "approved"},
        {Name: "reject",      Src: []string{"running"}, Dst: "rejected"},
        {Name: "advance",     Src: []string{"approved"}, Dst: "running"}, // next phase
    },
    fsm.Callbacks{
        "enter_state": func(ctx context.Context, e *fsm.Event) {
            log.Printf("Phase %s → %s (event: %s)", e.Src, e.Dst, e.Event)
        },
        "before_approve": func(ctx context.Context, e *fsm.Event) {
            // Validation before allowing transition
            if !allTasksCompleted() {
                e.Cancel(errors.New("not all tasks completed"))
            }
        },
    },
)

err := phaseFSM.Event(ctx, "start_phase") // triggers callbacks
```

**For Boomerang:** Model the 6-phase cycle as an FSM. Each phase transition requires explicit approval. Phase cannot advance without all tasks completing.

**Pros:**
- Declarative, readable state machine
- Callbacks for enter/leave/before/after transitions
- Async transitions supported (via `e.Async()`)
- Mermaid/Graphviz visualization

**Cons:**
- Another dependency (v1.0.3, stable)
- Callback-based (can get tangled with complex logic)

**URLs:**
- https://github.com/looplab/fsm
- https://pkg.go.dev/github.com/looplab/fsm@v1.0.3

---

## 6. Phase-Configurable Execution — Strategy Pattern

### 6.1 Phase Configuration Matrix

```go
type PhaseConfig struct {
    Name       string
    Steps      []StepConfig
    RequiresApproval bool
    MaxRetries int
    Timeout    time.Duration
}

type StepConfig struct {
    Name     string
    Agent    string
    Optional bool // true → skip if fails
    Timeout  time.Duration
    DependsOn []string // step dependencies
}

var phaseMatrix = map[string]PhaseConfig{
    "F0": {
        Name: "Research",
        Steps: []StepConfig{
            {Name: "patterns",  Agent: "zyro-phase-0-patterns",  Optional: false},
            {Name: "libraries", Agent: "zyro-phase-0-libraries", Optional: false},
            {Name: "skills",    Agent: "zyro-skills-find",       Optional: true},
        },
        RequiresApproval: false,
    },
    "F1": {
        Name: "Spec",
        Steps: []StepConfig{
            {Name: "spec-design", Agent: "zyro-sdd-spec", Optional: false},
            {Name: "spec-review", Agent: "zyro-sdd-verify", Optional: false},
        },
        RequiresApproval: true, // must be approved before F2
    },
}
```

### 6.2 Strategy Pattern for Step Execution

```go
type StepStrategy interface {
    Execute(ctx context.Context, step StepConfig) (*TaskResult, error)
}

type ParallelStrategy struct {
    maxConcurrency int
}

func (s *ParallelStrategy) Execute(ctx context.Context, steps []StepConfig) []*TaskResult {
    pool := pool.New().WithMaxGoroutines(s.maxConcurrency).WithContext(ctx)
    results := make([]*TaskResult, len(steps))
    
    for i, step := range steps {
        i, step := i, step
        pool.Go(func(ctx context.Context) error {
            result, err := runSubagent(ctx, step)
            results[i] = result
            return err
        })
    }
    pool.Wait()
    return results
}

type SequentialStrategy struct{}

func (s *SequentialStrategy) Execute(ctx context.Context, steps []StepConfig) []*TaskResult {
    var results []*TaskResult
    for _, step := range steps {
        result, err := runSubagent(ctx, step)
        results = append(results, result)
        if err != nil && !step.Optional {
            return results
        }
    }
    return results
}
```

**For Boomerang:** Phase matrix defines which strategy to use (parallel, sequential, or DAG-based). Optional steps don't block phase advancement.

---

## 7. Recommended Architecture for Boomerang Async

### 7.1 Recommended Stack (lightweight, no external deps)

| Concern | Solution | Why |
|---------|----------|-----|
| Error propagation | `errgroup.Group` | Drop-in for `sync.WaitGroup` |
| Context cancellation | `context.WithCancel` + `context.WithTimeout` | Already partially used |
| Non-blocking subprocess | `cmd.Start()` + `cmd.StdoutPipe()` | Replace `cmd.Output()` |
| Event notification | Channel-based event bus | Keep it simple, no deps |
| Phase state machine | `looplab/fsm` OR hand-rolled `select` | FSM for clarity |
| Phase config | YAML + Strategy pattern | Declarative, testable |

### 7.2 Recommended Stack (if dependencies are OK)

| Concern | Solution | Why |
|---------|----------|-----|
| All goroutine management | `sourcegraph/conc` | Pools, streams, panic-safety |
| Event bus | `asaskevich/EventBus` | Simple pub-sub |
| FSM | `looplab/fsm` | Rich callbacks, async support |
| Config | YAML via `gopkg.in/yaml.v3` | Already in go.mod |

### 7.3 Overkill (not recommended)

| Library | Why Not |
|---------|---------|
| Temporal/Cadence | Requires server, too heavy for CLI tool |
| Dagger | Container-centric, CI/CD focused |
| GoWorkflow | Overengineered for subagent delegation |

### 7.4 Critical Code Changes Needed

1. **`orchestrator.go` — `RunPhase`**: Instead of calling each step synchronously, launch steps as goroutines that communicate via channels. Add a `select` loop to handle:
   - Step completion events
   - User approval requests
   - Context cancellation
   - Timeouts

2. **`delegate.go` — `DelegateStep`**: 
   - Replace `cmd.Output()` with `cmd.Start()` + `cmd.StdoutPipe()`
   - Return results via channel instead of blocking on `wg.Wait()`
   - Add progress reporting channel for streaming output

3. **New: Event Bus**:
   - Add `internal/boomerang/events.go`
   - Define `SubagentStarted`, `SubagentCompleted`, `PhaseApprovalRequired` events
   - Orchestrator subscribes to drive state machine

4. **New: Phase FSM / Approval Gate**:
   - After each phase completes, emit approval-required event
   - Block phase transition until approval received (via channel)
   - Store approval state in the orchestrator

---

## 8. Code Snippets — Key Patterns

### 8.1 Async Delegate Step with Streaming

```go
func (o *BoomerangOrchestrator) DelegateStepAsync(
    ctx context.Context, dag *TaskDAG, phase string,
) (<-chan TaskResult, <-chan error) {
    
    resultCh := make(chan TaskResult, len(dag.Tasks))
    errCh := make(chan error, 1)
    
    go func() {
        defer close(resultCh)
        
        g, gCtx := errgroup.WithContext(ctx)
        g.SetLimit(5) // max 5 concurrent subagents
        
        for _, group := range dag.ParallelGroups {
            select {
            case <-gCtx.Done():
                errCh <- gCtx.Err()
                return
            default:
            }
            
            for _, taskIdx := range group {
                if taskIdx >= len(dag.Tasks) {
                    continue
                }
                task := dag.Tasks[taskIdx]
                
                g.Go(func() error {
                    cmd := exec.CommandContext(gCtx, "opencode", 
                        "subagent", task.Agent,
                        "--param", fmt.Sprintf("task=%s", task.Name),
                        "--param", fmt.Sprintf("phase=%s", phase),
                    )
                    
                    stdout, _ := cmd.StdoutPipe()
                    stderr, _ := cmd.StderrPipe()
                    
                    if err := cmd.Start(); err != nil {
                        return err
                    }
                    
                    // Stream output (non-blocking)
                    output, _ := io.ReadAll(stdout)
                    errOutput, _ := io.ReadAll(stderr)
                    
                    if err := cmd.Wait(); err != nil {
                        return fmt.Errorf("%s failed: %w (stderr: %s)", 
                            task.Name, err, string(errOutput))
                    }
                    
                    resultCh <- TaskResult{
                        TaskName: task.Name,
                        Success:  true,
                        Output:   string(output),
                    }
                    return nil
                })
            }
            
            // Wait for this group to complete before next group
            if err := g.Wait(); err != nil {
                errCh <- err
                return
            }
        }
    }()
    
    return resultCh, errCh
}
```

### 8.2 Phase State Machine with Approval Gate

```go
type PhaseState int

const (
    PhaseIdle      PhaseState = iota
    PhaseRunning
    PhaseApprovalRequired
    PhaseApproved
    PhaseRejected
    PhaseCompleted
    PhaseFailed
)

type PhaseMachine struct {
    state    PhaseState
    stateCh  chan PhaseState
    approveCh chan struct{}
    rejectCh chan struct{}
    mu       sync.Mutex
}

func (pm *PhaseMachine) Run(ctx context.Context, phase PhaseConfig, runFn func() error) error {
    pm.mu.Lock()
    pm.state = PhaseRunning
    pm.mu.Unlock()
    
    // Run phase in background
    errCh := make(chan error, 1)
    go func() {
        errCh <- runFn()
    }()
    
    select {
    case err := <-errCh:
        if err != nil {
            pm.setState(PhaseFailed)
            return err
        }
        if phase.RequiresApproval {
            pm.setState(PhaseApprovalRequired)
            // Wait for approval (blocks until user approves)
            select {
            case <-pm.approveCh:
                pm.setState(PhaseApproved)
            case <-pm.rejectCh:
                pm.setState(PhaseRejected)
                return fmt.Errorf("phase %s rejected", phase.Phase)
            case <-ctx.Done():
                pm.setState(PhaseFailed)
                return ctx.Err()
            }
        }
        pm.setState(PhaseCompleted)
        return nil
        
    case <-ctx.Done():
        pm.setState(PhaseFailed)
        return ctx.Err()
    }
}

func (pm *PhaseMachine) Approve() {
    pm.approveCh <- struct{}{}
}

func (pm *PhaseMachine) Reject() {
    pm.rejectCh <- struct{}{}
}
```

### 8.3 Event Bus for Orchestrator Notifications

```go
package boomerang

type EventType string

const (
    EventSubagentStarted   EventType = "subagent.started"
    EventSubagentCompleted EventType = "subagent.completed"
    EventSubagentFailed    EventType = "subagent.failed"
    EventPhaseStarted      EventType = "phase.started"
    EventPhaseApprovalReq  EventType = "phase.approval_required"
    EventPhaseCompleted    EventType = "phase.completed"
    EventPhaseFailed       EventType = "phase.failed"
    EventOrchestratorMsg   EventType = "orchestrator.message"
)

type Event struct {
    Type      EventType
    Phase     string
    TaskName  string
    Message   string
    Data      interface{}
    Timestamp time.Time
    Error     error
}

type EventBus struct {
    subscribers map[EventType][]chan Event
    mu          sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[EventType][]chan Event),
    }
}

func (eb *EventBus) Subscribe(eventType EventType, buffer int) chan Event {
    ch := make(chan Event, buffer)
    eb.mu.Lock()
    eb.subscribers[eventType] = append(eb.subscribers[eventType], ch)
    eb.mu.Unlock()
    return ch
}

func (eb *EventBus) Publish(event Event) {
    eb.mu.RLock()
    channels := eb.subscribers[event.Type]
    eb.mu.RUnlock()
    
    for _, ch := range channels {
        select {
        case ch <- event:
        default: // non-blocking: drop if consumer is slow
        }
    }
}

// In orchestrator:
// eventCh := eventBus.Subscribe(EventPhaseApprovalReq, 1)
// for event := range eventCh { handleApproval(event) }
```

### 8.4 Using `sourcegraph/conc` — DelegateStep Rewrite

```go
import "github.com/sourcegraph/conc/pool"

func (o *BoomerangOrchestrator) DelegateStepConc(
    ctx context.Context, dag *TaskDAG, phase string,
) (*DelegateResult, error) {
    
    result := &DelegateResult{
        TaskResults: make(map[string]TaskResult),
    }
    var mu sync.Mutex
    
    for _, group := range dag.ParallelGroups {
        p := pool.New().
            WithMaxGoroutines(5).
            WithErrors().
            WithContext(ctx)
        
        for _, taskIdx := range group {
            if taskIdx >= len(dag.Tasks) {
                continue
            }
            task := dag.Tasks[taskIdx]
            
            p.Go(func(ctx context.Context) error {
                cmd := exec.CommandContext(ctx, "opencode",
                    "subagent", task.Agent,
                    "--param", fmt.Sprintf("task=%s", task.Name),
                    "--param", fmt.Sprintf("phase=%s", phase),
                )
                
                output, err := cmd.Output()
                tr := TaskResult{
                    TaskName: task.Name,
                    Success:  err == nil,
                    Output:   string(output),
                }
                
                mu.Lock()
                result.TaskResults[task.Name] = tr
                if tr.Success {
                    result.NodesCreated++
                }
                mu.Unlock()
                
                return err
            })
        }
        
        // Wait for group — first error cancels context for others
        if err := p.Wait(); err != nil {
            // Option: return error OR continue (depending on phase config)
            return result, err
        }
    }
    
    return result, nil
}
```

---

## 9. Summary & Recommendations

### For immediate implementation (no new deps):
1. **Replace `sync.WaitGroup` with `errgroup.Group`** — keeps stdlib feel, adds error+cancellation
2. **Replace `cmd.Output()` with `cmd.Start()` + `cmd.StdoutPipe()`** — non-blocking process execution
3. **Add channel-based event bus** — decouple phases from UI
4. **Add approval gate channels** — block phase progression without blocking the orchestrator

### If adding dependencies is acceptable:
5. **Add `sourcegraph/conc`** — cleaner pools, panic safety, stream ordering
6. **Add `looplab/fsm`** — formal phase state machine with visualization
7. **Use YAML phase config** (already in go.mod) — declarative phase definitions

### Architectural Decision: Channel-based Orchestrator Loop

The orchestrator should become an event loop:

```
                  ┌─────────────────────────┐
                  │  Orchestrator Event Loop │
                  │  select {                │
                  │    case <-subagentDone   │
                  │    case <-approval       │
                  │    case <-ctx.Done()     │
                  │    case <-timeout        │
                  │    case <-userMsg        │
                  │  }                       │
                  └─────────────────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
  ┌──────────┐      ┌──────────┐      ┌──────────┐
  │ Phase 0  │      │ Phase 1  │      │ Phase N  │
  │ (async)  │ ──►  │ (approve)│ ──►  │ (async)  │
  └──────────┘      └──────────┘      └──────────┘
```

This allows the orchestrator to:
- Respond to user messages while phases run
- Collect subagent results as they complete
- Block between phases until approval
- Handle cancellation and timeouts at any point
- Publish progress events for UI updates

---

## References

| Resource | URL |
|----------|-----|
| Go Pipeline Patterns | https://go.dev/blog/pipelines |
| Go Context Patterns | https://go.dev/blog/context |
| errgroup docs | https://pkg.go.dev/golang.org/x/sync/errgroup |
| sourcegraph/conc | https://github.com/sourcegraph/conc |
| conc blog post | https://about.sourcegraph.com/blog/building-conc-better-structured-concurrency-for-go |
| os/exec docs | https://pkg.go.dev/os/exec |
| looplab/fsm | https://github.com/looplab/fsm |
| Temporal Go SDK | https://docs.temporal.io/dev-guide/go |
| Temporal GitHub | https://github.com/temporalio/temporal |
| Dagger | https://dagger.io |
| Asaskevich EventBus | https://github.com/asaskevich/EventBus |
| Go Code Review Comments (concurrency) | https://go.dev/wiki/CodeReviewComments#concurrency |
