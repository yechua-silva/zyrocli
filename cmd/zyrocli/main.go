package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var verbose bool

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

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
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
