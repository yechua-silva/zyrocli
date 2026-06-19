package boomerang

import (
	"context"
	"fmt"
	"time"
)

// DelegateStep reparte tareas del DAG a subagentes usando TaskManager.DispatchTask().
// Cada tarea se despacha de forma asíncrona y se espera su resultado con timeout de 30s.
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string) (*DelegateResult, error) {
	result := &DelegateResult{
		TaskResults: make(map[string]TaskResult),
	}

	for _, task := range dag.Tasks {
		taskID := o.taskManager.DispatchTask(ctx, task.Name, task.Agent, phase, nil)

		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		completedTask, err := o.taskManager.WaitTask(waitCtx, taskID)
		cancel()

		tr := TaskResult{
			TaskName: task.Name,
			Success:  err == nil && completedTask.Status == TaskDone,
		}
		if err != nil {
			tr.Output = fmt.Sprintf("Error: %v", err)
		} else {
			tr.Output = completedTask.Output
		}
		result.TaskResults[task.Name] = tr
		result.NodesCreated++
	}

	return result, nil
}
