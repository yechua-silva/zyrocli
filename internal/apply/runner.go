package apply

import (
	"context"
	"fmt"
	"sync"
)

// Runner manages concurrent task execution with a bounded goroutine pool.
// Tasks are fanned out to workers, results are collected via a channel,
// and fail-fast semantics cancel remaining tasks on first error.
type Runner struct {
	config PoolConfig
}

// NewRunner creates a Runner with the given pool configuration.
func NewRunner(config PoolConfig) *Runner {
	config = config.Validate()
	return &Runner{config: config}
}

// resultChanSize is the capacity of the buffered result channel.
// Set to the pool size so senders never block on the channel.
const resultChanSize = 128

// Run executes the given tasks using a goroutine pool with the configured
// pool size. Each task runs with its own context and timeout derived from
// the parent ctx. Results are collected via a buffered channel.
//
// If FailFast is true (default), the first task error cancels the context
// for all in-flight and remaining tasks.
//
// The function blocks until all tasks complete or are cancelled.
func (r *Runner) Run(ctx context.Context, tasks []Task) []Result {
	if len(tasks) == 0 {
		return nil
	}

	// Derive a cancelable context so fail-fast can stop all workers.
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	taskCh := make(chan Task, len(tasks))
	resultCh := make(chan Result, resultChanSize)
	var wg sync.WaitGroup

	// Start the goroutine pool
	for i := 0; i < r.config.PoolSize; i++ {
		wg.Add(1)
		go r.worker(runCtx, i, taskCh, resultCh, &wg, cancel)
	}

	// Enqueue all tasks
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	// Wait for all workers to finish, then close the result channel
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var results []Result
	for res := range resultCh {
		results = append(results, res)
	}

	return results
}

// worker pulls tasks from the channel and executes them with a per-task timeout.
func (r *Runner) worker(ctx context.Context, id int, tasks <-chan Task, results chan<- Result, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()

	for task := range tasks {
		// Check if context was already cancelled (fail-fast triggered)
		if ctx.Err() != nil {
			results <- Result{
				TaskID: task.ID,
				Name:   task.Name,
				Status: StatusSkipped,
				Output: "",
				Error:  ctx.Err(),
			}
			continue
		}

		// Create per-task timeout context
		taskCtx, taskCancel := context.WithTimeout(ctx, r.config.TaskTimeout)

		// Run the task in a goroutine so we can select on timeout/cancellation
		taskResult := make(chan struct {
			output string
			err    error
		}, 1)

		go func() {
			output, err := task.Execute()
			taskResult <- struct {
				output string
				err    error
			}{output, err}
		}()

		// Wait for either completion or timeout/cancellation
		select {
		case <-taskCtx.Done():
			// Timeout or parent cancellation
			result := Result{
				TaskID: task.ID,
				Name:   task.Name,
				Status: StatusTimeout,
				Output: "",
			}
			if errorsIs(taskCtx.Err(), context.DeadlineExceeded) {
				result.Error = fmt.Errorf("task %q timed out after %v", task.Name, r.config.TaskTimeout)
			} else {
				result.Error = taskCtx.Err()
			}
			results <- result
			taskCancel()
			// On timeout, trigger fail-fast
			if r.config.FailFast {
				cancel()
			}

		case tr := <-taskResult:
			taskCancel()
			if tr.err != nil {
				result := Result{
					TaskID: task.ID,
					Name:   task.Name,
					Status: StatusFailed,
					Output: tr.output,
					Error:  tr.err,
				}
				results <- result
				// Fail-fast: first error cancels the rest
				if r.config.FailFast {
					cancel()
				}
			} else {
				results <- Result{
					TaskID: task.ID,
					Name:   task.Name,
					Status: StatusSuccess,
					Output: tr.output,
				}
			}
		}
	}
}

// errorsIs is a small wrapper so we don't need to import errors just for this.
func errorsIs(err, target error) bool {
	return err == target
}
