package boomerang

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

// DelegateStep reparte tareas del DAG a subagentes OpenCode.
// Las tareas dentro de un ParallelGroup se ejecutan concurrentemente;
// los grupos se ejecutan en secuencia.
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string) (*DelegateResult, error) {
	result := &DelegateResult{
		TaskResults: make(map[string]TaskResult),
	}

	// Ejecutar grupos paralelos secuencialmente
	for _, group := range dag.ParallelGroups {
		var wg sync.WaitGroup
		var mu sync.Mutex

		for _, taskIdx := range group {
			if taskIdx >= len(dag.Tasks) {
				continue
			}
			task := dag.Tasks[taskIdx]

			wg.Add(1)
			go func(t TaskSpec) {
				defer wg.Done()

				tr := TaskResult{
					TaskName: t.Name,
					Success:  true,
				}

				// Ejecutar subagente OpenCode
				cmd := exec.CommandContext(ctx, "opencode",
					"subagent", t.Agent,
					"--param", fmt.Sprintf("task=%s", t.Name),
					"--param", fmt.Sprintf("phase=%s", phase),
				)

				output, err := cmd.Output()
				if err != nil {
					tr.Success = false
					tr.Output = fmt.Sprintf("error: %v", err)
				} else {
					tr.Output = string(output)
					tr.Nodes = 1
				}

				mu.Lock()
				result.TaskResults[t.Name] = tr
				if tr.Success {
					result.NodesCreated += tr.Nodes
				}
				mu.Unlock()
			}(task)
		}

		wg.Wait()
	}

	return result, nil
}

// opencodeExists verifica si OpenCode está disponible en el PATH.
func opencodeExists() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}
