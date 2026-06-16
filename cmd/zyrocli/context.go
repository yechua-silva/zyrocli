package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	helix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/taskcontext"
	"github.com/spf13/cobra"
)

var contextFormat string

// contextCmd fetches and displays task or project context from HelixDB.
var contextCmd = &cobra.Command{
	Use:   "context <id>",
	Short: "Get context from HelixDB (task or project)",
	Long: `Fetch and display full context from HelixDB.

If the ID is numeric, fetches task context (skills, code, docs, patterns).
If the ID is a project name, fetches project context.

Output formats:
  --format text    human-readable summary (default)
  --format json    structured JSON
  --format prompt  ready-to-inject prompt for subagents`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

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

		if err := client.EnsureStarted(context.Background()); err != nil {
			return fmt.Errorf("cannot connect to HelixDB: %w", err)
		}

		// Try numeric task ID first
		if id, err := strconv.ParseUint(query, 10, 64); err == nil {
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
			default:
				fmt.Fprintln(cmd.OutOrStdout(), tc.FormatText())
			}
			return nil
		}

		// Try project name
		projectNodes, err := client.TextSearch(context.Background(), "Project", "name", query, 1)
		if err != nil || len(projectNodes) == 0 {
			return fmt.Errorf("context: no task or project found for %q", query)
		}

		projectID := projectNodes[0].ID
		patterns, _ := client.GetOutgoing(context.Background(), projectID, "HAS_PATTERN")
		libs, _ := client.GetOutgoing(context.Background(), projectID, "USES_LIB")
		skills, _ := client.GetOutgoing(context.Background(), projectID, "REQUIRES_SKILL")
		specs, _ := client.TextSearch(context.Background(), "Spec", "project_id", fmt.Sprintf("%d", projectID), 1)
		tasks, _ := client.TextSearch(context.Background(), "Task", "project_id", fmt.Sprintf("%d", projectID), 20)

		result := map[string]any{
			"project_id": projectID,
			"patterns":   nodeListToMap(patterns),
			"libraries":  nodeListToMap(libs),
			"skills":     nodeListToMap(skills),
			"specs":      nodeListToMap(specs),
			"tasks":      nodeListToMap(tasks),
		}

		switch contextFormat {
		case "json":
			fmt.Fprintf(cmd.OutOrStdout(), "Project context for %s (id=%d)\n", query, projectID)
		case "prompt":
			fmt.Fprintf(cmd.OutOrStdout(), "# Project Context: %s\n\n", query)
			fmt.Fprintf(cmd.OutOrStdout(), "Patterns (%d):\n", len(patterns))
			for _, n := range patterns { fmt.Fprintf(cmd.OutOrStdout(), "- %v\n", n.Properties["name"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "Libraries (%d):\n", len(libs))
			for _, n := range libs { fmt.Fprintf(cmd.OutOrStdout(), "- %v (version %v)\n", n.Properties["name"], n.Properties["version"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "Skills (%d):\n", len(skills))
			for _, n := range skills { fmt.Fprintf(cmd.OutOrStdout(), "- %v\n", n.Properties["name"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "Tasks (%d):\n", len(tasks))
			for _, n := range tasks { fmt.Fprintf(cmd.OutOrStdout(), "- %v [%v]\n", n.Properties["name"], n.Properties["status"]) }
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "===== Project Context: %s (id=%d) =====\n\n", query, projectID)
			fmt.Fprintf(cmd.OutOrStdout(), "Patterns (%d):\n", len(patterns))
			for _, n := range patterns { fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", n.Properties["name"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "\nLibraries (%d):\n", len(libs))
			for _, n := range libs { fmt.Fprintf(cmd.OutOrStdout(), "  - %s v%s\n", n.Properties["name"], n.Properties["version"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "\nSkills (%d):\n", len(skills))
			for _, n := range skills { fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", n.Properties["name"]) }
			fmt.Fprintf(cmd.OutOrStdout(), "\nTasks (%d):\n", len(tasks))
			for _, n := range tasks { fmt.Fprintf(cmd.OutOrStdout(), "  - %s [%s]\n", n.Properties["name"], n.Properties["status"]) }
		}

		_ = result
		return nil
	},
}

func nodeListToMap(nodes []*helix.Node) []map[string]any {
	result := make([]map[string]any, len(nodes))
	for i, n := range nodes {
		m := map[string]any{"id": n.ID, "type": n.Label}
		for k, v := range n.Properties {
			m[k] = v
		}
		result[i] = m
	}
	return result
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.Flags().StringVarP(&contextFormat, "format", "f", "text", "output format: text, json, or prompt")
}
