package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/hardware"
	"github.com/secko/zyrocli/internal/opencode"
	"github.com/secko/zyrocli/internal/tui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install ZyroCLI ecosystem (config, skills, agents, MCP) globally",
	Long: `Install the ZyroCLI ecosystem into ~/.config/opencode/ and ~/.config/zyrocli/:

  1. Extracts embedded MCP Python tools to ~/.config/zyrocli/mcp-tools/
  2. Installs all embedded skills to ~/.config/opencode/skills/
  3. Verifies dependencies (uv, npm) and installs @neuledge/context
  4. Writes opencode.jsonc with global agents (SDD + Phase 0), MCP, and commands
  5. Configures OpenCode MCP integration

Run this once after installing zyrocli. No flags needed — the binary is self-contained.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ── 1. Ejecutar instalación multi-step con TUI ───────────
		steps := []tui.InstallStep{
			{
				Name: "Extrayendo MCP tools",
				Action: func() error {
					_, err := opencode.WriteMCPTools()
					return err
				},
			},
			{
				Name: "Instalando skills",
				Action: func() error {
					_, err := opencode.WriteAllSkills()
					return err
				},
			},
		{
			Name: "Verificando dependencias (uv, npm)",
			Action: func() error {
				var warnings []string

				// Verificar uv
				if _, err := exec.LookPath("uv"); err != nil {
					warnings = append(warnings, "uv no encontrado. Instalá uv: curl -LsSf https://astral.sh/uv/install.sh | sh")
				}

				// Verificar npm
				if _, err := exec.LookPath("npm"); err != nil {
					warnings = append(warnings, "npm no encontrado. Instalalo desde https://nodejs.org/")
				} else {
					// Instalar @neuledge/context antes de escribir la config (timeout 30s)
					cmd.Println("    Instalando @neuledge/context (documentación local)...")
					npmCtx, npmCancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer npmCancel()
					if err := exec.CommandContext(npmCtx, "npm", "install", "-g", "@neuledge/context").Run(); err != nil {
						if npmCtx.Err() == context.DeadlineExceeded {
							warnings = append(warnings, "@neuledge/context timeout (30s). Instalá manualmente: npm install -g @neuledge/context")
						} else {
							warnings = append(warnings, fmt.Sprintf("@neuledge/context no se pudo instalar: %v", err))
						}
					}
				}

				if len(warnings) > 0 {
					for _, w := range warnings {
						cmd.PrintErrf("    ⚠ %s\n", w)
					}
					cmd.PrintErrf("    ⚠ Algunos MCP servers podrían no funcionar. Podés instalarlos manualmente después.\n")
				} else {
					cmd.Println("    ✓ Dependencias listas")
				}
				return nil
			},
		},
		{
			Name: "Escribiendo configuración",
			Action: func() error {
				cfg := buildInstallConfig()
				var mcpWarnings []string

				// Verificar uv — necesario para helix-integration y gitmcp
				if _, err := exec.LookPath("uv"); err != nil {
					delete(cfg.MCP, "helix-integration")
					delete(cfg.MCP, "gitmcp")
					mcpWarnings = append(mcpWarnings,
						"uv no encontrado — MCP helix-integration y gitmcp no se registraron. Instalá uv: curl -LsSf https://astral.sh/uv/install.sh | sh")
				}

				// Verificar context — necesario para documentación offline
				if _, err := exec.LookPath("context"); err != nil {
					delete(cfg.MCP, "context")
					mcpWarnings = append(mcpWarnings,
						"context (Neuledge) no encontrado — MCP context no se registró. Instalá: npm install -g @neuledge/context")
				}

				for _, w := range mcpWarnings {
					cmd.PrintErrf("    ⚠ %s\n", w)
				}

				_, err := opencode.WriteGlobalConfig(cfg)
				return err
			},
		},
		{
			Name: "Verificando HelixDB",
			Action: func() error {
				client, err := helix.NewClient(cmd.Context())
				if err != nil {
					return fmt.Errorf("HelixDB: %w", err)
				}
				defer client.Close()
				if err := client.EnsureStarted(cmd.Context()); err != nil {
					return fmt.Errorf("HelixDB: %w", err)
				}
				return nil
			},
		},
		}

		if err := tui.RunInstall(steps); err != nil {
			return fmt.Errorf("install: %w", err)
		}
		cmd.Println()

		// ── 2. Menú interactivo: ¿Configurar Ollama? ────────────
		ok, err := tui.RunConfirm("¿Configurar modelos de IA local (Ollama)?")
		if err != nil {
			cmd.Printf("  ⚠ Confirm TUI: %v\n", err)
			ok = false
		}

		if ok {
			// ── 2a. Detectar GPU ────────────────────────────────
			cmd.Println()
			cmd.Println("  Detectando GPU...")
			gpuInfo, gpuErr := hardware.DetectGPU()
			if gpuErr == nil && gpuInfo != nil && gpuInfo.Detected {
				status := hardware.CheckOllamaGPUStatus()
				cmd.Printf("  ✓ GPU detectada: %s (%s)\n", gpuInfo.VendorName(), gpuInfo.Name)
				if gpuInfo.DriverVersion != "" {
					cmd.Printf("    Driver: %s\n", gpuInfo.DriverVersion)
				}

				// Show instructions based on status
				instructions := hardware.GPUInstructions(gpuInfo, status)
				if len(instructions) > 0 {
					cmd.Println()
					for _, line := range instructions {
						cmd.Println("  " + line)
					}
				}
			} else {
				cmd.Println("  ⚎ No se detectó GPU dedicada — modo CPU")
			}
			if gpuErr != nil {
				cmd.Printf("  ⚠ Error detectando GPU: %v\n", gpuErr)
			}

			// ── 2b. Configurar modelos IA ─────────────────────────
			cmd.Println()
			ok2, err := tui.RunConfirm("¿Configurar modelos IA ahora?")
			if err != nil {
				cmd.Printf("  ⚠ Confirm: %v\n", err)
			}
			if ok2 {
				cmd.Println("  Abriendo selector de modelos...")
				// TODO: integrar tui.RunModelsFlow() cuando esté en master
				cmd.Println("  Usa 'zyrocli' → 'Configurar modelos IA' para elegir modelos")
			} else {
				cmd.Println("  ⚎ Saltado. Usa 'zyrocli' → 'Configurar modelos IA' después")
			}
		} else {
			cmd.Println("  ⚎ Skipped AI model configuration")
		}

		// ── 4. find-skills global skill ──────────────────────────
		cmd.Println()
		cmd.Println("  Installing find-skills discovery skill...")
		if out, err := exec.Command("npx", "skills", "add", "vercel-labs/skills",
			"--skill", "find-skills", "-g", "-y").CombinedOutput(); err != nil {
			cmd.Printf("  ⚠ find-skills: %v\n  %s\n", err, string(out))
		} else {
			cmd.Println("  ✓ find-skills installed")
		}

		// ── 4.5 Plugin: /zyro-model ──────────────────────────────
		cmd.Println()
		cmd.Println("  Installing zyro-model plugin...")
		pluginHome, _ := os.UserHomeDir()
		pluginPath := filepath.Join(pluginHome, ".config", "opencode", "plugins", "zyro-model.ts")
		pluginContent := `/**
 * zyro-model — OpenCode slash command for per-agent model assignment.
 *
 * See: docs/spec-zyro-model-routing.md
 * Source: .config/opencode/plugins/zyro-model.ts
 */
import type { Plugin } from "@opencode-ai/plugin";
const ZYRO_AGENTS = [
  { name: "zyro-orchestrator",     description: "Coordinador - solo habla y delega",                 phase: "" },
  { name: "zyro-pre-f0",           description: "PRE-F0: Alineacion de dominio",                     phase: "PRE-F0" },
  { name: "zyro-phase-0-patterns", description: "F0: Busqueda de patrones similares",                phase: "F0" },
  { name: "zyro-phase-0-libraries",description: "F0: Investigacion de librerias",                    phase: "F0" },
  { name: "zyro-skills-find",      description: "F0: Descubrimiento de skills",                      phase: "F0" },
  { name: "zyro-skills-audit",     description: "F0: Validacion de skills descubiertas",             phase: "F0" },
  { name: "zyro-skills-apply",     description: "F0: Instalacion de skills aprobadas",               phase: "F0" },
  { name: "zyro-sdd-explore",      description: "F0: Exploracion de codebase y requerimientos",      phase: "F0" },
  { name: "zyro-sdd-spec",         description: "F1: Especificacion tecnica",                        phase: "F1" },
  { name: "zyro-sdd-propose",      description: "F2: Propuestas de cambio",                          phase: "F2" },
  { name: "zyro-sdd-design",       description: "F2: Diseno tecnico basado en Spec",                 phase: "F2" },
  { name: "zyro-sdd-tasks",        description: "F2: Division en tareas atomicas",                   phase: "F2" },
  { name: "zyro-sdd-apply",        description: "F3: Implementacion siguiendo specs, design y tasks", phase: "F3" },
  { name: "zyro-sdd-verify",       description: "F3: Verificacion contra specs y design",            phase: "F3" },
  { name: "zyro-sdd-archive",      description: "F4: Archivo de cambios completados",                phase: "F4" },
  { name: "to-issues",             description: "Generacion de GitHub Issues desde PRDs",            phase: "" },
];
export const ZyroModelPlugin: Plugin = async ({ client }) => {
  return {
    command: {
      "zyro-model": async (args: string) => {
        const parts = (args || "").trim().split(/\s+/);
        const sub = parts[0]?.toLowerCase();
        if (sub === "list" || !sub) {
          const { providers } = await client.config.providers();
          const config = await client.config.get();
          const agentConfig = config?.agent || {};
          let text = "## ZyroCLI - Model Assignments\n\n| Agent | Fase | Modelo Actual |\n|-------|------|--------------|\n";
          for (const agent of ZYRO_AGENTS) {
            const current = agentConfig[agent.name]?.model || "*(hereda del orchestrator)*";
            text += "| " + agent.name + " | " + (agent.phase || "-") + " | " + current + " |\n";
          }
          text += "\n### How to set models\n\n  zyrocli profile set <agent> <provider/model>\n  zyrocli profile tui\n";
          await client.session.prompt({ body: { noReply: true, parts: [{ type: "text", text }] } });
          await client.tui.showToast({ message: "OK /zyro-model: lista generada", variant: "success" });
        } else if (sub === "set" && parts.length >= 3) {
          const agentName = parts[1];
          const modelStr = parts.slice(2).join(" ");
          if (!ZYRO_AGENTS.some((a) => a.name === agentName)) {
            await client.tui.showToast({ message: "ERROR: Agente " + agentName + " no encontrado", variant: "error" });
            return;
          }
          await client.tui.appendPrompt({ body: { text: "!zyrocli profile set " + agentName + " " + modelStr } });
          await client.tui.showToast({ message: "Pre-cargado: Enter para ejecutar", variant: "info" });
        } else if (sub === "set-all" && parts.length >= 2) {
          const modelStr = parts.slice(1).join(" ");
          let cmd = "";
          for (const agent of ZYRO_AGENTS) cmd += "zyrocli profile set " + agent.name + " " + modelStr + " && ";
          cmd = cmd.replace(/ && $/, "");
          await client.tui.appendPrompt({ body: { text: "!" + cmd } });
          await client.tui.showToast({ message: "Pre-cargado Set All: " + modelStr + " - Enter para ejecutar", variant: "info" });
        } else {
          const helpText = "## /zyro-model - Ayuda\n\nUsage:\n  /zyro-model              - Mostrar asignaciones\n  /zyro-model set <a> <m>  - Asignar modelo a un agente\n  /zyro-model set-all <m>  - Mismo modelo para todos\n\nAgentes:\n" + ZYRO_AGENTS.map(a => "  " + a.name + " - " + a.description).join("\n");
          await client.session.prompt({ body: { noReply: true, parts: [{ type: "text", text: helpText }] } });
        }
      },
    },
  };
};
export default ZyroModelPlugin;
`
		if err := os.MkdirAll(filepath.Dir(pluginPath), 0755); err != nil {
			cmd.Printf("  ⚠ Error creating plugins dir: %v\n", err)
		} else if err := os.WriteFile(pluginPath, []byte(pluginContent), 0644); err != nil {
			cmd.Printf("  ⚠ Error writing plugin: %v\n", err)
		} else {
			cmd.Println("  ✓ zyro-model plugin installed")
		}

		// ── 5. Theme: Zorro Hacker logo → OpenCode tui.json ─────
		cmd.Println()
		cmd.Println("  Personalizing OpenCode...")
		if _, err := opencode.WriteZorroLogo(); err != nil {
			cmd.Printf("  ⚠ Error writing logo plugin: %v\n", err)
		} else {
			cmd.Println("  ✓ Zorro Hacker logo installed")
		}

		if err := opencode.UpdateTuiJSON(); err != nil {
			cmd.Printf("  ⚠ Error updating tui.json: %v\n", err)
		} else {
			cmd.Println("  ✓ OpenCode theme configured")
		}

		// ── 6. Self-update: detect old versions ─────────────────
		cmd.Println()
		selfUpdate(cmd)

		// ── 7. Summary ──────────────────────────────────────────
		cmd.Println()
		cmd.Println("✓ ZyroCLI ecosystem installed successfully!")
		cmd.Println()
		cmd.Println("  Next steps:")
		cmd.Println("    1. zyro onboard .                — Add existing project to ZyroCLI")
		cmd.Println("    2. zyrocli init <handoff.yaml>   — Create a new project")
		cmd.Println("    3. zyrocli profile tui            — Assign AI models per phase")
		cmd.Println()
		cmd.Println("  OpenCode is ready with Zorro Hacker 🦊")

		return nil
	},
}

// buildInstallConfig builds the opencode configuration for the installer.
func buildInstallConfig() *opencode.Config {
	// Resolve MCP tools directory (where embedded Python tools are extracted)
	home, _ := os.UserHomeDir()
	mcpDir := filepath.Join(home, ".config", "zyrocli", "mcp-tools")
	sddPerms := map[string]any{
		"read": "allow", "write": "deny", "edit": "deny", "bash": "deny",
	}
	sddArchivePerms := map[string]any{
		"read": "allow", "write": "deny", "edit": "deny", "bash": "allow",
		"task": map[string]any{"*": "deny"},
	}
	// permsWithTaskDeny clona un map base y agrega task:{"*":"deny"}.
	// Se usa para sdd-propose, sdd-design, sdd-tasks que comparten sddPerms
	// pero necesitan bloque task explícito.
	permsWithTaskDeny := func(base map[string]any) map[string]any {
		m := make(map[string]any, len(base)+1)
		for k, v := range base {
			m[k] = v
		}
		m["task"] = map[string]any{"*": "deny"}
		return m
	}
	defaultModel := "opencode-go/deepseek-v4-flash"

	return &opencode.Config{
		Schema: "https://opencode.ai/config.json",
		Agent: map[string]opencode.Agent{
			"zyro-orchestrator": {
				Mode: "primary", Model: defaultModel,
				Description: "ZyroCLI Orchestrator — solo habla y delega, nunca toca código",
				Prompt:      "{skill:zyro-orchestrator}",
				Permission: map[string]any{
					"read": "allow",
					"task": map[string]any{
						"*":                          "deny",
						"zyro-sdd-apply":             "allow",
						"zyro-sdd-verify":            "allow",
						"zyro-sdd-explore":           "allow",
						"zyro-sdd-propose":           "allow",
						"zyro-sdd-spec":              "allow",
						"zyro-sdd-design":            "allow",
						"zyro-sdd-tasks":             "allow",
						"zyro-sdd-archive":           "allow",
						"zyro-phase-0-patterns":      "allow",
						"zyro-phase-0-libraries":     "allow",
						"zyro-skills-find":           "allow",
						"zyro-skills-audit":          "allow",
						"zyro-skills-apply":          "allow",
						"zyro-pre-f0":                "allow",
					},
					"write": "deny", "edit": "deny", "bash": "deny",
					"glob": "deny", "grep": "deny", "list": "deny",
					"skill": "deny", "webfetch": "allow", "question": "allow",
				},
			},
			"zyro-phase-0-patterns": {
				Mode: "subagent", Model: defaultModel,
				Description: "Fase 0: busca patrones similares en internet y guarda en HelixDB",
				Prompt: "{skill:zyro-phase-0-patterns}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "write": "deny", "edit": "deny",
					"bash": "deny", "webfetch": "allow",
				},
			},
			"zyro-phase-0-libraries": {
				Mode: "subagent", Model: defaultModel,
				Description: "Fase 0: investiga librerías con Context + GitMCP, guarda en HelixDB",
				Prompt: "{skill:zyro-phase-0-libraries}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "deny", "webfetch": "allow",
					"write": "deny", "edit": "deny",
				},
			},
			"zyro-skills-find": {
				Mode: "subagent", Model: defaultModel,
				Description: "Fase 0: descubre skills en skills.sh y guarda en HelixDB",
				Prompt: "{skill:zyro-skills-find}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "allow", "webfetch": "allow",
					"write": "deny", "edit": "deny",
				},
			},
			"zyro-skills-audit": {
				Mode: "subagent", Model: defaultModel,
				Description: "Fase 0: valida audits de skills descubiertas, guarda en HelixDB",
				Prompt: "{skill:zyro-skills-audit}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "webfetch": "allow",
					"bash": "deny", "write": "deny", "edit": "deny",
				},
			},
			"zyro-skills-apply": {
				Mode: "subagent", Model: defaultModel,
				Description: "Fase 0: instala skills APROBADAS por el humano en el proyecto",
				Prompt: "{skill:zyro-skills-apply}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "allow", "write": "allow",
					"edit": "deny",
				},
			},
			"zyro-sdd-apply": {
				Mode: "subagent", Model: defaultModel,
				Description: "Implementa código siguiendo specs, design y tasks",
				Prompt: "{skill:zyro-sdd-apply}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "write": "allow", "edit": "allow",
					"bash": "allow",
				},
			},
			"zyro-sdd-verify": {
				Mode: "subagent", Model: defaultModel,
				Description: "Verifica implementación contra specs, design y tasks",
				Prompt: "{skill:zyro-sdd-verify}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "write": "deny", "edit": "deny",
					"bash": "allow",
				},
			},
			"zyro-sdd-explore": {
				Mode: "subagent", Model: defaultModel,
				Description: "Explora codebase y requerimientos antes de un cambio",
				Prompt: "{skill:zyro-sdd-explore}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "allow", "question": "allow",
					"write": "deny", "edit": "deny",
					"task": map[string]any{"*": "deny"},
				},
			},
			"zyro-sdd-propose": {
				Mode: "subagent", Model: defaultModel,
				Description: "Crea propuestas de cambio con intento, alcance y enfoque",
				Prompt: "{skill:zyro-sdd-propose}", Hidden: true,
				Permission: permsWithTaskDeny(sddPerms),
			},
			"zyro-sdd-spec": {
				Mode: "subagent", Model: defaultModel,
				Description: "F1: Diseña especificación técnica basada en hallazgos de Fase 0",
				Prompt: "{skill:zyro-sdd-spec}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "deny",
					"write": "deny", "edit": "deny",
				},
			},
			"zyro-sdd-design": {
				Mode: "subagent", Model: defaultModel,
				Description: "F2: Diseño técnico basado en Spec — crea nodo Design en HelixDB",
				Prompt: "{skill:zyro-sdd-design}", Hidden: true,
				Permission: permsWithTaskDeny(sddPerms),
			},
			"zyro-sdd-tasks": {
				Mode: "subagent", Model: defaultModel,
				Description: "F2: Divide el diseño en tareas atómicas — crea nodos Task en HelixDB",
				Prompt: "{skill:zyro-sdd-tasks}", Hidden: true,
				Permission: permsWithTaskDeny(sddPerms),
			},
			"zyro-sdd-archive": {
				Mode: "subagent", Model: defaultModel,
				Description: "Archiva cambios completados y sincroniza delta specs",
				Prompt: "{skill:zyro-sdd-archive}", Hidden: true,
				Permission: sddArchivePerms,
			},
			"zyro-pre-f0": {
				Mode: "subagent", Model: defaultModel,
				Description: "PRE-F0: Alineación de dominio — grill-me, domain-model, triage, improve-arch",
				Prompt: "{skill:zyro-pre-f0}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "deny", "webfetch": "allow", "question": "allow",
					"write": "deny", "edit": "deny",
				},
			},
			"to-issues": {
				Mode: "subagent", Model: defaultModel,
				Description: "Genera GitHub Issues desde PRDs y tasks",
				Prompt: "{skill:to-issues}", Hidden: true,
				Permission: map[string]any{
					"read": "allow", "bash": "allow", "webfetch": "allow",
					"write": "deny", "edit": "deny",
					"task": map[string]any{"*": "deny"},
				},
			},
		},
		MCP: map[string]opencode.MCPEntry{
			"helix-integration": {
				Type:    "local",
				Command: []string{"uv", "run", "--directory", mcpDir, "runner.py"},
			},
			"zyro-task-board": {
				Type:    "local",
				Command: []string{"zyrocli", "mcp-server"},
			},
			"context": {
				Type:    "local",
				Command: []string{"context", "serve"},
			},
			"gitmcp": {
				Type:    "local",
				Command: []string{"uvx", "mcp-server-git", "--repository", "."},
			},
		},
		Skills: &opencode.SkillsConfig{
			Paths: []string{"~/.config/opencode/skills"},
		},
	}
}

// selfUpdate checks for old zyrocli versions and handles self-update.
func selfUpdate(cmd *cobra.Command) {
	currentBin, err := os.Executable()
	if err != nil {
		return
	}

	otherBin, _ := exec.LookPath("zyrocli")
	if otherBin != "" {
		currentReal, _ := filepath.EvalSymlinks(currentBin)
		otherReal, _ := filepath.EvalSymlinks(otherBin)
		if currentReal != otherReal && otherReal != "" {
			cmd.Printf("  ⚠ Se encontró otra versión de zyrocli:\n")
			cmd.Printf("    Actual:  %s\n", currentReal)
			cmd.Printf("    Existente: %s\n", otherReal)
			cmd.Print("  ¿Reemplazar con la versión actual? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))
			if response == "y" || response == "yes" || response == "s" || response == "sí" {
				if err := copyFile(currentReal, otherReal); err != nil {
					cmd.Printf("    ⚠ Error al actualizar: %v\n", err)
				} else {
					cmd.Printf("    ✓ zyrocli actualizado en %s\n", otherReal)
				}
			}
		}
	}

	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin", "zyrocli")
	if localBin != currentBin {
		if err := copyFile(currentBin, localBin); err == nil {
			cmd.Printf("    ✓ Copia de respaldo en %s\n", localBin)
		}
	}
}

// copyFile copies a file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode())
}

func init() {
	rootCmd.AddCommand(installCmd)
}
