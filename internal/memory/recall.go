package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	dbhelix "github.com/secko/zyrocli/internal/db/helix"
)

// ---------------------------------------------------------------------------
// T-4.5: RecallMemories
// ---------------------------------------------------------------------------

// RecallMemories busca hechos relevantes usando búsqueda híbrida.
// Si no hay embeddings disponibles, cae a solo BM25 text search.
func (s *HelixEngramStore) RecallMemories(ctx context.Context, opts RecallOpts) ([]*MemoryResult, error) {
	if opts.MaxResults <= 0 {
		opts.MaxResults = 10
	}
	if opts.MinSalience <= 0 {
		opts.MinSalience = 0.2
	}

	// Defaults de HybridSearch
	searchOpts := dbhelix.DefaultHybridSearchOptions()
	searchOpts.MaxResults = opts.MaxResults

	// Si tenemos embedding service, computar embedding
	var embedding []float32
	if s.embeddingSvc != nil {
		emb, err := s.embeddingSvc.Embed(ctx, opts.QueryText)
		if err == nil {
			embedding = emb
		}
	}

	// Ejecutar búsqueda híbrida (o solo BM25 si embedding está vacío)
	results, err := dbhelix.HybridSearch(ctx, s.client, opts.QueryText, embedding, searchOpts)
	if err != nil {
		return nil, fmt.Errorf("memory: recall: %w", err)
	}

	// Convertir a MemoryResult
	memResults := make([]*MemoryResult, 0, len(results))
	for _, r := range results {
		fact := &Fact{
			ID:          int64(r.ID),
			Content:     r.Content,
			Salience:    r.Salience,
			Confidence:  r.Confidence,
			Phase:       r.Phase,
			ProjectID:   r.ProjectID,
			IsActive:    r.IsActive,
			IsStale:     r.IsStale,
			AccessCount: r.AccessCount,
			DecayRate:   r.DecayRate,
		}
		if r.Label != "" {
			fact.Type = FactType(r.Label)
		}
		// Parsear fechas si no están vacías
		if r.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, r.CreatedAt); err == nil {
				fact.CreatedAt = t
			}
		}
		if r.LastAccessedAt != "" {
			if t, err := time.Parse(time.RFC3339, r.LastAccessedAt); err == nil {
				fact.LastAccessedAt = t
			}
		}

		memResults = append(memResults, &MemoryResult{
			Fact:   *fact,
			Score:  r.Score,
			Source: r.Source,
		})
	}

	// Aplicar filtros post-búsqueda
	memResults = applyFilters(memResults, opts)

	return memResults, nil
}

// applyFilters aplica filtros de salience, tipo, stale, phase
func applyFilters(results []*MemoryResult, opts RecallOpts) []*MemoryResult {
	filtered := make([]*MemoryResult, 0)

	for _, r := range results {
		// Filtro por salience mínima
		if r.Fact.Salience < opts.MinSalience {
			continue
		}

		// Filtro por tipos específicos
		if len(opts.FactTypes) > 0 {
			found := false
			for _, t := range opts.FactTypes {
				if r.Fact.Type == t {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filtro por stale
		if !opts.IncludeStale && r.Fact.IsStale {
			continue
		}

		// Filtro por fase
		if opts.Phase != "" && r.Fact.Phase != opts.Phase {
			continue
		}

		filtered = append(filtered, r)
	}

	// Re-ordenar por score descendente
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	// Limitar a maxResults
	if len(filtered) > opts.MaxResults {
		filtered = filtered[:opts.MaxResults]
	}

	return filtered
}

// ---------------------------------------------------------------------------
// T-4.6: GetCausalChain
// ---------------------------------------------------------------------------

// GetCausalChain navega la cadena causal desde un fact
func (s *HelixEngramStore) GetCausalChain(ctx context.Context, factID int64, maxDepth int) ([]*Fact, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	// Usar el traversal causal de HelixDB
	chain, err := dbhelix.TraverseCausalChain(ctx, s.client, int(factID), maxDepth)
	if err != nil {
		return nil, fmt.Errorf("memory: causal chain from %d: %w", factID, err)
	}

	// Convertir a Facts
	facts := make([]*Fact, 0, len(chain))
	for _, fp := range chain {
		facts = append(facts, &Fact{
			ID:         int64(fp.Fact.ID),
			Type:       FactType(fp.Fact.Type),
			Content:    fp.Fact.Content,
			Salience:   fp.Fact.Salience,
			Confidence: fp.Fact.Confidence,
			Phase:      fp.Fact.Phase,
			IsActive:   fp.Fact.IsActive,
		})
	}

	return facts, nil
}

// formatMemoryForPrompt formatea una lista de MemoryResult como bloque
// de contexto inyectable en el prompt del agente.
func formatMemoryForPrompt(results []*MemoryResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## MEMORIA CAUSAL (hechos previos relevantes)\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] ", i+1, r.Fact.Type))
		sb.WriteString(r.Fact.Content)
		sb.WriteString(fmt.Sprintf(" (confianza: %.0f%%, fase: %s)\n",
			r.Fact.Confidence*100, r.Fact.Phase))
	}

	return sb.String()
}

// GetFactByID obtiene un fact por su ID (versión mejorada)
// Reemplaza la versión básica en store.go
func (s *HelixEngramStore) GetFactByID(ctx context.Context, factID int64) (*Fact, error) {
	// Usar el traversal para obtener el fact con sus relaciones
	facts, err := s.GetCausalChain(ctx, factID, 1)
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return nil, fmt.Errorf("memory: fact %d not found", factID)
	}
	return facts[0], nil
}
