// Package taskcontext fetches and formats task context from HelixDB.
package taskcontext

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	helix "github.com/secko/zyrocli/internal/db/helix"
)

// TaskContext holds the full context for a single task: skills, code nodes,
// documents, patterns, dependents, and dependencies.
type TaskContext struct {
	TaskID       uint64         `json:"task_id"`
	Skills       []*helix.Node  `json:"skills"`
	CodeNodes    []*helix.Node  `json:"code_nodes"`
	Docs         []*helix.Node  `json:"docs"`
	Patterns     []*helix.Node  `json:"patterns"`
	Dependents   []*helix.Node  `json:"dependents"`
	Dependencies []*helix.Node  `json:"dependencies"`
	Errors       []string       `json:"errors,omitempty"`
}

// GetTaskContext retrieves and assembles task context from HelixDB using 6 graph
// traversals:
//  1. skills:      GetOutgoing(task, "REQUIRES_SKILL")
//  2. code:        GetOutgoing(task, "REFERENCES")
//  3. docs:        GetIncoming(task, "HAS_TASK") → parent project → GetOutgoing(project, "HAS_DOC")
//  4. patterns:    GetIncoming(task, "HAS_TASK") → parent project → GetOutgoing(project, "HAS_PATTERN")
//  5. dependents:  GetIncoming(task, "DEPENDS_ON")
//  6. dependencies: GetOutgoing(task, "DEPENDS_ON")
//
// On partial failure, Errors are collected and partial results are returned (R-TASK-006).
// Returns ErrTaskNotFound when the task node does not exist (R-TASK-005).
func GetTaskContext(ctx context.Context, client *helix.Client, taskID uint64) (*TaskContext, error) {
	tc := &TaskContext{TaskID: taskID}

	// Step 0: Verify task node exists.
	taskNode, err := client.GetNode(ctx, "Task", int64(taskID))
	if err != nil {
		if errors.Is(err, helix.ErrNotFound) {
			return nil, helix.ErrTaskNotFound
		}
		return nil, fmt.Errorf("taskcontext: get task: %w", err)
	}

	// Step 1: Skills.
	tc.Skills, err = client.GetOutgoing(ctx, taskNode.ID, "REQUIRES_SKILL")
	if err != nil {
		tc.Errors = append(tc.Errors, fmt.Sprintf("skills: %v", err))
	}

	// Step 2: Code nodes.
	tc.CodeNodes, err = client.GetOutgoing(ctx, taskNode.ID, "REFERENCES")
	if err != nil {
		tc.Errors = append(tc.Errors, fmt.Sprintf("code: %v", err))
	}

	// Steps 3–4: Docs and patterns come from the parent project.
	projects, err := client.GetIncoming(ctx, taskNode.ID, "HAS_TASK")
	if err != nil || len(projects) == 0 {
		tc.Errors = append(tc.Errors, fmt.Sprintf("project: %v", err))
	} else {
		// Incoming returns the source nodes (parent projects).
		parentNode := projects[0]
		tc.Docs, err = client.GetOutgoing(ctx, parentNode.ID, "HAS_DOC")
		if err != nil {
			tc.Errors = append(tc.Errors, fmt.Sprintf("docs: %v", err))
		}
		tc.Patterns, err = client.GetOutgoing(ctx, parentNode.ID, "HAS_PATTERN")
		if err != nil {
			tc.Errors = append(tc.Errors, fmt.Sprintf("patterns: %v", err))
		}
	}

	// Step 5: Dependents (tasks that depend on this task).
	tc.Dependents, err = client.GetIncoming(ctx, taskNode.ID, "DEPENDS_ON")
	if err != nil {
		tc.Errors = append(tc.Errors, fmt.Sprintf("dependents: %v", err))
	}

	// Step 6: Dependencies (tasks this task depends on).
	tc.Dependencies, err = client.GetOutgoing(ctx, taskNode.ID, "DEPENDS_ON")
	if err != nil {
		tc.Errors = append(tc.Errors, fmt.Sprintf("dependencies: %v", err))
	}

	return tc, nil
}

// FormatJSON returns the context as an indented JSON string.
func (tc *TaskContext) FormatJSON() (string, error) {
	b, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("taskcontext: marshal json: %w", err)
	}
	return string(b), nil
}

// FormatPrompt returns the context formatted as a structured prompt with
// section headers per traversal (R-TASK-003).
func (tc *TaskContext) FormatPrompt() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Task %d Context\n\n", tc.TaskID))

	writePromptSection(&b, "Skills", tc.Skills)
	writePromptSection(&b, "Code", tc.CodeNodes)
	writePromptSection(&b, "Docs", tc.Docs)
	writePromptSection(&b, "Patterns", tc.Patterns)
	writePromptSection(&b, "Dependents", tc.Dependents)
	writePromptSection(&b, "Dependencies", tc.Dependencies)

	if len(tc.Errors) > 0 {
		b.WriteString("## Errors\n\n")
		for _, e := range tc.Errors {
			b.WriteString(fmt.Sprintf("- %s\n", e))
		}
	}

	return b.String()
}

// FormatText returns the context as a human-readable summary (R-TASK-004).
func (tc *TaskContext) FormatText() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Task %d Context\n", tc.TaskID))
	b.WriteString(strings.Repeat("=", 60) + "\n\n")

	writeTextSection(&b, "Skills", tc.Skills)
	writeTextSection(&b, "Code Nodes", tc.CodeNodes)
	writeTextSection(&b, "Docs", tc.Docs)
	writeTextSection(&b, "Patterns", tc.Patterns)
	writeTextSection(&b, "Dependents", tc.Dependents)
	writeTextSection(&b, "Dependencies", tc.Dependencies)

	if len(tc.Errors) > 0 {
		b.WriteString("Errors:\n")
		for _, e := range tc.Errors {
			b.WriteString(fmt.Sprintf("  - %s\n", e))
		}
	}

	return b.String()
}

// writePromptSection writes a ## section with the node name descriptions.
func writePromptSection(b *strings.Builder, name string, nodes []*helix.Node) {
	b.WriteString(fmt.Sprintf("## %s\n\n", name))
	if len(nodes) == 0 {
		b.WriteString("(none)\n\n")
		return
	}
	for _, n := range nodes {
		extra := ""
		if n.Properties != nil {
			if v, ok := n.Properties["name"]; ok {
				extra = fmt.Sprintf(" (%s)", v)
			}
		}
		b.WriteString(fmt.Sprintf("- %s%s\n", n.Type, extra))
	}
	b.WriteString("\n")
}

// writeTextSection writes a bullet list with node IDs and names.
func writeTextSection(b *strings.Builder, name string, nodes []*helix.Node) {
	b.WriteString(fmt.Sprintf("%s (%d):\n", name, len(nodes)))
	for _, n := range nodes {
		nameVal := ""
		if n.Properties != nil {
			if v, ok := n.Properties["name"]; ok {
				nameVal = fmt.Sprintf(" — %s", v)
			}
		}
		b.WriteString(fmt.Sprintf("  - [%d] %s%s\n", n.ID, n.Type, nameVal))
	}
	b.WriteString("\n")
}
