package taskcontext

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	helix "github.com/yechua-silva/zyrocli/internal/db/helix"
)

// ---------------------------------------------------------------------------
// FormatText
// ---------------------------------------------------------------------------

func TestFormatText(t *testing.T) {
	tc := &TaskContext{
		TaskID:      42,
		Description: "Implement login flow",
		Skills: []ContextItem{
			{Name: "go", Type: "skill"},
			{Name: "auth", Type: "skill"},
		},
		CodeNodes: []ContextItem{
			{Name: "auth.go", Summary: "Authentication handler", Type: "internal/server/auth.go"},
			{Name: "login.go", Summary: "Login page handler", Type: "internal/handler/login.go"},
			{Name: "user.go", Type: "internal/model/user.go"},
		},
		Documents: []ContextItem{
			{Name: "API Spec", Type: "spec"},
			{Name: "Architecture Overview", Type: "design"},
		},
		Patterns: []ContextItem{
			{Name: "Repository Pattern"},
			{Name: "Middleware Chain"},
		},
	}

	output := tc.FormatText()

	// Verify sections exist
	assert.Contains(t, output, "Context for Task #42")
	assert.Contains(t, output, "Implement login flow")
	assert.Contains(t, output, "Skills (2):")
	assert.Contains(t, output, "CodeNodes (3):")
	assert.Contains(t, output, "Documents (2):")
	assert.Contains(t, output, "Patterns (2):")

	// Verify items
	assert.Contains(t, output, "go")
	assert.Contains(t, output, "auth.go")
	assert.Contains(t, output, "Authentication handler")
	assert.Contains(t, output, "API Spec")
	assert.Contains(t, output, "Repository Pattern")
}

func TestFormatText_Empty(t *testing.T) {
	tc := &TaskContext{
		TaskID:      1,
		Description: "",
	}

	output := tc.FormatText()

	assert.Contains(t, output, "Context for Task #1")
	assert.Contains(t, output, "Skills (0):")
	assert.Contains(t, output, "(none)")
	assert.Contains(t, output, "CodeNodes (0):")
	assert.Contains(t, output, "Documents (0):")
	assert.Contains(t, output, "Patterns (0):")
}

// ---------------------------------------------------------------------------
// FormatJSON
// ---------------------------------------------------------------------------

func TestFormatJSON(t *testing.T) {
	tc := &TaskContext{
		TaskID:      7,
		Description: "Refactor database layer",
		Skills: []ContextItem{
			{Name: "sql", Type: "skill"},
		},
		CodeNodes: []ContextItem{
			{Name: "db.go", Type: "internal/db/db.go"},
		},
	}

	jsonStr, err := tc.FormatJSON()
	require.NoError(t, err)
	require.NotEmpty(t, jsonStr)

	// Verify it's valid JSON
	var parsed TaskContext
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), parsed.TaskID)
	assert.Equal(t, "Refactor database layer", parsed.Description)
	assert.Len(t, parsed.Skills, 1)
	assert.Len(t, parsed.CodeNodes, 1)
	assert.Equal(t, "db.go", parsed.CodeNodes[0].Name)
}

// ---------------------------------------------------------------------------
// FormatPrompt
// ---------------------------------------------------------------------------

func TestFormatPrompt(t *testing.T) {
	tc := &TaskContext{
		TaskID:      42,
		Description: "Fix login validation",
		Skills: []ContextItem{
			{Name: "go", Type: "skill"},
		},
		CodeNodes: []ContextItem{
			{Name: "auth.go", Summary: "Authentication module", Type: "src/auth.go"},
			{Name: "user.go", Type: "src/user.go"},
		},
		Documents: []ContextItem{
			{Name: "Spec v2", Type: "spec"},
		},
		Patterns: []ContextItem{
			{Name: "Validator Pattern"},
		},
	}

	output := tc.FormatPrompt()

	// Markdown sections
	assert.Contains(t, output, "## Context for Task #42")
	assert.Contains(t, output, "Fix login validation")
	assert.Contains(t, output, "### Skills Required")
	assert.Contains(t, output, "### Affected Code")
	assert.Contains(t, output, "### Reference Documents")
	assert.Contains(t, output, "### Patterns")

	// Items
	assert.Contains(t, output, "- go")
	assert.Contains(t, output, "**auth.go**")
	assert.Contains(t, output, "Authentication module")
	assert.Contains(t, output, "- Spec v2 (spec)")
	assert.Contains(t, output, "- Validator Pattern")
}

// ---------------------------------------------------------------------------
// GetTaskContext — mock HelixDB server
// ---------------------------------------------------------------------------

func TestGetTaskContext(t *testing.T) {
	// Mock server that responds based on the query name in the request body.
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyStr := string(body)

		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(bodyStr, "get_node"):
			// Return Task node with description
			w.Write([]byte(`{"node":[{"id":100,"label":"Task","project_id":"42","description":"Implement context command","name":"task-100"}]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "REQUIRES"):
			// Return Skills
			w.Write([]byte(`{"targets":[{"id":200,"label":"Skill","name":"go","summary":"","path":"","title":"","doc_type":""},{"id":201,"label":"Skill","name":"cobra","summary":"","path":"","title":"","doc_type":""}]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "REFERENCES"):
			// Return CodeNodes
			w.Write([]byte(`{"targets":[{"id":300,"label":"CodeNode","name":"context.go","summary":"CLI context command","path":"cmd/zyrocli/context.go","title":"","doc_type":""}]}`))

		case strings.Contains(bodyStr, "get_incoming") && strings.Contains(bodyStr, "HAS_TASK"):
			// Return Project node
			w.Write([]byte(`{"sources":[{"id":500,"label":"Project","name":"ZyroAgentCLI","summary":"","path":"","title":"","doc_type":""}]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "HAS_DOC"):
			// Return Documents
			w.Write([]byte(`{"targets":[{"id":400,"label":"Document","name":"","summary":"","path":"","title":"API Spec","doc_type":"spec"}]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "HAS_PATTERN"):
			// Return Patterns
			w.Write([]byte(`{"targets":[{"id":600,"label":"Pattern","name":"Repository","summary":"","path":"","title":"","doc_type":""}]}`))

		default:
			t.Logf("Unhandled query: %s", bodyStr)
			w.Write([]byte(`{}`))
		}
	}))
	defer mockSrv.Close()

	client, err := helix.NewClient(context.Background(), helix.WithBaseURL(mockSrv.URL))
	require.NoError(t, err)
	require.NotNil(t, client)

	tc, err := GetTaskContext(context.Background(), client, 100)
	require.NoError(t, err)
	require.NotNil(t, tc)

	// Verify task
	assert.Equal(t, uint64(100), tc.TaskID)
	assert.Equal(t, "Implement context command", tc.Description)

	// Verify skills
	require.Len(t, tc.Skills, 2)
	assert.Equal(t, "go", tc.Skills[0].Name)
	assert.Equal(t, "cobra", tc.Skills[1].Name)

	// Verify code nodes
	require.Len(t, tc.CodeNodes, 1)
	assert.Equal(t, "context.go", tc.CodeNodes[0].Name)
	assert.Equal(t, "CLI context command", tc.CodeNodes[0].Summary)
	assert.Equal(t, "cmd/zyrocli/context.go", tc.CodeNodes[0].Type)

	// Verify documents
	require.Len(t, tc.Documents, 1)
	assert.Equal(t, "API Spec", tc.Documents[0].Name)
	assert.Equal(t, "spec", tc.Documents[0].Type)

	// Verify patterns
	require.Len(t, tc.Patterns, 1)
	assert.Equal(t, "Repository", tc.Patterns[0].Name)
}

func TestGetTaskContext_TaskNotFound(t *testing.T) {
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"node":[]}`))
	}))
	defer mockSrv.Close()

	client, err := helix.NewClient(context.Background(), helix.WithBaseURL(mockSrv.URL))
	require.NoError(t, err)

	_, err = GetTaskContext(context.Background(), client, 999)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "taskcontext: get task 999")
}

func TestGetTaskContext_NoRelations(t *testing.T) {
	// Solo task, sin skills, codenodes, docs ni patterns
	mockSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyStr := string(body)

		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(bodyStr, "get_node"):
			w.Write([]byte(`{"node":[{"id":200,"label":"Task","project_id":"42","description":"Simple task","name":"task-200"}]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "REQUIRES"):
			w.Write([]byte(`{"targets":[]}`))

		case strings.Contains(bodyStr, "get_outgoing") && strings.Contains(bodyStr, "REFERENCES"):
			w.Write([]byte(`{"targets":[]}`))

		case strings.Contains(bodyStr, "get_incoming") && strings.Contains(bodyStr, "HAS_TASK"):
			w.Write([]byte(`{"sources":[]}`))

		default:
			w.Write([]byte(`{}`))
		}
	}))
	defer mockSrv.Close()

	client, err := helix.NewClient(context.Background(), helix.WithBaseURL(mockSrv.URL))
	require.NoError(t, err)

	tc, err := GetTaskContext(context.Background(), client, 200)
	require.NoError(t, err)
	require.NotNil(t, tc)

	assert.Equal(t, uint64(200), tc.TaskID)
	assert.Equal(t, "Simple task", tc.Description)
	assert.Empty(t, tc.Skills)
	assert.Empty(t, tc.CodeNodes)
	assert.Empty(t, tc.Documents)
	assert.Empty(t, tc.Patterns)
}
