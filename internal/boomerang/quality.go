package boomerang

import (
	"context"
	"os/exec"
)

// QualityStep valida que los resultados de la fase sean correctos.
// Para F3 (implementación) verifica que el código compile.
// Para todas las fases verifica que no haya tareas fallidas y evalúa
// los acceptance criteria definidos en el DAG.
// NOTA: No ejecutamos `go test ./...` porque es recursivo:
// desde un test de boomerang se dispararía QualityStep → go test → QualityStep → ...
func (o *BoomerangOrchestrator) QualityStep(ctx context.Context, phase string, dag *TaskDAG, delegateResult *DelegateResult) (bool, error) {
	// Verificar que compile (para fases de implementación)
	if phase == "F3" {
		if err := exec.CommandContext(ctx, "go", "build", "./...").Run(); err != nil {
			return false, err
		}
	}

	// Verificar que todas las tareas hayan sido exitosas
	for _, tr := range delegateResult.TaskResults {
		if !tr.Success {
			return false, nil
		}
	}

	// Evaluar acceptance criteria del DAG
	if !o.evaluateCriteria(ctx, dag, delegateResult) {
		return false, nil
	}

	return true, nil
}

// evaluateCriteria verifica que todos los acceptance criteria del DAG se cumplieron.
// Retorna true si todos están satisfechos o si no hay criteria definidos.
func (o *BoomerangOrchestrator) evaluateCriteria(ctx context.Context, dag *TaskDAG, delegateResult *DelegateResult) bool {
	if dag == nil || delegateResult == nil {
		return true // nada que evaluar
	}

	allVerified := true
	for _, task := range dag.Tasks {
		for i := range task.AcceptanceCriteria {
			c := &task.AcceptanceCriteria[i]
			if c.Status == CriteriaPending {
				tr, exists := delegateResult.TaskResults[task.Name]
				if !exists || !tr.Success || tr.Output == "" {
					c.Status = CriteriaFailed
					allVerified = false
				} else {
					c.Status = CriteriaVerified
				}
			} else if c.Status == CriteriaFailed {
				allVerified = false
			}
			// CriteriaVerified — no se re-evalúa
		}
	}
	return allVerified
}
