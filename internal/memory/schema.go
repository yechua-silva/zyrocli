package memory

import "time"

// FactType enum para tipos de hechos
type FactType string

const (
	FactDecision    FactType = "decision"
	FactError       FactType = "error"
	FactPreference  FactType = "preference"
	FactPattern     FactType = "pattern"
	FactDependency  FactType = "dependency"
	FactObservation FactType = "observation"
)

// CausalEdgeType enum para aristas causales
type CausalEdgeType string

const (
	EdgeCaused      CausalEdgeType = "CAUSED"
	EdgePrecedes    CausalEdgeType = "PRECEDES"
	EdgeContradicts CausalEdgeType = "CONTRADICTS"
	EdgeSupports    CausalEdgeType = "SUPPORTS"
	EdgeRequires    CausalEdgeType = "REQUIRES"
	EdgeDerivesFrom CausalEdgeType = "DERIVES_FROM"
	EdgeReferences  CausalEdgeType = "REFERENCES"
)

// Fact representa un hecho atómico en la memoria causal
type Fact struct {
	ID             int64          `json:"$id"`
	Type           FactType       `json:"type"`
	Content        string         `json:"content"`
	Embedding      []float32      `json:"embedding,omitempty"`
	Salience       float64        `json:"salience"`        // 0.0-1.0, importancia
	Confidence     float64        `json:"confidence"`      // 0.0-1.0, fiabilidad
	Source         string         `json:"source"`          // "agent:F0" | "extractor:llm" | "user:input"
	Phase          string         `json:"phase"`           // F0-F4
	CreatedAt      time.Time      `json:"created_at"`
	LastAccessedAt time.Time      `json:"last_accessed_at"`
	AccessCount    int64          `json:"access_count"`
	DecayRate      float64        `json:"decay_rate"` // default 0.05 (Ebbinghaus)
	ExpiresAt      time.Time      `json:"expires_at"`
	IsActive       bool           `json:"is_active"`
	IsStale        bool           `json:"is_stale"`
	ProjectID      string         `json:"project_id"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// CausalEdge representa una arista causal entre dos Facts
type CausalEdge struct {
	ID         int64          `json:"$id"`
	FromID     int64          `json:"from_id"`
	ToID       int64          `json:"to_id"`
	Type       CausalEdgeType `json:"type"`
	CreatedAt  time.Time      `json:"created_at"`
	Properties map[string]any `json:"properties,omitempty"`
}

// ContradictionPair representa un par de hechos contradictorios
type ContradictionPair struct {
	FactA      Fact    `json:"fact_a"`
	FactB      Fact    `json:"fact_b"`
	Similarity float64 `json:"similarity"`
}

// RecallOpts opciones para recuperar memoria
type RecallOpts struct {
	QueryText    string     `json:"query_text"`
	MaxResults   int        `json:"max_results"`
	MinSalience  float64    `json:"min_salience"`
	FactTypes    []FactType `json:"fact_types,omitempty"`
	IncludeStale bool       `json:"include_stale"`
	Phase        string     `json:"phase,omitempty"`
	ProjectID    string     `json:"project_id,omitempty"`
}

// MemoryResult resultado de una consulta de memoria
type MemoryResult struct {
	Fact   Fact    `json:"fact"`
	Score  float64 `json:"score"`
	Source string  `json:"source"` // "vector" | "text" | "hybrid"
}

// DecayConfig configuración del decaimiento
type DecayConfig struct {
	BaseDecayRate     float64 `json:"base_decay_rate"`     // 0.05
	AccessBoost       float64 `json:"access_boost"`        // 0.3
	SalienceThreshold float64 `json:"salience_threshold"`  // 0.15
	MaxSalience       float64 `json:"max_salience"`        // 1.0
	DefaultExpiryDays int     `json:"default_expiry_days"` // 90
}

// DefaultDecayConfig retorna configuración por defecto
func DefaultDecayConfig() DecayConfig {
	return DecayConfig{
		BaseDecayRate:     0.05,
		AccessBoost:       0.3,
		SalienceThreshold: 0.15,
		MaxSalience:       1.0,
		DefaultExpiryDays: 90,
	}
}

// ContradictionStrategy estrategia para resolver contradicciones
type ContradictionStrategy string

const (
	StrategyNewestWins         ContradictionStrategy = "newest_wins"
	StrategyHighestConfidence  ContradictionStrategy = "highest_confidence"
	StrategyKeepBoth           ContradictionStrategy = "keep_both"
)

// FactRow representa un renglón de la tabla de hechos en memoria.
// Usado para resultados de queries directas a la base de datos.
type FactRow struct {
	ID         int64   `json:"id"`
	Type       string  `json:"type"`
	Content    string  `json:"content"`
	Salience   float64 `json:"salience"`
	Confidence float64 `json:"confidence"`
	Phase      string  `json:"phase"`
	IsActive   bool    `json:"is_active"`
}
