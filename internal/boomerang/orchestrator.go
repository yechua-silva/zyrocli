package boomerang

import (
	"context"
	"time"

	"github.com/secko/zyrocli/internal/boundari"
	"github.com/secko/zyrocli/internal/memory"
)

// PhaseConfig configura la ejecución de una fase
type PhaseConfig struct {
	Phase       string
	TaskDesc    string
	ProjectID   string
	MemoryLimit int
	Iterations  int
	Timeout     time.Duration
}

// PhaseResult resultado completo de una fase Boomerang
type PhaseResult struct {
	Phase        string        `json:"phase"`
	Success      bool          `json:"success"`
	Iterations   int           `json:"iterations"`
	MemoryUsed   int           `json:"memory_used"`
	TasksPlanned int           `json:"tasks_planned"`
	NodesCreated int           `json:"nodes_created"`
	GitStatus    string        `json:"git_status"`
	QualityOK    bool          `json:"quality_ok"`
	FactsSaved   int           `json:"facts_saved"`
	Duration     time.Duration `json:"duration_ms"`
	Error        string        `json:"error,omitempty"`
}

// DelegateResult resultado de delegar tareas a subagentes
type DelegateResult struct {
	NodesCreated int
	TaskResults  map[string]TaskResult
}

// TaskResult resultado de una tarea individual
type TaskResult struct {
	TaskName string
	Success  bool
	Output   string
	Nodes    int
}

// TaskDAG grafo de tareas para una fase
type TaskDAG struct {
	Tasks          []TaskSpec `json:"tasks"`
	Deps           [][2]int   `json:"deps"`
	ParallelGroups [][]int    `json:"parallel_groups"`
}

// TaskSpec especificación de una tarea individual
type TaskSpec struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Agent       string   `json:"agent"`
	Tags        []string `json:"tags,omitempty"`
	DependsOn   []int    `json:"depends_on,omitempty"`
}

// SaveResult resultado de guardar decisiones en memoria causal
type SaveResult struct {
	FactsSaved     int `json:"facts_saved"`
	EdgesCreated   int `json:"edges_created"`
	Contradictions int `json:"contradictions"`
}

// BoomerangOrchestrator ejecuta el ciclo de 6 pasos
type BoomerangOrchestrator struct {
	memoryStore    memory.EngramStore
	boundariLoader func(string) (*boundari.Policy, error)
	maxIterations  int
}

// NewBoomerangOrchestrator crea un nuevo orquestador
func NewBoomerangOrchestrator(
	store memory.EngramStore,
	bl func(string) (*boundari.Policy, error),
) *BoomerangOrchestrator {
	return &BoomerangOrchestrator{
		memoryStore:    store,
		boundariLoader: bl,
		maxIterations:  3,
	}
}

// RunPhase ejecuta el ciclo completo de 6 pasos Boomerang
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
	start := time.Now()
	result := &PhaseResult{Phase: config.Phase}

	// Paso 1: MEMORY — consultar memoria causal
	// (implementado en memory.go)
	memoryCtx, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)
	if err != nil {
		return nil, err
	}
	result.MemoryUsed = len(memoryCtx)

	// Paso 2: THINK — planificar DAG de tareas
	// (implementado en think.go)
	dag, err := o.ThinkStep(ctx, config.Phase, memoryCtx)
	if err != nil {
		return nil, err
	}
	result.TasksPlanned = len(dag.Tasks)

	// Paso 3: DELEGATE — repartir tareas a subagentes
	// (implementado en delegate.go)
	delegateResult, err := o.DelegateStep(ctx, dag, config.Phase)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result, nil
	}
	result.NodesCreated = delegateResult.NodesCreated

	// Paso 4: GIT — verificar estado del repo
	// (implementado en git.go)
	gitStatus, err := o.GitStep(ctx)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result, nil
	}
	result.GitStatus = gitStatus

	// Paso 5: QUALITY — validar resultados (loop de retry)
	// (implementado en quality.go)
	for i := 0; i < o.maxIterations; i++ {
		qualityOK, err := o.QualityStep(ctx, config.Phase, dag, delegateResult)
		if err == nil && qualityOK {
			result.QualityOK = true
			result.Iterations = i + 1
			break
		}
		if i < o.maxIterations-1 {
			// Redelegar tareas fallidas
			delegateResult, _ = o.DelegateStep(ctx, dag, config.Phase)
		}
	}

	// Paso 6: SAVE — guardar decisiones en memoria causal
	// (implementado en save.go)
	saveResult, err := o.SaveStep(ctx, config.Phase, delegateResult, nil)
	if err == nil {
		result.FactsSaved = saveResult.FactsSaved
	}

	result.Success = result.QualityOK
	result.Duration = time.Since(start)

	return result, nil
}
