package boomerang

import (
	"context"
	"fmt"
	"strings"

	"github.com/secko/zyrocli/internal/memory"
	"github.com/secko/zyrocli/internal/tokens"
)

// MemoryStep consulta memoria causal relevante para la fase actual.
// Retorna un string formateado con los hechos previos, o vacío si no hay
// memoria disponible.
func (o *BoomerangOrchestrator) MemoryStep(ctx context.Context, phase, taskDesc string) (string, error) {
	if o.memoryStore == nil {
		return "", nil
	}

	results, err := o.memoryStore.RecallMemories(ctx, memory.RecallOpts{
		QueryText:   taskDesc,
		MaxResults:  10,
		MinSalience: 0.2,
		Phase:       phase,
	})
	if err != nil {
		return "", fmt.Errorf("boomerang memory: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## MEMORIA CAUSAL (%d hechos previos)\n\n", len(results)))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s (%.0f%% confianza, fase %s)\n",
			i+1, r.Fact.Type, r.Fact.Content, r.Fact.Confidence*100, r.Fact.Phase))
	}

	memoryCtx := sb.String()

	// Medir tokens del contexto inyectado
	_ = tokens.Count(memoryCtx) // usado por el orchestrator

	return memoryCtx, nil
}
