package helix

import (
	"context"
	"fmt"

	helixsdk "github.com/helixdb/helix-db/sdks/go"
)

// CreateEdge creates a directed edge fromID → toID with label and properties.
// Returns the new edge ID.
func (c *Client) CreateEdge(ctx context.Context, fromID, toID int64, label string, props map[string]interface{}) (int64, error) {
	q := helixsdk.WriteQuery("create_edge").
		VarAs("edge",
			helixsdk.G().
				N(helixsdk.NodeID(uint64(fromID))).
				AddE(label, helixsdk.NodeID(uint64(toID)), mapToProps(props)).
				Project(helixsdk.ProjectPropAs("$id", "id")),
		).
		Returning("edge")

	var result struct {
		Edge []struct {
			ID int64 `json:"id"`
		} `json:"edge"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return 0, fmt.Errorf("helix: create edge: %w", err)
	}

	if len(result.Edge) == 0 {
		return 0, fmt.Errorf("helix: create edge: no edge returned")
	}

	return result.Edge[0].ID, nil
}

// GetOutgoing returns destination nodes reachable from nodeID via edgeLabel.
// Properties returned include: name, summary, path, title, doc_type (non-empty only).
func (c *Client) GetOutgoing(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	q := helixsdk.ReadQuery("get_outgoing").
		VarAs("targets",
			helixsdk.G().
				N(helixsdk.NodeID(uint64(nodeID))).
				OutE(edgeLabel).
				OtherN().
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
					helixsdk.ProjectProp("name"),
					helixsdk.ProjectProp("summary"),
					helixsdk.ProjectProp("path"),
					helixsdk.ProjectProp("title"),
					helixsdk.ProjectProp("doc_type"),
				),
		).
		Returning("targets")

	var result struct {
		Targets []struct {
			ID      int64  `json:"id"`
			Label   string `json:"label"`
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Path    string `json:"path"`
			Title   string `json:"title"`
			DocType string `json:"doc_type"`
		} `json:"targets"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: get outgoing: %w", err)
	}

	nodes := make([]*Node, 0, len(result.Targets))
	for _, t := range result.Targets {
		n := &Node{ID: t.ID, Label: t.Label, Properties: make(map[string]interface{})}
		if t.Name != "" {
			n.Properties["name"] = t.Name
		}
		if t.Summary != "" {
			n.Properties["summary"] = t.Summary
		}
		if t.Path != "" {
			n.Properties["path"] = t.Path
		}
		if t.Title != "" {
			n.Properties["title"] = t.Title
		}
		if t.DocType != "" {
			n.Properties["doc_type"] = t.DocType
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// GetIncoming returns source nodes that point to nodeID via edgeLabel.
// Properties returned include: name, summary, path, title, doc_type (non-empty only).
func (c *Client) GetIncoming(ctx context.Context, nodeID int64, edgeLabel string) ([]*Node, error) {
	q := helixsdk.ReadQuery("get_incoming").
		VarAs("sources",
			helixsdk.G().
				N(helixsdk.NodeID(uint64(nodeID))).
				InE(edgeLabel).
				OtherN().
				Project(
					helixsdk.ProjectPropAs("$id", "id"),
					helixsdk.ProjectPropAs("$label", "label"),
					helixsdk.ProjectProp("name"),
					helixsdk.ProjectProp("summary"),
					helixsdk.ProjectProp("path"),
					helixsdk.ProjectProp("title"),
					helixsdk.ProjectProp("doc_type"),
				),
		).
		Returning("sources")

	var result struct {
		Sources []struct {
			ID      int64  `json:"id"`
			Label   string `json:"label"`
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Path    string `json:"path"`
			Title   string `json:"title"`
			DocType string `json:"doc_type"`
		} `json:"sources"`
	}

	if err := c.inner.Exec(ctx, q, &result); err != nil {
		return nil, fmt.Errorf("helix: get incoming: %w", err)
	}

	nodes := make([]*Node, 0, len(result.Sources))
	for _, s := range result.Sources {
		n := &Node{ID: s.ID, Label: s.Label, Properties: make(map[string]interface{})}
		if s.Name != "" {
			n.Properties["name"] = s.Name
		}
		if s.Summary != "" {
			n.Properties["summary"] = s.Summary
		}
		if s.Path != "" {
			n.Properties["path"] = s.Path
		}
		if s.Title != "" {
			n.Properties["title"] = s.Title
		}
		if s.DocType != "" {
			n.Properties["doc_type"] = s.DocType
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

// DeleteEdge removes an edge by ID.
func (c *Client) DeleteEdge(ctx context.Context, edgeID int64) error {
	q := helixsdk.WriteQuery("delete_edge").
		VarAs("deleted",
			helixsdk.G().E(helixsdk.EdgeID(uint64(edgeID))).
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
		return fmt.Errorf("helix: delete edge: %w", err)
	}

	return nil
}
