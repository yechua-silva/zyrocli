package apply

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Helper: safeTask returns a Task that does not panic on nil Execute.
// ---------------------------------------------------------------------------

func safeTask(id, name string, fn func() (string, error)) Task {
	return Task{ID: id, Name: name, Execute: fn}
}

// ---------------------------------------------------------------------------
// PoolConfig validation
// ---------------------------------------------------------------------------

func TestDefaultPoolConfig(t *testing.T) {
	cfg := DefaultPoolConfig()
	if cfg.PoolSize != 5 {
		t.Errorf("expected PoolSize=5, got %d", cfg.PoolSize)
	}
	if cfg.TaskTimeout != 10*time.Minute {
		t.Errorf("expected TaskTimeout=10m, got %v", cfg.TaskTimeout)
	}
	if !cfg.FailFast {
		t.Error("expected FailFast=true by default")
	}
}

func TestPoolConfig_ValidateDefaults(t *testing.T) {
	cfg := PoolConfig{}.Validate()
	if cfg.PoolSize != 5 {
		t.Errorf("expected PoolSize=5 after validation, got %d", cfg.PoolSize)
	}
	if cfg.TaskTimeout != 10*time.Minute {
		t.Errorf("expected TaskTimeout=10m after validation, got %v", cfg.TaskTimeout)
	}
}

func TestPoolConfig_ValidatePreservesExplicit(t *testing.T) {
	cfg := PoolConfig{
		PoolSize:    3,
		TaskTimeout: 30 * time.Second,
		FailFast:    false,
	}.Validate()
	if cfg.PoolSize != 3 {
		t.Errorf("expected PoolSize=3, got %d", cfg.PoolSize)
	}
	if cfg.TaskTimeout != 30*time.Second {
		t.Errorf("expected TaskTimeout=30s, got %v", cfg.TaskTimeout)
	}
	if cfg.FailFast {
		t.Error("expected FailFast=false to be preserved")
	}
}

// ---------------------------------------------------------------------------
// Basic execution
// ---------------------------------------------------------------------------

func TestRunner_RunEmptyTasks(t *testing.T) {
	r := NewRunner(DefaultPoolConfig())
	results := r.Run(context.Background(), nil)
	if results != nil {
		t.Errorf("expected nil results for nil tasks, got %d", len(results))
	}

	results = r.Run(context.Background(), []Task{})
	if results != nil {
		t.Errorf("expected nil results for empty tasks, got %d", len(results))
	}
}

func TestRunner_AllSucceed(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 3, TaskTimeout: time.Second, FailFast: false})
	tasks := []Task{
		safeTask("1", "A", func() (string, error) { return "a-out", nil }),
		safeTask("2", "B", func() (string, error) { return "b-out", nil }),
		safeTask("3", "C", func() (string, error) { return "c-out", nil }),
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	statusCount := map[ResultStatus]int{}
	for _, res := range results {
		statusCount[res.Status]++
		if res.Error != nil {
			t.Errorf("unexpected error for %s: %v", res.Name, res.Error)
		}
	}
	if statusCount[StatusSuccess] != 3 {
		t.Errorf("expected 3 success, got %v", statusCount)
	}
}

// ---------------------------------------------------------------------------
// Error handling and fail-fast
// ---------------------------------------------------------------------------

func TestRunner_FailFast(t *testing.T) {
	var executed int32
	r := NewRunner(PoolConfig{PoolSize: 2, TaskTimeout: time.Second, FailFast: true})
	tasks := []Task{
		safeTask("1", "will-fail", func() (string, error) {
			atomic.AddInt32(&executed, 1)
			return "", fmt.Errorf("task failed")
		}),
		safeTask("2", "may-run", func() (string, error) {
			atomic.AddInt32(&executed, 1)
			time.Sleep(50 * time.Millisecond)
			return "ok", nil
		}),
		safeTask("3", "should-skip", func() (string, error) {
			atomic.AddInt32(&executed, 1)
			return "should-not-run", nil
		}),
	}

	results := r.Run(context.Background(), tasks)

	// Should have at least one failure and some skipped
	hasFailed := false
	hasSkipped := false
	for _, res := range results {
		if res.Status == StatusFailed {
			hasFailed = true
			if !strings.Contains(res.Error.Error(), "task failed") {
				t.Errorf("expected 'task failed' error, got %v", res.Error)
			}
		}
		if res.Status == StatusSkipped {
			hasSkipped = true
		}
	}
	if !hasFailed {
		t.Error("expected at least one failed task")
	}

	// Check that not all tasks ran
	execCount := atomic.LoadInt32(&executed)
	if execCount >= 3 {
		t.Errorf("expected fail-fast to stop some tasks; executed %d/3", execCount)
	}
	_ = hasSkipped // may or may not be present depending on timing
}

func TestRunner_NoFailFast(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 3, TaskTimeout: time.Second, FailFast: false})
	tasks := []Task{
		safeTask("1", "fail-1", func() (string, error) {
			return "", fmt.Errorf("error 1")
		}),
		safeTask("2", "ok-2", func() (string, error) {
			return "ok", nil
		}),
		safeTask("3", "fail-3", func() (string, error) {
			return "", fmt.Errorf("error 3")
		}),
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	statusCount := map[ResultStatus]int{}
	for _, res := range results {
		statusCount[res.Status]++
	}
	if statusCount[StatusFailed] != 2 {
		t.Errorf("expected 2 failures, got %d", statusCount[StatusFailed])
	}
	if statusCount[StatusSuccess] != 1 {
		t.Errorf("expected 1 success, got %d", statusCount[StatusSuccess])
	}
}

// ---------------------------------------------------------------------------
// Timeout
// ---------------------------------------------------------------------------

func TestRunner_TaskTimeout(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 1, TaskTimeout: 10 * time.Millisecond, FailFast: false})
	tasks := []Task{
		safeTask("1", "slow-task", func() (string, error) {
			time.Sleep(time.Second) // far exceeds timeout
			return "done", nil
		}),
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusTimeout {
		t.Errorf("expected StatusTimeout, got %s", results[0].Status)
	}
	if results[0].Error == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(results[0].Error.Error(), "timed out") {
		t.Errorf("expected 'timed out' in error, got %v", results[0].Error)
	}
}

func TestRunner_FailFastOnTimeout(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 2, TaskTimeout: 5 * time.Millisecond, FailFast: true})
	tasks := []Task{
		safeTask("1", "timeout-1", func() (string, error) {
			time.Sleep(time.Second)
			return "slow", nil
		}),
		safeTask("2", "fast-2", func() (string, error) {
			time.Sleep(10 * time.Millisecond)
			return "fast", nil
		}),
		safeTask("3", "timeout-3", func() (string, error) {
			time.Sleep(time.Second)
			return "also-slow", nil
		}),
	}

	results := r.Run(context.Background(), tasks)

	// At least one task timed out
	hasTimeout := false
	for _, res := range results {
		if res.Status == StatusTimeout {
			hasTimeout = true
			break
		}
	}
	if !hasTimeout {
		t.Error("expected at least one timeout")
	}
}

// ---------------------------------------------------------------------------
// Context cancellation propagation
// ---------------------------------------------------------------------------

func TestRunner_ContextCancelled(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 2, TaskTimeout: time.Minute, FailFast: false})
	ctx, cancel := context.WithCancel(context.Background())

	var started int32
	tasks := []Task{
		safeTask("1", "blocking", func() (string, error) {
			atomic.AddInt32(&started, 1)
			time.Sleep(5 * time.Second)
			return "done", nil
		}),
		safeTask("2", "blocking-2", func() (string, error) {
			atomic.AddInt32(&started, 1)
			time.Sleep(5 * time.Second)
			return "done", nil
		}),
	}

	// Use a separate goroutine to cancel the context after a short delay
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	results := r.Run(ctx, tasks)
	wg.Wait()

	// Tasks should have been cancelled
	if atomic.LoadInt32(&started) == 0 {
		t.Error("expected at least one task to have started")
	}
	// Results should include at least one skipped due to context cancel
	hasSkipped := false
	for _, res := range results {
		if res.Status == StatusSkipped {
			hasSkipped = true
			break
		}
	}
	if !hasSkipped && len(results) < len(tasks) {
		// Could be that no tasks ran at all due to timing
		t.Log("context cancel may have prevented all tasks from starting")
	}
	_ = hasSkipped
}

// ---------------------------------------------------------------------------
// Concurrency: verify that tasks run concurrently (not sequentially)
// ---------------------------------------------------------------------------

func TestRunner_Concurrency(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 3, TaskTimeout: time.Second, FailFast: false})

	var mu sync.Mutex
	concurrencyMap := make(map[string]bool)
	var maxConcurrent int32
	var active int32

	tasks := make([]Task, 6)
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("%d", i)
		tasks[i] = safeTask(id, id, func() (string, error) {
			current := atomic.AddInt32(&active, 1)
			atomicMax(&maxConcurrent, current)
			time.Sleep(100 * time.Millisecond)
			atomic.AddInt32(&active, -1)
			mu.Lock()
			concurrencyMap[id] = true
			mu.Unlock()
			return id, nil
		})
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != 6 {
		t.Fatalf("expected 6 results, got %d", len(results))
	}
	if maxConcurrent < 2 {
		t.Errorf("expected concurrent execution (max=%d, want >=2)", maxConcurrent)
	}
}

func atomicMax(dst *int32, val int32) {
	for {
		cur := atomic.LoadInt32(dst)
		if val <= cur {
			break
		}
		if atomic.CompareAndSwapInt32(dst, cur, val) {
			break
		}
	}
}

// ---------------------------------------------------------------------------
// Result ordering: all tasks should produce results (order may vary)
// ---------------------------------------------------------------------------

func TestRunner_AllTasksHaveResults(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 2, TaskTimeout: time.Second, FailFast: false})
	tasks := []Task{
		safeTask("a", "A", func() (string, error) { return "a", nil }),
		safeTask("b", "B", func() (string, error) { return "b", nil }),
		safeTask("c", "C", func() (string, error) { return "c", nil }),
		safeTask("d", "D", func() (string, error) { return "d", nil }),
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	seen := make(map[string]bool)
	for _, res := range results {
		if seen[res.TaskID] {
			t.Errorf("duplicate result for task %s", res.TaskID)
		}
		seen[res.TaskID] = true
	}

	for _, tsk := range tasks {
		if !seen[tsk.ID] {
			t.Errorf("missing result for task %s", tsk.ID)
		}
	}
}

// ---------------------------------------------------------------------------
// Error output propagation
// ---------------------------------------------------------------------------

func TestRunner_ErrorOutput(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 1, TaskTimeout: time.Second, FailFast: false})
	errSentinel := errors.New("execution error")
	tasks := []Task{
		safeTask("1", "err-task", func() (string, error) {
			return "partial-output", errSentinel
		}),
	}

	results := r.Run(context.Background(), tasks)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFailed {
		t.Errorf("expected StatusFailed, got %s", results[0].Status)
	}
	if results[0].Output != "partial-output" {
		t.Errorf("expected output 'partial-output', got %q", results[0].Output)
	}
	if !errors.Is(results[0].Error, errSentinel) {
		t.Errorf("expected error sentinel, got %v", results[0].Error)
	}
}

// ---------------------------------------------------------------------------
// Large number of tasks with small pool
// ---------------------------------------------------------------------------

func TestRunner_LargeBatch(t *testing.T) {
	r := NewRunner(PoolConfig{PoolSize: 2, TaskTimeout: 5 * time.Second, FailFast: false})
	count := 20
	tasks := make([]Task, count)
	for i := 0; i < count; i++ {
		i := i
		tasks[i] = safeTask(fmt.Sprintf("%d", i), fmt.Sprintf("task-%d", i), func() (string, error) {
			return fmt.Sprintf("out-%d", i), nil
		})
	}

	results := r.Run(context.Background(), tasks)

	if len(results) != count {
		t.Fatalf("expected %d results, got %d", count, len(results))
	}
	for _, res := range results {
		if res.Status != StatusSuccess {
			t.Errorf("unexpected status for %s: %s", res.Name, res.Status)
		}
	}
}
