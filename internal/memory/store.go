package memory

import (
	"context"
	"fmt"
	"time"

	dbhelix "github.com/yechua-silva/zyrocli/internal/db/helix"
)

// HelixEngramStore implementa EngramStore sobre HelixDB
type HelixEngramStore struct {
	client       *dbhelix.Client
	embeddingSvc *dbhelix.EmbeddingService
	defaultDecay float64
}

// NewHelixEngramStore crea un nuevo store
func NewHelixEngramStore(client *dbhelix.Client, embeddingSvc *dbhelix.EmbeddingService) *HelixEngramStore {
	return &HelixEngramStore{
		client:       client,
		embeddingSvc: embeddingSvc,
		defaultDecay: 0.05,
	}
}

// ---------------------------------------------------------------------------
// T-4.3: Store implementation SaveFact
// ---------------------------------------------------------------------------

// SaveFact guarda un hecho computando embedding si es necesario.
func (s *HelixEngramStore) SaveFact(ctx context.Context, fact *Fact) (int64, error) {
	if fact.Content == "" {
		return 0, fmt.Errorf("memory: fact content is required")
	}

	// Computar embedding si no tiene
	if len(fact.Embedding) == 0 && s.embeddingSvc != nil {
		emb, err := s.embeddingSvc.Embed(ctx, fact.Content)
		if err == nil {
			fact.Embedding = emb
		}
		// Si falla, continuamos sin embedding (BM25 fallback)
	}

	// Construir propiedades
	props := map[string]interface{}{
		"type":       string(fact.Type),
		"content":    fact.Content,
		"salience":   fact.Salience,
		"confidence": fact.Confidence,
		"source":     fact.Source,
		"phase":      fact.Phase,
		"decay_rate": fact.DecayRate,
		"is_active":  fact.IsActive,
		"project_id": fact.ProjectID,
	}

	if !fact.CreatedAt.IsZero() {
		props["created_at"] = fact.CreatedAt.Format(time.RFC3339)
	}
	if len(fact.Metadata) > 0 {
		props["metadata"] = fact.Metadata
	}

	// Crear nodo en HelixDB
	q := dbhelix.CreateFact("Fact", props, fact.Embedding)
	var result struct {
		Fact struct {
			ID int64 `json:"id"`
		} `json:"fact"`
	}
	if err := s.client.Exec(ctx, q, &result); err != nil {
		return 0, fmt.Errorf("memory: save fact: %w", err)
	}

	if result.Fact.ID == 0 {
		return 0, fmt.Errorf("memory: save fact: empty response")
	}

	return result.Fact.ID, nil
}

// SaveFactsBatch guarda múltiples hechos en batch.
// Computa embeddings en batch para eficiencia, luego guarda individualmente.
func (s *HelixEngramStore) SaveFactsBatch(ctx context.Context, facts []*Fact) ([]int64, error) {
	if len(facts) == 0 {
		return nil, nil
	}

	// Batch de embeddings
	if s.embeddingSvc != nil {
		var texts []string
		needEmbedding := make([]int, 0)
		for i, f := range facts {
			if len(f.Embedding) == 0 {
				texts = append(texts, f.Content)
				needEmbedding = append(needEmbedding, i)
			}
		}

		if len(texts) > 0 {
			embeddings, err := s.embeddingSvc.EmbedBatch(ctx, texts)
			if err == nil {
				for j, emb := range embeddings {
					facts[needEmbedding[j]].Embedding = emb
				}
			}
			// Si falla, continuamos sin embeddings (BM25 fallback)
		}
	}

	// Guardar individualmente
	ids := make([]int64, 0, len(facts))
	for _, f := range facts {
		id, err := s.SaveFact(ctx, f)
		if err != nil {
			return ids, fmt.Errorf("memory: batch save at %d: %w", len(ids), err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// AddCausalEdge crea una arista causal entre dos Facts.
func (s *HelixEngramStore) AddCausalEdge(ctx context.Context, edge *CausalEdge) error {
	if edge.FromID == 0 || edge.ToID == 0 {
		return fmt.Errorf("memory: from_id and to_id are required")
	}

	props := map[string]interface{}{
		"type": string(edge.Type),
	}
	if !edge.CreatedAt.IsZero() {
		props["created_at"] = edge.CreatedAt.Format(time.RFC3339)
	}

	q := dbhelix.CreateEdge(int(edge.FromID), int(edge.ToID), string(edge.Type), props)
	return s.client.Exec(ctx, q, nil)
}

// ---------------------------------------------------------------------------
// T-4.4: Batch operations
// ---------------------------------------------------------------------------

// ReinforceSalience refuerza la importancia de hechos accedidos.
// Fórmula: salience += 0.3 * (1 - salience)
func (s *HelixEngramStore) ReinforceSalience(ctx context.Context, factIDs []int64) error {
	for _, id := range factIDs {
		fact, err := s.GetFactByID(ctx, id)
		if err != nil {
			continue // si no encuentra, sigue con el siguiente
		}

		// Fórmula de refuerzo: salience += 0.3 * (1 - salience)
		fact.Salience += 0.3 * (1 - fact.Salience)
		if fact.Salience > 1.0 {
			fact.Salience = 1.0
		}
		fact.AccessCount++
		fact.LastAccessedAt = time.Now()

		// Actualizar en HelixDB
		props := map[string]interface{}{
			"salience":         fact.Salience,
			"access_count":     fact.AccessCount,
			"last_accessed_at": fact.LastAccessedAt.Format(time.RFC3339),
		}

		if err := s.client.UpdateNode(ctx, id, props); err != nil {
			return fmt.Errorf("memory: reinforce salience for fact %d: %w", id, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------


