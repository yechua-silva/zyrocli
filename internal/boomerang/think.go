package boomerang

import (
	"context"
	"fmt"
)

// ThinkStep planifica un DAG de tareas según la fase y el contexto de memoria
// recuperado en el paso anterior. Cada fase tiene un DAG predefinido que
// maximiza el paralelismo posible.
func (o *BoomerangOrchestrator) ThinkStep(ctx context.Context, phase, memoryContext string) (*TaskDAG, error) {
	switch phase {
	case "F0":
		return generateDAGForPhase0(), nil
	case "F1":
		return generateDAGForPhase1(), nil
	case "F2":
		return generateDAGForPhase2(), nil
	case "F3":
		return generateDAGForPhase3(), nil
	case "F4":
		return generateDAGForPhase4(), nil
	default:
		return nil, fmt.Errorf("boomerang: unknown phase %s", phase)
	}
}

// generateDAGForPhase0 retorna el DAG de F0: 3 tareas de investigación en paralelo.
func generateDAGForPhase0() *TaskDAG {
	dag := &TaskDAG{ParallelGroups: [][]int{{0, 1, 2}}}
	dag.Tasks = []TaskSpec{
		{ID: 1, Name: "patterns", Description: "Buscar patrones similares", Agent: "zyro-phase-0-patterns", Tags: []string{"research"}},
		{ID: 2, Name: "libraries", Description: "Investigar librerías", Agent: "zyro-phase-0-libraries", Tags: []string{"research"}},
		{ID: 3, Name: "skills", Description: "Detectar skills necesarias", Agent: "zyro-skills-find", Tags: []string{"research"}},
	}
	return dag
}

// generateDAGForPhase1 retorna el DAG de F1: spec + review secuenciales.
func generateDAGForPhase1() *TaskDAG {
	dag := &TaskDAG{}
	dag.Tasks = []TaskSpec{
		{ID: 1, Name: "spec-design", Description: "Diseñar especificación técnica", Agent: "zyro-sdd-spec", Tags: []string{"design"}},
		{ID: 2, Name: "spec-review", Description: "Revisar especificación", Agent: "zyro-sdd-verify", Tags: []string{"review"}},
	}
	dag.Deps = [][2]int{{1, 0}}
	return dag
}

// generateDAGForPhase2 retorna el DAG de F2: diseño técnico.
func generateDAGForPhase2() *TaskDAG {
	dag := &TaskDAG{}
	dag.Tasks = []TaskSpec{
		{ID: 1, Name: "technical-design", Description: "Diseño técnico detallado", Agent: "zyro-sdd-design", Tags: []string{"design"}},
	}
	return dag
}

// generateDAGForPhase3 retorna el DAG de F3: implementación + verificación en paralelo.
func generateDAGForPhase3() *TaskDAG {
	dag := &TaskDAG{ParallelGroups: [][]int{{0, 1}}}
	dag.Tasks = []TaskSpec{
		{ID: 1, Name: "implement", Description: "Implementar cambios según spec", Agent: "zyro-sdd-apply", Tags: []string{"implementation"}},
		{ID: 2, Name: "verify", Description: "Verificar implementación", Agent: "zyro-sdd-verify", Tags: []string{"verification"}},
	}
	return dag
}

// generateDAGForPhase4 retorna el DAG de F4: archivo y cierre.
func generateDAGForPhase4() *TaskDAG {
	dag := &TaskDAG{}
	dag.Tasks = []TaskSpec{
		{ID: 1, Name: "archive", Description: "Archivar y cerrar fase", Agent: "zyro-sdd-archive", Tags: []string{"close"}},
	}
	return dag
}
