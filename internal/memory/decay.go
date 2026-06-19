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

	nodes, err := s.client.FindNodes(ctx, "Fact", map[string]interface{}{
		"is_active": true,
	})
	if err != nil {
		return fmt.Errorf("memory: find active facts: %w", err)
	}

	config := DefaultDecayConfig()

	for _, node := range nodes {
		fact, err := s.GetFactByID(ctx, node.ID)
		if err != nil {
			continue // si falla, sigue con el siguiente
		}
		if err := s.applyDecayToFact(ctx, fact, config); err != nil {
			return fmt.Errorf("memory: decay fact %d: %w", node.ID, err)
		}
	}

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
	// BUG #7.5: LastAccessedAt zero value causa time.Since ~1.7e17 horas → decaimiento instantáneo
	if fact.LastAccessedAt.IsZero() {
		fact.LastAccessedAt = fact.CreatedAt
		if fact.LastAccessedAt.IsZero() {
			fact.LastAccessedAt = time.Now()
		}
	}
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

	props := map[string]interface{}{
		"salience":  fact.Salience,
		"is_stale":  fact.IsStale,
		"is_active": fact.IsActive,
	}
	if err := s.client.UpdateNode(ctx, fact.ID, props); err != nil {
		return fmt.Errorf("memory: apply decay to fact %d: %w", fact.ID, err)
	}
	return nil
}

// DecayConfigFromDefaults retorna DecayConfig con valores por defecto
// Útil para tests y para uso rápido
func DecayConfigFromDefaults() DecayConfig {
	return DefaultDecayConfig()
}
