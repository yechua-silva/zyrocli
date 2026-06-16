package memory

import "context"

// EngramStore define las operaciones de memoria causal.
// Implementación concreta: HelixEngramStore (store.go)
type EngramStore interface {
	// SaveFact guarda un hecho con su embedding en HelixDB.
	// Si embedding está vacío, lo computa automáticamente.
	SaveFact(ctx context.Context, fact *Fact) (int64, error)

	// SaveFactsBatch guarda múltiples hechos en batch.
	// Computa embeddings en batch para eficiencia.
	SaveFactsBatch(ctx context.Context, facts []*Fact) ([]int64, error)

	// AddCausalEdge crea una arista causal entre dos Facts.
	AddCausalEdge(ctx context.Context, edge *CausalEdge) error

	// RecallMemories busca hechos relevantes usando búsqueda híbrida.
	// Si no hay embeddings disponibles, cae a BM25 text search.
	RecallMemories(ctx context.Context, opts RecallOpts) ([]*MemoryResult, error)

	// GetCausalChain navega la cadena causal desde un fact.
	GetCausalChain(ctx context.Context, factID int64, maxDepth int) ([]*Fact, error)

	// GetFactByID obtiene un fact por su ID.
	GetFactByID(ctx context.Context, factID int64) (*Fact, error)

	// DetectContradictions detecta hechos contradictorios por similitud semántica.
	DetectContradictions(ctx context.Context, projectID string, threshold float64) ([]ContradictionPair, error)

	// ResolveContradiction resuelve una contradicción usando la estrategia indicada.
	ResolveContradiction(ctx context.Context, pair ContradictionPair, strategy ContradictionStrategy) error

	// ReinforceSalience refuerza la importancia de hechos accedidos.
	ReinforceSalience(ctx context.Context, factIDs []int64) error

	// DecayAndRefresh aplica decaimiento temporal (Ebbinghaus) a todos los facts.
	DecayAndRefresh(ctx context.Context, projectID string) error
}
