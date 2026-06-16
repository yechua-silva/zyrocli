package memory

import (
	"context"
	"fmt"
	"math"
	"time"
)

// DecayAndRefresh aplica decaimiento temporal (Ebbinghaus) a todos los facts del proyecto
func (s *HelixEngramStore) DecayAndRefresh(ctx context.Context, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("memory: project_id is required")
	}

	// TODO: cargar todos los Facts activos del proyecto desde HelixDB
	// Por ahora, la implementación asume que se pasan facts externamente
	// o se integra con un listado de facts

	return nil
}

// calculateEbbinghausDecay calcula la nueva salience usando la curva de olvido
// Fórmula: newSalience = salience * e^(-decayRate * daysSinceAccess)
func calculateEbbinghausDecay(salience, decayRate float64, daysSinceAccess float64) float64 {
	if daysSinceAccess <= 0 {
		return salience
	}
	return salience * math.Exp(-decayRate*daysSinceAccess)
}

// applyDecayToFact aplica decaimiento a un fact individual
func (s *HelixEngramStore) applyDecayToFact(ctx context.Context, fact *Fact, config DecayConfig) error {
	daysSinceAccess := time.Since(fact.LastAccessedAt).Hours() / 24.0

	if daysSinceAccess <= 0 {
		return nil
	}

	// Aplicar fórmula de Ebbinghaus
	newSalience := calculateEbbinghausDecay(fact.Salience, fact.DecayRate, daysSinceAccess)

	// Si la salience bajó del threshold, marcar como stale
	if newSalience < config.SalienceThreshold {
		fact.IsStale = true
	}

	// Si expiró, marcar como inactivo
	if !fact.ExpiresAt.IsZero() && time.Now().After(fact.ExpiresAt) {
		fact.IsActive = false
	}

	fact.Salience = newSalience

	// TODO: actualizar en HelixDB
	_ = ctx
	return nil
}

// DecayConfigFromDefaults retorna DecayConfig con valores por defecto
// Útil para tests y para uso rápido
func DecayConfigFromDefaults() DecayConfig {
	return DefaultDecayConfig()
}
