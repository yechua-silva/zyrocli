package taskcontext

// TaskContext contiene todo el contexto relevante para una task
type TaskContext struct {
	TaskID      uint64
	Description string
	Skills      []ContextItem
	CodeNodes   []ContextItem
	Documents   []ContextItem
	Patterns    []ContextItem
}

// ContextItem representa un elemento de contexto (skill, code node, document, pattern)
type ContextItem struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
	Type    string `json:"type,omitempty"`
}
