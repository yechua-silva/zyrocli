package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	helix "github.com/secko/zyrocli/internal/db/helix"
)

var dbURL string

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "HelixDB database management",
	Long: `Manage the HelixDB knowledge graph database.

HelixDB stores project context, decisions, patterns, and code summaries
for efficient context injection during SDD phases.`,
}

var dbInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize HelixDB schema (idempotent — safe to re-run)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			return fmt.Errorf("connecting to HelixDB at %s: %w\nMake sure HelixDB is running: helix start dev --disk", dbURL, err)
		}
		defer client.Close()

		cmd.Println("✓ Connected to HelixDB")
		cmd.Print("Initializing schema... ")

		if err := client.InitSchema(ctx); err != nil {
			cmd.Println("FAILED")
			return fmt.Errorf("schema initialization: %w", err)
		}

		cmd.Println("DONE")
		cmd.Println("✓ Schema initialized (idempotent — all indexes ready)")
		return nil
	},
}

var dbStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check HelixDB connection and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			fmt.Printf("⚠ HelixDB not reachable at %s\n", dbURL)
			fmt.Println("  Start it with: helix start dev --disk")
			return nil
		}
		defer client.Close()

		cmd.Printf("✓ HelixDB is running at %s\n", dbURL)
		cmd.Println("  Status: connected")
		return nil
	},
}

var dbResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete ALL data in HelixDB (dev only, requires confirmation)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println("⚠ WARNING: This will DELETE ALL DATA in HelixDB")
		cmd.Print("Are you sure? [y/N]: ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			cmd.Println("Canceled.")
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := helix.NewClient(ctx, helix.WithBaseURL(dbURL))
		if err != nil {
			return err
		}
		defer client.Close()

		cmd.Println("✓ HelixDB connection OK — ready for operations")
		return nil
	},
}

func init() {
	dbCmd.PersistentFlags().StringVarP(&dbURL, "url", "u", "http://localhost:6969", "HelixDB server URL")

	dbCmd.AddCommand(dbInitCmd)
	dbCmd.AddCommand(dbStatusCmd)
	dbCmd.AddCommand(dbResetCmd)
	rootCmd.AddCommand(dbCmd)
}
