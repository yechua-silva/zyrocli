package memory

import (
	"context"
	"fmt"
	"math"
	"time"

	dbhelix "github.com/yechua-silva/zyrocli/internal/db/helix"
)

// DetectContradictions detecta hechos contradictorios por similitud semántica
func (s *HelixEngramStore) DetectContradictions(ctx context.Context, projectID string, threshold float64) ([]ContradictionPair, error) {
	if threshold <= 0 {
		threshold = 0.85
	}

	// Si no hay servicio de embeddings, no podemos detectar contradicciones
	if s.embeddingSvc == nil {
		return nil, nil
	}

	// Obtener todos los facts activos del proyecto
	// TODO: implementar listado de facts activos
	// Por ahora, usar FindContradictions de HelixDB
	contradictions, err := dbhelix.FindContradictions(ctx, s.client, threshold)
	if err != nil {
		return nil, fmt.Errorf("memory: detect contradictions: %w", err)
	}

	// Convertir a ContradictionPair
	pairs := make([]ContradictionPair, 0, len(contradictions))
	for i := 0; i < len(contradictions); i += 2 {
		if i+1 < len(contradictions) {
			pairs = append(pairs, ContradictionPair{
				FactA: Fact{
					ID:         int64(contradictions[i].ID),
					Type:       FactType(contradictions[i].Type),
					Content:    contradictions[i].Content,
					Salience:   contradictions[i].Salience,
					Phase:      contradictions[i].Phase,
					IsActive:   contradictions[i].IsActive,
				},
				FactB: Fact{
					ID:         int64(contradictions[i+1].ID),
					Type:       FactType(contradictions[i+1].Type),
					Content:    contradictions[i+1].Content,
					Salience:   contradictions[i+1].Salience,
					Phase:      contradictions[i+1].Phase,
					IsActive:   contradictions[i+1].IsActive,
				},
				Similarity: 0.95, // threshold por defecto
			})
		}
	}

	return pairs, nil
}

// ResolveContradiction resuelve una contradicción usando la estrategia indicada
func (s *HelixEngramStore) ResolveContradiction(ctx context.Context, pair ContradictionPair, strategy ContradictionStrategy) error {
	switch strategy {
	case StrategyNewestWins:
		// El fact más reciente gana, el más viejo se desactiva
		if pair.FactA.CreatedAt.After(pair.FactB.CreatedAt) {
			return s.deactivateFact(ctx, pair.FactB.ID, "contradicted by newer fact")
		}
		return s.deactivateFact(ctx, pair.FactA.ID, "contradicted by newer fact")

	case StrategyHighestConfidence:
		// El fact con mayor confianza gana
		if pair.FactA.Confidence >= pair.FactB.Confidence {
			return s.deactivateFact(ctx, pair.FactB.ID, "contradicted by higher confidence fact")
		}
		return s.deactivateFact(ctx, pair.FactA.ID, "contradicted by higher confidence fact")

	case StrategyKeepBoth:
		// Registrar la ambigüedad, ambos son válidos
		return s.AddCausalEdge(ctx, &CausalEdge{
			FromID:    pair.FactA.ID,
			ToID:      pair.FactB.ID,
			Type:      EdgeContradicts,
			CreatedAt: time.Now(),
			Properties: map[string]any{
				"similarity": pair.Similarity,
				"resolved":   false,
			},
		})

	default:
		return fmt.Errorf("memory: unknown contradiction strategy: %s", strategy)
	}
}

// deactivateFact marca un fact como inactivo en HelixDB
func (s *HelixEngramStore) deactivateFact(ctx context.Context, factID int64, reason string) error {
	if err := s.client.UpdateNode(ctx, factID, map[string]interface{}{
		"is_active":           false,
		"is_stale":            true,
		"deactivated_at":      time.Now().Format(time.RFC3339),
		"deactivation_reason": reason,
	}); err != nil {
		return fmt.Errorf("memory: deactivate fact %d: %w", factID, err)
	}
	return nil
}

// cosineSimilarity calcula la similitud del coseno entre dos vectores
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
