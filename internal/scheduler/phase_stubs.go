package scheduler

import (
	"context"
	"fmt"

	"github.com/secko/zyrocli/internal/handoff"
)

// F1Runner implements PhaseRunner for F1: Planificación.
type F1Runner struct{}

// Run executes the F1 stub: parses handoff.yaml, prints a contract summary,
// and calls a Skill Advisor placeholder.
func (r *F1Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: "timeout"}, ctx.Err()
	default:
	}

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("parse failed: %v", err)}, nil
	}

	summary := fmt.Sprintf("Project: %s | Version: %s", payload.Project.Name, payload.Version)
	fmt.Printf("  ✓ Contract parsed: %s\n", summary)
	fmt.Println("  ℹ F1: Skill Advisor placeholder (not yet implemented)")

	return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: summary}, nil
}

// Name returns the phase identifier.
func (r *F1Runner) Name() Phase { return PhaseF1 }

// F2Runner implements PhaseRunner for F2: Especificación.
type F2Runner struct{}

// Run executes the F2 stub: prints a banner and returns success.
func (r *F2Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "timeout"}, ctx.Err()
	default:
	}

	fmt.Println("  ℹ F2: Especificación — not yet implemented")
	return &Result{Phase: PhaseF2, Status: StatusSuccess, Summary: "F2 stub complete"}, nil
}

// Name returns the phase identifier.
func (r *F2Runner) Name() Phase { return PhaseF2 }

// F3Runner implements PhaseRunner for F3: Implementación.
type F3Runner struct{}

// Run executes the F3 stub: prints a banner and returns success.
func (r *F3Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "timeout"}, ctx.Err()
	default:
	}

	fmt.Println("  ℹ F3: Implementación — not yet implemented")
	return &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: "F3 stub complete"}, nil
}

// Name returns the phase identifier.
func (r *F3Runner) Name() Phase { return PhaseF3 }

// F4Runner implements PhaseRunner for F4: Cierre.
type F4Runner struct{}

// Run executes the F4 stub: prints a banner and returns success.
func (r *F4Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: "timeout"}, ctx.Err()
	default:
	}

	fmt.Println("  ℹ F4: Cierre — not yet implemented")
	return &Result{Phase: PhaseF4, Status: StatusSuccess, Summary: "F4 stub complete"}, nil
}

// Name returns the phase identifier.
func (r *F4Runner) Name() Phase { return PhaseF4 }
