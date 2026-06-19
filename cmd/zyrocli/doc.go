package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/secko/zyrocli/internal/doc"
	"github.com/spf13/cobra"
)

var (
	docDir  string
	docOnly string
)

var docCmd = &cobra.Command{
	Use:   "doc <subcommand>",
	Short: "Manage project documentation index and export",
	Long: `Manage documentation tooling: index, search, sync, and export.

Subcommands:
  sync    Generate index, export ARCHITECTURE.md and CHANGELOG.md

The doc-index is stored at .zyro/doc-index.yaml and serves as the
local bridge between Engram's persistent memory and filesystem docs.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var docSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Generate doc index and export architecture documentation",
	Long: `Run the full doc sync cycle:

  1. GenerateIndex — build doc index from conventions.yaml and active changes
  2. SaveIndex — write .zyro/doc-index.yaml
  3. Export — render ARCHITECTURE.md and CHANGELOG.md from templates
  4. UpdateGraph — compare with previous state and persist if significant

The sync is idempotent; running it multiple times produces the same result.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine project root
		projectRoot := docDir
		if projectRoot == "" {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("doc sync: getting working directory: %w", err)
			}
			projectRoot = cwd
		}

		// Check if .zyro/conventions.yaml exists
		convPath := filepath.Join(projectRoot, ".zyro", "conventions.yaml")
		if _, err := os.Stat(convPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("doc sync: .zyro/conventions.yaml not found in %s\nRun this command from the project root", projectRoot)
			}
			return fmt.Errorf("doc sync: checking conventions: %w", err)
		}

		cmd.Printf("📚 Syncing documentation for %s...\n", projectRoot)

		idx, err := doc.Sync(projectRoot)
		if err != nil {
			// Graph update errors are non-fatal
			return fmt.Errorf("doc sync: %w", err)
		}

		cmd.Printf("✓ Index generated: %d entries\n", len(idx.Entries))
		cmd.Printf("✓ Index saved: .zyro/doc-index.yaml\n")
		cmd.Printf("✓ ARCHITECTURE.md generated\n")
		cmd.Printf("✓ CHANGELOG.md generated\n")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(docCmd)

	docCmd.AddCommand(docSyncCmd)
	docSyncCmd.Flags().StringVarP(&docDir, "dir", "d", "", "project root directory (default: current directory)")
}
