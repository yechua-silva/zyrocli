package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/handoff"
	"github.com/secko/zyrocli/internal/scaffold"
	"github.com/spf13/cobra"
)

var dryRun bool

var initCmd = &cobra.Command{
	Use:   "init <handoff.yaml>",
	Short: "Initialize project from handoff and open in OpenCode",
	Long: `Parse a handoff.yaml v2.0 contract, scaffold the project,
auto-start HelixDB, create the project node, and launch OpenCode.

Usage:
  zyrocli init todo-handoff.yaml

Use --dry-run to just parse and validate without creating anything.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		// ── Parse and validate ──────────────────────────────────────
		payload, err := handoff.Parse(path)
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}
		if err := handoff.Validate(payload); err != nil {
			return fmt.Errorf("init: validation failed:\n%v", err)
		}

		// ── Dry-run mode: just validate ─────────────────────────────
		if dryRun {
			cmd.Printf("✓ handoff valid for project: %s (%s)\n", payload.Project.Name, payload.Project.Language)
			return nil
		}

		// ── Auto-start HelixDB ──────────────────────────────────────
		helixClient, err := helix.NewClient(context.Background())
		if err == nil {
			if err := helixClient.EnsureStarted(context.Background()); err != nil {
				cmd.PrintErrln("⚠ HelixDB not available (project will work, but MCP tools need it)")
			} else {
				if projectNode, err := helixClient.CreateNode(context.Background(), "Project", map[string]any{
					"name":    payload.Project.Name,
					"repo":    payload.Project.Repository,
					"problem": payload.ValidatedIdea.Problem,
				}); err == nil {
					cmd.Printf("  HelixDB project node: %d\n", projectNode.ID)
				}
				_ = helixClient.Close()
			}
		}

		// ── Read raw handoff for template ───────────────────────────
		var rawBytes []byte
		if path == "-" {
			rawBytes = []byte{}
		} else {
			rawBytes, err = os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("init: reading handoff: %w", err)
			}
		}

		// ── Scaffold project ────────────────────────────────────────
		cfg := scaffold.Config{
			ProjectName:     payload.Project.Name,
			Language:        payload.Project.Language,
			Module:          payload.Project.Repository,
			Problem:         payload.ValidatedIdea.Problem,
			SuccessCriteria: payload.UserStory.Acceptance,
			ScaffoldDir:     payload.Project.Name,
			LaunchOpenCode:  true,
			RawHandoff:      string(rawBytes),
			Version:         payload.Version,
			Source:          payload.Source.System,
		}

		// Check opencode availability
		if _, err := exec.LookPath("opencode"); err != nil {
			cmd.PrintErrln("⚠ opencode not found in PATH. Install it separately. Scaffold still created.")
			cfg.LaunchOpenCode = false
		}

		result, err := scaffold.Run(cfg)
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		cmd.Printf("✓ Project scaffolded at %s/\n", result.TargetDir)
		cmd.Printf("  Files created: %d\n", result.FilesCreated)

		// ── Launch OpenCode ─────────────────────────────────────────
		if cfg.LaunchOpenCode {
			cmd.Printf("  Opening OpenCode in %s/ ...\n", result.TargetDir)
			openCmd := exec.Command("opencode", result.TargetDir)
			openCmd.Stdin = os.Stdin
			openCmd.Stdout = os.Stdout
			openCmd.Stderr = os.Stderr
			_ = openCmd.Run()
			cmd.Println("OpenCode session ended. Happy coding!")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "only parse and validate, do not create anything")
}
