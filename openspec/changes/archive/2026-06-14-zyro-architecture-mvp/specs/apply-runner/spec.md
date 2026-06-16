# Delta Spec: Apply Runner

## NEW Requirements

### Requirement: Task Types

The apply package MUST define `Task`, `Result`, `PoolConfig`, and `ResultStatus` types. A `Task` MUST carry an ID, a human-readable Name, and an Execute function returning (string, error). A `Result` MUST carry the TaskID, Name, Status, Output string, and optional Error. `PoolConfig` MUST configure PoolSize, TaskTimeout, and FailFast flags. `DefaultPoolConfig()` MUST return a config with PoolSize=5, TaskTimeout=10m, FailFast=true.

#### Scenario: Default config
- GIVEN no pool configuration
- WHEN `DefaultPoolConfig()` is called
- THEN PoolSize is 5, TaskTimeout is 10 minutes, FailFast is true

#### Scenario: Zero-value validation
- GIVEN a `PoolConfig{}` with zero values
- WHEN `Validate()` is called
- THEN PoolSize defaults to 5, TaskTimeout defaults to 10m

### Requirement: Task Runner with Goroutine Pool

The apply package MUST provide a `Runner` type that manages concurrent task execution with a bounded goroutine pool. The pool size MUST be configurable via `PoolConfig.PoolSize`. The runner MUST derive a cancelable context from the parent context so fail-fast can stop all workers. Tasks MUST be fanned out to workers via a buffered channel.

#### Scenario: Empty tasks
- GIVEN a Runner
- WHEN `Run(ctx, nil)` or `Run(ctx, [])` is called
- THEN nil is returned

#### Scenario: All tasks succeed
- GIVEN 3 tasks that all return success
- WHEN `Run(ctx, tasks)` is called
- THEN all 3 results have StatusSuccess

#### Scenario: Concurrent execution
- GIVEN 6 tasks each taking 100ms with PoolSize=3
- WHEN `Run(ctx, tasks)` is called
- THEN at least 2 tasks run concurrently

#### Scenario: Large batch
- GIVEN 20 fast tasks with PoolSize=2
- WHEN `Run(ctx, tasks)` is called
- THEN all 20 results are returned with StatusSuccess

### Requirement: Result Collection via Channel

The runner MUST collect results via a buffered `chan Result`. The channel MUST have capacity sufficient to prevent sender blocking. After all workers finish, the channel MUST be closed and results aggregated into a slice.

#### Scenario: All tasks have results
- GIVEN 4 tasks
- WHEN the runner completes
- THEN exactly 4 results are returned, one per task ID

#### Scenario: Error output propagation
- GIVEN a task that returns both output and error
- WHEN the runner executes it
- THEN the Result contains the output string AND the error

### Requirement: Per-Task Timeout

Each task MUST run within its own timeout derived from `PoolConfig.TaskTimeout`. If a task exceeds the timeout, it MUST receive a `StatusTimeout` result with a descriptive error. The timeout is implemented via `context.WithTimeout` per task.

#### Scenario: Task timeout
- GIVEN a task that sleeps longer than the configured TaskTimeout
- WHEN the runner executes it
- THEN the result Status is "timeout" and the error mentions "timed out"

#### Scenario: Fail-fast on timeout
- GIVEN FailFast=true and one task that times out
- WHEN the runner executes
- THEN remaining tasks are skipped with StatusSkipped

### Requirement: Fail-Fast Error Handling

When `FailFast` is true (default), the first error MUST cancel the context for all in-flight and remaining tasks. Workers MUST check the context before starting each task and skip with `StatusSkipped` if cancelled. When `FailFast` is false, all tasks MUST execute regardless of individual failures.

#### Scenario: First error cancels rest
- GIVEN 3 tasks with FailFast=true, where the first task fails
- WHEN `Run(ctx, tasks)` is called
- THEN at least one task has StatusFailed and at least one has StatusSkipped

#### Scenario: No fail-fast
- GIVEN 3 tasks with FailFast=false, where 2 fail
- WHEN `Run(ctx, tasks)` is called
- THEN all 3 execute; 2 have StatusFailed and 1 has StatusSuccess

### Requirement: Context Propagation

The runner MUST accept an external `context.Context` and propagate cancellation to all workers. If the context is cancelled externally, workers MUST skip pending tasks with `StatusSkipped`.

#### Scenario: External context cancellation
- GIVEN a Runner and a cancellable context
- WHEN the context is cancelled during execution
- THEN in-flight tasks return and pending tasks are skipped
