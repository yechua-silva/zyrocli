package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	helix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/git"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks and link code changes",
}

var taskCreateCmd = &cobra.Command{
	Use:   "create [description]",
	Short: "Create a new task in HelixDB",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			return fmt.Errorf("connecting to HelixDB: %w", err)
		}
		defer client.Close()

		id, err := client.CreateNode(ctx, "Task", map[string]interface{}{
			"description": args[0],
			"status":      "active",
			"created_at":  time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			return fmt.Errorf("creating task: %w", err)
		}

		cmd.Printf("✓ Task created: %s (id: %d)\n", args[0], id)
		return nil
	},
}

var taskLinkCmd = &cobra.Command{
	Use:   "link [task-id]",
	Short: "Link code changes to a task via git diff",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ref, _ := cmd.Flags().GetString("ref")
		dir, _ := cmd.Flags().GetString("dir")

		// Parse task ID
		var taskID uint64
		if _, err := fmt.Sscanf(args[0], "%d", &taskID); err != nil {
			return fmt.Errorf("invalid task-id: %s", args[0])
		}

		// Get changed files
		files, err := git.ChangedFiles(ref, dir)
		if err != nil {
			return fmt.Errorf("git diff: %w", err)
		}

		if len(files) == 0 {
			cmd.Println("No changed files found.")
			return nil
		}

		// Connect to HelixDB
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			return fmt.Errorf("connecting to HelixDB: %w", err)
		}
		defer client.Close()

		// Link files to task
		linked, err := client.LinkTaskToCodeNodes(ctx, taskID, files)
		if err != nil {
			return fmt.Errorf("linking task: %w", err)
		}

		cmd.Printf("✓ Linked %d CodeNodes to task %s\n", linked, args[0])
		for _, f := range files {
			if f.IsRename() {
				cmd.Printf("  ♻ %s → %s\n", f.OldPath, f.Path)
			} else if f.IsDeleted() {
				cmd.Printf("  ✗ %s (deleted, skipped)\n", f.Path)
			} else {
				cmd.Printf("  • %s\n", f.Path)
			}
		}

		return nil
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			return fmt.Errorf("connecting to HelixDB: %w", err)
		}
		defer client.Close()

		nodes, err := client.FindNodes(ctx, "Task", nil)
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		if len(nodes) == 0 {
			cmd.Println("No tasks found.")
			return nil
		}

		cmd.Printf("Tasks (%d):\n", len(nodes))
		for _, n := range nodes {
			desc, _ := n.Properties["description"].(string)
			status, _ := n.Properties["status"].(string)
			cmd.Printf("  • #%d [%s] %s\n", n.ID, status, desc)
		}

		return nil
	},
}

func init() {
	taskLinkCmd.Flags().StringP("ref", "r", "HEAD", "Git ref to diff against")
	taskLinkCmd.Flags().StringP("dir", "d", ".", "Working directory")

	taskCmd.AddCommand(taskCreateCmd)
	taskCmd.AddCommand(taskLinkCmd)
	taskCmd.AddCommand(taskListCmd)
	rootCmd.AddCommand(taskCmd)
}
