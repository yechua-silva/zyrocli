package main

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/yechua-silva/zyrocli/internal/opencode"
	"github.com/spf13/cobra"
)

// runProfileTUI is the cobra RunE for "profile tui".
// Shows current assignments and tells the user to use /zyro-model in OpenCode.
func runProfileTUI(cmd *cobra.Command, args []string) error {
	cmd.Println("╭──────────────────────────────────────────────────────────────╮")
	cmd.Println("│                                                              │")
	cmd.Println("│  Para configurar modelos por agente, usá /zyro-model         │")
	cmd.Println("│  directamente en OpenCode.                                   │")
	cmd.Println("│                                                              │")
	cmd.Println("│  Ejemplos:                                                   │")
	cmd.Println("│    /zyro-model              → Ver asignaciones actuales     │")
	cmd.Println("│    /zyro-model set <a> <m>  → Asignar modelo a un agente    │")
	cmd.Println("│    /zyro-model set-all <m>  → Mismo modelo para todos       │")
	cmd.Println("│                                                              │")
	cmd.Println("│  También podés usar:                                         │")
	cmd.Println("│    zyrocli profile list     → Ver asignaciones              │")
	cmd.Println("│    zyrocli profile set ...  → Asignar modelo desde terminal  │")
	cmd.Println("│                                                              │")
	cmd.Println("╰──────────────────────────────────────────────────────────────╯")
	cmd.Println()

	// Show current assignments
	configPath := opencode.GetEffectiveConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg opencode.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	if len(cfg.Agent) == 0 {
		return nil
	}

	names := make([]string, 0, len(cfg.Agent))
	for name := range cfg.Agent {
		names = append(names, name)
	}
	sort.Strings(names)

	cmd.Println("Asignaciones actuales:")
	cmd.Println("──────────────────────")
	for _, name := range names {
		model := cfg.Agent[name].Model
		if model == "" {
			model = "(default)"
		}
		cmd.Printf("  %-30s %s\n", name, model)
	}

	return nil
}
