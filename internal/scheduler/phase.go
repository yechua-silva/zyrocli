package scheduler

import (
	"context"
	"time"

	"github.com/secko/zyrocli/internal/boomerang"
)

// Phase identifies an SDD phase.
type Phase string

const (
	PhaseF0 Phase = "F0"
	PhaseF1 Phase = "F1"
	PhaseF2 Phase = "F2"
	PhaseF3 Phase = "F3"
	PhaseF4 Phase = "F4"
)

// AllPhases is the ordered list of all phases.
var AllPhases = []Phase{PhaseF0, PhaseF1, PhaseF2, PhaseF3, PhaseF4}

// Status represents a phase execution result status.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFail    Status = "fail"
	StatusAbort   Status = "abort"
)

// Config holds scheduler configuration from handoff.yaml governance.
type Config struct {
	Mode         string                      // governance.mode
	Module       string                      // governance.module
	GoVersion    string                      // governance.go_version
	MaxTasks     int                         // limits.max_tasks
	MaxLines     int                         // limits.max_lines
	MaxLoops     int                         // limits.max_loops
	PhaseTimeout time.Duration               // parsed from limits.phase_timeout
	MemoryHooks  *MemoryHooks                // hooks de memoria causal (T-4.9)
	Boomerang    *boomerang.BoomerangOrchestrator // orquestador Boomerang (T-5.12)
}

// Result holds the outcome of a single phase execution.
type Result struct {
	Phase         Phase
	Status        Status
	Summary       string
	Error         error
	MemoryContext string // contexto de memoria causal inyectado (T-4.9)
}

// PhaseRunner is the interface each phase must implement.
type PhaseRunner interface {
	// Run executes the phase and returns a Result.
	Run(ctx context.Context, cfg *Config) (*Result, error)
	// Name returns the phase identifier.
	Name() Phase
}

// MacroPhaseRunner wraps a function as a PhaseRunner with optional validator.
type MacroPhaseRunner struct {
	fn        func(ctx context.Context, cfg *Config) (*Result, error)
	phase     Phase
	validator *HarnessValidator
}

// NewMacroPhaseRunner creates a MacroPhaseRunner for the given phase and function.
func NewMacroPhaseRunner(phase Phase, fn func(ctx context.Context, cfg *Config) (*Result, error)) *MacroPhaseRunner {
	return &MacroPhaseRunner{
		fn:    fn,
		phase: phase,
	}
}

// WithValidator attaches a harness validator to this runner.
func (m *MacroPhaseRunner) WithValidator(v *HarnessValidator) *MacroPhaseRunner {
	m.validator = v
	return m
}

// Run executes the wrapped function and validates the transition.
func (m *MacroPhaseRunner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: m.phase, Status: StatusFail, Summary: "cancelled"}, ctx.Err()
	default:
	}

	result, err := m.fn(ctx, cfg)
	if err != nil {
		return result, err
	}

	if m.validator != nil {
		m.validator.SetCurrent(m.phase)
		if err := m.validator.ValidateTransition(m.phase, m.phase, result.Status == StatusSuccess); err != nil {
			return result, err
		}
	}

	return result, nil
}

// Name returns the phase identifier.
func (m *MacroPhaseRunner) Name() Phase {
	return m.phase
}

// DefaultValidator returns a HarnessValidator for all phases.
func DefaultValidator() *HarnessValidator {
	return NewHarnessValidator(AllPhases)
}

// Validator is the interface for phase transition validation.
type Validator interface {
	SetCurrent(phase Phase)
	ValidateTransition(from, to Phase, approved bool) error
	NextPhase() string
}
