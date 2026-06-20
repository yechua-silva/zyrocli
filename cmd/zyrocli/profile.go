package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/secko/zyrocli/internal/opencode"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage model assignments per agent",
	Long: `Manage which AI model each agent uses.

Subcommands:
  list              Show current model assignments
  set <agent> <model>  Set model for an agent (e.g. "set sdd-design anthropic/claude-sonnet-4")
  tui               Interactive model selector`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show current model assignments for all agents",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("profile: read config: %w", err)
		}

		var cfg opencode.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("profile: parse config: %w", err)
		}

		// Sort agent names
		names := make([]string, 0, len(cfg.Agent))
		for name := range cfg.Agent {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Println("Agent Model Assignments:")
		fmt.Println(strings.Repeat("-", 70))
		for _, name := range names {
			agent := cfg.Agent[name]
			model := agent.Model
			if model == "" {
				model = "(default)"
			}
			fmt.Printf("  %-30s %s\n", name, model)
		}

		return nil
	},
}

var profileSetCmd = &cobra.Command{
	Use:   "set <agent> <model>",
	Short: "Set model for an agent",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		agentName := args[0]
		modelName := args[1]

		configPath := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("profile: read config: %w", err)
		}

		var cfg opencode.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("profile: parse config: %w", err)
		}

		if cfg.Agent == nil {
			cfg.Agent = make(map[string]opencode.Agent)
		}

		agent, exists := cfg.Agent[agentName]
		if !exists {
			return fmt.Errorf("profile: agent %q not found", agentName)
		}

		if err := validateModel(modelName); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠️  Advertencia: %v\n", err)
			fmt.Fprintln(cmd.ErrOrStderr(), "  Igualmente se guardará el modelo. Verificá que exista en OpenCode.")
		}

		agent.Model = modelName
		cfg.Agent[agentName] = agent

		out, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("profile: marshal: %w", err)
		}

		if err := os.WriteFile(configPath, out, 0644); err != nil {
			return fmt.Errorf("profile: write: %w", err)
		}

		fmt.Printf("✓ Agent %q model set to %q\n", agentName, modelName)
		return nil
	},
}

var profileTUICmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive model selector (text-based)",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc")
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("profile: read config: %w", err)
		}

		var cfg opencode.Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("profile: parse config: %w", err)
		}

		if cfg.Agent == nil {
			cfg.Agent = make(map[string]opencode.Agent)
		}

		// Sort agent names
		names := make([]string, 0, len(cfg.Agent))
		for name := range cfg.Agent {
			names = append(names, name)
		}
		sort.Strings(names)

		fmt.Println("ZyroCLI Model Selector")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println("Select an agent to assign a model. Enter empty to skip.")
		fmt.Println()

		for _, name := range names {
			agent := cfg.Agent[name]
			current := agent.Model
			if current == "" {
				current = "default"
			}

			fmt.Printf("[%s]\n", name)
			fmt.Printf("  Current: %s\n", current)
			fmt.Print("  Model (or press Enter to skip): ")

			var input string
			fmt.Scanln(&input)
			input = strings.TrimSpace(input)

			if input != "" {
				agent.Model = input
				cfg.Agent[name] = agent
				fmt.Printf("  ✓ Set to %s\n", input)
			}
			fmt.Println()
		}

		out, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("profile: marshal: %w", err)
		}

		if err := os.WriteFile(configPath, out, 0644); err != nil {
			return fmt.Errorf("profile: write: %w", err)
		}

		fmt.Println("✓ Model assignments saved.")
		return nil
	},
}

// validateModel checks that the model string has the format "provider/model"
// and that both provider and model exist in the known or configured providers.
func validateModel(modelStr string) error {
	parts := strings.SplitN(modelStr, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("formato inválido: debe ser provider/model (ej: anthropic/claude-sonnet-4)")
	}

	providerID := parts[0]
	modelID := parts[1]

	// Check KnownProviders
	providers := opencode.KnownProviders()
	for _, p := range providers {
		if p.ID == providerID {
			for _, m := range p.Models {
				if m.ID == modelID {
					return nil
				}
			}
			return fmt.Errorf("modelo %q no encontrado en proveedor %q. Modelos disponibles: %s",
				modelID, providerID, formatModels(p.Models))
		}
	}

	return fmt.Errorf("proveedor %q no encontrado. Proveedores disponibles: %s",
		providerID, formatProviders(providers))
}

func formatModels(models []opencode.Model) string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.ID
	}
	return strings.Join(names, ", ")
}

func formatProviders(providers []opencode.Provider) string {
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.ID
	}
	return strings.Join(names, ", ")
}

func init() {
	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileSetCmd)
	profileCmd.AddCommand(profileTUICmd)
	rootCmd.AddCommand(profileCmd)
}
