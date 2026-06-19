package main

import (
	"context"
	"fmt"
	"strings"

	dbhelix "github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/memory"
	"github.com/secko/zyrocli/internal/setup"
	"github.com/spf13/cobra"
)

var doctorTokens bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnostica y repara la configuración del CLI",
	Long: `Revisa el estado del CLI ZyroAgent:

  1. Archivo de configuración (~/.zyro/config.yaml)
  2. Dependencias del sistema (go, uv, docker, helixdb, git)
  3. Paths de configuración
  4. Permisos de directorios
  5. Estado de HelixDB

Con --fix intenta reparar automáticamente los problemas encontrados.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fix, _ := cmd.Flags().GetBool("fix")

		if doctorTokens {
			return showTokenMeasurements(cmd)
		}

		return setup.RunDoctor(fix)
	},
}

// showTokenMeasurements consulta HelixDB y muestra tabla de ahorro de tokens.
func showTokenMeasurements(cmd *cobra.Command) error {
	ctx := context.Background()

	// Intentar conectar a HelixDB
	client, err := dbhelix.NewClient(ctx, dbhelix.WithBaseURL("http://localhost:6969"))
	if err != nil {
		cmd.Println("⚠️  No se pudo conectar a HelixDB para obtener mediciones.")
		cmd.Println("   Asegúrate de que HelixDB esté corriendo en localhost:6969")
		return nil
	}
	defer client.Close()

	store := memory.NewHelixEngramStore(client, nil)

	// Consultar Facts de tipo "measurement"
	results, err := store.RecallMemories(ctx, memory.RecallOpts{
		QueryText:   "measurement",
		MaxResults:  100,
		MinSalience: 0.0,
	})
	if err != nil {
		cmd.Printf("⚠️  Error consultando mediciones: %v\n", err)
		return nil
	}

	// Filtrar solo measurements
	type Measurement struct {
		Phase   string
		Without int64
		With    int64
		Count   int
	}
	phases := make(map[string]*Measurement)

	for _, r := range results {
		if r.Fact.Type != "measurement" {
			continue
		}
		// Parsear content: "phase=F0 without=123 with=45"
		phase := r.Fact.Phase
		if phase == "" {
			continue
		}
		if _, ok := phases[phase]; !ok {
			phases[phase] = &Measurement{Phase: phase}
		}
		phases[phase].Count++
	}

	if len(phases) == 0 {
		cmd.Println("📊 No hay mediciones de tokens disponibles.")
		cmd.Println("   Las mediciones se recolectan automáticamente al ejecutar fases con Boomerang.")
		cmd.Println("   Ejecuta: zyro run --phase F0")
		return nil
	}

	// Mostrar tabla
	cmd.Println("\n📊 Ahorro de Tokens por Fase")
	cmd.Println(strings.Repeat("─", 60))
	cmd.Printf("%-8s %-14s %-14s %-10s %s\n", "Fase", "Sin Boomerang", "Con Boomerang", "Ahorro", "Muestras")
	cmd.Println(strings.Repeat("─", 60))

	for _, p := range phases {
		if p.Count < 3 {
			cmd.Printf("%-8s %-14s %-14s %-10s %s\n", p.Phase, "—", "—", "⏳ N<3", fmt.Sprintf("%d muestras", p.Count))
		} else {
			// Calcular promedios — necesitaríamos datos más precisos
			cmd.Printf("%-8s %-14s %-14s %-10s %s\n", p.Phase, "—", "—", "—", fmt.Sprintf("%d muestras ✅", p.Count))
		}
	}
	cmd.Println(strings.Repeat("─", 60))
	cmd.Println("  ⏳ N<3: Datos insuficientes para reportar. Mínimo 3 muestras por fase.")
	cmd.Println("  Las mediciones precisas requieren integrar tiktoken o similar.")
	cmd.Println()

	return nil
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().Bool("fix", false, "Reparar problemas detectados automáticamente")
	doctorCmd.Flags().BoolVar(&doctorTokens, "tokens", false, "Mostrar tabla de ahorro de tokens por fase")
}
