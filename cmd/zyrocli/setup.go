package main

import (
	"github.com/yechua-silva/zyrocli/internal/setup"
	"github.com/yechua-silva/zyrocli/internal/tui"
	"github.com/spf13/cobra"
)

var setupDryRun bool
var setupVerbose bool
var setupForce bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Verifica e instala todas las dependencias necesarias",
	Long: `Verifica que todas las dependencias necesarias existen en el sistema
(go, uv, docker, helixdb, git) y, si falta alguna, intenta instalarla
automáticamente.

FLUJO:
  1. Detecta sistema operativo y arquitectura
  2. Verifica cada dependencia (go, uv, docker, helixdb, git)
  3. Instala las que faltan (uv, helixdb se instalan automáticamente;
     go, docker, git requieren instalación manual)
  4. Genera configuración en ~/.zyro/config.yaml

Idempotente: si una dependencia ya está instalada, la salta.

Flags:
  --dry-run  Muestra qué se haría sin ejecutar nada
  --verbose  Output detallado de cada paso
  --force    Reinstalar aunque ya exista`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// ── 0. Brand (solo ZYRO 3D, sin Zorro, sin borde) ──────
		tui.PrintBrand("ZyroCLI — Verificación de Dependencias")
		cmd.Println()

		// ── 1. Check dependencias ───────────────────────────────
		if err := setup.RunSetup(setupDryRun, setupVerbose, setupForce); err != nil {
			return err
		}

		// ── 2. Configurar modelos IA ────────────────────────────
		cmd.Println()
		ok, err := tui.RunConfirm("¿Configurar modelos IA ahora?")
		if err != nil {
			cmd.Printf("  ⚠ Confirm: %v\n", err)
		}
		if ok {
			cmd.Println("  Usa 'zyrocli' → 'Configurar modelos IA' para elegir modelos")
		} else {
			cmd.Println("  ⚎ Saltado. Usa 'zyrocli' → 'Configurar modelos IA' después")
		}

		cmd.Println()
		cmd.Println("✓ Setup complete! Run 'zyrocli install' to configure OpenCode.")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVarP(&setupDryRun, "dry-run", "n", false, "Show what would be done")
	setupCmd.Flags().BoolVarP(&setupVerbose, "verbose", "v", false, "Detailed output")
	setupCmd.Flags().BoolVarP(&setupForce, "force", "f", false, "Force reinstall")
}
