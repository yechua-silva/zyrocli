package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yechua-silva/zyrocli/internal/opencode"
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
		configPath := opencode.GetEffectiveConfigPath()
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

		configPath := opencode.GetEffectiveConfigPath()
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
			// Agent not found in config — create it
			agent = opencode.Agent{
				Mode: "subagent",
			}
			// Try to find the canonical mode from profile_agents.go
			for _, a := range zyroAgents {
				if a.Name == agentName {
					agent.Mode = a.DefaultMode
					break
				}
			}
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
	Short: "Interactive model selector",
	Long:  `Interactive TUI to assign AI models to each ZyroCLI agent.
Uses the same visual style as the installer.`,
	RunE: runProfileTUI,
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
