package helix

import (
	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// SearchResultRow resultado de búsqueda híbrida.
type SearchResultRow struct {
	ID      int     `json:"id"`
	Label   string  `json:"label"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
	Source  string  `json:"source"` // "vector" | "text"
}

// HybridSearchOptions opciones para búsqueda híbrida.
type HybridSearchOptions struct {
	MaxResults int
	NodeLabels []string
	MinScore   float64
}

// DefaultHybridSearchOptions retorna opciones por defecto.
func DefaultHybridSearchOptions() HybridSearchOptions {
	return HybridSearchOptions{
		MaxResults: 10,
		MinScore:   0.0,
	}
}

// ---------------------------------------------------------------------------
// Query Builders — retornan helixsdk.Request listos para ejecutar
// ---------------------------------------------------------------------------

// FindTask busca una tarea por nombre en un proyecto.
func FindTask(name string, projectID int) helixsdk.Request {
	return helixsdk.ReadQuery("find_task").
		VarAs("task",
			helixsdk.G().NWithLabel("Task").
				Where(helixsdk.PredEq("name", name)).
				Where(helixsdk.PredEq("project_id", projectID)),
		).
		Returning("task")
}

// UpsertCodeNode crea o actualiza un CodeNode.
// Primero busca un nodo existente; si no existe, crea uno nuevo.
func UpsertCodeNode(projectID int, path, summary, language, hash string) helixsdk.Request {
	return helixsdk.WriteQuery("upsert_code_node").
		VarAs("existing",
			helixsdk.G().NWithLabel("CodeNode").
				Where(helixsdk.PredEq("path", path)).
				Where(helixsdk.PredEq("project_id", projectID)),
		).
		VarAsIf("node",
			helixsdk.VarEmpty("existing"),
			helixsdk.G().AddN("CodeNode", helixsdk.Props{
				helixsdk.Prop("path", path),
				helixsdk.Prop("summary", summary),
				helixsdk.Prop("language", language),
				helixsdk.Prop("hash", hash),
				helixsdk.Prop("project_id", projectID),
			}),
		).
		Returning("node")
}

// FindProject busca un proyecto por nombre.
func FindProject(name string) helixsdk.Request {
	return helixsdk.ReadQuery("find_project").
		VarAs("project",
			helixsdk.G().NWithLabel("Project").
				Where(helixsdk.PredEq("name", name)),
		).
		Returning("project")
}

// ListFactsByPhase lista Facts activos de una fase.
func ListFactsByPhase(phase string, limit int) helixsdk.Request {
	if limit <= 0 {
		limit = 10
	}
	return helixsdk.ReadQuery("list_facts").
		VarAs("facts",
			helixsdk.G().NWithLabel("Fact").
				Where(helixsdk.PredEq("phase", phase)).
				Where(helixsdk.PredEq("is_active", true)).
				Limit(limit),
		).
		Returning("facts")
}

// ---------------------------------------------------------------------------
// Query Builders — Parte 2 (T-2.5)
// ---------------------------------------------------------------------------

// CreateFact crea un nodo Fact con embedding opcional.
func CreateFact(label string, props map[string]interface{}, embedding []float32) helixsdk.Request {
	var pairs helixsdk.Props
	for k, v := range props {
		pairs = append(pairs, helixsdk.Prop(k, v))
	}
	q := helixsdk.WriteQuery("create_fact").
		VarAs("fact", helixsdk.G().AddN(label, pairs))
	if embedding != nil {
		q = q.VarAs("fact_with_embedding",
			helixsdk.G().N(helixsdk.NodeVar("fact")).
				SetProperty("embedding", embedding),
		)
	}
	return q.Returning("fact")
}

// CreateEdge crea un edge entre dos nodos.
func CreateEdge(fromID, toID int, edgeType string, props map[string]interface{}) helixsdk.Request {
	var pairs helixsdk.Props
	for k, v := range props {
		pairs = append(pairs, helixsdk.Prop(k, v))
	}
	return helixsdk.WriteQuery("create_edge").
		VarAs("edge",
			helixsdk.G().N(helixsdk.NodeID(uint64(fromID))).
				AddE(edgeType, helixsdk.NodeID(uint64(toID)), pairs),
		).
		Returning("edge")
}

// FindSkills busca skills por texto.
func FindSkills(query string, limit int) helixsdk.Request {
	if limit <= 0 {
		limit = 10
	}
	return helixsdk.ReadQuery("find_skills").
		VarAs("skills",
			helixsdk.G().NWithLabel("Skill").
				TextSearchNodes("Skill", "name", query, limit),
		).
		Returning("skills")
}

// FindPatterns busca patrones por texto y lenguaje.
func FindPatterns(query, language string, limit int) helixsdk.Request {
	if limit <= 0 {
		limit = 10
	}
	q := helixsdk.ReadQuery("find_patterns").
		VarAs("patterns",
			helixsdk.G().NWithLabel("Pattern").
				TextSearchNodes("Pattern", "name", query, limit),
		)
	if language != "" {
		q = q.VarAs("filtered_patterns",
			helixsdk.G().N(helixsdk.NodeVar("patterns")).
				Where(helixsdk.PredEq("language", language)),
		)
		return q.Returning("patterns", "filtered_patterns")
	}
	return q.Returning("patterns")
}

// AddIndex crea un índice si no existe.
func AddIndex(label, property, indexType string) helixsdk.Request {
	var spec helixsdk.IndexSpec
	switch indexType {
	case "text":
		spec = helixsdk.NodeTextIndex(label, property)
	case "vector":
		spec = helixsdk.NodeVectorIndex(label, property)
	case "equality":
		spec = helixsdk.NodeEqualityIndex(label, property)
	case "range":
		spec = helixsdk.NodeRangeIndex(label, property)
	default:
		spec = helixsdk.NodeTextIndex(label, property)
	}
	return helixsdk.WriteQuery("add_index").
		VarAs("index",
			helixsdk.G().CreateIndexIfNotExists(spec),
		).
		Returning("index")
}
