package main

import (
	"context"
	"fmt"
	"os"

	helix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/taskcontext"
	"github.com/spf13/cobra"
)

var (
	contextFormat string
)

// contextCmd fetches and displays task context from HelixDB.
//
// Deprecated: use "task_context" MCP tool via OpenCode instead.
var contextCmd = &cobra.Command{
	Use:   "context <task-id>",
	Short: "Get task context from HelixDB (DEPRECATED)",
	Long: `Fetch and display full context for a task: skills, code nodes,
documents, and patterns from HelixDB.

DEPRECATED: use the "task_context" MCP tool inside OpenCode instead.
This command is kept for backward compatibility.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Deprecation warning before any existing logic.
		fmt.Fprintln(cmd.ErrOrStderr(), "⚠ DEPRECATED: use MCP tool task_context via OpenCode")

		// --- existing logic below ---

		taskID := args[0]

		// Build HelixDB client.
		helixURL := os.Getenv("HELIX_URL")
		if helixURL == "" {
			helixURL = "http://localhost:6969"
		}
		opts := []helix.Option{helix.WithBaseURL(helixURL)}
		if pid := os.Getenv("HELIX_PROJECT_ID"); pid != "" {
			opts = append(opts, helix.WithProjectID(pid))
		}

		client, err := helix.NewClient(context.Background(), opts...)
		if err != nil {
			return fmt.Errorf("context: HelixDB connection failed: %w", err)
		}
		defer client.Close()

		// Auto-start HelixDB if needed
		if err := client.EnsureStarted(context.Background()); err != nil {
			return fmt.Errorf("cannot connect to HelixDB: %w", err)
		}

		// Parse task ID.
		var id uint64
		if _, err := fmt.Sscanf(taskID, "%d", &id); err != nil {
			return fmt.Errorf("context: invalid task ID %q: %w", taskID, err)
		}

		tc, err := taskcontext.GetTaskContext(context.Background(), client, id)
		if err != nil {
			return fmt.Errorf("context: task not found: %w", err)
		}

		switch contextFormat {
		case "json":
			s, err := tc.FormatJSON()
			if err != nil {
				return fmt.Errorf("context: format json: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), s)
		case "prompt":
			fmt.Fprintln(cmd.OutOrStdout(), tc.FormatPrompt())
		default: // text
			fmt.Fprintln(cmd.OutOrStdout(), tc.FormatText())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().StringVarP(&contextFormat, "format", "f", "text", "output format: text, json, or prompt")
}
