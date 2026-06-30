package boomerang

// CriteriaStatus representa el estado de un criterio de aceptación.
type CriteriaStatus string

const (
	CriteriaPending  CriteriaStatus = "pending"
	CriteriaVerified CriteriaStatus = "verified"
	CriteriaFailed   CriteriaStatus = "failed"
)

// AcceptanceCriteria define un criterio de aceptación individual asociado
// a una tarea del DAG.
type AcceptanceCriteria struct {
	ID          string         `json:"id" yaml:"id"`
	Description string         `json:"description" yaml:"description"`
	Phase       string         `json:"phase" yaml:"phase"`
	Status      CriteriaStatus `json:"status" yaml:"status"`
	Source      string         `json:"source" yaml:"source"`
	TaskID      string         `json:"task_id,omitempty" yaml:"task_id,omitempty"`
}

// CriteriaSummary proporciona un resumen agregado de acceptance criteria.
type CriteriaSummary struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Verified int `json:"verified"`
	Failed   int `json:"failed"`
}

// NewCriteriaSummary computa un CriteriaSummary a partir de un slice de criteria.
func NewCriteriaSummary(criteria []AcceptanceCriteria) *CriteriaSummary {
	s := &CriteriaSummary{}
	for _, c := range criteria {
		s.Total++
		switch c.Status {
		case CriteriaPending:
			s.Pending++
		case CriteriaVerified:
			s.Verified++
		case CriteriaFailed:
			s.Failed++
		}
	}
	return s
}

// ExtractCriteriaFromDAG recolecta todos los acceptance criteria de las tareas del DAG.
func ExtractCriteriaFromDAG(dag *TaskDAG) []AcceptanceCriteria {
	if dag == nil {
		return nil
	}
	var all []AcceptanceCriteria
	for _, task := range dag.Tasks {
		if len(task.AcceptanceCriteria) > 0 {
			all = append(all, task.AcceptanceCriteria...)
		}
	}
	return all
}
