package boomerang

import (
	"context"
	"fmt"
	"time"

	"github.com/secko/zyrocli/internal/boundari"
	"github.com/secko/zyrocli/internal/memory"
)

// SaveStep guarda decisiones y hechos en memoria causal.
// Cada resultado de tarea se convierte en un Fact persistido en EngramStore.
// Si enforcer no es nil, se verifica la política antes de cada SaveFact.
// acceptanceCriteria es opcional — si se provee, registra el estado de cada criterion como fact.
func (o *BoomerangOrchestrator) SaveStep(ctx context.Context, phase string, delegateResult *DelegateResult, logData []byte, enforcer *boundari.Enforcer, acceptanceCriteria []AcceptanceCriteria) (*SaveResult, error) {
	result := &SaveResult{}

	if o.memoryStore == nil {
		return result, nil
	}

	// Guardar cada resultado de tarea como un Fact
	for _, tr := range delegateResult.TaskResults {
		// CheckTool antes de cada SaveFact
		if enforcer != nil {
			checkResult := enforcer.CheckTool("save_to_helix", map[string]any{
				"task_name": tr.TaskName,
				"phase":     phase,
			})
			boundari.LogAudit(boundari.AuditEvent{
				Phase:   phase,
				Tool:    "save_to_helix",
				Allowed: checkResult.Allowed,
				Reason:  checkResult.Reason,
			})
			if !checkResult.Allowed {
				continue // saltar este fact
			}
		}

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

	// Si hay acceptance criteria, guardar su estado como hechos en memoria causal
	if len(acceptanceCriteria) > 0 {
		for _, c := range acceptanceCriteria {
			if o.memoryStore == nil {
				continue
			}

			if enforcer != nil {
				checkResult := enforcer.CheckTool("save_to_helix", map[string]any{
					"criteria_id": c.ID,
					"phase":       phase,
				})
				boundari.LogAudit(boundari.AuditEvent{
					Phase:   phase,
					Tool:    "save_to_helix",
					Allowed: checkResult.Allowed,
					Reason:  checkResult.Reason,
				})
				if !checkResult.Allowed {
					continue
				}
			}

			content := fmt.Sprintf("Criteria %s [%s]: %s → %s", c.ID, c.Phase, c.Description, c.Status)
			fact := &memory.Fact{
				Type:       memory.FactDecision,
				Content:    truncate(content, 200),
				Salience:   0.6,
				Confidence: 0.9,
				Source:     fmt.Sprintf("boomerang:criteria:%s", phase),
				Phase:      phase,
				IsActive:   true,
				DecayRate:  0.05,
				CreatedAt:  time.Now(),
			}
			if id, err := o.memoryStore.SaveFact(ctx, fact); err == nil && id > 0 {
				result.FactsSaved++
			}
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
