package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/handoff"
	"github.com/secko/zyrocli/internal/scaffold"
	"github.com/spf13/cobra"
)

var (
	scaffoldFlag bool
	opencodeFlag bool
)

var initCmd = &cobra.Command{
	Use:   "init <file>",
	Short: "Initialize project from handoff contract",
	Long: `Parse and validate a handoff.yaml v2.0 contract.
Accepts a file path or "-" to read from stdin.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]

		payload, err := handoff.Parse(path)
		if err != nil {
			return fmt.Errorf("init: %w", err)
		}

		if err := handoff.Validate(payload); err != nil {
			return fmt.Errorf("init: validation failed:\n%v", err)
		}

		// --opencode requires --scaffold
		if opencodeFlag && !scaffoldFlag {
			return fmt.Errorf("init: flag --opencode requires --scaffold")
		}

		if scaffoldFlag {
			// Auto-start HelixDB so the project can use it immediately
			helixClient, err := helix.NewClient(context.Background())
			if err == nil {
				if err := helixClient.EnsureStarted(context.Background()); err != nil {
					cmd.PrintErrln("⚠ HelixDB not available (project will work, but MCP tools need it)")
				} else {
					// Create the project node in HelixDB
					if projectNode, err := helixClient.CreateNode(context.Background(), "Project", map[string]any{
						"name":    payload.Project.Name,
						"repo":    payload.Project.Repository,
						"problem": payload.ValidatedIdea.Problem,
					}); err == nil {
						cmd.Printf("  HelixDB project node: %d\n", projectNode.ID)
						_ = helixClient.Close()
					}
				}
			}

			// Read raw handoff content for template reference
			var rawBytes []byte
			if path == "-" {
				// stdin was already consumed by Parse; no raw content available
				rawBytes = []byte{}
			} else {
				rawBytes, err = os.ReadFile(path)
				if err != nil {
					return fmt.Errorf("init: reading handoff: %w", err)
				}
			}

			// Map handoff payload fields to scaffold config
			cfg := scaffold.Config{
				ProjectName:     payload.Project.Name,
				Language:        payload.Project.Language,
				Module:          payload.Project.Repository,
				Problem:         payload.ValidatedIdea.Problem,
				SuccessCriteria: payload.UserStory.Acceptance,
				ScaffoldDir:     payload.Project.Name,
				LaunchOpenCode:  opencodeFlag,
				RawHandoff:      string(rawBytes),
				Version:         payload.Version,
				Source:          payload.Source.System,
			}

			// Check opencode availability in the CLI layer
			if opencodeFlag {
				if _, err := exec.LookPath("opencode"); err != nil {
					cmd.PrintErrln("⚠ opencode not found in PATH. Install it to use --opencode. Scaffold still created.")
					cfg.LaunchOpenCode = false
				}
			}

			result, err := scaffold.Run(cfg)
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}

			cmd.Printf("✓ Project scaffolded at %s/\n", result.TargetDir)
			cmd.Printf("  Files created: %d\n", result.FilesCreated)

			if opencodeFlag && cfg.LaunchOpenCode {
				openCmd := exec.Command("opencode", result.TargetDir)
				openCmd.Stdin = os.Stdin
				openCmd.Stdout = os.Stdout
				openCmd.Stderr = os.Stderr
				_ = openCmd.Run()
				cmd.Println("OpenCode session ended. Happy coding!")
			}
		} else {
			summary, _ := json.MarshalIndent(payload, "", "  ")
			cmd.Printf("OK — handoff contract v2.0 parsed successfully:\n%s\n", string(summary))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&scaffoldFlag, "scaffold", "s", false, "generate project scaffold from handoff contract")
	initCmd.Flags().BoolVarP(&opencodeFlag, "opencode", "o", false, "launch OpenCode in scaffolded project (requires --scaffold)")
}
