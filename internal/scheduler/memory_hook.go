package scheduler

import (
	"context"
	"fmt"
	"strings"

	"github.com/secko/zyrocli/internal/memory"
)

// MemoryHooks conecta la memoria causal con el scheduler.
// Inyecta contexto relevante antes de ejecutar una fase (PrePhase)
// y extrae/guarda hechos después de ejecutar una fase (PostPhase).
type MemoryHooks struct {
	store             memory.EngramStore
	factExtractorPath string
}

// NewMemoryHooks crea hooks de memoria causal.
// store: implementación de EngramStore (p.ej. HelixEngramStore).
// factExtractorPath: ruta al script Python fact_extractor.py.
func NewMemoryHooks(store memory.EngramStore, factExtractorPath string) *MemoryHooks {
	return &MemoryHooks{
		store:             store,
		factExtractorPath: factExtractorPath,
	}
}

// PrePhase recupera memorias relevantes y las formatea como contexto
// inyectable en el prompt del agente antes de ejecutar una fase.
// Retorna string vacío si no hay memorias o si el store es nil.
func (h *MemoryHooks) PrePhase(ctx context.Context, phase Phase, taskDesc string) (string, error) {
	if h.store == nil {
		return "", nil
	}

	results, err := h.store.RecallMemories(ctx, memory.RecallOpts{
		QueryText:   taskDesc,
		MaxResults:  10,
		MinSalience: 0.2,
		Phase:       string(phase),
	})
	if err != nil {
		return "", fmt.Errorf("memory pre-phase: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	return formatMemoryForPrompt(results), nil
}

// PostPhase extrae hechos del log de conversación y los persiste en memoria.
// Si el extractor Python está configurado, lo ejecuta para extracción avanzada.
// Por ahora, es un placeholder que no ejecuta el extractor.
// TODO: ejecutar python fact_extractor.py --input <log> --phase <phase>
func (h *MemoryHooks) PostPhase(ctx context.Context, phase Phase, conversationLog string) error {
	if h.store == nil || conversationLog == "" {
		return nil
	}

	// Placeholder: cuando el extractor Python esté listo, se ejecutará aquí.
	// h.runFactExtractor(conversationLog, string(phase))
	_ = h.factExtractorPath

	return nil
}

// formatMemoryForPrompt formatea una lista de MemoryResult como bloque
// de contexto inyectable en el prompt del agente.
func formatMemoryForPrompt(results []*memory.MemoryResult) string {
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
