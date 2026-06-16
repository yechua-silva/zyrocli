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
	Short: "ZyroCLI — Orquestador para desarrollo asistido por IA",
	Long: `ZyroCLI orquesta el pipeline SDD: especificar, diseñar, implementar,
verificar y archivar. Cada fase es un subcomando ejecutado por
agentes de IA especializados.`,
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
	rootCmd.AddCommand(versionCmd)

	// Soporte para --version como flag (antes que el subcomando)
	rootCmd.Flags().Bool("version", false, "print version information")
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
			fmt.Printf("zyrocli %s (commit: %s, built: %s)\n", version, commit, date)
			return
		}
		_ = cmd.Help()
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
