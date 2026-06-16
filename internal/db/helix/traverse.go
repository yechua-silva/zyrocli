// Package helix — Traversals (T-2.8, T-2.9).
//
// Implementa navegación de grafos: contexto de proyecto,
// descubrimiento cross-project de skills, y cadenas causales.
package helix

import (
	"context"
	"fmt"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// ---------------------------------------------------------------------------
// Tipos de salida
// ---------------------------------------------------------------------------

// ProjectContext agrupa contexto completo de un proyecto.
type ProjectContext struct {
	Project  ProjectRow   `json:"project"`
	Tasks    []TaskRow    `json:"tasks"`
	Skills   []SkillRow   `json:"skills"`
	Patterns []PatternRow `json:"patterns"`
}

// FactWithPath asocia un fact con su camino causal.
type FactWithPath struct {
	Fact     FactRow `json:"fact"`
	Depth    int     `json:"depth"`
	RelType  string  `json:"relation_type"`
}

// ---------------------------------------------------------------------------
// T-2.8: Traversals básicos
// ---------------------------------------------------------------------------

// DiscoverCrossProjectSkills descubre proyectos que comparten un skill
// navegando Skill ← REQUIRES_SKILL ← Project.
func DiscoverCrossProjectSkills(ctx context.Context, client *Client, skillName string) ([]ProjectRow, error) {
	q := helixsdk.ReadQuery("cross_project_skills").
		VarAs("skills",
			helixsdk.G().NWithLabel("Skill").
				Where(helixsdk.PredEq("name", skillName)),
		).
		VarAs("projects",
			helixsdk.G().N(helixsdk.NodeVar("skills")).In("REQUIRES_SKILL"),
		)

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, fmt.Errorf("discover skills: %w", err)
	}

	return parseProjects(raw)
}

// TraverseProjectContext arma el contexto completo de un proyecto navegando
// out-edges: HAS_TASK, REQUIRES_SKILL, FOLLOWS.
func TraverseProjectContext(ctx context.Context, client *Client, projectID int) (*ProjectContext, error) {
	q := helixsdk.ReadQuery("project_context").
		VarAs("project",
			helixsdk.G().N(helixsdk.NodeID(uint64(projectID))).ValueMap(),
		).
		VarAs("tasks",
			helixsdk.G().N(helixsdk.NodeID(uint64(projectID))).Out("HAS_TASK").ValueMap(),
		).
		VarAs("skills",
			helixsdk.G().N(helixsdk.NodeID(uint64(projectID))).Out("REQUIRES_SKILL").ValueMap(),
		).
		VarAs("patterns",
			helixsdk.G().N(helixsdk.NodeID(uint64(projectID))).Out("FOLLOWS").ValueMap(),
		)

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, fmt.Errorf("traverse project: %w", err)
	}

	ctx2 := &ProjectContext{}
	if proj, ok := raw["project"].([]interface{}); ok && len(proj) > 0 {
		if m, ok := proj[0].(map[string]interface{}); ok {
			ctx2.Project = parseProjectRow(m)
		}
	}
	if tasks, ok := raw["tasks"].([]interface{}); ok {
		for _, t := range tasks {
			if m, ok := t.(map[string]interface{}); ok {
				ctx2.Tasks = append(ctx2.Tasks, parseTaskRow(m))
			}
		}
	}
	if skills, ok := raw["skills"].([]interface{}); ok {
		for _, s := range skills {
			if m, ok := s.(map[string]interface{}); ok {
				ctx2.Skills = append(ctx2.Skills, parseSkillRow(m))
			}
		}
	}
	if patterns, ok := raw["patterns"].([]interface{}); ok {
		for _, p := range patterns {
			if m, ok := p.(map[string]interface{}); ok {
				ctx2.Patterns = append(ctx2.Patterns, parsePatternRow(m))
			}
		}
	}

	return ctx2, nil
}

// ---------------------------------------------------------------------------
// T-2.9: Traversals causales
// ---------------------------------------------------------------------------

// TraverseCausalChain navega la cadena causal desde un fact usando
// Repeat(Out(CAUSED|PRECEDES|DERIVES_FROM)) con maxDepth.
func TraverseCausalChain(ctx context.Context, client *Client, factID, maxDepth int) ([]FactWithPath, error) {
	if maxDepth <= 0 {
		maxDepth = 5
	}

	q := helixsdk.ReadQuery("causal_chain").
		VarAs("source",
			helixsdk.G().N(helixsdk.NodeID(uint64(factID))),
		).
		VarAs("chain",
			helixsdk.G().N(helixsdk.NodeVar("source")).
				Repeat(
					helixsdk.Repeat(
						helixsdk.Sub().Out("CAUSED", "PRECEDES", "DERIVES_FROM"),
					).WithMaxDepth(maxDepth),
				).
				Dedup().
				ValueMap(),
		)

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, fmt.Errorf("traverse causal: %w", err)
	}

	return parseFactsWithPath(raw, "chain"), nil
}

// FindContradictions busca nodos con label CONTRADICTS.
func FindContradictions(ctx context.Context, client *Client, threshold float64) ([]FactRow, error) {
	if threshold <= 0 {
		threshold = 0.85
	}

	q := helixsdk.ReadQuery("find_contradictions").
		VarAs("contradictions",
			helixsdk.G().NWithLabel("CONTRADICTS").ValueMap(),
		)

	var raw map[string]interface{}
	if err := client.Exec(ctx, q, &raw); err != nil {
		return nil, fmt.Errorf("find contradictions: %w", err)
	}

	return parseContradictions(raw)
}

// ---------------------------------------------------------------------------
// Helpers de parseo para rows de la BD
// ---------------------------------------------------------------------------

func parseProjectRow(m map[string]interface{}) ProjectRow {
	r := ProjectRow{}
	if id, ok := m["$id"].(float64); ok {
		r.ID = int(id)
	}
	if name, ok := m["name"].(string); ok {
		r.Name = name
	}
	if desc, ok := m["description"].(string); ok {
		r.Description = desc
	}
	if status, ok := m["status"].(string); ok {
		r.Status = status
	}
	return r
}

func parseTaskRow(m map[string]interface{}) TaskRow {
	r := TaskRow{}
	if id, ok := m["$id"].(float64); ok {
		r.ID = int(id)
	}
	if name, ok := m["name"].(string); ok {
		r.Name = name
	}
	if phase, ok := m["phase"].(string); ok {
		r.Phase = phase
	}
	if status, ok := m["status"].(string); ok {
		r.Status = status
	}
	return r
}

func parseSkillRow(m map[string]interface{}) SkillRow {
	r := SkillRow{}
	if id, ok := m["$id"].(float64); ok {
		r.ID = int(id)
	}
	if name, ok := m["name"].(string); ok {
		r.Name = name
	}
	if typ, ok := m["type"].(string); ok {
		r.Type = typ
	}
	return r
}

func parsePatternRow(m map[string]interface{}) PatternRow {
	r := PatternRow{}
	if id, ok := m["$id"].(float64); ok {
		r.ID = int(id)
	}
	if name, ok := m["name"].(string); ok {
		r.Name = name
	}
	if desc, ok := m["description"].(string); ok {
		r.Description = desc
	}
	return r
}

func parseFactRow(m map[string]interface{}) FactRow {
	r := FactRow{}
	if id, ok := m["$id"].(float64); ok {
		r.ID = int(id)
	}
	if typ, ok := m["fact_type"].(string); ok {
		r.Type = typ
	}
	if content, ok := m["content"].(string); ok {
		r.Content = content
	}
	if salience, ok := m["salience"].(float64); ok {
		r.Salience = salience
	}
	if confidence, ok := m["confidence"].(float64); ok {
		r.Confidence = confidence
	}
	if phase, ok := m["phase"].(string); ok {
		r.Phase = phase
	}
	if active, ok := m["is_active"].(bool); ok {
		r.IsActive = active
	}
	return r
}

// ---------------------------------------------------------------------------
// Helpers de parseo para colecciones
// ---------------------------------------------------------------------------

func parseProjects(raw map[string]interface{}) ([]ProjectRow, error) {
	projects, ok := raw["projects"].([]interface{})
	if !ok {
		return nil, nil
	}
	var result []ProjectRow
	for _, p := range projects {
		if m, ok := p.(map[string]interface{}); ok {
			result = append(result, parseProjectRow(m))
		}
	}
	return result, nil
}

func parseFactsWithPath(raw map[string]interface{}, key string) []FactWithPath {
	var result []FactWithPath
	if items, ok := raw[key].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				fp := FactWithPath{}
				fp.Fact = parseFactRow(m)
				if depth, ok := m["depth"].(float64); ok {
					fp.Depth = int(depth)
				}
				if rel, ok := m["relation_type"].(string); ok {
					fp.RelType = rel
				}
				result = append(result, fp)
			}
		}
	}
	return result
}

func parseContradictions(raw map[string]interface{}) ([]FactRow, error) {
	var result []FactRow
	if items, ok := raw["contradictions"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, parseFactRow(m))
			}
		}
	}
	return result, nil
}
