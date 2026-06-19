package boomerang

import (
	"context"
	"time"

	"github.com/secko/zyrocli/internal/boundari"
	"github.com/secko/zyrocli/internal/memory"
	"github.com/secko/zyrocli/internal/tokens"
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

// MeasurementCallback se llama después de cada fase con datos de medición.
type MeasurementCallback func(Measurement)

// Measurement contiene datos de una medición de tokens.
type Measurement struct {
	Phase            string `json:"phase"`
	TaskDescription  string `json:"task_description"`
	WithoutBoomerang int64  `json:"without_boomerang"` // tokens estimados del codebase
	WithBoomerang    int64  `json:"with_boomerang"`    // tokens del contexto inyectado
	OutputTokens     int64  `json:"output_tokens"`     // tokens de respuesta estimados
	CreatedAt        string `json:"created_at"`
}

// BoomerangOrchestrator ejecuta el ciclo de 6 pasos
type BoomerangOrchestrator struct {
	memoryStore         memory.EngramStore
	boundariLoader      func(string) (*boundari.Policy, error)
	taskManager         *TaskManager
	maxIterations       int
	measurementCallback MeasurementCallback
}

// NewBoomerangOrchestrator crea un nuevo orquestador
func NewBoomerangOrchestrator(
	store memory.EngramStore,
	bl func(string) (*boundari.Policy, error),
	tm *TaskManager,
	callback MeasurementCallback,
) *BoomerangOrchestrator {
	return &BoomerangOrchestrator{
		memoryStore:         store,
		boundariLoader:      bl,
		taskManager:         tm,
		maxIterations:       3,
		measurementCallback: callback,
	}
}

// RunPhase ejecuta el ciclo de pasos Boomerang.
// Es un wrapper que convierte PhaseConfig → PhaseConfigV2 por backward compatibility.
func (o *BoomerangOrchestrator) RunPhase(ctx context.Context, config PhaseConfig) (*PhaseResult, error) {
	return o.runPhaseV2(ctx, config.ToV2())
}

// runPhaseV2 ejecuta el ciclo de pasos Boomerang según la configuración PhaseConfigV2.
// Soporta skip matrix, failure policy y modo async (a implementar en fases futuras).
func (o *BoomerangOrchestrator) runPhaseV2(ctx context.Context, config PhaseConfigV2) (*PhaseResult, error) {
	start := time.Now()
	result := &PhaseResult{Phase: config.Phase}

	// Determinar matriz y steps a ejecutar
	matrix := config.SkipMatrix
	if matrix == nil {
		matrix = DefaultPhaseMatrix()
	}
	steps := config.Steps
	if steps == nil {
		steps = matrix.ActiveSteps(config.Phase)
	}

	// Variables compartidas entre steps
	var memoryCtx string
	var dag *TaskDAG
	var delegateResult *DelegateResult
	var gitStatus string
	var qualityOK bool
	var saveResult *SaveResult
	var qualityRan bool

	// Ejecutar steps en orden
	for _, step := range steps {
		switch step {
		case StepMemory:
			mc, err := o.MemoryStep(ctx, config.Phase, config.TaskDesc)
			if err != nil {
				return nil, err
			}
			memoryCtx = mc
			result.MemoryUsed = len(memoryCtx)

		case StepThink:
			d, err := o.ThinkStep(ctx, config.Phase, memoryCtx)
			if err != nil {
				return nil, err
			}
			dag = d
			result.TasksPlanned = len(dag.Tasks)

		case StepDelegate:
			if dag == nil {
				continue
			}
			dr, err := o.DelegateStep(ctx, dag, config.Phase)
			if err != nil {
				result.Error = err.Error()
				result.Duration = time.Since(start)
				return result, nil
			}
			delegateResult = dr
			result.NodesCreated = delegateResult.NodesCreated

		case StepGit:
			gs, err := o.GitStep(ctx)
			if err != nil {
				result.Error = err.Error()
				result.Duration = time.Since(start)
				return result, nil
			}
			gitStatus = gs
			result.GitStatus = gitStatus

		case StepQuality:
			qualityRan = true
			for i := 0; i < o.maxIterations; i++ {
				qok, err := o.QualityStep(ctx, config.Phase, dag, delegateResult)
				if err == nil && qok {
					qualityOK = true
					result.QualityOK = true
					result.Iterations = i + 1
					break
				}
				if i < o.maxIterations-1 {
					if delegateResult != nil && dag != nil {
						delegateResult, _ = o.DelegateStep(ctx, dag, config.Phase)
					}
				}
			}

		case StepSave:
			if delegateResult == nil {
				delegateResult = &DelegateResult{TaskResults: map[string]TaskResult{}}
			}
			sr, err := o.SaveStep(ctx, config.Phase, delegateResult, nil)
			if err == nil {
				saveResult = sr
				result.FactsSaved = saveResult.FactsSaved
			}
		}
	}

	// Estimar tokens (legacy measurement)
	withoutTokens := tokens.Count("Execute phase " + config.Phase + ": " + config.TaskDesc + ". Codebase context: ~3000 chars baseline.")
	withTokens := tokens.Count(memoryCtx)

	// Guardar medición si hay callback
	if o.measurementCallback != nil {
		o.measurementCallback(Measurement{
			Phase:            config.Phase,
			TaskDescription:  config.TaskDesc,
			WithoutBoomerang: withoutTokens,
			WithBoomerang:    withTokens,
			OutputTokens:     0,
			CreatedAt:        time.Now().UTC().Format(time.RFC3339),
		})
	}

	// Success: si Quality se ejecutó, usar su resultado; si no, la fase es exitosa
	if qualityRan {
		result.Success = qualityOK
	} else {
		result.Success = true
	}
	result.Duration = time.Since(start)

	return result, nil
}
