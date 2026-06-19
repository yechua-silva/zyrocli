package boomerang

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/secko/zyrocli/internal/apply"
)

// TaskID es un identificador único de tarea.
type TaskID string

// TaskStatus representa el estado de una tarea.
type TaskStatus int

const (
	TaskPending  TaskStatus = iota
	TaskRunning
	TaskDone
	TaskFailed
	TaskCancelled
)

func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "pending"
	case TaskRunning:
		return "running"
	case TaskDone:
		return "done"
	case TaskFailed:
		return "failed"
	case TaskCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

// MarshalJSON serializes TaskStatus as a string so LLMs can read it.
// Without this, TaskStatus would be serialized as an integer (0,1,2,3,4),
// which confuses the LLM orchestrator.
func (s TaskStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON deserializes TaskStatus from a string or number.
// Supports both formats for backward compatibility:
//   - string: "pending", "running", "done", "failed", "cancelled"
//   - number: 0, 1, 2, 3, 4
func (s *TaskStatus) UnmarshalJSON(data []byte) error {
	// Try as string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		switch str {
		case "pending":
			*s = TaskPending
		case "running":
			*s = TaskRunning
		case "done":
			*s = TaskDone
		case "failed":
			*s = TaskFailed
		case "cancelled":
			*s = TaskCancelled
		default:
			*s = TaskPending
		}
		return nil
	}

	// Fallback: try as number (for backward compatibility with stored data)
	var num int
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*s = TaskStatus(num)
	return nil
}

// Task representa una tarea de subagente en ejecución o completada.
type Task struct {
	ID        TaskID            `json:"id"`
	Name      string            `json:"name"`
	Agent     string            `json:"agent"`
	Params    map[string]string `json:"params,omitempty"`
	Phase     string            `json:"phase"`
	Status    TaskStatus        `json:"status"`
	Output    string            `json:"output,omitempty"`
	Error     string            `json:"error,omitempty"`
	StartedAt time.Time         `json:"started_at"`
	DoneAt    time.Time         `json:"done_at,omitempty"`
}

// TaskManager gestiona un pool de tareas asíncronas.
type TaskManager struct {
	mu            sync.RWMutex
	tasks         map[TaskID]*Task
	counter       int64
	notify        map[TaskID][]chan struct{}
	maxConcurrent int
	activeCount   int
	runner        *apply.Runner
}

// NewTaskManager crea un nuevo TaskManager.
// maxConcurrent = 0 significa sin límite.
func NewTaskManager(maxConcurrent int) *TaskManager {
	return &TaskManager{
		tasks:         make(map[TaskID]*Task),
		notify:        make(map[TaskID][]chan struct{}),
		maxConcurrent: maxConcurrent,
	}
}

// NewTaskManagerWithRunner creates a TaskManager with an apply.Runner for actual task execution.
func NewTaskManagerWithRunner(maxConcurrent int, runner *apply.Runner) *TaskManager {
	tm := NewTaskManager(maxConcurrent)
	tm.runner = runner
	return tm
}

// SetRunner sets the apply.Runner for task execution.
func (tm *TaskManager) SetRunner(runner *apply.Runner) {
	tm.runner = runner
}

// nextID genera un TaskID único.
func (tm *TaskManager) nextID(phase, name string) TaskID {
	tm.mu.Lock()
	tm.counter++
	id := TaskID(fmt.Sprintf("%d-%s-%s-%d", time.Now().Unix(), phase, name, tm.counter))
	tm.mu.Unlock()
	return id
}

// DispatchTask registra y lanza una tarea de forma asíncrona.
// Retorna inmediatamente con el TaskID.
func (tm *TaskManager) DispatchTask(ctx context.Context, name, agent, phase string, params map[string]string) TaskID {
	id := tm.nextID(phase, name)

	task := &Task{
		ID:        id,
		Name:      name,
		Agent:     agent,
		Params:    params,
		Phase:     phase,
		Status:    TaskPending,
		StartedAt: time.Now(),
	}

	tm.mu.Lock()
	tm.tasks[id] = task
	tm.mu.Unlock()

	go tm.executeTask(ctx, task)
	return id
}

// executeTask marca la tarea como despachada y usa el Runner si está configurado.
// Si hay un apply.Runner, ejecuta la tarea a través del pool de gorutinas con
// timeout y fail-fast. Si no hay runner, usa el comportamiento anterior (stub).
func (tm *TaskManager) executeTask(ctx context.Context, task *Task) {
	tm.mu.Lock()
	task.Status = TaskRunning
	tm.activeCount++
	tm.mu.Unlock()

	// If we have a runner, use it for actual execution with pool, timeout, and fail-fast
	if tm.runner != nil {
		// (c) Check if context was cancelled before even starting
		if ctx.Err() != nil {
			tm.failTask(task, fmt.Sprintf("context cancelled: %v", ctx.Err()))
			return
		}

		applyTask := apply.Task{
			ID:   string(task.ID),
			Name: task.Name,
			Execute: func() (string, error) {
				// TODO: In the future, this should actually launch the subagent
				// via OpenCode or the skill system. For now, marks the task
				// as dispatched — the orchestrator handles the real execution.
				return "Tarea despachada. El orquestador la ejecutará.", nil
			},
		}

		results := tm.runner.Run(ctx, []apply.Task{applyTask})

		// (b) Runner returned an error — log and fail
		if len(results) == 0 {
			tm.failTask(task, "runner returned no results")
			return
		}

		result := results[0]

		// (a) Check all statuses — build summary if not all success
		switch result.Status {
		case apply.StatusSuccess:
			tm.CompleteTask(task.ID, result.Output)
		case apply.StatusTimeout:
			tm.failTask(task, "task timed out")
		default:
			errMsg := "task execution failed"
			if result.Error != nil {
				errMsg = result.Error.Error()
			}
			tm.failTask(task, errMsg)
		}
		return
	}

	// Fallback: original stub behavior (no runner configured)
	tm.CompleteTask(task.ID, "Tarea despachada. El orquestador la ejecutará.")
}

// CompleteTask marca una tarea como completada exitosamente.
// La llama el orquestador cuando termina de ejecutar la tarea.
func (tm *TaskManager) CompleteTask(id TaskID, output string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, ok := tm.tasks[id]
	if !ok {
		return false
	}
	if task.Status != TaskRunning {
		return false
	}

	task.Status = TaskDone
	task.Output = output
	task.DoneAt = time.Now()
	tm.activeCount--
	tm.notifyWaiters(id)
	return true
}

// failTask marca la tarea como fallida y notifica waiters.
func (tm *TaskManager) failTask(task *Task, errMsg string) {
	tm.mu.Lock()
	task.Status = TaskFailed
	task.Error = errMsg
	task.DoneAt = time.Now()
	tm.activeCount--
	tm.notifyWaiters(task.ID)
	tm.mu.Unlock()
}

// notifyWaiters desbloquea todos los Wait() pendientes para una tarea.
func (tm *TaskManager) notifyWaiters(id TaskID) {
	for _, ch := range tm.notify[id] {
		close(ch)
	}
	delete(tm.notify, id)
}

// GetTask retorna una copia de la tarea (segura para lectura externa).
func (tm *TaskManager) GetTask(id TaskID) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	t, ok := tm.tasks[id]
	if !ok {
		return nil, false
	}
	// Retornar copia para seguridad concurrente
	copy := *t
	return &copy, true
}

// WaitTask bloquea hasta que la tarea termine o el contexto se cancele.
// Retorna la tarea finalizada.
func (tm *TaskManager) WaitTask(ctx context.Context, id TaskID) (*Task, error) {
	// Primera verificación sin bloqueo
	tm.mu.RLock()
	t, ok := tm.tasks[id]
	if ok && (t.Status == TaskDone || t.Status == TaskFailed || t.Status == TaskCancelled) {
		copy := *t
		tm.mu.RUnlock()
		return &copy, nil
	}
	tm.mu.RUnlock()

	// Suscribirse a notificaciones
	done := make(chan struct{})
	tm.mu.Lock()
	tm.notify[id] = append(tm.notify[id], done)
	tm.mu.Unlock()

	// Esperar
	select {
	case <-done:
		// Tarea completada, retornar resultado
		tm.mu.RLock()
		t := tm.tasks[id]
		var copy *Task
		if t != nil {
			c := *t
			copy = &c
		}
		tm.mu.RUnlock()
		if copy == nil {
			return nil, fmt.Errorf("task %s not found after wait", id)
		}
		return copy, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ListTasks retorna todas las tareas, opcionalmente filtradas por fase.
func (tm *TaskManager) ListTasks(phase string) []*Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var result []*Task
	for _, t := range tm.tasks {
		if phase == "" || t.Phase == phase {
			copy := *t
			result = append(result, &copy)
		}
	}
	return result
}

// CancelTask cancela una tarea en ejecución.
func (tm *TaskManager) CancelTask(id TaskID) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	t, ok := tm.tasks[id]
	if !ok {
		return false
	}
	if t.Status != TaskPending && t.Status != TaskRunning {
		return false
	}

	t.Status = TaskCancelled
	t.Error = "cancelled by user"
	t.DoneAt = time.Now()
	tm.notifyWaiters(id)
	return true
}

// ActiveCount retorna cuántas tareas están ejecutándose actualmente.
func (tm *TaskManager) ActiveCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.activeCount
}

// PhaseDone verifica si todas las tareas de una fase están completadas.
func (tm *TaskManager) PhaseDone(phase string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	for _, t := range tm.tasks {
		if t.Phase == phase {
			switch t.Status {
			case TaskPending, TaskRunning:
				return false
			}
		}
	}
	return true
}

// WaitPhase bloquea hasta que TODAS las tareas de una fase estén completadas.
func (tm *TaskManager) WaitPhase(ctx context.Context, phase string) error {
	// Primera verificación sin bloqueo
	if tm.PhaseDone(phase) {
		return nil
	}

	// Polling con backoff + signals de tareas completadas
	// Usamos un ticker para re-verificar periódicamente
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tm.PhaseDone(phase) {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// PhaseSummary retorna un resumen de estado de una fase.
type PhaseSummary struct {
	Phase     string `json:"phase"`
	Total     int    `json:"total"`
	Done      int    `json:"done"`
	Failed    int    `json:"failed"`
	Running   int    `json:"running"`
	Pending   int    `json:"pending"`
	Cancelled int    `json:"cancelled"`
	AllDone   bool   `json:"all_done"`
}

func (tm *TaskManager) PhaseSummary(phase string) PhaseSummary {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	s := PhaseSummary{Phase: phase}
	for _, t := range tm.tasks {
		if t.Phase == phase {
			s.Total++
			switch t.Status {
			case TaskDone:
				s.Done++
			case TaskFailed:
				s.Failed++
			case TaskRunning:
				s.Running++
			case TaskPending:
				s.Pending++
			case TaskCancelled:
				s.Cancelled++
			}
		}
	}
	s.AllDone = (s.Running == 0 && s.Pending == 0)
	return s
}
