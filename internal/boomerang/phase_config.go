package boomerang

import "time"

// FailurePolicy define cómo manejar fallos de subagentes durante Delegate.
type FailurePolicy int

const (
	// FailurePolicyFailFast cancela todos los subagentes si uno falla (default para F3).
	FailurePolicyFailFast FailurePolicy = iota
	// FailurePolicyContinueOnError recolecta errores sin abortar (default para F0, F1, F2).
	FailurePolicyContinueOnError
)

func (p FailurePolicy) String() string {
	switch p {
	case FailurePolicyFailFast:
		return "fail_fast"
	case FailurePolicyContinueOnError:
		return "continue_on_error"
	default:
		return "unknown"
	}
}

// PhaseConfigV2 configura una fase con soporte de skip matrix y modo async.
type PhaseConfigV2 struct {
	Phase         string
	TaskDesc      string
	ProjectID     string
	Steps         []Step          // nil = usar SkipMatrix
	Timeout       time.Duration
	Parallelism   int             // subagentes concurrentes máx. (default 3)
	AsyncMode     bool            // true = event loop, false = síncrono legacy
	SkipMatrix    PhaseStepMatrix // nil = usar DefaultPhaseMatrix()
	FailurePolicy FailurePolicy
}

// DefaultPhaseConfigV2 crea una PhaseConfigV2 con valores por defecto para una fase.
func DefaultPhaseConfigV2(phase string) PhaseConfigV2 {
	return PhaseConfigV2{
		Phase:         phase,
		Parallelism:   3,
		AsyncMode:     false,
		FailurePolicy: FailurePolicyFailFast,
	}
}

// ToV2 convierte PhaseConfig legacy a PhaseConfigV2 para backward compatibility.
func (c PhaseConfig) ToV2() PhaseConfigV2 {
	return PhaseConfigV2{
		Phase:     c.Phase,
		TaskDesc:  c.TaskDesc,
		ProjectID: c.ProjectID,
		Timeout:   c.Timeout,
		AsyncMode: false,
	}
}
