package helix

// Node represents a graph node in HelixDB.
type Node struct {
	ID         int64          `json:"id"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
}

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

// IndexSpec describes an index configuration on a node property.
type IndexSpec struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"`
	Fields []string `json:"fields"`
}
