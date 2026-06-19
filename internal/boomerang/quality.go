package boomerang

import (
	"context"
	"os/exec"
)

// QualityStep valida que los resultados de la fase sean correctos.
// Para F3 (implementación) verifica que el código compile.
// Para todas las fases verifica que no haya tareas fallidas.
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

	return true, nil
}
