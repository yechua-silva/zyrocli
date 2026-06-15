package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/secko/zyrocli/internal/handoff"
	"github.com/secko/zyrocli/internal/scaffold"
	"github.com/secko/zyrocli/internal/scheduler"
	"github.com/spf13/cobra"
)

var runPhase string

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Execute SDD pipeline (F0→F1→F2→F3→F4)",
	Long: `Execute the 5-phase SDD pipeline (F0→F1→F2→F3→F4) sequentially
with human-validation approval gates after each phase. All phases require
explicit approval before proceeding — there is no automatic mode.

Flags:
  --phase F0   Run a single phase only (F0, F1, F2, F3, or F4)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if project is already initialized
		projectDir := "." // current directory, or read from handoff.yaml project.name
		if _, err := os.Stat("handoff.yaml"); err == nil {
			payload, err := handoff.Parse("handoff.yaml")
			if err == nil && payload.Project.Name != "" {
				projectDir = payload.Project.Name
			}
		}

		state, err := scaffold.ReadState(projectDir)
		if err != nil {
			return fmt.Errorf("run: checking project state: %w", err)
		}

		if state != nil && state.Initialized {
			cmd.Printf("✓ Proyecto ya inicializado: %s\n", state.ProjectName)
			cmd.Println("  Abriendo OpenCode...")

			openCmd := exec.Command("opencode", projectDir)
			openCmd.Stdin = os.Stdin
			openCmd.Stdout = os.Stdout
			openCmd.Stderr = os.Stderr
			return openCmd.Run()
		}

		// Check handoff.yaml exists
		if _, err := os.Stat("handoff.yaml"); os.IsNotExist(err) {
			return fmt.Errorf("run: handoff.yaml not found in current directory\nRun 'zyrocli init <file>' first")
		}

		// Load config from handoff.yaml
		cfg, err := scheduler.LoadConfig("handoff.yaml")
		if err != nil {
			return fmt.Errorf("run: %w", err)
		}

		// Build phase runners in order (F0→F4)
		runners := []scheduler.PhaseRunner{
			&scheduler.F0Runner{},
			&scheduler.F1Runner{},
			&scheduler.F2Runner{},
			&scheduler.F3Runner{},
			&scheduler.F4Runner{},
		}

		s := scheduler.NewScheduler(cfg, runners)

		ctx := context.Background()
		var results []*scheduler.Result

		if runPhase != "" {
			// Validate phase
			phase := scheduler.Phase(runPhase)
			valid := false
			for _, p := range scheduler.AllPhases {
				if p == phase {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("run: invalid phase %q, must be one of: F0, F1, F2, F3, F4", runPhase)
			}

			cmd.Printf("▶ Running phase %s...\n", phase)
			result, err := s.RunPhase(ctx, phase)
			if err != nil {
				return fmt.Errorf("run: %w", err)
			}
			results = append(results, result)
		} else {
			cmd.Println("▶ Iniciando el proceso de desarrollo (F0 → F1 → F2 → F3 → F4)")

			var err error
			results, err = s.Run(ctx)
			if err != nil {
				return fmt.Errorf("run: %w", err)
			}
		}

		// Print summary
		cmd.Println("\n📋 Results:")
		for _, r := range results {
			status := "✓"
			if r.Status == scheduler.StatusFail {
				status = "✗"
			} else if r.Status == scheduler.StatusAbort {
				status = "⊘"
			}
			cmd.Printf("  %s %s: %s (%s)\n", status, r.Phase, r.Summary, r.Status)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().StringVarP(&runPhase, "phase", "p", "", "run a single phase only (F0, F1, F2, F3, F4)")
}
