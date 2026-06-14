package scheduler

import (
	"context"
	"fmt"
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

	for _, phase := range s.phases {
		phaseCtx, cancel := context.WithTimeout(ctx, s.config.PhaseTimeout)

		result, err := phase.Run(phaseCtx, s.config)
		cancel()

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

		results = append(results, result)

		// If the phase itself reported failure or abort, stop execution
		if result.Status == StatusFail || result.Status == StatusAbort {
			return results, fmt.Errorf("scheduler: phase %s ended with status %s", phaseName, result.Status)
		}

		// Mandatory approval gate — always prompts for human validation
		approved, err := PromptApproval(result.Phase, result.Summary)
		if err != nil {
			return results, fmt.Errorf("scheduler: approval error: %w", err)
		}
		if !approved {
			results = append(results, &Result{
				Phase:   result.Phase,
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
			return p.Run(phaseCtx, s.config)
		}
	}
	return nil, fmt.Errorf("scheduler: phase %s not found", phase)
}
