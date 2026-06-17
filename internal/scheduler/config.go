package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/secko/zyrocli/internal/boomerang"
	"github.com/secko/zyrocli/internal/boundari"
	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/handoff"
	"github.com/secko/zyrocli/internal/memory"
)

// LoadConfig reads handoff.yaml and extracts scheduler configuration.
func LoadConfig(path string) (*Config, error) {
	payload, err := handoff.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	timeout := 10 * time.Minute // default
	if payload.Limits.PhaseTimeout != "" {
		timeout, err = time.ParseDuration(payload.Limits.PhaseTimeout)
		if err != nil {
			return nil, fmt.Errorf("load config: invalid phase_timeout %q: %w", payload.Limits.PhaseTimeout, err)
		}
	}

	maxLoops := payload.Limits.MaxLoops
	if maxLoops == 0 {
		maxLoops = 5 // default
	}

	return &Config{
		Mode:         payload.Governance.Mode,
		Module:       payload.Governance.Module,
		GoVersion:    payload.Governance.GoVersion,
		MaxTasks:     payload.Limits.MaxTasks,
		MaxLines:     payload.Limits.MaxLines,
		MaxLoops:     maxLoops,
		PhaseTimeout: timeout,
	}, nil
}

// NewDefaultConfig crea una Config con Boomerang inicializado.
// Si no puede inicializar el store de memoria, retorna una config
// sin Boomerang (fallback modo legacy).
func NewDefaultConfig(projectDir string) *Config {
	cfg := &Config{
		Mode:         "interactive",
		MaxLoops:     5,
		PhaseTimeout: 10 * time.Minute,
	}

	// Inicializar store de memoria (HelixDB)
	helixClient, err := helix.NewClient(context.Background(),
		helix.WithBaseURL("http://localhost:6969"),
	)
	if err != nil {
		log.Printf("[scheduler] No se pudo conectar a HelixDB: %v (modo legacy)", err)
		return cfg
	}

	store := memory.NewHelixEngramStore(helixClient, nil)

	// Inicializar BoomerangOrchestrator con callback de medición
	boomer := boomerang.NewBoomerangOrchestrator(
		store,
		func(phase string) (*boundari.Policy, error) {
			return boundari.LoadPolicy(phase, []string{projectDir})
		},
		func(m boomerang.Measurement) {
			// Guardar medición como Fact en HelixDB
			fact := &memory.Fact{
				Type:       memory.FactType("measurement"),
				Content:    fmt.Sprintf("phase=%s without=%d with=%d", m.Phase, m.WithoutBoomerang, m.WithBoomerang),
				Salience:   0.3,
				Confidence: 1.0,
				Source:     "boomerang:measurement",
				Phase:      m.Phase,
				IsActive:   false,
				DecayRate:  1.0, // no decae — son datos históricos
			}
			if _, err := store.SaveFact(context.Background(), fact); err != nil {
				log.Printf("[boomerang] Error guardando medición: %v", err)
			}
		},
	)

	cfg.Boomerang = boomer
	cfg.MemoryHooks = NewMemoryHooks(store, "")

	return cfg
}
