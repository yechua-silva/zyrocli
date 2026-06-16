package boomerang

import (
	"context"
	"fmt"
	"time"

	"github.com/secko/zyrocli/internal/memory"
)

// SaveStep guarda decisiones y hechos en memoria causal.
// Cada resultado de tarea se convierte en un Fact persistido en EngramStore.
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte) (*SaveResult, error) {
	result := &SaveResult{}

	if o.memoryStore == nil {
		return result, nil
	}

	// Guardar cada resultado de tarea como un Fact
	for _, tr := range delegateResult.TaskResults {
		if tr.Output == "" {
			continue
		}

		fact := &memory.Fact{
			Type:       memory.FactDecision,
			Content:    fmt.Sprintf("Fase %s - %s: %s", phase, tr.TaskName, truncate(tr.Output, 200)),
			Salience:   0.7,
			Confidence: 0.8,
			Source:     fmt.Sprintf("boomerang:%s", phase),
			Phase:      phase,
			IsActive:   true,
			DecayRate:  0.05,
			CreatedAt:  time.Now(),
		}

		id, err := o.memoryStore.SaveFact(ctx, fact)
		if err == nil && id > 0 {
			result.FactsSaved++
		}
	}

	return result, nil
}

// truncate acorta un string a maxLen caracteres, agregando "..." si es necesario.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
