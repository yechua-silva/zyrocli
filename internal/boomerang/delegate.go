package boomerang

import (
	"context"
	"fmt"
	"time"

	"github.com/yechua-silva/zyrocli/internal/boundari"
)

// DelegateStep reparte tareas del DAG a subagentes usando TaskManager.DispatchTask().
// Cada tarea se despacha de forma asíncrona y se espera su resultado con timeout de 30s.
// Si enforcer no es nil, se verifica la política antes de cada DispatchTask.
func (o *BoomerangOrchestrator) DelegateStep(ctx context.Context, dag *TaskDAG, phase string, enforcer *boundari.Enforcer) (*DelegateResult, error) {
	result := &DelegateResult{
		TaskResults: make(map[string]TaskResult),
	}

	for _, task := range dag.Tasks {
		// CheckTool antes de cada DispatchTask
		if enforcer != nil {
			checkResult := enforcer.CheckTool("dispatch_task", map[string]any{
				"task_name": task.Name,
				"agent":     task.Agent,
				"phase":     phase,
			})
			boundari.LogAudit(boundari.AuditEvent{
				Phase:   phase,
				Tool:    "dispatch_task",
				Allowed: checkResult.Allowed,
				Reason:  checkResult.Reason,
			})
			if !checkResult.Allowed {
				tr := TaskResult{
					TaskName: task.Name,
					Success:  false,
					Output:   fmt.Sprintf("denied by boundari: %s", checkResult.Reason),
				}
				result.TaskResults[task.Name] = tr
				result.NodesCreated++
				continue // saltar esta tarea
			}
		}

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
