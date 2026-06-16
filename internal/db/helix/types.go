package helix

// Edge represents a directed edge between two nodes.
type Edge struct {
	ID         int64          `json:"id"`
	SourceID   int64          `json:"source_id"`
	TargetID   int64          `json:"target_id"`
	Relation   string         `json:"relation"`
	Properties map[string]any `json:"properties,omitempty"`
}

// SearchResult holds a node with its relevance score.
type SearchResult struct {
	Node  *Node   `json:"node"`
	Score float64 `json:"score,omitempty"`
}

// Row types para resultados de queries — todos con tags JSON.

// TaskRow representa un nodo Task en HelixDB.
type TaskRow struct {
	ID          int    `json:"$id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Phase       string `json:"phase"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// CodeNodeRow representa un nodo CodeNode.
type CodeNodeRow struct {
	ID       int    `json:"$id"`
	Path     string `json:"path"`
	Summary  string `json:"summary"`
	Language string `json:"language"`
	Hash     string `json:"hash"`
}

// FactRow representa un nodo Fact (memoria causal).
type FactRow struct {
	ID         int     `json:"$id"`
	Type       string  `json:"fact_type"`
	Content    string  `json:"content"`
	Salience   float64 `json:"salience"`
	Confidence float64 `json:"confidence"`
	Phase      string  `json:"phase"`
	IsActive   bool    `json:"is_active"`
}

// ProjectRow representa un nodo Project.
type ProjectRow struct {
	ID           int    `json:"$id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CurrentPhase string `json:"current_phase"`
}

// SkillRow representa un nodo Skill.
type SkillRow struct {
	ID      int    `json:"$id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version"`
}

// PatternRow representa un nodo Pattern.
type PatternRow struct {
	ID          int    `json:"$id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Language    string `json:"language"`
}
