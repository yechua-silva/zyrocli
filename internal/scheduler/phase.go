package scheduler

import (
	"context"
	"time"
)

// Phase identifies an SDD phase.
type Phase string

const (
	PhaseF1 Phase = "F1"
	PhaseF2 Phase = "F2"
	PhaseF3 Phase = "F3"
	PhaseF4 Phase = "F4"
)

// AllPhases is the ordered list of all phases.
var AllPhases = []Phase{PhaseF1, PhaseF2, PhaseF3, PhaseF4}

// Status represents a phase execution result status.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFail    Status = "fail"
	StatusAbort   Status = "abort"
)

// Config holds scheduler configuration from handoff.yaml governance.
type Config struct {
	Mode         string        // governance.mode
	Module       string        // governance.module
	GoVersion    string        // governance.go_version
	MaxTasks     int           // limits.max_tasks
	MaxLines     int           // limits.max_lines
	MaxLoops     int           // limits.max_loops
	PhaseTimeout time.Duration // parsed from limits.phase_timeout
}

// Result holds the outcome of a single phase execution.
type Result struct {
	Phase   Phase
	Status  Status
	Summary string
	Error   error
}

// PhaseRunner is the interface each phase must implement.
type PhaseRunner interface {
	// Run executes the phase and returns a Result.
	Run(ctx context.Context, cfg *Config) (*Result, error)
	// Name returns the phase identifier.
	Name() Phase
}
