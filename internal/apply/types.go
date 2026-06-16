package apply

import "time"

// ResultStatus represents the outcome status of a task execution.
type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusFailed  ResultStatus = "failed"
	StatusTimeout ResultStatus = "timeout"
	StatusSkipped ResultStatus = "skipped"
)

// Task represents a unit of work to be executed by the Runner.
type Task struct {
	ID      string
	Name    string
	Execute func() (string, error)
}

// Result carries the outcome of a single task execution.
type Result struct {
	TaskID string
	Name   string
	Status ResultStatus
	Output string
	Error  error
}

// PoolConfig defines the configuration for the task runner pool.
type PoolConfig struct {
	PoolSize    int
	TaskTimeout time.Duration
	FailFast    bool
}

// DefaultPoolConfig returns a PoolConfig with sensible defaults:
// pool size 5, task timeout 10 minutes, fail-fast enabled.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		PoolSize:    5,
		TaskTimeout: 10 * time.Minute,
		FailFast:    true,
	}
}

// Validate checks that the pool config has sane values and returns
// defaults for any zero-valued fields.
func (c PoolConfig) Validate() PoolConfig {
	if c.PoolSize <= 0 {
		c.PoolSize = 5
	}
	if c.TaskTimeout <= 0 {
		c.TaskTimeout = 10 * time.Minute
	}
	return c
}
