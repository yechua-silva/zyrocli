package boomerang

import (
	"context"
	"os/exec"
)

// QualityStep valida que los resultados de la fase sean correctos.
// Para F3 (implementación) verifica que el código compile y los tests pasen.
// Para todas las fases verifica que no haya tareas fallidas.
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

	// Verificar tests (para fases de implementación)
	if phase == "F3" {
		if err := exec.CommandContext(ctx, "go", "test", "./...").Run(); err != nil {
			return false, err
		}
	}

	return true, nil
}
