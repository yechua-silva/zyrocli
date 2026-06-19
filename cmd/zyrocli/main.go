package main

import (
	"fmt"
	"os"

	"github.com/secko/zyrocli/internal/tui"
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

	// Soporte para --version como flag
	rootCmd.Flags().Bool("version", false, "print version information")
	rootCmd.Run = handleMenu
}

// handleMenu muestra el menú principal y ejecuta la opción seleccionada.
func handleMenu(cmd *cobra.Command, args []string) {
	// --version flag
	if showVersion, _ := cmd.Flags().GetBool("version"); showVersion {
		fmt.Printf("zyrocli %s (commit: %s, built: %s)\n", version, commit, date)
		return
	}

	// Si hay args, mostrar help normal
	if len(args) > 0 {
		cmd.Help()
		return
	}

	// Mostrar menú TUI
	for {
		fmt.Print("\033[2J\033[H")
		choice := tui.RunMainMenu()

		switch choice {
		case "install":
			runInstallFlow()
		case "setup":
			runSetupFlow()
		case "models":
			runModelsFlow()
		case "autostart":
			runAutostartFlow()
		case "about":
			runAboutFlow()
		case "exit":
			fmt.Println("👋 Hasta luego!")
			return
		default:
			return
		}
	}
}

// runInstallFlow ejecuta instalación completa (todo en uno).
func runInstallFlow() {
	fmt.Print("\033[2J\033[H")
	// Pasos de instalación actual
	installCmd.RunE(rootCmd, []string{})
	fmt.Println()
	tui.PrintSuccess("Instalación base completada")

	// Preguntar si configurar modelos
	fmt.Print("\033[2J\033[H")
	ok, _ := tui.RunConfirm("¿Configurar modelos de IA ahora?")
	if !ok {
		fmt.Println(tui.Info("Puedes configurarlos después con 'zyrocli' → Configurar modelos IA"))
		return
	}

	models := tui.RunModelsFlow()
	if models == nil {
		return
	}

	// Mostrar resumen GPU
	fmt.Println()
	fmt.Println(tui.GPUSummary())

	fmt.Println()
	tui.PrintSuccess("Instalación completa")
}

// runSetupFlow ejecuta configuración de servicios.
func runSetupFlow() {
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println(tui.RenderBrand())
	fmt.Println()

	// Verificar HelixDB
	helix := tui.CheckHelixDB()
	fmt.Println(tui.FormatServiceStatus(helix))

	// Verificar Ollama
	ollama := tui.CheckOllama()
	fmt.Println(tui.FormatServiceStatus(ollama))

	// GPU
	fmt.Println()
	fmt.Println(tui.GPUSummary())

	fmt.Println()
	fmt.Println(tui.RenderSeparator())
	fmt.Println()

	// Preguntar si iniciar servicios que no corren
	if !helix.Running || !ollama.Running {
		start, _ := tui.RunConfirm("¿Iniciar servicios que no están corriendo?")
		if start {
			if !helix.Running {
				// TODO: iniciar HelixDB
				fmt.Println(tui.Info("Para iniciar HelixDB: helix up"))
			}
			if !ollama.Running {
				// TODO: iniciar Ollama
				fmt.Println(tui.Info("Para iniciar Ollama: ollama serve"))
			}
		}
	}
}

// runModelsFlow ejecuta configuración de modelos IA.
func runModelsFlow() {
	fmt.Print("\033[2J\033[H")
	// Verificar Ollama primero
	ollama := tui.CheckOllama()
	if !ollama.Running {
		fmt.Println(tui.ErrorStr("Ollama no está corriendo. Inicia con: ollama serve"))
		start, _ := tui.RunConfirm("¿Iniciar Ollama ahora?")
		if start {
			fmt.Println(tui.Info("Ejecuta 'ollama serve' en otra terminal o configura auto-inicio"))
			return
		}
		return
	}

	models := tui.RunModelsFlow()
	if models == nil {
		return
	}

	// Resumen
	fmt.Println()
	fmt.Println(tui.Success("Modelos seleccionados:"))
	fmt.Println(tui.Info("Embeddings: " + models["embedding_model"]))
	fmt.Println(tui.Info("Chat: " + models["chat_model"]))

	// Probar modelos
	fmt.Print("\033[2J\033[H")
	test, _ := tui.RunConfirm("¿Probar los modelos ahora?")
	if !test {
		return
	}

	fmt.Println()
	fmt.Print("\033[2J\033[H")
	fmt.Println(tui.Info("Probando embeddings..."))
	er := tui.TestEmbedding(models["embedding_model"], 30)
	fmt.Println(tui.FormatTestResult(er))

	fmt.Println()
	fmt.Print("\033[2J\033[H")
	fmt.Println(tui.Info("Probando chat..."))
	cr := tui.TestChat(models["chat_model"], 60)
	fmt.Println(tui.FormatTestResult(cr))
}

// runAutostartFlow configura inicio automático de servicios.
func runAutostartFlow() {
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println(tui.RenderBrand())
	fmt.Println()
	fmt.Println(tui.Info("Configurando inicio automático de servicios..."))
	fmt.Println()

	// HelixDB
	result := tui.SetupHelixAutostart()
	fmt.Println(result)

	// Ollama
	result2 := tui.SetupOllamaAutostart()
	fmt.Println(result2)

	fmt.Println()
	fmt.Println(tui.Success("Auto-inicio configurado. Los servicios arrancarán al iniciar sesión."))
}

func runAboutFlow() {
	fmt.Print("\033[2J\033[H")
	fmt.Println()
	fmt.Println(tui.RenderBrand())
	fmt.Println()

	aboutText := `ZyroCLI (zyrocli) es una herramienta CLI escrita en Go que actúa como configurador e instalador del ecosistema ZyroAgentCLI.

No es un orquestador de agentes — su rol es preparar el entorno:

  • Instalar skills
  • Registrar MCPs
  • Configurar OpenCode
  • Verificar HelixDB
  • Gestionar modelos Ollama (embeddings + chat)
  • Configurar auto-inicio de servicios

El punto de entrada es zyrocli, que lanza una TUI interactiva con un menú principal. Desde ahí el usuario navega por flujos de instalación o configuración.

El stack visual usa el ASCII logo "ZYRO-CLI" con tema oscuro naranja/amber sobre fondo casi negro.`

	fmt.Println(tui.Info(aboutText))
	fmt.Println()
	tui.RunConfirm("Volver al menú principal")
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
