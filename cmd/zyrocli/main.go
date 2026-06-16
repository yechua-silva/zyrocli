package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose bool
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "zyrocli",
	Short: "ZyroCLI — Gentle AI orchestration for structured SDD workflows",
	Long: `ZyroCLI orchestrates the SDD lifecycle: propose, spec, design, tasks,
apply, verify, and archive. Each phase is a subcommand powered by
domain-specific internal packages.`,
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Println("ZyroCLI — verbose mode enabled")
		}
		_ = cmd.Help()
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("zyrocli %s (commit: %s, built: %s)\n", version, commit, date)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.Flags().BoolP("version", "", false, "print version information")
	rootCmd.AddCommand(versionCmd)

	// Override root Run to intercept --version
	originalRun := rootCmd.Run
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Printf("zyrocli %s (commit: %s, built: %s)\n", version, commit, date)
			return
		}
		originalRun(cmd, args)
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
