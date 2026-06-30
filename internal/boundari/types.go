package boundari

import "time"

// Action determina qué hacer con una tool
type Action string

const (
	ActionAllow    Action = "allow"
	ActionDeny     Action = "deny"
	ActionApproval Action = "require_approval"
)

// ToolRule define la política para una tool específica
type ToolRule struct {
	Name             string         `yaml:"name"`
	Action           Action         `yaml:"action"`
	RequireApproval  bool           `yaml:"require_approval,omitempty"`
	Conditions       map[string]any `yaml:"conditions,omitempty"`
}

// Budget define límites de ejecución
type Budget struct {
	MaxToolCalls   int     `yaml:"max_tool_calls,omitempty"`
	MaxRuntimeSecs int     `yaml:"max_runtime_seconds,omitempty"`
	MaxCostUSD     float64 `yaml:"max_cost_usd,omitempty"`
}

// Policy representa una política Boundari completa
type Policy struct {
	Version     string     `yaml:"version"`
	Phase       string     `yaml:"phase"`
	Description string     `yaml:"description"`
	Budget      Budget     `yaml:"budget"`
	Tools       []ToolRule `yaml:"tools"`
}

// GetRule busca una ToolRule por nombre. Retorna nil si no existe.
func (p *Policy) GetRule(name string) *ToolRule {
	for i := range p.Tools {
		if p.Tools[i].Name == name {
			return &p.Tools[i]
		}
	}
	return nil
}

// EnforcementResult resultado de verificar una tool
type EnforcementResult struct {
	Allowed bool
	Reason  string
	Tool    string
	Phase   string
}

// AuditEvent evento de auditoría
type AuditEvent struct {
	Timestamp  time.Time `json:"timestamp"`
	Phase      string    `json:"phase"`
	Tool       string    `json:"tool"`
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
	DurationMs int64     `json:"duration_ms,omitempty"`
}

// BudgetUsage consumo actual del presupuesto
type BudgetUsage struct {
	ToolCalls int           `json:"tool_calls"`
	Runtime   time.Duration `json:"runtime"`
	CostUSD   float64       `json:"cost_usd"`
	StartedAt time.Time     `json:"started_at"`
}
