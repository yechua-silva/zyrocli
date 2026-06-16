package taskcontext

import (
	"context"
	"fmt"

	helix "github.com/secko/zyrocli/internal/db/helix"
)

// GetTaskContext recupera contexto completo para una task desde HelixDB
// Incluye: Skills, CodeNodes, Documents, Patterns
func GetTaskContext(ctx context.Context, client *helix.Client, taskID uint64) (*TaskContext, error) {
	tc := &TaskContext{TaskID: taskID}

	// 1. Obtener la task
	taskNode, err := client.GetNode(ctx, int64(taskID))
	if err != nil {
		return nil, fmt.Errorf("taskcontext: get task %d: %w", taskID, err)
	}

	if desc, ok := taskNode.Properties["description"].(string); ok {
		tc.Description = desc
	}

	// 2. Skills que requiere la task
	skills, err := client.GetOutgoing(ctx, int64(taskID), "REQUIRES")
	if err == nil {
		for _, n := range skills {
			name, _ := n.Properties["name"].(string)
			tc.Skills = append(tc.Skills, ContextItem{
				Name: name,
				Type: "skill",
			})
		}
	}

	// 3. CodeNodes referenciados por la task
	codeNodes, err := client.GetOutgoing(ctx, int64(taskID), "REFERENCES")
	if err == nil {
		for _, n := range codeNodes {
			name, _ := n.Properties["name"].(string)
			summary, _ := n.Properties["summary"].(string)
			path, _ := n.Properties["path"].(string)
			tc.CodeNodes = append(tc.CodeNodes, ContextItem{
				Name:    name,
				Summary: summary,
				Type:    path,
			})
		}
	}

	// 4. Documents + Patterns del proyecto (via la task → project)
	// Task → In(HAS_TASK) → Project → Out(HAS_DOC) + Out(HAS_PATTERN)
	projects, err := client.GetIncoming(ctx, int64(taskID), "HAS_TASK")
	if err == nil && len(projects) > 0 {
		projectID := projects[0].ID

		// Documents
		docs, err := client.GetOutgoing(ctx, projectID, "HAS_DOC")
		if err == nil {
			for _, n := range docs {
				title, _ := n.Properties["title"].(string)
				docType, _ := n.Properties["doc_type"].(string)
				tc.Documents = append(tc.Documents, ContextItem{
					Name: title,
					Type: docType,
				})
			}
		}

		// Patterns
		patterns, err := client.GetOutgoing(ctx, projectID, "HAS_PATTERN")
		if err == nil {
			for _, n := range patterns {
				name, _ := n.Properties["name"].(string)
				tc.Patterns = append(tc.Patterns, ContextItem{
					Name: name,
					Type: "pattern",
				})
			}
		}
	}

	return tc, nil
}
