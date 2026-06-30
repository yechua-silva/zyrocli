package boomerang

import "fmt"

// Step identifica cada paso del ciclo Boomerang.
type Step int

const (
	StepMemory   Step = iota // 0
	StepThink                // 1
	StepDelegate             // 2
	StepGit                  // 3
	StepQuality              // 4
	StepSave                 // 5
)

func (s Step) String() string {
	switch s {
	case StepMemory:
		return "Memory"
	case StepThink:
		return "Think"
	case StepDelegate:
		return "Delegate"
	case StepGit:
		return "Git"
	case StepQuality:
		return "Quality"
	case StepSave:
		return "Save"
	default:
		return "Unknown"
	}
}

type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepDone
	StepSkipped
	StepFailed
)

func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "pending"
	case StepRunning:
		return "running"
	case StepDone:
		return "done"
	case StepSkipped:
		return "skipped"
	case StepFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type StepOutput struct {
	Step     Step   `json:"step"`
	TaskName string `json:"task_name,omitempty"`
	Output   string `json:"output,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// PhaseStepMatrix define qué pasos ejecutar en cada fase.
type PhaseStepMatrix map[string][]Step

// DefaultPhaseMatrix retorna la matriz canónica con los pasos por defecto para F0-F4.
func DefaultPhaseMatrix() PhaseStepMatrix {
	return PhaseStepMatrix{
		"PRE-F0": {StepMemory, StepThink, StepDelegate, StepSave},
		"F0":     {StepMemory, StepThink, StepDelegate, StepSave},
		"F1": {StepMemory, StepThink, StepDelegate, StepSave},
		"F2": {StepMemory, StepThink, StepDelegate, StepSave},
		"F3": {StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave},
		"F4": {StepMemory, StepDelegate, StepGit, StepSave},
	}
}

// ShouldRun indica si un step debe ejecutarse en una fase dada.
// Para fases no definidas en la matriz, retorna true (default seguro).
func (m PhaseStepMatrix) ShouldRun(phase string, step Step) bool {
	steps, ok := m[phase]
	if !ok {
		return true
	}
	for _, s := range steps {
		if s == step {
			return true
		}
	}
	return false
}

// ActiveSteps retorna la lista de steps activos para una fase.
// Retorna una copia del slice para evitar mutaciones externas de la matriz.
// Para fases no definidas, retorna todos los 6 steps.
func (m PhaseStepMatrix) ActiveSteps(phase string) []Step {
	steps, ok := m[phase]
	if !ok {
		return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
	}
	result := make([]Step, len(steps))
	copy(result, steps)
	return result
}

// AllSteps retorna todos los steps en orden.
func AllSteps() []Step {
	return []Step{StepMemory, StepThink, StepDelegate, StepGit, StepQuality, StepSave}
}

// ErrInvalidMatrix indica que una matriz de fases es inválida.
type ErrInvalidMatrix struct {
	Phase  string
	Reason string
}

func (e *ErrInvalidMatrix) Error() string {
	return fmt.Sprintf("skip matrix: phase %s: %s", e.Phase, e.Reason)
}

// ValidateMatrix valida que una matriz cumpla con todas las reglas:
//   - Todas las fases requeridas (PRE-F0, F0-F4) están definidas
//   - Cada fase tiene al menos un step
//   - No hay steps duplicados
//   - No hay valores de step inválidos
//   - F4 debe incluir StepSave
//   - F3 debe incluir StepQuality y StepGit
func ValidateMatrix(matrix PhaseStepMatrix) error {
	requiredPhases := []string{"PRE-F0", "F0", "F1", "F2", "F3", "F4"}
	maxStep := int(StepSave) // 5

	for _, phase := range requiredPhases {
		steps, ok := matrix[phase]
		if !ok {
			return &ErrInvalidMatrix{Phase: phase, Reason: "phase not defined in matrix"}
		}
		if len(steps) == 0 {
			return &ErrInvalidMatrix{Phase: phase, Reason: "must have at least one step"}
		}

		seen := make(map[Step]bool)
		for _, s := range steps {
			if int(s) < 0 || int(s) > maxStep {
				return &ErrInvalidMatrix{
					Phase: phase, Reason: fmt.Sprintf("invalid step value %d", s),
				}
			}
			if seen[s] {
				return &ErrInvalidMatrix{
					Phase: phase, Reason: fmt.Sprintf("duplicate step %s", s),
				}
			}
			seen[s] = true
		}

		hasStep := func(s Step) bool { return seen[s] }

		if phase == "F4" && !hasStep(StepSave) {
			return &ErrInvalidMatrix{Phase: phase, Reason: "F4 must include Save step"}
		}
		if phase == "F3" {
			if !hasStep(StepQuality) {
				return &ErrInvalidMatrix{Phase: phase, Reason: "F3 must include Quality step"}
			}
			if !hasStep(StepGit) {
				return &ErrInvalidMatrix{Phase: phase, Reason: "F3 must include Git step"}
			}
		}
	}
	return nil
}
