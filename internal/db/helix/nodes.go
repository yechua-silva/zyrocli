package helix

import (
	"context"
	"errors"
	"fmt"
	"strings"

	helixsdk "github.com/helixdb/helix-db/sdks/go"

	"github.com/secko/zyrocli/internal/git"
)

// projectIDStr converts a uint64 project node ID to its property string form.
// CodeNode's project_id property stores the project's HelixDB node ID as a string.
func projectIDStr(id uint64) string {
	return fmt.Sprintf("%d", id)
}

// mapToProps converts a map to HelixDB Props (ordered key-value pairs).
func mapToProps(props map[string]interface{}) helixsdk.Props {
	result := make(helixsdk.Props, 0, len(props))
	for k, v := range props {
		result = append(result, helixsdk.Prop(k, v))
	}
	return result
}

// CreateNode creates a node with the given label and properties,
// auto-injecting project_id if configured. Returns the new node ID.
func (c *Client) CreateNode(ctx context.Context, label string, props map[string]interface{}) (int64, error) {
	props = c.InjectProject(props)

	q := helixsdk.WriteQuery("create_node").
		VarAs("node",
			helixsdk.G().AddN(label, mapToProps(props)).
				Project(helixsdk.ProjectPropAs("$id", "id")),
		).
		Returning("node")

	var result struct {
		Node []struct {
			ID int64 `json:"id"`
		} `json:"node"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return 0, fmt.Errorf("helix: create node: %w", err)
	}

	if len(result.Node) == 0 {
		return 0, ErrNotFound
	}

	return result.Node[0].ID, nil
}

// GetNode retrieves a node by ID with its key properties.
// Returns ErrNotFound if the node does not exist.
// Properties includes: project_id, description, and any other stored fields.
func (c *Client) GetNode(ctx context.Context, id int64) (*Node, error) {
	q := helixsdk.ReadQuery("get_node").
		VarAs("node",
			helixsdk.G().N(helixsdk.NodeID(uint64(id))).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
					helixsdk.ProjectProp("project_id"),
					helixsdk.ProjectProp("description"),
					helixsdk.ProjectProp("name"),
				),
		).
		Returning("node")

	var result struct {
		Node []struct {
			ID          int64  `json:"id"`
			Label       string `json:"label"`
			ProjectID   string `json:"project_id"`
			Description string `json:"description"`
			Name        string `json:"name"`
		} `json:"node"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: get node: %w", err)
	}

	if len(result.Node) == 0 {
		return nil, fmt.Errorf("%w: id=%d", ErrNotFound, id)
	}

	node := result.Node[0]

	props := map[string]interface{}{
		"project_id": node.ProjectID,
	}
	if node.Description != "" {
		props["description"] = node.Description
	}
	if node.Name != "" {
		props["name"] = node.Name
	}

	return &Node{
		ID:    node.ID,
		Label: node.Label,
		Properties: props,
	}, nil
}

// UpdateNode updates properties of an existing node.
// The project_id property cannot be overwritten — it is stripped from the input.
func (c *Client) UpdateNode(ctx context.Context, id int64, props map[string]interface{}) error {
	// Prevent project_id overwrite.
	delete(props, "project_id")

	t := helixsdk.G().N(helixsdk.NodeID(uint64(id)))
	for k, v := range props {
		t = t.SetProperty(k, v)
	}
	t = t.Count()

	q := helixsdk.WriteQuery("update_node").
		VarAs("updated", t).
		Returning("updated")

	var result struct {
		Updated []struct {
			Count int `json:"count"`
		} `json:"updated"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return fmt.Errorf("helix: update node: %w", err)
	}

	return nil
}

// DeleteNode removes a node by ID.
func (c *Client) DeleteNode(ctx context.Context, id int64) error {
	q := helixsdk.WriteQuery("delete_node").
		VarAs("deleted",
			helixsdk.G().N(helixsdk.NodeID(uint64(id))).
				Drop().
				Count(),
		).
		Returning("deleted")

	var result struct {
		Deleted []struct {
			Count int `json:"count"`
		} `json:"deleted"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return fmt.Errorf("helix: delete node: %w", err)
	}

	return nil
}

// FindNodes searches for nodes by label and optional property filters,
// automatically scoped to the configured project.
func (c *Client) FindNodes(ctx context.Context, label string, filters map[string]interface{}) ([]*Node, error) {
	// Build project-scoped predicate chain.
	conds := []helixsdk.SourcePredicate{
		helixsdk.SourceEq("$label", label),
	}
	if c.projectID != "" {
		conds = append(conds, helixsdk.SourceEq("project_id", c.projectID))
	}
	for k, v := range filters {
		conds = append(conds, helixsdk.SourceEq(k, v))
	}

	var where helixsdk.SourcePredicate
	if len(conds) == 1 {
		where = conds[0]
	} else {
		where = helixsdk.SourceAnd(conds...)
	}

	q := helixsdk.ReadQuery("find_nodes").
		VarAs("nodes",
			helixsdk.G().NWhere(where).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
				),
		).
		Returning("nodes")

	var result struct {
		Nodes []struct {
			ID    int64  `json:"id"`
			Label string `json:"label"`
		} `json:"nodes"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: find nodes: %w", err)
	}

	nodes := make([]*Node, 0, len(result.Nodes))
	for _, n := range result.Nodes {
		nodes = append(nodes, &Node{ID: n.ID, Label: n.Label})
	}
	return nodes, nil
}

// FindSharedSkills busca Skills sin filtro de project_id.
// Skills son globales — no tienen project_id.
func (c *Client) FindSharedSkills(ctx context.Context, label string) ([]*Node, error) {
	q := helixsdk.ReadQuery("find_shared_skills").
		VarAs("skills",
			helixsdk.G().NWhere(helixsdk.SourceEq("$label", label)).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
					helixsdk.ProjectProp("name"),
					helixsdk.ProjectProp("type"),
					helixsdk.ProjectProp("source_url"),
				),
		).
		Returning("skills")

	var result struct {
		Skills []struct {
			ID        int64  `json:"id"`
			Label     string `json:"label"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			SourceURL string `json:"source_url"`
		} `json:"skills"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: find shared skills: %w", err)
	}

	nodes := make([]*Node, 0, len(result.Skills))
	for _, s := range result.Skills {
		nodes = append(nodes, &Node{
			ID:    s.ID,
			Label: s.Label,
			Properties: map[string]interface{}{
				"name":       s.Name,
				"type":       s.Type,
				"source_url": s.SourceURL,
			},
		})
	}
	return nodes, nil
}

// CreateSkill crea un Skill si no existe por nombre (unique).
// Skills son globales — NO se inyecta project_id.
func (c *Client) CreateSkill(ctx context.Context, name, skillType, sourceURL string) (int64, error) {
	// 1. Buscar Skill existente por nombre (unique)
	q := helixsdk.ReadQuery("find_skill_by_name").
		VarAs("skills",
			helixsdk.G().NWhere(helixsdk.SourceAnd(
				helixsdk.SourceEq("$label", "Skill"),
				helixsdk.SourceEq("name", name),
			)).
				Project(helixsdk.ProjectPropAs("$id", "id")),
		).
		Returning("skills")

	var findResult struct {
		Skills []struct {
			ID int64 `json:"id"`
		} `json:"skills"`
	}

	if err := c.inner.Exec(ctx, q, &findResult); err != nil {
		return 0, fmt.Errorf("helix: find skill: %w", err)
	}

	// 2. Si existe → return ID
	if len(findResult.Skills) > 0 {
		return findResult.Skills[0].ID, nil
	}

	// 3. Si no existe → crear con name, type, source_url (SIN project_id)
	props := mapToProps(map[string]interface{}{
		"name":       name,
		"type":       skillType,
		"source_url": sourceURL,
	})

	createQ := helixsdk.WriteQuery("create_skill").
		VarAs("skill",
			helixsdk.G().AddN("Skill", props).
				Project(helixsdk.ProjectPropAs("$id", "id")),
		).
		Returning("skill")

	var createResult struct {
		Skill []struct {
			ID int64 `json:"id"`
		} `json:"skill"`
	}

	if err := c.inner.Exec(ctx, createQ, &createResult); err != nil {
		return 0, fmt.Errorf("helix: create skill: %w", err)
	}

	if len(createResult.Skill) == 0 {
		return 0, ErrNotFound
	}

	return createResult.Skill[0].ID, nil
}

// LinkSkillToProject crea edge REQUIRES_SKILL con required_level.
func (c *Client) LinkSkillToProject(ctx context.Context, projectID, skillID uint64, level string) (int64, error) {
	return c.CreateEdge(ctx, int64(projectID), int64(skillID), "REQUIRES_SKILL", map[string]interface{}{
		"required_level": level,
	})
}

// GetProjectSkills obtiene skills de un proyecto via traversal Project → Out("REQUIRES_SKILL") → Skills.
func (c *Client) GetProjectSkills(ctx context.Context, projectID uint64) ([]*Node, error) {
	return c.GetOutgoing(ctx, int64(projectID), "REQUIRES_SKILL")
}

// UpsertSkill crea o actualiza un Skill por nombre.
// Si existe, actualiza type y source_url. Si no existe, crea uno nuevo.
func (c *Client) UpsertSkill(ctx context.Context, name, skillType, sourceURL string) (int64, error) {
	// Buscar por nombre
	q := helixsdk.ReadQuery("find_skill_by_name").
		VarAs("skills",
			helixsdk.G().NWhere(helixsdk.SourceAnd(
				helixsdk.SourceEq("$label", "Skill"),
				helixsdk.SourceEq("name", name),
			)).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectProp("name"),
					helixsdk.ProjectProp("type"),
					helixsdk.ProjectProp("source_url"),
				),
		).
		Returning("skills")

	var findResult struct {
		Skills []struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			SourceURL string `json:"source_url"`
		} `json:"skills"`
	}

	if err := c.inner.Exec(ctx, q, &findResult); err != nil {
		return 0, fmt.Errorf("helix: find skill: %w", err)
	}

	if len(findResult.Skills) > 0 {
		// Existe — actualizar type y source_url
		id := findResult.Skills[0].ID
		if err := c.UpdateNode(ctx, id, map[string]interface{}{
			"type":       skillType,
			"source_url": sourceURL,
		}); err != nil {
			return 0, fmt.Errorf("helix: update skill: %w", err)
		}
		return id, nil
	}

	// No existe — crear nuevo
	return c.CreateSkill(ctx, name, skillType, sourceURL)
}

// ---------------------------------------------------------------------------
// CodeNode operations
// ---------------------------------------------------------------------------

// findCodeNodeByPath searches for a CodeNode by (project_id, path).
// projectID is the HelixDB node ID of the Project, converted to string for the property lookup.
func (c *Client) findCodeNodeByPath(ctx context.Context, projectID uint64, path string) (*Node, error) {
	conds := []helixsdk.SourcePredicate{
		helixsdk.SourceEq("$label", "CodeNode"),
		helixsdk.SourceEq("project_id", projectIDStr(projectID)),
		helixsdk.SourceEq("path", path),
	}

	var where helixsdk.SourcePredicate
	if len(conds) == 1 {
		where = conds[0]
	} else {
		where = helixsdk.SourceAnd(conds...)
	}

	q := helixsdk.ReadQuery("find_codenode_by_path").
		VarAs("nodes",
			helixsdk.G().NWhere(where).
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
					helixsdk.ProjectProp("project_id"),
					helixsdk.ProjectProp("path"),
					helixsdk.ProjectProp("hash"),
					helixsdk.ProjectProp("summary"),
					helixsdk.ProjectProp("name"),
				),
		).
		Returning("nodes")

	var result struct {
		Nodes []struct {
			ID        int64  `json:"id"`
			Label     string `json:"label"`
			ProjectID string `json:"project_id"`
			Path      string `json:"path"`
			Hash      string `json:"hash"`
			Summary   string `json:"summary"`
			Name      string `json:"name"`
		} `json:"nodes"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: find codenode by path: %w", err)
	}

	if len(result.Nodes) == 0 {
		return nil, fmt.Errorf("%w: CodeNode project_id=%s path=%s", ErrNotFound, projectIDStr(projectID), path)
	}

	n := result.Nodes[0]
	return &Node{
		ID:    n.ID,
		Label: n.Label,
		Properties: map[string]interface{}{
			"project_id": n.ProjectID,
			"path":       n.Path,
			"hash":       n.Hash,
			"summary":    n.Summary,
			"name":       n.Name,
		},
	}, nil
}

// UpsertCodeNode creates or updates a CodeNode identified by (project_id, path).
// Returns (nodeID, changed, error). changed=true if the node was created or its hash changed.
func (c *Client) UpsertCodeNode(ctx context.Context, projectID uint64, path, name, summary, hash string, imports []string) (int64, bool, error) {
	// 1. Search for existing CodeNode by (project_id, path)
	existing, err := c.findCodeNodeByPath(ctx, projectID, path)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return 0, false, fmt.Errorf("helix: upsert codenode find: %w", err)
	}

	if existing != nil {
		// 2. Same hash → no change
		existingHash, _ := existing.Properties["hash"].(string)
		if existingHash == hash {
			return existing.ID, false, nil
		}

		// 3. Different hash → update summary + hash
		updates := map[string]interface{}{
			"summary": summary,
			"hash":    hash,
			"name":    name,
		}
		if err := c.UpdateNode(ctx, existing.ID, updates); err != nil {
			return 0, false, fmt.Errorf("helix: upsert codenode update: %w", err)
		}
		return existing.ID, true, nil
	}

	// 4. Not found → create new CodeNode with project_id + edge HAS_CODENODE
	props := map[string]interface{}{
		"project_id": projectIDStr(projectID),
		"path":       path,
		"name":       name,
		"summary":    summary,
		"hash":       hash,
	}

	nodeID, err := c.CreateNode(ctx, "CodeNode", props)
	if err != nil {
		return 0, false, fmt.Errorf("helix: upsert codenode create: %w", err)
	}

	// Create HAS_CODENODE edge from Project to CodeNode
	if _, err := c.CreateEdge(ctx, int64(projectID), nodeID, "HAS_CODENODE", nil); err != nil {
		return 0, false, fmt.Errorf("helix: upsert codenode edge: %w", err)
	}

	return nodeID, true, nil
}

// GetCodeNodesByProject returns all CodeNodes reachable from a Project via HAS_CODENODE edges.
func (c *Client) GetCodeNodesByProject(ctx context.Context, projectID uint64) ([]*Node, error) {
	return c.GetOutgoing(ctx, int64(projectID), "HAS_CODENODE")
}

// ---------------------------------------------------------------------------
// Task → CodeNode linking
// ---------------------------------------------------------------------------

// LinkTaskToCodeNodes conecta una task con CodeNodes para los archivos modificados.
// Si un CodeNode no existe para (project_id, path), lo crea como stub.
func (c *Client) LinkTaskToCodeNodes(ctx context.Context, taskID uint64, files []git.ChangedFile) (int, error) {
	linked := 0

	// Obtener projectID de la task
	taskNode, err := c.GetNode(ctx, int64(taskID))
	if err != nil {
		return 0, fmt.Errorf("helix: get task: %w", err)
	}

	// Extraer project_id de las props de la task
	projectID := extractProjectID(taskNode)

	for _, f := range files {
		if f.IsDeleted() {
			continue // no linkear archivos borrados
		}

		path := f.Path
		if f.IsRename() {
			path = f.Path // usar nuevo path
		}

		// Buscar o crear CodeNode
		nodeID, err := c.ensureCodeNode(ctx, projectID, path)
		if err != nil {
			continue // skip errors, continue with others
		}

		// Crear edge REFERENCES si no existe
		_, err = c.CreateEdge(ctx, int64(taskID), nodeID, "REFERENCES", nil)
		if err != nil {
			continue
		}

		linked++
	}

	return linked, nil
}

// ensureCodeNode busca o crea un CodeNode mínimo para un path.
func (c *Client) ensureCodeNode(ctx context.Context, projectID uint64, path string) (int64, error) {
	// Intentar buscar existente
	existing, err := c.findCodeNodeByPath(ctx, projectID, path)
	if err == nil && existing != nil {
		return existing.ID, nil
	}

	// No existe → crear stub mínimo
	name := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		name = path[idx+1:]
	}

	return c.CreateNode(ctx, "CodeNode", map[string]interface{}{
		"name":       name,
		"path":       path,
		"project_id": projectIDStr(projectID),
		"summary":    "", // stub, se completa después
	})
}

// extractProjectID helper para obtener project_id de un nodo.
func extractProjectID(node *Node) uint64 {
	if node == nil || node.Properties == nil {
		return 0
	}
	// Intentar distintos formatos
	if id, ok := node.Properties["project_id"].(float64); ok {
		return uint64(id)
	}
	if id, ok := node.Properties["project_id"].(uint64); ok {
		return id
	}
	if idStr, ok := node.Properties["project_id"].(string); ok {
		// Parsear string como uint64
		var id uint64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err == nil {
			return id
		}
	}
	return 0
}
