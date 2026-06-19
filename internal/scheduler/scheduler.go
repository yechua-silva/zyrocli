package scheduler

import (
	"context"
	"fmt"

	"github.com/secko/zyrocli/internal/boomerang"
)

// Scheduler manages the execution of SDD phases as a sequential DAG
// with mandatory approval gates between each phase.
type Scheduler struct {
	phases []PhaseRunner
	config *Config
}

// NewScheduler creates a new Scheduler with the given phase runners and config.
// Approval prompts always run after each phase — there is no auto mode.
func NewScheduler(cfg *Config, runners []PhaseRunner) *Scheduler {
	return &Scheduler{
		phases: runners,
		config: cfg,
	}
}

// Run executes all phases sequentially with approval gates.
// Each phase runs within its own context with the configured PhaseTimeout.
// If a phase fails or is aborted, remaining phases are skipped.
func (s *Scheduler) Run(ctx context.Context) ([]*Result, error) {
	var results []*Result
	validator := NewHarnessValidator(AllPhases)

	for _, phase := range s.phases {
		validator.SetCurrent(phase.Name())

		phaseCtx, cancel := context.WithTimeout(ctx, s.config.PhaseTimeout)

		// MEMORIA CAUSAL: inyectar contexto antes de ejecutar la fase
		// BUG #5: en modo Boomerang, PrePhase se salta porque Boomerang.RunPhase
		// arranca con MemoryStep() que ya hace recall — de lo contrario hay doble recall.
		memoryContext := ""
		if s.config.MemoryHooks != nil && s.config.Boomerang == nil {
			mc, err := s.config.MemoryHooks.PrePhase(phaseCtx, phase.Name(), s.config.Module)
			if err == nil && mc != "" {
				memoryContext = mc
			}
		}

		var result *Result
		var err error

		// Si hay Boomerang, usarlo (ciclo completo de 6 pasos)
		if s.config.Boomerang != nil {
			boomerangResult, boomerErr := s.config.Boomerang.RunPhase(phaseCtx, boomerang.PhaseConfig{
				Phase:    string(phase.Name()),
				TaskDesc: s.config.Module,
			})
			if boomerErr != nil {
				results = append(results, &Result{
					Phase: phase.Name(), Status: StatusFail, Summary: boomerErr.Error(),
				})
				cancel()
				return results, fmt.Errorf("boomerang phase %s: %w", phase.Name(), boomerErr)
			}
			result = &Result{
				Phase:   Phase(boomerangResult.Phase),
				Status:  StatusSuccess,
				Summary: fmt.Sprintf("Boomerang: %d tasks, quality=%v, facts=%d",
					boomerangResult.TasksPlanned, boomerangResult.QualityOK, boomerangResult.FactsSaved),
			}
		} else {
			result, err = phase.Run(phaseCtx, s.config)
		}
		cancel()

		// MEMORIA CAUSAL: guardar hechos después de ejecutar la fase
		if s.config.MemoryHooks != nil && result != nil {
			_ = s.config.MemoryHooks.PostPhase(phaseCtx, phase.Name(), result.Summary)
		}

		// Determine the phase name from the runner or result
		phaseName := phase.Name()
		if result != nil && result.Phase != "" {
			phaseName = result.Phase
		}

		if err != nil {
			results = append(results, &Result{
				Phase:   phaseName,
				Status:  StatusFail,
				Summary: err.Error(),
			})
			return results, fmt.Errorf("scheduler: phase %s failed: %w", phaseName, err)
		}

		result.MemoryContext = memoryContext
		results = append(results, result)

		// If the phase itself reported failure or abort, stop execution
		if result.Status == StatusFail || result.Status == StatusAbort {
			return results, fmt.Errorf("scheduler: phase %s ended with status %s", phaseName, result.Status)
		}

		// Mandatory approval gate
		approved, err := ApprovalGate(result.Phase, result.Summary)
		if err != nil {
			return results, fmt.Errorf("scheduler: approval error: %w", err)
		}

		// HarnessValidator enforces human validation
		if err := validator.ValidateTransition(phaseName, phaseName, approved); err != nil {
			return results, fmt.Errorf("scheduler: harness validation: %w", err)
		}

		if !approved {
			results = append(results, &Result{
				Phase:   phaseName,
				Status:  StatusAbort,
				Summary: "rejected by user",
			})
			return results, nil
		}
	}

	return results, nil
}

// RunPhase executes a single phase by its Phase identifier.
// Only the matching runner is invoked; no approval gates apply.
func (s *Scheduler) RunPhase(ctx context.Context, phase Phase) (*Result, error) {
	for _, p := range s.phases {
		if p.Name() == phase {
			phaseCtx, cancel := context.WithTimeout(ctx, s.config.PhaseTimeout)
			defer cancel()

			// MEMORIA CAUSAL: inyectar contexto antes de ejecutar
			if s.config.MemoryHooks != nil {
				mc, err := s.config.MemoryHooks.PrePhase(phaseCtx, phase, s.config.Module)
				if err == nil && mc != "" {
					// TODO: inyectar mc en config o contexto para que phase.Run la use
					_ = mc
				}
			}

			return p.Run(phaseCtx, s.config)
		}
	}
	return nil, fmt.Errorf("scheduler: phase %s not found", phase)
}

// HarnessValidator enforces that phase transitions require human approval.
// It acts as a deterministic gate between macro-phases: without explicit
// human validation, no transition is allowed.
type HarnessValidator struct {
	currentPhase Phase
	allPhases    []Phase
}

// NewHarnessValidator creates a validator for the given phase sequence.
func NewHarnessValidator(phases []Phase) *HarnessValidator {
	return &HarnessValidator{
		allPhases: phases,
	}
}

// ValidateTransition checks that moving from `from` to `to` is valid:
//   - `approved` MUST be true (human validation is required)
//   - The transition MUST follow the sequential DAG order
func (h *HarnessValidator) ValidateTransition(from Phase, to Phase, approved bool) error {
	if !approved {
		return fmt.Errorf("harness: transition %s→%s blocked: human approval required", from, to)
	}

	fromIdx := -1
	toIdx := -1
	for i, p := range h.allPhases {
		if p == from {
			fromIdx = i
		}
		if p == to {
			toIdx = i
		}
	}
	if fromIdx == -1 {
		return fmt.Errorf("harness: unknown phase %q in transition %s→%s", from, from, to)
	}
	// Allow same-phase validation (phase just completed)
	if toIdx == -1 || toIdx == fromIdx {
		return nil
	}
	if toIdx != fromIdx+1 {
		return fmt.Errorf("harness: invalid transition %s→%s: must be sequential", from, to)
	}
	return nil
}

// CurrentPhase returns the currently active phase.
func (h *HarnessValidator) CurrentPhase() Phase {
	return h.currentPhase
}

// NextPhase returns the next phase in the sequence, or empty if at the end.
func (h *HarnessValidator) NextPhase() Phase {
	for i, p := range h.allPhases {
		if p == h.currentPhase && i+1 < len(h.allPhases) {
			return h.allPhases[i+1]
		}
	}
	return ""
}

// SetCurrent sets the current phase for the validator.
func (h *HarnessValidator) SetCurrent(phase Phase) {
	h.currentPhase = phase
}
